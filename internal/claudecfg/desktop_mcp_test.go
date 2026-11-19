package claudecfg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Copying a profile's MCP servers into Claude Desktop: local programs are added, remote servers and
// ${VAR} secrets are left out, a different server with the same name is kept unless asked, the rest of
// the file and its key order stay, and a backup is made.
func TestCopyToDesktopMCP(t *testing.T) {
	p := filepath.Join(t.TempDir(), "claude_desktop_config.json")
	os.WriteFile(p, []byte("{\n  \"preferences\": {\"x\": 1},\n  \"mcpServers\": {\n    \"files\": {\"command\": \"node\", \"args\": [\"old.js\"]},\n    \"same\": {\"command\": \"same\"}\n  }\n}\n"), 0o600)
	tg := Target{ID: "desktop:" + p, App: "Claude Desktop", Path: p}
	servers := map[string]json.RawMessage{
		"git":    json.RawMessage(`{"command": "uvx", "args": ["mcp-server-git"]}`),
		"files":  json.RawMessage(`{"command": "node", "args": ["new.js"]}`),
		"same":   json.RawMessage(`{"command": "same"}`),
		"remote": json.RawMessage(`{"type": "http", "url": "https://example.com/mcp"}`),
		"gh":     json.RawMessage(`{"command": "npx", "env": {"GITHUB_TOKEN": "${GITHUB_TOKEN}"}}`),
	}
	plan, err := PlanDesktopMCP(tg, servers, false)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range plan {
		got[c.Name] = c.Action
	}
	want := map[string]string{"git": "add", "files": "keep", "same": "same", "remote": "skip", "gh": "skip"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %s, want %s", k, got[k], v)
		}
	}
	_, backup, err := CopyToDesktopMCP(tg, servers, false)
	if err != nil || backup == "" {
		t.Fatalf("copy: %v (backup %q)", err, backup)
	}
	b, _ := os.ReadFile(p)
	s := string(b)
	if !strings.Contains(s, "mcp-server-git") || !strings.Contains(s, "old.js") || strings.Contains(s, "example.com") || strings.Contains(s, "GITHUB_TOKEN") {
		t.Fatalf("result:\n%s", s)
	}
	if strings.Index(s, "preferences") > strings.Index(s, "mcpServers") {
		t.Error("the key order of the file stays")
	}
	if _, _, err := CopyToDesktopMCP(tg, servers, true); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), "new.js") {
		t.Error("--replace overwrites a different server with the same name")
	}
	if _, err := PlanDesktopMCP(Target{ID: "code-user", Path: p}, servers, false); err == nil {
		t.Error("only Claude Desktop configs take MCP servers this way")
	}
}
