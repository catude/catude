package paths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func touch(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func labels(fs []File) string {
	var s []string
	for _, f := range fs {
		if f.Exists {
			s = append(s, f.Label)
		}
	}
	return strings.Join(s, " | ")
}

func TestWindowsDesktopAllLayouts(t *testing.T) {
	r := t.TempDir()
	e := Env{GOOS: "windows", Home: r, AppData: filepath.Join(r, "Roaming"), LocalAppData: filepath.Join(r, "Local")}
	touch(t, filepath.Join(e.AppData, "Claude", "claude_desktop_config.json"))
	touch(t, filepath.Join(e.LocalAppData, "Packages", "Claude_aaa", "LocalCache", "Roaming", "Claude", "claude_desktop_config.json"))
	touch(t, filepath.Join(e.LocalAppData, "Packages", "Claude_bbb", "LocalCache", "Roaming", "Claude", "claude_desktop_config.json"))
	touch(t, filepath.Join(e.LocalAppData, "Claude-Data", "claude_x", "claude_desktop_config.json"))
	got := labels(e.DesktopConfigs())
	if strings.Count(got, "Microsoft Store") != 2 || !strings.Contains(got, "installer") || !strings.Contains(got, "Claude-Data") {
		t.Fatalf("missing Windows layouts: %s", got)
	}
}

// Only documented Linux paths: ~/.config/Claude (MCP docs), $XDG_CONFIG_HOME when set (labelled as not
// documented), the Windows side under WSL — and no Flatpak or Snap guesses.
func TestLinuxDesktopDocumentedPathsAndWSL(t *testing.T) {
	r := t.TempDir()
	home := filepath.Join(r, "home")
	win := filepath.Join(r, "mnt", "c", "Users", "ana")
	e := Env{GOOS: "linux", Home: home, XDGConfigHome: filepath.Join(home, "cfg"), WSL: true, WindowsHomes: []string{win}}
	touch(t, filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"))
	touch(t, filepath.Join(home, "cfg", "Claude", "claude_desktop_config.json"))
	touch(t, filepath.Join(home, ".var", "app", "com.example.Claude", "config", "Claude", "claude_desktop_config.json"))
	touch(t, filepath.Join(win, "AppData", "Local", "Packages", "Claude_pzs", "LocalCache", "Roaming", "Claude", "claude_desktop_config.json"))
	fs := e.DesktopConfigs()
	got := labels(fs)
	for _, want := range []string{"Claude Desktop (Linux)", "not documented", "Microsoft Store) · Windows via WSL"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	for _, f := range fs {
		if strings.Contains(f.Path, ".var") || strings.Contains(f.Path, "snap") {
			t.Errorf("undocumented path listed: %s", f.Path)
		}
	}
	if e.ManagedDir() != "/etc/claude-code" || !strings.HasSuffix(filepath.ToSlash(e.ManagedMCP()), "/etc/claude-code/managed-mcp.json") {
		t.Errorf("managed paths: %s %s", e.ManagedDir(), e.ManagedMCP())
	}
	found := false
	for _, f := range e.MemoryFiles() {
		found = found || (f.Label == "organisation memory (managed)" && filepath.ToSlash(f.Path) == "/etc/claude-code/CLAUDE.md")
	}
	if !found {
		t.Error("managed CLAUDE.md (/etc/claude-code/CLAUDE.md) missing")
	}
}

func TestWindowsHomesFromWSLSkipsSystemProfiles(t *testing.T) {
	r := t.TempDir()
	for _, u := range []string{"Public", "Default", "ana", "luis"} {
		if err := os.MkdirAll(filepath.Join(r, u, "AppData"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got := windowsHomesFromWSL(r)
	if len(got) != 2 {
		t.Fatalf("got %v, want ana and luis only", got)
	}
}

func TestManagedSettingsPerOS(t *testing.T) {
	for goos, want := range map[string]string{"windows": "ClaudeCode", "darwin": "/Library/Application Support/ClaudeCode", "linux": "/etc/claude-code"} {
		if got := (Env{GOOS: goos, ProgramFiles: `C:\Program Files`}).ManagedSettings(); !strings.Contains(filepath.ToSlash(got), want) {
			t.Errorf("%s: %s", goos, got)
		}
	}
}

func TestInstancesProfilesAndStaleCopies(t *testing.T) {
	r := t.TempDir()
	e := Env{GOOS: "linux", Home: r, ClaudeDir: filepath.Join(r, ".claude")}
	touch(t, filepath.Join(r, ".claude", "settings.json"))
	touch(t, filepath.Join(r, ".claude-work", "settings.json"))
	touch(t, filepath.Join(r, ".claude-swap-backup", "x"))
	touch(t, filepath.Join(r, ".claude.json.tmp.123.abc"))
	touch(t, filepath.Join(r, ".claude.json.backup"))
	touch(t, filepath.Join(r, ".vscode", "extensions", "anthropic.claude-code-2.1.326-linux-x64", "resources", "native-binary", "claude"))
	touch(t, filepath.Join(r, ".local", "bin", "claude"))
	ds := e.ConfigDirs()
	if len(ds) != 3 || !ds[0].Active {
		t.Fatalf("config dirs: %+v", ds)
	}
	for _, d := range ds {
		if strings.Contains(d.Label, "backup") && (!d.Backup || d.HasData) {
			t.Errorf("backup folder must be flagged and not inspected: %+v", d)
		}
	}
	if s := e.StaleCopies(); len(s) != 2 {
		t.Errorf("stale copies: %v", s)
	}
	in := e.CodeInstances()
	var srcs []string
	for _, i := range in {
		srcs = append(srcs, i.Source+"@"+i.Version)
	}
	got := strings.Join(srcs, " ")
	if !strings.Contains(got, "native installer@") || !strings.Contains(got, "extension vscode@2.1.326") {
		t.Errorf("instances: %s", got)
	}
}
