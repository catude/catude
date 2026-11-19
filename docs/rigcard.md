# Export and import

```sh
catude export my-setup.json       # your setup in one file, without secrets
catude import my-setup.json       # on another machine (or a URL): see what it brings, audit it, install it as a work profile
```

The file is a Rigcard, described below. You only need this page if you want to read or write one by hand.

# Rigcard 0.1

**A Rigcard describes a Claude setup**: the marketplaces, plugins, skills, agents, commands, MCP
servers and safe settings you use, each with a version and a link. Publish it at a URL you own
(`yourdomain.dev/rig`) and anyone can load it, compare it with their machine and adopt what they like.

Status: **draft 0.1**, implemented by catude (`rig export`, `rig show`). Feedback welcome before 1.0.

```sh
catude rig export -o rigcard.json --summary "Go + TDD, security-first" --owner "@ada"
catude rig show ada.dev/rig          # validate, audit, and diff against this machine
```

## In practice: define your profile in a JSON file and load it from a URL

1. **Make the profile** on your machine (Profiles → `n`, or `catude profile save my-setup`).
2. **Export it as a bundle**: `catude profile export my-setup ./my-rig`. You get `my-rig/rigcard.json` plus the
   profile's skills, agents and commands, each with its sha256 in the card. Variable *values* never leave
   your machine: MCP servers keep only the names of the variables they need.
3. **Publish it**: push the folder to a GitHub repository, or put `rigcard.json` (and the folders next to it)
   on any https site.
4. **Anyone loads it**:

   ```sh
   catude profile load gh:you/my-rig            # rigcard.json at the root of the repository
   catude profile load gh:you/rigs@v2           # a tag, branch or commit
   catude profile load you.dev/rig              # a web page that links its card, or you.dev/rig.json
   catude profile load ./my-rig/rigcard.json    # a folder: a shared drive, a zip you sent
   ```

   catude validates the card, downloads only what is pinned, checks every file against its sha256, audits it,
   shows every program the profile would start and every server it would contact, and asks.

You can also write the JSON by hand: a card that only lists plugins from public marketplaces and skills from
GitHub (repo + path + commit) needs no files of its own. The full format is below.

## Design goals

1. **Compose, don't replace.** Plugins and marketplaces use Claude Code's own identifiers and source vocabulary;
   MCP servers follow the MCP Registry `server.json` model (variables have a name and `isSecret`, never a value).
2. **Never a secret.** A card with a credential-shaped value anywhere is invalid.
3. **Verifiable.** Sources pin to a commit; local artifacts carry a SHA-256 tree hash.
4. **Safe to load.** Reading a card never runs anything; adopting it goes through official commands, item by item.

## Discovery

Given the user's input, an importer tries, in order:

| Input | Requests |
|---|---|
| `host/path` (e.g. `ada.dev/rig`) | `https://host/path` with `Accept: application/vnd.rigcard+json, application/json;q=0.9`. If it returns JSON, done. If it returns HTML, follow `<link rel="alternate" type="application/vnd.rigcard+json" href="…">` (https only). Then try `https://host/path.json` |
| `host` | `https://host/.well-known/rigcard.json` (provisional; may be an index `{"cards":[{"name","url"}]}` in a later version) |
| `gh:owner/repo[@ref]` | `https://raw.githubusercontent.com/owner/repo/<ref or HEAD>/rigcard.json` |
| a local path | the file |

So a personal site can serve a human page at `/rig` and the card at `/rig.json`, linked with:

```html
<link rel="alternate" type="application/vnd.rigcard+json" href="/rig.json">
```

Media type: `application/vnd.rigcard+json` (vendor tree, not registered).

## Document

```jsonc
{
  "$schema": "https://catude.dev/schema/rigcard-0.1.json",   // optional
  "rigcard": "0.1",                                            // required
  "id": "https://ada.dev/rig",                                 // canonical URL
  "updated": "2026-11-17",
  "owner": { "name": "Ada", "handle": "@ada", "url": "https://ada.dev" },
  "summary": "Go + TDD, security-first",
  "license": "MIT",
  "harnesses": [{ "id": "claude-code", "version": "2.1.330", "primary": true }],
  "models": [{ "id": "claude-opus-5-5", "role": "main", "effort": "high" }],
  "marketplaces": [
    { "name": "ada-tools",
      "source": { "source": "github", "repo": "ada/claude-tools", "ref": "3f1c…40 hex…" } }
  ],
  "plugins": [{ "id": "superpowers@claude-plugins-official", "version": "7.0.2", "enabled": true }],
  "skills": [
    { "name": "tdd", "description": "Test first…",
      "source": { "type": "github", "repo": "ada/dotclaude", "path": "skills/tdd", "commit": "…40 hex…" },
      "sha256": "…64 hex tree hash…" }
  ],
  "agents": [], "commands": [],
  "hooks": [{ "event": "PreToolUse", "matcher": "Bash", "summary": "blocks rm -rf", "executesCode": true }],
  "mcpServers": [
    { "name": "linear", "transport": "http", "url": "https://mcp.linear.app/mcp",
      "headers": [{ "name": "Authorization", "isSecret": true, "isRequired": true }] },
    { "name": "postgres", "transport": "stdio", "command": "uvx", "args": ["postgres-mcp==0.3.0"],
      "env": [{ "name": "DATABASE_URL", "isSecret": true }] }
  ],
  "settings": { "model": "opus", "effortLevel": "high", "permissions": { "deny": ["Read(./.env)"] } },
  "desktop": { "connectors": ["Linear", "Notion"] },
  "practices": [{ "title": "Plan before coding", "body": "…" }]
}
```

