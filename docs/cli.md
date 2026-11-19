---
description: >-
  Complete catude command-line reference: why, health, agents, audit, context, instructions, doctor, ls,
  sessions, profiles, presets, models, graph, export, import, rig, card, install and every flag.
---

# Command line

Everything in the app is also a command. `ct` is the same program as `catude`. Every listing accepts `--json`,
and flags can go before or after the arguments. `catude help` prints the summary.

## At a glance

```sh
catude why                            # where your tokens went, and what to change
catude health                         # tool calls, failures, API retries, compactions, cache hit, by day and version
catude agents                         # the agent map: every subagent, its runs, who launched it, tokens
catude instructions [--fix]           # which CLAUDE.md, CLAUDE.local.md and AGENTS.md Claude Code reads here
catude audit                          # security findings (exit code 1 if any is high)
catude context                        # what fills your context window, measured by Claude Code
catude doctor                         # every file Claude reads here, and whether it exists
catude ls mcp --live                  # MCP servers including claude.ai connectors, with health
catude profile new web --skills frontend-design,tdd --mcp playwright
ct switch                             # pick the active work profile from a list
eval "$(catude shell-init zsh)"       # from now on, plain `claude` starts the active profile
catude models compare "opus 5.5 high" "fable 5.2 medium" --for coding
catude export my-setup.json           # your setup in one file, without secrets
```

## Global flags

