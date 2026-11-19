package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/claudecfg"
)

// cfgState is the Config tab: every configuration file of every Claude app found on this machine, with
// every documented setting of that app — set or not — grouped by category.
type cfgState struct {
	targets []claudecfg.Target
	sel     int
	focus   int // 0 files, 1 values
	entries []claudecfg.Entry
	nFile   int // entries that are in the file
	err     string
	esel    int
	editing bool
	input   string
	loaded  string // target id the entries belong to
	onlySet bool   // show only what the file sets
	ps      cfgPresets
}

func (m *model) ensureConfig() {
	if len(m.cfg.targets) == 0 {
		m.cfg.targets = claudecfg.WithEffective(claudecfg.Targets(m.env))
	}
	m.loadCfgEntries(false)
}

// catOrder puts the categories people change most first and environment variables last.
func catOrder(c string) string {
	switch c {
	case "Model":
		return "0"
	case "Permissions":
		return "1"
	case "UI":
		return "2"
	case "Other":
		return "y"
	case "Environment variables":
		return "z"
	}
	return "5" + c
}

// loadCfgEntries reads the selected file (small JSON: fast enough to do inline) and adds every
// documented setting that is not in it.
func (m *model) loadCfgEntries(force bool) {
	c := &m.cfg
	if c.sel >= len(c.targets) {
		return
	}
	t := c.targets[c.sel]
	if !force && c.loaded == t.ID {
		return
	}
	if t.ID == claudecfg.EffectiveID {
		es := claudecfg.Effective(c.targets)
		sort.SliceStable(es, func(i, j int) bool { return catOrder(es[i].Category) < catOrder(es[j].Category) })
		c.entries, c.nFile, c.err, c.loaded = es, len(es), "", t.ID
		c.esel = clamp(c.esel, 0, len(c.entries)-1)
		return
	}
	if t.ID != "desktop-policy" {
		_, statErr := os.Stat(t.Path)
		c.targets[c.sel].Exists = statErr == nil
	}
	es, err := claudecfg.Read(t)
	c.err = ""
	if err != nil {
		c.err = err.Error()
	}
	c.nFile = 0
	for _, e := range es {
		if e.Set {
			c.nFile++
		}
	}
	all := append(es, claudecfg.Documented(t, es)...)
	for i := range all {
		if all[i].Category == "" {
			all[i].Category = "Other"
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if ca, cb := catOrder(a.Category), catOrder(b.Category); ca != cb {
			return ca < cb
		}
		if a.Set != b.Set {
			return a.Set // what you set comes first in each category
		}
		if a.Deprecated != b.Deprecated {
			return b.Deprecated
		}
		return strings.ToLower(a.Key) < strings.ToLower(b.Key)
	})
	c.entries = all
	c.loaded = t.ID
	c.esel = clamp(c.esel, 0, len(c.entries)-1)
}

func (m model) cfgVisible() []int {
	f := strings.ToLower(m.filter)
	var out []int
	for i, e := range m.cfg.entries {
		if m.cfg.onlySet && !e.Set {
			continue
		}
		if f == "" || strings.Contains(strings.ToLower(e.Key+" "+e.Value+" "+e.Help+" "+e.Category), f) {
			out = append(out, i)
		}
	}
	return out
}

func (m model) cfgSelected() (claudecfg.Target, *claudecfg.Entry) {
	c := m.cfg
	if c.sel >= len(c.targets) {
		return claudecfg.Target{}, nil
	}
	vis := m.cfgVisible()
	if c.focus != 1 || c.esel >= len(vis) {
		return c.targets[c.sel], nil
	}
	return c.targets[c.sel], &m.cfg.entries[vis[c.esel]]
}

// cfgDone reports a write and reloads everything that depends on the file.
func (m model) cfgDone(t claudecfg.Target, what, backup string, err error) (model, tea.Cmd) {
	if err != nil {
		m.status = gFail + " " + err.Error()
		return m, nil
	}
	if what == "" { // nothing to do (a variable already off)
		return m, nil
	}
	m.status = fmt.Sprintf("%s %s · %s", gOK, what, shorten(t.Path, 40))
	if backup != "" {
		m.status += " · backup " + shorten(backup, 36)
	}
	if t.Restart != "" {
		m.status += " · applies to " + t.Restart
	}
	key := ""
	if _, e := m.cfgSelected(); e != nil {
		key = e.Key
	}
	m.loadCfgEntries(true)
	if key != "" { // keep the cursor on the same setting after it moved between "set" and "not set"
		for j, i := range m.cfgVisible() {
			if m.cfg.entries[i].Key == key {
				m.cfg.esel = j
			}
		}
	}
	return m, m.load()
}

