# Windows 10 and Windows 11

catude is developed and tested on Windows 11. Windows 10 is supported: the differences that matter are the
console, which this page covers, and nothing else in catude depends on the Windows version.

| | Windows 11 | Windows 10 |
|---|---|---|
| Minimum | any | version 1607 or later (Go programs require Windows 10; colour support arrived in 1511) |
| Default terminal | Windows Terminal | the classic console (conhost), unless you installed Windows Terminal |
| Glyphs catude chooses | full set (`✓ ▲ ✗`, rounded borders) in Windows Terminal | console-safe set (`√ ▲ x`, square borders): only characters every console font has |
| Colours | yes | yes, unless "Use legacy console" is ticked (see below) |
| PATH changes by `catude install` | your user PATH, no administrator rights | same |
| Claude Desktop from the Microsoft Store | its real config under `%LOCALAPPDATA%\Packages\Claude_…` is shown | same |
| PowerShell used internally | Windows PowerShell 5.1 (built in) | same |

## Recommended: Windows Terminal

On Windows 10, install **Windows Terminal** from the Microsoft Store. catude works in the classic console,
but Windows Terminal draws the full glyph set and the mouse works better.

## "Use legacy console"

The classic console has an option, *Use legacy console*, that turns off colours and cursor control. The
interactive interface cannot work there, so catude checks before opening it and explains how to fix it:
right-click the title bar → **Properties** → untick **Use legacy console** → open a new window. Every command
(`catude doctor`, `ls`, `audit`, `sessions`, `profile list`, `graph`…) still works there as plain text.

## What was checked

- Builds for Windows amd64 and arm64 on every change; all tests run on Windows 11.
- The binary was run inside the classic console host (`conhost.exe`, the Windows 10 default), where catude
  picks the console-safe glyph set and square borders.
- No API newer than Windows 10 is used: console mode through `GetConsoleMode`/`SetConsoleMode`, processes
  through WMI (`Get-CimInstance Win32_Process`), the user PATH through .NET `SetEnvironmentVariable`.

Not yet run on a physical Windows 10 machine: if something looks wrong there, `catude debug` prints what a
bug report needs.
