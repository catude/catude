// Package rigcard implements Rigcard 0.1: a JSON manifest to publish a whole Claude setup at a URL
// (e.g. https://example.dev/rig) and to inspect someone else's before adopting any of it.
//
// Design rules: pinned versions and hashes, secrets never included (names only), hooks and anything
// that executes code are references that importers never activate on their own. See docs/rigcard.md.
package rigcard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/catude/catude/internal/redact"
)

// Version is the Rigcard spec version this package reads and writes.
const Version = "0.1"

// MediaType of a Rigcard document.
const MediaType = "application/vnd.rigcard+json"

type Card struct {
	Schema       string        `json:"$schema,omitempty"`
	Rigcard      string        `json:"rigcard"`
	ID           string        `json:"id,omitempty"`
	Updated      string        `json:"updated,omitempty"`
	Owner        *Owner        `json:"owner,omitempty"`
	Summary      string        `json:"summary,omitempty"`
	License      string        `json:"license,omitempty"`
	Harnesses    []Harness     `json:"harnesses,omitempty"`
	Models       []Model       `json:"models,omitempty"`
	Marketplaces []Marketplace `json:"marketplaces,omitempty"`
	Plugins      []Plugin      `json:"plugins,omitempty"`
	Skills       []Artifact    `json:"skills,omitempty"`
	Agents       []Artifact    `json:"agents,omitempty"`
	Commands     []Artifact    `json:"commands,omitempty"`
	Hooks        []HookRef     `json:"hooks,omitempty"`
	MCPServers   []MCPServer   `json:"mcpServers,omitempty"`
	Settings     *Settings     `json:"settings,omitempty"`
	Desktop      *Desktop      `json:"desktop,omitempty"`
	Practices    []Practice    `json:"practices,omitempty"`
}

type Owner struct {
	Name   string `json:"name,omitempty"`
	Handle string `json:"handle,omitempty"`
	URL    string `json:"url,omitempty"`
	Art    string `json:"art,omitempty"` // optional ASCII art the owner chose for themselves (max 12 lines x 40 columns)
}

type Harness struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

type Model struct {
	ID     string `json:"id"`
	Role   string `json:"role,omitempty"`
	Effort string `json:"effort,omitempty"`
}

// Source points to where an artifact lives. Same vocabulary as Claude Code plugin sources.
type Source struct {
	Type   string `json:"type,omitempty"`   // git | https | github | local
	Source string `json:"source,omitempty"` // marketplace style: github | git | url
	Repo   string `json:"repo,omitempty"`
	URL    string `json:"url,omitempty"`
	Path   string `json:"path,omitempty"`
	Ref    string `json:"ref,omitempty"`
	Commit string `json:"commit,omitempty"`
}

// Pinned reports whether the source is fixed to an immutable revision.
// The revision that is really downloaded is Commit when present, otherwise Ref: only that one counts.
func (s Source) Pinned() bool { return isSHA(s.Revision()) }

// Revision is the git revision a download uses ("" when the source names none).
func (s Source) Revision() string {
	if s.Commit != "" {
		return s.Commit
	}
	return s.Ref
}

type Marketplace struct {
	Name   string `json:"name"`
	Source Source `json:"source"`
}

type Plugin struct {
	ID      string `json:"id"` // name@marketplace
	Version string `json:"version,omitempty"`
	Enabled bool   `json:"enabled"`
}

// Artifact is a skill, agent or command: name, where to get it and the hash to verify it.
type Artifact struct {
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Scope       string  `json:"scope,omitempty"`
	Source      *Source `json:"source,omitempty"`
	SHA256      string  `json:"sha256,omitempty"`
	ManualOnly  bool    `json:"manualOnly,omitempty"`
}

type HookRef struct {
	Event        string  `json:"event"`
	Matcher      string  `json:"matcher,omitempty"`
	Summary      string  `json:"summary,omitempty"`
	ExecutesCode bool    `json:"executesCode"`
	Source       *Source `json:"source,omitempty"`
	SHA256       string  `json:"sha256,omitempty"`
}

