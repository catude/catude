# Profiles and accounts

The **Profiles** tab has five parts: work profiles, installations, configuration folders, accounts and stale copies.

## Work profiles

A work profile is **a set of skills, agents, commands, MCP servers, plugins and safe settings for one kind of
work** (`gamedesign`, `swe`, `data`, `writing`) that Claude Code loads for one session. Your `~/.claude` is never
modified.

### Create one

In the TUI, Profiles → `n` opens the profile wizard:

1. **Choose.** Every skill, agent, command, MCP server and plugin on this machine in one list. Type to search
   (fuzzy, like fzf), `space` ticks an item, `tab` ticks everything shown. The top line shows what you chose and
   an estimate of the tokens the chosen skills add at start.
2. **Name and options.** Name, model (keep yours, opus, sonnet, haiku) and mode (on top of your setup, or only
   this profile). `enter` saves: files are copied, hashed and audited in the background.

`e` on a profile reopens the wizard with its items ticked. You can also mark items with `space` in the Skills,
Agents, Commands, MCP or Plugins tabs and press `P`. Or from the command line:

```sh
catude profile new gamedesign --skills level-design,narrative --agents playtester --mcp blender \
  --summary "Game design" --model opus
```

Files are **copied** into the profile, so later edits to the originals do not change it. MCP secrets become
`${ENV_VAR}` references: set those variables yourself. Hooks are never part of a profile.

### Save your current setup

Profiles → `a` opens the wizard with your current setup already ticked: your own skills, agents and commands
(not the ones plugins or claude.ai bring), your MCP servers and your enabled plugins. Untick what you do not
want, name it, save. From the command line: `catude profile save NAME`. It is a frozen copy: later changes to
`~/.claude` do not alter it, so you can always go back to it.

### Use one, switch between them

In the Profiles tab, `enter` starts Claude Code with the selected profile in its saved mode and `x` in the
other mode; you return to catude when Claude exits. `s` makes it the **active profile** (marked `●` and shown
in the header): `catude claude` then starts it, and after a one-time shell setup so does plain `claude`:

```sh
eval "$(catude shell-init zsh)"        # add to ~/.zshrc (bash: same with bash)
catude shell-init fish | source         # ~/.config/fish/config.fish
Invoke-Expression (& catude shell-init powershell | Out-String)   # $PROFILE

catude profile switch gamedesign       # from now on `claude` = game design profile
catude profile switch none             # back to your normal setup
ct switch                               # pick from a numbered list (ct = catude)
catude profile current
```

The Context tab (`p`) measures the same folder with any profile, so you can compare what each costs.

From the command line:

```sh
catude profile use gamedesign                 # your setup + the profile
catude profile use gamedesign --exclusive     # only the profile (and Claude Code's built-in skills)
catude profile use gamedesign --isolated      # a separate CLAUDE_CONFIG_DIR with its own login
catude profile use gamedesign -- --resume     # anything after -- goes to claude
```

| Mode | What Claude Code loads | How |
|---|---|---|
| default | everything you normally have **plus** the profile | `--plugin-dir`, `--mcp-config`, `--settings` |
| `--exclusive` | built-in skills + the profile only: no user skills, user plugins, user hooks, user MCP servers or claude.ai connectors. Project settings still apply | adds `--strict-mcp-config --setting-sources project,local` |
| `--isolated` | nothing from `~/.claude`; the profile has its own configuration folder and you log in once for it | `CLAUDE_CONFIG_DIR=<profile>/home` |

Measured on Claude Code 2.1.326 with a machine that has ~20 personal skills and ~30 MCP servers and connectors:
25.2k tokens at start without a profile, 19.5k with `--exclusive` (profile skill loaded, personal ones gone).

### Check that it really loads

`v` in the Profiles tab (or `catude profile check NAME`) starts Claude Code headless with the profile and
asks it what it loaded. Nothing is sent to the model. Each item is reported as `✓` loaded at start, `○`
manual only (skills with `disable-model-invocation` and commands without a description: you type them, they
cost no context) or `✗` not loaded, with the reason.

### Use its MCP servers in Claude Desktop

`catude profile desktop NAME` copies the profile's local MCP servers (the ones that start a program) into
Claude Desktop's `claude_desktop_config.json`. It shows what it would do first, asks, and backs the file up.
It leaves out remote servers (add those in Claude Desktop under Settings, Connectors), servers that use
`${VAR}` references (catude never writes a secret into a file: add them by hand) and servers with the
same name and a different definition, unless you pass `--replace`. With several Claude Desktop installs,
choose one with `--config FILE`. Restart Claude Desktop afterwards.

