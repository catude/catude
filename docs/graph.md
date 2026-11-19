# Graph: everything your Claude uses

The **Graph** tab (and `catude graph`) shows your whole setup as one picture of things and how they relate.

| Things | Relations |
|---|---|
| Claude Code, Claude Desktop, your claude.ai account | Claude Desktop's Code tab *uses* Claude Code; both are *signed in to* the account |
| Configuration files of each app | an app *reads* a file |
| Instructions (`CLAUDE.md` and the organisation's) | Claude Code *loads instructions* |
| Skills, agents, commands | a file, a plugin or the account *has* them |
| Hooks | a settings file *defines* a hook; the hook *runs* a tool |
| MCP servers | a file *defines* a server; the server *starts* a program, *needs* variables, *connects to* a host |
| Plugins and marketplaces | Claude Code *enables* a plugin; a marketplace *offers* it |
| Tools (node, npx, uvx, python, docker, rtk…) | started by MCP servers and hooks; missing ones are flagged |
| Work profiles | a profile *includes* skills, agents, MCP servers |
| Folders on disk | Claude Code *stores* transcripts, checkpoints, caches (sizes from the last Storage measurement) |
| Processes running now | Claude *runs* a session; a session *started* an MCP server, which *started* its interpreter… |

Things that need attention are marked `▲`: a tool something needs but is not installed, an MCP server that
failed or needs authentication, a secret written in plain text in a config file.

## In the TUI

The graph is a tree: branches (`├─`, `└─`) show what each node uses, and a scrollbar on the right with
the position (`12–40 of 230`) shows how much is left above and below.

`↑` `↓` move · `enter` / `space` opens or closes a node · `+` opens · `-` closes it or jumps to its parent ·
`E` / `C` expand or collapse everything · `←` `→` (or `tab` / `shift+tab`) change tab, as everywhere · `/` finds any node by name or kind · `w` opens the graph as a
diagram in your browser (`~/catude-graph.html`, drawn by Mermaid, which the page loads from jsDelivr) and
saves it for GitHub as `~/catude-graph.md` · `r` rebuilds it.

The tree appears at once: files, settings, skills, MCP servers and tools first (tools are located, not
started), then the running processes are added in the background. The right side lists what the selected node uses and
what uses it. Nodes that appear in several places are expanded once and marked "shown above".

## From the command line

```sh
catude graph                 # indented outline
catude graph mermaid         # Mermaid flowchart (GitHub and most Markdown viewers draw it)
catude graph mermaid-full    # without grouping long lists
catude graph html > rig.html # a web page that draws it
catude graph dot | dot -Tsvg > rig.svg   # Graphviz
catude graph --json          # nodes and edges, for your own tools
```

## Privacy

The graph never holds secret values: MCP servers contribute only the names of their variables, and process
command lines are redacted. Mermaid and DOT exports carry labels only (no paths of your projects and no
command lines), but they do contain the names of your skills, servers and hosts: review before sharing.
`catude card` is the numbers-only alternative.
