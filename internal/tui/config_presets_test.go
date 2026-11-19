package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// p in the Config tab opens the presets; enter previews, enter again applies to the selected file.
func TestConfigPresetsPanel(t *testing.T) {
	m, env := testModel(t)
	file := filepath.Join(env.ClaudeDir, "settings.json")
	os.WriteFile(file, []byte("{\n  \"model\": \"opus\"\n}\n"), 0o644)
	nm, _ := m.gotoTab(tConfig)
	m = nm.(model)
	m, _ = key(m, "p")
	if !m.cfg.ps.open {
		t.Fatal("p opens the presets")
	}
	// find "Quieter UI"-like preset by searching
	for m.cfg.ps.sel < len(m.visiblePresets())-1 && !strings.Contains(m.visiblePresets()[m.cfg.ps.sel].ID, "updates-stable") {
		m, _ = key(m, "j")
	}
	v := stripANSI(m.View())
	if os.Getenv("SHOW") != "" {
		t.Log("\n" + v)
	}
	for _, want := range []string{"Configuration presets", "Sources", "Changes in"} {
		if !strings.Contains(v, want) {
			t.Fatalf("missing %q:\n%s", want, v)
		}
	}
	m, _ = key(m, "enter")
	if m.cfg.ps.armed == "" {
		t.Fatal("first enter arms")
	}
	nm2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm2.(model)
	b, _ := os.ReadFile(file)
	if !strings.Contains(string(b), `"autoUpdatesChannel": "stable"`) || !strings.Contains(string(b), `"model": "opus"`) {
		t.Fatalf("after apply:\n%s\nstatus %s", b, m.status)
	}
	m, _ = key(m, "esc")
	if m.cfg.ps.open {
		t.Error("esc closes the presets")
	}
}
