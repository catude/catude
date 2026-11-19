// Package agentmap builds the agent map: every subagent Claude Code can use here (your agents, the
// project's, plugins' and the built-in ones) joined with what they really did in the last days: how often
// they ran, from which projects, who launched them, which tools they used and how many tokens they spent.
//
// Sources, read-only:
//   - definitions: the inventory (agents/*.md of every scope and plugin);
//   - runs: ~/.claude/projects/<project>/<session>/subagents/agent-<id>.meta.json, which Claude Code writes
//     for every subagent it starts (agentType, isBuiltIn, description, parentAgentId, model, finished…),
//     and agent-<id>.jsonl next to it, the subagent's own transcript (API usage per response, tool calls).
package agentmap

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/redact"
)

// Usage is the token accounting of a run, de-duplicated by response.
type Usage struct {
	Input, CacheCreate, CacheRead, Output int
}

// Fresh is what was not a cache read: new input, cache writes and output (where cost and quota go).
func (u Usage) Fresh() int { return u.Input + u.CacheCreate + u.Output }

// Total is every token the run processed or produced.
func (u Usage) Total() int { return u.Input + u.CacheCreate + u.CacheRead + u.Output }

func (u *Usage) add(o Usage) {
	u.Input += o.Input
	u.CacheCreate += o.CacheCreate
	u.CacheRead += o.CacheRead
	u.Output += o.Output
}

// Run is one subagent started by Claude Code.
type Run struct {
	ID          string
	Type        string // agentType
	BuiltIn     bool
	Description string // the short task description Claude gave it (redacted)
	Project     string // working folder
	Session     string // parent session id
	Parent      string // type of the agent that launched it ("" = the main conversation)
	Depth       int
	Model       string
	Background  bool
	Finished    bool
	Stopped     bool // stopped by the user
	Start, End  time.Time
	Responses   int
	Usage       Usage
	Tools       map[string]int // tool name → calls
}

// Agent is one entry of the map: a definition, its runs, or both.
type Agent struct {
	Name       string
	Defined    *inventory.Item // nil for built-in agents and agents whose definition is gone
	BuiltIn    bool
	Runs       []Run // newest first
	Usage      Usage
	Tools      map[string]int
	Projects   map[string]int
	LaunchedBy map[string]int // "main" or the parent agent type → runs
	Models     map[string]int
	Stopped    int
	LastRun    time.Time
	Avg        time.Duration // average wall time of finished runs
}

// Map is the whole picture.
type Map struct {
	Agents []Agent // used ones first (by runs), then defined but unused
	Days   int
	Runs   int
	Usage  Usage
	Files  int // meta files read
}

// Unused lists defined agents that did not run in the period: candidates to remove (their descriptions
// are loaded into every session so Claude can pick them).
func (m Map) Unused() []Agent {
	var out []Agent
	for _, a := range m.Agents {
		if a.Defined != nil && len(a.Runs) == 0 {
			out = append(out, a)
		}
	}
	return out
}

type meta struct {
	AgentType     string `json:"agentType"`
	IsBuiltIn     *bool  `json:"isBuiltIn"`
	Description   string `json:"description"`
	ParentAgentID string `json:"parentAgentId"`
	SpawnDepth    int    `json:"spawnDepth"`
	RequestShape  string `json:"requestShape"`
	Model         string `json:"model"`
	Finished      bool   `json:"finished"`
	StoppedByUser bool   `json:"stoppedByUser"`
}

// Builtins are the agents Claude Code ships with (code.claude.com/docs/en/sub-agents).
var Builtins = map[string]string{
	"general-purpose":   "researching complex questions and multi-step tasks",
	"Explore":           "read-only search across a codebase",
	"Plan":              "research for plan mode",
	"fork":              "a copy of the conversation that works on a side task",
	"statusline-setup":  "configures the status line",
	"claude-code-guide": "answers questions about Claude Code and the API",
	"web-fetch":         "reads a web page for the conversation",
}

