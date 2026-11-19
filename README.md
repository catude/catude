<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo-dark.svg">
    <img alt="catude" src="assets/logo-light.svg" width="320">
  </picture>
</p>

<h3 align="center">Find out why your Claude Code usage runs out, trim what loads into every session,<br>and see everything that configures Claude Code and Claude Desktop on your machine.</h3>

<p align="center">
  A terminal app and command line. One binary, no account, no telemetry. Windows, macOS, Linux and WSL.
</p>

<p align="center">
  <a href="#install"><b>Install</b></a> ·
  <a href="#what-it-answers">What it answers</a> ·
  <a href="#command-line">Command line</a> ·
  <a href="docs/usage.md">Docs</a> ·
  <a href="docs/security.md">Security</a> ·
  <a href="docs/comparison.md">Alternatives</a>
</p>

<p align="center">
  <a href="https://github.com/catude/catude/actions/workflows/ci.yml"><img alt="Tests" src="https://github.com/catude/catude/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://github.com/catude/catude/actions/workflows/codeql.yml"><img alt="CodeQL" src="https://github.com/catude/catude/actions/workflows/codeql.yml/badge.svg"></a>
  <a href="https://github.com/catude/catude/releases"><img alt="Latest release" src="https://img.shields.io/github/v/release/catude/catude?color=D97757"></a>
  <a href="#security-you-can-check"><img alt="Signed build provenance" src="https://img.shields.io/badge/releases-signed%20provenance-D97757"></a>
  <img alt="Windows, macOS, Linux, WSL" src="https://img.shields.io/badge/runs%20on-Windows%20%C2%B7%20macOS%20%C2%B7%20Linux%20%C2%B7%20WSL-informational">
  <img alt="No telemetry" src="https://img.shields.io/badge/telemetry-none-success">
  <img alt="10 languages" src="https://img.shields.io/badge/languages-10-blueviolet">
  <img alt="MIT" src="https://img.shields.io/badge/license-MIT-blue">
</p>

<p align="center"><img alt="TL;DR: where your tokens went, switches per project, the agent map, work profiles, presets, Discover, config, audit, live sessions" src="assets/tldr.gif" width="900"></p>

> **TL;DR**
> - **`catude why`**: where your Claude Code tokens went (cache rebuilt after pauses, subagents, `claude -p`, the context before you type) and what to change.
> - **Switch things off per project**: `x` turns a skill or MCP server off for one project, with the settings Claude Code documents.
> - **Agent map**: which subagents run, who launches them, the tools they use and what they cost.
> - **Work profiles**: one set of skills, agents, MCP servers and plugins per kind of work; `ct switch` changes which one plain `claude` starts with.
> - **Presets of the best**: start from the published setups of Boris Cherny, Simon Willison, Mitchell Hashimoto, Matt Pocock and 11 more, each with its sources.
> - **Discover**: 177 plugins, 89 skills, 55 marketplaces and 25 MCP servers, checked at their source; `i` twice installs with the official `claude` command.
> - **Config**: every documented setting of Claude Code and Claude Desktop, which file wins, and 30 presets that quote their source.
> - **Audit**: 25 rules for secrets in permission rules, `curl | sh`, hidden Unicode and other risks in what you installed.
> - **Live**: what every running session is doing, with a notice (or a system notification) when one waits for you.
> - **One binary**: no account, no telemetry, reads local files only. Windows, macOS, Linux, WSL. 10 languages.

## Try it

<details open>
<summary><b>Windows</b> (PowerShell)</summary>

```powershell
irm https://raw.githubusercontent.com/catude/catude/main/scripts/install.ps1 | iex
```

Then open a new terminal and run `catude why` or `catude`.
</details>

<details>
<summary><b>Linux, macOS, WSL</b> (experimental)</summary>

```sh
curl -fsSL https://raw.githubusercontent.com/catude/catude/main/scripts/install.sh | sh
```

