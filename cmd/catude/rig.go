package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/catude/catude/internal/catalog"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/rigcard"
)

const rigUsage = `catude rig: share and load Claude setups (Rigcard 0.1, docs/rigcard.md)

  catude rig export [-o rigcard.json] [--summary TEXT] [--owner NAME] [--id URL]
        describe your setup: marketplaces, plugins, skills/agents/commands (sha256), MCP servers
        (variable NAMES only, never values), safe settings. Refuses to write if a secret slipped in.
  catude rig show SOURCE [--json]
        load someone's card and compare it with this machine. Nothing is installed: you get the
        official commands to run. SOURCE: ./file.json · host/path · host · gh:owner/repo[@ref]
`

func rigCmd(env paths.Env, args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(errw, rigUsage)
		return 2
	}
	sub, args := args[0], args[1:]
	fs := flag.NewFlagSet("rig", flag.ContinueOnError)
	fs.SetOutput(errw)
	fs.Usage = func() { fmt.Fprint(errw, rigUsage) }
	output := fs.String("o", "", "write to this file instead of stdout")
	summary := fs.String("summary", "", "one line about this setup")
	owner := fs.String("owner", "", "your name or handle")
	id := fs.String("id", "", "canonical URL where the card will be published")
	asJSON := fs.Bool("json", false, "machine-readable output")
	var pos []string
	for len(args) > 0 {
		if err := fs.Parse(args); err != nil {
			return 2
		}
		args = fs.Args()
		if len(args) > 0 {
			pos, args = append(pos, args[0]), args[1:]
		}
	}
	switch sub {
	case "export":
		return rigExport(env, *output, rigcard.ExportOptions{Summary: *summary, Owner: rigcard.Owner{Name: *owner}, ID: *id}, out, errw)
	case "show":
		if len(pos) != 1 {
			fmt.Fprint(errw, rigUsage)
			return 2
		}
		return rigShow(env, pos[0], *asJSON, out, errw)
	case "help", "-h", "--help":
		fmt.Fprint(out, rigUsage)
		return 0
	}
	fmt.Fprintf(errw, "catude rig: unknown subcommand %q\n\n%s", sub, rigUsage)
	return 2
}

func rigExport(env paths.Env, output string, o rigcard.ExportOptions, out, errw io.Writer) int {
	c, err := rigcard.Export(inventory.Load(env), o)
	if err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	b = append(b, '\n')
	if ps := rigcard.Validate(c, b); rigcard.Rejected(ps) {
		for _, p := range ps {
			fmt.Fprintf(errw, "%s: %s: %s\n", p.Severity, p.Where, p.Msg)
		}
		fmt.Fprintln(errw, "catude: the card did not pass validation; nothing was written")
		return 1
	}
	if output == "" {
		out.Write(b)
		return 0
	}
	if err := os.WriteFile(output, b, 0o644); err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	fmt.Fprintf(errw, "wrote %s · %d plugins, %d skills, %d agents, %d commands, %d MCP servers\nreview it before publishing: names of private MCP servers and projects may still be sensitive\n",
		output, len(c.Plugins), len(c.Skills), len(c.Agents), len(c.Commands), len(c.MCPServers))
	return 0
}

