# How catude measures context

catude never estimates tokens from characters. It uses two exact sources.

## 1. Context of a session (Sessions tab, `catude sessions`)

Claude Code writes every API response to `~/.claude/projects/<project>/<session>.jsonl`, including the `usage`
the API returned. From that file:

- **Context in use** = `input_tokens + cache_creation_input_tokens + cache_read_input_tokens` of the **last**
  main-thread response. Each request sends the whole conversation, so the last one *is* the context. Adding up
  every response measures traffic, not context.
- **De-duplication.** A streamed response is written as several lines with the same `message.id` and `requestId`.
  Counting lines inflates totals about 3×; catude keeps one per pair.
- **Subagents** (`isSidechain: true`) run in their own context and are excluded.
- **Compaction.** A `compact_boundary` entry resets the count; the number of compactions is shown.
- **Window.** 1M tokens for current models (Opus/Sonnet 4.6 and later, Fable); 200k for Claude 3 and 4.0–4.5 unless the model id carries `[1m]`.
- **Cache hit rate** = cache reads ÷ (cache reads + cache writes + uncached input), over the session.

Live view: the session you open is re-read every 2 s, so you can watch it grow while you work.

## 2. Composition of the context (`c` key, `catude context`)

What makes up those tokens before you type anything (system prompt, tool definitions, MCP tools by server, each
memory file, each skill's name and description) comes from Claude Code's own `/context` command, run headless:

```
claude -p /context
```

catude parses its tables. It takes 20–40 seconds because Claude Code starts every MCP server. If a future Claude
Code changes the format, catude reports *format not recognised* instead of guessing.

## Trimming it

In the Context tab, choose one of the biggest consumers with `↑` `↓` and press `x` to switch that skill or MCP
server off for this project only (the settings Claude Code documents: `skillOverrides`, `disabledMcpServers`,
`disabledMcpjsonServers`). `r` measures again to see the difference. Instruction files are not switched off: edit
or move them.

## What is *not* exact

- Skills only cost their `name` + `description` until they are used; manual-only skills cost nothing. The body is
  loaded when the skill runs.
- MCP tools from servers with deferred loading cost little until a tool is searched for; `/context` reports them
  separately and catude shows that split.
- The token cost of a claude.ai connector you have not authenticated cannot be measured: it has no tools until you do.
