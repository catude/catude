package tui

import (
	"context"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/claudecfg"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/procs"
	"github.com/catude/catude/internal/profiles"
	"github.com/catude/catude/internal/tools"
)

// LR_PERF=1 go test -run GraphPerf -v ./internal/tui  (real machine)
func TestGraphPerfRealMachine(t *testing.T) {
	if os.Getenv("LR_PERF") == "" {
		t.Skip()
	}
	wd, _ := os.Getwd()
	env := paths.Detect(wd)
	inv := inventory.Load(env)
	for _, step := range []struct {
		name string
		fn   func()
	}{
		{"tools.Detect", func() { tools.Detect(context.Background(), inv) }},
		{"claudecfg.Targets", func() { claudecfg.Targets(env) }},
		{"profiles.List", func() { profiles.List() }},
		{"CodeInstances", func() { env.CodeInstances() }},
		{"procs.List", func() { procs.List(context.Background()) }},
	} {
		t1 := time.Now()
		step.fn()
		t.Logf("%-18s %v", step.name, time.Since(t1))
	}
	t0 := time.Now()
	buildGraph(context.Background(), env, inv, nil, false)
	t.Logf("first step (no processes) %v", time.Since(t0))
	t0 = time.Now()
	g := BuildGraph(context.Background(), env, inv, true)
	t.Logf("BuildGraph %v: %d nodes, %d edges", time.Since(t0), len(g.Nodes), len(g.Edges))
	m := newModel(env, "perf")
	m.w, m.h, m.inv, m.loaded = 160, 45, inv, true
	nm, _ := m.Update(graphMsg{g: g})
	m = nm.(model)
	m.tab = tGraph
	t0 = time.Now()
	_ = m.View()
	t.Logf("first View %v (%d rows)", time.Since(t0), len(m.graphRows()))
	t0 = time.Now()
	for i := 0; i < 20; i++ {
		nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = nm.(model)
		_ = m.View()
	}
	t.Logf("20 × (down + View) %v", time.Since(t0))
	t0 = time.Now()
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	m = nm.(model)
	_ = m.View()
	t.Logf("expand all + View %v (%d rows)", time.Since(t0), len(m.graphRows()))
	t0 = time.Now()
	for i := 0; i < 20; i++ {
		nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = nm.(model)
		_ = m.View()
	}
	t.Logf("20 × (down + View) expanded %v", time.Since(t0))
	t0 = time.Now()
	s := m.gr.g.Mermaid(false)
	t.Logf("Mermaid %v (%d bytes)", time.Since(t0), len(s))
}
