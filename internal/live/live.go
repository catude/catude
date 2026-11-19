// Package live shows the Claude Code sessions running right now and what each one is doing.
//
// Sources, both on this machine only:
//   - ~/.claude/sessions/<pid>.json — "one small file per running session, used to detect concurrent
//     sessions and crashes" (Claude Code docs, claude-directory). It names the session, its folder,
//     version, entry point (cli, VS Code…) and status. Only <pid>.json files are read; the .key files
//     next to them are never opened.
//   - the end of the session's transcript (~/.claude/projects/<folder>/<session>.jsonl): the tools being
//     run, their results, the model, the effort level, the to-do list and the last text.
//
// Commands and paths are redacted and shortened before they are shown.
package live

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/catude/catude/internal/redact"
)

// Step is one tool call.
type Step struct {
	ID     string
	Tool   string // Bash, Edit, Read, Agent, mcp server › tool…
	Detail string // what it acts on: a command, a file, a query (redacted)
	At     time.Time
	Done   bool
	Err    bool
	DoneAt time.Time
	Sub    []Step // what a subagent is doing (Agent / Task steps)
}

// Todo is the session's to-do list (TodoWrite).
type Todo struct {
	Done, Total int
	Current     string
}

// Session is one running Claude Code session.
type Session struct {
	PID        int
	ID         string
	Name       string // Claude Code's short name (storefront-01)
	Title      string // the session title, when Claude Code generated one
	Cwd        string
	Version    string
	Entrypoint string // cli | claude-vscode | …
	Status     string // busy | idle | … as Claude Code writes it
	Updated    time.Time
	Started    time.Time
	Alive      bool
	Transcript string
	Model      string
	Effort     string
	Context    int // tokens in the context window after the last answer
	Steps      []Step
	Todo       Todo
	LastText   string
	Activity   time.Time // last line written to the transcript
	// Unrecognized: the transcript has content but no line in a shape catude understands (Claude Code
	// changed its format); the card says so instead of looking empty
	Unrecognized bool
}

// Working says whether the session is doing something now.
func (s Session) Working() bool {
	if s.Status == "busy" {
		return true
	}
	for _, st := range s.Steps {
		if !st.Done && time.Since(st.At) < 10*time.Minute {
			return true
		}
	}
	return false
}

// Running is the tool call in progress, if any.
func (s Session) Running() (Step, bool) {
	for i := len(s.Steps) - 1; i >= 0; i-- {
		if !s.Steps[i].Done {
			return s.Steps[i], true
		}
	}
	return Step{}, false
}

var (
	rePIDFile = regexp.MustCompile(`^\d{1,10}\.json$`)
	reID      = regexp.MustCompile(`^[0-9a-fA-F-]{8,64}$`)
)

