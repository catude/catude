package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/i18n"
	"github.com/catude/catude/internal/live"
	"github.com/catude/catude/internal/paths"
)

func liveSample() []live.Session {
	now := time.Now()
	return []live.Session{
		{PID: 1, Name: "shop-01", Title: "Fix the checkout", Cwd: `C:\work\shop`, Entrypoint: "cli", Status: "busy", Model: "claude-opus-5-5",
			Effort: "high", Version: "2.1.325", Context: 400000, Activity: now, Todo: live.Todo{Done: 1, Total: 3, Current: "Fixing the total"},
			Steps: []live.Step{
				{Tool: "Read", Detail: "cart.go", At: now.Add(-50 * time.Second), Done: true},
				{Tool: "Agent", Detail: "Explore: find callers", At: now.Add(-20 * time.Second),
					Sub: []live.Step{{Tool: "Grep", Detail: "Total(", At: now.Add(-5 * time.Second)}}},
			}},
		{PID: 2, Name: "api-07", Title: "Invoices", Cwd: "/srv/api", Entrypoint: "claude-vscode", Status: "idle", Model: "claude-sonnet-5-5",
			Activity: now.Add(-9 * time.Minute), LastText: "Done: deployed 1.4.2",
			Steps: []live.Step{{Tool: "Bash", Detail: "Deploy", At: now.Add(-10 * time.Minute), Done: true}}},
	}
}

// The Live tab draws one card per running session: what it is, what it is doing now, its last steps
// and its subagents, with every line fitting the card.
func TestLiveView(t *testing.T) {
	m := newModel(paths.Env{ClaudeDir: t.TempDir()}, "test")
	m.w, m.h, m.loaded, m.tab = 140, 40, true, tLive
	m.lv.list, m.lv.at = liveSample(), time.Now()
	out := m.liveView(36)
	plain := stripANSI(out)
	if os.Getenv("SHOW") != "" {
		t.Log("\n" + out)
	}
	for _, want := range []string{"Live sessions", "2 open · 1 working", "shop-01 · Fix the checkout", "working", "terminal · Opus 5.5 high",
		"to-do 1/3", "Fixing the total", "Read", "cart.go", "Agent", "subagent", "Grep", "api-07", "VS Code", "waiting for you"} {
		if !strings.Contains(plain, want) {
			t.Errorf("missing %q in:\n%s", want, plain)
		}
	}
	for _, l := range strings.Split(out, "\n") {
		if lipgloss.Width(l) > m.w-6 {
			t.Errorf("line wider than the pane (%d): %q", lipgloss.Width(l), stripANSI(l))
		}
	}
}

// Translated, the cards keep their width even with double-width characters.
func TestLiveViewTranslated(t *testing.T) {
	if _, err := i18n.Catalog("ja"); err != nil {
		t.Skip("no Japanese catalog yet")
	}
	defer i18n.Set("en")
	i18n.Set("ja")
	m := newModel(paths.Env{ClaudeDir: t.TempDir()}, "test")
	m.w, m.h, m.loaded, m.tab = 120, 40, true, tLive
	m.lv.list, m.lv.at = liveSample(), time.Now()
	out := m.liveView(36)
	if strings.Contains(stripANSI(out), "waiting for you") {
		t.Error("the Japanese view still shows English")
	}
	for _, l := range strings.Split(out, "\n") {
		if lipgloss.Width(l) > m.w-6 {
			t.Errorf("line wider than the pane (%d): %q", lipgloss.Width(l), stripANSI(l))
		}
	}
	if os.Getenv("SHOW") != "" {
		t.Log("\n" + out + "\n" + m.View())
	}
}
