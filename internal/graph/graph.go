// Package graph builds one picture of everything a Claude setup uses: the apps, the files each one
// reads, what those files define (hooks, MCP servers, plugins), what each of those runs (programs,
// environment variables, remote hosts), the skills, agents and commands, work profiles, folders on
// disk and the processes running right now — and how they depend on each other.
//
// The graph never holds secret values: MCP servers contribute the NAMES of their variables, command
// lines of processes arrive already redacted, and exports (Mermaid, DOT) carry labels only.
package graph

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/catude/catude/internal/claudecfg"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/procs"
	"github.com/catude/catude/internal/profiles"
	"github.com/catude/catude/internal/redact"
	"github.com/catude/catude/internal/tools"
)

// Node is one thing in the setup.
type Node struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"` // app account config memory folder skill agent command hook mcp plugin marketplace tool env host profile process
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
	Path   string `json:"path,omitempty"`
	Warn   bool   `json:"warn,omitempty"` // something to look at (missing tool, problem, secret variable in plain text)
}

// Edge is a relation: From —Rel→ To.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Rel  string `json:"rel"`
}

// Graph is the whole picture.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
	idx   map[string]int
	seen  map[string]bool
}

// Input is everything the graph is built from (all already loaded by the caller).
type Input struct {
	Env      paths.Env
	Version  string // Claude Code version
	Inv      inventory.Inventory
	Tools    []tools.Tool
	Profiles []profiles.Profile
	Targets  []claudecfg.Target
	Procs    []procs.Proc     // already filtered to Claude's tree
	Folders  map[string]int64 // folder path → bytes, when measured
}

func (g *Graph) add(n Node) string {
	if g.idx == nil {
		g.idx, g.seen = map[string]int{}, map[string]bool{}
	}
	if i, ok := g.idx[n.ID]; ok {
		if n.Warn {
			g.Nodes[i].Warn = true
		}
		return n.ID
	}
	g.idx[n.ID] = len(g.Nodes)
	g.Nodes = append(g.Nodes, n)
	return n.ID
}

func (g *Graph) link(from, to, rel string) {
	k := from + "\x00" + to + "\x00" + rel
	if from == "" || to == "" || from == to || g.seen[k] {
		return
	}
	g.seen[k] = true
	g.Edges = append(g.Edges, Edge{from, to, rel})
}

// Node returns a node by id.
func (g *Graph) Node(id string) (Node, bool) {
	i, ok := g.idx[id]
	if !ok {
		return Node{}, false
	}
	return g.Nodes[i], true
}

// Out and In list the edges leaving and entering a node, in a stable order.
func (g *Graph) Out(id string) []Edge { return g.edges(id, true) }
func (g *Graph) In(id string) []Edge  { return g.edges(id, false) }

func (g *Graph) edges(id string, out bool) []Edge {
	var es []Edge
	for _, e := range g.Edges {
		if (out && e.From == id) || (!out && e.To == id) {
			es = append(es, e)
		}
	}
	return es
}

func norm(p string) string { return strings.ToLower(filepath.Clean(p)) }

