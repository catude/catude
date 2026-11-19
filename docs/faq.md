---
description: >-
  Frequently asked questions about catude: plan limits, whether it changes your Claude configuration, the
  Microsoft Store version of Claude Desktop, cost, privacy and supported systems.
faq:
  - q: "Are catude's token numbers the same as my Claude plan limits?"
    a: "No. Plan limits are computed by Anthropic and also count your other devices and claude.ai. catude shows what this machine recorded, de-duplicated, with where the tokens went."
  - q: "Does catude change my Claude configuration?"
    a: "Only when you press a key that does so. Before each change it copies the file to <user cache>/catude/backups/ and leaves the rest of the file untouched."
  - q: "Is catude an official Anthropic tool?"
    a: "No. catude is an independent open-source project (MIT), not affiliated with or endorsed by Anthropic."
  - q: "Does catude work with the Microsoft Store version of Claude Desktop?"
    a: "Yes. The Store app reads a virtualised copy of its configuration under %LOCALAPPDATA%\\Packages\\Claude_*; catude shows and edits the copy the app actually reads."
  - q: "Does catude send my data anywhere?"
    a: "No. There is no telemetry, account or API key. It goes online only when you ask for something that needs it, such as importing a URL you name or the public model data of the Models tab."
  - q: "Is catude free?"
    a: "Yes. catude is free and open source under the MIT licence."
---

# FAQ

## Are the token numbers the same as my plan limits?

No. Plan limits are computed by Anthropic and also count your other devices and claude.ai. catude shows what this
machine recorded, which is what you can act on: the same responses, de-duplicated, with where they went. With
[claude-swap](https://github.com/realiti4/claude-swap), the Usage tab also shows when each plan window resets.

## Does it change my Claude configuration?

Only when you press a key that does so (switch something off, change a setting, install a profile, clean a
folder). Before each change it copies the file to `<user cache>/catude/backups/` and leaves the rest of the file
untouched.

## Is it an official Anthropic tool?

No. catude is an independent open-source project, not affiliated with or endorsed by Anthropic. The paths and
settings it reads come from Anthropic's documentation.

## Does it send my data anywhere?

No. No telemetry, account or API key. It goes online only when you ask for something that needs it: importing a
URL you name, installing through the official `claude` commands, or the public model data of the Models tab.
See [Privacy and safety](privacy.md).

## Is it free?

Yes, free and open source under the MIT licence.

## Does it work with Claude Desktop, or only Claude Code?

Both. It reads Claude Code's settings and transcripts and Claude Desktop's configuration (every install, and its
policies). The Live and Sessions tabs also show Claude Code sessions started from Claude Desktop's Code tab.

## I use the Microsoft Store version of Claude Desktop. Does it work?

Yes. The Store app reads a virtualised copy of its configuration under `%LOCALAPPDATA%\Packages\Claude_*\…`, so
editing `%APPDATA%\Claude\` has no effect. catude shows and edits the copy the app actually reads.

## Which systems does it run on?

Windows 10 and 11, macOS, Linux and WSL, on x86-64 and ARM64. The Linux and macOS builds are experimental:
tested on every change, with little real-world use so far.

## My terminal shows strange symbols

Run `catude --glyphs safe` (or `ascii`), or set it in preferences (`,`). See
[Accessibility](accessibility.md). On an old Windows console, use Windows Terminal.

## How do I report a bug?

Open an issue on [GitHub](https://github.com/catude/catude/issues) with the output of `catude debug` (it contains
no secrets). Security problems go privately through [SECURITY.md](../SECURITY.md).

## Where do the screenshots come from?

From an invented machine (user `alex`, invented projects, sessions and usage) built by the repository's demo
script. The command-line pictures are the real output of catude on that machine.
