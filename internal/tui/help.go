package tui

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/i18n"
)

var helpSections = []struct {
	title string
	keys  [][2]string
}{
	{"Everywhere", [][2]string{
		{"?", "this help (any key closes it)"},
		{",  F2", "preferences: language, visible tabs, lite view, theme, accent colour, glyphs"},
		{"1…9 0", "jump to a tab (the number shown next to its name)"},
		{"Caps Lock", "shortcuts work in upper case too (except G, and P/R/D where they have their own action)"},
		{"← → tab", "previous / next tab"},
		{"↑ ↓ j k", "move · pgup pgdown g G: page, top, bottom"},
		{"/", "filter the list (enter keeps it, esc clears it)"},
		{"r", "reload / measure again"},
		{"q ctrl+c", "quit"},
		{"mouse", "wheel scrolls · click a tab to open it"},
		{"F12 ctrl+o", "debug panel: versions, terminal, render time, background jobs"},
	}},
	{"Live", [][2]string{
		{"↑ ↓", "choose a running session"},
		{"enter", "open its conversation with live context"},
		{"o", "open its folder"},
	}},
	{"Config", [][2]string{
		{"↑ ↓ enter", "choose a Claude app file · open its settings"},
		{"enter", "change the selected setting"},
		{"space", "switch yes/no · next allowed value"},
		{"d", "remove the setting from the file (back to the default)"},
		{"a", "only what the file sets / every documented setting"},
		{"e", "open the whole file in your editor (backup first)"},
		{"esc", "back to the list of files"},
	}},
	{"Skills · Agents · Commands · MCP · Plugins", [][2]string{
		{"e", "edit the file (a backup is taken first, then it is re-validated)"},
		{"space", "mark the item for a new work profile"},
		{"P", "save the marked items as a work profile"},
		{"m", "MCP: check health with claude mcp list"},
		{"x", "MCP and Skills: switch it off for this project only (skills: name only, then only by /name, then off)"},
	}},
	{"Context", [][2]string{
		{"r", "measure again (claude /context, 20–40 s, in the background)"},
		{"p", "measure with a work profile instead, to compare"},
		{"↑ ↓", "scroll"},
	}},
	{"Models", [][2]string{
		{"↑ ↓", "compare the selected model with the one you use"},
		{"t", "kind of work: all, coding, reasoning, math & science, agents, knowledge"},
		{"space", "compare everything with the selected model instead"},
		{"r", "download the public results again"},
	}},
	{"Sessions", [][2]string{
		{"enter", "open project / open session (live context)"},
		{"n", "name the conversation, as /rename does"},
		{"f", "search the text of every conversation"},
		{"R  c", "resume the session in Claude Code, in its folder"},
		{"x", "export the conversation to Markdown"},
		{"D D", "move the session to catude's trash (restorable)"},
		{"esc", "back"},
	}},
	{"Graph", [][2]string{
		{"enter + -", "open or close a node · - also jumps to its parent"},
		{"E C", "expand or collapse everything"},
		{"/", "find any node by name or kind"},
		{"w", "open the graph as a diagram in your browser"},
		{"r", "rebuild (re-reads files, tools and running processes)"},
	}},
	{"Usage", [][2]string{
		{"[ ]", "7 / 14 / 30 days"},
		{"r", "read the transcripts again"},
	}},
	{"Tools", [][2]string{
		{"r", "check installed tools and versions again"},
	}},
	{"Profiles", [][2]string{
		{"n", "new work profile (choose skills, agents, MCP, plugins; name; mode)"},
		{"a", "save your current setup as a profile (everything preselected)"},
		{"p", "presets: work profiles based on the public setups of well-known practitioners"},
		{"v", "verify: ask Claude Code whether every item of the profile loads"},
		{"e", "edit the selected work profile"},
		{"s", "switch: make it the active profile (catude claude / shell integration)"},
		{"enter", "start Claude Code with it · x: only this profile"},
		{"d d", "delete the selected work profile"},
		{"i i", "install / upgrade claude-swap (accounts)"},
	}},
	{"Storage", [][2]string{
		{"o", "open the folder"},
		{"c c", "clean a safe item (only files older than 1 hour)"},
	}},
	{"Discover", [][2]string{{"i i", "install with the official claude command"}}},
}

func (m model) helpView(h int) string {
	var b strings.Builder
	b.WriteString(sTitle.Render(T("Keys")) + sMuted.Render("  "+F("glyphs: %s · theme: %s · language: %s", Glyphs, Theme, i18n.Current().Name)) + "\n")
	for _, s := range helpSections {
		b.WriteString("\n" + sHeading.Render(T(s.title)) + "\n")
		for _, k := range s.keys {
			b.WriteString("  " + padRight(k[0], 11) + " " + T(k[1]) + "\n")
		}
	}
	lines := strings.Split(b.String(), "\n")
	if len(lines) > h {
		// two columns when the terminal is short
		// (each line is cut to its column: wrapping would push the two columns apart)
		half := (len(lines) + 1) / 2
		lw, rw := m.w/2-2, m.w-m.w/2-4
		cut := func(ls []string, w int) string {
			out := make([]string, len(ls))
			for i, l := range ls {
				out[i] = truncate(l, w)
			}
			return strings.Join(out, "\n")
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(lw+2).Render(cut(lines[:half], lw)), cut(lines[half:], rw))
	}
	return b.String()
}

// tabAt maps a click on the tab bar to a tab (uses the same labels the bar renders).
func (m model) tabAt(x int) (tab, bool) {
	pos := 0
	for _, i := range m.visibleTabs() {
		w := lipgloss.Width(sTab.Render(m.tabLabel(tab(i))))
		if x >= pos && x < pos+w {
			return tab(i), true
		}
		pos += w
	}
	return 0, false
}

func (m model) mouse(e tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch {
	case e.Button == tea.MouseButtonWheelUp:
		return m.key(tea.KeyMsg{Type: tea.KeyUp})
	case e.Button == tea.MouseButtonWheelDown:
		return m.key(tea.KeyMsg{Type: tea.KeyDown})
	case e.Action == tea.MouseActionRelease && e.Button == tea.MouseButtonLeft && e.Y == 1:
		if t, ok := m.tabAt(e.X); ok {
			return m.gotoTab(t)
		}
	}
	return m, nil
}

var envReduceMotion = os.Getenv("CATUDE_REDUCE_MOTION") != ""

var reduceMotion = envReduceMotion

var spinFrames = []string{"◐", "◓", "◑", "◒"}

func (m model) spinner() string {
	if reduceMotion {
		return gBusy
	}
	return spinFrames[m.spin%len(spinFrames)]
}
