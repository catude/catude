package sessions

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
	"sync"
	"time"
)

// DayUsage is the token usage of one day (local time), from the API usage recorded in transcripts.
// Responses are de-duplicated (message.id + requestId); subagent work is included, because it is billed.
type DayUsage struct {
	Day       time.Time
	Input     int
	Output    int
	CacheW    int
	CacheR    int
	Responses int
	ByModel   map[string]int // input+output+cache writes per model
}

// Billable is what counts towards limits most: new input, cache writes and output (cache reads are cheap).
func (d DayUsage) Billable() int { return d.Input + d.CacheW + d.Output }

// ProjectUsage aggregates a project over the period.
type ProjectUsage struct {
	Dir      string
	Billable int
}

// UsageReport covers the last N days.
type UsageReport struct {
	Days     []DayUsage
	Projects []ProjectUsage
	Models   map[string]int
	Files    int
	Drain    Drain       // where the fresh tokens went
	Health   []HealthDay // one per day, oldest first
}

type fileKey struct {
	path string
	size int64
	mod  time.Time
}

type fileUsage struct {
	recs   []respRec   // every unique response of the file, in file order
	events []healthEvt // failed tool calls and compactions
	names  []string    // entry points, versions and models referenced by recs
	cwd    string      // working folder of the conversation
}

// respRec is one API response: when, its tokens, how Claude Code was started, which version and model.
// key (message id + request id) de-duplicates across files too: a conversation resumed or forked into
// another folder copies its history, and the copied responses must not be counted twice.
type respRec struct {
	t                 time.Time
	key               string
	in, cw, cr, out   int
	calls             int    // tool calls in the response
	entry, ver, model uint16 // indexes into fileUsage.names
}

// healthEvt is a tool call that came back as an error, a compaction, or an API error Claude Code retried.
// uuid de-duplicates copies.
type healthEvt struct {
	t    time.Time
	uuid string
	kind byte // 'e' failed tool call, 'c' compaction, 'r' API error retried
}

// HealthDay is how the conversations of one day went: tool calls, how many failed, compactions, and
// fresh tokens per response. A day that stands out is worth a look (a new version, a new MCP server).
type HealthDay struct {
	Day         time.Time
	Responses   int
	ToolCalls   int
	ToolErrors  int
	Compactions int
	APIRetries  int // API errors Claude Code retried (overloaded, rate limited, network)
	Fresh       int
	CacheRead   int
	InputAll    int // new input + cache writes + cache reads
}

// CacheHit is the share of the input read from the cache that day.
func (h HealthDay) CacheHit() float64 {
	if h.InputAll == 0 {
		return 0
	}
	return float64(h.CacheRead) / float64(h.InputAll)
}

// ErrorRate is the share of tool calls that failed.
func (h HealthDay) ErrorRate() float64 {
	if h.ToolCalls == 0 {
		return 0
	}
	return float64(h.ToolErrors) / float64(h.ToolCalls)
}

// UsageCache remembers per-file results so refreshing only reads transcripts that changed.
type UsageCache struct {
	mu sync.Mutex
	m  map[fileKey]fileUsage
}

