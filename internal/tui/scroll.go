package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// scrollbar draws a vertical bar for a window of `visible` lines starting at `start` out of `total`.
// It returns one cell per visible line: a thumb (█) where the window is, a track (│) elsewhere, and
// arrows at the ends when there is more above or below. Nothing is drawn when everything fits.
func scrollbar(total, start, visible int) []string {
	if visible <= 0 {
		return nil
	}
	out := make([]string, visible)
	if total <= visible {
		for i := range out {
			out[i] = " "
		}
		return out
	}
	thumb := max(1, visible*visible/total)
	maxStart := total - visible
	pos := 0
	if maxStart > 0 {
		pos = (visible - thumb) * min(start, maxStart) / maxStart
	}
	for i := range out {
		switch {
		case i >= pos && i < pos+thumb:
			out[i] = sTitle.Render("█")
		default:
			out[i] = sMuted.Render("│")
		}
	}
	if start > 0 {
		out[0] = sTitle.Render("▲")
	}
	if start+visible < total {
		out[visible-1] = sTitle.Render("▼")
	}
	return out
}

// withScrollbar pads each line to width-1 cells and adds the scrollbar in the last column. lines are
// the rows currently shown (at most visible); missing rows are filled with blanks so the bar keeps its
// height.
func withScrollbar(lines []string, total, start, visible, width int) string {
	bar := scrollbar(total, start, visible)
	if total <= visible {
		return strings.Join(lines, "\n")
	}
	pad := lipgloss.NewStyle().Width(width - 1).MaxWidth(width - 1)
	out := make([]string, visible)
	for i := 0; i < visible; i++ {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		out[i] = pad.Render(l) + bar[i]
	}
	return strings.Join(out, "\n")
}

// position is the "12–40 of 230" hint shown under long lists.
func position(start, shown, total int) string {
	if total <= shown {
		return ""
	}
	end := min(total, start+shown)
	return sMuted.Render(" " + itoa(start+1) + "–" + itoa(end) + " of " + itoa(total))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
