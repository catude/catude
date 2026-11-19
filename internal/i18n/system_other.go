//go:build !windows

package i18n

// systemLanguages: on Linux and macOS terminals the locale variables (LANG, LC_ALL) already say it.
func systemLanguages() []string { return nil }