type sessionFile struct {
	PID        int    `json:"pid"`
	SessionID  string `json:"sessionId"`
	Cwd        string `json:"cwd"`
	StartedAt  int64  `json:"startedAt"`
	Version    string `json:"version"`
	Kind       string `json:"kind"`
	Entrypoint string `json:"entrypoint"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	UpdatedAt  int64  `json:"updatedAt"`
}

// Reader keeps what it already found between refreshes (transcript paths, parsed tails), so a refresh
// only reads what changed.
type Reader struct {
	ClaudeDir string
	Alive     func(pid int) bool // defaults to the operating system check
	paths     map[string]string  // session id → transcript
	tails     map[string]tailCache
}

type tailCache struct {
	size  int64
	mod   time.Time
	state Session
}

// MaxTail is how much of the end of a transcript is read.
const MaxTail = 768 << 10

// files reads Claude Code's per-process status files (~/.claude/sessions/<pid>.json) of live processes.
func (r *Reader) files() []Session {
	alive := r.Alive
	if alive == nil {
		alive = processAlive
	}
	dir := filepath.Join(r.ClaudeDir, "sessions")
	ents, _ := os.ReadDir(dir)
	var out []Session
	for _, e := range ents {
		if e.IsDir() || !rePIDFile.MatchString(e.Name()) {
			continue
		}
		b, err := readSmall(filepath.Join(dir, e.Name()), 64<<10)
		if err != nil {
			continue
		}
		var f sessionFile
		if json.Unmarshal(b, &f) != nil || f.PID <= 0 || !reID.MatchString(f.SessionID) {
			continue
		}
		s := Session{PID: f.PID, ID: f.SessionID, Name: clean(f.Name, 60), Cwd: clean(f.Cwd, 400), Version: clean(f.Version, 30),
			Entrypoint: clean(f.Entrypoint, 30), Status: clean(f.Status, 30)}
		if f.UpdatedAt > 0 {
			s.Updated = time.UnixMilli(f.UpdatedAt)
		}
		if f.StartedAt > 0 {
			s.Started = time.UnixMilli(f.StartedAt)
		}
		s.Alive = alive(f.PID)
		if !s.Alive {
			continue // a leftover of a session that ended or crashed
		}
		out = append(out, s)
	}
	return out
}

// Brief lists the running sessions from their small status files only, without reading transcripts:
// cheap enough to run every few seconds to notice a session that finished its turn.
func (r *Reader) Brief() []Session { return r.files() }

// Sessions lists the running sessions, working ones first, then by last activity.
func (r *Reader) Sessions() []Session {
	if r.paths == nil {
		r.paths, r.tails = map[string]string{}, map[string]tailCache{}
	}
	out := r.files()
	for i := range out {
		out[i].Transcript = r.transcript(out[i].ID)
		if out[i].Transcript != "" {
			r.fill(&out[i])
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Working() != out[j].Working() {
			return out[i].Working()
		}
		return latest(out[i]).After(latest(out[j]))
	})
	return out
}

func latest(s Session) time.Time {
	if s.Activity.After(s.Updated) {
		return s.Activity
	}
	return s.Updated
}

func (r *Reader) transcript(id string) string {
	if p, ok := r.paths[id]; ok {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	m, _ := filepath.Glob(filepath.Join(r.ClaudeDir, "projects", "*", id+".jsonl"))
	if len(m) == 0 {
		return ""
	}
	r.paths[id] = m[0]
	return m[0]
}

func readSmall(p string, max int64) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, max))
}

// fill reads the end of the transcript (again only when it changed).
func (r *Reader) fill(s *Session) {
	fi, err := os.Stat(s.Transcript)
	if err != nil || !fi.Mode().IsRegular() {
		return
	}
	if c, ok := r.tails[s.ID]; ok && c.size == fi.Size() && c.mod.Equal(fi.ModTime()) {
		copyTail(s, c.state)
		s.Steps = withSubagents(s.Transcript, s.Steps)
		return
	}
	var t Session
	t.Activity = fi.ModTime()
	parseTail(s.Transcript, fi.Size(), &t)
	r.tails[s.ID] = tailCache{fi.Size(), fi.ModTime(), t}
	copyTail(s, t)
	s.Steps = withSubagents(s.Transcript, s.Steps)
}

func copyTail(s *Session, t Session) {
	s.Unrecognized = t.Unrecognized
	s.Title, s.Model, s.Effort, s.Context, s.Todo, s.LastText, s.Activity = t.Title, t.Model, t.Effort, t.Context, t.Todo, t.LastText, t.Activity
	s.Steps = append([]Step(nil), t.Steps...)
}

type line struct {
	Type        string          `json:"type"`
	IsSidechain bool            `json:"isSidechain"`
	Timestamp   string          `json:"timestamp"`
	Effort      string          `json:"effort"`
	AITitle     string          `json:"aiTitle"`
	Title       string          `json:"title"`
	Message     json.RawMessage `json:"message"`
}

type message struct {
	Model   string          `json:"model"`
	Content json.RawMessage `json:"content"`
	Usage   *struct {
		Input       int `json:"input_tokens"`
		CacheCreate int `json:"cache_creation_input_tokens"`
		CacheRead   int `json:"cache_read_input_tokens"`
	} `json:"usage"`
}

type block struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Text      string          `json:"text"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	IsError   bool            `json:"is_error"`
}