func (m model) cfgApply(t claudecfg.Target, e claudecfg.Entry, value string) (model, tea.Cmd) {
	backup, err := claudecfg.Set(t, e, value)
	what := e.Key + " = " + value
	if e.Env && e.Type == "yes/no" && value == "false" {
		what = e.Key + " removed (off)"
		if !e.Set {
			what = ""
		}
	}
	return m.cfgDone(t, what, backup, err)
}

func (m model) cfgUnset(t claudecfg.Target, e claudecfg.Entry) (model, tea.Cmd) {
	backup, err := claudecfg.Unset(t, e)
	what := e.Key + " removed: back to the default"
	if e.Default != "" {
		what += " (" + e.Default + ")"
	}
	return m.cfgDone(t, what, backup, err)
}

// cfgKey handles keys of the Config tab. handled=false lets the global keys run.
func (m model) cfgKey(k tea.KeyMsg) (model, tea.Cmd, bool) {
	c := &m.cfg
	s := k.String()
	if c.ps.open && !c.editing {
		return m.presetsKey(k)
	}
	if s == "p" && !c.editing {
		c.ps.open, c.ps.armed = true, ""
		m.status = T("presets are written to the file selected on the left")
		return m, nil, true
	}
	if c.editing {
		t, e := m.cfgSelected()
		switch s {
		case "esc":
			c.editing, c.input = false, ""
		case "enter":
			c.editing = false
			if e == nil {
				return m, nil, true
			}
			nm, cmd := m.cfgApply(t, *e, c.input)
			return nm, cmd, true
		case "backspace":
			if r := []rune(c.input); len(r) > 0 {
				c.input = string(r[:len(r)-1])
			}
		default:
			if k.Type == tea.KeyRunes || s == " " {
				c.input += string(k.Runes)
			}
		}
		return m, nil, true
	}
	n := len(m.cfgVisible())
	switch s {
	case "up", "k":
		if c.focus == 0 {
			c.sel = clamp(c.sel-1, 0, len(c.targets)-1)
			c.esel = 0
			m.loadCfgEntries(false)
		} else {
			c.esel = clamp(c.esel-1, 0, n-1)
		}
	case "down", "j":
		if c.focus == 0 {
			c.sel = clamp(c.sel+1, 0, len(c.targets)-1)
			c.esel = 0
			m.loadCfgEntries(false)
		} else {
			c.esel = clamp(c.esel+1, 0, n-1)
		}
	case "pgdown":
		if c.focus == 1 {
			c.esel = clamp(c.esel+10, 0, n-1)
		}
	case "pgup":
		if c.focus == 1 {
			c.esel = clamp(c.esel-10, 0, n-1)
		}
	case "home", "g":
		c.esel = 0
	case "end", "G":
		c.esel = max(0, n-1)
	case "esc":
		if c.focus == 1 {
			c.focus, m.filter = 0, ""
		} else {
			return m, nil, false
		}
	case "a":
		c.onlySet, c.esel = !c.onlySet, 0
		m.status = map[bool]string{true: "showing only what this file sets (a: show every documented setting)", false: "showing every documented setting (a: only what this file sets)"}[c.onlySet]
	case "r":
		c.targets = claudecfg.WithEffective(claudecfg.Targets(m.env))
		c.sel = clamp(c.sel, 0, len(c.targets)-1)
		m.loadCfgEntries(true)
		m.status = "configuration files read again"
	case "e":
		t, _ := m.cfgSelected()
		if t.ID == "desktop-policy" {
			m.status = "policies are set by an administrator (" + t.Path + "), not edited here"
			return m, nil, true
		}
		if t.ID == claudecfg.EffectiveID {
			m.status = T("this view only shows the result: open the file the value comes from to change it")
			return m, nil, true
		}
		nm, cmd := m.editPath(t.Path, t.ID == "code-managed", t.State)
		return nm, cmd, true
	case "d", "delete", "backspace":
		t, e := m.cfgSelected()
		if c.focus != 1 || e == nil {
			return m, nil, s == "d"
		}
		if t.ID == claudecfg.EffectiveID {
			m.status = T("this view only shows the result: open the file the value comes from to change it")
			return m, nil, true
		}
		if !e.Set {
			m.status = e.Key + " is not set in this file: it already uses the default"
			return m, nil, true
		}
		nm, cmd := m.cfgUnset(t, *e)
		return nm, cmd, true
	case "enter", " ":
		t, e := m.cfgSelected()
		if c.focus == 0 {
			if s == "enter" {
				c.focus, c.esel = 1, 0
			}
			return m, nil, true
		}
		if e == nil {
			return m, nil, true
		}
		if !e.Editable {
			m.status = e.Key + " cannot be changed here"
			switch {
			case t.ReadOnly:
				m.status += ": " + t.Note
			case e.Secret || (e.Env && e.Value == claudecfg.NotSet):
				m.status += ": it holds a credential. Keep it in your password manager or the system environment, not in a settings file"
			case e.Type == "list" || e.Type == "object":
				m.status += " inline (lists and objects): press e to open the file in your editor"
				if e.Set {
					m.status += " · d removes it"
				}
			default:
				m.status += ": press e to open the file in your editor"
			}
			return m, nil, true
		}
		switch {
		case e.Type == "yes/no":
			next := "true"
			if e.Value == "true" || (e.Env && e.Set) {
				next = "false"
			}
			nm, cmd := m.cfgApply(t, *e, next)
			return nm, cmd, true
		case len(e.Choices) > 0 && s == " ":
			next := ""
			for i, ch := range e.Choices {
				if ch == e.Value {
					next = e.Choices[(i+1)%len(e.Choices)]
				}
			}
			if next == "" {
				next = e.Choices[0]
			}
			if strings.Contains(next, "<") || (e.Key == "permissions.defaultMode" && next == "bypassPermissions") { // placeholders and bypass are typed, not cycled into
				for i, ch := range e.Choices {
					if ch == next {
						next = e.Choices[(i+1)%len(e.Choices)]
					}
				}
			}
			nm, cmd := m.cfgApply(t, *e, next)
			return nm, cmd, true
		default:
			c.editing, c.input = true, ""
			if e.Set && !e.Secret {
				c.input = e.Value
			}
		}
	default:
		return m, nil, false
	}
	return m, nil, true
}