| Flag | Meaning |
|---|---|
| `--project DIR` | the project folder whose settings, skills and MCP servers are read (default: the current folder) |
| `--json` | machine-readable output. Raw file contents are never part of it, and secrets stay masked |
| `--lang CODE` | interface language: `en`, `es`, `ja`, `zh-Hans`, `zh-Hant`, `ko`, `de`, `fr`, `pt-BR`, `ru` (default: your system's) |
| `--glyphs unicode\|safe\|ascii` | what to draw with, for terminals and fonts that lack symbols (or `CATUDE_GLYPHS`) |
| `--theme auto\|dark\|light\|high-contrast\|mono` | colours (or `CATUDE_THEME`) |
| `--lite` | the terminal app with only the tabs you chose (`ctlite` is the same) |

## The terminal app

| Command | What it does |
|---|---|
| `catude` · `ct` | open the terminal app in the current folder. On the very first run it offers to install itself |
| `catude --project DIR` | open it for another project |
| `catude --lite` · `ctlite` | only the tabs you chose in preferences (`,`) |

Tabs and keys: [Tabs and keys](usage.md).

## Usage and cost

| Command | What it does |
|---|---|
| `catude why [--days N]` | where your tokens went: cache rebuilt after pauses, subagents, `claude -p`, start-up context, the most expensive conversations, and what to change. Default 30 days. [Guide](why.md) |
| `catude health [--days N]` | tool calls, failed tool calls, API retries, compactions, cache hit and tokens per response, by day and by Claude Code version. [Guide](why.md#your-own-evidence-when-something-feels-worse) |
| `catude agents [--days N] [--json]` | the agent map: every subagent, how often it ran, who launched it, tools, models and tokens. [Guide](agents.md) |
| `catude sessions [--limit N] [--json]` | exact context of recent sessions (default 15) |
| `catude context [DIR] [--json]` | exact composition of the context window, measured by `claude /context`. [Guide](context.md) |

## What Claude reads, and whether it is safe

| Command | What it does |
|---|---|
| `catude doctor` | every file Claude Code and Claude Desktop read on this machine and for this project, and whether it exists |
| `catude instructions [--fix] [--yes]` | which `CLAUDE.md`, `CLAUDE.local.md` and `AGENTS.md` Claude Code reads here, and the auto memory; `--fix` makes it read what it skips. [Guide](instructions.md) |
| `catude audit [--json]` | security findings with the fix for each. Exit code 1 if any finding is high. [Rules](security.md#audit-rules) |
| `catude ls KIND [--json] [--live]` | list one kind of item. `KIND`: `skills`, `agents`, `commands`, `hooks`, `mcp`, `plugins`, `marketplaces`, `available`, `settings`, `memory`. `--live` (with `mcp`) also asks Claude Code for claude.ai connectors and health |
| `catude instances` | every Claude Code binary, configuration folder (`CLAUDE_CONFIG_DIR`) and leftover config copy |
| `catude graph [FORMAT] [--json]` | everything your Claude uses and how it connects. No format: counts. `mermaid`, `mermaid-full`, `html` (a page with the diagram), `dot` (Graphviz). [Guide](graph.md) |

### Audit in CI

```yaml
- run: catude audit --project .     # fails the job on any high-severity finding
```

## Work profiles

A work profile is a set of skills, agents, commands, MCP servers and plugins for one kind of work. Full guide:
[Work profiles](profiles.md).

| Command | What it does |
|---|---|
| `catude profile list` | your profiles |
| `catude profile save NAME [--replace]` | save your current setup as a profile |
| `catude profile new NAME [--skills a,b] [--agents x] [--commands y] [--mcp m] [--plugins p@market] [--summary TEXT] [--model M]` | build a profile from items already on this machine (copied, not linked) |
| `catude profile load SOURCE [--name NAME] [--yes] [--replace] [--allow-unpinned] [--accept-risk] [--allow-programs]` | install a profile from a Rigcard: `./file.json`, `host/path`, `host`, `gh:owner/repo[@ref]` or `preset:NAME`. Validated, downloaded only from pinned sources, hash-checked and audited before you confirm |
| `catude profile switch NAME\|none` · `catude switch [NAME\|none]` | make NAME the active profile (no NAME: pick from a list) |
| `catude profile current` | the active profile |
| `catude profile show NAME` | what a profile contains |
| `catude profile check NAME [--exclusive]` | ask Claude Code itself whether every item of the profile loads |
| `catude profile use NAME [--exclusive] [--layered] [--isolated] [-- claude arguments...]` | start Claude Code with the profile. `--exclusive` skips your user settings, plugins and MCP servers; `--layered` loads it on top of your setup even if it defaults to exclusive; `--isolated` uses a separate `CLAUDE_CONFIG_DIR` with its own login |
| `catude profile desktop NAME [--config FILE] [--replace] [--yes]` | copy the profile's local MCP servers into Claude Desktop (preview, confirmation and a backup first) |
| `catude profile export NAME DIR` | a shareable bundle: `rigcard.json` and the profile's files, with hashes |
| `catude profile rm NAME` | delete a profile |
| `catude profile dir` | where profiles are stored |
| `catude claude [ARGS...]` | start Claude Code with the active profile, if any |
| `catude shell-init zsh\|bash\|fish\|powershell` | print the shell integration that makes plain `claude` use the active profile |
| `catude presets` | work profiles based on the public setups of well-known practitioners. [Presets](presets.md) |

Profiles never install hooks. MCP secrets are `${ENV_VAR}` references you set yourself.

Shell integration:

=== "zsh / bash"

    ```sh
    eval "$(catude shell-init zsh)"     # or bash; add the line to ~/.zshrc or ~/.bashrc
    ```

=== "fish"

    ```sh
    catude shell-init fish | source     # add to ~/.config/fish/config.fish
    ```

=== "PowerShell"

    ```powershell
    Invoke-Expression (& catude shell-init powershell | Out-String)   # add to $PROFILE
    ```

## Share and move a setup

| Command | What it does |
|---|---|
| `catude export [FILE]` | your setup in one file without secrets, to share it or move it to another machine |
| `catude import FILE\|URL [--name N]` | check, show and install a setup someone exported, as a work profile (audited first) |
| `catude rig export [-o FILE] [--summary TEXT] [--owner NAME] [--id URL]` | describe your setup as a Rigcard: marketplaces, plugins, skills, agents and commands (with sha256), MCP servers (variable names only, never values), safe settings. Refuses to write if a secret slipped in |
| `catude rig show SOURCE [--json]` | load someone's card and compare it with this machine. Nothing is installed: you get the official commands to run |
| `catude card [--svg FILE]` | a shareable summary of your setup: numbers only, no names, no secrets |

Format: [Export and import](rigcard.md).

## Models and catalog

| Command | What it does |
|---|---|
| `catude models [--for KIND]` | which Claude model and effort level is stronger, on public benchmarks, with cost per task. `KIND`: `coding`, `reasoning`, `"math & science"`, `agents`, `knowledge` |
| `catude models compare A B [--for KIND]` | compare two models and efforts, for example `"opus 5.5 high" "fable 5.2 medium"` |
| `catude models refresh` | download the public data again |
| `catude discover [SECTION] [QUERY]` | the curated catalog. `SECTION`: `marketplaces`, `plugins`, `mcp`, `skills`, `tools`, `practices`. [Catalog](catalog.md) |

## Install and maintenance

| Command | What it does |
|---|---|
| `catude install [--yes] [--no-ct] [--shell]` | put `catude`, `ct` and `ctlite` on your PATH. [Install](install.md) |
| `catude uninstall [--yes]` | remove them from your PATH (profiles and settings stay) |
| `catude debug` | versions, terminal, paths and state for bug reports (no secrets) |
| `catude snapshot [--live]` | render every tab as plain text, for CI, screenshots and bug reports |
| `catude version` | the version |

!!! warning "Before attaching a snapshot to a bug report"
    `catude snapshot > snapshot.txt` contains your paths and item names (secret values are already masked).
    Review it first. `catude debug` contains no secrets.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | success |
| `1` | an error, or `catude audit` found at least one high-severity finding |
| `2` | a flag or argument was not understood |