// Build assembles the graph.
func Build(in Input) Graph {
	var g Graph
	codeDetail := ""
	if in.Version != "" {
		codeDetail = "version " + in.Version
	}
	code := g.add(Node{ID: "app:code", Kind: "app", Label: "Claude Code", Detail: codeDetail})
	desktop := ""
	cfgByPath := map[string]string{}
	for _, t := range in.Targets {
		if !t.Exists {
			continue
		}
		id := g.add(Node{ID: "config:" + norm(t.Path), Kind: "config", Label: t.Label, Detail: t.Note, Path: t.Path})
		cfgByPath[norm(t.Path)] = id
		if t.App == "Claude Desktop" {
			if desktop == "" {
				desktop = g.add(Node{ID: "app:desktop", Kind: "app", Label: "Claude Desktop", Detail: "its Code tab also uses Claude Code's files"})
				g.link(desktop, code, "Code tab uses")
			}
			g.link(desktop, id, "reads")
		} else {
			g.link(code, id, "reads")
		}
	}
	toolByBin := map[string]string{}
	for _, t := range in.Tools {
		if !t.Found && len(t.UsedBy) == 0 {
			continue
		}
		detail := t.Purpose
		if t.Version != "" {
			detail = t.Version + " · " + detail
		}
		if !t.Found {
			detail = "NOT INSTALLED · " + detail
		}
		id := g.add(Node{ID: "tool:" + t.Name, Kind: "tool", Label: t.Name, Detail: detail, Path: t.Path, Warn: !t.Found && len(t.UsedBy) > 0})
		for _, b := range t.Bins {
			toolByBin[b] = id
		}
	}
	account := ""
	acct := func() string {
		if account == "" {
			account = g.add(Node{ID: "account:claude.ai", Kind: "account", Label: "claude.ai account", Detail: "connectors, skills and plugins that reach every Claude app through your login"})
			g.link(code, account, "signed in to")
			if desktop != "" {
				g.link(desktop, account, "signed in to")
			}
		}
		return account
	}
	owner := func(it inventory.Item) string { // where an item comes from
		switch {
		case it.Scope == paths.ScopeClaudeAI || it.Scope == paths.ScopeSynced:
			return acct()
		case it.Scope == paths.ScopePlugin && it.Origin != "":
			return g.add(Node{ID: "plugin:" + strings.ToLower(it.Origin), Kind: "plugin", Label: it.Origin})
		}
		if id, ok := cfgByPath[norm(it.Path)]; ok && it.Path != "" {
			return id
		}
		return code
	}
	names := map[string]string{} // "kind:name" → node id, for profiles
	seenItem := map[string]bool{}
	for _, it := range in.Inv.Items {
		var kind, rel string
		switch it.Kind {
		case inventory.Skill:
			kind, rel = "skill", "has skill"
		case inventory.Agent:
			kind, rel = "agent", "has agent"
		case inventory.Command:
			kind, rel = "command", "has command"
		case inventory.Hook:
			kind, rel = "hook", "defines hook"
		case inventory.MCP:
			kind, rel = "mcp", "defines MCP"
		case inventory.Plugin:
			kind, rel = "plugin", "enables plugin"
		case inventory.Marketplace:
			kind, rel = "marketplace", "knows marketplace"
		case inventory.Memory:
			kind, rel = "memory", "loads instructions"
		default:
			continue
		}
		id := kind + ":" + strings.ToLower(it.Name)
		if kind == "hook" || kind == "memory" {
			id += "@" + norm(it.Path)
		}
		if kind == "memory" && it.Content == "" {
			continue // the file does not exist
		}
		n := Node{ID: id, Kind: kind, Label: it.Name, Detail: clip(it.Description, 160), Path: it.Path, Warn: len(it.Problems) > 0}
		if kind == "plugin" && !it.Enabled {
			n.Detail = strings.TrimSpace("disabled · " + n.Detail)
		}
		if it.Status != "" {
			n.Detail = strings.TrimSpace(it.Status + " · " + n.Detail)
			n.Warn = n.Warn || it.Status == "failed" || it.Status == "needs-auth"
		}
		g.add(n)
		names[kind+":"+strings.ToLower(it.Name)] = id
		from := owner(it)
		if kind == "plugin" || kind == "marketplace" {
			from = code
		}
		if kind == "mcp" && seenItem[id] { // the same server in another file: one more "defines" edge
			g.link(from, id, rel)
			continue
		}
		seenItem[id] = true
		g.link(from, id, rel)
		switch kind {
		case "mcp":
			linkMCP(&g, id, it, toolByBin)
		case "hook":
			for _, w := range strings.Fields(it.Content) {
				if t, ok := toolByBin[binName(strings.Trim(w, `"'`))]; ok {
					g.link(id, t, "runs")
					break
				}
			}
		}
	}
	// plugins come from marketplaces: "name@marketplace"
	for _, n := range g.Nodes {
		if n.Kind != "plugin" {
			continue
		}
		if i := strings.LastIndex(n.Label, "@"); i > 0 {
			if m, ok := names["marketplace:"+strings.ToLower(n.Label[i+1:])]; ok {
				g.link(m, n.ID, "offers")
			}
		}
	}
	for _, p := range in.Profiles {
		id := g.add(Node{ID: "profile:" + strings.ToLower(p.Name), Kind: "profile", Label: p.Name, Detail: p.Summary(), Path: p.Dir})
		g.link(code, id, "can start with profile")
		for kind, list := range map[string][]string{"skill": p.Skills, "agent": p.Agents, "command": p.Commands, "mcp": p.MCP, "plugin": p.Plugins} {
			for _, n := range list {
				to, ok := names[kind+":"+strings.ToLower(n)]
				if !ok {
					to = g.add(Node{ID: kind + ":" + strings.ToLower(n) + "@profile:" + strings.ToLower(p.Name), Kind: kind, Label: n, Detail: "only in this profile"})
				}
				g.link(id, to, "includes")
			}
		}
	}
	var folders []string
	for p := range in.Folders {
		folders = append(folders, p)
	}
	sort.Strings(folders)
	for _, p := range folders {
		id := g.add(Node{ID: "folder:" + norm(p), Kind: "folder", Label: filepath.Base(p), Detail: humanBytes(in.Folders[p]), Path: p})
		g.link(code, id, "stores")
	}
	pidNode := map[int]string{}
	for _, p := range in.Procs {
		id := g.add(Node{ID: fmt.Sprintf("process:%d", p.PID), Kind: "process", Label: fmt.Sprintf("%s (pid %d)", p.Name, p.PID), Detail: p.Cmd, Path: p.Cwd})
		pidNode[p.PID] = id
		if p.Root {
			from := code
			if desktop != "" && strings.Contains(strings.ToLower(p.Cmd+p.Name), "anthropicclaude") || strings.EqualFold(strings.TrimSuffix(p.Name, ".exe"), "claude-desktop") {
				from = firstNonEmpty(desktop, code)
			}
			g.link(from, id, "running now")
		} else if parent, ok := pidNode[p.PPID]; ok {
			g.link(parent, id, "started")
		}
		if t, ok := toolByBin[binName(p.Name)]; ok && !p.Root {
			g.link(id, t, "is")
		}
	}
	return g
}

