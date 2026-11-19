package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/ctxcomp"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/profiles"
)

// ctxState is the Context tab: the composition of the context window at the start of a session in
// the current folder, as measured by Claude Code itself (claude -p /context), optionally with a work
// profile loaded. Results are cached on disk and refreshed in the background.
type ctxState struct {
	comp    *ctxcomp.Composition
	at      time.Time
	profile string // "" = your normal setup
	running bool
	started time.Time
	err     string
	scroll  int
	sel     int            // selected row of the biggest consumers (x switches it off here)
	byProf  map[string]int // last measured total per profile ("" = none), for comparison
}

type ctxCached struct {
	Comp ctxcomp.Composition
	At   time.Time
}

type ctxMsg struct {
	profile string
	comp    ctxcomp.Composition
	err     error
}

const ctxStale = 30 * time.Minute

func (m model) ctxCacheName(profile string) string {
	return "context-" + cacheKey(m.env.ProjectDir, m.env.ClaudeDir, profile)
}

// loadCtxCache shows the last measurement instantly.
func (m *model) loadCtxCache() {
	var c ctxCached
	if readCache(m.ctxCacheName(m.ctx.profile), &c) {
		m.ctx.comp, m.ctx.at = &c.Comp, c.At
		if m.ctx.byProf == nil {
			m.ctx.byProf = map[string]int{}
		}
		m.ctx.byProf[m.ctx.profile] = c.Comp.Used
	}
}

// refreshCtx measures again in the background.
func (m *model) refreshCtx() tea.Cmd {
	if m.ctx.running {
		return nil
	}
	m.ctx.running, m.ctx.started, m.ctx.err = true, time.Now(), ""
	dir, _ := m.env.RunDir() // an untrusted project is measured from a neutral folder: its hooks and MCP servers do not start
	prof, name := m.ctx.profile, m.ctxCacheName(m.ctx.profile)
	var extra []string
	if prof != "" {
		if p, err := profiles.Get(prof); err == nil {
			extra, _ = profiles.Args(p, profiles.LaunchOptions{Exclusive: p.Options.Exclusive})
		}
	}
	return func() tea.Msg {
		c, err := ctxcomp.Run(context.Background(), "", dir, extra...)
		if err == nil {
			writeCache(name, ctxCached{c, time.Now()})
		}
		return ctxMsg{prof, c, err}
	}
}

// ensureCtx starts a refresh when the tab is opened and the data is missing or old.
func (m *model) ensureCtx() tea.Cmd {
	if m.ctx.comp == nil {
		m.loadCtxCache()
	}
	if m.ctx.comp == nil || time.Since(m.ctx.at) > ctxStale {
		return m.refreshCtx()
	}
	return nil
}

func (m *model) onCtxMsg(msg ctxMsg) {
	if msg.profile != m.ctx.profile {
		return // the user switched profile meanwhile
	}
	m.ctx.running = false
	if msg.err != nil {
		m.ctx.err = msg.err.Error()
		return
	}
	c := msg.comp
	m.ctx.comp, m.ctx.at = &c, time.Now()
	if m.ctx.byProf == nil {
		m.ctx.byProf = map[string]int{}
	}
	m.ctx.byProf[msg.profile] = c.Used
}

// cycleCtxProfile switches between "your setup" and each work profile.
func (m *model) cycleCtxProfile() tea.Cmd {
	names := []string{""}
	for _, p := range m.workProfiles {
		names = append(names, p.Name)
	}
	i := 0
	for j, n := range names {
		if n == m.ctx.profile {
			i = j
		}
	}
	m.ctx.profile = names[(i+1)%len(names)]
	m.ctx.comp, m.ctx.running, m.ctx.scroll = nil, false, 0
	return m.ensureCtx()
}