// Variable is an environment variable or header an MCP server needs. Never a value.
type Variable struct {
	Name       string `json:"name"`
	IsSecret   bool   `json:"isSecret"`
	IsRequired bool   `json:"isRequired,omitempty"`
	Default    string `json:"default,omitempty"` // only for non-secret values (e.g. a host name)
}

type MCPServer struct {
	Name      string     `json:"name"`
	Transport string     `json:"transport"` // stdio | http | sse
	URL       string     `json:"url,omitempty"`
	Command   string     `json:"command,omitempty"`
	Args      []string   `json:"args,omitempty"`
	Env       []Variable `json:"env,omitempty"`
	Headers   []Variable `json:"headers,omitempty"`
	Connector bool       `json:"connector,omitempty"` // claude.ai account connector: each user authenticates
}

// Settings is the allow-listed subset of Claude Code settings a card may carry.
type Settings struct {
	Model       string       `json:"model,omitempty"`
	EffortLevel string       `json:"effortLevel,omitempty"`
	OutputStyle string       `json:"outputStyle,omitempty"`
	Permissions *Permissions `json:"permissions,omitempty"`
}

// Permissions may only restrict (deny/ask); "allow" is forbidden in cards.
type Permissions struct {
	Deny []string `json:"deny,omitempty"`
	Ask  []string `json:"ask,omitempty"`
}

type Desktop struct {
	Connectors []string `json:"connectors,omitempty"`
}

type Practice struct {
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
}

// ---- validation ----

// Problem found while validating a card.
type Problem struct {
	Severity string // reject | warn
	Where    string
	Msg      string
}

// MaxItems bounds every list in a card.
const MaxItems = 500

