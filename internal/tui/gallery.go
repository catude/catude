package tui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/presets"
	"github.com/catude/catude/internal/profiles"
)

// galleryState is the preset gallery (Profiles → p): work profiles based on the public setups of
// well-known practitioners, with their process, tools, agents and sources.
type galleryState struct {
	open   bool
	list   []presets.Preset
	err    string
	sel    int
	scroll int
	armed  string
	busy   string
}

func (m *model) openGallery() {
	ps, err := presets.Load()
	m.gal = galleryState{open: true, list: ps}
	if err != nil {
		m.gal.err = err.Error()
	}
}

func (m model) galleryKey(k tea.KeyMsg) (model, tea.Cmd) {
	g := &m.gal
	switch k.String() {
	case "esc", "q", "p", "Q", "P":
		g.open = false
	case "up", "k":
		g.sel, g.scroll, g.armed = clamp(g.sel-1, 0, len(g.list)-1), 0, ""
	case "down", "j":
		g.sel, g.scroll, g.armed = clamp(g.sel+1, 0, len(g.list)-1), 0, ""
	case "pgdown", "J":
		g.scroll += 5
	case "pgup", "K":
		g.scroll = max(0, g.scroll-5)
	case "enter", "l", "L":
		if g.sel >= len(g.list) || g.busy != "" {
			return m, nil
		}
		p := g.list[g.sel]
		if g.armed != p.ID {
			g.armed = p.ID
			m.status = "press enter again to download the pinned files of " + p.Name + ", verify and audit them, and save the work profile " + p.ID
			return m, nil
		}
		g.armed, g.busy = "", p.ID
		m.status = m.spinner() + " loading preset " + p.ID + " (download pinned files, verify, audit)…"
		return m, loadPreset(m.inv, p)
	}
	return m, nil
}

func loadPreset(inv inventory.Inventory, p presets.Preset) tea.Cmd {
	return func() tea.Msg {
		card, notes := p.Card()
		st, err := profiles.Build(context.Background(), p.ID, card, profiles.BuildOptions{Inventory: &inv})
		if err != nil {
			if st != nil {
				st.Discard()
			}
			return profileSavedMsg{err: err}
		}
		defer st.Discard()
		if n := st.HighFindings(); n > 0 {
			return profileSavedMsg{err: fmt.Errorf("preset %s not saved: %d high-severity audit findings in its files", p.ID, n)}
		}
		// replace only a profile that was made from this same preset: never someone's own profile
		if old, err := profiles.Get(p.ID); err == nil && old.Card.ID != "preset:"+p.ID {
			return profileSavedMsg{err: fmt.Errorf("a profile named %s already exists and was not made from this preset: rename or delete it first", p.ID)}
		}
		pr, err := st.Commit(true)
		if err != nil {
			return profileSavedMsg{err: err}
		}
		notes = append(notes, st.Notes...)
		if len(st.Missing) > 0 {
			notes = append(notes, "plugins to install yourself: "+strings.Join(st.Missing, " ; "))
		}
		if len(st.Secrets) > 0 {
			notes = append(notes, "set "+strings.Join(st.Secrets, ", "))
		}
		return profileSavedMsg{pr, notes, nil}
	}
}

func installedTool(name string) bool {
	for _, b := range strings.FieldsFunc(strings.ToLower(name), func(r rune) bool { return r == '/' || r == ' ' || r == ',' || r == '(' || r == ')' }) {
		if _, err := exec.LookPath(b); err == nil {
			return true
		}
	}
	return false
}

func (m model) galleryView(h int) string {
	g := m.gal
	listW := 30
	var left strings.Builder
	left.WriteString(sHeading.Render(T("Presets")) + "\n" + sMuted.Render("based on public setups") + "\n\n")
	for i, p := range g.list {
		l := fmt.Sprintf("%-26s", clipEnd(p.Name, 26))
		if i == g.sel {
			left.WriteString(sSel.Render(gCursor+" "+l) + "\n")
		} else {
			left.WriteString("  " + l + "\n")
		}
	}
	if len(g.list) == 0 {
		left.WriteString(sMuted.Render(firstNonEmpty(g.err, "no presets embedded")))
	}
	right := ""
	if g.sel < len(g.list) {
		right = m.presetDetail(g.list[g.sel], max(30, m.w-listW-10), h)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(listW).Render(left.String()), right)
}

