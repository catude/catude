// Package paths finds every place where Claude keeps configuration on this machine:
// Claude Code (user, project, local and managed scopes) and Claude Desktop
// (classic install, Microsoft Store/MSIX virtualised install and the newer Claude-Data layout).
// It never writes anything.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Env holds the resolved base locations. Build it with Detect; tests build it by hand.
type Env struct {
	GOOS         string
	Home         string
	ClaudeDir    string // ~/.claude or $CLAUDE_CONFIG_DIR
	ClaudeJSON   string // ~/.claude.json (user + local-scope MCP, per-project state)
	ProjectDir   string // directory the user is working in (may be empty)
	AppData      string // Windows %APPDATA%
	LocalAppData string // Windows %LOCALAPPDATA%
	ProgramFiles string // Windows %ProgramFiles%

	XDGConfigHome string   // Linux $XDG_CONFIG_HOME
	WSL           bool     // running inside Windows Subsystem for Linux
	WindowsHomes  []string // under WSL: Windows user folders (/mnt/c/Users/<name>)
	PathDirs      []string // $PATH, to find every claude binary
}

// Scope of a configuration file, from lowest to highest precedence in Claude Code.
type Scope string

const (
	ScopeUser     Scope = "user"
	ScopeProject  Scope = "project"
	ScopeLocal    Scope = "local"
	ScopeManaged  Scope = "managed"
	ScopePlugin   Scope = "plugin"
	ScopeSynced   Scope = "synced" // skills synced from claude.ai
	ScopeDesktop  Scope = "desktop"
	ScopeClaudeAI Scope = "claude.ai" // connectors attached to the claude.ai account
)

// File is a configuration file Claude may read, with whether it exists right now.
type File struct {
	Scope  Scope
	Label  string
	Path   string
	Exists bool
}

// Detect resolves the environment of the current user. projectDir may be "".
func Detect(projectDir string) Env {
	home, _ := os.UserHomeDir()
	claudeDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if claudeDir == "" {
		claudeDir = filepath.Join(home, ".claude")
	}
	if projectDir != "" {
		if abs, err := filepath.Abs(projectDir); err == nil {
			projectDir = abs
		}
	}
	e := Env{
		GOOS:          runtime.GOOS,
		Home:          home,
		ClaudeDir:     claudeDir,
		ClaudeJSON:    filepath.Join(home, ".claude.json"),
		ProjectDir:    projectDir,
		AppData:       os.Getenv("APPDATA"),
		LocalAppData:  os.Getenv("LOCALAPPDATA"),
		ProgramFiles:  os.Getenv("ProgramFiles"),
		XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"),
		PathDirs:      filepath.SplitList(os.Getenv("PATH")),
	}
	if runtime.GOOS == "linux" && isWSL() {
		e.WSL = true
		e.WindowsHomes = windowsHomesFromWSL("/mnt/c/Users")
	}
	return e
}

// CATUDE_NO_WSL=1 turns the Windows-side scan off (used by tests, and by people who do not want it).
func isWSL() bool {
	if os.Getenv("CATUDE_NO_WSL") != "" {
		return false
	}
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}
	b, err := os.ReadFile("/proc/sys/kernel/osrelease")
	return err == nil && strings.Contains(strings.ToLower(string(b)), "microsoft")
}

