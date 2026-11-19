package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/sessions"
	"github.com/catude/catude/internal/tools"
)

// ---- Tools tab: runtimes, hooks' tools and companion tools ----

type toolsState struct {
	list    []tools.Tool
	running bool
	at      time.Time
	sel     int
}

type toolsMsg struct{ ts []tools.Tool }

func (m *model) ensureTools() tea.Cmd {
	if m.tools.running || (!m.tools.at.IsZero() && time.Since(m.tools.at) < 5*time.Minute) {
		return nil
	}
	m.tools.running = true
	inv := m.inv
	return func() tea.Msg { return toolsMsg{tools.Detect(context.Background(), inv)} }
}

func (m model) toolsView(h int) string {
	var b strings.Builder
	state := "checked " + ago(m.tools.at) + " · r check again"
	if m.tools.running {
		state = m.spinner() + " checking versions…"
	}
	b.WriteString(sHeading.Render(T("Tools your Claude setup uses")) + sMuted.Render("  "+state) + "\n")
	missing := 0
	for _, t := range m.tools.list {
		if !t.Found && len(t.UsedBy) > 0 {
			missing++
		}
	}
	if missing > 0 {
		b.WriteString(sHigh.Render(fmt.Sprintf("%s %d tools are needed by your MCP servers or hooks but are not installed: those servers fail to start", gFail, missing)) + "\n")
	}
	var lines []string
	cat := ""
	order := map[string]int{"runtime": 0, "hook & context": 1, "accounts & usage": 2, "security": 3, "dev": 4}
	ts := append([]tools.Tool{}, m.tools.list...)
	sort.SliceStable(ts, func(i, j int) bool { return order[ts[i].Category] < order[ts[j].Category] })
	idx := 0
	for _, t := range ts {
		if t.Category != cat {
			cat = t.Category
			lines = append(lines, "", sHeading.Render(cat))
		}
		mark := markOff()
		state := sMuted.Render("not installed")
		switch {
		case t.Stub:
			mark, state = markWarn(), sWarn.Render("Store stub only")
		case t.Found:
			mark, state = markOK(), firstNonEmpty(t.Version, "installed")
		case len(t.UsedBy) > 0:
			mark, state = markFail(), sHigh.Render("missing")
		}
		used := ""
		if len(t.UsedBy) > 0 {
			used = sMuted.Render(" · used by " + clipEnd(strings.Join(t.UsedBy, ", "), 60))
		}
		l := fmt.Sprintf("%s%-14s %-22s %s%s", mark, t.Name, clipEnd(stripStyles(state, 22), 22), sMuted.Render(clipEnd(t.Purpose, 46)), used)
		if !t.Found && t.Install != "" {
			l += sMuted.Render(" · install: " + t.Install)
		}
		l = lipgloss.NewStyle().MaxWidth(m.w - 10).Render(l)
		if idx == m.tools.sel {
			l = sSel.Render(gCursor + " " + stripStyles(l, m.w-10))
		} else {
			l = "  " + l
		}
		lines = append(lines, l)
		idx++
	}
	head := strings.Count(b.String(), "\n")
	room := max(3, h-head-2)
	if len(lines) > room {
		lines = lines[:room]
	}
	b.WriteString(strings.Join(lines, "\n"))
	if m.tools.sel < len(ts) {
		if p := ts[m.tools.sel].Path; p != "" {
			b.WriteString("\n\n" + sMuted.Render(p))
		}
	}
	return b.String()
}

// ---- Usage tab: tokens per day, model and project, from local transcripts ----

type usageState struct {
	rep     *sessions.UsageReport
	running bool
	at      time.Time
	days    int
	cache   *sessions.UsageCache
}

type usageMsg struct {
	rep sessions.UsageReport
	err error
}

func (m *model) ensureUsage() tea.Cmd {
	if m.usage.running || (m.usage.rep != nil && time.Since(m.usage.at) < 2*time.Minute) {
		return nil
	}
	if m.usage.days == 0 {
		m.usage.days = 14
	}
	if m.usage.cache == nil {
		m.usage.cache = &sessions.UsageCache{}
	}
	m.usage.running = true
	c, ps, days := m.usage.cache, m.projects, m.usage.days
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		r, err := c.Usage(ctx, ps, days)
		return usageMsg{r, err}
	}
}

func (m model) usageView(h int) string {
	var b strings.Builder
	state := "from your transcripts · measured " + ago(m.usage.at) + " · r refresh · [ ] 7 / 14 / 30 days"
	if m.usage.running {
		state = m.spinner() + " reading transcripts of the last " + fmt.Sprint(m.usage.days) + " days (first time can take a minute)…"
	}
	b.WriteString(sHeading.Render(fmt.Sprintf("Usage · last %d days", m.usage.days)) + sMuted.Render("  "+state) + "\n")
	for _, a := range m.accts {
		if !a.Active {
			continue
		}
		if w5, w7 := a.FiveHour(), a.SevenDay(); w5 != nil || w7 != nil {
			line := "Plan limits (claude-swap): "
			if w5 != nil {
				line += "5 h " + bar(w5.Pct/100, 12) + fmt.Sprintf(" %.0f%%", w5.Pct) + windowNote(w5) + "  "
			}
			if w7 != nil {
				line += "7 days " + bar(w7.Pct/100, 12) + fmt.Sprintf(" %.0f%%", w7.Pct) + windowNote(w7)
			}
			b.WriteString(line + "\n")
		}
	}
	r := m.usage.rep
	if r == nil {
		return b.String()
	}
	top := 1
	total, cacheR := 0, 0
	for _, d := range r.Days {
		top = max(top, d.Billable())
		total += d.Billable()
		cacheR += d.CacheR
	}
	b.WriteString(sMuted.Render(fmt.Sprintf("%s tokens processed (new input + cache writes + output) · %s read from cache · %d transcripts. Plan limits are computed by Anthropic; this is what your machine recorded.", human(total), human(cacheR), r.Files)) + "\n\n")
	if s := healthLine(r.Health); s != "" {
		b.WriteString(sMuted.Render(s) + "\n\n")
	}
	if s := drainView(r.Drain, m.w); s != "" {
		b.WriteString(s + "\n")
	}
	for _, d := range r.Days {
		label := d.Day.Format("Mon 02 Jan")
		if d.Day.Format("2006-01-02") == time.Now().Format("2006-01-02") {
			label = "today     "
		}
		b.WriteString(fmt.Sprintf("  %s %s %8s  %s\n", label, bar(float64(d.Billable())/float64(top), 40), human(d.Billable()), sMuted.Render(fmt.Sprintf("%d responses", d.Responses))))
	}
	type kv struct {
		k string
		v int
	}
	var models []kv
	for k, v := range r.Models {
		models = append(models, kv{k, v})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].v > models[j].v })
	b.WriteString("\n" + sHeading.Render(T("By model")) + "\n")
	for i, x := range models {
		if i == 6 {
			break
		}
		b.WriteString(fmt.Sprintf("  %-34s %8s  %s\n", clipEnd(x.k, 34), human(x.v), bar(float64(x.v)/float64(max(1, total)), 20)))
	}
	b.WriteString("\n" + sHeading.Render(T("By project")) + "\n")
	for i, p := range r.Projects {
		if i == 8 {
			break
		}
		name := filepath.Base(p.Dir)
		if c, ok := m.projCWD[p.Dir]; ok {
			name = c
		}
		b.WriteString(fmt.Sprintf("  %-60s %8s  %s\n", shorten(name, 60), human(p.Billable), bar(float64(p.Billable)/float64(max(1, total)), 20)))
	}
	lines := strings.Split(b.String(), "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}
