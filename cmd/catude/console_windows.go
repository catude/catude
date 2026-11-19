//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// consoleSupportsVT reports whether the Windows console understands ANSI escape sequences, turning the
// mode on if needed. Windows 10 supports it from version 1511, but the "Use legacy console" setting of
// the classic console turns it off; the interactive interface would then print raw escape codes.
func consoleSupportsVT() (ok bool, reason string) {
	h := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return true, "" // not a console (redirected, or a terminal emulator that is not conhost): nothing to check
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true, ""
	}
	if err := windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING|windows.ENABLE_PROCESSED_OUTPUT); err != nil {
		return false, "this console cannot show colours or move the cursor (ANSI escape codes)"
	}
	return true, ""
}
