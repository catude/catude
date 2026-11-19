// Package models says which Claude model and effort level is stronger, with no account and no API key.
//
// Sources (both public, downloaded only when the Models tab or `catude models` asks for them):
//
//   - Epoch AI Benchmarking Hub, https://epoch.ai/benchmarks — benchmark results per model version, with
//     the effort level in the version name (claude-opus-5-5_high). Epoch's own data is CC BY 4.0; results
//     Epoch collected from other projects keep their source and are credited to it. Files that come from
//     Artificial Analysis are skipped.
//   - models.dev, https://models.dev — prices per million tokens (MIT).
//
// The download is kept in a private per-user cache for a day.
package models

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	Attribution = "Data: Epoch AI Benchmarking Hub (CC BY 4.0, epoch.ai/benchmarks) and the projects it credits · prices: models.dev (MIT)"
	SiteURL     = "https://epoch.ai/benchmarks"
	cacheTTL    = 24 * time.Hour
	maxDownload = 40 << 20
	maxFile     = 20 << 20
	maxFiles    = 400
)

var (
	EpochURL     = "https://epoch.ai/data/benchmark_data.zip"
	ModelsDevURL = "https://models.dev/api.json"
	client       = &http.Client{Timeout: 60 * time.Second}
)

// Efforts are the reasoning effort levels, weakest first.
var Efforts = []string{"none", "low", "medium", "high", "xhigh", "max"}

// Categories group benchmarks by the kind of work they measure ("según el contexto").
var Categories = []string{"all", "coding", "reasoning", "math & science", "agents", "knowledge"}

// Bench is one benchmark.
type Bench struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Source   string `json:"source,omitempty"` // who ran it, when Epoch collected it from another project
}

// Model is one benchmarked model version: a model at one effort level.
type Model struct {
	ID       string             `json:"id"`    // claude-opus-5-5_high
	Base     string             `json:"base"`  // claude-opus-5-5
	Group    string             `json:"group"` // Claude Opus 5.5
	Effort   string             `json:"effort"`
	Released string             `json:"released,omitempty"`
	Scores   map[string]float64 `json:"scores"`         // benchmark → % of the best result anyone got on it
	Raw      map[string]float64 `json:"raw"`            // benchmark → the published score
	Cost     map[string]float64 `json:"cost,omitempty"` // benchmark → USD per task, when published
	ECI      float64            `json:"eci,omitempty"`  // Epoch Capabilities Index of the model (all efforts)
	PriceIn  float64            `json:"priceIn,omitempty"`
	PriceOut float64            `json:"priceOut,omitempty"`
	// cache prices (USD per 1M tokens) from models.dev; 0 when not published
	PriceCacheRead  float64 `json:"priceCacheRead,omitempty"`
	PriceCacheWrite float64 `json:"priceCacheWrite,omitempty"`
}

// CostUSD estimates what a number of tokens costs at list prices. Without published cache prices it uses
// Anthropic's documented ratios: a cache write costs 1.25 times the input price (5-minute cache) and a
// cache read 0.1 times (platform.claude.com/docs/en/about-claude/pricing).
func (m Model) CostUSD(input, cacheWrite, cacheRead, output int) float64 {
	cw, cr := m.PriceCacheWrite, m.PriceCacheRead
	if cw == 0 {
		cw = 1.25 * m.PriceIn
	}
	if cr == 0 {
		cr = 0.1 * m.PriceIn
	}
	return (float64(input)*m.PriceIn + float64(cacheWrite)*cw + float64(cacheRead)*cr + float64(output)*m.PriceOut) / 1e6
}

var reDateSuffix = regexp.MustCompile(`^-[0-9]{8}$`)

