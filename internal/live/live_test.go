package live

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeSession(t *testing.T, dir string, pid int, id, status string) {
	t.Helper()
	b := fmt.Sprintf(`{"pid":%d,"sessionId":%q,"cwd":"C:\\work\\shop","startedAt":%d,"version":"2.1.325","kind":"interactive","entrypoint":"cli","name":"shop-01","status":%q,"updatedAt":%d}`,
		pid, id, time.Now().Add(-time.Hour).UnixMilli(), status, time.Now().UnixMilli())
	if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d.json", pid)), []byte(b), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSessionsAndActivity(t *testing.T) {
	root := t.TempDir()
	sdir := filepath.Join(root, "sessions")
	pdir := filepath.Join(root, "projects", "C--work-shop")
	os.MkdirAll(sdir, 0o700)
	os.MkdirAll(pdir, 0o700)
	id := "11111111-2222-3333-4444-555555555555"
	writeSession(t, sdir, 4242, id, "busy")
	writeSession(t, sdir, 4343, "99999999-2222-3333-4444-555555555555", "idle") // its process is gone
	// the .key files next to them are never read
	os.WriteFile(filepath.Join(sdir, "4242.deadbeef.key"), []byte("SECRET"), 0o600)
	now := time.Now().UTC()
	ts := func(d time.Duration) string { return now.Add(d).Format(time.RFC3339Nano) }
	transcript := strings.Join([]string{
		`{"type":"ai-title","aiTitle":"Fix the checkout"}`,
		`{"type":"assistant","timestamp":"` + ts(-50*time.Second) + `","effort":"high","message":{"model":"claude-opus-5-5","usage":{"input_tokens":10,"cache_read_input_tokens":200000,"cache_creation_input_tokens":0},"content":[{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"C:\\work\\shop\\cart.go"}}]}}`,
		`{"type":"user","timestamp":"` + ts(-49*time.Second) + `","message":{"content":[{"type":"tool_result","tool_use_id":"t1"}]}}`,
		`{"type":"assistant","timestamp":"` + ts(-40*time.Second) + `","message":{"model":"claude-opus-5-5","content":[{"type":"tool_use","id":"t2","name":"TodoWrite","input":{"todos":[{"content":"read","status":"completed"},{"content":"fix","status":"in_progress","activeForm":"Fixing the total"},{"content":"test","status":"pending"}]}}]}}`,
		`{"type":"user","timestamp":"` + ts(-39*time.Second) + `","message":{"content":[{"type":"tool_result","tool_use_id":"t2"}]}}`,
		`{"type":"assistant","timestamp":"` + ts(-30*time.Second) + `","message":{"model":"claude-opus-5-5","content":[{"type":"text","text":"**Running** the tests now"}]}}`,
		`{"type":"assistant","timestamp":"` + ts(-20*time.Second) + `","message":{"model":"claude-opus-5-5","content":[{"type":"tool_use","id":"t3","name":"Bash","input":{"command":"go test ./... --token=sk-ant-api03-` + strings.Repeat("A", 90) + `"}}]}}`,
		`{"type":"assistant","timestamp":"` + ts(-10*time.Second) + `","message":{"content":[{"type":"tool_use","id":"t4","name":"mcp__github__create_issue","input":{"title":"x"}}]}}`,
	}, "\n") + "\n"
	os.WriteFile(filepath.Join(pdir, id+".jsonl"), []byte(transcript), 0o600)

	r := &Reader{ClaudeDir: root, Alive: func(pid int) bool { return pid == 4242 }}
	ss := r.Sessions()
	if len(ss) != 1 {
		t.Fatalf("only the session whose process runs is listed: %+v", ss)
	}
	s := ss[0]
	if s.Name != "shop-01" || s.Title != "Fix the checkout" || s.Model != "claude-opus-5-5" || s.Effort != "high" || s.Context != 200010 {
		t.Errorf("session: %+v", s)
	}
	if !s.Working() || s.Todo.Done != 1 || s.Todo.Total != 3 || s.Todo.Current != "Fixing the total" {
		t.Errorf("todo/working: %+v", s.Todo)
	}
	if s.LastText != "Running the tests now" {
		t.Errorf("last text without Markdown: %q", s.LastText)
	}
	if len(s.Steps) != 3 || s.Steps[0].Tool != "Read" || s.Steps[0].Detail != "cart.go" || !s.Steps[0].Done {
		t.Fatalf("steps: %+v", s.Steps)
	}
	if strings.Contains(s.Steps[1].Detail, "sk-ant-") {
		t.Errorf("secrets in commands must be redacted: %q", s.Steps[2].Detail)
	}
	run, ok := s.Running()
	if !ok || run.Tool != "github › create_issue" {
		t.Errorf("running: %+v", run)
	}
	// a refresh with no change reuses the parsed tail
	if again := r.Sessions(); len(again) != 1 || len(again[0].Steps) != 3 {
		t.Error("refresh")
	}
}

func TestToolDetails(t *testing.T) {
	for _, c := range []struct{ tool, in, want string }{
		{"Bash", `{"command":"npm test\nmore","description":"Run the tests"}`, "Run the tests"},
		{"Bash", `{"command":"npm test\nmore"}`, "npm test"},
		{"WebFetch", `{"url":"https://example.com/a/b?q=1"}`, "example.com"},
		{"Agent", `{"description":"find callers","subagent_type":"Explore"}`, "Explore: find callers"},
		{"Grep", `{"pattern":"TODO"}`, "TODO"},
	} {
		if got := detail(c.tool, []byte(c.in)); got != c.want {
			t.Errorf("%s: %q, want %q", c.tool, got, c.want)
		}
	}
}

// Task tools (TaskCreate / TaskUpdate) fill the to-do line like TodoWrite does.
func TestTaskTools(t *testing.T) {
	var tl taskList
	tl.apply("TaskCreate", []byte(`{"subject":"Read the cart","activeForm":"Reading the cart"}`))
	tl.apply("TaskCreate", []byte(`{"subject":"Fix the total","activeForm":"Fixing the total"}`))
	tl.apply("TaskCreate", []byte(`{"subject":"Write tests"}`))
	tl.apply("TaskUpdate", []byte(`{"taskId":"1","status":"completed"}`))
	tl.apply("TaskUpdate", []byte(`{"taskId":"2","status":"in_progress"}`))
	tl.apply("TaskUpdate", []byte(`{"taskId":"9","status":"completed"}`)) // unknown id: ignored
	if got := tl.todo(); got.Done != 1 || got.Total != 3 || got.Current != "Fixing the total" {
		t.Errorf("tasks: %+v", got)
	}
	tl.apply("TaskUpdate", []byte(`{"taskId":"3","status":"deleted"}`))
	if got := tl.todo(); got.Total != 2 {
		t.Errorf("deleted tasks are not counted: %+v", got)
	}
}

// A transcript full of lines in an unknown shape is reported, not shown as an empty session.
func TestUnrecognizedFormat(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.jsonl")
	var b strings.Builder
	for i := 0; i < 8; i++ {
		b.WriteString(`{"kind":"turn","payload":{"text":"hello"}}` + "\n")
	}
	os.WriteFile(p, []byte(b.String()), 0o600)
	var s Session
	parseTail(p, int64(b.Len()), &s)
	if !s.Unrecognized {
		t.Error("an unknown format must be flagged")
	}
	os.WriteFile(p, []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hi"}]}}`+"\n"), 0o600)
	s = Session{}
	parseTail(p, 80, &s)
	if s.Unrecognized {
		t.Error("a known format is not flagged")
	}
}
