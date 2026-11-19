package presets

import "strings"

// A 5-row block font for A–Z. Monograms are made of initials only: catude never draws people's faces
// (that would use their likeness without consent). A person can ship their own ASCII in their Rigcard.
var font = map[rune][5]string{
	'A': {" ███ ", "█   █", "█████", "█   █", "█   █"},
	'B': {"████ ", "█   █", "████ ", "█   █", "████ "},
	'C': {" ████", "█    ", "█    ", "█    ", " ████"},
	'D': {"████ ", "█   █", "█   █", "█   █", "████ "},
	'E': {"█████", "█    ", "████ ", "█    ", "█████"},
	'F': {"█████", "█    ", "████ ", "█    ", "█    "},
	'G': {" ████", "█    ", "█  ██", "█   █", " ████"},
	'H': {"█   █", "█   █", "█████", "█   █", "█   █"},
	'I': {"█████", "  █  ", "  █  ", "  █  ", "█████"},
	'J': {"  ███", "    █", "    █", "█   █", " ███ "},
	'K': {"█   █", "█  █ ", "███  ", "█  █ ", "█   █"},
	'L': {"█    ", "█    ", "█    ", "█    ", "█████"},
	'M': {"█   █", "██ ██", "█ █ █", "█   █", "█   █"},
	'N': {"█   █", "██  █", "█ █ █", "█  ██", "█   █"},
	'O': {" ███ ", "█   █", "█   █", "█   █", " ███ "},
	'P': {"████ ", "█   █", "████ ", "█    ", "█    "},
	'Q': {" ███ ", "█   █", "█ █ █", "█  █ ", " ██ █"},
	'R': {"████ ", "█   █", "████ ", "█  █ ", "█   █"},
	'S': {" ████", "█    ", " ███ ", "    █", "████ "},
	'T': {"█████", "  █  ", "  █  ", "  █  ", "  █  "},
	'U': {"█   █", "█   █", "█   █", "█   █", " ███ "},
	'V': {"█   █", "█   █", "█   █", " █ █ ", "  █  "},
	'W': {"█   █", "█   █", "█ █ █", "██ ██", "█   █"},
	'X': {"█   █", " █ █ ", "  █  ", " █ █ ", "█   █"},
	'Y': {"█   █", " █ █ ", "  █  ", "  █  ", "  █  "},
	'Z': {"█████", "   █ ", "  █  ", " █   ", "█████"},
}

// Monogram draws the initials in a 5-row block font (every row the same width).
func Monogram(initials string) []string {
	rows := make([]string, 5)
	for i, r := range []rune(strings.ToUpper(initials)) {
		g, ok := font[r]
		if !ok {
			continue
		}
		for j := range rows {
			if i > 0 {
				rows[j] += " "
			}
			rows[j] += g[j]
		}
	}
	return rows
}

// Art is what the gallery shows in the corner: the person's own ASCII if they published one
// (cleaned and bounded), otherwise the monogram.
func (p Preset) Art() []string {
	if a := strings.TrimRight(p.ASCII, "\n"); a != "" {
		lines := strings.Split(a, "\n")
		if len(lines) > 12 {
			lines = lines[:12]
		}
		for i, l := range lines {
			if r := []rune(l); len(r) > 32 {
				lines[i] = string(r[:32])
			}
		}
		return lines
	}
	return Monogram(p.Initials())
}
