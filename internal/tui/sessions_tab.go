package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/sessions"
)

// Session actions requested most by Claude Code users: search every conversation, resume from any
// folder, export to Markdown, and remove sessions (to catude's trash, restorable).

type searchState struct {
	typing  bool
	query   string
	running bool
	hits    []sessions.Hit
	sel     int
	err     string
}

type searchMsg struct {
	query string
	hits  []sessions.Hit
	err   error
}

type exportedMsg struct {
	path string
	err  error
}

type trashedMsg struct {
	dir, path string
	err       error
}

type resumedMsg struct{ err error }

type projInfo struct {
	cwd    map[string]string // project dir → working directory
	latest *sessions.Session
}

type projInfoMsg projInfo

// loadProjInfo finds each project's folder from the first bytes of its newest transcript and measures
// the latest session, off the UI thread.
func (m model) loadProjInfo(ps []sessions.Project) tea.Cmd {
	c := m.cache
	return func() tea.Msg {
		out := projInfo{cwd: map[string]string{}}
		for _, p := range ps {
			for i, f := range p.Sessions {
				if i >= 3 {
					break
				}
				if cwd, _ := sessions.QuickInfo(f); cwd != "" {
					out.cwd[p.Dir] = cwd
					break
				}
			}
		}
		if len(ps) > 0 {
			if s, err := c.Get(ps[0].Sessions[0]); err == nil {
				out.latest = &s
			}
		}
		return projInfoMsg(out)
	}
}

func (m model) projectCWD(p sessions.Project) string {
	if c, ok := m.projCWD[p.Dir]; ok {
		return c
	}
	return filepath.Base(p.Dir)
}

func (m *model) runSearch() tea.Cmd {
	q, ps := m.search.query, m.projects
	m.search.running, m.search.hits, m.search.sel, m.search.err = true, nil, 0, ""
	m.level = 3
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		hits, err := sessions.Search(ctx, ps, q, 200)
		return searchMsg{q, hits, err}
	}
}

// selectedSession is the transcript under the cursor (session list or search results).
func (m model) selectedSession() (path, title string) {
	switch m.level {
	case 1, 2:
		if m.level == 2 && m.live.Path != "" {
			return m.live.Path, m.live.Title
		}
		if m.sessSel < len(m.sessList) {
			s := m.sessList[m.sessSel]
			return s.Path, s.Title
		}
	case 3:
		if m.search.sel < len(m.search.hits) {
			h := m.search.hits[m.search.sel]
			return h.Path, h.Title
		}
	}
	return "", ""
}

func (m model) resumeSession() (tea.Model, tea.Cmd) {
	path, _ := m.selectedSession()
	if path == "" {
		m.status = "choose a session first"
		return m, nil
	}
	id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	if !sessions.ValidID(id) {
		m.status = "cannot resume: " + clipEnd(clean(id), 40) + " is not a session id"
		return m, nil
	}
	cwd, _ := sessions.QuickInfo(path)
	if !sessions.LocalPath(cwd) {
		m.status = "cannot resume: the session's folder is not a local path"
		return m, nil
	}
	if fi, err := os.Stat(cwd); cwd == "" || err != nil || !fi.IsDir() {
		m.status = "cannot resume: the session's folder no longer exists (" + cwd + ")"
		return m, nil
	}
	bin, err := exec.LookPath("claude")
	if err != nil {
		m.status = "claude is not on PATH"
		return m, nil
	}
	cmd := exec.Command(bin, "--resume", id)
	cmd.Dir = cwd
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return resumedMsg{err} })
}

func (m model) exportSession() tea.Cmd {
	path, _ := m.selectedSession()
	if path == "" {
		return nil
	}
	c := m.cache
	return func() tea.Msg {
		s, err := c.Get(path)
		if err != nil {
			return exportedMsg{err: err}
		}
		msgs, err := sessions.Messages(path, false)
		if err != nil {
			return exportedMsg{err: err}
		}
		home, _ := os.UserHomeDir()
		dir := filepath.Join(home, "claude-exports")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return exportedMsg{err: err}
		}
		name := s.Start.Local().Format("2006-01-02") + "-" + slug(firstNonEmpty(s.Title, s.ID)) + ".md"
		out := filepath.Join(dir, name)
		return exportedMsg{out, os.WriteFile(out, []byte(sessions.Markdown(s, msgs)), 0o600)} // a conversation can hold anything: private by default
	}
}

func (m *model) trashSession() tea.Cmd {
	path, title := m.selectedSession()
	if path == "" {
		return nil
	}
	if m.trashArmed != path {
		m.trashArmed = path
		m.status = "press D again to move “" + clipEnd(firstNonEmpty(title, filepath.Base(path)), 40) + "” to catude's trash (restorable)"
		return nil
	}
	m.trashArmed = ""
	claudeDir := m.env.ClaudeDir
	return func() tea.Msg {
		root, _ := os.UserCacheDir()
		dir, err := sessions.Trash(claudeDir, path, filepath.Join(root, "catude", "trash"))
		return trashedMsg{dir, path, err}
	}
}

func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
		if b.Len() > 60 {
			break
		}
	}
	return strings.Trim(b.String(), "-")
}

func (m model) searchView(h int) []string {
	var lines []string
	switch {
	case m.search.running:
		lines = append(lines, m.spinner()+" searching every conversation for “"+m.search.query+"”…")
	case m.search.err != "":
		lines = append(lines, sWarn.Render(gWarn+" "+m.search.err))
	case len(m.search.hits) == 0:
		lines = append(lines, sMuted.Render("no conversation contains “"+m.search.query+"”"))
	}
	start := clamp(m.search.sel-(h-8)/2, 0, max(0, len(m.search.hits)-(h-8)/2))
	for i := start; i < len(m.search.hits) && len(lines) < h-4; i += 1 {
		hit := m.search.hits[i]
		l1 := fmt.Sprintf("%s  %-40s %s", hit.Time.Format("02 Jan 15:04"), clipEnd(clean(firstNonEmpty(hit.Title, filepath.Base(hit.Path))), 40), sMuted.Render(shorten(hit.Project, max(10, m.w-70))))
		l2 := "    " + sMuted.Render(clipEnd(clean(hit.Snippet), max(20, m.w-12)))
		if i == m.search.sel {
			l1 = sSel.Render(gCursor + " " + stripStyles(l1, m.w-10))
		} else {
			l1 = "  " + l1
		}
		lines = append(lines, l1, l2)
	}
	return lines
}