// PriceFor finds the priced model behind a model id as Claude Code writes it (claude-opus-5-5,
// claude-sonnet-4-5-20250929, claude-opus-4-6[1m]…): the longest base that starts the id.
func (d Data) PriceFor(id string) (Model, bool) {
	id = strings.ToLower(id)
	if i := strings.IndexByte(id, '['); i >= 0 {
		id = id[:i]
	}
	var best Model
	for _, m := range d.Models {
		if m.PriceIn > 0 && m.Base != "" && strings.HasPrefix(id, strings.ToLower(m.Base)) && len(m.Base) > len(best.Base) {
			if rest := id[len(m.Base):]; rest == "" || reDateSuffix.MatchString(rest) { // claude-opus-5 must not price claude-opus-5-5
				best = m
			}
		}
	}
	return best, best.Base != ""
}

// Label is the short display name: "Opus 5.5 · high".
func (m Model) Label() string {
	b := strings.TrimSpace(strings.TrimPrefix(m.Group, "Claude "))
	if m.Effort == "" {
		return b
	}
	return b + " · " + m.Effort
}

// Blended is the price of 1M tokens at 3 input : 1 output.
func (m Model) Blended() float64 { return (3*m.PriceIn + m.PriceOut) / 4 }

// Data is what is cached.
type Data struct {
	Models  []Model   `json:"models"`
	Benches []Bench   `json:"benches"`
	Fetched time.Time `json:"fetched"`
	Updated string    `json:"updated,omitempty"` // newest result date in the data
}

// Stale says whether the data should be downloaded again.
func (d Data) Stale() bool { return len(d.Models) == 0 || time.Since(d.Fetched) > cacheTTL }

func (d Data) bench(name string) Bench {
	for _, b := range d.Benches {
		if b.Name == name {
			return b
		}
	}
	return Bench{Name: name, Category: "other"}
}

// ---- download ----

func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "catude")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s answered %s", req.URL.Host, resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload+1))
	if err == nil && len(b) > maxDownload {
		err = fmt.Errorf("%s: download larger than %d MB", req.URL.Host, maxDownload>>20)
	}
	return b, err
}

// Fetch downloads both sources and builds the data. Prices are optional: without them the
// comparison still works.
func Fetch(ctx context.Context) (Data, error) {
	zb, err := get(ctx, EpochURL)
	if err != nil {
		return Data{}, fmt.Errorf("Epoch AI: %w", err)
	}
	d, err := ParseEpoch(zb)
	if err != nil {
		return Data{}, fmt.Errorf("Epoch AI: %w", err)
	}
	if pb, err := get(ctx, ModelsDevURL); err == nil {
		applyPrices(&d, pb)
	}
	d.Fetched = time.Now()
	return d, nil
}

// ---- Epoch ----

var (
	reVersion = regexp.MustCompile(`^(claude-[a-z0-9\-]+?)_(none|low|medium|high|xhigh|max)$`)
	reFile    = regexp.MustCompile(`^[a-z0-9_\-]{1,80}\.csv$`)
	reDate    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
	notScore  = map[string]bool{"release date": true, "organization": true, "country": true, "scaffold": true, "tools": true, "name": true, "notes": true, "source": true, "id": true, "model_group": true}
	pretty    = map[string]string{"arc_agi": "ARC-AGI", "arc_agi_2": "ARC-AGI-2", "cursorbench": "CursorBench", "critpt": "CritPt", "dtbench": "DTBench",
		"lmca": "LMCA", "gpqa_diamond": "GPQA Diamond", "terminalbench": "Terminal-Bench", "hle": "Humanity's Last Exam", "swe_bench_verified": "SWE-bench Verified",
		"aider_polyglot": "Aider Polyglot", "simpleqa_verified": "SimpleQA Verified", "chess_puzzles": "Chess puzzles", "apex_agents": "APEX Agents", "deepswe": "DeepSWE", "frontierswe": "FrontierSWE", "frontiercode": "FrontierCode", "swe_bench_pro": "SWE-bench Pro"}
	catWords = []struct{ cat, words string }{
		{"coding", "cursor swe terminal aider code cyber exploit gso algotune ale_bench webdev"},
		{"math & science", "frontiermath math gpqa critpt hle physics chem science scicode"},
		{"reasoning", "arc_agi enigma chess bbh puzzle reason"},
		{"agents", "agent apex balrog gdp osworld tau browse research lmca dtbench forecast"},
		{"knowledge", "simpleqa mmlu trivia geobench fiction lambada bool_q hella"},
	}
)

