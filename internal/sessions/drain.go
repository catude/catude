package sessions

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Drain explains where the tokens of a period went, from the same transcripts as the usage report.
// "Fresh" tokens are new input, cache writes and output: what weighs on plan limits and cost (cache
// reads are cheap). Every number is measured; the TUI and `catude why` turn them into sentences.
type Drain struct {
	Fresh     int            // all fresh tokens in the period
	Subagents int            // fresh tokens spent by subagents
	ByEntry   map[string]int // fresh tokens by how Claude Code was started (cli, claude-vscode, sdk-cli…)

	// cache rebuilt after a pause: a response that wrote more than RebuildMin tokens to the cache after
	// more than PauseMin without activity in that conversation, when the cached prefix had expired
	Rebuilds      int
	RebuildTokens int

	// context of the first response of each conversation: what is loaded before you type
	Starts   int
	StartAvg int
	StartMax int
	CacheHit float64 // cache reads / all input read, over the period

	ByVersion []VersionDrain // main conversations only, newest version first
	Top       []SessionDrain // most expensive conversations (with their subagents)
}

// VersionDrain is the average fresh tokens per response under one Claude Code version.
type VersionDrain struct {
	Version   string
	Responses int
	PerResp   int
}

// SessionDrain is one conversation and what it cost, its subagents included.
type SessionDrain struct {
	Project   string
	Path      string
	Start     time.Time
	Fresh     int
	Subagents int
	Entry     string
}

const (
	PauseMin   = 5 * time.Minute
	RebuildMin = 20_000
)

type drainAcc struct {
	d        Drain
	sessions map[string]*SessionDrain
	vers     map[string]*VersionDrain
	read     int // cache reads
	allIn    int // input + cache writes + cache reads
	startSum int
}

func newDrain() *drainAcc {
	return &drainAcc{d: Drain{ByEntry: map[string]int{}}, sessions: map[string]*SessionDrain{}, vers: map[string]*VersionDrain{}}
}

// add counts the responses of one transcript for which count is true (in the period, not already counted
// in another file). Every response, counted or not, still sets the time of the previous one, so pauses
// and the first response of a conversation are judged on the whole file.
func (a *drainAcc) add(project, path string, fu fileUsage, count func(respRec) bool) {
	sub := strings.Contains(filepath.ToSlash(path), "/subagents/")
	key := path
	if sub { // .../<session>/subagents/agent-x.jsonl belongs to .../<session>.jsonl
		key = filepath.Dir(filepath.Dir(path)) + ".jsonl"
	}
	s := a.sessions[key]
	if s == nil {
		s = &SessionDrain{Project: project, Path: key}
		a.sessions[key] = s
	}
	recs := append([]respRec(nil), fu.recs...)
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].t.Before(recs[j].t) })
	var prev time.Time
	first := true
	for _, r := range recs {
		if !count(r) {
			prev, first = r.t, false
			continue
		}
		fresh := r.in + r.cw + r.out
		a.d.Fresh += fresh
		s.Fresh += fresh
		a.read += r.cr
		a.allIn += r.in + r.cw + r.cr
		entry := fu.names[r.entry]
		if entry == "" {
			entry = "unknown"
		}
		a.d.ByEntry[entry] += fresh
		if sub {
			a.d.Subagents += fresh
			s.Subagents += fresh
		} else {
			if s.Start.IsZero() || r.t.Before(s.Start) {
				s.Start, s.Entry = r.t, entry
			}
			if first {
				ctx := r.in + r.cw + r.cr
				a.d.Starts++
				a.startSum += ctx
				a.d.StartMax = max(a.d.StartMax, ctx)
			}
			if v := fu.names[r.ver]; v != "" {
				vd := a.vers[v]
				if vd == nil {
					vd = &VersionDrain{Version: v}
					a.vers[v] = vd
				}
				vd.Responses++
				vd.PerResp += fresh // summed here, averaged in finish
			}
			if !prev.IsZero() && r.t.Sub(prev) > PauseMin && r.cw > RebuildMin {
				a.d.Rebuilds++
				a.d.RebuildTokens += r.cw
			}
		}
		prev, first = r.t, false
	}
}

func (a *drainAcc) finish() Drain {
	d := a.d
	if d.Starts > 0 {
		d.StartAvg = a.startSum / d.Starts
	}
	if a.allIn > 0 {
		d.CacheHit = float64(a.read) / float64(a.allIn)
	}
	for _, v := range a.vers {
		if v.Responses > 0 {
			v.PerResp /= v.Responses
		}
		d.ByVersion = append(d.ByVersion, *v)
	}
	sort.Slice(d.ByVersion, func(i, j int) bool { return versionLess(d.ByVersion[j].Version, d.ByVersion[i].Version) })
	for _, s := range a.sessions {
		if s.Fresh > 0 {
			d.Top = append(d.Top, *s)
		}
	}
	sort.Slice(d.Top, func(i, j int) bool { return d.Top[i].Fresh > d.Top[j].Fresh })
	if len(d.Top) > 10 {
		d.Top = d.Top[:10]
	}
	return d
}

