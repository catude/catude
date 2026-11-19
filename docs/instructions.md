---
description: >-
  Which CLAUDE.md, CLAUDE.local.md and AGENTS.md files Claude Code reads in a folder, the project's auto
  memory, and a fix for the files Claude Code silently skips.
---

# Instructions files

```sh
catude instructions          # what Claude Code reads here
catude instructions --fix    # make it read what it skips (asks first)
```

`catude instructions` follows the rules Anthropic documents ([memory](https://code.claude.com/docs/en/memory))
and prints, for the current folder:

- the **project instructions mode** and where it is set;
- every `CLAUDE.md`, `.claude/CLAUDE.md`, `CLAUDE.local.md` and `AGENTS.md` in this folder or above it, each
  marked as read or skipped, with the reason;
- skills in `.agents/skills`, which Claude Code never reads;
- the project's **auto memory** (the notes Claude keeps for this project) and whether `MEMORY.md` is over the
  part that loads (the first 200 lines or 25 KB).

## When `AGENTS.md` is skipped

| Mode (`/config`, "Project instructions") | What Claude Code reads |
|---|---|
| `claude-md-or-agents-md` (default) | `AGENTS.md` only when there is no `CLAUDE.md`, `.claude/CLAUDE.md` or `CLAUDE.local.md` in the folder or above it |
| `claude-md-and-agents-md` | both |
| `claude-md` | only `CLAUDE.md` |
| `managed-only` | neither |

`AGENTS.local.md`, `AGENTS.override.md` and anything under `.agents/` are never read. So a personal
`CLAUDE.local.md` silently stops a shared `AGENTS.md` in the default mode.

## What `--fix` does

- Adds an `@AGENTS.md` import to the instruction file that stops it (the documented way to share one file with
  other coding tools).
- Copies each skill in `.agents/skills` into `.claude/skills`, when no skill with that name exists there.

It shows every change and asks first (`--yes` skips the question). Nothing is written until you confirm.

The [audit](security.md#audit-rules) also reports an `AGENTS.md` that Claude Code ignores.
