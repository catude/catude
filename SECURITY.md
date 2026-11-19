# Security policy

catude reads files that can contain credentials and can run official `claude` commands, so we take reports seriously.

## Reporting

Please **do not open a public issue**. Use GitHub's private vulnerability reporting ("Report a vulnerability" in
the Security tab). Include the version (`catude version`), the OS and the steps to reproduce. Never include real
tokens: use fake ones.

We aim to acknowledge reports within 3 working days and to ship a fix or mitigation within 30 days.

## In scope

- A secret value shown unmasked in the TUI, CLI, JSON, snapshot or a Rigcard.
- catude running a command, a hook or a network request the user did not explicitly ask for.
- Rigcard fetching that reaches a private address, follows http, or exceeds its limits.
- A crafted file (skill, settings, transcript, card) that crashes catude or makes it write outside its backup folder.

## Out of scope

Findings the audit misses or reports by mistake (these are heuristics; please open a normal issue), and vulnerabilities in
Claude Code, Claude Desktop, claude-swap or third-party plugins (report them upstream).

## Model

See [docs/security.md](docs/security.md).