// versionLess compares dotted versions numerically (2.1.99 < 2.1.300).
func versionLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, nb := atoi(pa[i]), atoi(pb[i])
		if na != nb {
			return na < nb
		}
	}
	return len(pa) < len(pb)
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// Finding is one explanation of where tokens went, with its share of the fresh tokens (0..1). ID and Args
// let the TUI say it in the interface language; Text and Hint are the English sentences.
type Finding struct {
	Share float64
	ID    string   // subagents | rebuilds | headless | start | top | cache | version
	Args  []string // the values in the sentence, in order
	Text  string
	Hint  string
}

// Texts are the English sentences of each finding (%s are Args) and what to do about it.
var Texts = map[string][2]string{
	"subagents": {"Subagents spent %s fresh tokens (%s).",
		"Each subagent starts its own context and re-reads files. The Agents tab shows which ones and who launched them."},
	"rebuilds": {"The cache was rebuilt %s times after a pause of more than 5 minutes: %s tokens written again (%s).",
		"The prompt cache expires after a pause; the next message sends the whole conversation again. Long breaks in a big conversation cost the most: /compact or a new session before a break helps."},
	"headless": {"Headless runs (claude -p and the Agent SDK) spent %s (%s).",
		"Scripts, hooks and tools that start claude -p count against the same limits. Check what runs them."},
	"start": {"Each conversation starts with about %s tokens of context before your first message (largest %s).",
		"System prompt, tools, MCP servers, CLAUDE.md and skill descriptions. The Context tab shows what to trim."},
	"top": {"One conversation in %s (%s) spent %s (%s).",
		"Very long conversations re-send a big context with every message. Start a new one when the task changes."},
	"cache": {"Only %s of the input was read from the cache.",
		"Frequent changes to CLAUDE.md, tools or MCP servers in the middle of a conversation invalidate the cache."},
	"version": {"With Claude Code %s each response used %s fresh tokens on average, against %s with %s.",
		"Compare the changelog of that version; it can also be a different kind of work."},
}

// Explain turns the drain into findings, biggest share first. Only facts above small thresholds are said.
func (d Drain) Explain(human func(int) string) []Finding {
	if d.Fresh == 0 {
		return nil
	}
	var out []Finding
	share := func(n int) float64 { return float64(n) / float64(d.Fresh) }
	pct := func(n int) string { return itoaPct(share(n)) }
	add := func(sh float64, id string, args ...string) {
		t := Texts[id]
		vals := make([]any, len(args))
		for i, a := range args {
			vals[i] = a
		}
		out = append(out, Finding{Share: sh, ID: id, Args: args, Text: fmt.Sprintf(t[0], vals...), Hint: t[1]})
	}
	if d.Subagents > 0 && share(d.Subagents) >= 0.1 {
		add(share(d.Subagents), "subagents", human(d.Subagents), pct(d.Subagents))
	}
	if d.RebuildTokens > 0 && share(d.RebuildTokens) >= 0.03 {
		add(share(d.RebuildTokens), "rebuilds", strconv.Itoa(d.Rebuilds), human(d.RebuildTokens), pct(d.RebuildTokens))
	}
	if n := d.ByEntry["sdk-cli"] + d.ByEntry["sdk-ts"] + d.ByEntry["sdk-py"]; n > 0 && share(n) >= 0.05 {
		add(share(n), "headless", human(n), pct(n))
	}
	if d.Starts > 0 && d.StartAvg > 15_000 {
		add(0, "start", human(d.StartAvg), human(d.StartMax))
	}
	if len(d.Top) > 0 && share(d.Top[0].Fresh) >= 0.1 {
		t := d.Top[0]
		add(share(t.Fresh), "top", filepath.Base(t.Project), t.Start.Local().Format("02 Jan"), human(t.Fresh), pct(t.Fresh))
	}
	if d.CacheHit > 0 && d.CacheHit < 0.7 {
		add(0, "cache", itoaPct(d.CacheHit))
	}
	if len(d.ByVersion) >= 2 {
		a, b := d.ByVersion[0], d.ByVersion[1]
		if a.Responses >= 50 && b.Responses >= 50 && b.PerResp > 0 && float64(a.PerResp) > 1.3*float64(b.PerResp) {
			add(0, "version", a.Version, human(a.PerResp), human(b.PerResp), b.Version)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Share > out[j].Share })
	return out
}

func itoaPct(f float64) string { return strconv.Itoa(int(f*100+0.5)) + "%" }