// MaxSteps is how many recent tool calls are kept per session.
const MaxSteps = 12

func readTail(path string, size int64) []byte {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	off := int64(0)
	if size > MaxTail {
		off = size - MaxTail
	}
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return nil
	}
	b, _ := io.ReadAll(io.LimitReader(f, MaxTail))
	if off > 0 { // drop the partial first line
		if i := bytes.IndexByte(b, '\n'); i >= 0 {
			b = b[i+1:]
		}
	}
	return b
}

func parseTail(path string, size int64, s *Session) {
	b := readTail(path, size)
	byID := map[string]int{}
	var tl taskList
	objects, known := 0, 0
	defer func() {
		// many JSON lines but none of a shape this version understands: Claude Code changed its format
		s.Unrecognized = objects >= 5 && known == 0
	}()
	for _, raw := range bytes.Split(b, []byte("\n")) {
		if len(raw) == 0 || raw[0] != '{' {
			continue
		}
		objects++
		var l line
		if json.Unmarshal(raw, &l) != nil {
			continue
		}
		switch l.Type {
		case "ai-title", "summary", "custom-title":
			if t := firstNonEmpty(l.AITitle, l.Title); t != "" {
				s.Title = clean(t, 120)
			}
			known++
			continue
		case "assistant", "user":
			known++
		default:
			continue
		}
		if l.IsSidechain {
			continue
		}
		at, _ := time.Parse(time.RFC3339Nano, l.Timestamp)
		var m message
		if json.Unmarshal(l.Message, &m) != nil {
			continue
		}
		if l.Type == "assistant" {
			if m.Model != "" && m.Model != "<synthetic>" {
				s.Model = clean(m.Model, 60)
			}
			if l.Effort != "" {
				s.Effort = clean(l.Effort, 20)
			}
			if m.Usage != nil {
				s.Context = m.Usage.Input + m.Usage.CacheCreate + m.Usage.CacheRead
			}
		}
		var blocks []block
		if json.Unmarshal(m.Content, &blocks) != nil {
			continue // plain text user message
		}
		for _, bl := range blocks {
			switch {
			case bl.Type == "tool_use" && l.Type == "assistant":
				st := Step{ID: bl.ID, Tool: toolName(bl.Name), Detail: detail(bl.Name, bl.Input), At: at}
				if bl.Name == "TodoWrite" {
					s.Todo = todos(bl.Input)
					continue // shown as the to-do line, not as a step
				}
				if bl.Name == "TaskCreate" || bl.Name == "TaskUpdate" {
					tl.apply(bl.Name, bl.Input)
					s.Todo = tl.todo()
					continue
				}
				byID[bl.ID] = len(s.Steps)
				s.Steps = append(s.Steps, st)
			case bl.Type == "tool_result" && l.Type == "user":
				if i, ok := byID[bl.ToolUseID]; ok {
					s.Steps[i].Done, s.Steps[i].Err, s.Steps[i].DoneAt = true, bl.IsError, at
				}
			case bl.Type == "text" && l.Type == "assistant" && strings.TrimSpace(bl.Text) != "":
				s.LastText = clean(plain(firstLine(bl.Text)), 200)
			}
		}
	}
	if len(s.Steps) > MaxSteps {
		s.Steps = s.Steps[len(s.Steps)-MaxSteps:]
	}
	// a tool that never got its result in an old part of the transcript is not "running" any more
	for i := range s.Steps {
		if !s.Steps[i].Done && i < len(s.Steps)-3 {
			s.Steps[i].Done = true
		}
	}
}

