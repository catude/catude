package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/i18n"
	"github.com/catude/catude/internal/prefs"
)

// Options are the start-up choices that come from the command line.
type Options struct {
	Lite   bool        // start in the lite view (ctlite, --lite)
	Glyphs string      // --glyphs (wins over preferences)
	Theme  string      // --theme (wins over preferences)
	Prefs  prefs.Prefs // saved preferences
}

var (
	themeChoices  = []string{"auto", "dark", "light", "high-contrast", "mono"}
	glyphChoices  = []string{"auto", "unicode", "safe", "ascii"}
	accentChoices = []string{"clay", "amber", "blue", "green", "magenta", "cyan", "red"}
)

// tabID is the stable name of a tab in the preferences file.
func tabID(t tab) string { return strings.ToLower(tabNames[t]) }

// activeTabs are the tabs of the current view (full or lite), in their usual order.
func (m model) activeTabs() []tab {
	ids := m.prefs.Tabs
	if len(ids) == 0 {
		ids = prefs.DefaultTabs
	}
	if m.lite {
		ids = m.prefs.LiteTabs
		if len(ids) == 0 {
			ids = prefs.DefaultLite
		}
	}
	all := make([]tab, len(tabNames))
	for i := range all {
		all[i] = tab(i)
	}
	if len(ids) == 0 {
		return all
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var out []tab
	for _, t := range all {
		if want[tabID(t)] {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return all
	}
	return out
}

func (m model) tabPos(t tab) int {
	for i, x := range m.activeTabs() {
		if x == t {
			return i
		}
	}
	return -1
}

// stepTab is the tab d positions away in the current view.
func (m model) stepTab(d int) tab {
	ts := m.activeTabs()
	i := m.tabPos(m.tab)
	if i < 0 {
		return ts[0]
	}
	return ts[(i+d+len(ts))%len(ts)]
}

// applyPrefs applies look-and-feel preferences now and keeps the current tab valid.
func (m *model) applyPrefs() {
	g, th := m.opts.Glyphs, m.opts.Theme
	if g == "" {
		g = m.prefs.Glyphs
	}
	if th == "" {
		th = m.prefs.Theme
	}
	Accent = m.prefs.Accent
	reduceMotion = m.prefs.ReduceMotion || envReduceMotion
	Setup(g, th)
	if m.tabPos(m.tab) < 0 {
		m.tab = m.activeTabs()[0]
	}
	m.gen++
}

func (m *model) savePrefs() {
	m.prefs.Lite = m.lite
	if err := prefs.Save(m.prefs); err != nil {
		m.status = T("preferences not saved:") + " " + err.Error()
	}
	m.applyPrefs()
}

// prefsState is the preferences screen ("," or F2).
type prefsState struct {
	open bool
	sel  int
}

const prefsFixedRows = 9 // language, view, theme, glyphs, accent, motion, hints, bell, notification

func cycle(list []string, cur string, d int) string {
	if cur == "" {
		cur = list[0]
	}
	for i, v := range list {
		if v == cur {
			return list[(i+d+len(list))%len(list)]
		}
	}
	return list[0]
}

func (m model) prefsKey(k tea.KeyMsg) (model, tea.Cmd) {
	p := &m.pv
	rows := prefsFixedRows + len(tabNames)
	s := k.String()
	d := 1
	switch s {
	case "esc", ",", "f2", "q":
		p.open = false
		return m, nil
	case "up", "k":
		p.sel = clamp(p.sel-1, 0, rows-1)
		return m, nil
	case "down", "j":
		p.sel = clamp(p.sel+1, 0, rows-1)
		return m, nil
	case "left", "h":
		d = -1
	case "right", "l", " ", "enter":
	default:
		return m, nil
	}
	switch p.sel {
	case 0:
		m.prefs.Language = cycle(languageChoices(), m.prefs.Language, d)
		i18n.Set(m.prefs.Language)
	case 1:
		m.lite = !m.lite
	case 2:
		m.prefs.Theme = cycle(themeChoices, m.prefs.Theme, d)
	case 3:
		m.prefs.Glyphs = cycle(glyphChoices, m.prefs.Glyphs, d)
	case 4:
		m.prefs.Accent = cycle(accentChoices, m.prefs.Accent, d)
	case 5:
		m.prefs.ReduceMotion = !m.prefs.ReduceMotion
	case 6:
		m.prefs.HideHints = !m.prefs.HideHints
	case 7:
		m.prefs.Bell = !m.prefs.Bell
	case 8:
		m.prefs.Notify = !m.prefs.Notify
	default:
		id := tabID(tab(p.sel - prefsFixedRows))
		cur := m.activeTabs()
		set := map[string]bool{}
		for _, t := range cur {
			set[tabID(t)] = true
		}
		if set[id] && len(cur) == 1 {
			m.status = T("at least one tab must stay visible")
			return m, nil
		}
		set[id] = !set[id]
		var ids []string
		for i := range tabNames {
			if set[tabID(tab(i))] {
				ids = append(ids, tabID(tab(i)))
			}
		}
		if !m.lite && sameSet(ids, prefs.DefaultTabs) {
			ids = nil // the default stays future-proof: new core tabs appear by themselves
		}
		if m.lite {
			m.prefs.LiteTabs = ids
		} else {
			m.prefs.Tabs = ids
		}
	}
	m.savePrefs()
	return m, nil
}

func (m model) prefsView(h int) string {
	var b strings.Builder
	b.WriteString(sTitle.Render(T("Preferences")) + sMuted.Render("  "+T("↑↓ choose · ←→ or space change · saved as you go · esc close")+" · "+shorten(prefs.Path(), 46)) + "\n\n")
	val := func(list []string, cur string) string {
		if cur == "" {
			cur = list[0]
		}
		var parts []string
		for _, v := range list {
			if v == cur {
				parts = append(parts, sTitle.Render("["+choiceName(v)+"]"))
			} else {
				parts = append(parts, sMuted.Render(choiceName(v)))
			}
		}
		return strings.Join(parts, "  ")
	}
	onOff := func(on bool, yes, no string) string {
		if on {
			return sTitle.Render("["+T(yes)+"]") + "  " + sMuted.Render(T(no))
		}
		return sMuted.Render(T(yes)) + "  " + sTitle.Render("["+T(no)+"]")
	}
	fixed := []struct{ label, value, help string }{
		{T("Language"), val(languageChoices(), m.prefs.Language), languageHelp()},
		{T("View"), onOff(!m.lite, "full", "lite"), T("lite shows only the tabs you tick for it (ctlite opens catude in lite)")},
		{T("Theme"), val(themeChoices, m.prefs.Theme), T("auto follows your terminal background; mono uses no colour")},
		{T("Glyphs"), val(glyphChoices, m.prefs.Glyphs), F("auto picks what your terminal and font can draw (now: %s)", Glyphs)},
		{T("Accent colour"), val(accentChoices, m.prefs.Accent), T("the highlight colour of tabs, titles and the selection")},
		{T("Animations"), onOff(!m.prefs.ReduceMotion, "on", "off"), T("spinners while something loads")},
		{T("Key hints"), onOff(!m.prefs.HideHints, "shown", "hidden"), T("the line of shortcuts at the bottom (? always shows them all)")},
		{T("Notification"), onOff(m.prefs.Notify, "on", "off"), T("a system notification when a Claude Code session finishes and waits for you")},
		{T("Bell"), onOff(m.prefs.Bell, "on", "off"), T("ring when a Claude Code session finishes and waits for you (the bottom line always says so)")},
	}
	var lines []string
	for i, f := range fixed {
		l := padRight(f.label, 18) + " " + f.value
		if i == m.pv.sel {
			l = sHeading.Render(gCursor+" ") + l + sMuted.Render("   "+f.help)
		} else {
			l = "  " + l
		}
		lines = append(lines, l)
	}
	tabsTitle := T("Tabs in the full view")
	if m.lite {
		tabsTitle = T("Tabs in the lite view")
	}
	lines = append(lines, "", sHeading.Render(tabsTitle)+sMuted.Render("  "+T("(switch View above to choose the tabs of the other one)")))
	active := map[tab]bool{}
	for _, t := range m.activeTabs() {
		active[t] = true
	}
	var cells []string
	for i, n := range tabNames {
		box := "[ ]"
		if active[tab(i)] {
			box = sOK.Bold(true).Render("[x]")
		}
		c := box + " " + padRight(T(n), 14)
		if m.pv.sel == prefsFixedRows+i {
			c = sSel.Render(gCursor + " " + stripStyles(c, 24))
		} else {
			c = "  " + c
		}
		cells = append(cells, c)
	}
	// two columns when there is room, so every tab fits on small terminals
	if m.w >= 70 {
		half := (len(cells) + 1) / 2
		for i := 0; i < half; i++ {
			row := lipgloss.NewStyle().Width(34).Render(cells[i])
			if i+half < len(cells) {
				row += cells[i+half]
			}
			lines = append(lines, row)
		}
	} else {
		lines = append(lines, cells...)
	}
	if len(lines) > h-2 {
		// keep the selected row visible on short terminals
		start := clamp(m.pv.sel-(h-4), 0, len(lines)-(h-2))
		lines = lines[start : start+h-2]
	}
	return b.String() + strings.Join(lines, "\n")
}

// wiz2 reports whether a full-screen overlay is open (so tab-specific hints stay out of the way).
func (m model) wiz2() bool { return m.wiz != nil || m.help || m.gal.open || m.pv.open || m.debug }

// languageChoices are "auto" and every supported language code.
func languageChoices() []string {
	out := []string{"auto"}
	for _, l := range i18n.Languages {
		out = append(out, l.Code)
	}
	return out
}

// choiceName shows a preference value: languages by their own name, the rest translated.
func choiceName(v string) string {
	for _, l := range i18n.Languages {
		if l.Code == v {
			return l.Name
		}
	}
	return T(v)
}

func languageHelp() string {
	h := F("auto follows your system language (now: %s)", i18n.Current().Name)
	if i18n.Current().CJK && (Glyphs == "safe" || Glyphs == "ascii") {
		h += " · " + T("this console may not have a font for these characters: use Windows Terminal")
	}
	return h
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	in := map[string]bool{}
	for _, x := range a {
		in[x] = true
	}
	for _, x := range b {
		if !in[x] {
			return false
		}
	}
	return true
}

// hiddenTabs is how many tabs the current view leaves out.
func (m model) hiddenTabs() int { return len(tabNames) - len(m.activeTabs()) }