// windowsHomesFromWSL lists real Windows user folders (those with AppData), skipping system profiles.
func windowsHomesFromWSL(usersDir string) []string {
	entries, err := os.ReadDir(usersDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, d := range entries {
		switch strings.ToLower(d.Name()) {
		case "public", "default", "default user", "all users", "defaultapppool", "wsiaccount":
			continue
		}
		p := filepath.Join(usersDir, d.Name())
		if d.IsDir() && exists(filepath.Join(p, "AppData")) {
			out = append(out, p)
		}
	}
	return out
}

func exists(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

func file(scope Scope, label, path string) File {
	return File{Scope: scope, Label: label, Path: path, Exists: exists(path)}
}

// ManagedDir is the folder of enterprise policy files, as documented at
// https://code.claude.com/docs/en/managed-settings (managed-settings.json, managed-settings.d/,
// managed-mcp.json) and https://code.claude.com/docs/en/memory (CLAUDE.md).
func (e Env) ManagedDir() string {
	switch e.GOOS {
	case "windows":
		base := e.ProgramFiles
		if base == "" {
			base = `C:\Program Files`
		}
		return filepath.Join(base, "ClaudeCode")
	case "darwin":
		return "/Library/Application Support/ClaudeCode"
	default: // Linux and WSL
		return "/etc/claude-code"
	}
}

// ManagedSettings is the enterprise policy file (highest precedence, read-only for users).
func (e Env) ManagedSettings() string { return filepath.Join(e.ManagedDir(), "managed-settings.json") }

// ManagedMCP is the MCP servers file an administrator can deploy.
func (e Env) ManagedMCP() string { return filepath.Join(e.ManagedDir(), "managed-mcp.json") }

// SettingsFiles lists Claude Code settings in precedence order (later wins).
func (e Env) SettingsFiles() []File {
	fs := []File{file(ScopeUser, "user settings", filepath.Join(e.ClaudeDir, "settings.json"))}
	if e.ProjectDir != "" {
		fs = append(fs,
			file(ScopeProject, "project settings", filepath.Join(e.ProjectDir, ".claude", "settings.json")),
			file(ScopeLocal, "local settings", filepath.Join(e.ProjectDir, ".claude", "settings.local.json")))
	}
	return append(fs, file(ScopeManaged, "managed settings", e.ManagedSettings()))
}

// MemoryFiles lists CLAUDE.md files Claude Code loads for the current project.
func (e Env) MemoryFiles() []File {
	fs := []File{file(ScopeUser, "user memory", filepath.Join(e.ClaudeDir, "CLAUDE.md"))}
	if e.ProjectDir != "" {
		fs = append(fs,
			file(ScopeProject, "project memory", filepath.Join(e.ProjectDir, "CLAUDE.md")),
			file(ScopeProject, "project memory (.claude)", filepath.Join(e.ProjectDir, ".claude", "CLAUDE.md")),
			file(ScopeLocal, "local memory", filepath.Join(e.ProjectDir, "CLAUDE.local.md")))
	}
	return append(fs, file(ScopeManaged, "organisation memory (managed)", filepath.Join(e.ManagedDir(), "CLAUDE.md")))
}

// DesktopConfigs finds claude_desktop_config.json in every known layout. The Microsoft Store
// (MSIX) build virtualises %APPDATA% under %LOCALAPPDATA%\Packages\Claude_<id>\LocalCache\Roaming,
// so editing the classic path has no effect for that install: showing both is the point.
func (e Env) DesktopConfigs() []File {
	var out []File
	switch e.GOOS {
	case "windows":
		out = windowsDesktop(e.AppData, e.LocalAppData, "")
	case "darwin":
		out = append(out, file(ScopeDesktop, "Claude Desktop", filepath.Join(e.Home, "Library", "Application Support", "Claude", "claude_desktop_config.json")))
	default:
		// Claude Desktop for Linux is an official beta (.deb for Debian/Ubuntu, code.claude.com/docs/en/
		// desktop-linux). Anthropic does not document its config path; the MCP documentation
		// (modelcontextprotocol.io/docs/develop/connect-local-servers) gives ~/.config/Claude/. Only that
		// path is used — plus $XDG_CONFIG_HOME/Claude when the variable points elsewhere, labelled as
		// undocumented. Flatpak and Snap layouts are not documented anywhere official and are not guessed.
		out = append(out, file(ScopeDesktop, "Claude Desktop (Linux)", filepath.Join(e.Home, ".config", "Claude", "claude_desktop_config.json")))
		if x := e.XDGConfigHome; x != "" && filepath.Clean(x) != filepath.Clean(filepath.Join(e.Home, ".config")) {
			out = append(out, file(ScopeDesktop, "Claude Desktop (Linux, $XDG_CONFIG_HOME, not documented)", filepath.Join(x, "Claude", "claude_desktop_config.json")))
		}
		for _, h := range e.WindowsHomes { // WSL: the Windows side of the same machine
			out = append(out, windowsDesktop(filepath.Join(h, "AppData", "Roaming"), filepath.Join(h, "AppData", "Local"), " · Windows via WSL")...)
		}
	}
	return out
}

// windowsDesktop covers the three Windows layouts: installer, every Microsoft Store (MSIX) package
// and the newer Claude-Data folders. suffix labels them when seen from WSL.
func windowsDesktop(appData, localAppData, suffix string) []File {
	var out []File
	if appData != "" {
		out = append(out, file(ScopeDesktop, "Claude Desktop (installer)"+suffix, filepath.Join(appData, "Claude", "claude_desktop_config.json")))
	}
	if localAppData != "" {
		pkgs, _ := filepath.Glob(filepath.Join(localAppData, "Packages", "Claude_*"))
		sort.Strings(pkgs)
		for _, p := range pkgs {
			out = append(out, file(ScopeDesktop, "Claude Desktop (Microsoft Store)"+suffix, filepath.Join(p, "LocalCache", "Roaming", "Claude", "claude_desktop_config.json")))
		}
		for _, pattern := range []string{"*", filepath.Join("*", "*")} {
			out = append(out, globFiles(ScopeDesktop, "Claude Desktop (Claude-Data)"+suffix, filepath.Join(localAppData, "Claude-Data", pattern, "claude_desktop_config.json"))...)
		}
	}
	return out
}

func globFiles(scope Scope, label, pattern string) []File {
	found, _ := filepath.Glob(pattern)
	sort.Strings(found)
	out := make([]File, 0, len(found))
	for _, f := range found {
		out = append(out, file(scope, label, f))
	}
	return out
}

func (e Env) xdgConfig() string {
	if e.XDGConfigHome != "" {
		return e.XDGConfigHome
	}
	return filepath.Join(e.Home, ".config")
}

// ProjectsDir holds one folder per project with the session transcripts (*.jsonl).
func (e Env) ProjectsDir() string { return filepath.Join(e.ClaudeDir, "projects") }

// PluginsDir holds installed plugins, marketplaces and their caches.
func (e Env) PluginsDir() string { return filepath.Join(e.ClaudeDir, "plugins") }

// DesktopDataDirs are Claude Desktop's data folders: next to each config file found, and the
// Claude-Data folders of the newer Windows layout.
func (e Env) DesktopDataDirs() []string {
	seen := map[string]bool{}
	var out []string
	add := func(d string) {
		if d != "" && !seen[strings.ToLower(d)] {
			if fi, err := os.Stat(d); err == nil && fi.IsDir() {
				seen[strings.ToLower(d)] = true
				out = append(out, d)
			}
		}
	}
	for _, f := range e.DesktopConfigs() {
		add(filepath.Dir(f.Path))
	}
	if e.LocalAppData != "" {
		ds, _ := filepath.Glob(filepath.Join(e.LocalAppData, "Claude-Data", "*"))
		for _, d := range ds {
			add(d)
		}
	}
	return out
}
