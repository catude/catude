---
description: >-
  Let Claude or another AI assistant explain catude to you. The docs are published as llms.txt,
  llms-full.txt and plain Markdown per page, with ready-to-paste prompts.
---

# Ask your AI about catude

These docs are written for people and for AI assistants. You can hand them to Claude (or any assistant) and
ask it to walk you through catude, answer a question, or tell you which command fits what you want to do.

## The fastest way

Paste this into Claude:

```text
Read https://catude.github.io/catude/llms-full.txt and explain catude to me step by step.
I use Claude Code on <Windows / macOS / Linux>. Start with how to install it and the
first three commands I should try, and tell me what each one will show me.
```

Or open it directly: [Ask Claude about catude](https://claude.ai/new?q=Read%20https%3A%2F%2Fcatude.github.io%2Fcatude%2Fllms-full.txt%20and%20explain%20catude%20to%20me%20step%20by%20step.%20Start%20with%20how%20to%20install%20it%20and%20the%20first%20three%20commands%20I%20should%20try.){ .md-button .md-button--primary }

## On every page

At the top of each page:

- **Copy page as Markdown** puts the page on your clipboard, ready to paste into any assistant.
- **View as Markdown** opens the plain Markdown version (add `index.md` to any page address).
- **Ask Claude about this page** opens Claude with a prompt that points to that page.

## Machine-readable files

| File | What it is |
|---|---|
| [`llms.txt`](https://catude.github.io/catude/llms.txt) | the index of these docs in the [llms.txt](https://llmstxt.org/) format: a short description of catude and a link to the Markdown of every page |
| [`llms-full.txt`](https://catude.github.io/catude/llms-full.txt) | every page in one Markdown file, for assistants that read one URL |
| `<page>/index.md` | the Markdown of one page |
| [`sitemap.xml`](https://catude.github.io/catude/sitemap.xml) | every page, for crawlers |

## Inside Claude Code

If you already use Claude Code, the catude plugin lets Claude run catude on your machine and explain the result
for your own setup:

```text
/plugin install catude --marketplace catude/catude
```

Then ask with `/catude:doctor`, `/catude:audit`, `/catude:sessions` or `/catude:models`. See
[Install](install.md#inside-claude-code).

You can also ask Claude Code directly, for example:

```text
Run `catude why` and explain where my tokens went this month and what I should change first.
```

```text
Run `catude audit --json` and go through the high findings with me, one by one.
```

```text
Run `catude context --json` in this project and tell me which skills or MCP servers I could switch off here.
```

!!! note "What the assistant sees"
    catude's output masks secrets, and `--json` never includes raw file contents. Still, read what you paste:
    command output contains your paths and the names of your projects, skills and servers.

## Good questions to ask

- "Why does my Claude Code usage run out so fast, and how would catude help me find out?"
- "How do I switch an MCP server off for just one project with catude?"
- "What is a work profile in catude and when should I make one?"
- "Which catude command should I add to CI to catch unsafe Claude Code configuration?"
- "Is catude safe to run? What does it read and what does it never touch?"
