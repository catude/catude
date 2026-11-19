package tui

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/claudecfg"
	"github.com/catude/catude/internal/graph"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/procs"
	"github.com/catude/catude/internal/profiles"
	"github.com/catude/catude/internal/tools"
)

// graphState is the Graph tab: everything this Claude setup uses, as a tree you can open and close,
// with the relations of the selected node (what it uses, what uses it) on the right.
type graphState struct {
	g          *graph.Graph
	running    bool
	at         time.Time
	open       map[string]bool // expanded tree rows (by path key)
	sel        int
	rows       []graphRow // cached flattened tree
	rowsKey    string
	ver        int // bumps on every open/close
	closedRoot bool
}

type graphRow struct {
	key    string // unique position in the tree
	id     string // node id ("" for a group row)
	label  string
	depth  int
	rel    string
	kids   int
	group  bool
	warn   bool
	again  bool   // the node already appears elsewhere (shared)
	prefix string // tree branches drawn before the node: "│  ├─ "
}

// graphRows flattens the graph into the visible tree rows, with the branches of a tree drawing.
var graphRoots = []string{"app:code", "app:desktop", "account:claude.ai"}

func (m *model) graphRows() []graphRow {
	g := m.gr.g
	if g == nil {
		return nil
	}
	key := fmt.Sprint(m.gr.ver, m.gr.at.UnixNano(), m.filter)
	if m.gr.rowsKey == key && m.gr.rows != nil {
		return m.gr.rows
	}
	var rows []graphRow
	shown := map[string]bool{}
	f := strings.ToLower(m.filter)
	if f != "" { // a filter turns the tree into a flat list of matching nodes
		for _, n := range g.Nodes {
			if strings.Contains(strings.ToLower(n.Kind+" "+n.Label+" "+n.Detail), f) {
				rows = append(rows, graphRow{key: "f:" + n.ID, id: n.ID, label: n.Label, warn: n.Warn, kids: len(g.Out(n.ID)), rel: n.Kind})
			}
		}
		m.gr.rows, m.gr.rowsKey = rows, key
		return rows
	}
	type child struct {
		group  bool
		rel    string
		kind   string
		ids    []string
		gkey   string
		nodeID string
	}
	var walk func(id, rel, path string, depth int, prefix, branch string)
	walk = func(id, rel, path string, depth int, prefix, branch string) {
		n, ok := g.Node(id)
		if !ok || depth > 10 {
			return
		}
		k := path + "/" + id
		out := g.Out(id)
		row := graphRow{key: k, id: id, label: n.Label, depth: depth, rel: rel, kids: len(out), warn: n.Warn, again: shown[id], prefix: prefix + branch}
		rows = append(rows, row)
		if row.again && depth > 0 {
			return // already expanded elsewhere: shown once, referenced here
		}
		shown[id] = true
		if !m.gr.open[k] {
			return
		}
		// what the children of this row inherit: a vertical line while this row has siblings below
		next := prefix
		switch branch {
		case "├─ ":
			next += "│  "
		case "└─ ":
			next += "   "
		}
		// children grouped by relation and kind; long groups collapse into one row
		type gk struct{ rel, kind string }
		groups := map[gk][]string{}
		var order []gk
		for _, e := range out {
			to, _ := g.Node(e.To)
			x := gk{e.Rel, to.Kind}
			if _, ok := groups[x]; !ok {
				order = append(order, x)
			}
			groups[x] = append(groups[x], e.To)
		}
		var kids []child
		for _, x := range order {
			ids := groups[x]
			sort.SliceStable(ids, func(i, j int) bool {
				a, _ := g.Node(ids[i])
				b, _ := g.Node(ids[j])
				if a.Warn != b.Warn {
					return a.Warn
				}
				return strings.ToLower(a.Label) < strings.ToLower(b.Label)
			})
			if len(ids) > 6 {
				kids = append(kids, child{group: true, rel: x.rel, kind: x.kind, ids: ids, gkey: k + "/group:" + x.rel + ":" + x.kind})
				continue
			}
			for _, c := range ids {
				kids = append(kids, child{rel: x.rel, nodeID: c})
			}
		}
		for i, c := range kids {
			br := "├─ "
			if i == len(kids)-1 {
				br = "└─ "
			}
			if !c.group {
				walk(c.nodeID, c.rel, k, depth+1, next, br)
				continue
			}
			warn := false
			for _, id := range c.ids {
				if cn, _ := g.Node(id); cn.Warn {
					warn = true
				}
			}
			rows = append(rows, graphRow{key: c.gkey, label: fmt.Sprintf("%d %s", len(c.ids), plural(c.kind)), depth: depth + 1, rel: c.rel, kids: len(c.ids), group: true, warn: warn, prefix: next + br})
			if !m.gr.open[c.gkey] {
				continue
			}
			inner := next + "│  "
			if br == "└─ " {
				inner = next + "   "
			}
			for j, id := range c.ids {
				b2 := "├─ "
				if j == len(c.ids)-1 {
					b2 = "└─ "
				}
				walk(id, "", c.gkey, depth+2, inner, b2)
			}
		}
	}
	for _, r := range graphRoots {
		if _, ok := g.Node(r); ok {
			if m.gr.open == nil {
				m.gr.open = map[string]bool{}
			}
			if _, set := m.gr.open["/"+r]; !set && r == "app:code" && !m.gr.closedRoot {
				m.gr.open["/"+r] = true // the main app starts open
			}
			walk(r, "", "", 0, "", "")
		}
	}
	m.gr.rows, m.gr.rowsKey = rows, key
	return rows
}

