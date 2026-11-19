package agentmap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
)

func put(t *testing.T, p, s string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Definitions and runs are joined: usage is de-duplicated per response, parents are named, defined agents
// that never ran are listed as unused, built-in agents appear without a definition.
func TestBuild(t *testing.T) {
	c := t.TempDir()
	sub := filepath.Join(c, "projects", "-w-shop", "sess-1", "subagents")
	put(t, filepath.Join(sub, "agent-a1.meta.json"), `{"agentType":"reviewer","description":"Review the diff","spawnDepth":1,"requestShape":"foreground","finished":true}`)
	line := func(ts, id, req string, in, out int, tool string) string {
		content := `[{"type":"text","text":"ok"}]`
		if tool != "" {
			content = `[{"type":"tool_use","name":"` + tool + `","input":{}}]`
		}
		return `{"type":"assistant","cwd":"/w/shop","timestamp":"` + ts + `","requestId":"` + req + `","message":{"id":"` + id + `","model":"claude-sonnet-5","content":` + content +
			`,"usage":{"input_tokens":` + itoa(in) + `,"cache_creation_input_tokens":0,"cache_read_input_tokens":100,"output_tokens":` + itoa(out) + `}}}`
	}
	put(t, filepath.Join(sub, "agent-a1.jsonl"), strings.Join([]string{
		`{"type":"user","cwd":"/w/shop","timestamp":"2026-11-19T10:00:00Z","message":{"content":"go"}}`,
		line("2026-11-19T10:00:05Z", "m1", "r1", 10, 5, "Read"),
		line("2026-11-19T10:00:05Z", "m1", "r1", 10, 5, "Read"), // the same response written twice
		line("2026-11-19T10:01:00Z", "m2", "r2", 20, 7, "Grep"),
	}, "\n")+"\n")
	put(t, filepath.Join(sub, "agent-b2.meta.json"), `{"agentType":"Explore","isBuiltIn":true,"description":"Find callers","parentAgentId":"a1","spawnDepth":2,"requestShape":"background","stoppedByUser":true}`)
	put(t, filepath.Join(sub, "agent-b2.jsonl"), line("2026-11-19T10:00:30Z", "m3", "r3", 1, 1, "Glob")+"\n")
	put(t, filepath.Join(sub, "agent-bad.meta.json"), `not json`)

	inv := inventory.Inventory{Items: []inventory.Item{
		{Kind: inventory.Agent, Name: "reviewer", Scope: paths.ScopeUser, Path: "/u/reviewer.md"},
		{Kind: inventory.Agent, Name: "reviewer", Scope: paths.ScopeProject, Path: "/p/reviewer.md"},
		{Kind: inventory.Agent, Name: "docs-writer", Scope: paths.ScopeUser},
	}}
	m := Build(c, inv, 36500)
	if m.Runs != 2 || len(m.Agents) != 3 {
		t.Fatalf("runs %d agents %d: %+v", m.Runs, len(m.Agents), m.Agents)
	}
	rev := m.Agents[0]
	if rev.Name != "reviewer" && m.Agents[1].Name == "reviewer" {
		rev = m.Agents[1]
	}
	if rev.Defined == nil || rev.Defined.Scope != paths.ScopeProject {
		t.Errorf("the project definition wins over yours: %+v", rev.Defined)
	}
	if rev.Usage.Input != 30 || rev.Usage.Output != 12 || rev.Runs[0].Responses != 2 {
		t.Errorf("usage counted once per response: %+v (%d responses)", rev.Usage, rev.Runs[0].Responses)
	}
	if rev.Tools["Read"] != 2 || rev.Tools["Grep"] != 1 || rev.Projects["/w/shop"] != 1 || rev.LaunchedBy["main"] != 1 {
		t.Errorf("tools, projects, launched by: %v %v %v", rev.Tools, rev.Projects, rev.LaunchedBy)
	}
	if rev.Avg.Seconds() != 60 {
		t.Errorf("average time of finished runs: %v", rev.Avg)
	}
	var ex Agent
	for _, a := range m.Agents {
		if a.Name == "Explore" {
			ex = a
		}
	}
	if !ex.BuiltIn || ex.LaunchedBy["reviewer"] != 1 || ex.Stopped != 1 || !ex.Runs[0].Background {
		t.Errorf("built-in, launched by another agent, stopped, background: %+v", ex)
	}
	if u := m.Unused(); len(u) != 1 || u[0].Name != "docs-writer" {
		t.Errorf("unused: %+v", u)
	}
}

func itoa(n int) string {
	s := ""
	if n == 0 {
		return "0"
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
