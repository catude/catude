package tui

import (
	"fmt"
	"strings"

	"github.com/catude/catude/internal/accounts"
)

func (m model) profilesView(h int) string {
	var b strings.Builder
	b.WriteString(m.workProfilesView() + "\n")
	b.WriteString(sHeading.Render(T("Claude Code installations")) + "\n")
	if len(m.instances) == 0 {
		b.WriteString(sMuted.Render("  none found on PATH, npm, native installer or editor extensions") + "\n")
	}
	for _, in := range m.instances {
		b.WriteString(fmt.Sprintf("  %-26s %-9s %s\n", in.Source, firstNonEmpty(in.Version, "?"), sMuted.Render(shorten(in.Path, m.w-46))))
	}
	b.WriteString("\n" + sHeading.Render(T("Configuration folders")) + sMuted.Render("  CLAUDE_CONFIG_DIR · enter = inspect it in every tab") + "\n")
	for i, d := range m.cfgDirs {
		tag := ""
		switch {
		case d.Active:
			tag = sOK.Render(" ● inspecting")
		case d.Backup:
			tag = sWarn.Render(" backup/credentials · never read")
		case !d.HasData:
			tag = sMuted.Render(" (empty)")
		}
		l := fmt.Sprintf("%-28s %s", d.Label, shorten(d.Path, m.w-60))
		if i+len(m.workProfiles) == m.profSel {
			l = sSel.Render(gCursor + " " + stripStyles(l, m.w-30))
		} else {
			l = "  " + l
		}
		b.WriteString(l + tag + "\n")
	}
	b.WriteString("\n" + sHeading.Render(T("Accounts")) + "\n")
	b.WriteString(m.accountsView())
	if stale := m.stale; len(stale) > 0 {
		b.WriteString("\n" + sWarn.Render(fmt.Sprintf("%d leftover copies of ~/.claude.json may hold old secrets (see Audit)", len(stale))) + "\n")
	}
	lines := strings.Split(b.String(), "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}

// accountsView lists the accounts managed by claude-swap with their 5 h / 7 day usage.
func (m model) accountsView() string {
	var b strings.Builder
	if m.switcher == nil {
		b.WriteString(sMuted.Render("  claude-swap not found · press i twice to install claude-swap=="+accounts.PinnedVersion+" with uv") + "\n")
		return b.String()
	}
	b.WriteString(sMuted.Render(fmt.Sprintf("  claude-swap %s · enter = switch to that account · i i = install/upgrade to %s", firstNonEmpty(m.switcher.Version, "?"), accounts.PinnedVersion)) + "\n")
	for _, w := range m.switcher.Warnings() {
		b.WriteString(sWarn.Render("  ! "+w) + "\n")
	}
	if m.acctErr != "" {
		b.WriteString(sWarn.Render("  "+m.acctErr) + "\n")
	}
	for i, a := range m.accts {
		who := a.Email
		if a.OrganizationName != "" {
			who += " · " + a.OrganizationName
		}
		l := fmt.Sprintf("#%d %-44s", a.Number, clipEnd(who, 44))
		if i+len(m.workProfiles)+len(m.cfgDirs) == m.profSel {
			l = sSel.Render(gCursor + " " + stripStyles(l, 50))
		} else {
			l = "  " + l
		}
		usage := sMuted.Render(" usage unknown")
		if w5, w7 := a.FiveHour(), a.SevenDay(); w5 != nil || w7 != nil {
			usage = ""
			if w5 != nil {
				usage += " 5h " + bar(w5.Pct/100, 10) + fmt.Sprintf(" %3.0f%%", w5.Pct)
			}
			if w7 != nil {
				usage += "  7d " + bar(w7.Pct/100, 10) + fmt.Sprintf(" %3.0f%%", w7.Pct)
				if w7.WillLastToReset != nil && !*w7.WillLastToReset {
					usage += sHigh.Render(" runs out before reset")
				}
			}
		}
		active := ""
		if a.Active {
			active = sOK.Render(" ● active")
		}
		b.WriteString(l + usage + active + "\n")
	}
	return b.String()
}