// editPath opens any configuration file in the user's editor after backing it up.
func (m model) editPath(path string, managed, state bool) (model, tea.Cmd) {
	if path == "" {
		return m, nil
	}
	if managed {
		m.status = "the organisation policy file cannot be edited from catude"
		return m, nil
	}
	if _, err := os.Stat(path); err != nil {
		m.status = "the file does not exist yet: set a value first (enter on a setting) and catude creates it"
		return m, nil
	}
	backup, err := backupFile(path)
	if err != nil {
		m.status = "backup failed, not editing: " + err.Error()
		return m, nil
	}
	cmd, err := editorCmd(path)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.status = "editing " + path + " (backup: " + backup + ")"
	if state {
		m.status = "close Claude Code first: it rewrites this file while running · " + m.status
	}
	m.cfg.loaded = ""
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return editDoneMsg{path, err} })
}

func (m model) configView(h int) string {
	c := m.cfg
	if c.ps.open {
		return m.presetsView(h)
	}
	leftW := clamp(m.w*30/100, 30, 52)
	rightW := max(30, m.w-leftW-10)
	var l strings.Builder
	l.WriteString(sHeading.Render(T("Claude apps on this machine")) + "\n")
	app := ""
	for i, t := range c.targets {
		if t.App != app {
			app = t.App
			l.WriteString("\n" + sTitle.Render(app) + "\n")
		}
		mark := markOff()
		switch {
		case t.Exists && t.ReadOnly:
			mark = sMuted.Render(gMarked) + " "
		case t.Exists:
			mark = markOK()
		}
		line := mark + clipEnd(t.Label, leftW-6)
		if i == c.sel {
			st := sSel
			if c.focus == 1 {
				st = sHeading
			}
			line = st.Render(gCursor + " " + stripStyles(line, leftW-4))
		} else {
			line = "  " + line
		}
		l.WriteString(line + "\n")
	}
	l.WriteString("\n" + sMuted.Render(lipgloss.NewStyle().Width(leftW-2).Render(
		gOK+" exists  "+gOff+" not created yet  "+gMarked+" read-only\n\n"+
			"How they relate: Claude Code in the terminal reads only its own files. The Code tab of Claude Desktop uses Claude Code's files plus Desktop's MCP servers. Connectors, skills and plugins from claude.ai reach both through your account.\n\n"+
			"Every documented setting is listed (Anthropic docs, "+claudecfg.RefUpdated()+").")))

	var r strings.Builder
	if c.sel < len(c.targets) {
		t := c.targets[c.sel]
		r.WriteString(sHeading.Render(t.App+" · "+t.Label) + "\n")
		if t.Path != "" {
			r.WriteString(sMuted.Render(shorten(t.Path, rightW-2)) + "\n")
		}
		r.WriteString(lipgloss.NewStyle().Width(rightW).Render(sMuted.Render(t.Note)) + "\n")
		if c.err != "" {
			r.WriteString(sHigh.Render(gFail+" "+c.err+" · press e to fix it in your editor") + "\n")
		}
		vis := m.cfgVisible()
		view := fmt.Sprintf("%d set · %d documented settings", c.nFile, len(c.entries))
		if c.onlySet {
			view = fmt.Sprintf("%d set (a: show all %d)", c.nFile, len(c.entries))
		}
		if m.filter != "" {
			view += fmt.Sprintf(" · %d match %q", len(vis), m.filter)
		}
		effective := t.ID == claudecfg.EffectiveID
		if effective {
			view = F("%d settings in effect in this project", c.nFile)
			if m.filter != "" {
				view += fmt.Sprintf(" · %d match %q", len(vis), m.filter)
			}
		}
		if !t.Exists && t.ID != "desktop-policy" {
			view += " · the file does not exist yet: setting a value creates it"
		}
		r.WriteString(sMuted.Render(view) + "\n")

		// lines with category headers; the window follows the selected setting
		type line struct {
			text string
			pos  int // position in vis, -1 for headers
		}
		keyW := clamp(rightW*42/100, 18, 44)
		valW := max(8, rightW-keyW-8)
		var lines []line
		cat := ""
		for j, i := range vis {
			e := c.entries[i]
			if e.Category != cat {
				cat = e.Category
				lines = append(lines, line{sTitle.Render("── " + cat + " "), -1})
			}
			mark := "  "
			if e.Set {
				mark = sOK.Render(gMarked) + " "
			}
			val := clipEnd(clean(e.Value), valW)
			switch {
			case effective:
				fromW := min(22, valW/2)
				val = fmt.Sprintf("%-*s %s", valW-fromW-1, clipEnd(clean(e.Value), valW-fromW-1), sMuted.Render(clipEnd(e.From, fromW)))
			case e.Secret:
				val = sWarn.Render(val + " secret")
			case !e.Set:
				val = gOff + " default"
				if e.Default != "" {
					val += " " + e.Default
				}
				val = sMuted.Render(clipEnd(val, valW))
			case !e.Editable:
				val = sMuted.Render(val)
			}
			name := clipEnd(e.Key, keyW)
			if e.Deprecated {
				name = clipEnd(e.Key+" (old)", keyW)
			}
			text := fmt.Sprintf("%s%-*s %s", mark, keyW, name, val)
			if !e.Set && !e.Editable {
				text = sMuted.Render(stripStyles(text, rightW))
			}
			if c.focus == 1 && j == c.esel {
				text = sSel.Render(gCursor + " " + stripStyles(text, rightW-4))
			} else {
				text = "  " + text
			}
			lines = append(lines, line{text, j})
		}
		cur := 0
		for i, ln := range lines {
			if ln.pos == c.esel {
				cur = i
			}
		}
		used := strings.Count(r.String(), "\n")
		room := max(3, h-used-5)
		start := 0
		if c.focus == 1 {
			start = clamp(cur-room+3, 0, max(0, len(lines)-room))
		}
		var shown []string
		for i := start; i < len(lines) && i < start+room; i++ {
			shown = append(shown, lines[i].text)
		}
		if len(vis) == 0 && c.err == "" {
			shown = append(shown, sMuted.Render("  nothing matches"))
		}
		r.WriteString(withScrollbar(shown, len(lines), start, min(room, max(1, len(lines))), rightW+2) + "\n")
		if p := position(start, room, len(lines)); p != "" {
			r.WriteString(p + "\n")
		}
		if c.focus == 1 && c.esel < len(vis) {
			e := c.entries[vis[c.esel]]
			var info []string
			if e.Help != "" {
				info = append(info, e.Help)
			}
			kind := e.Type
			if len(e.Choices) > 0 {
				kind = "one of: " + strings.Join(e.Choices, ", ")
			}
			info = append(info, kind)
			if e.Default != "" {
				info = append(info, "default "+e.Default)
			}
			if e.Scope != "" && !strings.HasPrefix(strings.ToLower(e.Scope), "any") {
				info = append(info, "read from: "+e.Scope)
			}
			if e.Deprecated {
				info = append(info, "deprecated")
			}
			if !e.Documented && e.Set {
				info = append(info, "not in Anthropic's settings reference")
			}
			if effective {
				info = append(info, F("from: %s", e.From))
				if e.Merged {
					info = append(info, T("lists from every file combine"))
				}
				if len(e.Overrides) > 0 {
					info = append(info, F("wins over: %s", strings.Join(e.Overrides, ", ")))
				}
			}
			r.WriteString(lipgloss.NewStyle().Width(rightW).Render(sMuted.Render(strings.Join(info, " · "))))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(leftW).Render(l.String()), "  ", r.String())
}
