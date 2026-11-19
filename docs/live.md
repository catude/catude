# Live: what your Claude Code sessions are doing right now

The **Live** tab (tab `3`) shows every Claude Code session running on this computer (in terminals, VS Code
or the Code tab of Claude Desktop) as a card that updates every second:

- name and title of the session, its folder, where it runs (terminal, VS Code…), model and effort level,
  Claude Code version, and how full its context window is;
- whether it is **working** or **waiting for you**, and how long ago it last did something;
- its to-do list (done / total, and the task in progress);
- a timeline of its last tool calls (`Read cart.go`, `Bash Run the tests`, `github › create_issue`) with
  a mark for done, failed or running (and for how long);
- under a running subagent (Agent / Task), what that subagent is doing;
- the first line of its last answer.

Working sessions come first and show more steps. Two columns of cards on wide terminals.

`↑` `↓` choose a session · `enter` opens its conversation with live context (Sessions tab) · `o` opens its folder.

## When a session finishes

While catude is open, in any tab, the bottom line says when a session finishes its turn and waits for you
("storefront-01 finished and is waiting for you"). Turn on **Bell** in preferences (`,`) to also hear it, and **Notification** for a system notification (Windows, macOS with `osascript`, Linux with `notify-send`). This reads
only the small `<pid>.json` files every two seconds, so it costs nothing noticeable.

If a card says that Claude Code writes its transcript in a format catude does not read yet, Claude Code
changed the format of its transcripts: update catude. The rest of the card (status, folder, version) still
comes from the session file.

## Where the information comes from

Only files on this computer, read-only:

| | |
|---|---|
| `~/.claude/sessions/<pid>.json` | one small file per running session, documented by Anthropic ("used to detect concurrent sessions and crashes"): session id, folder, version, entry point, name and status (`busy`, `idle`). catude reads only these `<pid>.json` files; **the `.key` files next to them are never opened** |
| the process itself | whether the process of that file still runs (a leftover of a crashed session is not shown) |
| the end of the session's transcript | `~/.claude/projects/<folder>/<session>.jsonl`, at most the last 768 KB: tool calls and results, model, effort, to-do list, last text; subagent transcripts that changed in the last two minutes |

Commands and arguments are redacted (tokens and keys are masked) and shortened before they are shown.
Nothing is sent anywhere. A refresh only re-reads a transcript when it has changed, so the tab costs
almost nothing while open, and it stops refreshing when you switch to another tab.