func benchName(file string) string {
	s := strings.TrimSuffix(strings.TrimSuffix(file, ".csv"), "_external")
	if p, ok := pretty[s]; ok {
		return p
	}
	s = strings.ReplaceAll(s, "_", " ")
	if s != "" {
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}

func category(file string) string {
	f := strings.ToLower(file)
	for _, c := range catWords {
		for _, w := range strings.Fields(c.words) {
			if strings.Contains(f, w) {
				return c.cat
			}
		}
	}
	return "other"
}

func readCSV(b []byte) [][]string {
	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))))
	r.FieldsPerRecord, r.LazyQuotes = -1, true
	rows, _ := r.ReadAll()
	return rows
}

func col(header []string, names ...string) int {
	for _, n := range names {
		for i, h := range header {
			if strings.EqualFold(strings.TrimSpace(h), n) {
				return i
			}
		}
	}
	return -1
}

func clean(s string, n int) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) || (r >= 0x200b && r <= 0x200f) || (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069) {
			return -1
		}
		return r
	}, strings.TrimSpace(s))
	if len(s) > n {
		s = s[:n]
	}
	return s
}

func parseNum(s string) (float64, bool) {
	s = strings.TrimSuffix(strings.TrimSpace(s), "%")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// ParseEpoch reads the Benchmarking Hub ZIP and keeps the Claude model versions with an effort level.
func ParseEpoch(zb []byte) (Data, error) {
	zr, err := zip.NewReader(bytes.NewReader(zb), int64(len(zb)))
	if err != nil {
		return Data{}, err
	}
	files := map[string][]byte{}
	for i, f := range zr.File {
		if i >= maxFiles {
			break
		}
		name := path.Base(f.Name)
		dir := path.Dir(f.Name)
		if !reFile.MatchString(name) || f.UncompressedSize64 > maxFile || strings.Contains(f.Name, "..") {
			continue
		}
		key := name
		if dir != "." && dir != "" {
			key = path.Base(dir) + "/" + name
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(rc, maxFile+1))
		rc.Close()
		if err == nil && len(b) <= maxFile {
			files[key] = b
		}
	}
	meta := map[string]struct {
		group, released string
	}{}
	for _, r := range readCSV(files["model_metadata.csv"]) {
		if len(r) >= 3 && strings.HasPrefix(r[0], "claude-") {
			meta[r[0]] = struct{ group, released string }{clean(r[1], 60), clean(r[2], 10)}
		}
	}
	if len(meta) == 0 {
		return Data{}, errors.New("model_metadata.csv missing or without Claude models")
	}
	scoreCol := map[string]string{} // file → score column, from benchmark_metadata.csv
	if rows := readCSV(files["benchmark_metadata.csv"]); len(rows) > 1 {
		fi, si := col(rows[0], "source_file"), col(rows[0], "score_column")
		for _, r := range rows[1:] {
			if fi >= 0 && si >= 0 && fi < len(r) && si < len(r) && r[fi] != "" {
				scoreCol[r[fi]] = r[si]
			}
		}
	}
	byID := map[string]*Model{}
	var benches []Bench
	newest := ""
	names := make([]string, 0, len(files))
	for k := range files {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, file := range names {
		if strings.Contains(file, "/") || file == "model_metadata.csv" || file == "benchmark_metadata.csv" {
			continue
		}
		rows := readCSV(files[file])
		if len(rows) < 2 || len(rows[0]) < 2 || !strings.EqualFold(rows[0][0], "Model version") {
			continue
		}
		h := rows[0]
		if col(h, "AA model slug") >= 0 || bytes.Contains(bytes.ToLower(files[file]), []byte("artificialanalysis")) {
			continue // Artificial Analysis data: not ours to use
		}
		si := col(h, scoreCol[file])
		if si < 0 {
			for i := 1; i < len(h); i++ {
				if !notScore[strings.ToLower(strings.TrimSpace(h[i]))] {
					si = i
					break
				}
			}
		}
		ci, srcI, di := col(h, "Cost per task", "Cost"), col(h, "Source"), col(h, "Started at", "Release date")
		best, source := 0.0, ""
		vals := map[string]float64{}
		costs := map[string]float64{}
		for _, r := range rows[1:] {
			if si >= len(r) || len(r) == 0 {
				continue
			}
			v, ok := parseNum(r[si])
			if !ok || v < 0 {
				continue
			}
			best = math.Max(best, v)
			id := strings.TrimSpace(r[0])
			if reVersion.MatchString(id) && !strings.Contains(id, "fallback") {
				if old, seen := vals[id]; !seen || v > old {
					vals[id] = v
				}
				if ci >= 0 && ci < len(r) {
					if c, ok := parseNum(r[ci]); ok && c > 0 {
						costs[id] = c
					}
				}
				if di >= 0 && di < len(r) && reDate.MatchString(r[di]) && r[di][:10] > newest {
					newest = r[di][:10]
				}
			}
			if srcI >= 0 && srcI < len(r) && source == "" {
				source = clean(r[srcI], 120)
			}
		}
		if len(vals) == 0 || best <= 0 {
			continue
		}
		name := benchName(file)
		benches = append(benches, Bench{Name: name, Category: category(file), Source: source})
		for id, v := range vals {
			m := byID[id]
			if m == nil {
				mm := reVersion.FindStringSubmatch(id)
				md, ok := meta[id]
				if !ok {
					continue
				}
				m = &Model{ID: id, Base: mm[1], Effort: mm[2], Group: md.group, Released: md.released, Scores: map[string]float64{}, Raw: map[string]float64{}}
				byID[id] = m
			}
			m.Raw[name] = v
			m.Scores[name] = 100 * v / best
			if c, ok := costs[id]; ok {
				if m.Cost == nil {
					m.Cost = map[string]float64{}
				}
				m.Cost[name] = c
			}
		}
	}
	eci := map[string]float64{}
	if rows := readCSV(files["epoch_capabilities_index/eci_scores.csv"]); len(rows) > 1 {
		mi, ei := col(rows[0], "Model"), col(rows[0], "eci")
		for _, r := range rows[1:] {
			if mi >= 0 && ei >= 0 && ei < len(r) {
				if v, ok := parseNum(r[ei]); ok {
					eci[r[mi]] = v
				}
			}
		}
	}
	var out []Model
	for _, m := range byID {
		m.ECI = eci[m.Group]
		out = append(out, *m)
	}
	if len(out) == 0 {
		return Data{}, errors.New("no Claude results with an effort level")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return Data{Models: out, Benches: benches, Updated: newest}, nil
}

// ---- models.dev ----

func applyPrices(d *Data, b []byte) {
	var all map[string]json.RawMessage
	if json.Unmarshal(b, &all) != nil {
		return
	}
	var prov struct {
		Models map[string]struct {
			Cost struct {
				Input      float64 `json:"input"`
				Output     float64 `json:"output"`
				CacheRead  float64 `json:"cache_read"`
				CacheWrite float64 `json:"cache_write"`
			} `json:"cost"`
		} `json:"models"`
	}
	if json.Unmarshal(all["anthropic"], &prov) != nil {
		return
	}
	for i := range d.Models {
		if p, ok := prov.Models[d.Models[i].Base]; ok && p.Cost.Input >= 0 && p.Cost.Output >= 0 {
			d.Models[i].PriceIn, d.Models[i].PriceOut = p.Cost.Input, p.Cost.Output
			d.Models[i].PriceCacheRead, d.Models[i].PriceCacheWrite = max(0, p.Cost.CacheRead), max(0, p.Cost.CacheWrite)
		}
	}
}

// ---- cache ----

func CachePath(dir string) string { return filepath.Join(dir, "models-epoch.json") }

func ReadCache(dir string) Data {
	var d Data
	if b, err := os.ReadFile(CachePath(dir)); err == nil && len(b) < maxDownload {
		_ = json.Unmarshal(b, &d)
	}
	return d
}

func WriteCache(dir string, d Data) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "models-*.tmp")
	if err != nil {
		return err
	}
	_ = tmp.Chmod(0o600)
	_, werr := tmp.Write(b)
	tmp.Close()
	if werr != nil {
		os.Remove(tmp.Name())
		return werr
	}
	return os.Rename(tmp.Name(), CachePath(dir))
}

// Load returns the cached data, downloading again when it is older than a day (or refresh is set).
// When the download fails, the previous data is returned with the error.
func Load(ctx context.Context, dir string, refresh bool) (Data, error) {
	d := ReadCache(dir)
	if !refresh && !d.Stale() {
		return d, nil
	}
	fresh, err := Fetch(ctx)
	if err != nil {
		return d, err
	}
	_ = WriteCache(dir, fresh)
	return fresh, nil
}

// ---- ranking and comparison ----

// Recent keeps the model versions of Claude models released in the 15 months before the newest one.
func Recent(ms []Model) []Model {
	newest := ""
	for _, m := range ms {
		if m.Released > newest {
			newest = m.Released
		}
	}
	cut := ""
	if t, err := time.Parse("2006-01-02", newest); err == nil {
		cut = t.AddDate(0, -15, 0).Format("2006-01-02")
	}
	var out []Model
	for _, m := range ms {
		if m.Released >= cut {
			out = append(out, m)
		}
	}
	return out
}

func inCat(d Data, bench, cat string) bool {
	return cat == "" || cat == "all" || d.bench(bench).Category == cat
}

// Core picks the benchmarks of a category measured for nearly as many models as the most measured one,
// so averages compare like with like.
func Core(d Data, ms []Model, cat string) []string {
	count := map[string]int{}
	for _, m := range ms {
		for b := range m.Scores {
			if inCat(d, b, cat) {
				count[b]++
			}
		}
	}
	top := 0
	for _, n := range count {
		top = max(top, n)
	}
	var out []string
	for b, n := range count {
		if n >= 2 && float64(n) >= 0.6*float64(top) {
			out = append(out, b)
		}
	}
	sort.Strings(out)
	return out
}

// Score is the average "% of the best result" over the core benchmarks this model was measured on,
// with how many of them it covers. Fewer than half of them gives no score.
func Score(m Model, core []string) (float64, int) {
	sum, n := 0.0, 0
	for _, b := range core {
		if v, ok := m.Scores[b]; ok {
			sum += v
			n++
		}
	}
	if n == 0 || n*2 < len(core) {
		return 0, n
	}
	return sum / float64(n), n
}

// Ranked sorts models by Score on a category, strongest first; models without a score go last.
func Ranked(d Data, ms []Model, cat string) ([]Model, []string) {
	core := Core(d, ms, cat)
	out := append([]Model(nil), ms...)
	sort.SliceStable(out, func(i, j int) bool {
		a, _ := Score(out[i], core)
		b, _ := Score(out[j], core)
		if a != b {
			return a > b
		}
		return out[i].ID < out[j].ID
	})
	return out, core
}

// Find resolves "opus 5.5 high", "claude-opus-5-5:high" or "Fable 5.2 medium" to one model version.
// Without an effort, medium is used. ok is false when that effort is not benchmarked (closest given).
func Find(ms []Model, query string) (Model, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	effort := ""
	for _, sep := range []string{":", "@", "_", " "} {
		if i := strings.LastIndex(q, sep); i > 0 {
			if e := normEffort(q[i+1:]); e != "" {
				effort, q = e, strings.TrimSpace(q[:i])
				break
			}
		}
	}
	fam := family(q)
	var cands []Model
	for _, m := range ms {
		if (fam != "" && family(m.Base) == fam) || (fam == "" && q != "" && strings.Contains(strings.ToLower(m.Group+" "+m.Base), q)) {
			cands = append(cands, m)
		}
	}
	if len(cands) == 0 {
		return Model{}, false
	}
	want := effort
	if want == "" {
		want = "medium"
	}
	best, bestD := cands[0], 99
	for _, c := range cands {
		if dd := dist(c.Effort, want); dd < bestD || (dd == bestD && len(c.Scores) > len(best.Scores)) {
			best, bestD = c, dd
		}
	}
	return best, effort == "" || best.Effort == effort
}

func normEffort(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "none":
		return "none"
	case "low":
		return "low"
	case "medium", "med":
		return "medium"
	case "high":
		return "high"
	case "xhigh", "x-high", "extra-high", "extra high":
		return "xhigh"
	case "max", "maximum":
		return "max"
	}
	return ""
}