// Validate checks a card. Any "reject" problem means the card must not be used.
func Validate(c Card, raw []byte) []Problem {
	var p []Problem
	rej := func(w, m string) { p = append(p, Problem{"reject", w, m}) }
	warn := func(w, m string) { p = append(p, Problem{"warn", w, m}) }
	if c.Rigcard != Version {
		rej("rigcard", fmt.Sprintf("unsupported version %q (this catude reads %s)", c.Rigcard, Version))
	}
	if c.Owner != nil && c.Owner.Art != "" {
		lines := strings.Split(strings.TrimRight(c.Owner.Art, "\n"), "\n")
		if len(lines) > 12 {
			rej("owner.art", "more than 12 lines")
		}
		for _, l := range lines {
			if len([]rune(l)) > 40 {
				rej("owner.art", "a line is wider than 40 columns")
				break
			}
			for _, r := range l {
				if unicode.IsControl(r) || (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) || r == 0x1B {
					rej("owner.art", "control or bidi characters")
					break
				}
			}
		}
	}
	if raw != nil && redact.Text(string(raw)) != string(raw) {
		rej("card", "contains a credential-shaped value: cards must never carry secrets")
	}
	for name, n := range map[string]int{"plugins": len(c.Plugins), "skills": len(c.Skills), "agents": len(c.Agents), "commands": len(c.Commands), "hooks": len(c.Hooks), "mcpServers": len(c.MCPServers), "marketplaces": len(c.Marketplaces)} {
		if n > MaxItems {
			rej(name, fmt.Sprintf("%d items (max %d)", n, MaxItems))
		}
	}
	for _, m := range c.Marketplaces {
		if !safeName(m.Name) {
			rej("marketplace "+m.Name, "unsafe name")
		}
		if strings.EqualFold(m.Name, "claude-plugins-official") || strings.HasPrefix(strings.ToLower(m.Name), "anthropic") {
			if m.Source.Repo != "" && !strings.HasPrefix(strings.ToLower(m.Source.Repo), "anthropics/") {
				rej("marketplace "+m.Name, "impersonates an official Anthropic marketplace")
			}
		}
		if !m.Source.Pinned() {
			warn("marketplace "+m.Name, "not pinned to a commit: its contents can change after you review it")
		}
	}
	for _, a := range append(append(append([]Artifact{}, c.Skills...), c.Agents...), c.Commands...) {
		if !safeName(a.Name) {
			rej("artifact "+a.Name, "unsafe name (control/bidi characters, path separators or '..')")
		}
		if a.Source != nil && a.Source.Type == "bundle" {
			// a bundle file travels with the card: its hash is the only thing that ties it to the card
			if a.SHA256 == "" {
				rej(a.Name, "bundled artifact without sha256")
			}
			if a.Source.Path == "" || strings.HasPrefix(a.Source.Path, "/") || strings.Contains(a.Source.Path, "..") || strings.ContainsAny(a.Source.Path, `\:`) {
				rej(a.Name, "bundled artifact with an unsafe path")
			}
			continue
		}
		if a.Source != nil && a.Source.Type != "local" {
			if !a.Source.Pinned() && a.SHA256 == "" {
				warn(a.Name, "neither pinned to a commit nor protected by sha256")
			}
			if u := firstNonEmpty(a.Source.URL, a.Source.Repo); strings.HasPrefix(strings.ToLower(u), "http://") {
				rej(a.Name, "plain http source")
			}
		}
	}
	for _, h := range c.Hooks {
		if h.ExecutesCode {
			warn("hook "+h.Event, "executes code: an importer must never enable it automatically")
		}
	}
	if raw != nil {
		if where := controlChars(raw); where != "" {
			rej("card", "contains control characters in "+where+": a card is plain text")
		}
	}
	for _, pl := range c.Plugins {
		if !rePluginID.MatchString(pl.ID) {
			rej("plugin "+pl.ID, "a plugin id is name@marketplace (letters, digits, . _ -)")
		}
	}
	for _, m := range c.Marketplaces {
		if m.Source.Repo != "" && !reRepoID.MatchString(m.Source.Repo) {
			rej("marketplace "+m.Name, "repository must be owner/name")
		}
		if u := m.Source.URL; u != "" && (!strings.HasPrefix(u, "https://") || strings.ContainsAny(u, " \t\"'"+"`"+"$;|&<>")) {
			rej("marketplace "+m.Name, "url must be a plain https address")
		}
	}
	for _, s := range c.MCPServers {
		if !safeName(s.Name) || !reMCPName.MatchString(s.Name) {
			rej("mcp "+s.Name, "unsafe name")
		}
		// Claude Code expands ${VAR} in MCP definitions when the profile starts: a card must not be able
		// to place a variable of YOUR environment into a URL, an argument or a default value.
		if strings.Contains(s.URL, "$") || strings.ContainsAny(s.URL, " \t\"'"+"`"+"<>") {
			rej("mcp "+s.Name, "url contains characters that are not allowed ($, quotes, spaces)")
		}
		if s.URL != "" && !strings.HasPrefix(strings.ToLower(s.URL), "https://") && !strings.HasPrefix(strings.ToLower(s.URL), "http://") {
			rej("mcp "+s.Name, "url must be http(s)")
		}
		if strings.ContainsAny(s.Command, "$;|&<>"+"`"+"\"'\n\r") {
			rej("mcp "+s.Name, "command contains shell characters")
		}
		for _, a := range s.Args {
			if strings.Contains(a, "$") || strings.ContainsAny(a, "`"+"\n\r") {
				rej("mcp "+s.Name, "an argument contains $ or a line break")
				break
			}
		}
		for _, v := range append(append([]Variable{}, s.Env...), s.Headers...) {
			if !reVarName.MatchString(v.Name) {
				rej("mcp "+s.Name, "variable or header name "+v.Name+" is not allowed")
			}
			if strings.Contains(v.Default, "$") || strings.ContainsAny(v.Default, "\n\r") {
				rej("mcp "+s.Name, "default value of "+v.Name+" contains $")
			}
		}
		if strings.HasPrefix(strings.ToLower(s.URL), "http://") && !strings.Contains(s.URL, "localhost") && !strings.Contains(s.URL, "127.0.0.1") {
			warn("mcp "+s.Name, "plain http remote server")
		}
		if s.Transport == "stdio" && len(s.Args) > 0 && unpinnedRunner(s.Command, s.Args) {
			warn("mcp "+s.Name, s.Command+" without a pinned package version")
		}
	}
	return p
}