Then open a new terminal and run `catude why` or `catude`. The Linux and macOS builds are tested on every change but
have little real-world use so far: [please report what you find](https://github.com/catude/catude/issues/new/choose).
</details>

<details>
<summary><b>With Go</b> (any system)</summary>

```sh
go install github.com/catude/catude/cmd/catude@latest
```
</details>

Each script downloads the [latest release](https://github.com/catude/catude/releases/latest), checks its checksum and
puts `catude` and the short `ct` on your PATH, without administrator rights. More options in [Install](#install).

<p align="center"><img alt="catude: where the tokens went, the agent map, every file Claude reads and the live sessions" src="assets/demo.gif" width="900"></p>

## What it answers

| Question | Where |
|---|---|
| **Why did my usage run out this week?** | [`catude why`](#why-your-usage-runs-out), Usage tab |
| **What loads into every session before I type, and how do I trim it for one project?** | [Context, MCP and Skills tabs](#trim-what-loads-into-every-session) |
| **Which subagents run, who launches them and what do they cost?** | [Agents tab, `catude agents`](#the-agent-map) |
| **Did something get worse after an update?** | [`catude health`](#your-own-evidence-when-something-feels-worse) |
| **Is anything I installed unsafe?** | [Audit tab, `catude audit`](#audit-what-you-installed) |
| **What is each running session doing right now?** | [Live tab](#live-what-every-session-is-doing) |
| **How do I keep one setup per kind of work?** | [Work profiles](#work-profiles-one-setup-per-kind-of-work) |
| **Which file wins when a setting is in three places?** | [Config tab](#every-setting-and-which-one-wins) |

Everything comes from files already on your machine: Claude Code's transcripts and settings, and Claude Desktop's
configuration. catude changes nothing unless you press a key that does, backs up every file first, and never shows
or exports a secret.

## Install

Each line downloads the latest release, checks its checksum and puts `catude` and the short `ct` on your PATH.
No administrator rights are needed.

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/catude/catude/main/scripts/install.ps1 | iex
```

```sh
# Linux, macOS, WSL
curl -fsSL https://raw.githubusercontent.com/catude/catude/main/scripts/install.sh | sh
```

```sh
# With Go 1.24 or later
go install github.com/catude/catude/cmd/catude@latest
```

Or download the archive for your system from [Releases](../../releases). The first run offers to install itself,
with the answers already selected; `catude uninstall` removes it. Then:

```sh
catude why      # where your tokens went
catude          # the terminal app (or: ct)
```

> Works on Windows 10 and 11, macOS, Linux and WSL. The Linux and macOS builds are **experimental**: tested on every
> change, but with little real-world use so far ([Linux notes](docs/linux.md), [Windows notes](docs/windows.md)).

## Why your usage runs out

<p align="center"><img alt="catude why: subagents, cache rebuilt after pauses, one expensive conversation, claude -p runs and the context before the first message" src="assets/screenshots/cli-why.png" width="900"></p>

Counting tokens says *how much*. catude says *where they went*, from the API usage Claude Code records in your
transcripts:

- **Cache rebuilt after a pause.** The prompt cache expires after a few minutes without activity; the next message
  sends the whole conversation again. catude counts those rebuilds and what they cost.
- **Subagents.** Each one starts its own context and re-reads files.
- **Headless runs.** Scripts, hooks and tools that start `claude -p` count against the same limits.
- **The context before your first message**: system prompt, tools, MCP servers, `CLAUDE.md` and skill descriptions.
- **The most expensive conversations**, and **whether a new Claude Code version** uses more tokens per response.

A conversation resumed or forked into another folder copies its history; catude counts those responses once. The
Usage tab shows the same next to your days, models and projects, and, with
[claude-swap](https://github.com/realiti4/claude-swap), when each plan window resets and whether your pace lasts.

<p align="center"><img alt="Usage tab: where the tokens went over 7, 14 and 30 days" src="assets/quota.gif" width="900"></p>

## Trim what loads into every session

<p align="center"><img alt="Switching a skill off for one project, then an MCP server from the Context tab" src="assets/switch.gif" width="900"></p>

The Context tab asks Claude Code itself (`claude /context`) what fills the window at the start of a session in this
folder, and lists the biggest consumers you control. Select one and press `x` to switch it off **for this project
only**; the MCP and Skills tabs do the same and show each item's cost in tokens. catude uses the settings Claude Code
documents for this (`disabledMcpServers`, `disabledMcpjsonServers`, `skillOverrides`): nothing is uninstalled, and `x`
again turns it back on. For skills, `x` cycles through *name only*, *only when you type /name* and *off*.

## The agent map

<p align="center"><img alt="Agents tab: every subagent with its runs, who launched it, tools, models and cost" src="assets/agents.gif" width="900"></p>

Every subagent Claude Code can use here (yours, the project's, plugins' and the built-in ones) joined with what it
really did in the last 30 days: how often it ran, who launched it (your conversation or another agent), the tools it
used, the models, the tokens and an estimated cost. Agents you defined but never use are flagged, because their
descriptions load into every session. `catude agents` prints the same map.

## Your own evidence when something feels worse

<p align="center"><img alt="catude health: tool calls, failures, API retries, compactions, cache hit and tokens per response by day and version" src="assets/screenshots/cli-health.png" width="900"></p>

`catude health` shows, day by day and per Claude Code version: tool calls and how many failed, API errors Claude
Code retried, compactions, the share of input read from the cache, and fresh tokens per response. When a session
feels worse, this is what changed on your machine.

## Audit what you installed

<p align="center"><img alt="Audit tab: findings with the fix for each" src="assets/screenshots/audit.png" width="900"></p>

Skills, hooks, plugins and MCP servers run with your permissions. 25 rules look for what usually goes wrong:
passwords or tokens saved inside permission rules (what "don't ask again" can leave in `settings.local.json`),
commands that delete data or rewrite history allowed without asking, plaintext secrets in MCP configs,
`curl | sh`, prompt-injection phrases, hidden Unicode, unpinned `npx`, an `AGENTS.md` that Claude Code silently
ignores. Every finding says why and what to do; `catude audit` exits with code 1 on high findings, so it fits in CI.
[Rules](docs/security.md#audit-rules)

## Live: what every session is doing

<p align="center"><img alt="Live tab: running sessions with their current tool, subagents, to-do list and context" src="assets/live.gif" width="900"></p>

Each Claude Code session running on the machine (terminal, VS Code or Claude Desktop) as a card updated every
second: working or waiting for you, the tool running and for how long, its last steps, what its subagents are doing,
its to-do list, the model and effort, and how full the context is. In any tab, catude tells you when a session
finishes and waits for you; turn on the bell or a system notification in preferences. [How it works](docs/live.md)

## Work profiles: one setup per kind of work

<p align="center"><img alt="Profiles tab: frontend, backend and writing work profiles" src="assets/screenshots/profiles.png" width="900"></p>

A work profile is a set of skills, agents, commands, MCP servers and plugins for one kind of work: frontend,
backend, writing. Press `n` to build one, `a` to save your current setup, `s` to make it the one plain `claude`
starts with. Start from one of the [15 presets below](#presets-and-where-they-come-from), load one from a file or
URL (validated, pinned, hash-checked and audited first), or copy a profile's MCP servers into Claude Desktop with
`catude profile desktop NAME`. [Guide](docs/profiles.md)

## Presets, and where they come from

<p align="center"><img alt="Presets gallery: work profiles based on the public setups of known practitioners" src="assets/screenshots/presets.png" width="900"></p>

Two kinds of presets, and every one says where it comes from. Nothing is invented: each preset quotes or links the
public source it is based on, and catude shows that source before you apply anything.

**Work profiles based on what practitioners published** (Profiles tab, `p`; `catude presets`). Only files under an
open licence and pinned to a commit are installed; ideas without published files become a profile you complete.

<table>
<tr>
<td width="33%" valign="top"><pre> ███  ████ 
█   █ █   █
█████ ████ 
█   █ █  █ 
█   █ █   █</pre><b>Armin Ronacher</b><br><sub>Simple, observable tooling (Makefiles, logs, CLIs) over clever automation</sub><br><br><a href="https://github.com/mitsuhiko/agent-stuff">mitsuhiko/agent-stuff</a><br><a href="https://lucumr.pocoo.org/2025/6/12/agentic-coding/">lucumr.pocoo.org</a><br><br><code>catude profile load preset:armin-ronacher</code></td>
<td width="33%" valign="top"><pre>████   ████
█   █ █    
████  █    
█   █ █    
████   ████</pre><b>Boris Cherny</b><br><sub>A mostly vanilla setup run in parallel; give Claude a way to verify its own work</sub><br><br><a href="https://threadreaderapp.com/thread/2007179832300581177">his threads on how the Claude Code team works</a><br><br><code>catude profile load preset:boris-cherny-claude-code</code></td>
<td width="33%" valign="top"><pre>████  █   █
█   █ █   █
█   █ █████
█   █ █   █
████  █   █</pre><b>Dex Horthy</b><br><sub>The whole workflow built around managing context: research, plan, implement</sub><br><br><a href="https://github.com/humanlayer/advanced-context-engineering-for-coding-agents/blob/main/ace-fca.md">humanlayer/advanced-context-engineering-for-coding-agents</a><br><a href="https://github.com/humanlayer/12-factor-agents">humanlayer/12-factor-agents</a><br><br><code>catude profile load preset:dex-horthy</code></td>
</tr>
<tr>
<td width="33%" valign="top"><pre> ████ █   █
█     █   █
█  ██ █████
█   █ █   █
 ████ █   █</pre><b>Geoffrey Huntley</b><br><sub>The "Ralph" loop: the same prompt fed to a fresh agent until the work is done</sub><br><br><a href="https://ghuntley.com/ralph/">ghuntley.com/ralph</a><br><a href="https://github.com/ghuntley/how-to-build-a-coding-agent">ghuntley/how-to-build-a-coding-agent</a><br><br><code>catude profile load preset:geoffrey-huntley-ralph</code></td>
<td width="33%" valign="top"><pre>█   █ ████ 
█   █ █   █
█████ ████ 
█   █ █  █ 
█   █ █   █</pre><b>Harper Reed</b><br><sub>Spec, then plan, then small test-driven steps</sub><br><br><a href="https://github.com/harperreed/dotfiles/tree/master/.claude">harperreed/dotfiles</a><br><a href="https://harper.blog/2025/02/16/my-llm-codegen-workflow-atm/">harper.blog</a><br><br><code>catude profile load preset:harper-reed</code></td>
<td width="33%" valign="top"><pre>█████
  █  
  █  
  █  
█████</pre><b>IndyDevDan</b><br><sub>Reusable prompts, subagents and hooks instead of vibe coding</sub><br><br><a href="https://github.com/disler/claude-code-hooks-mastery">disler/claude-code-hooks-mastery</a><br><br><code>catude profile load preset:indydevdan</code></td>
</tr>
<tr>
<td width="33%" valign="top"><pre>  ███ █   █
    █ █   █
    █ █   █
█   █  █ █ 
 ███    █  </pre><b>Jesse Vincent</b><br><sub>Superpowers: mandatory skills for brainstorming, planning and TDD</sub><br><br><a href="https://github.com/obra/superpowers">obra/superpowers</a><br><a href="https://blog.fsck.com/2025/10/09/superpowers/">blog.fsck.com</a><br><br><code>catude profile load preset:jesse-vincent-superpowers</code></td>
<td width="33%" valign="top"><pre>█   █ █   █
█  █  █  █ 
███   ███  
█  █  █  █ 
█   █ █   █</pre><b>Kieran Klaassen</b><br><sub>Compound engineering: every task makes the next one easier</sub><br><br><a href="https://github.com/EveryInc/compound-engineering-plugin">EveryInc/compound-engineering-plugin</a><br><br><code>catude profile load preset:kieran-klaassen-compound-engineering</code></td>
<td width="33%" valign="top"><pre>█     █████
█       █  
█       █  
█       █  
█████   █  </pre><b>Lauren Tan</b><br><sub>pstack: make one agent trustworthy on a whole problem before going wide</sub><br><br><a href="https://github.com/cursor/plugins/tree/main/pstack">cursor/plugins: pstack</a><br><br><code>catude profile load preset:lauren-tan-pstack</code></td>
</tr>
<tr>
<td width="33%" valign="top"><pre>█   █ ████ 
██ ██ █   █
█ █ █ ████ 
█   █ █    
█   █ █    </pre><b>Matt Pocock</b><br><sub>Small, composable skills that keep you in control</sub><br><br><a href="https://github.com/mattpocock/skills">mattpocock/skills</a><br><br><code>catude profile load preset:matt-pocock-skills</code></td>
<td width="33%" valign="top"><pre>█   █ █   █
██ ██ █   █
█ █ █ █████
█   █ █   █
█   █ █   █</pre><b>Mitchell Hashimoto</b><br><sub>Real agents, not chat; when an agent fails, fix the harness</sub><br><br><a href="https://github.com/ghostty-org/ghostty/blob/main/AGENTS.md">ghostty AGENTS.md</a><br><a href="https://mitchellh.com/writing/my-ai-adoption-journey">mitchellh.com</a><br><br><code>catude profile load preset:mitchell-hashimoto</code></td>
<td width="33%" valign="top"><pre>████   ████
█   █ █    
████   ███ 
█         █
█     ████ </pre><b>Peter Steinberger</b><br><sub>Short prompts, screenshots and many agents in parallel</sub><br><br><a href="https://github.com/steipete/agent-scripts">steipete/agent-scripts</a><br><a href="https://steipete.me/posts/just-talk-to-it">steipete.me</a><br><br><code>catude profile load preset:peter-steinberger</code></td>
</tr>
<tr>
<td width="33%" valign="top"><pre> ████ █   █
█     █   █
 ███  █ █ █
    █ ██ ██
████  █   █</pre><b>Simon Willison</b><br><sub>Agentic engineering patterns: spend the time saved on testing and review</sub><br><br><a href="https://simonwillison.net/guides/agentic-engineering-patterns/">simonwillison.net</a><br><br><code>catude profile load preset:simon-willison-agentic-engineering</code></td>
<td width="33%" valign="top"><pre> ████ █   █
█      █ █ 
 ███    █  
    █   █  
████    █  </pre><b>Steve Yegge</b><br><sub>Beads: a dependency-aware issue graph as the agent's memory</sub><br><br><a href="https://github.com/gastownhall/beads">gastownhall/beads</a><br><br><code>catude profile load preset:steve-yegge</code></td>
<td width="33%" valign="top"><pre>█████ ████ 
  █   █   █
  █   ████ 
  █   █   █
  █   ████ </pre><b>Thorsten Ball</b><br><sub>An agent is an LLM, a loop and tools: brief it well</sub><br><br><a href="https://ampcode.com/how-to-build-an-agent">ampcode.com</a><br><br><code>catude profile load preset:thorsten-ball</code></td>
</tr>
</table>

**30 configuration presets** (Config tab, `p`). Each one is a small group of documented settings for one goal, with
the exact sentence of its source quoted next to it, and a preview of what it would change in your file:

| Goal | Presets | Sources |
|---|---|---|
| Privacy | no telemetry or feedback uploads, block non-essential traffic, keep transcripts 7 days | [data usage](https://code.claude.com/docs/en/data-usage), [settings reference](https://code.claude.com/docs/en/settings-reference) |
| Security | protect `.env` and credentials, strip credentials from subprocesses, no bypass mode, manual mode, audit setting changes | [Trail of Bits claude-code-config](https://github.com/trailofbits/claude-code-config), [permission modes](https://code.claude.com/docs/en/permission-modes) |
| Sandbox | hide AWS/SSH credentials from commands, sandboxed Bash with fewer prompts, strict sandbox | [sandboxing](https://code.claude.com/docs/en/sandboxing) |
| Lockdown | the enterprise-like strict example, and the lax one for long unattended runs | [anthropics/claude-code example settings](https://github.com/anthropics/claude-code/tree/main/examples/settings) |
| Cost | Sonnet as default, subagents on Haiku, no background requests while idle | [costs](https://code.claude.com/docs/en/costs), [model config](https://code.claude.com/docs/en/model-config), [subagents](https://code.claude.com/docs/en/sub-agents) |
| Git, UI, notifications, updates, Windows | ask before `git push`, no attribution, quieter UI, a bell or desktop notification when Claude needs you, stable channel, PowerShell tool | [hooks guide](https://code.claude.com/docs/en/hooks-guide), [setup](https://code.claude.com/docs/en/setup), [env vars](https://code.claude.com/docs/en/env-vars) |
| Telemetry | your own usage metrics on a local Prometheus endpoint | [monitoring usage](https://code.claude.com/docs/en/monitoring-usage) |

The **Discover** catalog follows the same rule: every plugin, skill and MCP server links to its repository and
licence, and installs only through the official `claude plugin` and `claude mcp` commands.

Presets reflect what each person or organisation published, at the link shown; they are not affiliated with or
endorsed by them. A correction or a new source is a welcome [pull request](CONTRIBUTING.md).

## Every setting, and which one wins

<p align="center"><img alt="Effective settings: the value Claude Code uses and the file it comes from" src="assets/screenshots/config-effective.png" width="900"></p>

The Config tab lists every documented setting of every Claude app found (more than 300 settings and 400 environment
variables), set or not, with what it does, its allowed values and its default. **Effective settings** shows, for each
key of this project, the value Claude Code really uses, the file it comes from and what it overrides. Toggle, cycle
or type a value; catude backs the file up, keeps its key order and refuses values that look like secrets. Press `p`
for 30 [presets](docs/config.md#presets) (no telemetry, protect `.env`, lower cost, sandboxed Bash…), each quoting
its source. [Details](docs/config.md)

## And also

<table>
<tr>
<td width="50%"><img alt="Sessions" src="assets/screenshots/sessions.png"><br><b>Sessions</b>: every conversation of the terminal and of Claude Desktop's Code tab, with its exact context, an estimated cost and a summary without AI (your first message, files changed, tools). <code>n</code> names it, <code>c</code> continues it in Claude Code, <code>f</code> searches every conversation.</td>
<td width="50%"><img alt="Models" src="assets/screenshots/models.png"><br><b>Models</b>: which Claude model and effort level is stronger for a kind of work, on public benchmarks, with cost per task and advice on the one you use. No account or key.</td>
</tr>
<tr>
<td width="50%"><img alt="Graph" src="assets/screenshots/graph.png"><br><b>Graph</b>: apps, config files, instructions, skills, hooks, MCP servers, the programs they start and running processes, as a tree; <code>w</code> opens it as a diagram.</td>
<td width="50%"><img alt="Discover" src="assets/screenshots/discover-tag.png"><br><b>Discover</b>: 177 plugins, 89 skills, 55 marketplaces and 25 MCP servers, checked at their source and tagged; installed with the official <code>claude plugin</code> command.</td>
</tr>
</table>

| | |
|---|---|
| **Instructions** | `catude instructions` says which `CLAUDE.md`, `CLAUDE.local.md` and `AGENTS.md` Claude Code reads here, and the project's auto memory. A `CLAUDE.local.md` silently stops `AGENTS.md`: `--fix` adds the import, and copies `.agents/skills` (which Claude Code never reads) into `.claude/skills` |
| **Every extension, explained** | Skills, agents, commands, hooks, MCP servers and plugins from every scope, each tab with one sentence on what they are; `e` edits a file after a backup |
| **Storage** | Transcripts, caches and Claude Desktop data with their sizes, safe cleaning, and a warning before Claude Code deletes old transcripts |
| **Export and import** | Your setup in one file without secrets, to move it or share it; importing shows what it brings and audits it first ([format](docs/rigcard.md)) |
| **Inside Claude Code** | The repository is a plugin marketplace: `/catude:doctor`, `/catude:audit`, `/catude:sessions`, `/catude:models` |
| **10 languages** | English, Español, 日本語, 简体中文, 繁體中文, 한국어, Deutsch, Français, Português (Brasil), Русский, with the terms of Anthropic's localized docs ([how](docs/translations.md)) |
| **Any terminal** | Detects what your terminal and font can draw, themes including high contrast, reduced motion, a lite view (`ctlite`) ([accessibility](docs/accessibility.md)) |

<details>
<summary><b>A walk through every tab</b></summary>

<p align="center"><img alt="A walk through all the tabs" src="assets/tour.gif" width="900"></p>

| | |
|---|---|
| ![Overview](assets/screenshots/overview.png) Overview | ![Config](assets/screenshots/config.png) Config |
| ![Live](assets/screenshots/live.png) Live | ![Context](assets/screenshots/context.png) Context |
| ![Usage](assets/screenshots/usage.png) Usage | ![Models](assets/screenshots/models.png) Models |
| ![Profiles](assets/screenshots/profiles.png) Profiles | ![Sessions](assets/screenshots/sessions.png) Sessions |
| ![Graph](assets/screenshots/graph.png) Graph | ![Skills](assets/screenshots/skills.png) Skills |
| ![Agents](assets/screenshots/agents.png) Agents | ![Commands](assets/screenshots/commands.png) Commands |
| ![MCP](assets/screenshots/mcp.png) MCP | ![Plugins](assets/screenshots/plugins.png) Plugins |
| ![Hooks](assets/screenshots/hooks.png) Hooks | ![Discover](assets/screenshots/discover.png) Discover |
| ![Tools](assets/screenshots/tools.png) Tools | ![Storage](assets/screenshots/storage.png) Storage |
| ![Audit](assets/screenshots/audit.png) Audit | ![Preferences](assets/screenshots/preferences.png) Preferences |
| ![catude agents](assets/screenshots/cli-agents.png) `catude agents` | ![Help](assets/screenshots/help.png) Help (`?`) |
</details>

## Command line

Everything in the app is also a command, and every listing accepts `--json`.

```sh
catude why                            # where your tokens went in the last 14 days, and what to change
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

Inside Claude Code: `/plugin install catude --marketplace catude/catude` adds `/catude:doctor`,
`/catude:audit`, `/catude:sessions` and `/catude:models`. Keys and every command: [docs/usage.md](docs/usage.md).

## Privacy and safety

1. **Local and read-only by default.** catude reads files on your machine. Every write is explicit, backed up
   first and reversible.
2. **No telemetry, account or API key.** It goes online only when you ask for something that needs it: importing
   a URL you name, installing through the official `claude` commands, or the public model data of the Models tab.
3. **Secrets stay masked.** Values that look like credentials are shown as `••••` and never exported. The session
   key files of Claude Code are never opened.
4. **Official commands and documented settings.** Plugins and MCP servers are installed with `claude plugin` and
   `claude mcp`; per-project switches use the settings Anthropic documents.
5. **Measured numbers.** When a number is an estimate (a cost at list prices), catude says so.

### Security you can check

| | |
|---|---|
| **Every change is scanned** | [CodeQL](https://github.com/catude/catude/actions/workflows/codeql.yml) (code scanning), `govulncheck` (known vulnerabilities in Go and its dependencies) and `gitleaks` (secrets) run on every pull request and push, next to the tests on Windows, macOS and Linux ([ci.yml](.github/workflows/ci.yml)) |
| **Protected main branch** | Changes arrive through pull requests that must pass every check; no force pushes |
| **Dependencies kept current** | Dependabot opens updates for Go modules and GitHub Actions; releases are built with the latest Go and its security fixes |
| **Verifiable downloads** | Every release has a `checksums.txt` and a signed build provenance attestation: `gh attestation verify catude_*.zip --repo catude/catude` proves a file was built here, from this source |
| **Small surface** | One static binary, no runtime, no telemetry, no account; it goes online only for what you ask |

[Security model](docs/security.md) · [Report a vulnerability](SECURITY.md) (privately, through GitHub)

## FAQ

<details>
<summary><b>Are the token numbers the same as my plan limits?</b></summary>

No. Plan limits are computed by Anthropic and also count your other devices and claude.ai. catude shows what this
machine recorded, which is what you can act on: the same responses, de-duplicated, with where they went.
</details>

<details>
<summary><b>Does it change my Claude configuration?</b></summary>

Only when you press a key that does so (switch something off, change a setting, install a profile, clean a folder).
Before each change it copies the file to `<user cache>/catude/backups/` and it leaves the rest of the file untouched.
</details>

<details>
<summary><b>Is it an official Anthropic tool?</b></summary>

No. catude is an independent open-source project, not affiliated with or endorsed by Anthropic. The paths and
settings it reads come from Anthropic's documentation.
</details>

<details>
<summary><b>I use the Microsoft Store version of Claude Desktop. Does it work?</b></summary>

Yes. The Store app reads a virtualised copy of its configuration under `%LOCALAPPDATA%\Packages\Claude_*\…`, so
editing `%APPDATA%\Claude\` has no effect. catude shows and edits the copy the app actually reads.
</details>

<details>
<summary><b>Where do the screenshots come from?</b></summary>

From an invented machine (user `alex`, invented projects, sessions and usage) built by
[scripts/demo](scripts/demo/README.md). The command-line pictures are the real output of catude on that machine.
</details>

## Alternatives

Other tools cover parts of this, among them [ccusage](https://github.com/ccusage/ccusage) (usage reports),
[cc-switch](https://github.com/farion1231/cc-switch) (providers), [claude-code-templates](https://github.com/davila7/claude-code-templates)
(a catalog) and [claude-swap](https://github.com/realiti4/claude-swap) (accounts, which catude uses for plan limits).
[docs/comparison.md](docs/comparison.md) says what each does better.

## Contributing

Pull requests are very welcome, and these need no Go: a [configuration preset](internal/claudecfg/config-presets.json)
with its public source, a [catalog entry](internal/catalog/catalog.json), a [practitioner preset](internal/presets/presets.json),
a translation fix backed by a source in the [glossaries](docs/translations.md), or a bug report with the output of
`catude debug` (it contains no secrets). Code: [CONTRIBUTING.md](CONTRIBUTING.md). Roadmap: [docs/roadmap.md](docs/roadmap.md).

---

<sub>catude is an independent project, not affiliated with or endorsed by Anthropic. "Claude" is a trademark of
Anthropic, PBC. Benchmark data © Epoch AI (CC BY 4.0) and the projects it credits. MIT © catude contributors.</sub>
