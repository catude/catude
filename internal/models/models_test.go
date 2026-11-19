package models

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// A small Benchmarking Hub ZIP with invented numbers (catude ships no benchmark data).
func fakeZip(t *testing.T) []byte {
	t.Helper()
	files := map[string]string{
		"model_metadata.csv": "model_version,model_group,date,display_name,organization\n" +
			"claude-alpha-9-1_medium,Claude Alpha 9.1,2026-11-01,x,Anthropic\n" +
			"claude-alpha-9-1_high,Claude Alpha 9.1,2026-11-01,x,Anthropic\n" +
			"claude-opus-5-5_medium,Claude Opus 5.5,2026-09-22,x,Anthropic\n" +
			"claude-opus-5-5_high,Claude Opus 5.5,2026-09-22,x,Anthropic\n" +
			"claude-sonnet-5-5_xhigh,Claude Sonnet 5.5,2026-09-28,x,Anthropic\n",
		"benchmark_metadata.csv": "benchmark,in_eci,source_file,score_column\nLMCA,True,lmca_external.csv,Score\n",
		"cursorbench_external.csv": "Model version,Score,Reasoning level,Cost per task,Source\n" +
			"claude-alpha-9-1_medium,0.55,Medium,4.66,https://cursor.com/cursorbench\n" +
			"claude-opus-5-5_high,0.58,High,3.97,https://cursor.com/cursorbench\n" +
			"claude-opus-5-5_medium,0.52,Medium,2.91,https://cursor.com/cursorbench\n" +
			"claude-sonnet-5-5_xhigh,0.50,Extra High,1.5,https://cursor.com/cursorbench\n" +
			"other-model_high,0.60,High,1,https://cursor.com/cursorbench\n",
		"arc_agi_2_external.csv": "Model version,Score,Release date\n" +
			"claude-alpha-9-1_medium,0.90,2026-11-01\nclaude-opus-5-5_high,0.80,2026-09-22\nclaude-opus-5-5_medium,0.70,2026-09-22\nclaude-sonnet-5-5_xhigh,0.60,2026-09-28\n",
		"lmca_external.csv": "Model version,Notes,Score\n" +
			"claude-alpha-9-1_medium,,70.0\nclaude-opus-5-5_high,,65.9\nclaude-opus-5-5_medium,,60\nclaude-sonnet-5-5_xhigh,,64\n",
		"scicode_external.csv":                    "Model version,Score,AA model slug\nclaude-opus-5-5_high,0.99,x\n",
		"epoch_capabilities_index/eci_scores.csv": "Model,Display name,eci\nClaude Opus 5.5,Claude Opus 5.5,168.7\n",
		"../evil.csv":                             "Model version,Score\nclaude-opus-5-5_high,1\n",
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	zw.Close()
	return buf.Bytes()
}

func TestParseEpoch(t *testing.T) {
	d, err := ParseEpoch(fakeZip(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Models) != 4 { // alpha high has no results
		t.Fatalf("models: %d", len(d.Models))
	}
	opus, ok := Find(d.Models, "opus 5.5 high")
	if !ok || opus.ECI != 168.7 || opus.Cost["CursorBench"] != 3.97 || opus.Raw["LMCA"] != 65.9 {
		t.Errorf("opus high: %+v", opus)
	}
	if _, bad := opus.Raw["Scicode"]; bad {
		t.Error("Artificial Analysis files must be skipped")
	}
	// % of the best result anyone got, including non-Claude models
	if got := opus.Scores["CursorBench"]; got < 96 || got > 97 {
		t.Errorf("normalised CursorBench: %v", got)
	}
	if d.Updated != "2026-11-01" {
		t.Errorf("updated: %q", d.Updated)
	}
	cats := map[string]string{}
	for _, b := range d.Benches {
		cats[b.Name] = b.Category
	}
	if cats["CursorBench"] != "coding" || cats["ARC-AGI-2"] != "reasoning" || cats["LMCA"] != "agents" {
		t.Errorf("categories: %v", cats)
	}
}

func TestFindAndCompare(t *testing.T) {
	d, _ := ParseEpoch(fakeZip(t))
	for q, want := range map[string]string{
		"opus 5.5 high":        "Opus 5.5 · high",
		"claude-opus-5-5":      "Opus 5.5 · medium", // no effort: medium
		"claude-opus-5-5_high": "Opus 5.5 · high",
		"sonnet-5-5:x-high":    "Sonnet 5.5 · xhigh",
	} {
		if m, ok := Find(d.Models, q); !ok || m.Label() != want {
			t.Errorf("%q → %q %v, want %q", q, m.Label(), ok, want)
		}
	}
	if m, ok := Find(d.Models, "sonnet 5.5 low"); ok || m.Label() != "Sonnet 5.5 · xhigh" {
		t.Errorf("missing effort gives the closest and says so: %q %v", m.Label(), ok)
	}
	a, _ := Find(d.Models, "opus 5.5 high")
	b, _ := Find(d.Models, "alpha 9.1 medium")
	v := Compare(d, a, b, "all")
	if v.Winner != "Alpha 9.1 · medium" || v.Shared != 3 || !strings.Contains(v.Summary, "2 of 3") {
		t.Errorf("all: %+v", v)
	}
	if !strings.Contains(strings.Join(v.Notes, " "), "Opus 5.5 · high wins CursorBench") {
		t.Errorf("notes: %v", v.Notes)
	}
	// according to the context: on coding Opus is ahead, on one benchmark only
	if v := Compare(d, a, b, "coding"); v.Winner != "Opus 5.5 · high" || !strings.Contains(v.Summary, "take it as a hint") || !strings.Contains(strings.Join(v.Notes, " "), "$3.97 vs $4.66") {
		t.Errorf("coding: %+v", v)
	}
	s, _ := Find(d.Models, "sonnet 5.5 xhigh")
	if v := Compare(d, a, s, "math & science"); v.Shared != 0 || !strings.Contains(v.Summary, "no benchmark in math & science") {
		t.Errorf("nothing shared: %+v", v)
	}
}

func TestRankedAndAdvice(t *testing.T) {
	d, _ := ParseEpoch(fakeZip(t))
	ranked, core := Ranked(d, d.Models, "all")
	if ranked[0].Label() != "Alpha 9.1 · medium" || len(core) != 3 {
		t.Errorf("ranked: %v core %v", ranked[0].Label(), core)
	}
	cur, _ := Find(d.Models, "opus 5.5 medium")
	got := strings.Join(Advice(d, d.Models, cur, "all"), "\n")
	if !strings.Contains(got, "for harder work: Alpha 9.1 · medium is stronger") {
		t.Errorf("advice:\n%s", got)
	}
	used := FromUsage(d.Models, map[string]int{"claude-opus-5-5-20260922": 9, "claude-sonnet-5-5": 1}, "high")
	if len(used) != 2 || used[0].Label() != "Opus 5.5 · high" || used[1].Label() != "Sonnet 5.5 · xhigh" {
		t.Errorf("from usage: %v", used)
	}
}

func TestFetchAndCache(t *testing.T) {
	zb := fakeZip(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("x-api-key") != "" {
			t.Error("no credentials are ever sent")
		}
		switch r.URL.Path {
		case "/zip":
			w.Write(zb)
		case "/api.json":
			w.Write([]byte(`{"anthropic":{"models":{"claude-opus-5-5":{"cost":{"input":4,"output":20}}}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	oz, om := EpochURL, ModelsDevURL
	EpochURL, ModelsDevURL = srv.URL+"/zip", srv.URL+"/api.json"
	defer func() { EpochURL, ModelsDevURL = oz, om }()

	dir := t.TempDir()
	d, err := Load(context.Background(), dir, false)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := Find(d.Models, "opus 5.5 high")
	if m.Blended() != 8 {
		t.Errorf("price from models.dev: %+v", m)
	}
	if c := ReadCache(dir); len(c.Models) != len(d.Models) || c.Stale() {
		t.Error("cache round trip")
	}
	ModelsDevURL = srv.URL + "/missing"
	if _, err := Fetch(context.Background()); err != nil {
		t.Errorf("prices are optional: %v", err)
	}
}

// With CATUDE_EPOCH_ZIP pointing at a real download, prints the answer to a real question.
func TestRealEpochZip(t *testing.T) {
	p := os.Getenv("CATUDE_EPOCH_ZIP")
	if p == "" {
		t.Skip("set CATUDE_EPOCH_ZIP")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	d, err := ParseEpoch(b)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := Find(d.Models, "opus 5.5 high")
	c, _ := Find(d.Models, "fable 5.2 medium")
	for _, cat := range Categories {
		v := Compare(d, a, c, cat)
		t.Logf("%-15s %s | %s", cat, v.Summary, strings.Join(v.Notes, " · "))
	}
	ranked, core := Ranked(d, Recent(d.Models), "all")
	t.Logf("core: %v", core)
	for i, m := range ranked {
		if i == 12 {
			break
		}
		s, n := Score(m, core)
		t.Logf("%-22s %5.1f (%d)", m.Label(), s, n)
	}
}

// Session ids find their priced model (dated and [1m] ids too) but never a shorter, older one.
func TestPriceForAndCost(t *testing.T) {
	d := Data{Models: []Model{{Base: "claude-opus-5", PriceIn: 5, PriceOut: 25}, {Base: "claude-sonnet-4-5", PriceIn: 3, PriceOut: 15, PriceCacheRead: 0.3, PriceCacheWrite: 3.75}}}
	if _, ok := d.PriceFor("claude-opus-5-5"); ok {
		t.Error("claude-opus-5 must not price claude-opus-5-5")
	}
	p, ok := d.PriceFor("claude-sonnet-4-5-20250929")
	if !ok || p.Base != "claude-sonnet-4-5" {
		t.Fatalf("dated id: %+v %v", p, ok)
	}
	if p, ok := d.PriceFor("Claude-Opus-5[1m]"); !ok || p.Base != "claude-opus-5" {
		t.Fatalf("[1m] id: %+v %v", p, ok)
	}
	// 1M input + 1M cache write + 1M cache read + 1M output at Opus 5 list prices: 5 + 6.25 + 0.5 + 25
	if c := d.Models[0].CostUSD(1e6, 1e6, 1e6, 1e6); c < 36.74 || c > 36.76 {
		t.Errorf("cost with the documented cache ratios: %v", c)
	}
	if c := p.CostUSD(0, 1e6, 1e6, 0); c < 4.04 || c > 4.06 {
		t.Errorf("published cache prices win over the ratios: %v", c)
	}
}