func (m model) presetDetail(p presets.Preset, w, h int) string {
	art := p.Art()
	artW := 0
	for _, a := range art {
		artW = max(artW, lipgloss.Width(a))
	}
	textW := max(20, w-artW-4)
	var head strings.Builder
	head.WriteString(sTitle.Render(clean(p.Name)) + "  " + sMuted.Render(clean(p.Handle)) + "\n")
	head.WriteString(lipgloss.NewStyle().Width(textW).Render(sMuted.Render(clean(p.Role))) + "\n\n")
	head.WriteString(lipgloss.NewStyle().Width(textW).Render(clean(p.Philosophy)) + "\n\n")
	head.WriteString(lipgloss.NewStyle().Width(textW).Render(sWarn.Render("Based on what "+p.Name+" has published. Not affiliated with or endorsed by them.") + sMuted.Render(" Sources below.")))
	box := sTitle.Render(strings.Join(art, "\n"))
	top := lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(textW+2).Render(head.String()), box)

	var b strings.Builder
	wrap := lipgloss.NewStyle().Width(w)
	if len(p.Process) > 0 {
		b.WriteString("\n" + sHeading.Render(T("Process")) + "\n")
		for i, s := range p.Process {
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, fmt.Sprintf("  %d. ", i+1), lipgloss.NewStyle().Width(w-6).Render(clean(s))) + "\n")
		}
	}
	if len(p.Tools) > 0 {
		var ts []string
		for _, t := range p.Tools {
			mark := gOff
			if installedTool(t) {
				mark = gOK
			}
			ts = append(ts, mark+" "+clean(t))
		}
		b.WriteString("\n" + sHeading.Render(T("Tools")) + sMuted.Render("  "+gOK+" = installed here") + "\n" + wrap.Render("  "+strings.Join(ts, "   ")) + "\n")
	}
	have := map[string]bool{}
	for _, it := range m.inv.Items {
		have[strings.ToLower(string(it.Kind)+":"+it.Name)] = true
	}
	list := func(title string, kind inventory.Kind, as []presets.Artifact) {
		if len(as) == 0 {
			return
		}
		b.WriteString("\n" + sHeading.Render(title) + "\n")
		for _, a := range as {
			mark := gOff
			if have[strings.ToLower(string(kind)+":"+a.Name)] {
				mark = gOK
			}
			pin := sMuted.Render(" " + a.Repo)
			if ok, why := a.Installable(); ok {
				pin += sMuted.Render("@" + a.Commit[:7] + " " + a.License)
			} else {
				pin += sWarn.Render(" (link only: " + why + ")")
			}
			b.WriteString(wrap.Render(fmt.Sprintf("  %s %s%s %s", mark, clean(a.Name), pin, sMuted.Render(clipEnd(clean(a.Description), 60)))) + "\n")
		}
	}
	list("Skills", inventory.Skill, p.Skills)
	list("Agents", inventory.Agent, p.Agents)
	list("Commands", inventory.Command, p.Commands)
	if len(p.Plugins)+len(p.Marketplaces) > 0 {
		b.WriteString("\n" + sHeading.Render(T("Plugins & marketplaces")) + "\n")
		for _, mk := range p.Marketplaces {
			b.WriteString("  " + sMuted.Render("claude plugin marketplace add ") + mk.Repo + "\n")
		}
		for _, pl := range p.Plugins {
			mark := gOff
			if have["plugin:"+strings.ToLower(pl.ID)] {
				mark = gOK
			}
			b.WriteString("  " + mark + " " + pl.ID + "\n")
		}
	}
	if len(p.MCP) > 0 {
		b.WriteString("\n" + sHeading.Render(T("MCP servers")) + "\n")
		for _, mc := range p.MCP {
			b.WriteString(wrap.Render("  "+clean(mc.Name)+sMuted.Render(" "+firstNonEmpty(mc.Add, mc.URL))) + "\n")
		}
	}
	if s := p.Settings; s.Model != "" || s.Notes != "" {
		b.WriteString("\n" + sHeading.Render(T("Settings")) + "\n" + wrap.Render("  "+strings.TrimSpace(s.Model+" "+s.EffortLevel+" "+clean(s.Notes))) + "\n")
	}
	if len(p.UsersWant) > 0 {
		b.WriteString("\n" + sHeading.Render(T("What its users ask for")) + "\n")
		for _, u := range p.UsersWant {
			b.WriteString(wrap.Render("  · "+clean(u.Point)+sMuted.Render(" "+u.URL)) + "\n")
		}
	}
	b.WriteString("\n" + sHeading.Render(T("Sources")) + "\n")
	for _, s := range p.Sources {
		b.WriteString(wrap.Render("  "+clean(s.Title)+sMuted.Render(" "+s.URL+" "+s.Date)) + "\n")
	}
	if p.FactChecked != "" {
		b.WriteString(sMuted.Render(wrap.Render("  checked against the sources on "+p.FactChecked)) + "\n")
	}
	if len(p.Unverified) > 0 {
		b.WriteString(sWarn.Render(wrap.Render("  not verified: "+clean(strings.Join(p.Unverified, "; ")))) + "\n")
	}
	lines := strings.Split(top+"\n"+b.String(), "\n")
	start := clamp(m.gal.scroll, 0, max(0, len(lines)-h))
	end := min(len(lines), start+h)
	return strings.Join(lines[start:end], "\n")
}
