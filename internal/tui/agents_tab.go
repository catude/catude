package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/agentmap"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/models"
)

// ---- Agents tab: the agent map ----
//
// Every subagent Claude Code can use here (yours, the project's, plugins', built-in) with what it really
// did in the last 30 days: runs, who launched it, tools, models, tokens and an estimated cost.

type agentsState struct {
	m       agentmap.Map
	running bool
	at      time.Time
	sel     int
}

type agentsMsg struct{ m agentmap.Map }

func (m *model) ensureAgents() tea.Cmd {
	if m.ag.running || (!m.ag.at.IsZero() && time.Since(m.ag.at) < 2*time.Minute) {
		return nil
	}
	m.ag.running = true
	dir, inv := m.env.ClaudeDir, m.inv
	return func() tea.Msg { return agentsMsg{agentmap.Build(dir, inv, 30)} }
}

func (m *model) onAgents(a agentmap.Map) {
	m.ag.m, m.ag.running, m.ag.at = a, false, time.Now()
	m.ag.sel = clamp(m.ag.sel, 0, len(a.Agents)-1)
}

func (m model) agentSelected() *agentmap.Agent {
	if m.ag.sel < 0 || m.ag.sel >= len(m.ag.m.Agents) {
		return nil
	}
	return &m.ag.m.Agents[m.ag.sel]
}

// agentItem is the definition of the selected agent, so e (edit) and space (mark) work as in other tabs.
func (m model) agentItem() *inventory.Item {
	if a := m.agentSelected(); a != nil {
		return a.Defined
	}
	return nil
}

// priceData is the models cache used for estimates (no download from here).
func (m model) priceData() models.Data {
	if m.mo.loaded {
		return m.mo.data
	}
	cachedPrices.once.Do(func() { cachedPrices.d = models.ReadCache(cacheDir()) })
	return cachedPrices.d
}

// runsCost estimates the cost of runs at list prices; known is false when a model has no price.
func (m model) runsCost(runs []agentmap.Run) (float64, bool) {
	d := m.priceData()
	total, known := 0.0, len(runs) > 0
	for _, r := range runs {
		p, ok := d.PriceFor(r.Model)
		if !ok {
			known = false
			continue
		}
		u := r.Usage
		total += p.CostUSD(u.Input, u.CacheCreate, u.CacheRead, u.Output)
	}
	return total, known
}

func (m model) agentsView(h int) string {
	am := m.ag.m
	if m.ag.running && m.ag.at.IsZero() {
		return sMuted.Render(m.spinner() + " " + T("reading the subagent runs of the last 30 days…"))
	}
	leftW := clamp(m.w*42/100, 40, 70)
	rightW := max(30, m.w-leftW-12)
	var l strings.Builder
	head := F("%d subagent runs in %d days · %s new tokens · %s cache reads", am.Runs, am.Days, human(am.Usage.Fresh()), human(am.Usage.CacheRead))
	l.WriteString(sHeading.Render(T("Agent map")) + "\n" + sMuted.Render(clipEnd(head, leftW-4)) + "\n\n")
	nameW := clamp(leftW-36, 10, 30)
	l.WriteString(sMuted.Render(fmt.Sprintf("  %-*s %5s %8s  %s", nameW, clipEnd(T("agent"), nameW), clipEnd(T("runs"), 5), clipEnd(T("new tok"), 8), T("last"))) + "\n")
	room := max(3, h-8)
	start := clamp(m.ag.sel-room+1, 0, max(0, len(am.Agents)-room))
	var rows []string
	for i := start; i < len(am.Agents) && len(rows) < room; i++ {
		a := am.Agents[i]
		mark := sMuted.Render("◦")
		switch {
		case a.Defined != nil:
			mark = sOK.Render(gMarked)
		case !a.BuiltIn:
			mark = sWarn.Render("?")
		}
		last := "-"
		if !a.LastRun.IsZero() {
			last = ago(a.LastRun)
		}
		line := fmt.Sprintf("%s %-*s %5d %8s  %s", mark, nameW, clipEnd(a.Name, nameW), len(a.Runs), human(a.Usage.Fresh()), clipEnd(last, 9))
		if len(a.Runs) == 0 {
			line = sMuted.Render(stripStyles(line, leftW))
		}
		if i == m.ag.sel {
			line = sSel.Render(gCursor + " " + stripStyles(line, leftW-4))
		} else {
			line = "  " + line
		}
		rows = append(rows, line)
	}
	if len(am.Agents) == 0 {
		rows = append(rows, sMuted.Render("  "+T("no agents defined and no subagent runs in this period")))
	}
	l.WriteString(withScrollbar(rows, len(am.Agents), start, min(room, max(1, len(am.Agents))), leftW) + "\n")
	l.WriteString("\n" + whatIsBlock(tAgents, leftW-2))
	l.WriteString(sMuted.Render(gMarked+" "+T("defined here")+"  ◦ "+T("built in")+"  ? "+T("no definition now")) + "\n")
	if u := am.Unused(); len(u) > 0 {
		l.WriteString(sWarn.Render(lipgloss.NewStyle().Width(leftW-2).Render(F("%d defined agents did not run in %d days: their descriptions still load into every session", len(u), am.Days))) + "\n")
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(leftW).Render(l.String()), "  ", m.agentDetail(rightW))
}

