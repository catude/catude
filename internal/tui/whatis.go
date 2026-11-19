package tui

import "github.com/charmbracelet/lipgloss"

// whatIs is one plain sentence on what the items of a tab are and when to use them, shown above the
// details: the difference between a skill, a subagent, a command, a hook, an MCP server and a plugin is
// the most common question about Claude Code's extensions.
func whatIs(t tab) string {
	switch t {
	case tSkills:
		return T("A skill is a folder with a SKILL.md: instructions and files Claude loads only when a task needs them. Only its name and description are always in context. Use one for a way of doing something you repeat.")
	case tAgents:
		return T("A subagent is a separate Claude with its own context, tools and model. The main conversation hands it a task and gets back a summary, so long searches and reviews do not fill your context.")
	case tCommands:
		return T("A command is a prompt saved in a Markdown file that you start by typing /name. Claude Code now treats commands like skills; the difference is that a skill can also be chosen by Claude itself.")
	case tHooks:
		return T("A hook is a shell command that Claude Code (not Claude) runs on an event: before or after a tool, when you send a message, when a session ends. Use one for what must always happen, such as formatting.")
	case tMCP:
		return T("An MCP server gives Claude tools to reach another system: GitHub, a database, a browser. The definitions of its tools take context in every session where it is on.")
	case tPlugins:
		return T("A plugin is a package of skills, subagents, commands, hooks and MCP servers that you install and switch on together from a marketplace.")
	case tCatalog:
		return T("Plugins offered by the marketplaces you added. Installing one runs the official command: claude plugin install name@marketplace.")
	}
	return ""
}

func whatIsBlock(t tab, w int) string {
	s := whatIs(t)
	if s == "" {
		return ""
	}
	return sMuted.Render(lipgloss.NewStyle().Width(max(20, w)).Render(s)) + "\n\n"
}
