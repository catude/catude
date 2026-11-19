# Changelog

## 0.1.0 (unreleased)

First public version.

- Renamed from lazyrig to catude (cat + Claude); short command `ct`, lite view `ctlite`. The first run moves the old
  preference, profile and cache folders.
- `catude why` and the Usage tab: where the tokens went (cache rebuilt after pauses, subagents, `claude -p` runs,
  context before the first message, most expensive conversations, cache hit rate, Claude Code versions).
- `catude health`: tool calls, failed tool calls, compactions and tokens per response by day and version.
- Agent map (Agents tab, `catude agents`): every subagent joined with its real runs, who launched it, tools, models,
  tokens and estimated cost; defined agents that never run are flagged.
- MCP and Skills tabs: `x` switches a server or skill off for this project only (`disabledMcpServers`,
  `disabledMcpjsonServers`, `skillOverrides`), and each row shows its cost in the context.
- Sessions: `n` names a conversation (the line `/rename` writes); the detail shows your first message, the files
  changed and the tools used.
- `catude instructions` and two audit rules: an `AGENTS.md` that Claude Code does not read (a `CLAUDE.local.md`
  stops it) and `.agents/skills`, which Claude Code never reads.
- Audit: passwords and tokens saved in permission rules (high) and commands that delete data allowed without
  asking (warn).
- Optional system notification when a session finishes (Windows, macOS, Linux).
- Each extension tab says in one sentence what a skill, subagent, command, hook, MCP server or plugin is.
- Context tab: choose one of the biggest consumers and `x` switches it off for this project.
- `catude instructions --fix`: adds the `@AGENTS.md` import to the file that stops it and copies `.agents/skills` into
  `.claude/skills` (asks first, backup); shows the project's auto memory and whether `MEMORY.md` is over what loads.
- Sessions: conversations of Claude Desktop's Code tab show their Desktop title and whether they are archived.
- `catude health`: API errors Claude Code retried and the cache hit rate, per day. Usage: when each plan window resets
  and whether the current pace lasts (with claude-swap).
- README rewritten around the questions catude answers, with GIFs and the real output of `catude why`, `agents`, `health`.
- Fixed: Usage counted twice the responses of a conversation resumed or forked into another folder.
- Fixed: `uninstall --help` uninstalled; uninstall now asks first.

- Config: **Effective settings** shows the value Claude Code uses for each key of this project, the file it
  comes from and what it wins over, following the documented precedence (lists combine; `fallbackModel` and
  `modelPicker` follow their own rules).
- A notice in any tab when a Claude Code session finishes and waits for you; optional bell (`,`).
- Sessions: estimated cost of each session at API list prices (models.dev), labelled as an estimate.
- Live: reads the to-do list of current models (TaskCreate / TaskUpdate) and says when a transcript is in a
  format catude does not know yet, instead of showing an empty card.
- The core tabs are shown at first; the rest are switched on in preferences, and the tab bar says how many.
- `catude profile desktop NAME`: copy a profile's local MCP servers into Claude Desktop (preview, consent,
  backup; remote servers and `${VAR}` secrets are left out).
- Claude Code plugin in the repository: `/catude:doctor`, `/catude:audit`, `/catude:sessions`, `/catude:models`.
- Weekly maintenance workflow: catalog stars and activity (archived or quiet repositories are marked in
  Discover) and a check of the settings reference against Anthropic's documentation.
- Fixed: settings files saved with a byte order mark (Notepad) were reported as invalid JSON in the inventory.
- Settings reference: three new environment variables; `disableSshSavedPasswords` is no longer on the
  settings page and is marked as such.

- Cat logo (light and dark), GitHub avatar and social preview image; release workflow with GoReleaser; issue and
  pull request templates.
- Inventory of Claude Code and Claude Desktop: skills, agents, commands, hooks, MCP servers, plugins,
  marketplaces, settings and memory across managed, user, project, local, plugin and claude.ai scopes.
- Claude Desktop on Windows (installer, Microsoft Store/MSIX, Claude-Data), macOS and Linux (the paths Anthropic
  documents); WSL also reads the Windows side.
- Live MCP health and claude.ai connectors (`claude mcp list`).
- Exact session context from API usage (de-duplicated) and composition from `claude -p /context`.
- Profiles tab: installs, `CLAUDE_CONFIG_DIR` folders, claude-swap accounts with usage, stale config copies.
- Live editing with backup and re-validation.
- Security audit with 23 rules; `catude audit` exits 1 on high findings.
- Discover: embedded catalog of marketplaces, plugins, MCP servers, skills, tools and practices, with safe
  double-press install through official commands.
- Rigcard 0.1: `catude rig export` and `catude rig show`.
- Work profiles: sets of skills, agents, commands, MCP servers and plugins per kind of work, created from the
  TUI (`space`, `P`) or `catude profile new`, loaded from Rigcards with pinning, hash checks and an audit, and
  started with `--plugin-dir`, `--exclusive` or `--isolated`.