func (m model) graphKey(k tea.KeyMsg) (model, tea.Cmd, bool) {
	rows := m.graphRows()
	s := k.String()
	switch s {
	case "up", "k":
		m.gr.sel = clamp(m.gr.sel-1, 0, len(rows)-1)
	case "down", "j":
		m.gr.sel = clamp(m.gr.sel+1, 0, len(rows)-1)
	case "pgup":
		m.gr.sel = clamp(m.gr.sel-10, 0, len(rows)-1)
	case "pgdown":
		m.gr.sel = clamp(m.gr.sel+10, 0, len(rows)-1)
	case "enter", " ":
		if m.gr.sel < len(rows) && rows[m.gr.sel].kids > 0 && m.filter == "" {
			key := rows[m.gr.sel].key
			if m.gr.open[key] {
				delete(m.gr.open, key)
				if key == "/app:code" {
					m.gr.closedRoot = true
				}
			} else {
				m.gr.open[key] = true
			}
			m.gr.ver++
			m.gr.rows = nil
		}
	case "+", "=": // open (arrows switch tabs, as everywhere)
		if m.gr.sel < len(rows) && rows[m.gr.sel].kids > 0 && m.filter == "" && !m.gr.open[rows[m.gr.sel].key] {
			m.gr.open[rows[m.gr.sel].key] = true
			m.gr.ver++
		}
	case "-", "backspace": // close, or jump to the parent
		if m.gr.sel < len(rows) && m.gr.open[rows[m.gr.sel].key] {
			delete(m.gr.open, rows[m.gr.sel].key)
			if rows[m.gr.sel].key == "/app:code" {
				m.gr.closedRoot = true // otherwise the main app opens again by itself
			}
			m.gr.ver++
		} else if m.gr.sel < len(rows) { // jump to the parent row
			d := rows[m.gr.sel].depth
			for i := m.gr.sel - 1; i >= 0; i-- {
				if rows[i].depth < d {
					m.gr.sel = i
					break
				}
			}
		}
	case "E":
		for _, r := range rows {
			if r.kids > 0 {
				m.gr.open[r.key] = true
			}
		}
		m.gr.ver++
	case "C":
		m.gr.open = map[string]bool{}
		m.gr.closedRoot = true
		m.gr.ver++
	case "r":
		m.gr.at, m.gr.rows = time.Time{}, nil
		return m, m.ensureGraph(), true
	case "w":
		if m.gr.g == nil {
			return m, nil, true
		}
		m.status = m.writeGraph()
	default:
		return m, nil, false
	}
	m.graphRows() // recomputed here after a change, so drawing never has to
	return m, nil, true
}

