package sessions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A conversation gets its summary without AI, and a name given with Rename wins over the generated title.
func TestSummaryAndRename(t *testing.T) {
	p := filepath.Join(t.TempDir(), "11111111-2222-3333-4444-555555555555.jsonl")
	os.WriteFile(p, []byte(strings.Join([]string{
		`{"type":"user","isMeta":true,"message":{"content":"<local-command-caveat>x</local-command-caveat>"}}`,
		`{"type":"user","timestamp":"2026-11-19T10:00:00Z","message":{"content":[{"type":"text","text":"Fix the  refund bug with token ghp_` + strings.Repeat("A", 36) + `"}]}}`,
		`{"type":"ai-title","aiTitle":"Refund bug"}`,
		`{"type":"assistant","timestamp":"2026-11-19T10:00:05Z","requestId":"r1","message":{"id":"m1","model":"claude-opus-5-5","usage":{"input_tokens":1,"output_tokens":1},"content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/w/refund.go"}},{"type":"tool_use","name":"Bash","input":{"command":"go test"}}]}}`,
		`{"type":"user","message":{"content":"second message"}}`,
	}, "\n")), 0o644)
	s, err := Analyze(p)
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "Refund bug" || s.Named || !strings.HasPrefix(s.FirstPrompt, "Fix the refund bug") || strings.Contains(s.FirstPrompt, "ghp_") {
		t.Fatalf("summary: %+v", s)
	}
	if s.Files["/w/refund.go"] != 1 || s.Tools["Bash"] != 1 {
		t.Errorf("files and tools: %v %v", s.Files, s.Tools)
	}
	if err := Rename(p, "  Refunds:   idempotency "); err != nil {
		t.Fatal(err)
	}
	s, _ = Analyze(p)
	if s.Title != "Refunds: idempotency" || !s.Named {
		t.Fatalf("renamed: %q %v", s.Title, s.Named)
	}
	if Rename(p, " ") == nil || Rename(filepath.Join(filepath.Dir(p), "bad name.jsonl"), "x") == nil {
		t.Error("empty names and odd files are refused")
	}
}