func linkMCP(g *Graph, id string, it inventory.Item, toolByBin map[string]string) {
	var s struct {
		Command string            `json:"command"`
		Env     map[string]string `json:"env"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	if json.Unmarshal([]byte(it.Content), &s) != nil {
		return
	}
	if s.Command != "" {
		if t, ok := toolByBin[binName(s.Command)]; ok {
			g.link(id, t, "starts")
		} else {
			g.link(id, g.add(Node{ID: "tool:" + binName(s.Command), Kind: "tool", Label: binName(s.Command), Detail: "program started by an MCP server", Warn: !tools.Resolvable(s.Command)}), "starts")
		}
	}
	if s.URL != "" {
		if u, err := url.Parse(s.URL); err == nil && u.Host != "" {
			g.link(id, g.add(Node{ID: "host:" + strings.ToLower(u.Host), Kind: "host", Label: u.Host, Detail: "remote MCP server", Warn: u.Scheme == "http"}), "connects to")
		}
	}
	vars := map[string]string{}
	for k, v := range s.Env {
		vars[k] = v
	}
	for k, v := range s.Headers {
		vars[k] = v
	}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := vars[k]
		secretPlain := v != "" && !redact.IsEnvRef(v) && (redact.KeyIsSecret(k) || redact.LooksSecret(v))
		detail := "variable or header of an MCP server (value not shown)"
		if secretPlain {
			detail = "SECRET WRITTEN IN PLAIN TEXT in the config file · " + detail
		}
		g.link(id, g.add(Node{ID: "env:" + k, Kind: "env", Label: k, Detail: detail, Warn: secretPlain}), "needs")
	}
}

func binName(c string) string {
	b := strings.ToLower(filepath.Base(strings.ReplaceAll(c, `\`, "/")))
	for _, ext := range []string{".exe", ".cmd", ".bat", ".ps1"} {
		b = strings.TrimSuffix(b, ext)
	}
	return b
}

func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// Counts returns how many nodes there are of each kind.
func (g *Graph) Counts() map[string]int {
	out := map[string]int{}
	for _, n := range g.Nodes {
		out[n.Kind]++
	}
	return out
}

// ---- exports (labels only: never paths of your projects, never command lines) ----

func safeLabel(s string) string {
	s = strings.NewReplacer(`"`, "'", "\n", " ", "[", "(", "]", ")", "{", "(", "}", ")", "|", "/", "<", "‹", ">", "›", "`", "'").Replace(s)
	return clip(s, 60)
}

// Mermaid renders the graph as a Mermaid flowchart (GitHub, GitLab and most Markdown viewers draw it).
// Leaves that are numerous (skills, connectors…) are grouped per owner to keep the picture readable
// unless full is set.
func (g *Graph) Mermaid(full bool) string {
	var b strings.Builder
	b.WriteString("flowchart LR\n")
	ids := map[string]string{}
	id := func(n string) string {
		if v, ok := ids[n]; ok {
			return v
		}
		ids[n] = fmt.Sprintf("n%d", len(ids))
		return ids[n]
	}
	shape := func(n Node) string {
		l := safeLabel(n.Label)
		switch n.Kind {
		case "app", "account":
			return `(["` + l + `"])`
		case "config", "memory", "folder":
			return `[/"` + l + `"/]`
		case "tool", "process":
			return `[["` + l + `"]]`
		case "env", "host":
			return `{{"` + l + `"}}`
		}
		return `["` + l + `"]`
	}
	grouped := map[string]bool{}
	if !full {
		// group leaves: owner + rel + kind with more than 4 leaves becomes one node "N skills"
		type key struct{ from, rel, kind string }
		count := map[key][]string{}
		for _, e := range g.Edges {
			to, _ := g.Node(e.To)
			// leaves are grouped even when a profile also points at them: the summary stays readable
			if len(g.Out(e.To)) == 0 && !strings.HasPrefix(e.From, "profile:") {
				k := key{e.From, e.Rel, to.Kind}
				count[k] = append(count[k], e.To)
			}
		}
		keys := make([]key, 0, len(count))
		for k := range count {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].from+keys[i].rel+keys[i].kind < keys[j].from+keys[j].rel+keys[j].kind
		})
		for _, k := range keys {
			if len(count[k]) <= 3 {
				continue
			}
			for _, n := range count[k] {
				grouped[n] = true
			}
			gid := id("group:" + k.from + k.rel + k.kind)
			fmt.Fprintf(&b, "  %s[\"%d %ss\"]\n  %s -->|\"%s\"| %s\n", gid, len(count[k]), k.kind, id(k.from), safeLabel(k.rel), gid)
		}
	}
	for _, n := range g.Nodes {
		if grouped[n.ID] {
			continue
		}
		fmt.Fprintf(&b, "  %s%s\n", id(n.ID), shape(n))
		if n.Warn {
			fmt.Fprintf(&b, "  style %s stroke:#EF4444,stroke-width:2px\n", id(n.ID))
		}
	}
	for _, e := range g.Edges {
		if grouped[e.To] || grouped[e.From] {
			continue
		}
		fmt.Fprintf(&b, "  %s -->|\"%s\"| %s\n", id(e.From), safeLabel(e.Rel), id(e.To))
	}
	return b.String()
}

// DOT renders the graph for Graphviz.
func (g *Graph) DOT() string {
	var b strings.Builder
	b.WriteString("digraph catude {\n  rankdir=LR;\n  node [shape=box, fontname=\"monospace\"];\n")
	q := func(s string) string {
		return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
	}
	for _, n := range g.Nodes {
		attr := ""
		if n.Warn {
			attr = ", color=red"
		}
		fmt.Fprintf(&b, "  %s [label=%s%s];\n", q(n.ID), q(n.Kind+": "+clip(n.Label, 60)), attr)
	}
	for _, e := range g.Edges {
		fmt.Fprintf(&b, "  %s -> %s [label=%s];\n", q(e.From), q(e.To), q(e.Rel))
	}
	b.WriteString("}\n")
	return b.String()
}