// withSubagents adds, under each running Agent/Task step, the latest tool calls of the subagents that
// wrote to their transcript in the last two minutes.
func withSubagents(transcript string, steps []Step) []Step {
	running := -1
	for i := range steps {
		if !steps[i].Done && (steps[i].Tool == "Agent" || steps[i].Tool == "Task") {
			running = i
		}
	}
	if running < 0 {
		return steps
	}
	dir := filepath.Join(strings.TrimSuffix(transcript, ".jsonl"), "subagents")
	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	type f struct {
		p   string
		mod time.Time
		sz  int64
	}
	var fresh []f
	for _, p := range files {
		if fi, err := os.Stat(p); err == nil && time.Since(fi.ModTime()) < 2*time.Minute {
			fresh = append(fresh, f{p, fi.ModTime(), fi.Size()})
		}
	}
	sort.Slice(fresh, func(i, j int) bool { return fresh[i].mod.After(fresh[j].mod) })
	if len(fresh) > 4 {
		fresh = fresh[:4]
	}
	out := append([]Step(nil), steps...)
	for _, x := range fresh {
		var sub Session
		parseSubTail(x.p, x.sz, &sub)
		if n := len(sub.Steps); n > 0 {
			last := sub.Steps[n-1]
			last.Detail = strings.TrimSpace(last.Detail)
			out[running].Sub = append(out[running].Sub, last)
		}
	}
	return out
}

// parseSubTail is parseTail for a subagent transcript, whose lines are all sidechain.
func parseSubTail(path string, size int64, s *Session) {
	b := readTailN(path, size, min(size, 128<<10))
	byID := map[string]int{}
	for _, raw := range bytes.Split(b, []byte("\n")) {
		var l line
		if len(raw) == 0 || json.Unmarshal(raw, &l) != nil || (l.Type != "assistant" && l.Type != "user") {
			continue
		}
		at, _ := time.Parse(time.RFC3339Nano, l.Timestamp)
		var m message
		var blocks []block
		if json.Unmarshal(l.Message, &m) != nil || json.Unmarshal(m.Content, &blocks) != nil {
			continue
		}
		for _, bl := range blocks {
			if bl.Type == "tool_use" {
				byID[bl.ID] = len(s.Steps)
				s.Steps = append(s.Steps, Step{ID: bl.ID, Tool: toolName(bl.Name), Detail: detail(bl.Name, bl.Input), At: at})
			} else if bl.Type == "tool_result" {
				if i, ok := byID[bl.ToolUseID]; ok {
					s.Steps[i].Done, s.Steps[i].Err = true, bl.IsError
				}
			}
		}
	}
}

func readTailN(path string, size, n int64) []byte {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	if _, err := f.Seek(size-n, io.SeekStart); err != nil {
		return nil
	}
	b, _ := io.ReadAll(io.LimitReader(f, n))
	if size > n { // drop the partial first line
		if i := bytes.IndexByte(b, '\n'); i >= 0 {
			b = b[i+1:]
		}
	}
	return b
}

// toolName shortens MCP tool names: mcp__github__create_issue → github › create_issue.
func toolName(n string) string {
	if strings.HasPrefix(n, "mcp__") {
		parts := strings.SplitN(strings.TrimPrefix(n, "mcp__"), "__", 2)
		if len(parts) == 2 {
			srv := parts[0]
			if strings.HasPrefix(srv, "claude_ai_") {
				srv = strings.ReplaceAll(strings.TrimPrefix(srv, "claude_ai_"), "_", " ")
			}
			return clean(srv+" › "+parts[1], 60)
		}
	}
	return clean(n, 40)
}

