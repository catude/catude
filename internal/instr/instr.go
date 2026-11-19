// Package instr says which instruction files Claude Code reads for a project: CLAUDE.md, CLAUDE.local.md
// and AGENTS.md, following the documented rules (code.claude.com/docs/en/memory, "AGENTS.md"):
//
//   - by default (claude-md-or-agents-md) AGENTS.md is read only when there is no CLAUDE.md,
//     .claude/CLAUDE.md or CLAUDE.local.md in the working directory or above it;
//   - claude-md-and-agents-md reads both; claude-md reads only CLAUDE.md; managed-only reads neither;
//   - the mode is set in /config ("Project instructions") or in pluginConfigs under
//     cc-plugin-agents-md@builtin (or agents-md@builtin before v2.1.285), read only from user settings,
//     --settings files and managed settings;
//   - AGENTS.local.md, AGENTS.override.md and anything under .agents/ are never read.
package instr

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/catude/catude/internal/paths"
)

// File is one instruction file found for the project.
type File struct {
	Path   string
	Kind   string // claude | agents | ignored
	Loaded bool
	Why    string
}

// Report is what Claude Code reads here.
type Report struct {
	Mode        string // the Project instructions value in effect
	ModeFrom    string // where it was set ("" = default)
	Files       []File
	AgentSkills []string // skill folders under .agents/skills, which Claude Code does not read
}

// Ignored lists AGENTS.md files that exist but are not read (and not imported from a CLAUDE.md).
func (r Report) Ignored() []File {
	var out []File
	for _, f := range r.Files {
		if f.Kind == "agents" && !f.Loaded {
			out = append(out, f)
		}
	}
	return out
}

var reImport = regexp.MustCompile(`(?m)(^|\s)@(\./)?(\.claude/)?AGENTS\.md\b`)