var (
	rePluginID = regexp.MustCompile(`^[A-Za-z0-9][\w.-]*@[A-Za-z0-9][\w.-]*$`)
	reRepoID   = regexp.MustCompile(`^[A-Za-z0-9][\w.-]*/[A-Za-z0-9][\w.-]*$`)
	reMCPName  = regexp.MustCompile(`^[A-Za-z0-9][\w.-]*$`)
	reVarName  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
)

// controlChars reports where a card carries control characters (escape sequences, carriage returns…).
// Only line breaks are allowed, and only inside the owner's ASCII art.
func controlChars(raw []byte) string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	var walk func(x any, path string) string
	walk = func(x any, path string) string {
		switch t := x.(type) {
		case string:
			for _, r := range t {
				if r == '\n' && path == "owner.art" {
					continue
				}
				if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) || (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) {
					return firstNonEmpty(path, "a value")
				}
			}
		case map[string]any:
			for k, val := range t {
				p := k
				if path != "" {
					p = path + "." + k
				}
				if w := walk(k, p); w != "" {
					return w
				}
				if w := walk(val, p); w != "" {
					return w
				}
			}
		case []any:
			for _, val := range t {
				if w := walk(val, path); w != "" {
					return w
				}
			}
		}
		return ""
	}
	return walk(v, "")
}

// Rejected reports whether any problem is blocking.
func Rejected(ps []Problem) bool {
	for _, p := range ps {
		if p.Severity == "reject" {
			return true
		}
	}
	return false
}

func safeName(s string) bool {
	if s == "" || strings.Contains(s, "..") || strings.ContainsAny(s, `/\:<>"|?*`) || len(s) > 128 || strings.HasSuffix(s, ".") || strings.HasSuffix(s, " ") || strings.HasPrefix(s, " ") {
		return false
	}
	base := strings.ToLower(s)
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	switch base {
	case "con", "prn", "aux", "nul", "com1", "com2", "com3", "com4", "com5", "com6", "com7", "com8", "com9", "lpt1", "lpt2", "lpt3", "lpt4", "lpt5", "lpt6", "lpt7", "lpt8", "lpt9":
		return false // reserved device names on Windows
	}
	for _, r := range s {
		if unicode.IsControl(r) || (r >= 0x200B && r <= 0x200F) || (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) {
			return false
		}
	}
	return true
}

func unpinnedRunner(cmd string, args []string) bool {
	c := strings.ToLower(filepath.Base(strings.ReplaceAll(cmd, `\`, "/")))
	c = strings.TrimSuffix(c, ".exe")
	if c != "npx" && c != "uvx" && c != "bunx" && c != "pipx" {
		return false
	}
	for _, a := range args {
		if (strings.Contains(a, "@") && !strings.HasSuffix(a, "@latest") && strings.IndexFunc(a[strings.LastIndex(a, "@"):], unicode.IsDigit) > 0) || strings.Contains(a, "==") {
			return false
		}
	}
	return true
}

func isSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// ---- hashing ----

// TreeHash is the canonical hash of a directory: for each regular file, sorted by slash-separated
// relative path, the line "path\x00sha256hex\n"; the result is the SHA-256 of all lines.
// Symlinks are refused so a tree cannot point outside itself.
func TreeHash(dir string) (string, error) {
	type entry struct{ rel, sum string }
	var es []entry
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("symlink not allowed: %s", p)
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		h := sha256.Sum256(b)
		es = append(es, entry{filepath.ToSlash(rel), hex.EncodeToString(h[:])})
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(es) == 0 {
		return "", errors.New("empty directory")
	}
	sort.Slice(es, func(i, j int) bool { return es[i].rel < es[j].rel })
	h := sha256.New()
	for _, e := range es {
		fmt.Fprintf(h, "%s\x00%s\n", e.rel, e.sum)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// FileHash is the SHA-256 of one file.
func FileHash(p string) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// SafeName reports whether s can be used as a file or folder name coming from a card.
func SafeName(s string) bool { return safeName(s) }
