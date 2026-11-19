package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/graph"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
)

// w in the Graph tab writes the Markdown and the web page and opens the page.
func TestGraphWriteOpensPage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var opened string
	old := openFolder
	openFolder = func(p string) error { opened = p; return nil }
	defer func() { openFolder = old }()

	env := paths.Env{GOOS: "linux", Home: home, ClaudeDir: filepath.Join(home, ".claude")}
	g := graph.Build(graph.Input{Env: env, Inv: inventory.Inventory{}, Version: "2.1.0"})
	m := newModel(env, "test")
	m.w, m.h, m.loaded, m.tab = 140, 40, true, tGraph
	nm, _ := m.Update(graphMsg{g: g})
	m = nm.(model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})
	m = nm.(model)
	page := filepath.Join(home, "catude-graph.html")
	if opened != page {
		t.Fatalf("the page must be opened, got %q (status %q)", opened, m.status)
	}
	b, err := os.ReadFile(page)
	if err != nil || !strings.Contains(string(b), `<pre class="mermaid">`) || !strings.Contains(string(b), "flowchart LR") || !strings.Contains(string(b), `securityLevel: "strict"`) {
		t.Fatalf("page: %v\n%s", err, b)
	}
	if md, err := os.ReadFile(filepath.Join(home, "catude-graph.md")); err != nil || !strings.Contains(string(md), "```mermaid") {
		t.Fatalf("markdown: %v", err)
	}
}

// In Graph, the arrows change tab like everywhere else (back to Sessions, forward to the next visible tab); nodes open
// and close with enter, + and -.
func TestGraphArrowsChangeTab(t *testing.T) {
	env := paths.Env{GOOS: "linux", Home: t.TempDir()}
	inv := inventory.Inventory{Items: []inventory.Item{{Kind: inventory.Skill, Name: "tdd", Scope: "user"}}}
	g := graph.Build(graph.Input{Env: env, Inv: inv, Version: "2.1.0"})
	m := newModel(env, "test")
	m.w, m.h, m.loaded = 140, 40, true
	nm, _ := m.Update(graphMsg{g: g})
	m = nm.(model)
	for _, k := range []tea.KeyMsg{{Type: tea.KeyLeft}, {Type: tea.KeyShiftTab}} {
		m.tab = tGraph
		nm, _ = m.Update(k)
		if got := nm.(model).tab; got != tSessions {
			t.Errorf("%v in Graph must go back to Sessions, got %s", k, tabNames[got])
		}
	}
	m.tab = tGraph
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := nm.(model).tab; got != tMCP {
		t.Errorf("→ in Graph must go forward, got %s", tabNames[got])
	}
	m.tab = tGraph
	rows := m.graphRows()
	open := m.gr.open[rows[0].key]
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-")})
	m = nm.(model)
	if m.gr.open[rows[0].key] == open && open {
		t.Error("- closes the selected node")
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	if !nm.(model).gr.open[rows[0].key] {
		t.Error("+ opens the selected node")
	}
}

// Labels are escaped in the page: a skill named like markup cannot inject HTML.
func TestMermaidHTMLEscapes(t *testing.T) {
	var g graph.Graph
	g = graph.Build(graph.Input{Inv: inventory.Inventory{Items: []inventory.Item{{Kind: inventory.Skill, Name: `<script>alert(1)</script>`, Scope: "user"}}}})
	page := g.MermaidHTML()
	if strings.Contains(page, "<script>alert") {
		t.Fatal("labels must be escaped")
	}
}
