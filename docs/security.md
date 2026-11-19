# Security model

catude looks at the files that decide what Claude can read, run and send. It is built so that looking is safe.

## What catude does and does not do

| | |
|---|---|
| **Reads** | Claude Code and Claude Desktop configuration, skills, agents, commands, hooks, plugins and session transcripts, with size limits. A corrupt file becomes a visible problem, never a crash |
| **Never reads** | `.credentials.json`, the OS keychain, claude-swap's store, or folders that look like backups/credential stores |
| **Writes** | Only when you press `e` on a file you own (after a backup to `<user cache>/catude/backups/`), or `rig export -o FILE` |
| **Runs** | `claude -p /context` (`c`), `claude mcp list` (`m`, `--live`), `cswap list/switch` (Profiles), and, only after you press `i` twice, an official `claude plugin …` / `claude mcp add …` command or the pinned claude-swap install. Always without a shell, never with values you did not see |
| **Network** | None of its own, except fetching a Rigcard you name, and the public model data of the Models tab (epoch.ai, models.dev: plain downloads, no credentials, once a day when you open it). No telemetry, no update checks |
| **Secrets** | Masked everywhere (TUI, CLI, JSON, Rigcards): by variable or header name, by known credential shapes (Anthropic, OpenAI, GitHub, Slack, AWS, Google, JWT, private keys…) and by entropy. `${VAR}` references are recognised as safe |

Catalog commands are checked before running: only `claude plugin marketplace add`, `claude plugin install` and
`claude mcp add` are allowed, shell metacharacters are refused, and commands with placeholders such as
`YOUR_TOKEN` are shown for you to run yourself, never executed.

## Audit rules

`catude audit` and the **Audit** tab. Severity: **high** (exit code 1), **warn**, **info**.

| Rule | Severity | Looks for |
|---|---|---|
| `remote-code-execution` | high | `curl … \| sh`, `wget … \| bash`, `irm … \| iex` and similar in skills, agents, commands |
| `hook-remote-code` | high | the same inside a hook command |
| `destructive-command` | high | `rm -rf /`, `git push --force`, `DROP TABLE`, `Remove-Item -Recurse -Force`… in instructions |
| `hook-destructive` | high | destructive commands inside a hook |
| `prompt-injection-phrase` | high | "ignore previous instructions" and similar outside quotes |
| `hidden-unicode` | high | bidi overrides, zero-width and tag characters that hide text from humans |
| `credential-exfiltration` | high | instructions that read credential files or env vars and send them out |
| `secret-in-text` | high | a credential-shaped value inside a skill, agent, command or memory file |
| `secret-in-hook` | high | a credential inside a hook command |
| `project-hook` | info | hooks defined by the project (they run for anyone who opens it) |
| `mcp-plaintext-secret` | high | tokens written in `env` or `headers` of an MCP server instead of `${VAR}` |
| `mcp-secret-in-args` | high | tokens passed as command-line arguments (visible in the process list) |
| `mcp-plain-http` | warn | remote MCP over `http://` |
| `mcp-unpinned-package` | warn | `npx` / `uvx` / `pipx run` without an exact version |
| `mcp-inline-code` | high | an MCP server whose command is a shell or interpreter running code from the command line (`sh -c`, `bash -lc`, `cmd /c`, `powershell -EncodedCommand`, `python -c`, `node -e`, `env … sh -c`) or a Windows script host (`mshta`, `rundll32`, `regsvr32`…) |
| `mcp-command-not-found` | warn | the program an MCP server starts (npx, uvx, docker…) is not installed or not on PATH, so it fails silently |
| `claude-json-large` | warn | `~/.claude.json` above 8 MiB (Claude Code slows down) |
| `tls-verification-disabled` | warn | `NODE_TLS_REJECT_UNAUTHORIZED=0`, `*TLS*VERIFY*=false`, `--insecure`, `-k` |
| `bypass-permissions` | high | `defaultMode: bypassPermissions` in settings |
| `broad-shell-allow` | warn | `Bash`, `Bash(*)` or `PowerShell(*)` in `permissions.allow` |
| `secret-in-permission` | high | a password or token inside a permission rule, usually saved by "don't ask again" in `settings.local.json` (shown masked) |
| `destructive-allow` | warn | commands that delete data or rewrite history (`rm`, `sudo`, `git push --force`, `git reset --hard`, `DROP TABLE`…) allowed without asking |
| `invalid-settings` | high | a settings file that is not valid JSON (Claude Code may ignore it) |
| `invalid-definition` | warn | a skill, agent, command or memory file with broken frontmatter |
| `third-party-marketplace` | info | marketplaces outside Anthropic's: review what their plugins run |
| `stale-config-copies` | warn | leftover `~/.claude.json` copies that may hold old tokens |

Findings are heuristics. A finding is a prompt to look, not a verdict; a clean audit is not a guarantee.

## Hardening in detail

- **Terminal escape injection.** Every frame of the interface and every line of command-line output passes
  through one filter: only colour sequences survive (and never "conceal"); window-title, clipboard (OSC 52)
  and hyperlink sequences, cursor and screen control, C0/C1 control characters and invalid UTF-8 are removed.
  Command-line output carries no escape sequences and no carriage returns at all.
- **Headless Claude only in trusted folders.** To measure context or check MCP health catude runs
  `claude -p /context` and `claude mcp list`. Headless mode skips Claude Code's trust prompt, so in a folder you
  have not trusted (a repository you just cloned) catude runs them from an empty neutral folder instead and
  says so. Child processes get `NoDefaultCurrentDirectoryInExePath=1`, so on Windows a repository cannot
  replace `node`, `git` or `claude` with its own copy.
- **Archives** (profiles from GitHub): regular files only, no `..`, absolute paths, drive letters, alternate data
  streams, Windows device names (`CON`, `NUL`, `COM1`…), names ending in a dot or space, or names that differ
  only in case; caps on unpacked size and entries; nothing is marked executable.
- **Network** (Rigcards): https only; the resolved address must be public (private, loopback, link-local,
  CGNAT, documentation, benchmarking, reserved, 6to4 and NAT64-embedded addresses are refused); at most three
  https redirects and one HTML alternate link; size and time limits.
- **Writes**: settings edits use a random temporary name, refuse links, keep the file's permissions (new files
  are private), and validate the result; the storage clean never follows a link, a junction or a special file;
  session exports are private (0600); session ids are validated before they become arguments or paths; UNC paths
  (`\\host\share`) are never probed, because on Windows that alone sends your login to the host.
- **Consent**: loading someone else's card lists, in full, every program it starts and every server it
  contacts with the variables it reads, and asks separately for the programs (`--allow-programs` to accept in
  scripts).

This list comes from an independent review of the code; the tests in `*_test.go` reproduce each attack.

## Rigcards

See the normative rules in [rigcard.md](rigcard.md#security-rules-for-importers): https only, SSRF guard, size
limits, pinned sources, hash verification, no secrets, hooks never auto-enabled.

## Reporting a vulnerability

See [SECURITY.md](../SECURITY.md).
