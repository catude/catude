package profiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/catude/catude/internal/audit"
	"github.com/catude/catude/internal/redact"
	"github.com/catude/catude/internal/rigcard"
)

// BundleResult describes an exported bundle.
type BundleResult struct {
	Dir     string
	Files   int
	Blocked []audit.Finding // what stopped the export (secrets in the files)
}

// ExportBundle writes an installed profile as a folder anyone can load: rigcard.json plus the profile's
// skills, agents and commands, each with its sha256 in the card. Publish the folder as a GitHub repository
// (catude profile load gh:owner/repo) or hand it over as it is (catude profile load DIR/rigcard.json).
//
// The export refuses to run when the audit finds a secret in the files, MCP servers keep only the NAMES
// of their variables, and hooks are never part of a profile.
func ExportBundle(p Profile, dir, summary string) (BundleResult, error) {
	res := BundleResult{Dir: dir}
	for _, f := range auditDir(p.Dir) {
		if f.Severity == audit.High && (f.Rule == "secret-in-text" || f.Rule == "mcp-plaintext-secret" || f.Rule == "mcp-secret-in-args" || f.Rule == "secret-in-hook") {
			res.Blocked = append(res.Blocked, f)
		}
	}
	// every file that would be shared is checked, not only the SKILL.md the audit reads
	filepath.WalkDir(p.PluginDir(), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil || len(b) > MaxFile {
			return nil
		}
		if redact.Text(string(b)) != string(b) {
			rel, _ := filepath.Rel(p.PluginDir(), path)
			res.Blocked = append(res.Blocked, audit.Finding{Severity: audit.High, Rule: "secret-in-file", Item: filepath.ToSlash(rel), Path: path,
				Detail: "contains a credential-shaped value", Fix: "remove it from the file before sharing"})
		}
		return nil
	})
	if len(res.Blocked) > 0 {
		return res, fmt.Errorf("not exported: %d secrets found in the profile's files (remove them first)", len(res.Blocked))
	}
	if ents, err := os.ReadDir(dir); err == nil && len(ents) > 0 {
		return res, errors.New("the folder is not empty: choose a new folder for the bundle")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return res, err
	}
	c := p.Card
	c.Rigcard, c.Updated, c.ID = rigcard.Version, time.Now().UTC().Format(time.RFC3339), ""
	if summary != "" {
		c.Summary = summary
	}
	c.Hooks = nil
	// a shared card never carries values: only the names of the variables
	c.MCPServers = append([]rigcard.MCPServer{}, c.MCPServers...)
	for i := range c.MCPServers {
		s := c.MCPServers[i]
		s.URL = redact.URL(s.URL)
		s.Args = redact.Args(append([]string{}, s.Args...))
		s.Env, s.Headers = stripDefaults(s.Env), stripDefaults(s.Headers)
		c.MCPServers[i] = s
	}
	bundle := func(kind, sub string, names []string, isDir bool) ([]rigcard.Artifact, error) {
		var out []rigcard.Artifact
		for _, n := range names {
			rel := sub + "/" + n
			src := filepath.Join(p.PluginDir(), sub, n)
			if !isDir {
				rel += ".md"
				src += ".md"
			}
			dst := filepath.Join(dir, filepath.FromSlash(rel))
			var err error
			if isDir {
				err = copyTree(src, dst)
			} else {
				err = copyFile(src, dst)
			}
			if err != nil {
				return nil, fmt.Errorf("%s %s: %w", kind, n, err)
			}
			h, err := hashOf(dst, isDir)
			if err != nil {
				return nil, err
			}
			desc := ""
			for _, a := range append(append(append([]rigcard.Artifact{}, p.Card.Skills...), p.Card.Agents...), p.Card.Commands...) {
				if a.Name == n {
					desc = a.Description
				}
			}
			out = append(out, rigcard.Artifact{Name: n, Description: desc, Source: &rigcard.Source{Type: "bundle", Path: rel}, SHA256: h})
			res.Files += countFiles(dst)
		}
		return out, nil
	}
	var err error
	if c.Skills, err = bundle("skill", "skills", p.Skills, true); err != nil {
		return res, err
	}
	if c.Agents, err = bundle("agent", "agents", p.Agents, false); err != nil {
		return res, err
	}
	if c.Commands, err = bundle("command", "commands", p.Commands, false); err != nil {
		return res, err
	}
	raw, err := jsonIndent(c)
	if err != nil {
		return res, err
	}
	if ps := rigcard.Validate(c, raw); rigcard.Rejected(ps) {
		return res, fmt.Errorf("the bundle did not pass validation: %v", ps)
	}
	if err := writeFile(filepath.Join(dir, "rigcard.json"), raw); err != nil {
		return res, err
	}
	readme := "# " + p.Name + "\n\nA catude work profile (Rigcard " + rigcard.Version + ").\n\n```sh\ncatude profile load gh:OWNER/REPO --name " + p.Name + "\n```\n\n" +
		"catude validates the card, checks every file against the sha256 in `rigcard.json`, audits it and asks before installing. Hooks are never installed.\n"
	return res, writeFile(filepath.Join(dir, "README.md"), []byte(readme))
}

func stripDefaults(vs []rigcard.Variable) []rigcard.Variable {
	out := make([]rigcard.Variable, len(vs))
	for i, v := range vs {
		v.Default = ""
		v.IsRequired = true
		out[i] = v
	}
	return out
}

func countFiles(p string) int {
	n := 0
	filepath.WalkDir(p, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}