func (m model) agentDetail(w int) string {
	a := m.agentSelected()
	if a == nil {
		return ""
	}
	var b strings.Builder
	wrap := func(s string) string { return lipgloss.NewStyle().Width(w).Render(s) }
	b.WriteString(sHeading.Render(a.Name) + "\n")
	switch {
	case a.Defined != nil:
		d := a.Defined
		b.WriteString(sMuted.Render(string(d.Scope)+" · "+shorten(d.Path, w-12)) + "\n")
		if d.Description != "" {
			b.WriteString(wrap(clean(d.Description)) + "\n")
		}
		for _, x := range d.Detail {
			b.WriteString(sMuted.Render(x) + "\n")
		}
	case a.BuiltIn:
		desc := agentmap.Builtins[a.Name]
		b.WriteString(sMuted.Render(T("built into Claude Code")))
		if desc != "" {
			b.WriteString(sMuted.Render(": " + desc))
		}
		b.WriteString("\n")
	default:
		b.WriteString(sWarn.Render(wrap(T("ran in this period, but no definition with this name is installed now (removed, or from a plugin that is off)"))) + "\n")
	}
	b.WriteString("\n")
	if len(a.Runs) == 0 {
		b.WriteString(sMuted.Render(wrap(F("no runs in the last %d days. Its description is still loaded into every session so Claude can choose it: remove it if you do not need it.", m.ag.m.Days))))
		return lipgloss.NewStyle().Width(w).Render(b.String())
	}
	bg := 0
	for _, r := range a.Runs {
		if r.Background {
			bg++
		}
	}
	line := F("%d runs · %d in the background · %d stopped by you", len(a.Runs), bg, a.Stopped)
	if a.Avg > 0 {
		line += " · " + F("%s on average", a.Avg.Round(time.Second))
	}
	b.WriteString(line + "\n")
	u := a.Usage
	b.WriteString(F("tokens: %s new input · %s cache writes · %s output · %s cache reads", human(u.Input), human(u.CacheCreate), human(u.Output), human(u.CacheRead)) + "\n")
	if c, ok := m.runsCost(a.Runs); ok {
		share := ""
		if t := m.ag.m.Usage.Fresh(); t > 0 {
			share = " · " + F("%.0f%% of all subagent new tokens", 100*float64(u.Fresh())/float64(t))
		}
		b.WriteString(F("estimated ~%s at API list prices", usd(c)) + sMuted.Render(share) + "\n")
	}
	counts := func(title string, mp map[string]int, n int) {
		var parts []string
		for _, k := range agentmap.Top(mp, n) {
			parts = append(parts, fmt.Sprintf("%s %d", k, mp[k]))
		}
		if len(parts) > 0 {
			b.WriteString(wrap(sTitle.Render(title+": ")+strings.Join(parts, " · ")) + "\n")
		}
	}
	launched := map[string]int{}
	for k, v := range a.LaunchedBy {
		if k == "main" {
			k = T("your conversation")
		}
		launched[k] = v
	}
	counts(T("launched by"), launched, 4)
	counts(T("models"), a.Models, 3)
	counts(T("tools"), a.Tools, 8)
	projects := map[string]int{}
	for k, v := range a.Projects {
		projects[shorten(k, 40)] = v
	}
	counts(T("projects"), projects, 3)
	b.WriteString("\n" + sTitle.Render(T("Recent runs")) + "\n")
	runs := append([]agentmap.Run(nil), a.Runs...)
	sort.SliceStable(runs, func(i, j int) bool { return runs[i].Start.After(runs[j].Start) })
	for i, r := range runs {
		if i == 6 {
			break
		}
		when := "-"
		if !r.Start.IsZero() {
			when = r.Start.Local().Format("02 Jan 15:04")
		}
		b.WriteString(fmt.Sprintf("%s  %7s  %s\n", sMuted.Render(when), human(r.Usage.Fresh()), clipEnd(clean(r.Description), max(10, w-24))))
	}
	return lipgloss.NewStyle().Width(w).Render(b.String())
}