func (m model) contextView(h int) string {
	var b strings.Builder
	who := "your normal setup"
	if m.ctx.profile != "" {
		who = "work profile " + m.ctx.profile
	}
	b.WriteString(sHeading.Render(T("Context at the start of a session")) + sMuted.Render("  in "+shorten(m.env.ProjectDir, 40)+" · with "+who) + "\n")
	if _, trusted := m.env.RunDir(); !trusted {
		b.WriteString(sWarn.Render(gWarn+" This folder is not trusted in Claude Code yet, so its own hooks, MCP servers and CLAUDE.md are not started or counted: you see your user-level setup. Open the folder with claude once and accept the trust prompt to include them.") + "\n")
	}
	switch {
	case m.ctx.running && m.ctx.comp == nil:
		b.WriteString("\n" + m.spinner() + fmt.Sprintf(" asking Claude Code (claude /context)… %ds. It starts every MCP server, usually 20–40 s. You can keep using the other tabs.\n", int(time.Since(m.ctx.started).Seconds())))
	case m.ctx.running:
		b.WriteString(sMuted.Render(fmt.Sprintf("%s refreshing in the background… %ds · showing the measurement from %s", m.spinner(), int(time.Since(m.ctx.started).Seconds()), ago(m.ctx.at))) + "\n")
	case m.ctx.comp != nil:
		b.WriteString(sMuted.Render("measured "+ago(m.ctx.at)+" by Claude Code · r refresh · p compare with a work profile") + "\n")
	}
	if m.ctx.err != "" {
		b.WriteString(sWarn.Render(gWarn+" "+m.ctx.err) + "\n")
	}
	c := m.ctx.comp
	if c == nil {
		if !m.ctx.running {
			b.WriteString("\n" + sMuted.Render("press r to measure") + "\n")
		}
		return b.String()
	}
	pct := 0.0
	if c.Max > 0 {
		pct = float64(c.Used) / float64(c.Max)
	}
	b.WriteString(fmt.Sprintf("\n%s  %s of %s  %s\n", sHeading.Render(T("Used before you type")), human(c.Used), human(c.Max), bar(pct, 30)+fmt.Sprintf(" %.1f%%", 100*pct)))
	if len(m.ctx.byProf) > 1 {
		var parts []string
		keys := make([]string, 0, len(m.ctx.byProf))
		for k := range m.ctx.byProf {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s %s", firstNonEmpty(k, "your setup"), human(m.ctx.byProf[k])))
		}
		b.WriteString(sMuted.Render("  compare: "+strings.Join(parts, " · ")) + "\n")
	}
	var lines []string
	section := func(t string, rs []ctxcomp.Row, limit int) {
		if len(rs) == 0 {
			return
		}
		top := 1
		for _, r := range rs {
			top = max(top, r.Tokens)
		}
		lines = append(lines, "", sHeading.Render(t))
		for i, r := range rs {
			if limit > 0 && i == limit {
				lines = append(lines, sMuted.Render(fmt.Sprintf("  … %d more", len(rs)-limit)))
				break
			}
			lines = append(lines, fmt.Sprintf("  %-46s %8s %s", clipEnd(clean(r.Name), 46), human(r.Tokens), bar(float64(r.Tokens)/float64(top), 20)))
		}
	}
	// the biggest single consumers first: what to remove to save the most
	big := ctxBig(c)
	var cats []ctxcomp.Row // free space and the autocompact reserve are not content: keep the bars readable
	for _, r := range c.Categories {
		if n := strings.ToLower(r.Name); !strings.Contains(n, "free space") && !strings.Contains(n, "autocompact") {
			cats = append(cats, r)
		}
	}
	section("Categories", cats, 0)
	selLine := -1
	if len(big) > 0 {
		top := max(1, big[0].Tokens)
		lines = append(lines, "", sHeading.Render(T("Biggest consumers you control"))+sMuted.Render("  "+T("↑↓ choose · x off in this project (deferred MCP tools load only when searched)")))
		for i, r := range big {
			if i == 12 {
				lines = append(lines, sMuted.Render(fmt.Sprintf("  … %d more", len(big)-12)))
				break
			}
			label := r.label
			if it := m.ctxItem(r); it != nil {
				if o := m.offLabel(*it); o != "" {
					label += " · " + o
				}
			}
			l := fmt.Sprintf("%-46s %8s %s", clipEnd(clean(label), 46), human(r.Tokens), bar(float64(r.Tokens)/float64(top), 20))
			if i == m.ctx.sel {
				l, selLine = sSel.Render(gCursor+" "+l), len(lines)
			} else {
				l = "  " + l
			}
			lines = append(lines, l)
		}
	}
	section("Instructions (CLAUDE.md, rules, imports)", c.Memory, 0)
	section("Skills (name + description are always loaded)", c.Skills, 0)
	section("MCP tools by server (deferred: loaded when searched)", c.MCPByServer(), 0)
	head := strings.Count(b.String(), "\n")
	room := max(1, h-head)
	start := clamp(m.ctx.scroll, 0, max(0, len(lines)-room))
	if selLine >= 0 && (selLine < start || selLine >= start+room) {
		start = clamp(selLine-room/2, 0, max(0, len(lines)-room))
	}
	end := min(len(lines), start+room)
	b.WriteString(withScrollbar(lines[start:end], len(lines), start, min(room, len(lines)), m.w-6))
	if end < len(lines) {
		b.WriteString("\n" + sMuted.Render(fmt.Sprintf("↓ %d more lines", len(lines)-end)))
	}
	return b.String()
}

