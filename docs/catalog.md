# The Discover catalog

The **Discover** tab and `catude discover` show a curated list of the Claude ecosystem: 177 plugins, 89 skills and
skill collections, 55 marketplaces, 25 MCP servers, companion tools and sourced practices. Every plugin and skill
was checked at its source (licence that allows reuse, a commit in the last nine months, real `SKILL.md` files,
the plugin present in its marketplace) and carries tags from a controlled list (`tags` at the top of
`catalog.json`).

Search with `/`: a word matches names and descriptions, `#tag` matches a tag (`#testing`, `#frontend`,
`#security`, `#docs`, `#devops`…).


| Section | What | Install command |
|---|---|---|
| marketplaces | Anthropic's official ones and the most used community marketplaces | `claude plugin marketplace add owner/repo` |
| plugins | Plugins from those marketplaces, by category (security, workflow, frontend, data…) | `claude plugin install name@marketplace` |
| mcp | Popular MCP servers, official ones marked | `claude mcp add …` |
| skills | Notable skill collections and single skills | link to the source |
| tools | Companion tools: usage trackers, status lines, account switchers, GUIs, scanners | link to the source |
| practices | Short, sourced rules (each with the link that backs it) | none |

`✔` marks what is already on this machine; `●` marks official entries; `★` is the GitHub star count when the
catalog was generated; **quiet** marks a repository that is archived or had no change in the year before that
date (the detail says since when).

## Keeping it current

`.github/workflows/maintenance.yml` runs every Monday. `scripts/refresh-catalog.mjs` updates the stars, the
date of the last change and the archived flag of every entry with a repository and opens a pull request; a
repository that no longer exists is listed in it for a person to decide. `scripts/check-settings-drift.mjs`
compares the settings reference with Anthropic's documentation and opens an issue when settings or
environment variables appear or disappear.

## Rules for entries

- Embedded in the binary: Discover works offline and never phones home.
- Every link is https. Every install command is an official `claude plugin` or `claude mcp add` command; anything
  that needs a value (usually a token) is shown with a placeholder and is never run by catude.
- Star counts are a snapshot (see `generated` in `internal/catalog/catalog.json`), not live data.
- Inclusion is not an endorsement or a security review. Third-party plugins run code with your permissions:
  read them, then run `catude audit`.

## Adding or fixing an entry

Edit `internal/catalog/catalog.json` and open a pull request with the source link. `go test ./internal/catalog`
checks that URLs are https and install commands are official. Prefer entries that are maintained, licensed and
have a clear owner.
