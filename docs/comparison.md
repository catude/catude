# catude and the alternatives

Plenty of good tools touch part of what catude does. This page says what each is best at, so you can pick one
or combine them.

| Tool | Kind | Best at | Where catude differs |
|---|---|---|---|
| Claude Code `/context`, `/mcp`, `/plugin`, `/doctor` | built in | Authoritative, inside one session | catude sees the whole machine without a session, both apps and every profile, and reuses `/context` for exact numbers |
| [cc-switch](https://github.com/farion1231/cc-switch) | desktop app (Tauri) | Switching **providers** (endpoints, keys, models) across Claude Code, Codex, Gemini CLI…, local proxy with failover, MCP and skill sync | catude is a terminal tool, read-only by default, with exact context, an audit and Claude Desktop MSIX support; it switches **accounts** (via claude-swap), not providers |
| [claude-code-templates](https://github.com/davila7/claude-code-templates) | catalog + `npx` installer + web | Huge catalog of agents, commands, MCP and settings, analytics dashboard | catude installs only through official `claude plugin` commands and audits what is already installed |
| [ccusage](https://github.com/ccusage/ccusage) | CLI | Cost and token **usage reports** by day, month, session and block | catude measures **context** (what fills the window), not spend; both de-duplicate the same way |
| [claude-hud](https://github.com/jarrodwatts/claude-hud), [ccstatusline](https://github.com/sirmalloc/ccstatusline) | status lines | Live context and usage inside the running session | catude is outside the session and covers every session and project |
| [claude-swap](https://github.com/realiti4/claude-swap) | CLI | Multiple Claude accounts, usage per account | catude uses it as its account back end and adds the TUI |
| [opcode](https://github.com/winfunc/opcode) | desktop GUI | Session browsing, custom agents, checkpoints | catude is a small terminal binary focused on configuration |
| ccboard, ccexp, lazyclaude, lazyskills, mcpick, ccmcp | TUIs and CLIs | Browsing config, skills or MCP servers; toggling MCP servers per session (mcpick, ccmcp) | catude combines these views with Claude Desktop, live claude.ai connectors and an audit; per-session MCP toggling is on the roadmap |
| agentshield, cc-audit | scanners | Deeper static analysis, SARIF output, baselines | catude's audit is built in and quick; SARIF and baselines are on the roadmap |
| ClaudeTools (PowerShell) | scripts | Fixing the Claude Desktop MSIX config path on Windows | catude shows every Desktop copy on every OS |

## What catude took from them

- From **ccboard**: show the *effective* configuration: which scope wins and why (done for settings in Config).
- From **mcpick / ccmcp**: the token cost of each MCP server and quick enable/disable (roadmap).
- From **cc-audit / agentshield**: SARIF output and baselines for CI (roadmap).
- From **cc-switch**: syncing MCP servers between Claude Desktop and Claude Code (roadmap 0.2).
- From **claude-code-templates**: a browsable catalog, done as the embedded Discover tab.

If a tool here does one job better, use it. catude tries to be the place where everything is visible at once.