func ago(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()))
	}
	return t.Format("2 Jan")
}

// CachedContext returns the last context measurement saved for a project ("your normal setup"), if any.
func CachedContext(env paths.Env) (used, max int, at time.Time, ok bool) {
	m := model{env: env}
	var c ctxCached
	if !readCache(m.ctxCacheName(""), &c) {
		return 0, 0, time.Time{}, false
	}
	return c.Comp.Used, c.Comp.Max, c.At, true
}

// bigRow is one consumer of the context that you can switch off: a skill or an MCP server.
type bigRow struct {
	label  string
	kind   string // skill | memory | mcp
	name   string
	Tokens int
}

func ctxBig(c *ctxcomp.Composition) []bigRow {
	var big []bigRow
	for _, r := range c.Skills {
		big = append(big, bigRow{"skill · " + r.Name, "skill", r.Name, r.Tokens})
	}
	for _, r := range c.Memory {
		big = append(big, bigRow{"memory · " + shorten(r.Name, 36), "memory", r.Name, r.Tokens})
	}
	for _, r := range c.MCPByServer() {
		big = append(big, bigRow{"MCP (deferred) · " + strings.TrimPrefix(r.Name, "claude_ai_"), "mcp", r.Name, r.Tokens})
	}
	sort.SliceStable(big, func(i, j int) bool { return big[i].Tokens > big[j].Tokens })
	if len(big) > 12 {
		big = big[:12]
	}
	return big
}

// ctxItem finds the installed skill or MCP server behind a row of the Context tab.
func (m model) ctxItem(r bigRow) *inventory.Item {
	for i := range m.inv.Items {
		it := &m.inv.Items[i]
		switch {
		case r.kind == "skill" && it.Kind == inventory.Skill && it.Name == r.name:
			return it
		case r.kind == "mcp" && it.Kind == inventory.MCP && (it.Name == r.name || strings.TrimPrefix(r.name, "claude_ai_") == strings.ReplaceAll(it.Name, " ", "_")):
			return it
		}
	}
	return nil
}

// ctxToggle switches the selected consumer off (or on) for this project.
func (m model) ctxToggle() (model, tea.Cmd) {
	if m.ctx.comp == nil {
		return m, nil
	}
	big := ctxBig(m.ctx.comp)
	if m.ctx.sel >= len(big) {
		return m, nil
	}
	r := big[m.ctx.sel]
	if r.kind == "memory" {
		m.status = T("instruction files are not switched off here: edit or move the file")
		return m, nil
	}
	it := m.ctxItem(r)
	if it == nil {
		m.status = T("not found among the installed skills and MCP servers (it may come from a plugin: use /plugin)")
		return m, nil
	}
	nm, cmd := m.toggleItem(it)
	nm.status += " · " + T("r measures again")
	return nm, cmd
}
