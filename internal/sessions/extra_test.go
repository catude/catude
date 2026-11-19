package sessions

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const transcript = `{"type":"user","uuid":"u1","cwd":"/work/game","timestamp":"2026-11-17T10:00:00Z","message":{"role":"user","content":"Design the first level of the platformer"}}
{"type":"assistant","uuid":"a1","cwd":"/work/game","timestamp":"2026-11-17T10:00:05Z","message":{"id":"m1","role":"assistant","model":"claude-opus-5-5","content":[{"type":"text","text":"Here is a level with three jumps 🎮"}],"usage":{"input_tokens":10,"output_tokens":5}}}
{"type":"assistant","uuid":"a1","cwd":"/work/game","timestamp":"2026-11-17T10:00:05Z","message":{"id":"m1","role":"assistant","content":[{"type":"text","text":"Here is a level with three jumps 🎮"}]}}
{"type":"assistant","uuid":"a2","isSidechain":true,"message":{"role":"assistant","content":[{"type":"text","text":"subagent noise"}]}}
{"type":"ai-title","aiTitle":"Platformer level design"}
`

func write(t *testing.T, dir, name, s string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte(s), 0o644)
	return p
}

func TestQuickInfoMessagesMarkdown(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, "proj/abc.jsonl", transcript)
	cwd, title := QuickInfo(p)
	if cwd != "/work/game" || title != "Platformer level design" {
		t.Fatalf("quick info: %q %q", cwd, title)
	}
	ms, err := Messages(p, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 || ms[0].Role != "user" || ms[1].Role != "assistant" {
		t.Fatalf("messages (duplicates and subagents must be dropped): %+v", ms)
	}
	s, _ := Analyze(p)
	md := Markdown(s, ms)
	if !strings.Contains(md, "## You") || !strings.Contains(md, "three jumps") || strings.Contains(md, "subagent noise") {
		t.Fatalf("markdown:\n%s", md)
	}
}

func TestSearchAndTrash(t *testing.T) {
	d := t.TempDir()
	claude := filepath.Join(d, ".claude")
	p := write(t, claude, "projects/proj/abc.jsonl", transcript)
	write(t, claude, "projects/proj/other.jsonl", `{"type":"user","uuid":"x","message":{"role":"user","content":"unrelated"}}`+"\n")
	write(t, claude, "projects/proj/abc/subagents/s.jsonl", "{}\n")
	write(t, claude, "file-history/abc/v1", "x")
	ps, _ := ListProjects(filepath.Join(claude, "projects"))
	hits, err := Search(context.Background(), ps, "THREE jumps", 10)
	if err != nil || len(hits) != 1 || hits[0].Path != p || !strings.Contains(hits[0].Snippet, "three jumps") || hits[0].Title != "Platformer level design" {
		t.Fatalf("search: %+v %v", hits, err)
	}
	if h, _ := Search(context.Background(), ps, "subagent noise", 10); len(h) != 0 {
		t.Error("subagent lines must not match")
	}
	dir, err := Trash(claude, p, filepath.Join(d, "trash"))
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{p, filepath.Join(claude, "projects/proj/abc"), filepath.Join(claude, "file-history/abc")} {
		if _, err := os.Stat(gone); err == nil {
			t.Errorf("%s still there", gone)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "RESTORE.txt")); err != nil {
		t.Error("trash must explain how to restore")
	}
	if _, err := os.Stat(filepath.Join(claude, "projects/proj/other.jsonl")); err != nil {
		t.Error("another session was touched")
	}
}

func TestValidIDAndLocalPath(t *testing.T) {
	for _, ok := range []string{"abc", "0f3c9a2e-1b4d-4c5e-8f6a-7b8c9d0e1f2a", "agent_1"} {
		if !ValidID(ok) {
			t.Errorf("%q is a valid id", ok)
		}
	}
	for _, bad := range []string{"", "--settings=evil.json", "-p", "--dangerously-skip-permissions", ".", "..", "a/b", `a\b`, "a b", "a;b", strings.Repeat("a", 65)} {
		if ValidID(bad) {
			t.Errorf("%q must not be accepted as a session id", bad)
		}
	}
	d := t.TempDir()
	if _, err := Trash(d, filepath.Join(d, "--settings=x.jsonl"), filepath.Join(d, "trash")); err == nil {
		t.Error("Trash must refuse a name that is not a session id")
	}
	unc := string([]byte{'\\', '\\'}) + "evil" + string([]byte{'\\'}) + "share"
	if LocalPath(unc) || LocalPath("//evil/share") || LocalPath("") || !LocalPath(`C:\work`) || !LocalPath("/home/a") {
		t.Error("LocalPath")
	}
}
