package tui

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/live"
)

// Every 2 seconds (the main tick) catude reads Claude Code's small status files (~/.claude/sessions,
// no transcripts) and, in any tab, says when a session finishes its turn: busy before, anything else now.
// With the Bell preference on it also rings the terminal bell.

type watchState struct {
	reader  *live.Reader
	running bool
	status  map[string]string // session id → status at the last read; nil before the first read
}

type watchMsg struct{ ss []live.Session }

func (m *model) watchSessions() tea.Cmd {
	if m.wt.running || m.env.ClaudeDir == "" {
		return nil
	}
	if m.wt.reader == nil {
		m.wt.reader = &live.Reader{ClaudeDir: m.env.ClaudeDir}
	}
	m.wt.running = true
	r := m.wt.reader
	return func() tea.Msg { return watchMsg{r.Brief()} }
}

func (m *model) onWatch(ss []live.Session) tea.Cmd {
	m.wt.running = false
	prev := m.wt.status
	m.wt.status = map[string]string{}
	var done []string
	for _, s := range ss {
		m.wt.status[s.ID] = s.Status
		if prev != nil && prev[s.ID] == "busy" && s.Status != "busy" {
			done = append(done, sessionLabel(s))
		}
	}
	if len(done) == 0 {
		return nil
	}
	m.status = gOK + " " + F("%s finished and is waiting for you", strings.Join(done, ", "))
	var cmds []tea.Cmd
	if m.prefs.Bell {
		cmds = append(cmds, ringBell)
	}
	if m.prefs.Notify {
		cmds = append(cmds, desktopNotify("Claude Code", F("%s finished and is waiting for you", strings.Join(done, ", "))))
	}
	return tea.Batch(cmds...)
}

func sessionLabel(s live.Session) string {
	if s.Name != "" {
		return s.Name
	}
	if s.Cwd != "" {
		return filepath.Base(s.Cwd)
	}
	return s.ID[:min(8, len(s.ID))]
}

// ringBell writes one BEL byte in a single write, so it never lands inside the screen update.
var ringBell = func() tea.Msg {
	os.Stdout.Write([]byte("\a"))
	return nil
}
