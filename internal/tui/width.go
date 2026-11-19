package tui

import "github.com/charmbracelet/x/ansi"

// Widths are measured in terminal cells, not runes: Chinese, Japanese and Korean characters take two
// cells, so padding and cutting by rune count would break every column once the interface is translated.

// truncate cuts s (which may hold colours) to w cells, ending with … when it was longer.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= w {
		return s
	}
	return ansi.Truncate(s, w, "…")
}

// padRight pads s with spaces to w cells (or cuts it).
func padRight(s string, w int) string {
	s = truncate(s, w)
	if n := w - ansi.StringWidth(s); n > 0 {
		return s + spaces(n)
	}
	return s
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}

// ansiWidth is the width of s in terminal cells.
func ansiWidth(s string) int { return ansi.StringWidth(s) }