func rank(e string) int {
	for i, x := range Efforts {
		if x == e {
			return i
		}
	}
	return -1
}

func dist(a, b string) int {
	ra, rb := rank(a), rank(b)
	if ra < 0 || rb < 0 {
		return 50
	}
	if ra > rb {
		return ra - rb
	}
	return rb - ra
}

var reFamily = regexp.MustCompile(`(?i)(fable|opus|sonnet|haiku|mythos)[\s\-_]*(\d+)(?:[\.\-_](\d{1,2})(?:\D|$))?`)

func family(s string) string {
	m := reFamily.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	f := strings.ToLower(m[1]) + " " + m[2]
	if m[3] != "" {
		f += "." + m[3]
	}
	return f
}

// Verdict is a short comparison of two model versions.
type Verdict struct {
	Winner  string   // label of the stronger one, "" when they are even or not comparable
	Summary string   // one line
	Notes   []string // where each one wins, price and cost per task
	Shared  int      // benchmarks both were measured on
}

// Close is how many points of "% of best" count as even: smaller gaps are within noise.
const Close = 3.0

// Compare says which of two model versions is stronger on the benchmarks of a category that both
// were measured on.
func Compare(d Data, a, b Model, cat string) Verdict {
	var v Verdict
	var winsA, winsB []string
	sumA, sumB := 0.0, 0.0
	var costA, costB float64
	costBench := ""
	for name, sa := range a.Scores {
		sb, ok := b.Scores[name]
		if !ok || !inCat(d, name, cat) {
			continue
		}
		v.Shared++
		sumA += sa
		sumB += sb
		switch {
		case sa-sb >= Close:
			winsA = append(winsA, name)
		case sb-sa >= Close:
			winsB = append(winsB, name)
		}
		if ca, cb := a.Cost[name], b.Cost[name]; ca > 0 && cb > 0 && costBench == "" {
			costA, costB, costBench = ca, cb, name
		}
	}
	sort.Strings(winsA)
	sort.Strings(winsB)
	ctx := ""
	if cat != "" && cat != "all" {
		ctx = " in " + cat
	}
	if v.Shared == 0 {
		v.Summary = fmt.Sprintf("no benchmark%s has measured both %s and %s yet", ctx, a.Label(), b.Label())
		return v
	}
	avgA, avgB := sumA/float64(v.Shared), sumB/float64(v.Shared)
	n := fmt.Sprintf("%d benchmarks", v.Shared)
	if v.Shared == 1 {
		n = "1 benchmark, so take it as a hint"
	}
	switch {
	case len(winsA) > len(winsB) && avgA-avgB >= Close/2:
		v.Winner = a.Label()
		v.Summary = fmt.Sprintf("%s is stronger than %s%s: better on %d of %s", a.Label(), b.Label(), ctx, len(winsA), n)
	case len(winsB) > len(winsA) && avgB-avgA >= Close/2:
		v.Winner = b.Label()
		v.Summary = fmt.Sprintf("%s is stronger than %s%s: better on %d of %s", b.Label(), a.Label(), ctx, len(winsB), n)
	default:
		v.Summary = fmt.Sprintf("%s and %s are about even%s (%s)", a.Label(), b.Label(), ctx, n)
	}
	if len(winsA) > 0 && len(winsB) > 0 {
		v.Notes = append(v.Notes, fmt.Sprintf("%s wins %s; %s wins %s", a.Label(), list(winsA), b.Label(), list(winsB)))
	} else if len(winsA)+len(winsB) > 0 && v.Winner == "" {
		v.Notes = append(v.Notes, "the differences are small")
	}
	if costBench != "" {
		v.Notes = append(v.Notes, fmt.Sprintf("cost per task on %s: $%.2f vs $%.2f", costBench, costA, costB))
	}
	if pa, pb := a.Blended(), b.Blended(); pa > 0 && pb > 0 && a.Base != b.Base {
		switch r := pa / pb; {
		case r > 1.15:
			v.Notes = append(v.Notes, fmt.Sprintf("%s costs %s as much per token", a.Label(), times(r)))
		case r < 1/1.15:
			v.Notes = append(v.Notes, fmt.Sprintf("%s costs %s as much per token", b.Label(), times(1/r)))
		default:
			v.Notes = append(v.Notes, "same price per token")
		}
	}
	if a.Base == b.Base {
		v.Notes = append(v.Notes, "same model: higher effort thinks longer, so it is slower and spends more tokens per answer")
	}
	return v
}

