package tui

import "github.com/catude/catude/internal/i18n"

// T and F translate interface text (see internal/i18n). Messages are written in English in the code.
func T(s string) string { return i18n.T(s) }

func F(format string, args ...any) string { return i18n.F(format, args...) }
