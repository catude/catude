// Package sessions measures Claude Code sessions exactly from the usage the API returned,
// as stored in ~/.claude/projects/<project>/<session>.jsonl. No token is estimated:
//
//   - Context in use = input + cache creation + cache read of the LAST API response
//     (summing every response, as many tools do, measures traffic, not context).
//   - Claude Code writes several lines per streamed response; each response is counted once,
//     keyed by message.id + requestId. Without this, totals are inflated about 3x.
//   - Subagent turns (isSidechain) live in their own context and are excluded.
//   - A compact_boundary resets the context: the last response before it no longer counts.
package sessions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Usage is the token accounting of one API response.
type Usage struct {
	Input       int `json:"input_tokens"`
	CacheCreate int `json:"cache_creation_input_tokens"`
	CacheRead   int `json:"cache_read_input_tokens"`
	Output      int `json:"output_tokens"`
}

// Context returns the tokens the response had in its context window.
func (u Usage) Context() int { return u.Input + u.CacheCreate + u.CacheRead }

// Session is the exact measurement of one transcript.
type Session struct {
	Path        string
	ID          string
	Title       string
	Model       string
	CWD         string
	Branch      string
	Start, End  time.Time
	Responses   int   // unique API responses (deduplicated)
	Compactions int   // compact_boundary markers
	Last        Usage // last response after the last compaction
	Total       Usage // sum of unique responses (traffic, not context)
	Bytes       int64
	Malformed   int // lines that could not be parsed (reported, never fatal)

	// a summary without AI: the name you gave it, your first message, files changed and tools used
	Named       bool           // Title is the name given with /rename (or catude), not the generated one
	FirstPrompt string         // redacted, shortened
	Files       map[string]int // files edited or written → times
	Tools       map[string]int // tool calls of the main conversation
}

// Window is the context window of the session's model.
func (s Session) Window() int { return WindowFor(s.Model) }

// ContextTokens is what the session occupies now.
func (s Session) ContextTokens() int { return s.Last.Context() }

// ContextRatio is ContextTokens / Window, between 0 and 1.
func (s Session) ContextRatio() float64 {
	w := s.Window()
	if w == 0 {
		return 0
	}
	return float64(s.ContextTokens()) / float64(w)
}

// CacheHitRatio is the share of processed input that was read from the prompt cache.
func (s Session) CacheHitRatio() float64 {
	t := s.Total.Context()
	if t == 0 {
		return 0
	}
	return float64(s.Total.CacheRead) / float64(t)
}

var legacyWindow = regexp.MustCompile(`(haiku|sonnet|opus)-4-[0-5]\b|claude-3`)

// WindowFor maps a model id to its context window. Current models (Opus/Sonnet/Haiku 5.x,
// Fable, Opus/Sonnet 4.6+) have 1M; Claude 3 and 4.0-4.5 models have 200K unless "[1m]" is requested.
func WindowFor(model string) int {
	if strings.Contains(model, "[1m]") {
		return 1_000_000
	}
	if legacyWindow.MatchString(model) {
		return 200_000
	}
	return 1_000_000
}