- Status glyphs that never render as double-width emoji.
- Profile wizard (fzf-like picker, name, model, mode), active profile with `catude claude` and `shell-init`.
- Context tab: background, cached `claude /context` with biggest consumers and per-profile comparison.
- Storage tab: sizes of every Claude Code / Desktop / scratchpad location, safe cleaning, retention warning.
- Sessions: full-text search across conversations, resume in the right folder, Markdown export, restorable trash.
- Terminal compatibility: unicode / WGL4-safe / ASCII glyph sets chosen automatically, emoji cleaning, themes
  (dark, light, high contrast, mono), `?` help, number keys, mouse, scrolling tab bar.
- Config presets (`p` in Config): 30 groups of documented settings, each sourced and quoted (Anthropic docs and
  example settings, Trail of Bits), previewed against the selected file and applied after a backup.
- Discover catalog: 177 plugins, 89 skills, 55 marketplaces and 25 MCP servers, verified at their sources and
  tagged; `#tag` search.
- `catude export` and `catude import`: move or share a setup as one file.
- Sessions: `c` resumes the selected session in Claude Code, like `R`.
- README rewritten with a tour, guides and FAQ; screenshots and GIFs made from an invented machine by
  scripts/demo (sandboxed fixture home, scripted keys, SVG/PNG/GIF rendering), with no real data in them.
- scripts/set-owner.mjs points the module path, installers and docs at the GitHub home.
- Live tab (tab 3): every running Claude Code session and what it is doing, updated every second (status,
  running tool, timeline of tool calls, subagents, to-do list, model, effort, context), from Claude Code's
  documented session files and the end of each transcript.
- Interface in 10 languages (en, es, ja, zh-Hans, zh-Hant, ko, de, fr, pt-BR, ru), detected from the system or
  chosen with `,` / `--lang`; columns measured in terminal cells so CJK text stays aligned.
- Models tab and `catude models`: which Claude model and effort level is stronger for each kind of work, with
  cost per task, price and advice on the model you use. Public data (Epoch AI, models.dev): no account, no key.
- Config tab (now tab 2) lists every documented setting of every Claude app: Claude Code settings and
  environment variables, `~/.claude.json` global options (changed only while Claude Code is closed), Claude
  Desktop's file and policies. It has categories, descriptions, defaults, search, and `d` to go back to the default.
- Graph appears at once (tools located instead of started, processes added in the background: from 9–36 s to
  0.2 s on a real machine); `w` opens the diagram in the browser (`catude graph html`).
- Graph as a real tree with branches, `enter` / `+` / `-` to open and close, `E` / `C` for everything; scrollbars and
  position hints on every long list.
- Installer: first run offers `catude install` (pre-selected): user PATH, `ct` short name, optional shell
  integration; `catude uninstall`; release scripts with checksum verification.
- `ct switch` (numbered picker), `catude profile save NAME` and Profiles → `a`: save the current setup.
- Usage tab (tokens per day / model / project), Tools tab (runtimes and companion tools, missing commands,
  Microsoft Store python stub), audit rules `mcp-command-not-found` and `claude-json-large`.
- Shortcuts work with Caps Lock.
- Preset gallery (Profiles → `p`, `catude presets`, `profile load preset:ID`): 15 work profiles based on
  public setups, with sources, monogram art, pinned and openly licensed files only.
- `catude profile check` and `v`: Claude Code itself confirms what a profile loads (loaded at start vs
  manual-only skills and commands).
- Claude Code version in the header, debug panel (F12 / ctrl+o) and `catude debug`.
- Config tab: every Claude app's configuration file (Claude Code scopes, each Claude Desktop install) with
  inline, order-preserving, backed-up edits and an explanation of which app reads what.
- Preferences (`,`): visible tabs, lite view, theme, accent colour, glyphs, animations, key hints.
  `ctlite` / `--lite` open the lite view.
- `catude card [--svg]`: a shareable, numbers-only summary of your setup.
- The installer notices an older installed copy and offers to update it; `catude version` shows the commit.
- Presets are announced in the Profiles tab.
- Graph tab and `catude graph [mermaid|dot|--json]`: files, hooks, MCP servers, tools, variables, hosts,
  profiles, folders and running processes, with relations.
- Rigcard bundles: `profile export` writes a card plus the profile's files with hashes; `profile load`
  reads them from a folder or a GitHub repository. Discovery tested end to end over HTTPS.
- Security hardening after an independent review: terminal escape filtering everywhere, headless Claude
  only in trusted folders, stricter card validation, Windows-safe archive paths, link-safe writes,
  argument-injection fixes, full consent screen for programs a card starts.
- Linux paths follow Anthropic's documentation; Linux and macOS builds are marked experimental;
  `scripts/package` builds archives with correct permissions and checksums.
- Windows: detects consoles without ANSI support (Windows 10 legacy console) before opening the interface.
- Presets corrected against their sources and showing what their users ask for.
- Rigcard `owner.art`: optional, bounded ASCII art chosen by the card's owner.
- Performance: cached rows, audit only when something changed, project info off the UI thread; every frame
  renders in a few milliseconds.