func list(s []string) string {
	if len(s) > 3 {
		return strings.Join(s[:3], ", ") + fmt.Sprintf(" +%d", len(s)-3)
	}
	return strings.Join(s, ", ")
}

func times(r float64) string {
	if r >= 1.95 {
		return fmt.Sprintf("%.0f×", math.Round(r))
	}
	return fmt.Sprintf("%.1f×", r)
}

// Advice looks at the model version a person uses and names what is worth knowing in a category:
// something as strong that costs less per token, and something clearly stronger.
func Advice(d Data, ms []Model, cur Model, cat string) []string {
	ranked, core := Ranked(d, ms, cat)
	sc, _ := Score(cur, core)
	if sc == 0 {
		return []string{cur.Label() + " has too few results in this category to compare"}
	}
	var cheaper, stronger *Model
	for i := range ranked {
		m := ranked[i]
		s, _ := Score(m, core)
		if m.ID == cur.ID || s == 0 {
			continue
		}
		if s >= sc-Close && m.Blended() > 0 && cur.Blended() > 0 && m.Blended() < cur.Blended()*0.8 && (cheaper == nil || m.Blended() < cheaper.Blended()) {
			cheaper = &ranked[i]
		}
		if s >= sc+3 && stronger == nil {
			stronger = &ranked[i] // ranked: the first one found is the strongest; keep looking for a cheaper strong one
		}
		if s >= sc+3 && stronger != nil && m.Blended() > 0 && m.Blended() < stronger.Blended() {
			stronger = &ranked[i]
		}
	}
	var out []string
	if cheaper != nil && stronger != nil && cheaper.ID == stronger.ID {
		v := Compare(d, *stronger, cur, cat)
		return []string{fmt.Sprintf("%s; it also costs %.0f%% less per token", v.Summary, 100*(1-cheaper.Blended()/cur.Blended()))}
	}
	if cheaper != nil {
		out = append(out, fmt.Sprintf("%s is about as strong and costs %.0f%% less per token", cheaper.Label(), 100*(1-cheaper.Blended()/cur.Blended())))
	}
	if stronger != nil {
		v := Compare(d, *stronger, cur, cat)
		out = append(out, "for harder work: "+v.Summary)
	}
	if len(out) == 0 {
		out = append(out, cur.Label()+" is a good choice here: nothing cheaper is as strong, nothing is clearly stronger")
	}
	return out
}

// FromUsage maps model ids recorded in transcripts ("claude-opus-5-5-20260922") and the configured
// effort level to benchmarked versions, most used first.
func FromUsage(ms []Model, used map[string]int, effort string) []Model {
	type kv struct {
		id string
		n  int
	}
	var ids []kv
	for k, n := range used {
		ids = append(ids, kv{k, n})
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].n > ids[j].n })
	seen := map[string]bool{}
	var out []Model
	for _, x := range ids {
		f := family(x.id)
		if f == "" || seen[f] {
			continue
		}
		q := f
		if effort != "" {
			q += " " + effort
		}
		if m, _ := Find(ms, q); m.ID != "" {
			seen[f] = true
			out = append(out, m)
		}
	}
	return out
}
