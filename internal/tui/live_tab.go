package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/live"
	"github.com/catude/catude/internal/sessions"
)

// Live tab: the Claude Code sessions running right now and what each one is doing, refreshed every
// second while the tab is open.

type liveState struct {
	reader  *live.Reader
	list    []live.Session
	at      time.Time
	running bool
	sel     int
	started bool // the 1-second refresh loop is running
}

type liveTickMsg time.Time
type liveListMsg struct{ ss []live.Session }

func liveTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return liveTickMsg(t) })
}

// ensureLive reads the sessions now and starts the refresh loop (it stops by itself when the tab closes).
func (m *model) ensureLive() tea.Cmd {
	if m.lv.reader == nil {
		m.lv.reader = &live.Reader{ClaudeDir: m.env.ClaudeDir}
	}
	cmds := []tea.Cmd{m.readLive()}
	if !m.lv.started {
		m.lv.started = true
		cmds = append(cmds, liveTick())
	}
	return tea.Batch(cmds...)
}

func (m *model) readLive() tea.Cmd {
	if m.lv.running || m.lv.reader == nil {
		return nil
	}
	m.lv.running = true
	r := m.lv.reader
	return func() tea.Msg { return liveListMsg{r.Sessions()} }
}

func (m *model) onLiveTick() tea.Cmd {
	if m.tab != tLive {
		m.lv.started = false // stop refreshing while another tab is shown
		return nil
	}
	m.spin++
	return tea.Batch(m.readLive(), liveTick())
}

func (m *model) onLiveList(ss []live.Session) {
	m.lv.running, m.lv.list, m.lv.at = false, ss, time.Now()
	m.lv.sel = clamp(m.lv.sel, 0, len(ss)-1)
}

func (m model) liveKey(k tea.KeyMsg) (model, tea.Cmd, bool) {
	switch k.String() {
	case "up", "k":
		m.lv.sel = clamp(m.lv.sel-1, 0, len(m.lv.list)-1)
	case "down", "j":
		m.lv.sel = clamp(m.lv.sel+1, 0, len(m.lv.list)-1)
	case "enter":
		if m.lv.sel < len(m.lv.list) && m.lv.list[m.lv.sel].Transcript != "" {
			s := m.lv.list[m.lv.sel]
			ses, err := m.cache.Get(s.Transcript)
			if err != nil {
				m.status = err.Error()
				return m, nil, true
			}
			m.tab, m.level, m.live = tSessions, 2, ses
			return m, m.refreshLive(), true
		}
	case "o":
		if m.lv.sel < len(m.lv.list) && m.lv.list[m.lv.sel].Cwd != "" {
			if err := openFolder(m.lv.list[m.lv.sel].Cwd); err != nil {
				m.status = err.Error()
			}
		}
	default:
		return m, nil, false
	}
	return m, nil, true
}

func entrypointName(e string) string {
	switch e {
	case "cli":
		return T("terminal")
	case "claude-vscode":
		return "VS Code"
	case "claude-desktop", "desktop":
		return "Claude Desktop"
	case "sdk-ts", "sdk-py", "sdk-cli":
		return "SDK"
	case "":
		return ""
	}
	return e
}

func ago2(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return F("%ds", int(d.Seconds()))
	case d < time.Hour:
		return F("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return F("%dh", int(d.Hours()))
	}
	return F("%dd", int(d.Hours()/24))
}

func modelShort(id string) string {
	id = strings.TrimPrefix(id, "claude-")
	parts := strings.Split(id, "-")
	if len(parts) >= 2 {
		name := strings.ToUpper(parts[0][:1]) + parts[0][1:]
		ver := parts[1]
		if len(parts) >= 3 && len(parts[2]) <= 2 {
			ver += "." + parts[2]
		}
		return name + " " + ver
	}
	return id
}

// stepLine draws one tool call of the timeline.
func (m model) stepLine(st live.Step, prefix string, w int, sub bool) string {
	mark := sOK.Render(gOK)
	when := ""
	switch {
	case !st.Done:
		mark = sTitle.Render(m.spinner())
		when = sTitle.Render(F("running %s", ago2(st.At)))
	case st.Err:
		mark = sWarn.Render(gFail)
	}
	if when == "" && !st.At.IsZero() {
		layout := "15:04:05"
		if st.At.Local().Format("2006-01-02") != time.Now().Format("2006-01-02") {
			layout = "02 Jan 15:04"
		}
		when = sMuted.Render(st.At.Local().Format(layout))
	}
	tool := kindStyle("tool").Render(padRight(st.Tool, max(10, min(28, ansiWidth(st.Tool)))))
	if sub {
		tool = sMuted.Render(T("subagent")+" › ") + tool
	}
	left := sMuted.Render(prefix) + mark + " " + tool
	if st.Detail != "" {
		left += "  " + clean(st.Detail)
	}
	room := w - lipgloss.Width(when) - 2
	left = truncate(left, room)
	return left + strings.Repeat(" ", max(1, w-lipgloss.Width(left)-lipgloss.Width(when))) + when
}

