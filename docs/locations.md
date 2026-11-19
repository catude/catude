# Where Claude keeps things

`catude doctor` prints this table for your machine, with the real paths and whether each file exists.

## Claude Code

| What | Where |
|---|---|
| User settings | `~/.claude/settings.json` (or `$CLAUDE_CONFIG_DIR/settings.json`) |
| Project settings | `.claude/settings.json` (shared), `.claude/settings.local.json` (yours, git-ignored) |
| Managed (policy) settings | Windows `C:\Program Files\ClaudeCode\managed-settings.json` · macOS `/Library/Application Support/ClaudeCode/managed-settings.json` · Linux `/etc/claude-code/managed-settings.json` |
| User MCP servers, account, per-project state | `~/.claude.json` (inside `$CLAUDE_CONFIG_DIR` when set) |
| Project MCP servers | `.mcp.json` in the project |
| Memory | `~/.claude/CLAUDE.md`, `CLAUDE.md` / `.claude/CLAUDE.md` in the project and its parents, `CLAUDE.local.md`, `@imports` |
| Skills · agents · commands | `~/.claude/{skills,agents,commands}/`, `.claude/{skills,agents,commands}/`, inside each plugin |
| Plugins and marketplaces | `~/.claude/plugins/` (`installed_plugins.json`, `known_marketplaces.json`, caches) |
| Sessions | `~/.claude/projects/<encoded project path>/<session id>.jsonl` |
| claude.ai connectors | **Not on disk.** Only `claude mcp list` reports them; catude asks it (`--live`, key `m`) |

Settings precedence: managed › command-line › local › project › user. Lists (permissions, hooks) are merged.

### Installs catude detects

`claude` on `PATH`, the npm global package (version from its `package.json`), the native installer
(`~/.local/bin/claude`), the legacy `~/.claude/local`, and the editor extensions of VS Code, VS Code Insiders,
VS Code Server (remote/WSL), Cursor, Windsurf and VSCodium; on WSL, also the ones on the Windows side.

### Profiles

Each `CLAUDE_CONFIG_DIR` folder is a separate profile with its own settings, sessions and login. catude lists
`~/.claude`, the current `$CLAUDE_CONFIG_DIR` and every `~/.claude-*` folder. Folders that look like backups or
credential stores (`*backup*`, `*swap*`) are listed but **never read**.

## Claude Desktop

| Install | Config file |
|---|---|
| Windows, installer (`.exe`) | `%APPDATA%\Claude\claude_desktop_config.json` |
| Windows, **Microsoft Store (MSIX)** | `%LOCALAPPDATA%\Packages\Claude_<id>\LocalCache\Roaming\Claude\claude_desktop_config.json`. The app sees this as `%APPDATA%\Claude` through file-system virtualisation, so editing the real `%APPDATA%` copy has no effect |
| Windows, newer builds | `%LOCALAPPDATA%\Claude-Data\…` |
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Linux (official beta, `.deb` for Debian/Ubuntu) | `~/.config/Claude/claude_desktop_config.json`, the path given by the MCP documentation; Anthropic's own Linux page does not document it. catude also looks under `$XDG_CONFIG_HOME/Claude/` when that variable points elsewhere, and labels it as not documented. Flatpak and Snap layouts are not documented anywhere official, so catude does not guess them |
| WSL | the Linux locations **and** the Windows ones under `/mnt/c/Users/<you>/`, labelled "· Windows via WSL" |

When more than one Desktop config exists, catude shows all of them, so you can see which one the app really reads.

## Stale copies

Claude Code sometimes leaves `~/.claude.json.tmp.*`, `.backup*`, `.bak*` or `.old*` copies next to `~/.claude.json`.
They can hold old MCP tokens. The **Profiles** tab and the `stale-config-copies` audit rule list them; delete them
yourself after checking.