var kindIcon = map[string]string{"app": "◆", "account": "◆", "config": "▣", "memory": "▣", "folder": "▣", "skill": "✦", "agent": "✦", "command": "✦",
	"hook": "▸", "mcp": "⇄", "plugin": "◆", "marketplace": "▣", "tool": "▸", "env": "·", "host": "⇄", "profile": "●", "process": "▸"}

func (m model) graphView(h int) string {
	var b strings.Builder
	state := "built " + ago(m.gr.at) + " · enter open/close · - parent · E/C all · / find · w diagram · r rebuild"
	if m.gr.running {
		state = m.spinner() + " reading files, tools and running processes…"
	}
	b.WriteString(sHeading.Render(T("Everything your Claude uses")) + sMuted.Render("  "+state) + "\n")
	g := m.gr.g
	if g == nil {
		return b.String()
	}
	c := g.Counts()
	var parts []string
	for _, k := range []string{"config", "skill", "agent", "command", "hook", "mcp", "plugin", "tool", "env", "host", "profile", "folder", "process"} {
		if c[k] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c[k], k))
		}
	}
	b.WriteString(sMuted.Render(clipEnd(strings.Join(parts, " · ")+fmt.Sprintf(" · %d relations", len(g.Edges)), m.w-8)) + "\n\n")
	mm := m
	rows := mm.graphRows()
	leftW := clamp(m.w*55/100, 40, 110)
	room := max(3, h-5)
	sel := clamp(m.gr.sel, 0, max(0, len(rows)-1))
	start := clamp(sel-room+3, 0, max(0, len(rows)-room))
	var l strings.Builder
	var lines []string
	for i := start; i < len(rows) && i < start+room; i++ {
		r := rows[i]
		n, _ := g.Node(r.id)
		fold := ""
		if r.kids > 0 && m.filter == "" {
			fold = "+ "
			if mm.gr.open[r.key] {
				fold = "− "
			}
		}
		node := clean(r.label)
		switch {
		case r.group:
			node = sMuted.Render("[") + node + sMuted.Render("]")
		case r.depth == 0 && m.filter == "":
			node = sTitle.Render(kindIcon[n.Kind] + " " + node)
		default:
			node = kindStyle(n.Kind).Render(kindIcon[n.Kind]) + " " + node
		}
		text := sMuted.Render(r.prefix) + sMuted.Render(fold)
		if r.rel != "" {
			text += sMuted.Render(r.rel+" › ") + node
		} else {
			text += node
		}
		if r.again && r.depth > 0 {
			text += sMuted.Render("  ↑ shown above")
		}
		if r.warn {
			text += " " + sWarn.Render(gWarn)
		}
		text = lipgloss.NewStyle().MaxWidth(leftW - 4).Render(text)
		if i == sel {
			text = sSel.Render(gCursor + " " + stripStyles(text, leftW-5))
		} else {
			text = "  " + text
		}
		lines = append(lines, text)
	}
	l.WriteString(withScrollbar(lines, len(rows), start, room, leftW-1) + "\n")
	l.WriteString(position(start, room, len(rows)) + "\n")
	if len(rows) == 0 {
		l.WriteString(sMuted.Render("  nothing matches") + "\n")
	}
	right := ""
	if sel < len(rows) {
		right = m.graphDetail(rows[sel], max(24, m.w-leftW-10))
	}
	return b.String() + lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(leftW).Render(l.String()), right)
}

