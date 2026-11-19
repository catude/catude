package main

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// A terminal frame (text with SGR colour codes) drawn as an SVG window. Every run of text is placed on
// its exact cell and stretched to its exact width, so the grid stays aligned whatever monospace font the
// viewer has.

const (
	cellW, cellH = 9.0, 19.0
	padX, padTop = 18.0, 46.0
	fg0, bg0     = "#E6E6E6", "#14161B"
)

type style struct {
	fg, bg          string
	bold, underline bool
	reverse         bool
}

var ansi16 = []string{"#1E1E1E", "#E06C75", "#98C379", "#E5C07B", "#61AFEF", "#C678DD", "#56B6C2", "#D0D0D0",
	"#5C6370", "#FF7A85", "#B5E890", "#FFD68A", "#80C8FF", "#E09BFF", "#7FDBE6", "#FFFFFF"}

func color256(n int) string {
	switch {
	case n < 16:
		return ansi16[n]
	case n < 232:
		n -= 16
		v := []int{0, 95, 135, 175, 215, 255}
		return fmt.Sprintf("#%02X%02X%02X", v[n/36], v[(n/6)%6], v[n%6])
	}
	g := 8 + (n-232)*10
	return fmt.Sprintf("#%02X%02X%02X", g, g, g)
}

func (s *style) apply(params []int) {
	if len(params) == 0 {
		*s = style{}
		return
	}
	for i := 0; i < len(params); i++ {
		p := params[i]
		switch {
		case p == 0:
			*s = style{}
		case p == 1:
			s.bold = true
		case p == 4:
			s.underline = true
		case p == 7:
			s.reverse = true
		case p == 22:
			s.bold = false
		case p == 24:
			s.underline = false
		case p == 27:
			s.reverse = false
		case p >= 30 && p <= 37:
			s.fg = ansi16[p-30]
		case p >= 90 && p <= 97:
			s.fg = ansi16[p-90+8]
		case p == 39:
			s.fg = ""
		case p >= 40 && p <= 47:
			s.bg = ansi16[p-40]
		case p >= 100 && p <= 107:
			s.bg = ansi16[p-100+8]
		case p == 49:
			s.bg = ""
		case (p == 38 || p == 48) && i+1 < len(params):
			var c string
			if params[i+1] == 2 && i+4 < len(params) {
				c = fmt.Sprintf("#%02X%02X%02X", params[i+2], params[i+3], params[i+4])
				i += 4
			} else if params[i+1] == 5 && i+2 < len(params) {
				c = color256(params[i+2])
				i += 2
			}
			if p == 38 {
				s.fg = c
			} else {
				s.bg = c
			}
		}
	}
}

type run struct {
	col, width int
	text       string
	st         style
}

// parse splits a frame into lines of styled runs.
func parse(frame string) [][]run {
	var out [][]run
	var st style
	for _, line := range strings.Split(strings.ReplaceAll(frame, "\r", ""), "\n") {
		var runs []run
		col := 0
		var cur strings.Builder
		curCol := 0
		flush := func() {
			if cur.Len() > 0 {
				t := cur.String()
				runs = append(runs, run{curCol, ansi.StringWidth(t), t, st})
				cur.Reset()
			}
		}
		for i := 0; i < len(line); {
			if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
				j := i + 2
				for j < len(line) && (line[j] < 0x40 || line[j] > 0x7e) {
					j++
				}
				if j < len(line) && line[j] == 'm' {
					flush()
					var ps []int
					for _, f := range strings.Split(line[i+2:j], ";") {
						if f == "" {
							ps = append(ps, 0)
							continue
						}
						n, _ := strconv.Atoi(f)
						ps = append(ps, n)
					}
					st.apply(ps)
					curCol = col
				}
				i = j + 1
				continue
			}
			r := []rune(line[i:])[0]
			s := string(r)
			if cur.Len() == 0 {
				curCol = col
			}
			cur.WriteString(s)
			col += ansi.StringWidth(s)
			i += len(s)
		}
		flush()
		out = append(out, runs)
	}
	return out
}

