package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/prefs"
	"github.com/catude/catude/internal/sessions"
)

func testModel(t *testing.T) (model, paths.Env) {
	t.Helper()
	r := t.TempDir()
	t.Setenv("CATUDE_HOME", filepath.Join(r, "ct"))
	t.Setenv("LOCALAPPDATA", r)
	t.Setenv("XDG_CACHE_HOME", r)
	t.Setenv("HOME", r)
	env := paths.Env{GOOS: "linux", Home: r, ClaudeDir: filepath.Join(r, ".claude"), ClaudeJSON: filepath.Join(r, ".claude.json"), ProjectDir: filepath.Join(r, "proj")}
	os.MkdirAll(env.ClaudeDir, 0o755)
	m := model{env: env, cursor: map[tab]int{}, cache: &sessions.Cache{}, w: 150, h: 44, loaded: true,
		marked: map[string]bool{}, rc: &rowCache{}, projCWD: map[string]string{}}
	return m, env
}

func typeText(m model, s string) model {
	for _, r := range s {
		m, _ = key(m, string(r))
	}
	return m
}

// Config tab: pick Claude Code's user settings, change the model, toggle a yes/no value, and see the
// file updated with its key order intact.
func TestConfigTabEditsASetting(t *testing.T) {
	m, env := testModel(t)
	file := filepath.Join(env.ClaudeDir, "settings.json")
	os.WriteFile(file, []byte("{\n  \"zeta\": 1,\n  \"model\": \"sonnet\",\n  \"alwaysThinkingEnabled\": true\n}\n"), 0o644)
	nm, _ := m.gotoTab(tConfig)
	m = nm.(model)
	v := m.View()
	for _, want := range []string{"Claude Code", "Your settings", "How they relate", "model", "sonnet"} {
		if !strings.Contains(v, want) {
			t.Fatalf("config view misses %q:\n%s", want, v)
		}
	}
	m, _ = key(m, "enter") // into the values
	if m.cfg.focus != 1 {
		t.Fatal("enter must open the values of the selected file")
	}
	m = selectCfg(t, m, "model")
	m, _ = key(m, "enter")
	if !m.cfg.editing || m.cfg.input != "sonnet" {
		t.Fatalf("enter on a text value must start editing with the current value: %+v", m.cfg)
	}
	for range "sonnet" {
		m, _ = m.keyBackspace()
	}
	m = typeText(m, "opus")
	m, _ = key(m, "enter")
	b, _ := os.ReadFile(file)
	if !strings.Contains(string(b), `"model": "opus"`) || strings.Index(string(b), "zeta") > strings.Index(string(b), "model") {
		t.Fatalf("file after edit:\n%s\nstatus: %s", b, m.status)
	}
	m = selectCfg(t, m, "alwaysThinkingEnabled")
	m, _ = key(m, " ")
	b, _ = os.ReadFile(file)
	if !strings.Contains(string(b), `"alwaysThinkingEnabled": false`) {
		t.Fatalf("space must toggle a yes/no value:\n%s\nstatus: %s", b, m.status)
	}
	m, _ = key(m, "esc")
	if m.cfg.focus != 0 {
		t.Fatal("esc returns to the list of files")
	}
}

// selectCfg moves the cursor to a setting, as ↑↓ would.
func selectCfg(t *testing.T, m model, k string) model {
	t.Helper()
	for j, i := range m.cfgVisible() {
		if m.cfg.entries[i].Key == k {
			for m.cfg.esel < j {
				m, _ = key(m, "j")
			}
			for m.cfg.esel > j {
				m, _ = key(m, "k")
			}
			return m
		}
	}
	t.Fatalf("setting %s not listed", k)
	return m
}

func (m model) keyBackspace() (model, tea.Cmd) {
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	return nm.(model), cmd
}

