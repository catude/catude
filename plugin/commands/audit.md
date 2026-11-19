---
description: Security findings in your Claude setup (hooks, MCP servers, permissions, secrets in files), explained with fixes
allowed-tools: Bash(catude audit:*)
---
Output of `catude audit --json` (secrets are already masked; if the command was not found, tell me to install catude and stop):

!`catude audit --json`

Group the findings by severity, most serious first. For each one say in one sentence what the risk is and the smallest change that removes it. Propose the changes, but do not edit any file until I say so.