func (m model) graphDetail(r graphRow, w int) string {
	g := m.gr.g
	wrap := lipgloss.NewStyle().Width(w)
	if r.group {
		return wrap.Render(sHeading.Render(r.label) + "\n\n" + sMuted.Render("enter opens the list"))
	}
	n, ok := g.Node(r.id)
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString(sHeading.Render(clean(n.Label)) + "\n" + sMuted.Render(n.Kind) + "\n")
	if n.Detail != "" {
		st := lipgloss.NewStyle()
		if n.Warn {
			st = sWarn
		}
		b.WriteString("\n" + st.Render(wrap.Render(clean(n.Detail))) + "\n")
	}
	if n.Path != "" {
		b.WriteString(sMuted.Render(wrap.Render(n.Path)) + "\n")
	}
	list := func(title string, es []graph.Edge, out bool) {
		if len(es) == 0 {
			return
		}
		b.WriteString("\n" + sHeading.Render(title) + "\n")
		for i, e := range es {
			if i == 12 {
				b.WriteString(sMuted.Render(fmt.Sprintf("  … %d more", len(es)-12)) + "\n")
				break
			}
			other := e.From
			if out {
				other = e.To
			}
			on, _ := g.Node(other)
			line := "  " + sMuted.Render(e.Rel+" ") + clean(on.Label) + sMuted.Render(" ("+on.Kind+")")
			if on.Warn {
				line += " " + sWarn.Render(gWarn)
			}
			b.WriteString(lipgloss.NewStyle().MaxWidth(w).Render(line) + "\n")
		}
	}
	list("Uses", g.Out(n.ID), true)
	list("Used by", g.In(n.ID), false)
	return b.String()
}

// kindStyle colours the node icon by what it is, so the tree reads at a glance.
func kindStyle(kind string) lipgloss.Style {
	switch kind {
	case "mcp", "host":
		return sTitle
	case "skill", "agent", "command":
		return sOK
	case "tool", "process", "hook":
		return sWarn.UnsetBold()
	case "env":
		return sMuted
	}
	return sHeading
}

// ensureGraph builds the graph in two steps so it appears at once: first files, settings, skills, MCP
// servers and tools (located, not started: milliseconds), then the running processes (about a second).
// A rebuild keeps showing the previous graph until the new one is ready.
func (m *model) ensureGraph() tea.Cmd {
	if m.gr.running || (m.gr.g != nil && time.Since(m.gr.at) < 2*time.Minute) {
		return nil
	}
	m.gr.running = true
	env, inv, ts := m.env, m.inv, m.tools.list
	return func() tea.Msg {
		return graphMsg{g: buildGraph(context.Background(), env, inv, ts, false), partial: true}
	}
}

// graphProcs is the second step: the same graph plus the processes running now.
func (m *model) graphProcs() tea.Cmd {
	env, inv, ts := m.env, m.inv, m.tools.list
	return func() tea.Msg { return graphMsg{g: buildGraph(context.Background(), env, inv, ts, true)} }
}

type graphMsg struct {
	g       graph.Graph
	partial bool // processes not read yet
}

// BuildGraph builds the whole graph (command line and tests).
func BuildGraph(ctx context.Context, env paths.Env, inv inventory.Inventory, withProcs bool) graph.Graph {
	return buildGraph(ctx, env, inv, nil, withProcs)
}

// buildGraph reuses the tools found by the Tools tab (with versions) when there are; otherwise it only
// locates them.
func buildGraph(ctx context.Context, env paths.Env, inv inventory.Inventory, ts []tools.Tool, withProcs bool) graph.Graph {
	if len(ts) == 0 {
		ts = tools.Locate(inv)
	}
	in := graph.Input{Env: env, Inv: inv, Tools: ts, Targets: claudecfg.Targets(env), Folders: map[string]int64{}}
	in.Profiles, _ = profiles.List()
	for _, i := range env.CodeInstances() {
		if i.Version != "" {
			in.Version = i.Version
			break
		}
	}
	// folder sizes come from the last Storage measurement, when there is one (never measured here: slow)
	var c stCached
	if readCache("storage-"+cacheKey(env.ClaudeDir), &c) {
		for _, e := range c.Entries {
			if !e.IsFile && e.Bytes > 0 && e.Group != "catude" {
				in.Folders[e.Path] = e.Bytes
			}
		}
	}
	if withProcs {
		all, err := procs.List(ctx)
		if err != nil || len(all) == 0 {
			all, err = procs.List(ctx) // the first query sometimes fails right after start on Windows
		}
		if err == nil {
			in.Procs = procs.ClaudeTree(all, os.Getpid())
		}
	}
	return graph.Build(in)
}