func rigShow(env paths.Env, src string, asJSON bool, out, errw io.Writer) int {
	c, raw, from, err := rigcard.Load(context.Background(), src)
	if err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	problems := rigcard.Validate(c, raw)
	var steps []rigcard.Step
	if !rigcard.Rejected(problems) {
		steps = rigcard.Plan(c, inventory.Load(env))
	}
	if asJSON {
		return writeJSON(out, map[string]any{"source": from, "card": c, "problems": problems, "plan": steps})
	}
	who := "unknown author"
	if c.Owner != nil && (c.Owner.Name != "" || c.Owner.Handle != "") {
		who = strings.TrimSpace(c.Owner.Name + " " + c.Owner.Handle)
	}
	fmt.Fprintf(out, "Rigcard %s · %s\n%s\n", c.Rigcard, who, from)
	if c.Owner != nil && c.Owner.Art != "" && !rigcard.Rejected(problems) {
		fmt.Fprintln(out, strings.TrimRight(c.Owner.Art, "\n")) // the owner's own ASCII art (validated: size, no control characters)
	}
	if c.Summary != "" {
		fmt.Fprintln(out, clip(c.Summary, 200))
	}
	fmt.Fprintln(out)
	for _, p := range problems {
		fmt.Fprintf(out, "%-6s %s: %s\n", strings.ToUpper(p.Severity), p.Where, p.Msg)
	}
	if rigcard.Rejected(problems) {
		fmt.Fprintln(out, "\nREJECTED: this card must not be used.")
		return 1
	}
	if len(problems) > 0 {
		fmt.Fprintln(out)
	}
	tw := tabwriter.NewWriter(out, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "STATE\tKIND\tNAME\tDETAIL")
	counts := map[string]int{}
	for _, s := range steps {
		counts[s.State]++
		fmt.Fprintf(tw, "%s %s\t%s\t%s\t%s\n", stateMark(s.State), s.State, s.Kind, clip(s.Name, 48), clip(s.Detail, 70))
	}
	tw.Flush()
	var cmds []string
	for _, s := range steps {
		if s.Cmd != "" && s.State != "have" {
			cmds = append(cmds, s.Cmd)
		}
	}
	fmt.Fprintf(out, "\n%d have · %d missing · %d different · %d to review\n", counts["have"], counts["missing"], counts["different"], counts["review"])
	if len(cmds) > 0 {
		fmt.Fprintln(out, "\nTo adopt what you are missing, review and run (catude never runs them for you here):")
		for _, c := range cmds {
			fmt.Fprintln(out, "  "+c)
		}
	}
	return 0
}

func stateMark(s string) string {
	switch s {
	case "have":
		return "✓"
	case "missing":
		return "+"
	case "different":
		return "~"
	}
	return "!"
}

func discoverCmd(env paths.Env, section, query string, asJSON bool, out, errw io.Writer) int {
	cat, err := catalog.Load()
	if err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	section = strings.TrimSuffix(strings.ToLower(section), "s")
	if section == "all" {
		section = ""
	}
	inv := inventory.Load(env)
	installed := map[string]bool{}
	for _, it := range inv.Items {
		installed[strings.ToLower(it.Name)] = true
	}
	var rows []catalog.Entry
	for _, e := range cat.Entries() {
		if section != "" && e.Section != section {
			continue
		}
		if q := strings.ToLower(query); q != "" && !strings.Contains(strings.ToLower(e.Name+" "+e.Description+" "+e.Category), q) {
			continue
		}
		rows = append(rows, e)
	}
	if asJSON {
		return writeJSON(out, rows)
	}
	if len(rows) == 0 {
		fmt.Fprintln(errw, "nothing matches; sections: marketplaces plugins mcp skills tools practices")
		return 0
	}
	tw := tabwriter.NewWriter(out, 2, 4, 2, ' ', 0)
	for _, e := range rows {
		mark := " "
		if e.Key != "" && installed[strings.ToLower(e.Key)] {
			mark = "✓"
		}
		badge := ""
		if e.Official {
			badge = "official"
		}
		if e.Stars > 0 {
			badge = strings.TrimSpace(badge + " ★" + human(e.Stars))
		}
		fmt.Fprintf(tw, "%s %s\t%s\t%s\t%s\n", mark, e.Section, clip(e.Name, 40), badge, clip(e.Description, 70))
		if e.Command != "" {
			fmt.Fprintf(tw, "\t\t\t\t  $ %s\n", e.Command)
		}
	}
	tw.Flush()
	fmt.Fprintf(out, "\n%d entries · catalog of %s · ✓ = already on this machine\n", len(rows), cat.Generated)
	return 0
}
