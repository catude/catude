package paths

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Instance is one installed copy of Claude Code (the CLI binary).
type Instance struct {
	Source  string // PATH, npm, native installer, VS Code extension…
	Path    string
	Version string
}

// ConfigDir is a Claude Code configuration folder (a "profile"): ~/.claude, $CLAUDE_CONFIG_DIR, ~/.claude-*.
type ConfigDir struct {
	Path    string
	Label   string
	Active  bool // the one this catude run is inspecting
	Backup  bool // looks like a backup/credential store: never read
	HasData bool // has settings, sessions or memory
}

var extVersion = regexp.MustCompile(`anthropic\.claude-code-(\d+\.\d+\.\d+)`)

// CodeInstances finds every Claude Code binary: on PATH, the npm package, the native installer and the
// copies bundled inside editor extensions (VS Code, Insiders, Cursor, Windsurf, VSCodium, remote/WSL servers).
func (e Env) CodeInstances() []Instance {
	var out []Instance
	seen := map[string]bool{}
	add := func(src, p, ver string) {
		key := strings.ToLower(filepath.Clean(p))
		if p == "" || seen[key] || !exists(p) {
			return
		}
		seen[key] = true
		out = append(out, Instance{Source: src, Path: p, Version: ver})
	}
	names := []string{"claude"}
	if e.GOOS == "windows" {
		names = []string{"claude.exe", "claude.cmd", "claude"}
	}
	for _, d := range e.PathDirs {
		for _, n := range names {
			p := filepath.Join(d, n)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				add("PATH", p, npmVersion(d))
				break // one entry per directory (claude + claude.cmd are the same install)
			}
		}
	}
	for _, p := range []string{filepath.Join(e.Home, ".local", "bin", exe(e, "claude")), filepath.Join(e.Home, ".claude", "local", exe(e, "claude"))} {
		add("native installer", p, "")
	}
	homes := []struct{ dir, suffix, bin string }{{e.Home, "", exe(e, "claude")}}
	for _, h := range e.WindowsHomes { // WSL: Windows-side installs of the same machine
		homes = append(homes, struct{ dir, suffix, bin string }{h, " (Windows)", "claude.exe"})
		add("native installer (Windows)", filepath.Join(h, ".local", "bin", "claude.exe"), "")
	}
	for _, h := range homes {
		for _, editor := range []string{".vscode", ".vscode-insiders", ".vscode-server", ".vscode-server-insiders", ".cursor", ".cursor-server", ".windsurf", ".vscode-oss"} {
			dirs, _ := filepath.Glob(filepath.Join(h.dir, editor, "extensions", "anthropic.claude-code-*"))
			sort.Strings(dirs)
			for _, d := range dirs {
				ver := ""
				if m := extVersion.FindStringSubmatch(filepath.Base(d)); m != nil {
					ver = m[1]
				}
				bin := h.bin
				if strings.Contains(filepath.Base(d), "win32") {
					bin = "claude.exe"
				}
				add("extension "+strings.TrimPrefix(editor, ".")+h.suffix, filepath.Join(d, "resources", "native-binary", bin), ver)
			}
		}
	}
	return out
}

func exe(e Env, n string) string {
	if e.GOOS == "windows" {
		return n + ".exe"
	}
	return n
}

// npmVersion reads the version of a global npm install next to a PATH entry (Windows: %APPDATA%\npm).
func npmVersion(binDir string) string {
	for _, pkg := range []string{
		filepath.Join(binDir, "node_modules", "@anthropic-ai", "claude-code", "package.json"),
		filepath.Join(binDir, "..", "lib", "node_modules", "@anthropic-ai", "claude-code", "package.json"),
	} {
		b, err := os.ReadFile(pkg)
		if err != nil {
			continue
		}
		var v struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(b, &v) == nil && v.Version != "" {
			return v.Version
		}
	}
	return ""
}

// ConfigDirs lists Claude Code configuration folders (profiles). Folders that look like backups or
// credential stores are listed but flagged so nothing reads them.
func (e Env) ConfigDirs() []ConfigDir {
	cands := []string{filepath.Join(e.Home, ".claude")}
	if v := os.Getenv("CLAUDE_CONFIG_DIR"); v != "" {
		cands = append(cands, v)
	}
	more, _ := filepath.Glob(filepath.Join(e.Home, ".claude-*"))
	sort.Strings(more)
	cands = append(cands, more...)
	winSide := map[string]bool{}
	for _, h := range e.WindowsHomes { // WSL: the Windows profile folders are separate profiles
		w := append([]string{filepath.Join(h, ".claude")}, mustGlob(filepath.Join(h, ".claude-*"))...)
		for _, p := range w {
			winSide[p] = true
		}
		cands = append(cands, w...)
	}
	var out []ConfigDir
	seen := map[string]bool{}
	for _, p := range cands {
		key := strings.ToLower(filepath.Clean(p))
		fi, err := os.Stat(p)
		if err != nil || !fi.IsDir() || seen[key] {
			continue
		}
		seen[key] = true
		name := strings.ToLower(filepath.Base(p))
		d := ConfigDir{Path: p, Label: filepath.Base(p), Active: strings.EqualFold(filepath.Clean(p), filepath.Clean(e.ClaudeDir)),
			Backup: strings.Contains(name, "backup") || strings.Contains(name, "swap")}
		if winSide[p] {
			d.Label += " (Windows)"
		}
		if !d.Backup {
			d.HasData = exists(filepath.Join(p, "settings.json")) || exists(filepath.Join(p, "projects")) || exists(filepath.Join(p, "CLAUDE.md"))
		}
		out = append(out, d)
	}
	return out
}

func mustGlob(pattern string) []string {
	m, _ := filepath.Glob(pattern)
	sort.Strings(m)
	return m
}

// StaleCopies finds leftover copies of ~/.claude.json (.backup, .tmp.*) that may still hold MCP secrets.
// Only names are returned; contents are never read.
func (e Env) StaleCopies() []string {
	var out []string
	for _, pat := range []string{".claude.json.tmp.*", ".claude.json.backup*", ".claude.json.bak*", ".claude.json.old*"} {
		m, _ := filepath.Glob(filepath.Join(e.Home, pat))
		out = append(out, m...)
	}
	sort.Strings(out)
	return out
}
