package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/claudecfg"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
)

// x in the MCP and Skills tabs switches the selected server or skill off for this project only, with
// the settings Claude Code documents (see claudecfg/toggles.go). Nothing is uninstalled.

// offName is the name Claude Code uses for a server in disabledMcpServers.
func offName(it inventory.Item) string {
	if it.Scope == paths.ScopeClaudeAI {
		return "claude.ai " + it.Name
	}
	return it.Name
}

// offLabel says how the item is switched off here ("" when it is on).
func (m model) offLabel(it inventory.Item) string {
	switch it.Kind {
	case inventory.MCP:
		if it.Scope == paths.ScopeProject && m.off.MCPJSON[it.Name] || it.Scope != paths.ScopeProject && m.off.MCP[offName(it)] {
			return T("off in this project")
		}
	case inventory.Skill:
		switch m.off.Skills[it.Name] {
		case "off":
			return T("off in this project")
		case "name-only":
			return T("name only in this project")
		case "user-invocable-only":
			return T("only by /name in this project")
		}
	}
	return ""
}

func (m model) toggleHere() (model, tea.Cmd) {
	return m.toggleItem(m.selectedItem())
}

func (m model) toggleItem(it *inventory.Item) (model, tea.Cmd) {
	if it == nil {
		return m, nil
	}
	if m.env.ProjectDir == "" {
		m.status = T("open catude in a project folder to switch things off for it")
		return m, nil
	}
	var backup string
	var err error
	what := ""
	switch it.Kind {
	case inventory.MCP:
		if it.Scope == paths.ScopeDesktop {
			m.status = T("Claude Desktop servers are not part of Claude Code projects: change them in Claude Desktop")
			return m, nil
		}
		fromJSON := it.Scope == paths.ScopeProject
		off := !(fromJSON && m.off.MCPJSON[it.Name] || !fromJSON && m.off.MCP[offName(*it)])
		backup, err = claudecfg.SetMCPOff(m.env, offName(*it), fromJSON, off)
		what = F("%s is on again in this project", it.Name)
		if off {
			what = F("%s is off in this project: new Claude Code sessions here will not connect it", it.Name)
		}
	case inventory.Skill:
		if it.Scope == paths.ScopePlugin {
			m.status = T("plugin skills are switched on and off with their plugin (/plugin)")
			return m, nil
		}
		cur, next := m.off.Skills[it.Name], ""
		for i, v := range claudecfg.SkillValues {
			if v == cur {
				next = claudecfg.SkillValues[(i+1)%len(claudecfg.SkillValues)]
			}
		}
		backup, err = claudecfg.SetSkillOverride(m.env, it.Name, next)
		switch next {
		case "":
			what = F("%s is on again in this project", it.Name)
		case "name-only":
			what = F("%s: Claude sees only its name here (x again: only by /name, then off)", it.Name)
		case "user-invocable-only":
			what = F("%s: only when you type /%s here", it.Name, it.Name)
		case "off":
			what = F("%s is off in this project", it.Name)
		}
	default:
		return m, nil
	}
	if err != nil {
		m.status = gFail + " " + err.Error()
		return m, nil
	}
	m.off = claudecfg.ProjectOff(m.env)
	m.gen++
	m.status = gOK + " " + what
	if backup != "" {
		m.status += " · " + F("backup %s", shorten(backup, 40))
	}
	return m, nil
}

// contextCost is what the item takes in the context window at the start of a session here, from the last
// /context measurement of the Context tab (0 when unknown).
func (m model) contextCost(it inventory.Item) int {
	c := m.ctx.comp
	if c == nil {
		return 0
	}
	switch it.Kind {
	case inventory.MCP:
		for _, r := range c.MCPByServer() {
			if r.Name == it.Name || r.Name == offName(it) {
				return r.Tokens
			}
		}
	case inventory.Skill:
		for _, r := range c.Skills {
			if r.Name == it.Name {
				return r.Tokens
			}
		}
	}
	return 0
}
