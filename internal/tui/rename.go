package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/live"
	"github.com/catude/catude/internal/sessions"
)

// n in the Sessions list names the selected conversation, as /rename does inside Claude Code.

type renameState struct {
	active bool
	input  string
}

func (m model) startRename() (model, tea.Cmd) {
	if m.level != 1 || m.sessSel >= len(m.sessList) {
		return m, nil
	}
	s := m.sessList[m.sessSel]
	m.ren = renameState{active: true}
	if s.Named {
		m.ren.input = s.Title
	}
	return m, nil
}

func (m model) renameKey(k tea.KeyMsg) (model, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.ren = renameState{}
	case "enter":
		name := strings.TrimSpace(m.ren.input)
		m.ren = renameState{}
		if name == "" || m.sessSel >= len(m.sessList) {
			return m, nil
		}
		s := m.sessList[m.sessSel]
		for _, l := range (&live.Reader{ClaudeDir: m.env.ClaudeDir}).Brief() {
			if l.ID == s.ID {
				m.status = T("this conversation is open in Claude Code: name it there with /rename")
				return m, nil
			}
		}
		if err := sessions.Rename(s.Path, name); err != nil {
			m.status = gFail + " " + err.Error()
			return m, nil
		}
		m.sessList[m.sessSel].Title, m.sessList[m.sessSel].Named = strings.Join(strings.Fields(name), " "), true
		m.status = gOK + " " + F("named %q · claude --resume shows this name", m.sessList[m.sessSel].Title)
	case "backspace":
		if r := []rune(m.ren.input); len(r) > 0 {
			m.ren.input = string(r[:len(r)-1])
		}
	default:
		if k.Type == tea.KeyRunes || k.String() == " " {
			m.ren.input += string(k.Runes)
		}
	}
	return m, nil
}

// summaryLines is the summary without AI of a conversation: your first message, files changed, tools.
func summaryLines(s sessions.Session, w int) []string {
	var out []string
	if s.FirstPrompt != "" {
		out = append(out, T("FIRST MESSAGE")+"  "+clipEnd(s.FirstPrompt, max(20, w-18)))
	}
	if len(s.Files) > 0 {
		var parts []string
		for _, f := range topKeys(s.Files, 6) {
			parts = append(parts, shorten(f, 40))
		}
		more := ""
		if len(s.Files) > 6 {
			more = " " + F("and %d more", len(s.Files)-6)
		}
		out = append(out, F("FILES CHANGED (%d)", len(s.Files))+"  "+clipEnd(strings.Join(parts, ", ")+more, max(20, w-22)))
	}
	if len(s.Tools) > 0 {
		var parts []string
		for _, t := range topKeys(s.Tools, 8) {
			parts = append(parts, fmt.Sprintf("%s %d", t, s.Tools[t]))
		}
		out = append(out, T("TOOLS")+"  "+clipEnd(strings.Join(parts, " · "), max(20, w-10)))
	}
	return out
}

func topKeys(m map[string]int, n int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && (m[keys[j]] > m[keys[j-1]] || m[keys[j]] == m[keys[j-1]] && keys[j] < keys[j-1]); j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	if len(keys) > n {
		keys = keys[:n]
	}
	return keys
}
