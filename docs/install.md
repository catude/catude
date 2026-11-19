---
description: >-
  Install catude on Windows, macOS, Linux or WSL in one line, without administrator rights. Update,
  uninstall, verify a download, and add the catude slash commands inside Claude Code.
---

# Install

catude is one static binary. Each line below downloads the latest release, checks its checksum and puts
`catude` and the short `ct` on your PATH. No administrator rights are needed.

=== "Windows (PowerShell)"

    ```powershell
    irm https://raw.githubusercontent.com/catude/catude/main/scripts/install.ps1 | iex
    ```

    Installs to `%LOCALAPPDATA%\Programs\catude` and adds it to your user PATH. Works on Windows 10 and 11
    ([Windows notes](windows.md)).

=== "Linux, macOS, WSL"

    ```sh
    curl -fsSL https://raw.githubusercontent.com/catude/catude/main/scripts/install.sh | sh
    ```

    Installs to `~/.local/bin`. The Linux and macOS builds are **experimental**: tested on every change,
    with little real-world use so far ([Linux notes](linux.md)).

=== "Go 1.24 or later"

    ```sh
    go install github.com/catude/catude/cmd/catude@latest
    ```

    Then run `catude install` once if you also want the short `ct` and `ctlite` commands.

=== "Download"

    Download the archive for your system from
    [Releases](https://github.com/catude/catude/releases), unpack it and run `catude`. The first run offers to
    install itself, with the answers already selected.

!!! note "Prefer to read the script first?"
    Download [`install.ps1`](https://github.com/catude/catude/blob/main/scripts/install.ps1) or
    [`install.sh`](https://github.com/catude/catude/blob/main/scripts/install.sh), read it, then run it
    (`sh install.sh --yes` accepts the pre-selected answers). Both refuse to install if the checksum does not
    match `checksums.txt` of the release.

## First steps

```sh
catude why      # where your Claude Code tokens went in the last 14 days, and what to change
catude          # the terminal app; ct is the same, ctlite opens only the tabs you chose
catude audit    # security findings in what you installed
```

Run `catude` in a project folder: that folder is the *project* whose settings, skills and MCP servers are read.
Change it with `--project DIR`. Press `?` in the app for every key. See [Tabs and keys](usage.md) and
[Command line](cli.md).

## What `catude install` asks

1. Copy catude to the per-user folder above and add it to your PATH (pre-selected).
2. Also create the short commands `ct` (= `catude`) and `ctlite` (= `catude --lite`) (pre-selected).
3. Make plain `claude` start your active [work profile](profiles.md) through shell integration (not selected;
   change it later with `ct switch`).

Flags: `--yes` accepts the pre-selected answers, `--no-ct` skips the short commands, `--shell` turns on the
shell integration.

## Update

Run the same install line again: it replaces the binary with the latest release. With Go, run the
`go install ...@latest` line again.

## Uninstall

```sh
catude uninstall          # asks first; --yes skips the question
```

It removes `catude`, `ct` and `ctlite` from your PATH. Your work profiles and preferences stay; delete them
by hand if you want (`catude profile dir` prints where they are). Backups catude made of your Claude files are in
`<user cache>/catude/backups/`.

## Verify a download

Every release has a `checksums.txt` and a signed build provenance attestation. With the GitHub CLI:

```sh
gh attestation verify catude_*.zip --repo catude/catude
```

This proves the file was built by this repository's release workflow, from this source. More in
[Privacy and safety](privacy.md#security-you-can-check).

## Inside Claude Code

The repository is also a Claude Code plugin marketplace. In Claude Code:

```text
/plugin install catude --marketplace catude/catude
```

That adds four slash commands that run catude and let Claude explain the result. They need the `catude`
program on your PATH and never change a file.

| Command | What Claude explains |
|---|---|
| `/catude:doctor` | which files Claude reads on this machine, which one wins, and anything missing or invalid |
| `/catude:audit` | the security findings in your skills, hooks, plugins and MCP servers |
| `/catude:sessions` | the context use of your recent sessions |
| `/catude:models` | which model and effort level to pick for your work |

## Requirements

- Windows 10 or 11, macOS, Linux or WSL, on x86-64 or ARM64.
- Claude Code and/or Claude Desktop installed (catude reads their files; it works with either or both).
- Optional: [claude-swap](https://github.com/realiti4/claude-swap) for plan limits and several Claude accounts
  (catude can install it for you from the Profiles tab).
- Any terminal. catude detects what your terminal and font can draw ([accessibility](accessibility.md)).
