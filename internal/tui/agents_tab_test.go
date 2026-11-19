package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
)

// The Agents tab shows the map: runs per agent, who launched them, and the definition of the selected one;
// e and space act on the definition as in the other tabs.
func TestAgentsTabShowsTheMap(t *testing.T) {
	m, env := testModel(t)
	sub := filepath.Join(env.ClaudeDir, "projects", "-w", "s1", "subagents")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "agent-a.meta.json"), []byte(`{"agentType":"Explore","isBuiltIn":true,"description":"Find the callers","finished":true}`), 0o644)
	os.WriteFile(filepath.Join(sub, "agent-a.jsonl"), []byte(`{"type":"assistant","cwd":"/w","timestamp":"2026-11-19T10:00:00Z","requestId":"r","message":{"id":"m","model":"claude-sonnet-5","content":[{"type":"tool_use","name":"Grep"}],"usage":{"input_tokens":1200,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":300}}}`+"\n"), 0o644)
	os.Chtimes(filepath.Join(sub, "agent-a.meta.json"), time.Now(), time.Now())
	m.inv = inventory.Inventory{Items: []inventory.Item{{Kind: inventory.Agent, Name: "reviewer", Scope: paths.ScopeUser, Path: filepath.Join(env.ClaudeDir, "agents", "reviewer.md"), Editable: true}}}
	m.tab = tAgents
	cmd := m.ensureAgents()
	if cmd == nil {
		t.Fatal("opening the tab reads the runs in the background")
	}
	nm, _ := m.Update(cmd())
	m = nm.(model)
	v := m.View()
	for _, want := range []string{"Agent map", "Explore", "reviewer", "Find the callers", "Grep 1", "your conversation"} {
		if !strings.Contains(v, want) {
			t.Fatalf("agents view misses %q:\n%s", want, v)
		}
	}
	m, _ = key(m, "j")
	if it := m.selectedItem(); it == nil || it.Name != "reviewer" {
		t.Fatalf("the selected agent's definition is the item e and space act on: %+v", it)
	}
	if !strings.Contains(m.View(), "did not run") {
		t.Error("a defined agent that never ran says so")
	}
}
