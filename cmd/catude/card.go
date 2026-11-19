package main

import (
	"flag"
	"fmt"
	"html"
	"io"
	"os"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/catude/catude/internal/audit"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/profiles"
	"github.com/catude/catude/internal/tui"
)

// The card is a small summary of a setup made to be shared (a screenshot, a README, a post).
// It contains numbers only — never names of skills, servers, projects or people, and never values —
// so it is safe to publish as it is.

type cardData struct {
	lines []string
}

func buildCard(env paths.Env) cardData {
	inv := inventory.Load(env)
	high := 0
	fs := audit.Run(inv)
	for _, f := range fs {
		if f.Severity == audit.High {
			high++
		}
	}
	ver := "not found"
	for _, in := range env.CodeInstances() {
		if in.Version != "" {
			ver = in.Version
			break
		}
	}
	desktop := 0
	for _, f := range env.DesktopConfigs() {
		if f.Exists {
			desktop++
		}
	}
	ownMCP, manual := 0, 0
	seenMCP := map[string]bool{} // the same server copied to several Claude Desktop files counts once
	for _, it := range inv.Of(inventory.MCP) {
		if it.Path != "" && !seenMCP[strings.ToLower(it.Name)] {
			seenMCP[strings.ToLower(it.Name)] = true
			ownMCP++
		}
	}
	for _, it := range inv.Of(inventory.Skill) {
		if !it.Invocable {
			manual++
		}
	}
	ps, _ := profiles.List()
	osName := map[string]string{"windows": "Windows", "darwin": "macOS", "linux": "Linux"}[runtime.GOOS]
	c := cardData{}
	add := func(format string, a ...any) { c.lines = append(c.lines, fmt.Sprintf(format, a...)) }
	add("Claude Code %s · %s%s", ver, osName, map[bool]string{true: " · Claude Desktop", false: ""}[desktop > 0])
	add("")
	add("%-4d skills        %-4d agents      %-4d commands", inv.Count(inventory.Skill), inv.Count(inventory.Agent), inv.Count(inventory.Command))
	add("%-4d MCP servers   %-4d plugins     %-4d hooks", ownMCP, inv.Count(inventory.Plugin), inv.Count(inventory.Hook))
	add("%-4d work profiles %-4d marketplaces", len(ps), inv.Count(inventory.Marketplace))
	add("")
	if used, max, _, ok := tui.CachedContext(env); ok && max > 0 {
		pct := float64(used) / float64(max)
		n := int(pct*24 + 0.5)
		if n < 1 && used > 0 {
			n = 1
		}
		add("context at start  %s of %s  %s%s %.1f%%", human(used), human(max), strings.Repeat("█", n), strings.Repeat("░", 24-n), 100*pct)
	} else {
		add("context at start  not measured yet (open catude → Context)")
	}
	if high == 0 {
		add("security audit    ✓ no high findings (%d notes)", len(fs))
	} else {
		add("security audit    ▲ %d to fix (%d findings)", high, len(fs))
	}
	if manual > 0 {
		add("%d skills load only on demand (0 tokens until used)", manual)
	}
	return c
}

func cardText(c cardData) string {
	w := 0
	for _, l := range c.lines {
		w = max(w, utf8.RuneCountInString(l))
	}
	title := " ≡ catude "
	var b strings.Builder
	b.WriteString("╭─" + title + strings.Repeat("─", w+2-utf8.RuneCountInString(title)-1) + "╮\n")
	for _, l := range c.lines {
		b.WriteString("│ " + l + strings.Repeat(" ", w-utf8.RuneCountInString(l)) + " │\n")
	}
	foot := " my Claude rig · numbers only, no names "
	b.WriteString("╰" + strings.Repeat("─", max(0, w+2-utf8.RuneCountInString(foot))) + foot + "╯\n")
	return b.String()
}

func cardSVG(c cardData) string {
	const cw, lh, pad = 8.6, 20, 22
	w := 0
	for _, l := range c.lines {
		w = max(w, utf8.RuneCountInString(l))
	}
	width := int(float64(w)*cw) + pad*2
	height := (len(c.lines)+3)*lh + pad
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="catude setup card">`+"\n", width, height, width, height)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" rx="12" fill="#111827" stroke="#374151"/>`+"\n", width, height)
	b.WriteString(`<g transform="translate(22 16) scale(0.4)"><rect x="2" y="2" width="60" height="60" rx="14" fill="#1F2937"/><rect x="12" y="15" width="40" height="8" rx="4" fill="#F59E0B"/><rect x="12" y="28" width="30" height="8" rx="4" fill="#F59E0B" fill-opacity="0.75"/><rect x="12" y="41" width="20" height="8" rx="4" fill="#F59E0B" fill-opacity="0.5"/><circle cx="47" cy="45" r="5" fill="#22C55E"/></g>` + "\n")
	font := `font-family="ui-monospace, SFMono-Regular, Menlo, Consolas, 'Liberation Mono', monospace"`
	fmt.Fprintf(&b, `<text x="56" y="35" %s font-size="16" font-weight="700" fill="#F9FAFB">cat<tspan fill="#F59E0B">ude</tspan></text>`+"\n", font)
	for i, l := range c.lines {
		fill := "#E5E7EB"
		if i == 0 {
			fill = "#9CA3AF"
		}
		fmt.Fprintf(&b, `<text x="%d" y="%d" %s font-size="14" fill="%s" xml:space="preserve">%s</text>`+"\n", pad, 66+i*lh, font, fill, html.EscapeString(l))
	}
	fmt.Fprintf(&b, `<text x="%d" y="%d" %s font-size="11" fill="#6B7280">my Claude rig · numbers only, no names</text>`+"\n", pad, height-12, font)
	b.WriteString("</svg>\n")
	return b.String()
}

func cardCmd(env paths.Env, args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("card", flag.ContinueOnError)
	fs.SetOutput(errw)
	svg := fs.String("svg", "", "also write the card as an SVG image to this file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	c := buildCard(env)
	fmt.Fprint(out, cardText(c))
	if *svg != "" {
		if err := os.WriteFile(*svg, []byte(cardSVG(c)), 0o644); err != nil {
			fmt.Fprintln(errw, "catude:", err)
			return 1
		}
		fmt.Fprintln(out, "saved "+*svg)
	}
	return 0
}
