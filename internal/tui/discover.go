package tui

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/catalog"
)

type installedMsg struct {
	what string
	err  error
}

var (
	discoverEntries []catalog.Entry
	discoverDate    string
	discoverErr     error
)

func init() {
	c, err := catalog.Load()
	if err != nil {
		discoverErr = err
		return
	}
	discoverEntries, discoverDate = c.Entries(), c.Generated
}

var sectionIcon = map[string]string{"marketplace": "▣", "plugin": "◆", "mcp": "⇄", "skill": "✦", "tool": "▸", "practice": "✓"}

// installedKeys lists names present on this machine so the catalog can mark them.
func (m model) installedKeys() map[string]bool {
	have := map[string]bool{}
	for _, it := range m.inv.Items {
		have[strings.ToLower(it.Name)] = true
	}
	return have
}

func (m model) discoverRows(match func(...string) bool) []row {
	if discoverErr != nil {
		return nil
	}
	have := m.installedKeys()
	var out []row
	for i := range discoverEntries {
		e := discoverEntries[i]
		tags := make([]string, 0, len(e.Tags))
		for _, t := range e.Tags {
			tags = append(tags, "#"+t)
		}
		if !match(append([]string{e.Section, e.Name, e.Description, e.Category}, tags...)...) {
			continue
		}
		mark := "  "
		if e.Key != "" && have[strings.ToLower(e.Key)] {
			mark = markOK()
		}
		label := sectionIcon[e.Section] + " " + e.Name
		if e.Official {
			label += sOK.Render(" ●")
		}
		if e.Stars > 0 {
			label += sMuted.Render(" ★" + humanInt(e.Stars))
		}
		if e.Quiet(discoverDate) {
			label += sWarn.Render(" " + T("quiet"))
		}
		if len(tags) > 0 {
			label += sMuted.Render("  " + strings.Join(tags[:min(3, len(tags))], " "))
		}
		out = append(out, row{text: mark + label, entry: &e, style: lipgloss.NewStyle()})
	}
	return out
}

func (m model) discoverDetail(e *catalog.Entry, w int) string {
	var b strings.Builder
	b.WriteString(sHeading.Render(e.Name) + "\n")
	meta := e.Section
	if e.Category != "" {
		meta += " · " + e.Category
	}
	if e.Official {
		meta += " · official"
	}
	if e.Stars > 0 {
		meta += " · ★ " + humanInt(e.Stars)
	}
	if e.License != "" {
		meta += " · " + e.License
	}
	switch {
	case e.Archived:
		meta += " · " + T("archived repository")
	case e.Quiet(discoverDate):
		meta += " · " + F("no changes since %s", e.Pushed)
	case e.Pushed != "":
		meta += " · " + F("last change %s", e.Pushed)
	}
	b.WriteString(sMuted.Render(meta) + "\n")
	if len(e.Tags) > 0 {
		b.WriteString(sTitle.Render("#"+strings.Join(e.Tags, "  #")) + "\n")
	}
	b.WriteString("\n")
	if e.Description != "" {
		b.WriteString(e.Description + "\n\n")
	}
	if e.URL != "" {
		b.WriteString(sMuted.Render("• ") + e.URL + "\n")
	}
	if e.Command != "" {
		b.WriteString("\n" + sHeading.Render(T("install")) + "\n" + e.Command + "\n\n")
		if _, err := catalog.Args(e.Command); err == nil {
			b.WriteString(sOK.Render("i  run it (press twice; Claude Code asks its own questions)"))
		} else if errors.Is(err, catalog.ErrNeedsInput) {
			b.WriteString(sWarn.Render("needs a value from you (usually a token): copy it and run it yourself"))
		}
	}
	b.WriteString("\n\n" + sMuted.Render("curated catalog · "+discoverDate+" · catude never installs anything without you"))
	return lipgloss.NewStyle().Width(w).Render(b.String())
}

// installSelected runs the selected entry's official command after a second press.
func (m model) installSelected() (tea.Model, tea.Cmd) {
	rs := m.rows()
	i := m.cursor[m.tab]
	if i < 0 || i >= len(rs) || rs[i].entry == nil || rs[i].entry.Command == "" {
		m.status = "nothing to install for this entry"
		return m, nil
	}
	e := rs[i].entry
	args, err := catalog.Args(e.Command)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	if m.armed != e.Command {
		m.armed = e.Command
		m.status = "press i again to run: " + e.Command
		return m, nil
	}
	m.armed = ""
	bin, err := exec.LookPath("claude")
	if err != nil {
		m.status = "claude is not on PATH"
		return m, nil
	}
	cmd := exec.Command(bin, args[1:]...)
	cmd.Dir = m.env.ProjectDir
	name := e.Name
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return installedMsg{name, err} })
}

func humanInt(n int) string {
	trim := func(f float64) string { return strings.TrimSuffix(fmt.Sprintf("%.1f", f), ".0") }
	switch {
	case n >= 1_000_000:
		return trim(float64(n)/1e6) + "M"
	case n >= 1000:
		return trim(float64(n)/1e3) + "k"
	}
	return fmt.Sprint(n)
}
