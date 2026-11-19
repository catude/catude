package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/tui"
)

// graphCmd prints everything the setup uses: as an indented outline (default), a Mermaid diagram,
// Graphviz DOT, or JSON. Mermaid and DOT carry labels only (no paths, no command lines).
func graphCmd(env paths.Env, pos []string, asJSON bool, out, errw io.Writer) int {
	g := tui.BuildGraph(context.Background(), env, inventory.Load(env), true)
	format := ""
	if len(pos) > 0 {
		format = pos[0]
	}
	switch {
	case asJSON:
		return writeJSON(out, g)
	case format == "mermaid":
		fmt.Fprint(out, g.Mermaid(false))
	case format == "mermaid-full":
		fmt.Fprint(out, g.Mermaid(true))
	case format == "html":
		fmt.Fprint(out, g.MermaidHTML())
	case format == "dot":
		fmt.Fprint(out, g.DOT())
	case format == "":
		c := g.Counts()
		kinds := make([]string, 0, len(c))
		for k := range c {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
		var parts []string
		for _, k := range kinds {
			parts = append(parts, fmt.Sprintf("%d %s", c[k], k))
		}
		fmt.Fprintf(out, "%d nodes, %d relations: %s\n\n", len(g.Nodes), len(g.Edges), strings.Join(parts, " · "))
		seen := map[string]bool{}
		var walk func(id, rel string, depth int)
		walk = func(id, rel string, depth int) {
			n, ok := g.Node(id)
			if !ok || depth > 6 {
				return
			}
			mark := ""
			if n.Warn {
				mark = "  ▲ " + n.Detail
			}
			again := ""
			if seen[id] {
				again = "  (shown above)"
			}
			fmt.Fprintf(out, "%s%s%s: %s%s%s\n", strings.Repeat("  ", depth), map[bool]string{true: rel + " → ", false: ""}[rel != ""], n.Kind, n.Label, again, clip(mark, 110))
			if seen[id] {
				return
			}
			seen[id] = true
			for _, e := range g.Out(id) {
				walk(e.To, e.Rel, depth+1)
			}
		}
		for _, root := range []string{"app:code", "app:desktop", "account:claude.ai"} {
			walk(root, "", 0)
		}
	default:
		fmt.Fprintln(errw, "catude graph: formats are mermaid, mermaid-full, dot (or --json)")
		return 2
	}
	return 0
}
