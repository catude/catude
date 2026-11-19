package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/catude/catude/internal/agentmap"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
)

// agentsCmd prints the agent map: which subagents exist here, how often each ran, who launched it, which
// tools it used and what it spent.
func agentsCmd(env paths.Env, days int, asJSON bool, out io.Writer) int {
	m := agentmap.Build(env.ClaudeDir, inventory.Load(env), days)
	if asJSON {
		type row struct {
			Name       string         `json:"name"`
			Scope      string         `json:"scope,omitempty"`
			Path       string         `json:"path,omitempty"`
			Model      string         `json:"definedModel,omitempty"`
			BuiltIn    bool           `json:"builtIn"`
			Runs       int            `json:"runs"`
			Tokens     agentmap.Usage `json:"tokens"`
			Tools      map[string]int `json:"tools"`
			LaunchedBy map[string]int `json:"launchedBy"`
			Models     map[string]int `json:"models"`
			Stopped    int            `json:"stoppedByUser"`
			LastRun    *time.Time     `json:"lastRun,omitempty"`
			AvgSeconds float64        `json:"avgSeconds,omitempty"`
		}
		var rows []row
		for _, a := range m.Agents {
			r := row{Name: a.Name, BuiltIn: a.BuiltIn, Runs: len(a.Runs), Tokens: a.Usage, Tools: a.Tools, LaunchedBy: a.LaunchedBy,
				Models: a.Models, Stopped: a.Stopped, AvgSeconds: a.Avg.Seconds()}
			if a.Defined != nil {
				r.Scope, r.Path = string(a.Defined.Scope), a.Defined.Path
				r.Model = definedModel(a.Defined)
			}
			if !a.LastRun.IsZero() {
				t := a.LastRun
				r.LastRun = &t
			}
			rows = append(rows, r)
		}
		e := json.NewEncoder(out)
		e.SetIndent("", "  ")
		e.Encode(map[string]any{"days": m.Days, "runs": m.Runs, "tokens": m.Usage, "agents": rows})
		return 0
	}
	fmt.Fprintf(out, "Agent map · last %d days · %d subagent runs · %s tokens\n\n", m.Days, m.Runs, human(m.Usage.Total()))
	tw := tabwriter.NewWriter(out, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "AGENT\tFROM\tRUNS\tTOKENS\tLAUNCHED BY\tTOP TOOLS\tLAST")
	for _, a := range m.Agents {
		from := "built in"
		if a.Defined != nil {
			from = string(a.Defined.Scope)
		} else if !a.BuiltIn {
			from = "not defined now"
		}
		last := "never"
		if !a.LastRun.IsZero() {
			last = a.LastRun.Local().Format("02 Jan 15:04")
		}
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\t%s\n", a.Name, from, len(a.Runs), human(a.Usage.Total()),
			strings.Join(agentmap.Top(a.LaunchedBy, 2), ", "), strings.Join(agentmap.Top(a.Tools, 3), ", "), last)
	}
	tw.Flush()
	if u := m.Unused(); len(u) > 0 {
		var names []string
		for _, a := range u {
			names = append(names, a.Name)
		}
		fmt.Fprintf(out, "\nDefined but not used in %d days: %s. Their descriptions are loaded into every session; remove the ones you do not need.\n", m.Days, strings.Join(names, ", "))
	}
	return 0
}

func definedModel(it *inventory.Item) string {
	for _, d := range it.Detail {
		if strings.HasPrefix(d, "model: ") {
			return strings.TrimPrefix(d, "model: ")
		}
	}
	return ""
}
