package tui

import (
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ---- terminal compatibility ----
//
// catude draws with three glyph sets so it looks right in every terminal and font:
//
//	unicode  modern terminals (Windows Terminal, VS Code, iTerm2, WezTerm, kitty, GNOME, Konsole…)
//	safe     only WGL4 characters, present in every console font (classic Windows console with
//	         Consolas / Lucida Console, the Linux text console, old SSH clients)
//	ascii    7-bit only, for terminals without UTF-8
//
// The UI is written with unicode glyphs; for the other sets the finished frame is translated
// one cell for one cell, so the layout never shifts. Text that comes from files (names,
// descriptions, titles) is cleaned with clean(): emoji, variation selectors, zero-width and
// bidi characters are removed before layout, because their width differs between terminals.
//
// Override with CATUDE_GLYPHS=unicode|safe|ascii and CATUDE_THEME=auto|dark|light|high-contrast|mono.

// Glyphs is the active glyph set name.
var Glyphs = "unicode"

// Theme is the active colour theme name.
var Theme = "auto"

// Accent is the highlight colour chosen in Preferences ("" = amber).
var Accent = ""

// every accent keeps at least 4.5:1 contrast against the text drawn on it and against the background
var accents = map[string]lipgloss.AdaptiveColor{
	"amber":   {Light: "#C2410C", Dark: "#F59E0B"},
	"blue":    {Light: "#1D4ED8", Dark: "#60A5FA"},
	"green":   {Light: "#15803D", Dark: "#4ADE80"},
	"magenta": {Light: "#A21CAF", Dark: "#E879F9"},
	"cyan":    {Light: "#0E7490", Dark: "#22D3EE"},
	"red":     {Light: "#B91C1C", Dark: "#F87171"},
}

var frameReplacer *strings.Replacer

const (
	gOK     = "✓"
	gWarn   = "▲"
	gFail   = "✗"
	gOff    = "○"
	gBusy   = "◌"
	gMarked = "●"
	gLogo   = "≡"
	gCursor = "›"
)

var safeMap = []string{
	"◐", "|", "◓", "/", "◑", "-", "◒", "\\",
	"✓", "√", "✗", "x", "★", "*", "▏", "|", "◌", "○", "╭", "┌", "╮", "┐", "╰", "└", "╯", "┘",
	"▣", "■", "◆", "♦", "⇄", "↔", "✦", "*", "▸", "►", "⏎", "↵",
}

var asciiMap = []string{
	"◐", "|", "◓", "/", "◑", "-", "◒", "\\", "≥", ">", "≈", "~",
	"✓", "+", "✗", "x", "▲", "!", "○", "-", "●", "*", "≡", "=", "…", "~", "█", "#", "·", ".",
	"│", "|", "─", "-", "╭", "+", "╮", "+", "╰", "+", "╯", "+", "┌", "+", "┐", "+", "└", "+", "┘", "+",
	"├", "+", "┤", "+", "┬", "+", "┴", "+", "┼", "+", "★", "*", "→", ">", "←", "<", "↑", "^", "↓", "v",
	"•", "*", "▏", "_", "◌", "o", "▣", "#", "◆", "*", "⇄", "=", "✦", "*", "▸", ">", "›", ">", "‹", "<",
	"░", ".", "√", "+", "■", "#", "♦", "*", "↔", "=", "►", ">", "↵", "<",
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n", "ç", "c", "à", "a", "è", "e",
	"ì", "i", "ò", "o", "ù", "u", "â", "a", "ê", "e", "î", "i", "ô", "o", "û", "u", "ä", "a", "ë", "e",
	"ï", "i", "ö", "o", "Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ü", "U", "Ñ", "N", "Ç", "C",
	"¿", "?", "¡", "!", "º", "o", "ª", "a", "€", "E", "°", "o",
	"«", "<", "»", ">", "—", "-", "–", "-", "‘", "'", "’", "'", "“", "\"", "”", "\"", "×", "x", "±", "+", "´", "'", "¨", "\"",
}

// Setup chooses glyphs and colours for the current terminal. Call it once before rendering.
func Setup(glyphs, theme string) {
	if glyphs == "" {
		glyphs = os.Getenv("CATUDE_GLYPHS")
	}
	if glyphs == "" || glyphs == "auto" {
		glyphs = DetectGlyphs(os.Getenv, runtime.GOOS)
	}
	Glyphs = glyphs
	switch glyphs {
	case "safe":
		frameReplacer = strings.NewReplacer(safeMap...)
	case "ascii":
		frameReplacer = strings.NewReplacer(asciiMap...)
	default:
		Glyphs, frameReplacer = "unicode", nil
	}
	if theme == "" {
		theme = os.Getenv("CATUDE_THEME")
	}
	if theme == "" || theme == "auto" {
		theme = "auto"
		if os.Getenv("NO_COLOR") != "" || lipgloss.ColorProfile() == termenv.Ascii {
			theme = "mono"
		}
	}
	applyTheme(theme)
}

// DetectGlyphs picks a glyph set from the environment (getenv is os.Getenv; injectable for tests).
func DetectGlyphs(getenv func(string) string, goos string) string {
	if goos != "windows" {
		loc := strings.ToUpper(getenv("LC_ALL") + getenv("LC_CTYPE") + getenv("LANG"))
		if loc != "" && !strings.Contains(loc, "UTF-8") && !strings.Contains(loc, "UTF8") {
			return "ascii"
		}
		if getenv("TERM") == "linux" { // the Linux text console
			return "safe"
		}
		if getenv("TERM") == "dumb" {
			return "ascii"
		}
		return "unicode"
	}
	modern := getenv("WT_SESSION") != "" || getenv("TERM_PROGRAM") != "" || getenv("WEZTERM_EXECUTABLE") != "" ||
		getenv("ALACRITTY_LOG") != "" || getenv("TERMINAL_EMULATOR") != "" || getenv("ConEmuANSI") == "ON" ||
		strings.Contains(getenv("TERM"), "xterm") || getenv("KITTY_WINDOW_ID") != ""
	if modern {
		return "unicode"
	}
	return "safe" // classic Windows console host (conhost) with Consolas / Lucida Console / raster fonts
}

// finish adapts a rendered frame to the active glyph set (cell-for-cell, layout preserved).
func finish(frame string) string {
	frame = SanitizeTerminal(frame, true)
	if frameReplacer == nil {
		return frame
	}
	frame = frameReplacer.Replace(frame)
	if Glyphs == "ascii" {
		var b strings.Builder
		b.Grow(len(frame))
		for _, r := range frame {
			if r < 0x80 {
				b.WriteRune(r)
			} else {
				b.WriteByte('?')
			}
		}
		frame = b.String()
	}
	return frame
}

// clean removes characters whose width or rendering differs between terminals and fonts (emoji,
// variation selectors, zero-width joiners, bidi controls, tag characters, control characters).
func clean(s string) string {
	ok := true
	for _, r := range s {
		if dropRune(r) {
			ok = false
			break
		}
	}
	if ok {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if !dropRune(r) {
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func dropRune(r rune) bool {
	switch {
	case r == '\n' || r == '\t':
		return false
	case unicode.IsControl(r):
		return true
	case r >= 0x1F000 && r <= 0x1FAFF, // emoji, pictographs, flags
		r >= 0x2600 && r <= 0x27BF, // misc symbols and dingbats (most render as emoji)
		r >= 0x2B00 && r <= 0x2BFF, // arrows/stars often drawn as emoji
		r >= 0xFE00 && r <= 0xFE0F, // variation selectors
		r == 0x200D || (r >= 0x200B && r <= 0x200F) || (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069),
		r >= 0xE0000 && r <= 0xE007F, // tags
		r == 0x20E3 || r == 0x3030 || r == 0x303D || r == 0x3297 || r == 0x3299 || r == 0x00A9 || r == 0x00AE || r == 0x2122:
		return true
	}
	return false
}

// ---- themes ----

var (
	accent   lipgloss.TerminalColor = lipgloss.AdaptiveColor{Light: "#C2410C", Dark: "#F59E0B"}
	muted    lipgloss.TerminalColor = lipgloss.AdaptiveColor{Light: "#4B5563", Dark: "#9CA3AF"}
	sTitle   lipgloss.Style
	sTab     lipgloss.Style
	sTabOn   lipgloss.Style
	sSel     lipgloss.Style
	sMuted   lipgloss.Style
	sHigh    lipgloss.Style
	sWarn    lipgloss.Style
	sOK      lipgloss.Style
	sPane    lipgloss.Style
	sStatus  lipgloss.Style
	sHeading lipgloss.Style
)

func init() { applyTheme("auto") }

func applyTheme(name string) {
	Theme = name
	border := lipgloss.RoundedBorder()
	switch Glyphs {
	case "safe":
		border = lipgloss.NormalBorder()
	case "ascii":
		border = lipgloss.ASCIIBorder()
	}
	ok := lipgloss.TerminalColor(lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#22C55E"})
	warn := lipgloss.TerminalColor(lipgloss.AdaptiveColor{Light: "#A16207", Dark: "#EAB308"})
	high := lipgloss.TerminalColor(lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#EF4444"})
	onAccent := lipgloss.TerminalColor(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#111827"})
	accent = lipgloss.AdaptiveColor{Light: "#C2410C", Dark: "#F59E0B"}
	muted = lipgloss.AdaptiveColor{Light: "#4B5563", Dark: "#9CA3AF"}
	switch name {
	case "dark":
		accent, muted, onAccent = lipgloss.Color("#F59E0B"), lipgloss.Color("#9CA3AF"), lipgloss.Color("#111827")
		ok, warn, high = lipgloss.Color("#22C55E"), lipgloss.Color("#EAB308"), lipgloss.Color("#EF4444")
	case "light":
		accent, muted, onAccent = lipgloss.Color("#C2410C"), lipgloss.Color("#4B5563"), lipgloss.Color("#FFFFFF")
		ok, warn, high = lipgloss.Color("#15803D"), lipgloss.Color("#A16207"), lipgloss.Color("#B91C1C")
	case "high-contrast":
		accent = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFF00"}
		muted = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}
		onAccent = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}
		ok = lipgloss.AdaptiveColor{Light: "#005A00", Dark: "#00FF00"}
		warn = lipgloss.AdaptiveColor{Light: "#7A4A00", Dark: "#FFD700"}
		high = lipgloss.AdaptiveColor{Light: "#A00000", Dark: "#FF6060"}
	}
	// the accent colour is the user's choice, except in the themes built for contrast
	if a, found := accents[Accent]; found && name != "high-contrast" && name != "mono" {
		switch name {
		case "dark":
			accent = lipgloss.Color(a.Dark)
		case "light":
			accent = lipgloss.Color(a.Light)
		default:
			accent = a
		}
	}
	if name == "mono" {
		plain := lipgloss.NewStyle()
		sTitle, sMuted, sStatus = plain.Bold(true), plain, plain
		sHigh, sWarn, sOK, sHeading = plain.Bold(true), plain.Bold(true), plain, plain.Bold(true).Underline(true)
		sTab, sTabOn = plain.Padding(0, 1), plain.Padding(0, 1).Reverse(true).Bold(true)
		sSel = plain.Reverse(true).Bold(true)
		sPane = plain.Border(border).Padding(0, 1)
		return
	}
	sTitle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	sTab = lipgloss.NewStyle().Padding(0, 1).Foreground(muted)
	sTabOn = lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(onAccent).Background(accent)
	sSel = lipgloss.NewStyle().Bold(true).Foreground(onAccent).Background(accent)
	sMuted = lipgloss.NewStyle().Foreground(muted)
	sHigh = lipgloss.NewStyle().Bold(true).Foreground(high)
	sWarn = lipgloss.NewStyle().Bold(true).Foreground(warn)
	sOK = lipgloss.NewStyle().Foreground(ok)
	sPane = lipgloss.NewStyle().Border(border).BorderForeground(muted).Padding(0, 1)
	sStatus = lipgloss.NewStyle().Foreground(muted)
	sHeading = lipgloss.NewStyle().Bold(true)
	if name == "high-contrast" {
		sTab = sTab.Underline(true)
		sMuted = lipgloss.NewStyle().Foreground(muted)
	}
}

// status column: glyph + space, coloured, never colour alone (the glyph carries the meaning).
func markOK() string   { return sOK.Bold(true).Render(gOK) + " " }
func markWarn() string { return sWarn.Render(gWarn) + " " }
func markFail() string { return sHigh.Render(gFail) + " " }
func markOff() string  { return sMuted.Render(gOff) + " " }

// Writer adapts command-line output to the active glyph set too.
func Writer(w io.Writer) io.Writer { return glyphWriter{w} }

type glyphWriter struct{ w io.Writer }

func (g glyphWriter) Write(p []byte) (int, error) {
	// command-line output is plain text: no escape sequence at all and no carriage returns, so text
	// from a file can neither hide part of a line nor overwrite one
	s := strings.ReplaceAll(ansiRe.ReplaceAllString(SanitizeTerminal(string(p), false), ""), "\r", "")
	if frameReplacer != nil {
		s = frameReplacer.Replace(s)
		if Glyphs == "ascii" {
			s = asciiOnly(s)
		}
	}
	if _, err := io.WriteString(g.w, s); err != nil {
		return 0, err
	}
	return len(p), nil
}

// sgrConceals reports whether a colour sequence hides text (parameter 8, in any spelling), skipping
// the arguments of extended colours (38;5;n, 38;2;r;g;b).
func sgrConceals(params string) bool {
	ps := strings.FieldsFunc(params, func(r rune) bool { return r == ';' || r == ':' })
	for i := 0; i < len(ps); i++ {
		n, err := strconv.Atoi(ps[i])
		if err != nil {
			return true // not a number: refuse the whole sequence
		}
		switch n {
		case 8:
			return true
		case 38, 48, 58:
			if i+1 < len(ps) {
				switch ps[i+1] {
				case "5":
					i += 2
				case "2":
					i += 4
				}
			}
		}
	}
	return false
}

func asciiOnly(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x80 {
			b.WriteRune(r)
		} else {
			b.WriteByte('?')
		}
	}
	return b.String()
}

// SanitizeTerminal is the last line of defence against terminal escape injection: text that comes
// from files (skill names, session titles, config values, Rigcards) must never be able to move the
// cursor, clear the screen, set the window title, write to the clipboard (OSC 52) or open hyperlinks.
// The only escape sequence that survives is colour/style (CSI … m) without "conceal"; every other
// ESC, every C0 control except newline (and tab outside frames) and every C1 control is removed.
func SanitizeTerminal(s string, frame bool) string {
	clean := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < 0x20 && c != '\n') || c == 0x7f || c == 0xc2 {
			clean = false
			break
		}
	}
	if clean && utf8.ValidString(s) { // a lone 0x80–0x9F byte is a control code on 8-bit terminals
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == 0x1b:
			j := i + 1
			if j < len(s) && s[j] == '[' {
				j++
				conceal := false
				start := j
				for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == ';' || s[j] == ':') {
					j++
				}
				if j < len(s) && s[j] == 'm' {
					conceal = sgrConceals(s[start:j])
					if !conceal {
						b.WriteString(s[i : j+1])
					}
					i = j + 1
					continue
				}
			}
			i++ // a lone or unknown escape: drop the ESC, the rest prints as plain text
		case c == '\n':
			b.WriteByte(c)
			i++
		case c == '\t':
			if frame {
				b.WriteByte(' ')
			} else {
				b.WriteByte(c)
			}
			i++
		case c == '\r' && !frame:
			b.WriteByte(c)
			i++
		case c < 0x20 || c == 0x7f:
			i++
		case c < 0x80:
			b.WriteByte(c)
			i++
		default:
			r, size := utf8.DecodeRuneInString(s[i:])
			if !(r >= 0x80 && r <= 0x9f) && !(r == utf8.RuneError && size == 1) {
				b.WriteString(s[i : i+size])
			}
			i += size
		}
	}
	return b.String()
}