func toSVG(frame string, cols, rows int) string {
	lines := parse(frame)
	w := padX*2 + float64(cols)*cellW
	h := padTop + float64(rows)*cellH + 16
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f">`, w, h, w, h)
	b.WriteString(`<style>text{font-family:"JetBrains Mono","Cascadia Mono","SF Mono",Menlo,Consolas,"DejaVu Sans Mono",monospace;font-size:15px;white-space:pre}</style>`)
	fmt.Fprintf(&b, `<rect width="%.0f" height="%.0f" rx="10" fill="%s"/>`, w, h, bg0)
	b.WriteString(`<circle cx="22" cy="20" r="6" fill="#FF5F57"/><circle cx="42" cy="20" r="6" fill="#FEBC2E"/><circle cx="62" cy="20" r="6" fill="#28C840"/>`)
	fmt.Fprintf(&b, `<text x="%.0f" y="25" fill="#8B93A1" text-anchor="middle" style="font-size:13px">catude</text>`, w/2)
	for y, runs := range lines {
		if y >= rows {
			break
		}
		top := padTop + float64(y)*cellH
		for _, r := range runs {
			fg, bg := r.st.fg, r.st.bg
			if r.st.reverse {
				fg, bg = bg, fg
				if fg == "" {
					fg = bg0
				}
				if bg == "" {
					bg = fg0
				}
			}
			x := padX + float64(r.col)*cellW
			if bg != "" {
				fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`, x, top, float64(r.width)*cellW, cellH, bg)
			}
			if strings.TrimSpace(r.text) == "" {
				continue
			}
			if fg == "" {
				fg = fg0
			}
			attrs := ""
			if r.st.bold {
				attrs += ` font-weight="bold"`
			}
			if r.st.underline {
				attrs += ` text-decoration="underline"`
			}
			// box-drawing characters become lines that fill their cell (fonts leave gaps between rows);
			// the text between them is placed on its cells and stretched to their exact width
			col := r.col
			var seg strings.Builder
			segCol := col
			flushText := func() {
				if t := seg.String(); strings.TrimSpace(t) != "" {
					fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="%s"%s textLength="%.1f" lengthAdjust="spacingAndGlyphs">%s</text>`,
						padX+float64(segCol)*cellW, top+cellH*0.76, fg, attrs, float64(ansi.StringWidth(t))*cellW, html.EscapeString(t))
				}
				seg.Reset()
			}
			for _, ch := range r.text {
				w := ansi.StringWidth(string(ch))
				if ch == '█' { // a full block fills its cell
					flushText()
					fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`, padX+float64(col)*cellW, top+2, cellW, cellH-4, fg)
					col += w
					segCol = col
					continue
				}
				if d, ok := boxPath(ch, padX+float64(col)*cellW, top); ok {
					flushText()
					fmt.Fprintf(&b, `<path d="%s" stroke="%s" stroke-width="1.3" fill="none"/>`, d, fg)
					col += w
					segCol = col
					continue
				}
				if seg.Len() == 0 {
					segCol = col
				}
				seg.WriteRune(ch)
				col += w
			}
			flushText()
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func svgAll(in string) error {
	b, err := os.ReadFile(filepath.Join(in, "frames.json"))
	if err != nil {
		return err
	}
	var info []frameInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return err
	}
	for _, f := range info {
		raw, err := os.ReadFile(filepath.Join(in, f.File+".ansi"))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(in, f.File+".svg"), []byte(toSVG(string(raw), termW, termH)), 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("%d SVG frames in %s\n", len(info), in)
	return nil
}

// boxPath draws a box-drawing character as lines across its whole cell.
func boxPath(ch rune, x, top float64) (string, bool) {
	l, r := x, x+cellW
	cx, cy := x+cellW/2, top+cellH/2
	t, btm := top, top+cellH
	f := func(format string, a ...any) string { return fmt.Sprintf(format, a...) }
	switch ch {
	case '│':
		return f("M%.1f %.1fV%.1f", cx, t, btm), true
	case '─':
		return f("M%.1f %.1fH%.1f", l, cy, r), true
	case '╭':
		return f("M%.1f %.1fQ%.1f %.1f %.1f %.1f", r, cy, cx, cy, cx, btm), true
	case '╮':
		return f("M%.1f %.1fQ%.1f %.1f %.1f %.1f", l, cy, cx, cy, cx, btm), true
	case '╰':
		return f("M%.1f %.1fQ%.1f %.1f %.1f %.1f", cx, t, cx, cy, r, cy), true
	case '╯':
		return f("M%.1f %.1fQ%.1f %.1f %.1f %.1f", cx, t, cx, cy, l, cy), true
	case '┌':
		return f("M%.1f %.1fH%.1fV%.1f", r, cy, cx, btm), true
	case '┐':
		return f("M%.1f %.1fH%.1fV%.1f", l, cy, cx, btm), true
	case '└':
		return f("M%.1f %.1fV%.1fH%.1f", cx, t, cy, r), true
	case '┘':
		return f("M%.1f %.1fV%.1fH%.1f", cx, t, cy, l), true
	case '├':
		return f("M%.1f %.1fV%.1fM%.1f %.1fH%.1f", cx, t, btm, cx, cy, r), true
	case '┤':
		return f("M%.1f %.1fV%.1fM%.1f %.1fH%.1f", cx, t, btm, cx, cy, l), true
	case '┬':
		return f("M%.1f %.1fH%.1fM%.1f %.1fV%.1f", l, cy, r, cx, cy, btm), true
	case '┴':
		return f("M%.1f %.1fH%.1fM%.1f %.1fV%.1f", l, cy, r, cx, t, cy), true
	case '┼':
		return f("M%.1f %.1fH%.1fM%.1f %.1fV%.1f", l, cy, r, cx, t, btm), true
	}
	return "", false
}
