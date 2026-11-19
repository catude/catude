package claudecfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catude/catude/internal/paths"
)

// Servers and skills switched off for one project land where Claude Code reads them, keep the rest of the
// files, and come back on cleanly; ~/.claude.json is never touched while Claude Code runs.
func TestProjectToggles(t *testing.T) {
	d := t.TempDir()
	proj := filepath.Join(d, "shop")
	env := paths.Env{ProjectDir: proj, ClaudeJSON: filepath.Join(d, ".claude.json")}
	os.MkdirAll(filepath.Join(proj, ".claude"), 0o755)
	os.WriteFile(env.ClaudeJSON, []byte(`{"numStartups": 3, "projects": {"`+strings.ReplaceAll(proj, `\`, `\\`)+`": {"allowedTools": []}}}`), 0o600)
	os.WriteFile(filepath.Join(proj, ".claude", "settings.local.json"), []byte(`{"model": "sonnet"}`), 0o600)

	old := CodeRunning
	defer func() { CodeRunning = old }()
	CodeRunning = func() int { return 1 }
	if _, err := SetMCPOff(env, "github", false, true); err == nil {
		t.Fatal("~/.claude.json must not change while Claude Code runs")
	}
	CodeRunning = func() int { return 0 }
	if _, err := SetMCPOff(env, "github", false, true); err != nil {
		t.Fatal(err)
	}
	if _, err := SetMCPOff(env, "docs", true, true); err != nil {
		t.Fatal(err)
	}
	if _, err := SetSkillOverride(env, "deploy", "off"); err != nil {
		t.Fatal(err)
	}
	o := ProjectOff(env)
	if !o.MCP["github"] || !o.MCPJSON["docs"] || o.Skills["deploy"] != "off" {
		t.Fatalf("off: %+v", o)
	}
	b, _ := os.ReadFile(env.ClaudeJSON)
	if !strings.Contains(string(b), `"numStartups": 3`) || !strings.Contains(string(b), "allowedTools") || strings.Count(string(b), "disabledMcpServers") != 1 {
		t.Fatalf("~/.claude.json keeps everything else, the list goes under the existing project:\n%s", b)
	}
	if _, err := SetSkillOverride(env, "deploy", "loud"); err == nil {
		t.Error("only documented skillOverrides values")
	}
	SetMCPOff(env, "github", false, false)
	SetMCPOff(env, "docs", true, false)
	SetSkillOverride(env, "deploy", "")
	o = ProjectOff(env)
	if len(o.MCP)+len(o.MCPJSON)+len(o.Skills) != 0 {
		t.Fatalf("back on: %+v", o)
	}
	if b, _ := os.ReadFile(filepath.Join(proj, ".claude", "settings.local.json")); strings.Contains(string(b), "skillOverrides") || strings.Contains(string(b), "disabledMcpjsonServers") || !strings.Contains(string(b), "sonnet") {
		t.Errorf("empty lists are removed, the rest stays:\n%s", b)
	}
}
