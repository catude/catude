# Using catude

Run `catude` in any folder. That folder is the *project* (project and local scopes are read from it); change it
with `--project DIR`.

## Tabs

The tabs most people need are shown at first: Overview, Config, Live, Context, Usage, Models, Profiles,
Sessions, Graph, MCP and Discover. The others (Skills, Agents, Commands, Plugins, Hooks, Available, Tools,
Storage, Audit) are one key away: `,` opens preferences, where `space` shows or hides each tab. The tab bar
says how many are hidden. Sessions also shows an estimated cost per session at API list prices.

| Tab | What you see | Keys |
|---|---|---|
| **Overview** | Every file Claude reads on this machine (Code and Desktop, all scopes) and whether it exists, item counts, the security summary and the live context of your latest session | |
| **Context** | What fills the context window at the start of a session here, measured by `claude /context` in the background and cached: categories, biggest consumers, instructions, skills, MCP servers ([details](context.md)) | `↑↓` `x` off in this project · `r` · `p` compare with a profile |
| **Config** (tab 2) | Every documented setting of every Claude app found (Claude Code scopes and environment variables, `~/.claude.json`, every Claude Desktop install and its policies), set or not: see and change them ([details](config.md)) | `enter` · `space` · `d` · `a` · `e` |
| **Skills** | Skills from user, project, plugins and claude.ai sync. Frontmatter is validated (name, description, length, folder match). Manual-only skills (`disable-model-invocation`) are marked: they cost no context | `e` edit · `x` off / name only in this project |
| **Agents** | The agent map: every subagent (yours, the project's, plugins', built in) with its runs in the last 30 days, who launched it, tools, models, tokens and estimated cost | `e` edit · `space` mark · `r` |
| **Commands** | Slash commands, same sources and checks as skills | `e` edit |
| **Hooks** | Every hook from settings files and plugins, by event and matcher. Commands are shown with secrets redacted | |
| **MCP** | Claude Code servers (user, project, local, plugin), Claude Desktop servers (installer, Microsoft Store, Claude-Data) and claude.ai connectors, with live health | `m` re-check · `x` off in this project |
| **Plugins** | Installed plugins with version and enabled state | |
| **Available** | Every plugin offered by the marketplaces you added; installed ones are ticked | |
| **Discover** | The curated catalog: marketplaces, plugins, MCP servers, skills, tools and best practices ([details](catalog.md)) | `i` `i` install |
| **Sessions** | Projects → sessions → live view: exact context, compactions, cache hit rate, refreshed every 2 s. Summary (first message, files changed, tools), estimated cost. Search every conversation, resume, export, trash | `n` name · `enter` · `f` · `R` · `x` · `D D` · `esc` |
| **Profiles** | Work profiles (start Claude with one), Claude Code installs, configuration folders (`CLAUDE_CONFIG_DIR` profiles), Claude accounts through claude-swap, stale config copies ([details](profiles.md)) | `n` · `e` · `s` · `enter` · `x` · `d d` · `i i` |
| **Usage** | Tokens per day, model and project from your transcripts; plan limits from claude-swap | `[` `]` · `r` |
| **Tools** | Runtimes and tools your MCP servers and hooks need (node, npx, uvx, python, docker, rtk…), versions, what is missing | `r` |
| **Storage** | Sizes of transcripts, caches, scratchpads and Claude Desktop data; safe cleaning; retention warning ([details](storage.md)) | `o` · `c c` · `r` |
| **Audit** | Security findings with the fix for each ([rules](security.md#audit-rules)) | |

## Keys

| Key | Action |
|---|---|
| `?` | every key for every tab |
| `,` · `F2` | preferences: visible tabs, lite view, theme, accent colour, glyphs, animations, key hints |
| `1`…`9` `0` | jump to the first ten tabs |
| `←` `→` `h` `l` `tab` `shift+tab` | switch tab |
| `↑` `↓` `j` `k` · `pgup` `pgdown` · `g` `G` | move |
| `/` | filter the current list (`enter` keeps the filter, `esc` clears it) |
| `space` | mark the selected skill, agent, command, MCP server or plugin for a new work profile |
| `P` | save the marked items as a work profile (asks for a name; built, audited, then installed) |
| `x` | Profiles: start Claude Code with only the selected work profile |
| `e` | open the selected file in `$VISUAL`, `$EDITOR` or Notepad. A copy goes to `<user cache>/catude/backups/` first; when the editor closes, catude re-reads and re-validates the file |
| `c` | exact context composition of the current project (runs `claude -p /context`, about 20–40 s) |
| `m` | ask Claude Code for MCP health (`claude mcp list`, about 20–40 s) |
| `enter` | Sessions: drill down · Profiles: inspect a folder or switch to an account |
| `i` `i` | Discover: run the official install command · Profiles: install claude-swap |
| `r` | reload everything |
| `q` · `ctrl+c` | quit |

Only files you own can be edited: plugin, synced and managed (policy) items are read-only.

## Command line

```
catude [--project DIR]                 open the TUI
catude doctor                          where Claude reads configuration on this machine
catude why [--days N]                  where your tokens went: cache rebuilt after pauses, subagents, claude -p, start-up context
catude health [--days N]               tool calls, failed tool calls, compactions and tokens per response by day and version
catude agents [--days N] [--json]      agent map: every subagent, its runs, who launched it, tools and tokens
catude instructions [--fix]            which CLAUDE.md, CLAUDE.local.md and AGENTS.md Claude Code reads here, auto memory; --fix makes it read what it skips
catude ls KIND [--json] [--live]       skills | agents | commands | hooks | mcp | plugins | marketplaces | available | settings | memory
catude audit [--json]                  security findings (exit code 1 if any is high)
catude sessions [--limit N] [--json]   exact context of recent sessions
catude context [DIR] [--json]          exact composition of the context window
catude instances                       every Claude Code binary, profile folder and stale config copy
catude discover [SECTION] [QUERY]      curated catalog; SECTION: marketplaces plugins mcp skills tools practices
catude profile list|new|load|use|show|rm|dir   work profiles (see docs/profiles.md)
catude rig export [-o FILE] [--summary TEXT] [--owner NAME] [--id URL]
catude rig show SOURCE [--json]        SOURCE: ./file.json · host/path · host · gh:owner/repo[@ref]
catude snapshot [--live]               render every tab as plain text (CI, screenshots, bug reports)
catude version
```

`--live` also asks Claude Code for claude.ai connectors and MCP health. Raw file contents are never part of the
JSON output.

### In CI

```yaml
- run: catude audit --project .     # fails the job on any high-severity finding
```

### Bug reports

`catude snapshot > snapshot.txt` renders every tab as text. Review it before attaching it: it contains your
paths and item names (secret values are already masked).
