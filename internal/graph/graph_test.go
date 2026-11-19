package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catude/catude/internal/claudecfg"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/procs"
	"github.com/catude/catude/internal/tools"
)

func TestBuildLinksEverythingAndLeaksNothing(t *testing.T) {
	r := t.TempDir()
	env := paths.Env{GOOS: "linux", Home: r, ClaudeDir: filepath.Join(r, ".claude"), ClaudeJSON: filepath.Join(r, ".claude.json"), ProjectDir: r}
	w := func(p, s string) { os.MkdirAll(filepath.Dir(p), 0o755); os.WriteFile(p, []byte(s), 0o644) }
	secret := "ghp_" + strings.Repeat("Zq9X", 10)
	w(filepath.Join(env.ClaudeDir, "skills", "tdd", "SKILL.md"), "---\nname: tdd\ndescription: Test first. Use when coding.\n---\nx\n")
	w(env.ClaudeJSON, `{"mcpServers":{"db":{"command":"uvx","args":["pg==1.0.0"],"env":{"DB_PASSWORD":"`+secret+`","DB_HOST":"localhost"}},"web":{"type":"http","url":"https://mcp.example.com/x","headers":{"Authorization":"${TOKEN}"}}}}`)
	w(filepath.Join(env.ClaudeDir, "settings.json"), `{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"rtk hook claude"}]}]}}`)
	inv := inventory.Load(env)
	g := Build(Input{Env: env, Version: "2.1.0", Inv: inv, Targets: claudecfg.Targets(env),
		Tools:   []tools.Tool{{Name: "uv / uvx", Bins: []string{"uvx", "uv"}, UsedBy: []string{"MCP db"}}, {Name: "rtk", Bins: []string{"rtk"}, Found: true, UsedBy: []string{"hook"}}},
		Procs:   []procs.Proc{{PID: 10, PPID: 1, Name: "claude", Root: true}, {PID: 11, PPID: 10, Name: "uvx", Cmd: "uvx pg==1.0.0"}},
		Folders: map[string]int64{filepath.Join(env.ClaudeDir, "projects"): 1 << 20}})
	has := func(from, rel, to string) bool {
		for _, e := range g.Edges {
			if strings.HasPrefix(e.From, from) && e.Rel == rel && strings.HasPrefix(e.To, to) {
				return true
			}
		}
		return false
	}
	for _, c := range [][3]string{{"app:code", "reads", "config:"}, {"config:", "defines MCP", "mcp:db"}, {"mcp:db", "starts", "tool:uv / uvx"},
		{"mcp:db", "needs", "env:DB_PASSWORD"}, {"mcp:web", "connects to", "host:mcp.example.com"}, {"hook:", "runs", "tool:rtk"},
		{"app:code", "has skill", "skill:tdd"}, {"app:code", "running now", "process:10"}, {"process:10", "started", "process:11"}, {"app:code", "stores", "folder:"}} {
		if !has(c[0], c[1], c[2]) {
			t.Errorf("missing relation %v", c)
		}
	}
	if n, _ := g.Node("env:DB_PASSWORD"); !n.Warn {
		t.Error("a secret written in plain text must be flagged")
	}
	if n, _ := g.Node("env:Authorization"); n.Warn {
		t.Error("a ${VAR} reference is fine")
	}
	if n, _ := g.Node("tool:uv / uvx"); !n.Warn {
		t.Error("a missing tool that something needs must be flagged")
	}
	for name, out := range map[string]string{"mermaid": g.Mermaid(false), "mermaid-full": g.Mermaid(true), "dot": g.DOT()} {
		if strings.Contains(out, secret) || strings.Contains(out, "localhost") {
			t.Errorf("%s export leaked a value", name)
		}
		if name != "dot" && strings.Contains(out, r) {
			t.Errorf("%s export leaked a local path", name)
		}
	}
	if !strings.HasPrefix(g.Mermaid(false), "flowchart LR") {
		t.Error("mermaid header")
	}
}