### Presets

Profiles → `p` opens a gallery of profiles based on the public setups of well-known practitioners. See
[presets.md](presets.md).

### Share one, load someone else's

A profile is described by a [Rigcard](rigcard.md) (`profile.json`), so it can live in a file, a repository or a
web page:

```sh
catude profile load ./gamedesign.json
catude profile load ada.dev/rig/gamedesign
catude profile load gh:ada/claude-profiles@v1 --name gamedesign
```

Before anything is installed, catude:

1. **Validates** the card: no secrets, no unsafe names, no impersonated official marketplaces, https only.
2. **Downloads only pinned sources**: a GitHub repo + path + 40-hex commit (or a file with its `sha256`).
   Unpinned sources are refused unless you pass `--allow-unpinned`. Downloads go through the same SSRF guard as
   cards (public addresses only, https redirects, size and time limits).
3. **Verifies hashes**: if the card gives a `sha256`, the downloaded files must match or nothing is installed.
4. **Extracts safely**: regular files only (no symlinks), no `..` or absolute paths, 2 MiB per file, 10 MiB per
   skill, nothing marked executable.
5. **Audits** the result with every [audit rule](security.md#audit-rules) and lists every script it contains.
6. **Asks you.** High-severity findings block the install unless you pass `--accept-risk`.
7. **Never installs hooks**, never writes `permissions.allow`, `env` or `apiKeyHelper`; plugins the card needs
   but you do not have are listed with the official command to install them, not installed.

A minimal profile file written by hand:

```json
{
  "rigcard": "0.1",
  "summary": "Game design",
  "skills": [
    { "name": "level-design",
      "source": { "type": "github", "repo": "ada/claude-skills", "path": "skills/level-design",
                  "commit": "3f1c9a0e5b7d2c4f6a8e1b3d5f7a9c0e2b4d6f8a" } }
  ],
  "plugins": [{ "id": "superpowers@claude-plugins-official", "enabled": true }],
  "mcpServers": [{ "name": "linear", "transport": "http", "url": "https://mcp.linear.app/mcp" }],
  "settings": { "model": "opus", "permissions": { "deny": ["Read(./.env)"] } }
}
```

Profiles live in `catude profile dir` (`%APPDATA%\catude\profiles`, `~/Library/Application Support/catude/profiles`
or `~/.config/catude/profiles`; override with `CATUDE_HOME`).

## Installations

Every Claude Code binary catude finds (PATH, npm, native installer, editor extensions, WSL/Windows) with its
version. Several versions at once is a common source of "it works in the terminal but not in VS Code".

## Configuration folders

Claude Code's official way to keep separate setups is one folder per profile:

```sh
alias claude-work='CLAUDE_CONFIG_DIR=~/.claude-work claude'
```

Each folder has its own settings, skills, MCP servers, sessions and login. catude lists `~/.claude`, the active
`$CLAUDE_CONFIG_DIR` and every `~/.claude-*` folder. Press `enter` on one to **inspect it**: every tab then shows
that profile, without changing anything. Folders whose name says *backup* or *swap* are listed but never opened,
because they usually hold credentials.

## Claude accounts (claude-swap)

To switch the account the default profile is logged into, catude uses
[claude-swap](https://github.com/realiti4/claude-swap) (MIT) when it is installed. catude only calls its public
commands (`cswap list --json` and `cswap switch N`) and **never reads or stores a token itself**.

- Each account shows its organisation, whether it is active, and usage bars for the 5-hour and 7-day limits, with
  a warning when it will run out before the limit resets.
- `enter` switches to the selected account. Restart open Claude sessions afterwards.
- `i` `i` installs or upgrades claude-swap with `uv tool install --force claude-swap==0.29.0` (a pinned version).

Things you should know before using claude-swap:

1. It keeps every account's refresh token on disk, **base64-encoded, not encrypted** (on Windows and Linux).
   Anything running as your user can read them.
2. Versions before 0.30 have a bug (#516) where `cswap run` with `CLAUDE_CONFIG_DIR` can log other sessions out.
3. `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` and `ANTHROPIC_BASE_URL` take precedence over the OAuth login and
   are not switched.
4. Automatically rotating accounts to get around usage limits may break your plan's terms. catude never does it.

## Stale copies

Leftover `~/.claude.json.*` copies are listed with a warning: they can hold old MCP tokens.
