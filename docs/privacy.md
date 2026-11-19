---
description: >-
  catude is local and read-only by default: no telemetry, no account, no API key, secrets always masked.
  What it reads, writes, runs and sends, and how to verify a release.
---

# Privacy and safety

1. **Local and read-only by default.** catude reads files on your machine. Every write is explicit, backed up
   first and reversible.
2. **No telemetry, account or API key.** It goes online only when you ask for something that needs it:
   importing a URL you name, installing through the official `claude` commands, or the public model data of the
   [Models](models.md) tab.
3. **Secrets stay masked.** Values that look like credentials are shown as `••••` and never exported. The
   session key files of Claude Code are never opened.
4. **Official commands and documented settings.** Plugins and MCP servers are installed with `claude plugin`
   and `claude mcp`; per-project switches use the settings Anthropic documents.
5. **Measured numbers.** When a number is an estimate (a cost at list prices), catude says so.

Before each change, catude copies the file to `<user cache>/catude/backups/` and leaves the rest of the file
untouched. The full list of what it reads, never reads, writes, runs and downloads is in the
[security model](security.md#what-catude-does-and-does-not-do).

## Security you can check

| | |
|---|---|
| **Every change is scanned** | [CodeQL](https://github.com/catude/catude/actions/workflows/codeql.yml) (code scanning), `govulncheck` (known vulnerabilities in Go and its dependencies) and `gitleaks` (secrets) run on every pull request and push, next to the tests on Windows, macOS and Linux |
| **Protected main branch** | Changes arrive through pull requests that must pass every check; no force pushes |
| **Dependencies kept current** | Dependabot opens updates for Go modules and GitHub Actions; releases are built with the latest Go and its security fixes |
| **Verifiable downloads** | Every release has a `checksums.txt` and a signed build provenance attestation: `gh attestation verify catude_*.zip --repo catude/catude` proves a file was built here, from this source |
| **Small surface** | One static binary, no runtime, no telemetry, no account; it goes online only for what you ask |

[Security model](security.md) · [Report a vulnerability](../SECURITY.md) (privately, through GitHub)