// Check builds the report for env.ProjectDir.
func Check(env paths.Env) Report {
	r := Report{Mode: "claude-md-or-agents-md"}
	for _, f := range []string{filepath.Join(env.ClaudeDir, "settings.json"), env.ManagedSettings()} {
		if m := modeIn(f); m != "" {
			r.Mode, r.ModeFrom = m, f
		}
	}
	if env.ProjectDir == "" {
		return r
	}
	var claude, agents []string
	imported := false
	for dir := filepath.Clean(env.ProjectDir); ; {
		// in the home folder, .claude/ holds your user files (~/.claude/CLAUDE.md), which do not count
		atHome := env.Home != "" && dir == filepath.Clean(env.Home)
		for _, n := range []string{"CLAUDE.md", filepath.Join(".claude", "CLAUDE.md"), "CLAUDE.local.md"} {
			p := filepath.Join(dir, n)
			if atHome && strings.HasPrefix(n, ".claude") {
				continue
			}
			if isFile(p) {
				claude = append(claude, p)
				if b, err := os.ReadFile(p); err == nil && reImport.Match(b) {
					imported = true
				}
			}
		}
		for _, n := range []string{"AGENTS.md", filepath.Join(".claude", "AGENTS.md")} {
			if p := filepath.Join(dir, n); isFile(p) && !(atHome && strings.HasPrefix(n, ".claude")) {
				agents = append(agents, p)
			}
		}
		for _, n := range []string{"AGENTS.local.md", "AGENTS.override.md"} {
			if p := filepath.Join(dir, n); isFile(p) {
				r.Files = append(r.Files, File{p, "ignored", false, "Claude Code never reads " + n})
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir || atHome {
			break
		}
		dir = parent
	}
	for _, p := range claude {
		loaded, why := r.Mode != "managed-only", "project instructions"
		if !loaded {
			why = "Project instructions is managed-only"
		}
		r.Files = append(r.Files, File{p, "claude", loaded, why})
	}
	for _, p := range agents {
		f := File{Path: p, Kind: "agents"}
		switch {
		case imported:
			f.Loaded, f.Why = true, "imported from a CLAUDE.md"
		case r.Mode == "claude-md-and-agents-md":
			f.Loaded, f.Why = true, "Project instructions reads CLAUDE.md and AGENTS.md"
		case r.Mode == "claude-md-or-agents-md" && len(claude) == 0:
			f.Loaded, f.Why = true, "no CLAUDE.md here, so AGENTS.md is read instead"
		case r.Mode == "claude-md-or-agents-md":
			f.Why = "not read: a CLAUDE.md, .claude/CLAUDE.md or CLAUDE.local.md exists here or above (" + filepath.Base(claude[0]) + ")"
		default:
			f.Why = "not read: Project instructions is " + r.Mode
		}
		r.Files = append(r.Files, f)
	}
	if ents, err := os.ReadDir(filepath.Join(env.ProjectDir, ".agents", "skills")); err == nil {
		for _, e := range ents {
			if e.IsDir() && isFile(filepath.Join(env.ProjectDir, ".agents", "skills", e.Name(), "SKILL.md")) {
				r.AgentSkills = append(r.AgentSkills, e.Name())
			}
		}
	}
	return r
}

func modeIn(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var s struct {
		PluginConfigs map[string]struct {
			Options struct {
				InstructionFiles string `json:"instructionFiles"`
			} `json:"options"`
		} `json:"pluginConfigs"`
	}
	if json.Unmarshal([]byte(strings.TrimPrefix(string(b), string(rune(0xFEFF)))), &s) != nil {
		return ""
	}
	for _, id := range []string{"cc-plugin-agents-md@builtin", "agents-md@builtin"} {
		switch v := s.PluginConfigs[id].Options.InstructionFiles; v {
		case "claude-md-or-agents-md", "claude-md-and-agents-md", "claude-md", "managed-only":
			return v
		}
	}
	return ""
}

func isFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

// Fix is one change that makes Claude Code read what it skips today.
type Fix struct {
	What string // shown before asking
	do   func() error
}

// Fixes lists the changes for this project: an @AGENTS.md import in the instruction file that stops it
// (documented in "Share one file with other coding tools"), and a copy of each .agents/skills skill into
// .claude/skills when no skill with that name exists there. Nothing is written until Apply.
func Fixes(env paths.Env, backup func(string) (string, error)) []Fix {
	r := Check(env)
	var out []Fix
	if ign := r.Ignored(); len(ign) > 0 && r.Mode == "claude-md-or-agents-md" {
		// the file to import from: the project's own CLAUDE.local.md or CLAUDE.md that stops AGENTS.md
		var target string
		for _, f := range r.Files {
			if f.Kind == "claude" && filepath.Dir(f.Path) == filepath.Clean(env.ProjectDir) && filepath.Base(f.Path) == "CLAUDE.local.md" {
				target = f.Path
			}
		}
		for _, f := range r.Files {
			if target == "" && f.Kind == "claude" {
				target = f.Path
			}
		}
		rel, err := filepath.Rel(filepath.Dir(target), ign[0].Path)
		if target != "" && err == nil {
			line := "@" + filepath.ToSlash(rel)
			out = append(out, Fix{What: "add the line " + line + " to " + target + " so Claude Code reads " + ign[0].Path + " too",
				do: func() error {
					if _, err := backup(target); err != nil {
						return err
					}
					f, err := os.OpenFile(target, os.O_WRONLY|os.O_APPEND, 0)
					if err != nil {
						return err
					}
					_, err = f.WriteString("\n" + line + "\n")
					return errors.Join(err, f.Close())
				}})
		}
	}
	for _, name := range r.AgentSkills {
		src := filepath.Join(env.ProjectDir, ".agents", "skills", name)
		dst := filepath.Join(env.ProjectDir, ".claude", "skills", name)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		out = append(out, Fix{What: "copy the skill " + name + " from .agents/skills to .claude/skills", do: func() error { return copyDir(src, dst) }})
	}
	return out
}

// Apply makes the change.
func (f Fix) Apply() error { return f.do() }

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.Type()&os.ModeSymlink != 0 {
			return nil // links are not followed: a skill folder must not pull files from elsewhere
		}
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

// Memory is Claude Code's auto memory for a project: notes Claude writes itself and reads back in every
// session (code.claude.com/docs/en/memory, "Auto memory"). MEMORY.md is the index; only its first 200
// lines or 25 KB load at the start of a session.
type Memory struct {
	Dir        string
	Files      []string // topic files, without MEMORY.md
	IndexLines int
	IndexBytes int
	Truncated  bool // MEMORY.md is over the 200 lines / 25 KB that load
}

var reNonAlnum = regexp.MustCompile(`[^A-Za-z0-9]`)

// AutoMemory finds the memory folder of the project: Claude Code names it after the git repository
// root (or the folder itself outside git), with every other character turned into "-".
func AutoMemory(env paths.Env) (Memory, bool) {
	if env.ProjectDir == "" || env.ClaudeDir == "" {
		return Memory{}, false
	}
	root := filepath.Clean(env.ProjectDir)
	for d := root; ; {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			root = d
			break
		}
		p := filepath.Dir(d)
		if p == d {
			break
		}
		d = p
	}
	want := reNonAlnum.ReplaceAllString(root, "-")
	ents, err := os.ReadDir(filepath.Join(env.ClaudeDir, "projects"))
	if err != nil {
		return Memory{}, false
	}
	for _, e := range ents {
		if !e.IsDir() || !strings.EqualFold(e.Name(), want) {
			continue
		}
		dir := filepath.Join(env.ClaudeDir, "projects", e.Name(), "memory")
		files, err := os.ReadDir(dir)
		if err != nil {
			return Memory{}, false
		}
		m := Memory{Dir: dir}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
				continue
			}
			if f.Name() == "MEMORY.md" {
				b, _ := os.ReadFile(filepath.Join(dir, f.Name()))
				m.IndexBytes, m.IndexLines = len(b), strings.Count(string(b), "\n")
				m.Truncated = m.IndexLines > 200 || m.IndexBytes > 25<<10
				continue
			}
			m.Files = append(m.Files, f.Name())
		}
		return m, true
	}
	return Memory{}, false
}