### Fields

| Field | Notes |
|---|---|
| `rigcard` | Version. Importers reject versions they do not know |
| `owner.art` | Optional ASCII art the owner chose for themselves: at most 12 lines of 40 columns, no control or bidi characters |
| `marketplaces[].source` | Same vocabulary as Claude Code marketplaces: `github` (`repo`, `ref`), `git`/`url` (`url`, `ref`). `ref` should be a 40-hex commit |
| `plugins[].id` | `name@marketplace`, exactly as `claude plugin install` takes it |
| `skills` · `agents` · `commands` | `source.type`: `github` (repo + path + commit), `https` (a single file; `sha256` required), `bundle` (a path next to the card: in the same folder, or in the same repository and revision when loaded with `gh:`; `sha256` required) or `local` (the author's own copy, not downloadable) |
| `hooks` | Described, not shipped as runnable config. `executesCode: true` must be shown to the user |
| `mcpServers[].env` / `headers` | **Names only**, with `isSecret` and `isRequired` |
| `settings` | Allow-list: `model`, `effortLevel`, `outputStyle`, `permissions.deny`, `permissions.ask`. **Forbidden:** `permissions.allow`, `env`, `apiKeyHelper`, `statusLine`, hooks |
| `desktop.connectors` | Names of claude.ai connectors; each person connects their own account |

### Tree hash

For a directory: list every regular file (symlinks are refused), with its path relative to the root using `/` and
UTF-8, sorted bytewise. For each file emit `path + "\0" + hex(sha256(content)) + "\n"`; the tree hash is
`hex(sha256(all of that))`. For a single file, `sha256` is the file's own hash.

## Security rules for importers

In addition to the rules below, catude rejects a card when: any value contains a control character (escape
sequences cannot reach your terminal); a plugin id, repository, MCP name or variable name has an unexpected
shape; an MCP URL, argument or default value contains `$` (Claude Code expands `${VAR}` from *your* environment,
so a card must not be able to place your variables into a URL or an argument); a URL source has no sha256.
Revisions count as pinned only when the revision actually downloaded is a 40-character commit.

These are normative.

1. **HTTPS only.** Refuse `http://`, and refuse to connect to loopback, private, link-local, CGNAT and cloud
   metadata addresses, checked on the *resolved* IP, so DNS rebinding fails. At most 3 redirects, https only.
2. **Limits.** 256 KiB per card, 30 s per request, 500 items per list. Names with control or bidi characters,
   path separators or `..` are rejected.
3. **No impersonation.** A marketplace named `claude-plugins-official` or `anthropic*` that does not come from
   `anthropics/` is rejected.
4. **No secrets.** A card containing a credential-shaped value (API keys, tokens, JWTs, private keys) is rejected
   as a whole. Exporters must strip values and run a secret scan before writing.
5. **Pinning.** Unpinned sources and `npx`/`uvx` without an exact version are flagged; importers must not
   auto-update them.
6. **Verify.** When a `sha256` is present, the fetched artifact must match it, or adoption aborts.
7. **Nothing runs on load.** Hooks are never enabled automatically; `statusLine` and command sources are never run.
8. **Consent per item.** Adoption shows what is added or changed and goes through official commands
   (`claude plugin …`, `claude mcp add …`) so Claude Code's own trust prompts still apply.
9. **Isolation (0.2).** "Try someone's setup" creates a separate `CLAUDE_CONFIG_DIR` profile instead of touching
   `~/.claude`.

## What catude does today

- `rig export`: builds a card from your inventory, hashes local skills/agents/commands, keeps only variable names,
  applies the settings allow-list, validates its own output and refuses to write it if a secret slipped through.
  **Review the card before publishing:** names of private MCP servers, skills or projects may themselves be
  confidential.
- `profile export NAME DIR` / `profile load DIR/rigcard.json` or `gh:owner/repo`: bundles, as described at the top.
- `rig show`: discovers, validates and diffs a card against this machine (✔ have · + missing · ~ different ·
  ! review) and prints the official commands to adopt what is missing. It runs nothing.
- `catude profile load SOURCE` installs a card as a [work profile](profiles.md#work-profiles): pinned downloads,
  hash verification, safe extraction, audit and consent, never hooks.
- Planned for 0.2: signatures (JWS over JCS or
  minisign), a JSON Schema, and an index format for hosts with several cards.

## Prior art

uses.tech and `/uses` pages (human pages, no machine format) · llms.txt (discovery by convention) · RFC 8615
`.well-known` · A2A agent cards · MCP Registry `server.json` · Claude Code marketplaces and `plugin.json` ·
skills.sh `skills-lock.json` · dotfiles, Brewfiles and devcontainer features.
