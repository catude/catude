# Storage, scratchpads and retention

The **Storage** tab shows what Claude Code, Claude Desktop and their scratchpads keep on disk. The list
appears at once (with the sizes measured last time); sizes are measured again in the background.

## What is listed

| Group | Items |
|---|---|
| Claude Code | session transcripts (`projects/`), checkpoints (`file-history/`), plans, debug logs, shell snapshots, session environments, paste and image caches, legacy folders (`todos/`, `statsig/`, `logs/`), caches, background tasks, usage data, backups, plugins, IDE lock files, agent memory, your skills/agents/commands, prompt history, `~/.claude.json` and any stale copies of it. Folders catude does not know are listed too, never hidden |
| Scratchpads | Claude Code's temporary folders: `%TEMP%\claude` on Windows, `/tmp/claude-<uid>` on Linux and macOS, or `$CLAUDE_CODE_TMPDIR` |
| Claude Desktop | logs, Electron caches, installed extensions and the Cowork virtual machine, for every Desktop install found (installer, Microsoft Store, Linux, macOS) |
| catude | edit backups and the trash of removed sessions |

`enter` on the transcripts row jumps to Sessions; `o` opens any folder in your file manager.

## Cleaning

Only items marked ✓ can be cleaned from catude, with `c` pressed twice:

- debug logs, plans, shell snapshots, session environments, paste/image caches, legacy folders, caches;
- scratchpads (each session folder is judged on its own, so a running session's scratchpad is kept);
- Claude Desktop logs and Electron caches (close Claude Desktop first);
- stale copies of `~/.claude.json`.

Whatever changed in the last hour is kept, so a running session is never disturbed. Transcripts, checkpoints,
plugins, backups, credentials, settings, `CLAUDE.md`, skills and `~/.claude.json` are never deleted from this
tab. To remove a conversation, use Sessions → `D D`, which moves it to a trash folder with instructions to
restore it.

## Automatic deletion of transcripts

Claude Code deletes transcripts (and their checkpoints and other per-session data) older than
`cleanupPeriodDays` (30 days unless you set it in `~/.claude/settings.json`). It does so silently. The Storage
tab shows the current value and how many sessions will be deleted within a week, so you can export them first
(Sessions → `x`) or raise the value:

```json
{ "cleanupPeriodDays": 365 }
```

## `~/.claude.json`

It holds your account, user MCP servers and per-project state, and grows with every project. Above ~8 MiB
Claude Code slows down. catude shows its size; never delete it. Claude Code keeps backups in
`~/.claude/backups/`.