// liveCard draws one session as a box: who, where, what it is doing now, its last steps.
func (m model) liveCard(s live.Session, w int, selected, expanded bool) string {
	inner := w - 4
	state := sMuted.Render(gOff + " " + T("waiting for you"))
	if s.Working() {
		state = sTitle.Render(m.spinner() + " " + T("working"))
	}
	if s.Status != "" && s.Status != "busy" && s.Status != "idle" {
		state = sWarn.Render(gWarn + " " + s.Status)
	}
	if !s.Activity.IsZero() {
		state += sMuted.Render(" · " + F("last activity %s ago", ago2(maxTime(s.Activity, s.Updated))))
	}
	title := s.Name
	if s.Title != "" {
		title += " · " + s.Title
	}
	head := sHeading.Render(truncate(clean(title), inner-lipgloss.Width(state)-2))
	head += strings.Repeat(" ", max(1, inner-lipgloss.Width(head)-lipgloss.Width(state))) + state

	var meta []string
	if s.Cwd != "" {
		meta = append(meta, live.BaseName(s.Cwd))
	}
	if e := entrypointName(s.Entrypoint); e != "" {
		meta = append(meta, e)
	}
	if s.Model != "" {
		mm := modelShort(s.Model)
		if s.Effort != "" {
			mm += " " + s.Effort
		}
		meta = append(meta, mm)
	}
	if s.Version != "" {
		meta = append(meta, "v"+s.Version)
	}
	line2 := sMuted.Render(strings.Join(meta, " · "))
	if s.Context > 0 {
		win := sessions.WindowFor(s.Model)
		ratio := float64(s.Context) / float64(max(1, win))
		ctx := T("context") + " " + bar(min(1, ratio), 12) + F(" %d%% · %s", int(100*ratio), human(s.Context))
		line2 = truncate(line2, inner-lipgloss.Width(ctx)-2)
		line2 += strings.Repeat(" ", max(1, inner-lipgloss.Width(line2)-lipgloss.Width(ctx))) + ctx
	}
	lines := []string{head, line2}
	if s.Todo.Total > 0 {
		t := F("to-do %d/%d", s.Todo.Done, s.Todo.Total) + " " + meter(float64(s.Todo.Done)/float64(s.Todo.Total), 10)
		if s.Todo.Current != "" {
			t += "  " + sTitle.Render(gCursor) + " " + clean(s.Todo.Current)
		}
		lines = append(lines, truncate(t, inner))
	}
	steps := s.Steps
	n := 2
	if expanded {
		n = 6
	}
	if len(steps) > n {
		steps = steps[len(steps)-n:]
	}
	for i, st := range steps {
		br := "├─ "
		if i == len(steps)-1 {
			br = "└─ "
		}
		lines = append(lines, m.stepLine(st, br, inner, false))
		for j, sub := range st.Sub {
			pre := "│  "
			if i == len(steps)-1 {
				pre = "   "
			}
			b2 := "├─ "
			if j == len(st.Sub)-1 {
				b2 = "└─ "
			}
			lines = append(lines, m.stepLine(sub, pre+b2, inner, true))
		}
	}
	if len(s.Steps) == 0 && s.Transcript == "" {
		lines = append(lines, sMuted.Render(T("no transcript yet")))
	}
	if s.Unrecognized {
		lines = append(lines, sWarn.Render(truncate(gWarn+" "+T("this Claude Code version writes its transcript in a format catude does not read yet: update catude"), inner)))
	}
	if s.LastText != "" && (expanded || len(s.Steps) == 0) {
		lines = append(lines, sMuted.Render(truncate("“"+clean(s.LastText)+"”", inner)))
	}
	box := sPane.Width(w - 2)
	if selected {
		box = box.BorderForeground(accent)
	}
	return box.Render(strings.Join(lines, "\n"))
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func (m model) liveView(h int) string {
	var b strings.Builder
	working := 0
	for _, s := range m.lv.list {
		if s.Working() {
			working++
		}
	}
	state := T("refreshes every second")
	if m.lv.at.IsZero() {
		state = m.spinner() + " " + T("looking for running sessions…")
	}
	b.WriteString(sHeading.Render(T("Live sessions")) + sMuted.Render("  "+F("%d open · %d working", len(m.lv.list), working)+" · "+state) + "\n\n")
	if len(m.lv.list) == 0 { // still looking (the header says so), or nothing running
		if !m.lv.at.IsZero() {
			b.WriteString(sMuted.Render(T("No Claude Code session is running. Start one (claude) and it appears here at once.")))
		}
		return b.String()
	}
	w := m.w - 6
	cols := 1
	if w >= 150 {
		cols = 2
	}
	cw := w / cols
	var cards []string
	for i, s := range m.lv.list {
		cards = append(cards, m.liveCard(s, cw, i == m.lv.sel, s.Working() || i == m.lv.sel))
	}
	// rows of cards; the window starts at the row of the selected card when it does not fit
	var rows []string
	for i := 0; i < len(cards); i += cols {
		end := min(len(cards), i+cols)
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cards[i:end]...))
	}
	room := h - 3
	selRow := clamp(m.lv.sel/cols, 0, len(rows)-1)
	start := 0
	used := 0
	for i := selRow; i >= 0; i-- { // as many rows above the selected one as fit
		used += lipgloss.Height(rows[i])
		if used > room && i < selRow {
			start = i + 1
			break
		}
	}
	var out []string
	used = 0
	for i := start; i < len(rows); i++ {
		hh := lipgloss.Height(rows[i])
		if used+hh > room && i > start {
			out = append(out, sMuted.Render(F("↓ %d more", len(rows)-i)))
			break
		}
		out = append(out, rows[i])
		used += hh
	}
	if start > 0 {
		out = append([]string{sMuted.Render(F("↑ %d more", start))}, out...)
	}
	b.WriteString(strings.Join(out, "\n"))
	return b.String()
}
