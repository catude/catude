# Linux (experimental)

The Linux build of catude is **experimental**. It is built and tested on every change (the whole test
suite runs on Linux, and the binary is exercised on Ubuntu under WSL), but it has had far less real-world use
than the Windows build. Please report anything that looks wrong: `catude debug` prints what a bug report needs.

```sh
# x86_64
tar -xzf catude_linux_amd64.tar.gz && ./catude install
# arm64
tar -xzf catude_linux_arm64.tar.gz && ./catude install
```

`catude install` copies the binary to `~/.local/bin` (with `ct` and `ctlite` next to it) and, if that folder
is not on your `PATH`, adds one line to `~/.zshrc`, `~/.bashrc` or `~/.profile`, marked `# added by catude`.

## Where catude looks on Linux

Only paths that Anthropic's documentation states are used. Where something is not documented, catude says
so instead of guessing.

| What | Path | Source |
|---|---|---|
| Claude Code configuration | `~/.claude/` (or `$CLAUDE_CONFIG_DIR`) | [claude-directory](https://code.claude.com/docs/en/claude-directory) |
| Account, your MCP servers, per-project state | `~/.claude.json` (inside `$CLAUDE_CONFIG_DIR` when set) | [claude-directory](https://code.claude.com/docs/en/claude-directory) |
| Organisation policy | `/etc/claude-code/managed-settings.json`, `managed-settings.d/`, `managed-mcp.json` | [managed-settings](https://code.claude.com/docs/en/managed-settings) |
| Organisation instructions | `/etc/claude-code/CLAUDE.md` | [memory](https://code.claude.com/docs/en/memory) |
| Your instructions | `~/.claude/CLAUDE.md`; project `./CLAUDE.md`, `./.claude/CLAUDE.md`, `./CLAUDE.local.md` | [memory](https://code.claude.com/docs/en/memory) |
| Session transcripts and other data | `~/.claude/projects/`, `file-history/`, `plans/`, `debug/`, `shell-snapshots/`, … | [claude-directory](https://code.claude.com/docs/en/claude-directory) |
| Plugins and marketplaces | `~/.claude/plugins/` (`installed_plugins.json`, `known_marketplaces.json`, `cache/`) | [plugins](https://code.claude.com/docs/en/plugins/loading) |
| Scratchpads | `/tmp/claude-<uid>/<project>/<session>/scratchpad/`, or the same under `$TMPDIR`; `CLAUDE_CODE_TMPDIR` moves it | [claude-directory](https://code.claude.com/docs/en/claude-directory#session-scratchpad-directory) |
| Native installer | `~/.local/bin/claude` → `~/.local/share/claude/versions/` | [setup](https://code.claude.com/docs/en/setup) |
| Credentials | `~/.claude/.credentials.json` (mode 0600); **never read by catude** | [authentication](https://code.claude.com/docs/en/authentication) |
| Claude Desktop (official beta, `.deb` for Debian/Ubuntu) | `~/.config/Claude/claude_desktop_config.json`, given by the MCP documentation; Anthropic's Linux page does not state it | [desktop-linux](https://code.claude.com/docs/en/desktop-linux), [MCP docs](https://modelcontextprotocol.io/docs/develop/connect-local-servers) |
| Claude Desktop logs | `~/.config/Claude/logs/` (`mcp.log`, `mcp-server-<name>.log`) | [MCP docs](https://modelcontextprotocol.io/docs/develop/connect-local-servers) |

Not documented, therefore not used: an XDG location for Claude Code (`~/.config/claude`), Flatpak or Snap
layouts for Claude Desktop, and a path for the VS Code extension. When `$XDG_CONFIG_HOME` points somewhere other
than `~/.config`, catude also checks `$XDG_CONFIG_HOME/Claude/` for Claude Desktop and labels that entry as
*not documented*.

Under WSL, catude reads the Linux side **and** the Windows side of the same machine (`/mnt/c/Users/<you>/…`),
labelled "Windows via WSL". Set `CATUDE_NO_WSL=1` to read the Linux side only.

## What is different on Linux

- **Processes** (Graph tab) come from `/proc`, including each process's working folder.
- **Opening folders** uses `xdg-open`; without a desktop it reports the path instead.
- **Install** edits a shell startup file instead of the Windows user `PATH`.
- **Colours and glyphs** follow the terminal: the Linux text console (`TERM=linux`) gets the safe glyph set,
  a non-UTF-8 locale gets ASCII.

## Known limits

- Not yet tried on Fedora, Arch, Alpine (musl) or NixOS; the binary is static, so it should run.
- Claude Desktop for Linux is itself a beta: its paths may change.
- macOS builds exist but have not been run on a real Mac.
