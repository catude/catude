package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/claudecfg"
)

// Presets panel of the Config tab: sourced groups of settings for one goal, previewed against the
// selected file before anything is written.

type cfgPresets struct {
	open  bool
	sel   int
	tag   string // "" = every tag
	armed string // id of the preset waiting for a second enter
}

func presetTags(ps []claudecfg.Preset) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range ps {
		for _, t := range p.Tags {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	sortStrings(out)
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// visiblePresets applies the tag and the text filter.
func (m model) visiblePresets() []claudecfg.Preset {
	ps, _ := claudecfg.Presets()
	f := strings.ToLower(m.filter)
	var out []claudecfg.Preset
	for _, p := range ps {
		if m.cfg.ps.tag != "" && !containsStr(p.Tags, m.cfg.ps.tag) {
			continue
		}
		if f != "" && !strings.Contains(strings.ToLower(p.Name+" "+p.Summary+" "+strings.Join(p.Tags, " ")+" "+p.By), f) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func (m model) presetsKey(k tea.KeyMsg) (model, tea.Cmd, bool) {
	ps := m.visiblePresets()
	p := &m.cfg.ps
	switch k.String() {
	case "esc", "p":
		p.open, p.armed = false, ""
		m.filter = ""
	case "up", "k":
		p.sel, p.armed = clamp(p.sel-1, 0, len(ps)-1), ""
	case "down", "j":
		p.sel, p.armed = clamp(p.sel+1, 0, len(ps)-1), ""
	case "t", "T":
		tags := append([]string{""}, presetTags(ps0())...)
		i := 0
		for j, t := range tags {
			if t == p.tag {
				i = j
			}
		}
		if k.String() == "T" {
			i = (i + len(tags) - 1) % len(tags)
		} else {
			i = (i + 1) % len(tags)
		}
		p.tag, p.sel, p.armed = tags[i], 0, ""
	case "enter":
		if p.sel >= len(ps) {
			return m, nil, true
		}
		pr := ps[p.sel]
		t, _ := m.cfgSelected()
		if why := pr.Unsupported(); why != "" {
			m.status = gWarn + " " + pr.Name + ": " + T(why)
			return m, nil, true
		}
		if p.armed != pr.ID {
			p.armed = pr.ID
			m.status = F("press enter again to apply %s to %s (a backup is taken first)", pr.Name, t.Label)
			return m, nil, true
		}
		p.armed = ""
		backup, err := claudecfg.ApplyPreset(t, pr)
		if err != nil {
			m.status = gFail + " " + err.Error()
			return m, nil, true
		}
		m.status = F("%s applied to %s", pr.Name, t.Label)
		if backup != "" {
			m.status += " · " + T("backup") + " " + shorten(backup, 36)
		}
		m.loadCfgEntries(true)
		return m, m.load(), true
	default:
		return m, nil, false
	}
	return m, nil, true
}

func ps0() []claudecfg.Preset { ps, _ := claudecfg.Presets(); return ps }

func riskMark(r string) string {
	switch r {
	case "high":
		return sHigh.Render(gWarn)
	case "medium":
		return sWarn.Render(gWarn)
	}
	return sOK.Render(gOK)
}

func (m model) presetsView(h int) string {
	ps := m.visiblePresets()
	t, _ := m.cfgSelected()
	leftW := clamp(m.w*36/100, 34, 54)
	rightW := max(30, m.w-leftW-10)
	var l strings.Builder
	tag := T("all")
	if m.cfg.ps.tag != "" {
		tag = m.cfg.ps.tag
	}
	l.WriteString(sHeading.Render(T("Configuration presets")) + "\n")
	l.WriteString(sMuted.Render(F("%d with sources · tag: %s (t changes)", len(ps), tag)) + "\n\n")
	room := max(3, h-5)
	sel := clamp(m.cfg.ps.sel, 0, max(0, len(ps)-1))
	start := clamp(sel-room+2, 0, max(0, len(ps)-room))
	var lines []string
	for i := start; i < len(ps) && i < start+room; i++ {
		p := ps[i]
		line := riskMark(p.Risk) + " " + truncate(p.Name, leftW-8)
		if p.Unsupported() != "" {
			line = sMuted.Render(stripStyles(line, leftW-6))
		}
		if i == sel {
			line = sSel.Render(gCursor + " " + stripStyles(line, leftW-6))
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}
	if len(ps) == 0 {
		lines = append(lines, sMuted.Render("  "+T("nothing matches")))
	}
	l.WriteString(withScrollbar(lines, len(ps), start, min(room, max(1, len(ps))), leftW-2))

	var r strings.Builder
	if sel < len(ps) {
		p := ps[sel]
		wrap := lipgloss.NewStyle().Width(rightW)
		r.WriteString(sTitle.Render(p.Name) + "\n")
		r.WriteString(sMuted.Render(F("by %s · tags: %s · meant for the %s settings", p.By, strings.Join(p.Tags, ", "), p.Scope)) + "\n\n")
		r.WriteString(wrap.Render(p.Summary) + "\n")
		if p.Risk != "low" {
			r.WriteString(wrap.Render(riskMark(p.Risk)+" "+sWarn.Render(F("risk %s: %s", p.Risk, p.RiskReason))) + "\n")
		}
		if why := p.Unsupported(); why != "" {
			r.WriteString(sWarn.Render(gWarn+" "+T(why)) + "\n")
		}
		r.WriteString("\n" + sHeading.Render(F("Changes in %s", t.Label)) + "\n")
		plan, err := claudecfg.Plan(t, p)
		switch {
		case err != nil:
			r.WriteString(sHigh.Render(err.Error()) + "\n")
		default:
			for _, c := range plan {
				if c.Same {
					r.WriteString("  " + sOK.Render(gOK) + " " + truncate(c.Key+" = "+c.After, rightW-6) + sMuted.Render("  "+T("already so")) + "\n")
					continue
				}
				r.WriteString("  " + sTitle.Render("+") + " " + truncate(c.Key+" = "+c.After, rightW-6) + "\n")
				if c.Before != "" {
					r.WriteString(sMuted.Render("      "+truncate(T("was")+" "+c.Before, rightW-8)) + "\n")
				}
			}
		}
		if p.Note != "" {
			r.WriteString("\n" + wrap.Render(sMuted.Render(p.Note)) + "\n")
		}
		r.WriteString("\n" + sHeading.Render(T("Sources")) + "\n")
		for _, s := range p.Sources {
			r.WriteString(sMuted.Render("  "+truncate(s.URL, rightW-4)) + "\n")
			r.WriteString(wrap.Render("    “"+truncate(s.Quote, rightW*2)+"”") + "\n")
		}
		if m.cfg.ps.armed == p.ID {
			r.WriteString("\n" + sSel.Render(" "+F("enter again: apply to %s", t.Label)+" "))
		}
	}
	right := strings.Join(strings.Split(r.String(), "\n")[:min(h, len(strings.Split(r.String(), "\n")))], "\n")
	return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(leftW).Render(l.String()), "  ", right)
}

// presetsHint is the key line of the presets panel.
func presetsHint() string {
	return T("↑↓ choose · t tag · / search · enter preview, enter again apply · esc back")
}

var _ = fmt.Sprint
