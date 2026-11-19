package graph

import (
	"fmt"
	"html"
	"strings"
)

// MermaidVersion is the Mermaid release the web page loads (pinned, from jsDelivr).
const MermaidVersion = "11.4.1"

// MermaidHTML is a self-contained web page that draws the graph with Mermaid in the browser. The diagram
// text is HTML-escaped (labels come from file names and settings); Mermaid runs with securityLevel
// "strict" (no HTML or scripts in labels). Drawing needs the internet once to load Mermaid; the
// diagram text is shown as a fallback when it cannot load.
func (g *Graph) MermaidHTML() string {
	src := html.EscapeString(g.Mermaid(false))
	c := g.Counts()
	var parts []string
	for _, k := range []string{"skill", "agent", "command", "hook", "mcp", "plugin", "tool", "process"} {
		if c[k] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c[k], k))
		}
	}
	summary := html.EscapeString(strings.Join(parts, " · "))
	return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>My Claude rig</title>
<style>
  :root { color-scheme: light dark; --bg: #ffffff; --fg: #1f2328; --muted: #656d76; }
  @media (prefers-color-scheme: dark) { :root { --bg: #0d1117; --fg: #e6edf3; --muted: #8b949e; } }
  body { margin: 0; padding: 16px; background: var(--bg); color: var(--fg); font: 14px/1.5 system-ui, sans-serif; }
  h1 { font-size: 18px; margin: 0 0 4px; }
  p { margin: 0 0 16px; color: var(--muted); }
  .wrap { overflow: auto; }
  pre.mermaid { margin: 0; }
  pre.mermaid:not([data-processed]) { white-space: pre; font: 12px/1.4 ui-monospace, monospace; color: var(--muted); }
</style>
</head>
<body>
<h1>My Claude rig</h1>
<p>` + summary + ` · made with catude · scroll to see all of it</p>
<div class="wrap"><pre class="mermaid">
` + src + `</pre></div>
<script type="module">
  import mermaid from "https://cdn.jsdelivr.net/npm/mermaid@` + MermaidVersion + `/dist/mermaid.esm.min.mjs";
  const dark = window.matchMedia("(prefers-color-scheme: dark)").matches;
  const fit = location.hash === "#fit"; // the whole diagram in the window (screenshots)
  mermaid.initialize({ startOnLoad: true, securityLevel: "strict", maxTextSize: 200000,
    theme: dark ? "dark" : "base",
    themeVariables: dark ? {} : { primaryColor: "#FFF4E0", primaryBorderColor: "#F59E0B", primaryTextColor: "#111111", lineColor: "#6B7280", fontFamily: "system-ui, sans-serif" },
    flowchart: { useMaxWidth: fit, htmlLabels: false, curve: "basis" } });
</script>
</body>
</html>
`
}
