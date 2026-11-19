# Packaging

How catude reaches package managers. Every channel installs the same archives that the release
workflow builds with GoReleaser when a `v*` tag is pushed (`.github/workflows/release.yml`).

| Channel | Who it serves | What it needs | State |
|---|---|---|---|
| GitHub release | everyone | a `v*` tag | ready |
| `go install` | Go users | the module path set with `scripts/set-owner.mjs` | ready |
| Claude Code plugin | people who live in Claude Code | the repository itself (`.claude-plugin/marketplace.json`) | ready |
| Homebrew tap | macOS, Linux | a repository `OWNER/homebrew-tap` and a token that can push to it | to do |
| Scoop bucket | Windows | a repository `OWNER/scoop-bucket` and the same token | to do |
| winget | Windows | a pull request to `microsoft/winget-pkgs` per version | to do |

## Homebrew and Scoop

Create the two empty repositories, add a fine-grained token with write access to both as the
`TAP_GITHUB_TOKEN` secret of the catude repository, pass it to the release job
(`TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}`), and add to `.goreleaser.yaml`:

```yaml
homebrew_casks:
  - name: catude
    repository: { owner: OWNER, name: homebrew-tap, token: "{{ .Env.TAP_GITHUB_TOKEN }}" }
    homepage: https://github.com/catude/catude
    description: Terminal UI to see and manage Claude Code and Claude Desktop
    license: MIT
scoops:
  - repository: { owner: OWNER, name: scoop-bucket, token: "{{ .Env.TAP_GITHUB_TOKEN }}" }
    homepage: https://github.com/catude/catude
    description: Terminal UI to see and manage Claude Code and Claude Desktop
    license: MIT
```

Then `brew install OWNER/tap/catude` and `scoop bucket add catude https://github.com/OWNER/scoop-bucket`
followed by `scoop install catude`. Check the keys against the GoReleaser version the workflow runs
(`goreleaser check`) before tagging.

## winget

winget needs a manifest per version in `microsoft/winget-pkgs`. After a release:

```powershell
wingetcreate new https://github.com/catude/catude/releases/download/vX.Y.Z/catude_X.Y.Z_windows_amd64.zip
```

Answer the prompts (identifier `OWNER.catude`, portable app, command `catude`), and submit the pull
request it prepares. Later versions: `wingetcreate update OWNER.catude --version X.Y.Z --urls … --submit`.
The first submission is reviewed by Microsoft and takes a few days.

## Before any channel

The program is not signed. Windows SmartScreen may warn on the first run of a downloaded `.exe`;
package managers verify the checksum of the archive but do not sign it either.
