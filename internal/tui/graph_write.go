package tui

import (
	"os"
	"path/filepath"
)

// writeGraph saves the graph as Mermaid — a Markdown file (GitHub draws it) and a web page that draws it
// in the browser — and opens the page. Only labels go into either file: no paths or command lines.
func (m model) writeGraph() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "graph not written: " + err.Error()
	}
	md := filepath.Join(home, "catude-graph.md")
	page := filepath.Join(home, "catude-graph.html")
	if err := os.WriteFile(md, []byte("# My Claude rig\n\n```mermaid\n"+m.gr.g.Mermaid(false)+"```\n"), 0o644); err != nil {
		return "graph not written: " + err.Error()
	}
	if err := os.WriteFile(page, []byte(m.gr.g.MermaidHTML()), 0o644); err != nil {
		return "graph not written: " + err.Error()
	}
	if err := openFolder(page); err != nil {
		return gOK + " diagram saved: open " + page + " in a browser · Markdown: " + md
	}
	return gOK + " opening the diagram in your browser (" + page + ") · Markdown for GitHub: " + md
}