// cache of parsed transcripts, keyed by path, size and time (a run that finished never changes)
var (
	cacheMu sync.Mutex
	cache   = map[string]cached{}
)

type cached struct {
	size int64
	mod  time.Time
	run  Run
}

// Build reads the runs of the last `days` days under claudeDir and joins them with the inventory.
func Build(claudeDir string, inv inventory.Inventory, days int) Map {
	if days <= 0 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)
	metas, _ := filepath.Glob(filepath.Join(claudeDir, "projects", "*", "*", "subagents", "agent-*.meta.json"))
	var runs []Run
	byID := map[string]string{} // agent id → type, to name parents
	for _, mp := range metas {
		fi, err := os.Stat(mp)
		if err != nil || fi.ModTime().Before(since) {
			continue
		}
		r, ok := readRun(mp)
		if !ok {
			continue
		}
		byID[r.ID] = r.Type
		runs = append(runs, r)
	}
	agents := map[string]*Agent{}
	get := func(name string) *Agent {
		if a, ok := agents[name]; ok {
			return a
		}
		a := &Agent{Name: name, Tools: map[string]int{}, Projects: map[string]int{}, LaunchedBy: map[string]int{}, Models: map[string]int{}}
		_, a.BuiltIn = Builtins[name]
		agents[name] = a
		return a
	}
	for _, it := range inv.Of(inventory.Agent) {
		it := it
		a := get(it.Name)
		if a.Defined == nil {
			a.Defined = &it // the highest scope wins, as in Claude Code (Of sorts by precedence)
		}
	}
	m := Map{Days: days, Files: len(runs)}
	for i := range runs {
		r := &runs[i]
		if r.Parent != "" {
			if t, ok := byID[r.Parent]; ok {
				r.Parent = t
			} else {
				r.Parent = "agent"
			}
		}
		a := get(r.Type)
		a.BuiltIn = a.BuiltIn || r.BuiltIn
		a.Runs = append(a.Runs, *r)
		a.Usage.add(r.Usage)
		for t, n := range r.Tools {
			a.Tools[t] += n
		}
		if r.Project != "" {
			a.Projects[r.Project]++
		}
		a.LaunchedBy[firstNonEmpty(r.Parent, "main")]++
		if r.Model != "" {
			a.Models[r.Model]++
		}
		if r.Stopped {
			a.Stopped++
		}
		if r.End.After(a.LastRun) {
			a.LastRun = r.End
		}
		m.Runs++
		m.Usage.add(r.Usage)
	}
	for _, a := range agents {
		sort.Slice(a.Runs, func(i, j int) bool { return a.Runs[i].Start.After(a.Runs[j].Start) })
		var total time.Duration
		n := 0
		for _, r := range a.Runs {
			if r.Finished && !r.Start.IsZero() && r.End.After(r.Start) {
				total += r.End.Sub(r.Start)
				n++
			}
		}
		if n > 0 {
			a.Avg = total / time.Duration(n)
		}
		m.Agents = append(m.Agents, *a)
	}
	sort.Slice(m.Agents, func(i, j int) bool {
		a, b := m.Agents[i], m.Agents[j]
		if len(a.Runs) != len(b.Runs) {
			return len(a.Runs) > len(b.Runs)
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	return m
}

func readRun(metaPath string) (Run, bool) {
	b, err := readSmall(metaPath, 64<<10)
	if err != nil {
		return Run{}, false
	}
	var mt meta
	if json.Unmarshal(b, &mt) != nil || mt.AgentType == "" {
		return Run{}, false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(metaPath), "agent-"), ".meta.json")
	transcript := strings.TrimSuffix(metaPath, ".meta.json") + ".jsonl"
	r := Run{ID: id, Type: clean(mt.AgentType, 80), Description: clean(redact.Text(mt.Description), 120), Depth: mt.SpawnDepth,
		Model: clean(mt.Model, 60), Background: mt.RequestShape == "background", Finished: mt.Finished, Stopped: mt.StoppedByUser,
		Session: filepath.Base(filepath.Dir(filepath.Dir(metaPath))), Parent: mt.ParentAgentID}
	if mt.IsBuiltIn != nil {
		r.BuiltIn = *mt.IsBuiltIn
	}
	if fi, err := os.Stat(transcript); err == nil {
		cacheMu.Lock()
		c, ok := cache[transcript]
		cacheMu.Unlock()
		if !ok || c.size != fi.Size() || !c.mod.Equal(fi.ModTime()) {
			c = cached{size: fi.Size(), mod: fi.ModTime(), run: readTranscript(transcript)}
			cacheMu.Lock()
			cache[transcript] = c
			cacheMu.Unlock()
		}
		t := c.run
		r.Project, r.Start, r.End, r.Responses, r.Usage, r.Tools = t.Project, t.Start, t.End, t.Responses, t.Usage, t.Tools
		if r.Model == "" {
			r.Model = t.Model
		}
	}
	if r.Tools == nil {
		r.Tools = map[string]int{}
	}
	return r, true
}

// readTranscript sums the API usage (one count per message id + request id) and the tool calls.
func readTranscript(p string) Run {
	r := Run{Tools: map[string]int{}}
	f, err := os.Open(p)
	if err != nil {
		return r
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	seen := map[string]bool{}
	for sc.Scan() {
		line := sc.Bytes()
		var l struct {
			Type      string `json:"type"`
			Cwd       string `json:"cwd"`
			Timestamp string `json:"timestamp"`
			RequestID string `json:"requestId"`
			Message   struct {
				ID      string          `json:"id"`
				Model   string          `json:"model"`
				Content json.RawMessage `json:"content"`
				Usage   *struct {
					Input       int `json:"input_tokens"`
					CacheCreate int `json:"cache_creation_input_tokens"`
					CacheRead   int `json:"cache_read_input_tokens"`
					Output      int `json:"output_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &l) != nil {
			continue
		}
		if t, err := time.Parse(time.RFC3339Nano, l.Timestamp); err == nil {
			if r.Start.IsZero() || t.Before(r.Start) {
				r.Start = t
			}
			if t.After(r.End) {
				r.End = t
			}
		}
		if r.Project == "" && l.Cwd != "" {
			r.Project = clean(l.Cwd, 400)
		}
		if l.Type != "assistant" {
			continue
		}
		if l.Message.Model != "" && !strings.HasPrefix(l.Message.Model, "<") {
			r.Model = clean(l.Message.Model, 60)
		}
		key := l.Message.ID + "|" + l.RequestID
		if u := l.Message.Usage; u != nil && !seen[key] {
			seen[key] = true
			r.Responses++
			r.Usage.add(Usage{u.Input, u.CacheCreate, u.CacheRead, u.Output})
		}
		var blocks []struct {
			Type string `json:"type"`
			Name string `json:"name"`
		}
		if json.Unmarshal(l.Message.Content, &blocks) == nil {
			for _, b := range blocks {
				if b.Type == "tool_use" && b.Name != "" {
					r.Tools[clean(b.Name, 80)]++
				}
			}
		}
	}
	return r
}

func readSmall(p string, max int64) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b := make([]byte, max)
	n, _ := f.Read(b)
	return b[:n], nil
}

// clean removes control characters (terminal escapes) and shortens.
func clean(s string, n int) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return ' '
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		s = string(r[:n-1]) + "…"
	}
	return s
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// Top returns the keys of a count map, highest first, at most n.
func Top(m map[string]int, n int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]] != m[keys[j]] {
			return m[keys[i]] > m[keys[j]]
		}
		return keys[i] < keys[j]
	})
	if len(keys) > n {
		keys = keys[:n]
	}
	return keys
}
