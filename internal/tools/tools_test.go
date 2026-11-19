package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
)

func TestDetectLinksMCPAndHooks(t *testing.T) {
	r := t.TempDir()
	env := paths.Env{GOOS: "linux", Home: r, ClaudeDir: filepath.Join(r, ".claude"), ClaudeJSON: filepath.Join(r, ".claude.json")}
	os.MkdirAll(env.ClaudeDir, 0o755)
	os.WriteFile(env.ClaudeJSON, []byte(`{"mcpServers":{"fs":{"command":"npx","args":["-y","x@1.0.0"]},"py":{"command":"C:/tools/uvx.exe","args":["y==1"]},"web":{"type":"http","url":"https://x/mcp"}}}`), 0o644)
	os.WriteFile(filepath.Join(env.ClaudeDir, "settings.json"), []byte(`{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"rtk hook claude"}]}]}}`), 0o644)
	t.Setenv("PATH", "") // nothing installed
	ts := Detect(context.Background(), inventory.Load(env))
	by := map[string]Tool{}
	for _, x := range ts {
		by[x.Name] = x
	}
	if by["npx"].Found || len(by["npx"].UsedBy) != 1 || by["npx"].UsedBy[0] != "MCP fs" {
		t.Errorf("npx: %+v", by["npx"])
	}
	if len(by["uv / uvx"].UsedBy) != 1 {
		t.Errorf("uvx with a Windows path must be recognised: %+v", by["uv / uvx"])
	}
	if len(by["rtk"].UsedBy) != 1 {
		t.Errorf("rtk hook: %+v", by["rtk"])
	}
	if len(ts[0].UsedBy) == 0 {
		t.Error("tools that something depends on must come first")
	}
	if Resolvable("definitely-not-a-command-xyz") || !Resolvable("") {
		t.Error("Resolvable")
	}
}
