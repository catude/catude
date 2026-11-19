# Roadmap

## 0.1: prototype (now)

Inventory of Code + Desktop (all scopes, MSIX, Linux, WSL) · live MCP and claude.ai connectors · exact session
context and `/context` composition · profiles, installs and claude-swap accounts · live editing with backup and
re-validation · 23-rule audit · Discover catalog with safe install · Rigcard export/show · JSON everywhere ·
effective settings (which file wins for each key) · notice when a session finishes · estimated cost per session ·
`profile desktop` (a profile's MCP servers into Claude Desktop) · Claude Code plugin (`/catude:doctor`,
`audit`, `sessions`, `models`) · weekly catalog and settings-reference checks.

## 0.2: trust and reach

- **Effective configuration** for MCP servers and hooks too (settings are done).
- **Desktop ↔ Code MCP diff**, both ways, with secrets moved to environment variables.
- **`rig apply`** into an isolated `CLAUDE_CONFIG_DIR` profile, item by item; Rigcard JSON Schema and signatures.
- **Audit**: SARIF output, baselines (`--baseline`) to fail CI only on new findings, pre-install audit of a plugin.
- **Install channels**: Homebrew tap, Scoop, winget ([how](packaging.md)), `npx catude`, `uv tool install catude`, AUR.

## 0.3: cost and control

- Token cost per MCP server and per skill before opening a session; quick enable/disable per project.
- Profiles per project (which profile a folder should use).
- Catalog refresh from a signed, versioned feed (opt-in).

## Later

Themes and key remapping · plugin API for extra tabs · a public Rigcard directory (opt-in, like uses.tech).