// Usage aggregates every transcript (main thread and subagents) touched in the last `days` days.
func (c *UsageCache) Usage(ctx context.Context, projects []Project, days int) (UsageReport, error) {
	since := time.Now().AddDate(0, 0, -days+1)
	since = time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, time.Local)
	rep := UsageReport{Models: map[string]int{}}
	days2 := map[string]*DayUsage{}
	dr := newDrain()
	seenAll := map[string]bool{}
	health := map[string]*HealthDay{}
	for i := 0; i < days; i++ {
		d := since.AddDate(0, 0, i)
		days2[d.Format("2006-01-02")] = &DayUsage{Day: d, ByModel: map[string]int{}}
		health[d.Format("2006-01-02")] = &HealthDay{Day: d}
	}
	type file struct {
		project string
		path    string
		fu      fileUsage
	}
	var all []file
	for _, p := range projects {
		if p.Latest.Before(since) {
			continue
		}
		for _, s := range p.Sessions {
			sub, _ := globSubagents(s)
			for _, f := range append([]string{s}, sub...) {
				if ctx.Err() != nil {
					return rep, ctx.Err()
				}
				st, err := os.Stat(f)
				if err != nil || st.ModTime().Before(since) {
					continue
				}
				all = append(all, file{p.Dir, f, c.get(fileKey{f, st.Size(), st.ModTime()})})
				rep.Files++
			}
		}
	}
	// A response found in several files (a conversation resumed or forked into another folder copies its
	// history) belongs to the original: the file that started first, and of equal starts the shorter one,
	// since the copy goes on growing.
	first := func(f file) time.Time {
		if len(f.fu.recs) == 0 {
			return time.Time{}
		}
		return f.fu.recs[0].t
	}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := first(all[i]), first(all[j])
		if !a.Equal(b) {
			return a.Before(b)
		}
		if len(all[i].fu.recs) != len(all[j].fu.recs) {
			return len(all[i].fu.recs) < len(all[j].fu.recs)
		}
		return all[i].path < all[j].path
	})
	byProject := map[string]int{}
	for _, f := range all {
		fu := f.fu
		counted := map[string]bool{}
		for _, r := range fu.recs {
			if r.t.Before(since) || seenAll[r.key] {
				continue
			}
			seenAll[r.key] = true
			counted[r.key] = true
			t := days2[r.t.Local().Format("2006-01-02")]
			if t == nil {
				continue
			}
			t.Input += r.in
			t.Output += r.out
			t.CacheW += r.cw
			t.CacheR += r.cr
			t.Responses++
			model := fu.names[r.model]
			t.ByModel[model] += r.in + r.cw + r.out
			rep.Models[model] += r.in + r.cw + r.out
			byProject[f.project] += r.in + r.cw + r.out
			if h := health[r.t.Local().Format("2006-01-02")]; h != nil {
				h.Responses++
				h.ToolCalls += r.calls
				h.Fresh += r.in + r.cw + r.out
				h.CacheRead += r.cr
				h.InputAll += r.in + r.cw + r.cr
			}
		}
		for _, e := range fu.events {
			if e.t.Before(since) || seenAll["e:"+e.uuid] {
				continue
			}
			seenAll["e:"+e.uuid] = true
			if h := health[e.t.Local().Format("2006-01-02")]; h != nil {
				switch e.kind {
				case 'c':
					h.Compactions++
				case 'r':
					h.APIRetries++
				default:
					h.ToolErrors++
				}
			}
		}
		dr.add(firstNonEmptyStr(fu.cwd, f.project), f.path, fu, func(r respRec) bool { return counted[r.key] })
	}
	for dir, n := range byProject {
		if n > 0 {
			rep.Projects = append(rep.Projects, ProjectUsage{dir, n})
		}
	}
	for i := 0; i < days; i++ {
		rep.Days = append(rep.Days, *days2[since.AddDate(0, 0, i).Format("2006-01-02")])
		rep.Health = append(rep.Health, *health[since.AddDate(0, 0, i).Format("2006-01-02")])
	}
	rep.Drain = dr.finish()
	sort.Slice(rep.Projects, func(i, j int) bool { return rep.Projects[i].Billable > rep.Projects[j].Billable })
	return rep, nil
}

func globSubagents(transcript string) ([]string, error) {
	dir := transcript[:len(transcript)-len(".jsonl")]
	ents, err := os.ReadDir(dir + string(os.PathSeparator) + "subagents")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() && len(e.Name()) > 6 && e.Name()[len(e.Name())-6:] == ".jsonl" {
			out = append(out, dir+string(os.PathSeparator)+"subagents"+string(os.PathSeparator)+e.Name())
		}
	}
	return out, nil
}

