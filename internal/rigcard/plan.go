package rigcard

import (
	"fmt"
	"strings"

	"github.com/catude/catude/internal/inventory"
)

// Step is one line of an adoption plan.
type Step struct {
	Kind   string // marketplace | plugin | skill | agent | command | mcp | connector | hook | setting
	Name   string
	State  string // have | missing | different | review
	Detail string
	Cmd    string // official command to run (never executed by catude without the user)
}

// Plan compares a card with the local inventory and proposes official commands.
func Plan(c Card, inv inventory.Inventory) []Step {
	have := map[inventory.Kind]map[string]inventory.Item{}
	for _, it := range inv.Items {
		if have[it.Kind] == nil {
			have[it.Kind] = map[string]inventory.Item{}
		}
		have[it.Kind][strings.ToLower(it.Name)] = it
	}
	var out []Step
	for _, m := range c.Marketplaces {
		s := Step{Kind: "marketplace", Name: m.Name, State: "missing"}
		if _, ok := have[inventory.Marketplace][strings.ToLower(m.Name)]; ok {
			s.State = "have"
		} else if m.Source.Repo != "" {
			s.Cmd = "claude plugin marketplace add " + m.Source.Repo
		} else {
			s.Cmd = "claude plugin marketplace add " + m.Source.URL
		}
		if !m.Source.Pinned() {
			s.Detail = "not pinned"
		}
		out = append(out, s)
	}
	for _, p := range c.Plugins {
		s := Step{Kind: "plugin", Name: p.ID, State: "missing", Cmd: "claude plugin install " + p.ID}
		if it, ok := have[inventory.Plugin][strings.ToLower(p.ID)]; ok {
			s.State, s.Cmd = "have", ""
			if v := strings.TrimPrefix(it.Description, "v"); p.Version != "" && v != p.Version {
				s.State, s.Detail = "different", fmt.Sprintf("you have %s, card has %s", v, p.Version)
			}
		}
		out = append(out, s)
	}
	arts := func(kind string, k inventory.Kind, as []Artifact) {
		for _, a := range as {
			s := Step{Kind: kind, Name: a.Name, State: "missing"}
			if _, ok := have[k][strings.ToLower(a.Name)]; ok {
				s.State = "have"
			}
			if a.Source != nil {
				switch {
				case a.Source.Type == "local":
					s.Detail = "author's local copy (not downloadable)"
				case a.Source.Pinned():
					s.Detail = fmt.Sprintf("%s %s @ %s", firstNonEmpty(a.Source.URL, a.Source.Repo), a.Source.Path, a.Source.Commit[:min(12, len(a.Source.Commit))])
				default:
					s.Detail = "unpinned source"
				}
			}
			if a.SHA256 != "" {
				s.Detail += " · sha256 " + a.SHA256[:12]
			}
			out = append(out, s)
		}
	}
	arts("skill", inventory.Skill, c.Skills)
	arts("agent", inventory.Agent, c.Agents)
	arts("command", inventory.Command, c.Commands)
	for _, m := range c.MCPServers {
		s := Step{Kind: "mcp", Name: m.Name, State: "missing"}
		if _, ok := have[inventory.MCP][strings.ToLower(m.Name)]; ok {
			s.State = "have"
		} else {
			s.Cmd = mcpAddCommand(m)
		}
		var secrets []string
		for _, v := range append(append([]Variable{}, m.Env...), m.Headers...) {
			if v.IsSecret {
				secrets = append(secrets, v.Name)
			}
		}
		if len(secrets) > 0 {
			s.Detail = "you provide: " + strings.Join(secrets, ", ")
		}
		out = append(out, s)
	}
	if c.Desktop != nil {
		for _, n := range c.Desktop.Connectors {
			s := Step{Kind: "connector", Name: n, State: "missing", Detail: "claude.ai → Settings → Connectors"}
			if _, ok := have[inventory.MCP][strings.ToLower(n)]; ok {
				s.State = "have"
			}
			out = append(out, s)
		}
	}
	for _, h := range c.Hooks {
		out = append(out, Step{Kind: "hook", Name: strings.TrimSpace(h.Event + " " + h.Matcher), State: "review", Detail: "runs code on your machine: review the script; never enabled automatically"})
	}
	return out
}

func mcpAddCommand(m MCPServer) string {
	var b strings.Builder
	b.WriteString("claude mcp add " + m.Name)
	if m.Transport != "stdio" && m.URL != "" {
		b.WriteString(" --transport " + m.Transport)
		for _, h := range m.Headers {
			b.WriteString(fmt.Sprintf(` --header "%s: ${%s}"`, h.Name, envName(h.Name)))
		}
		b.WriteString(" " + m.URL)
		return b.String()
	}
	for _, v := range m.Env {
		b.WriteString(fmt.Sprintf(" -e %s=<%s>", v.Name, map[bool]string{true: "secret", false: "value"}[v.IsSecret]))
	}
	b.WriteString(" -- " + m.Command)
	if len(m.Args) > 0 {
		b.WriteString(" " + strings.Join(m.Args, " "))
	}
	return b.String()
}

func envName(h string) string {
	return strings.ToUpper(strings.NewReplacer("-", "_", " ", "_").Replace(h))
}
