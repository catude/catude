package sessions

import (
	"os"
	"path/filepath"
	"testing"
)

// Claude Desktop's own title and archive state are found by the Claude Code session id.
func TestDesktopSessions(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "claude-code-sessions", "acct", "org", "local_x.json")
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte(`{"sessionId":"local_x","cliSessionId":"11111111-2222-3333-4444-555555555555","title":"Fix  login","isArchived":true}`), 0o644)
	os.WriteFile(filepath.Join(filepath.Dir(p), "local_bad.json"), []byte(`{"cliSessionId":"../etc"}`), 0o644)
	m := DesktopSessions([]string{d})
	if s, ok := m["11111111-2222-3333-4444-555555555555"]; !ok || s.Title != "Fix login" || !s.Archived || len(m) != 1 {
		t.Fatalf("%+v", m)
	}
}
