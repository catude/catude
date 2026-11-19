package sessions

import (
	"strings"
	"testing"
)

func asst(id, req string, in, cw, cr, out int, extra string) string {
	return `{"type":"assistant","requestId":"` + req + `","cwd":"/w","timestamp":"2026-11-17T10:00:00.000Z"` + extra +
		`,"message":{"id":"` + id + `","model":"claude-opus-5-5","usage":{"input_tokens":` + itoa(in) +
		`,"cache_creation_input_tokens":` + itoa(cw) + `,"cache_read_input_tokens":` + itoa(cr) + `,"output_tokens":` + itoa(out) + `}}}`
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func scanString(t *testing.T, lines ...string) Session {
	t.Helper()
	var s Session
	if err := Scan(strings.NewReader(strings.Join(lines, "\n")), &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStreamingDuplicatesCountOnce(t *testing.T) {
	s := scanString(t,
		asst("m1", "r1", 10, 100, 0, 5, ""),
		asst("m1", "r1", 10, 100, 0, 40, ""), // same response, later stream line with final output
		asst("m1", "r1", 10, 100, 0, 80, ""),
		asst("m2", "r2", 2, 50, 110, 20, ""),
	)
	if s.Responses != 2 {
		t.Fatalf("responses = %d, want 2 (deduplicated)", s.Responses)
	}
	if s.Total.Input != 12 || s.Total.CacheCreate != 150 || s.Total.CacheRead != 110 {
		t.Fatalf("totals counted duplicates: %+v", s.Total)
	}
	if s.Total.Output != 100 { // 80 (final of m1) + 20
		t.Fatalf("output = %d, want 100", s.Total.Output)
	}
}

func TestContextIsLastResponseNotSum(t *testing.T) {
	s := scanString(t, asst("a", "1", 5, 1000, 0, 1, ""), asst("b", "2", 3, 200, 1000, 1, ""))
	if got := s.ContextTokens(); got != 1203 {
		t.Fatalf("context = %d, want 1203 (last response only)", got)
	}
}

func TestSidechainExcluded(t *testing.T) {
	s := scanString(t, asst("a", "1", 1, 10, 0, 1, ""), asst("sub", "9", 1, 99999, 0, 1, `,"isSidechain":true`))
	if s.Responses != 1 || s.ContextTokens() != 11 {
		t.Fatalf("subagent leaked into main context: %+v", s)
	}
}

func TestCompactionResetsContext(t *testing.T) {
	s := scanString(t,
		asst("a", "1", 1, 500000, 0, 1, ""),
		`{"type":"system","subtype":"compact_boundary","timestamp":"2026-11-17T11:00:00Z"}`,
	)
	if s.Compactions != 1 || s.ContextTokens() != 0 {
		t.Fatalf("compaction not applied: compactions=%d ctx=%d", s.Compactions, s.ContextTokens())
	}
}

func TestTitleAndMalformedLines(t *testing.T) {
	s := scanString(t, `{"type":"ai-title","aiTitle":"Hello"}`, `{"type":"assistant","usage" broken json`, asst("a", "1", 1, 1, 1, 1, ""))
	if s.Title != "Hello" {
		t.Fatalf("title = %q", s.Title)
	}
	if s.Malformed != 1 {
		t.Fatalf("malformed = %d, want 1", s.Malformed)
	}
	if s.Responses != 1 {
		t.Fatal("a broken line must not stop the scan")
	}
}

func TestWindowFor(t *testing.T) {
	cases := map[string]int{
		"claude-opus-5-5": 1_000_000, "claude-sonnet-5-5": 1_000_000, "claude-haiku-4-5": 200_000,
		"claude-sonnet-4-5": 200_000, "claude-sonnet-4-5[1m]": 1_000_000, "claude-3-5-sonnet": 200_000, "claude-opus-4-6": 1_000_000,
	}
	for m, want := range cases {
		if got := WindowFor(m); got != want {
			t.Errorf("WindowFor(%s) = %d, want %d", m, got, want)
		}
	}
}
