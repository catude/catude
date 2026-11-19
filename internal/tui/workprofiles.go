package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/presets"
	"github.com/catude/catude/internal/profiles"
)

type workProfilesMsg struct {
	ps  []profiles.Profile
	err error
}

type profileSavedMsg struct {
	p     profiles.Profile
	notes []string
	err   error
}

type profileDoneMsg struct {
	name string
	err  error
}

func loadWorkProfiles() tea.Cmd {
	return func() tea.Msg {
		ps, err := profiles.List()
		return workProfilesMsg{ps, err}
	}
}

// markable reports whether items of the current tab can go into a work profile.
func (m model) markable() (inventory.Kind, bool) {
	k, ok := kindOf[m.tab]
	if !ok || k == inventory.Available || k == inventory.Hook {
		return k, false
	}
	return k, true
}

func markKey(k inventory.Kind, name string) string { return string(k) + ":" + name }

// toggleMark adds or removes the selected item from the profile being assembled.
func (m *model) toggleMark() {
	k, ok := m.markable()
	it := m.selectedItem()
	if !ok || it == nil {
		m.status = "space marks skills, agents, commands, MCP servers and plugins for a work profile"
		return
	}
	if k == inventory.MCP && it.Path == "" {
		m.status = it.Name + " is a claude.ai connector: it belongs to your account, not to a profile"
		return
	}
	key := markKey(k, it.Name)
	if m.marked[key] {
		delete(m.marked, key)
	} else {
		m.marked[key] = true
	}
	m.gen++
	m.status = fmt.Sprintf("%d marked for a new profile · P to save them as a profile · space to unmark", len(m.marked))
	m.move(1)
}

func (m model) selection() profiles.Selection {
	var s profiles.Selection
	keys := make([]string, 0, len(m.marked))
	for k := range m.marked {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		kind, name, _ := strings.Cut(key, ":")
		switch inventory.Kind(kind) {
		case inventory.Skill:
			s.Skills = append(s.Skills, name)
		case inventory.Agent:
			s.Agents = append(s.Agents, name)
		case inventory.Command:
			s.Commands = append(s.Commands, name)
		case inventory.MCP:
			s.MCP = append(s.MCP, name)
		case inventory.Plugin:
			s.Plugins = append(s.Plugins, name)
		}
	}
	return s
}

// saveProfile builds the profile from the marked items, audits it and installs it unless a high finding shows up.
func (m model) saveProfile(name string) tea.Cmd {
	inv, sel := m.inv, m.selection()
	return func() tea.Msg {
		st, err := profiles.FromLocal(context.Background(), name, inv, sel)
		if err != nil {
			return profileSavedMsg{err: err}
		}
		defer st.Discard()
		if n := st.HighFindings(); n > 0 {
			return profileSavedMsg{err: fmt.Errorf("not saved: %d high-severity audit findings in the selected items (see Audit)", n)}
		}
		p, err := st.Commit(false)
		notes := st.Notes
		if len(st.Secrets) > 0 {
			notes = append(notes, "set "+strings.Join(st.Secrets, ", "))
		}
		return profileSavedMsg{p, notes, err}
	}
}

// startProfile suspends the TUI and runs Claude Code with the selected work profile.
func (m model) startProfile(p profiles.Profile, exclusive bool) (tea.Model, tea.Cmd) {
	cmd, err := profiles.Command(p, profiles.LaunchOptions{Exclusive: exclusive})
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	cmd.Dir = m.env.ProjectDir
	name := p.Name
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return profileDoneMsg{name, err} })
}

func (m model) workProfilesView() string {
	var b strings.Builder
	b.WriteString(sHeading.Render(T("Work profiles")) + sMuted.Render("  n new · e edit · s switch ("+gMarked+" = active) · enter start Claude with it · d delete") + "\n")
	if m.wpErr != "" {
		b.WriteString(sWarn.Render("  "+m.wpErr) + "\n")
	}
	if len(m.workProfiles) == 0 {
		b.WriteString(sMuted.Render("  none yet · press n to create one (choose skills, agents, MCP, plugins) · or: catude profile load <card>") + "\n")
	}
	if ps, err := presets.Load(); err == nil && len(ps) > 0 {
		names := make([]string, 0, 5)
		for i, p := range ps {
			if i == 5 {
				break
			}
			names = append(names, p.Name)
		}
		b.WriteString("  " + sTitle.Render("p") + " presets: " + fmt.Sprint(len(ps)) + " ready-made profiles based on public setups (" + strings.Join(names, ", ") + "…)\n")
	}
	for i, p := range m.workProfiles {
		desc := firstNonEmpty(p.Err, p.Summary())
		mode := "+ your setup"
		if p.Options.Exclusive {
			mode = "only this"
		}
		act := "  "
		if p.Name == m.active {
			act = sOK.Bold(true).Render(gMarked) + " "
		}
		l := fmt.Sprintf("%s%-22s %-44s %-13s %s", act, clipEnd(p.Name, 22), clipEnd(desc, 44), mode, clipEnd(clean(p.Card.Summary), 30))
		if i == m.profSel {
			l = sSel.Render(gCursor + " " + stripStyles(l, m.w-10))
		} else if p.Err != "" {
			l = "  " + sWarn.Render(l)
		} else {
			l = "  " + l
		}
		b.WriteString(l + "\n")
	}
	return b.String()
}
