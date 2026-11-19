package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/catude/catude/internal/sessions"
)

// n names the selected conversation in the Sessions list; the name is written as /rename writes it.
func TestRenameSession(t *testing.T) {
	m, env := testModel(t)
	p := filepath.Join(env.ClaudeDir, "projects", "-w", "11111111-2222-3333-4444-555555555555.jsonl")
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte(`{"type":"ai-title","aiTitle":"Generated"}`+"\n"), 0o644)
	s, _ := sessions.Analyze(p)
	s.Path, s.ID = p, "11111111-2222-3333-4444-555555555555"
	m.tab, m.level, m.sessList = tSessions, 1, []sessions.Session{s}
	m, _ = key(m, "n")
	if !m.ren.active {
		t.Fatal("n starts naming")
	}
	m = typeText(m, "Refunds")
	m, _ = key(m, "enter")
	if got, _ := sessions.Analyze(p); got.Title != "Refunds" || !got.Named || m.sessList[0].Title != "Refunds" {
		t.Fatalf("renamed: %+v · %s", got, m.status)
	}
}
