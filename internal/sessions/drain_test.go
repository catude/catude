package sessions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The drain analysis: a cache rebuilt after a long pause, subagent spending, a conversation copied into
// another folder counted once, and the context of the first response.
func TestDrain(t *testing.T) {
	root := t.TempDir()
	now := time.Now().Add(-2 * time.Hour).UTC()
	line := func(at time.Time, id string, in, cw, cr, out int, entry string) string {
		return fmt.Sprintf(`{"type":"assistant","cwd":"/w/shop","entrypoint":%q,"version":"2.1.300","timestamp":%q,"requestId":"r-%s","message":{"id":%q,"model":"claude-opus-5-5","usage":{"input_tokens":%d,"cache_creation_input_tokens":%d,"cache_read_input_tokens":%d,"output_tokens":%d}}}`,
			entry, at.Format(time.RFC3339Nano), id, id, in, cw, cr, out)
	}
	main := strings.Join([]string{
		line(now, "a", 100, 30000, 0, 500, "cli"),                    // start: 30.1k of context
		line(now.Add(time.Minute), "b", 50, 1000, 30000, 400, "cli"), // normal turn
		line(now.Add(20*time.Minute), "c", 50, 40000, 0, 300, "cli"), // after a 19 min pause: cache rebuilt
	}, "\n") + "\n"
	p1 := filepath.Join(root, "-w-shop", "s1.jsonl")
	os.MkdirAll(filepath.Join(root, "-w-shop", "s1", "subagents"), 0o755)
	os.WriteFile(p1, []byte(main), 0o644)
	os.WriteFile(filepath.Join(root, "-w-shop", "s1", "subagents", "agent-x.jsonl"), []byte(line(now.Add(2*time.Minute), "d", 9000, 0, 0, 1000, "cli")+"\n"), 0o644)
	// the same conversation forked into another folder: its history is copied
	os.MkdirAll(filepath.Join(root, "-w-other"), 0o755)
	fork := main + line(now.Add(30*time.Minute), "e", 10, 0, 40000, 90, "cli") + "\n" // and goes on
	os.WriteFile(filepath.Join(root, "-w-other", "s2.jsonl"), []byte(fork), 0o644)

	ps, err := ListProjects(root)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := (&UsageCache{}).Usage(context.Background(), ps, 7)
	if err != nil {
		t.Fatal(err)
	}
	d := rep.Drain
	wantFresh := (100 + 30000 + 500) + (50 + 1000 + 400) + (50 + 40000 + 300) + (9000 + 1000) + (10 + 90)
	if d.Fresh != wantFresh {
		t.Errorf("fresh %d, want %d (the copied conversation must count once)", d.Fresh, wantFresh)
	}
	total := 0
	for _, day := range rep.Days {
		total += day.Billable()
	}
	if total != wantFresh {
		t.Errorf("daily totals %d, want %d", total, wantFresh)
	}
	if d.Rebuilds != 1 || d.RebuildTokens != 40000 {
		t.Errorf("rebuilds %d / %d", d.Rebuilds, d.RebuildTokens)
	}
	if d.Subagents != 10000 {
		t.Errorf("subagents %d", d.Subagents)
	}
	if d.Starts != 1 || d.StartAvg != 30100 {
		t.Errorf("starts %d avg %d", d.Starts, d.StartAvg)
	}
	if len(d.Top) < 2 || d.Top[0].Fresh != wantFresh-100 || d.Top[0].Subagents != 10000 || d.Top[0].Project != "/w/shop" {
		t.Errorf("top conversation, subagents included, with its real folder: %+v", d.Top)
	}
	fs := d.Explain(func(n int) string { return fmt.Sprint(n) })
	if len(fs) == 0 || !strings.Contains(fs[0].Text, "rebuilt") && !strings.Contains(fs[0].Text, "One conversation") {
		t.Errorf("explanations: %+v", fs)
	}
}

// Health per day: tool calls of every block of a response, failed tool results and compactions, counted
// once even when a conversation was copied into another folder.
func TestHealth(t *testing.T) {
	root := t.TempDir()
	at := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	body := strings.Join([]string{
		`{"type":"assistant","timestamp":"` + at + `","requestId":"r1","message":{"id":"m1","model":"claude-opus-5-5","usage":{"input_tokens":10,"output_tokens":5},"content":[{"type":"tool_use","name":"Bash"}]}}`,
		`{"type":"assistant","timestamp":"` + at + `","requestId":"r1","message":{"id":"m1","model":"claude-opus-5-5","usage":{"input_tokens":10,"output_tokens":9},"content":[{"type":"tool_use","name":"Read"}]}}`,
		`{"type":"user","uuid":"u1","timestamp":"` + at + `","message":{"content":[{"type":"tool_result","is_error":true,"content":"exit 1"}]}}`,
		`{"type":"system","subtype":"compact_boundary","uuid":"u2","timestamp":"` + at + `"}`,
	}, "\n") + "\n"
	for _, d := range []string{"-a", "-b"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
		os.WriteFile(filepath.Join(root, d, "s.jsonl"), []byte(body), 0o644)
	}
	ps, _ := ListProjects(root)
	rep, err := (&UsageCache{}).Usage(context.Background(), ps, 2)
	if err != nil {
		t.Fatal(err)
	}
	var h HealthDay
	for _, d := range rep.Health {
		h.Responses += d.Responses
		h.ToolCalls += d.ToolCalls
		h.ToolErrors += d.ToolErrors
		h.Compactions += d.Compactions
	}
	if h.Responses != 1 || h.ToolCalls != 2 || h.ToolErrors != 1 || h.Compactions != 1 {
		t.Fatalf("health: %+v", h)
	}
}