type line struct {
	Type        string `json:"type"`
	Subtype     string `json:"subtype"`
	IsSidechain bool   `json:"isSidechain"`
	RequestID   string `json:"requestId"`
	CWD         string `json:"cwd"`
	GitBranch   string `json:"gitBranch"`
	Timestamp   string `json:"timestamp"`
	AITitle     string `json:"aiTitle"`
	CustomTitle string `json:"customTitle"`
	IsMeta      bool   `json:"isMeta"`
	Message     struct {
		ID      string          `json:"id"`
		Model   string          `json:"model"`
		Usage   *Usage          `json:"usage"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

var (
	kAssistant = []byte(`"type":"assistant"`)
	kUsage     = []byte(`"usage"`)
	kTitle     = []byte(`"ai-title"`)
	kCustom    = []byte(`"custom-title"`)
	kUser      = []byte(`"type":"user"`)
	kCompact   = []byte(`"compact_boundary"`)
)

// MaxLine bounds a single JSONL line; longer lines are skipped and counted as malformed.
const MaxLine = 64 << 20

// Analyze reads one transcript. Unparseable lines are counted, not fatal.
func Analyze(path string) (Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return Session{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return Session{}, err
	}
	s := Session{Path: path, ID: strings.TrimSuffix(filepath.Base(path), ".jsonl"), Bytes: st.Size(), End: st.ModTime()}
	err = Scan(f, &s)
	return s, err
}

// Scan measures a transcript from any reader (used by Analyze and by tests).
func Scan(r io.Reader, s *Session) error {
	br := bufio.NewReaderSize(r, 1<<20)
	seen := map[string]bool{}
	lastKey := ""
	for {
		raw, err := readLine(br)
		if errors.Is(err, errLineTooLong) {
			s.Malformed++
			continue
		}
		if len(raw) > 0 {
			handle(raw, s, seen, &lastKey)
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

var errLineTooLong = errors.New("line too long")

func readLine(br *bufio.Reader) ([]byte, error) {
	var buf []byte
	for {
		chunk, err := br.ReadSlice('\n')
		if len(buf)+len(chunk) > MaxLine {
			for errors.Is(err, bufio.ErrBufferFull) { // discard the rest of the line
				_, err = br.ReadSlice('\n')
			}
			if err != nil && !errors.Is(err, io.EOF) {
				return nil, err
			}
			return nil, errLineTooLong
		}
		buf = append(buf, chunk...)
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return bytes.TrimRight(buf, "\r\n"), err
	}
}

func handle(raw []byte, s *Session, seen map[string]bool, lastKey *string) {
	isAssistant := bytes.Contains(raw, kAssistant) && bytes.Contains(raw, kUsage)
	wantPrompt := s.FirstPrompt == "" && bytes.Contains(raw, kUser)
	if !isAssistant && !wantPrompt && !bytes.Contains(raw, kTitle) && !bytes.Contains(raw, kCustom) && !bytes.Contains(raw, kCompact) {
		return
	}
	var l line
	if err := json.Unmarshal(raw, &l); err != nil {
		s.Malformed++
		return
	}
	if ts, err := time.Parse(time.RFC3339Nano, l.Timestamp); err == nil && (s.Start.IsZero() || ts.Before(s.Start)) {
		s.Start = ts
	}
	switch {
	case l.Type == "custom-title" && l.CustomTitle != "":
		s.Title, s.Named = l.CustomTitle, true
	case l.Type == "ai-title" && l.AITitle != "" && !s.Named:
		s.Title = l.AITitle
	case l.Type == "user" && wantPrompt && !l.IsSidechain && !l.IsMeta:
		if p := promptText(l.Message.Content); p != "" {
			s.FirstPrompt = p
		}
	case l.Type == "system" && l.Subtype == "compact_boundary" && !l.IsSidechain:
		s.Compactions++
		s.Last = Usage{} // the previous context is gone
		*lastKey = ""
	case l.Type == "assistant" && l.Message.Usage != nil && !l.IsSidechain:
		if s.CWD == "" {
			s.CWD = l.CWD
		}
		if l.GitBranch != "" {
			s.Branch = l.GitBranch
		}
		if l.Message.Model != "" && l.Message.Model != "<synthetic>" {
			s.Model = l.Message.Model
		}
		u := *l.Message.Usage
		key := l.Message.ID + "|" + l.RequestID
		switch {
		case !seen[key]:
			seen[key] = true
			s.Responses++
			s.Total.Input += u.Input
			s.Total.CacheCreate += u.CacheCreate
			s.Total.CacheRead += u.CacheRead
			s.Total.Output += u.Output
		case key == *lastKey && u.Output > s.Last.Output:
			// a later line of the same streamed response carries the final output count
			s.Total.Output += u.Output - s.Last.Output
		}
		*lastKey = key
		s.Last = u
		countTools(s, l.Message.Content)
	}
}

// Project groups the sessions of one working directory.
type Project struct {
	Dir      string // folder under ~/.claude/projects
	Sessions []string
	Latest   time.Time
	Bytes    int64
}

// ListProjects returns projects ordered by last activity. It only stats files (cheap).
func ListProjects(projectsDir string) ([]Project, error) {
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []Project
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(projectsDir, e.Name())
		files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		type fi struct {
			path string
			mod  time.Time
		}
		var fis []fi
		p := Project{Dir: dir}
		for _, f := range files {
			st, err := os.Stat(f)
			if err != nil {
				continue
			}
			p.Bytes += st.Size()
			fis = append(fis, fi{f, st.ModTime()})
			if st.ModTime().After(p.Latest) {
				p.Latest = st.ModTime()
			}
		}
		if len(fis) == 0 {
			continue
		}
		sort.Slice(fis, func(i, j int) bool { return fis[i].mod.After(fis[j].mod) })
		for _, x := range fis {
			p.Sessions = append(p.Sessions, x.path)
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Latest.After(out[j].Latest) })
	return out, nil
}

// Cache memoises Analyze by path, size and modification time. Safe for concurrent use.
type Cache struct {
	mu sync.Mutex
	m  map[string]cached
}

type cached struct {
	size int64
	mod  time.Time
	s    Session
}

// Get returns the measurement, re-reading the file only if it changed.
func (c *Cache) Get(path string) (Session, error) {
	st, err := os.Stat(path)
	if err != nil {
		return Session{}, err
	}
	c.mu.Lock()
	if c.m == nil {
		c.m = map[string]cached{}
	}
	v, ok := c.m[path]
	c.mu.Unlock()
	if ok && v.size == st.Size() && v.mod.Equal(st.ModTime()) {
		return v.s, nil
	}
	s, err := Analyze(path)
	if err != nil {
		return s, err
	}
	c.mu.Lock()
	c.m[path] = cached{st.Size(), st.ModTime(), s}
	c.mu.Unlock()
	return s, nil
}

// ProjectCWD returns the real working directory of a project, reading at most a few sessions.
func (c *Cache) ProjectCWD(p Project) string {
	for i, f := range p.Sessions {
		if i >= 3 {
			break
		}
		if s, err := c.Get(f); err == nil && s.CWD != "" {
			return s.CWD
		}
	}
	return filepath.Base(p.Dir)
}
