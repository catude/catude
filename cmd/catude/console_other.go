//go:build !windows

package main

// consoleSupportsVT: every terminal catude supports outside Windows understands ANSI escape codes.
func consoleSupportsVT() (ok bool, reason string) { return true, "" }