func (c *UsageCache) get(k fileKey) fileUsage {
	c.mu.Lock()
	if c.m == nil {
		c.m = map[fileKey]fileUsage{}
	}
	if fu, ok := c.m[k]; ok {
		c.mu.Unlock()
		return fu
	}
	c.mu.Unlock()
	fu := scanUsage(k.path)
	c.mu.Lock()
	c.m[k] = fu
	c.mu.Unlock()
	return fu
}

var (
	kToolUse  = []byte(`"type":"tool_use"`)
	kIsError  = []byte(`"is_error":true`)
	kAPIError = []byte(`"subtype":"api_error"`)
)

type usageLine struct {
	Type       string `json:"type"`
	RequestID  string `json:"requestId"`
	Timestamp  string `json:"timestamp"`
	Entrypoint string `json:"entrypoint"`
	Version    string `json:"version"`
	Cwd        string `json:"cwd"`
	Message    struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Usage *Usage `json:"usage"`
	} `json:"message"`
}

func scanUsage(path string) fileUsage {
	var fu fileUsage
	f, err := os.Open(path)
	if err != nil {
		return fu
	}
	defer f.Close()
	br := bufio.NewReaderSize(f, 1<<20)
	seen := map[string]int{} // key → index in recs
	idx := map[string]uint16{}
	name := func(s string) uint16 {
		if i, ok := idx[s]; ok {
			return i
		}
		fu.names = append(fu.names, s)
		idx[s] = uint16(len(fu.names) - 1)
		return idx[s]
	}
	for {
		raw, err := readLine(br)
		if len(raw) > 0 && bytes.Contains(raw, kAssistant) && bytes.Contains(raw, kUsage) {
			var l usageLine
			if json.Unmarshal(raw, &l) == nil && l.Message.Usage != nil && l.Message.Model != "<synthetic>" {
				ts, err := time.Parse(time.RFC3339Nano, l.Timestamp)
				if err == nil {
					key := l.Message.ID + "|" + l.RequestID
					u := *l.Message.Usage
					if fu.cwd == "" {
						fu.cwd = l.Cwd
					}
					calls := bytes.Count(raw, kToolUse) // each content block of a response is its own line
					if i, dup := seen[key]; !dup {
						fu.recs = append(fu.recs, respRec{t: ts, key: key, in: u.Input, cw: u.CacheCreate, cr: u.CacheRead, out: u.Output, calls: calls,
							entry: name(l.Entrypoint), ver: name(l.Version), model: name(l.Message.Model)})
						seen[key] = len(fu.recs) - 1
					} else {
						fu.recs[i].calls += calls
						if u.Output > fu.recs[i].out { // later line of the same streamed response
							fu.recs[i].out = u.Output
						}
					}
				}
			}
		} else if len(raw) > 0 && (bytes.Contains(raw, kIsError) || bytes.Contains(raw, kCompact) || bytes.Contains(raw, kAPIError)) {
			var l struct {
				UUID        string `json:"uuid"`
				Timestamp   string `json:"timestamp"`
				Subtype     string `json:"subtype"`
				IsSidechain bool   `json:"isSidechain"`
			}
			if json.Unmarshal(raw, &l) == nil && l.UUID != "" {
				if ts, err := time.Parse(time.RFC3339Nano, l.Timestamp); err == nil {
					switch {
					case l.Subtype == "compact_boundary" && !l.IsSidechain:
						fu.events = append(fu.events, healthEvt{ts, l.UUID, 'c'})
					case l.Subtype == "api_error":
						fu.events = append(fu.events, healthEvt{ts, l.UUID, 'r'})
					case l.Subtype == "" && bytes.Contains(raw, kIsError):
						fu.events = append(fu.events, healthEvt{ts, l.UUID, 'e'})
					}
				}
			}
		}
		if errors.Is(err, io.EOF) || (err != nil && !errors.Is(err, errLineTooLong)) {
			return fu
		}
	}
}

func firstNonEmptyStr(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
