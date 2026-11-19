package tui

import (
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/paths"
)

// run drains a command tree synchronously (for end-to-end checks only).
func drain(m model, cmd tea.Cmd) model {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if b, ok := msg.(tea.BatchMsg); ok {
		for _, c := range b {
			m = drain(m, c)
		}
		return m
	}
	if _, ok := msg.(spinMsg); ok || msg == nil {
		return m
	}
	nm, next := m.Update(msg)
	return drain(nm.(model), next)
}

// LR_E2E=1 go test -run E2E -v ./internal/tui  (uses the real machine and the real claude)
func TestE2ERealMachine(t *testing.T) {
	if os.Getenv("LR_E2E") == "" {
		t.Skip()
	}
	wd, _ := os.Getwd()
	m := newModel(paths.Detect(wd), "e2e")
	m.w, m.h = 150, 48
	t0 := time.Now()
	nm, cmd := m.gotoTab(tStorage)
	m = drain(nm.(model), cmd)
	t.Logf("storage measured in %v", time.Since(t0))
	t.Log("\n" + m.View())
	m.ctx.at = time.Time{} // force a fresh measurement
	t0 = time.Now()
	nm, cmd = m.gotoTab(tContext)
	m = drain(nm.(model), cmd)
	t.Logf("context measured in %v", time.Since(t0))
	t.Log("\n" + m.View())
}
