package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catude/catude/internal/claudecfg"
	"github.com/catude/catude/internal/ctxcomp"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
)

// x switches a .mcp.json server and a skill off for this project; the list says so; x again goes on.
func TestToggleHere(t *testing.T) {
	m, env := testModel(t)
	os.MkdirAll(filepath.Join(env.ProjectDir, ".claude"), 0o755)
	m.inv = inventory.Inventory{Items: []inventory.Item{
		{Kind: inventory.MCP, Name: "docs", Scope: paths.ScopeProject, Path: filepath.Join(env.ProjectDir, ".mcp.json")},
		{Kind: inventory.Skill, Name: "deploy", Scope: paths.ScopeUser},
	}}
	m.tab = tMCP
	m, _ = key(m, "x")
	if !claudecfg.ProjectOff(env).MCPJSON["docs"] || !strings.Contains(m.View(), "off in this project") {
		t.Fatalf("x switches the .mcp.json server off here: %s\n%s", m.status, m.View())
	}
	m, _ = key(m, "x")
	if claudecfg.ProjectOff(env).MCPJSON["docs"] {
		t.Fatal("x again switches it back on")
	}
	m.tab = tSkills
	for _, want := range []string{"name-only", "user-invocable-only", "off", ""} {
		m, _ = key(m, "x")
		if got := claudecfg.ProjectOff(env).Skills["deploy"]; got != want {
			t.Fatalf("x cycles the documented values: got %q, want %q (%s)", got, want, m.status)
		}
	}
}

// In the Context tab, x switches the selected consumer off for this project.
func TestContextToggle(t *testing.T) {
	m, env := testModel(t)
	os.MkdirAll(filepath.Join(env.ProjectDir, ".claude"), 0o755)
	m.inv = inventory.Inventory{Items: []inventory.Item{{Kind: inventory.Skill, Name: "deploy", Scope: paths.ScopeUser}}}
	m.ctx.comp = &ctxcomp.Composition{Used: 30000, Max: 200000, Skills: []ctxcomp.Row{{Name: "deploy", Tokens: 900}}}
	m.tab = tContext
	m, _ = key(m, "x")
	if claudecfg.ProjectOff(env).Skills["deploy"] != "name-only" || !strings.Contains(m.View(), "name only") {
		t.Fatalf("x in Context: %s", m.status)
	}
}