// detail says what a tool call acts on, without secrets.
func detail(tool string, in json.RawMessage) string {
	var v map[string]any
	if json.Unmarshal(in, &v) != nil {
		return ""
	}
	str := func(keys ...string) string {
		for _, k := range keys {
			if s, ok := v[k].(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
		return ""
	}
	var d string
	switch tool {
	case "Bash", "PowerShell":
		d = str("description")
		if d == "" {
			d = firstLine(str("command"))
		}
	case "Read", "Edit", "Write", "MultiEdit", "NotebookEdit":
		d = BaseName(str("file_path", "notebook_path"))
	case "Grep", "Glob":
		d = str("pattern")
	case "WebFetch":
		d = str("url")
		if i := strings.Index(d, "://"); i >= 0 {
			d = d[i+3:]
			if j := strings.IndexAny(d, "/?#"); j >= 0 {
				d = d[:j]
			}
		}
	case "WebSearch":
		d = str("query")
	case "Agent", "Task":
		d = str("description", "subagent_type")
		if t := str("subagent_type"); t != "" && t != d {
			d = t + ": " + d
		}
	case "Skill":
		d = str("skill", "name")
	case "TodoWrite":
		d = ""
	default:
		d = str("description", "query", "path", "file_path", "name", "url", "pattern", "command")
	}
	return clean(redact.Text(firstLine(d)), 120)
}

func todos(in json.RawMessage) Todo {
	var v struct {
		Todos []struct {
			Content    string `json:"content"`
			Status     string `json:"status"`
			ActiveForm string `json:"activeForm"`
		} `json:"todos"`
	}
	var t Todo
	if json.Unmarshal(in, &v) != nil {
		return t
	}
	t.Total = len(v.Todos)
	for _, x := range v.Todos {
		switch x.Status {
		case "completed":
			t.Done++
		case "in_progress":
			if t.Current == "" {
				t.Current = clean(redact.Text(firstNonEmpty(x.ActiveForm, x.Content)), 120)
			}
		}
	}
	return t
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return s
}

// plain drops the Markdown marks that would show as noise in a terminal line.
func plain(s string) string {
	s = strings.NewReplacer("**", "", "__", "", "`", "").Replace(s)
	return strings.TrimLeft(s, "#> ")
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// clean removes control and direction characters and shortens to n runes.
func clean(s string, n int) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) || (r >= 0x200b && r <= 0x200f) || (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069) {
			return -1
		}
		return r
	}, s)
	if r := []rune(s); len(r) > n {
		s = string(r[:n-1]) + "…"
	}
	return s
}

// BaseName is the last part of a path written on any system (a Windows path read on Linux or WSL too).
func BaseName(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

// taskList follows the task tools (TaskCreate / TaskUpdate), which some models and settings use instead of
// TodoWrite. Their input format is not documented, so this reads the common field names and ignores the rest.
// Tasks are numbered in creation order, as Claude Code shows them.
type taskList struct {
	order  []string
	status map[string]string
	label  map[string]string
}

func (tl *taskList) apply(tool string, in json.RawMessage) {
	var v map[string]any
	if json.Unmarshal(in, &v) != nil {
		return
	}
	if tl.status == nil {
		tl.status, tl.label = map[string]string{}, map[string]string{}
	}
	str := func(keys ...string) string {
		for _, k := range keys {
			switch x := v[k].(type) {
			case string:
				if strings.TrimSpace(x) != "" {
					return x
				}
			case float64:
				return fmt.Sprint(int(x))
			}
		}
		return ""
	}
	switch tool {
	case "TaskCreate":
		id := fmt.Sprint(len(tl.order) + 1)
		tl.order = append(tl.order, id)
		tl.status[id] = firstNonEmpty(str("status"), "pending")
		tl.label[id] = firstNonEmpty(str("activeForm"), str("subject", "content", "title", "description"))
	case "TaskUpdate":
		id := strings.TrimPrefix(str("taskId", "task_id", "id"), "#")
		if _, ok := tl.status[id]; !ok {
			return
		}
		if st := str("status"); st != "" {
			tl.status[id] = st
		}
		if lb := firstNonEmpty(str("activeForm"), str("subject")); lb != "" {
			tl.label[id] = lb
		}
	}
}

func (tl *taskList) todo() Todo {
	var t Todo
	for _, id := range tl.order {
		switch tl.status[id] {
		case "deleted":
			continue
		case "completed":
			t.Done++
		case "in_progress":
			if t.Current == "" {
				t.Current = clean(redact.Text(tl.label[id]), 120)
			}
		}
		t.Total++
	}
	return t
}