// Preferences: hide a tab, switch to the lite view, change the accent — all saved and applied at once.
func TestPreferencesTabsLiteAndTheme(t *testing.T) {
	defer func() { Accent = ""; Setup("unicode", "auto") }()
	m, _ := testModel(t)
	if len(m.activeTabs()) != len(prefs.DefaultTabs) || m.tabPos(tAudit) >= 0 {
		t.Fatal("the core tabs are visible by default, the rest are in preferences")
	}
	if !strings.Contains(m.View(), F("+%d tabs in ,", len(tabNames)-len(prefs.DefaultTabs))) {
		t.Error("the tab bar says how to show the other tabs")
	}
	m, _ = key(m, ",")
	if !m.pv.open {
		t.Fatal(", opens preferences")
	}
	m.pv.sel = prefsFixedRows + int(tAudit)
	m, _ = key(m, " ")
	if m.tabPos(tAudit) < 0 || len(m.activeTabs()) != len(prefs.DefaultTabs)+1 {
		t.Fatal("space on a hidden tab shows it in the full view")
	}
	if p := prefs.Load(); len(p.Tabs) != len(prefs.DefaultTabs)+1 {
		t.Fatalf("preferences must be saved: %+v", p)
	}
	m, _ = key(m, " ")
	if p := prefs.Load(); len(p.Tabs) != 0 {
		t.Fatalf("back to the default saves nothing, so new core tabs appear by themselves: %+v", p)
	}
	m.pv.sel = 1 // View (row 0 is the language)
	m, _ = key(m, " ")
	if !m.lite || len(m.activeTabs()) != len(prefs.DefaultLite) {
		t.Fatalf("lite view shows the lite tabs: %v", m.activeTabs())
	}
	if !strings.Contains(m.View(), "ctlite") {
		t.Error("the header says ctlite in the lite view")
	}
	m.pv.sel = 4 // accent colour
	m, _ = key(m, "l")
	if m.prefs.Accent != "amber" || Accent != "amber" { // the default is clay; the next one is amber
		t.Errorf("accent: %q / %q", m.prefs.Accent, Accent)
	}
	m, _ = key(m, "esc")
	// number keys and arrows follow the visible tabs
	m, _ = key(m, "2")
	if m.tab != m.activeTabs()[1] {
		t.Fatalf("2 opens the second visible tab, got %v", tabNames[m.tab])
	}
	last := m.activeTabs()[len(m.activeTabs())-1]
	for m.tab != last {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
		m = nm.(model)
	}
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if nm.(model).tab != m.activeTabs()[0] {
		t.Error("→ wraps around inside the visible tabs")
	}
	// the last visible tab cannot be hidden
	m.prefs.LiteTabs = []string{"overview"}
	m.pv = prefsState{open: true, sel: prefsFixedRows + int(tOverview)}
	m, _ = key(m, " ")
	if len(m.activeTabs()) != 1 {
		t.Fatal("at least one tab stays visible")
	}
}

// The effective view sits after the settings files, shows which file each value comes from and is read-only.
func TestConfigEffectiveView(t *testing.T) {
	m, env := testModel(t)
	os.WriteFile(filepath.Join(env.ClaudeDir, "settings.json"), []byte(`{"model": "opus", "spinnerTipsEnabled": true}`), 0o644)
	os.MkdirAll(filepath.Join(env.ProjectDir, ".claude"), 0o755)
	os.WriteFile(filepath.Join(env.ProjectDir, ".claude", "settings.local.json"), []byte(`{"spinnerTipsEnabled": false}`), 0o644)
	nm, _ := m.gotoTab(tConfig)
	m = nm.(model)
	if m.cfg.targets[m.cfg.sel].ID != "code-user" {
		t.Fatal("your settings stay the first file")
	}
	for m.cfg.targets[m.cfg.sel].ID != "code-effective" {
		before := m.cfg.sel
		m, _ = key(m, "j")
		if m.cfg.sel == before {
			t.Fatal("effective settings are listed")
		}
	}
	m, _ = key(m, "enter")
	m = selectCfg(t, m, "spinnerTipsEnabled")
	v := m.View()
	for _, want := range []string{"false", "project (only you)", "wins over: your settings (true)"} {
		if !strings.Contains(v, want) {
			t.Fatalf("effective view misses %q:\n%s", want, v)
		}
	}
	m, _ = key(m, " ")
	m, _ = key(m, "d")
	b, _ := os.ReadFile(filepath.Join(env.ProjectDir, ".claude", "settings.local.json"))
	if string(b) != `{"spinnerTipsEnabled": false}` {
		t.Fatalf("the effective view never writes: %s", b)
	}
}
