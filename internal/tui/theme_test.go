package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDetectGlyphs(t *testing.T) {
	env := func(kv ...string) func(string) string {
		m := map[string]string{}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return func(k string) string { return m[k] }
	}
	cases := []struct {
		goos string
		env  func(string) string
		want string
	}{
		{"windows", env("WT_SESSION", "x"), "unicode"},
		{"windows", env("TERM_PROGRAM", "vscode"), "unicode"},
		{"windows", env(), "safe"}, // classic console host
		{"linux", env("TERM", "linux", "LANG", "es_ES.UTF-8"), "safe"},
		{"linux", env("TERM", "xterm-256color", "LANG", "C"), "ascii"},
		{"linux", env("TERM", "xterm-256color", "LANG", "en_US.UTF-8"), "unicode"},
		{"darwin", env("TERM_PROGRAM", "Apple_Terminal", "LANG", "en_US.UTF-8"), "unicode"},
		{"linux", env("TERM", "dumb"), "ascii"},
	}
	for _, c := range cases {
		if got := DetectGlyphs(c.env, c.goos); got != c.want {
			t.Errorf("%s %v: got %s want %s", c.goos, c.env("TERM"), got, c.want)
		}
	}
}

// Every glyph set must keep the layout: translating a frame never changes its width.
func TestGlyphSetsKeepLayout(t *testing.T) {
	defer Setup("unicode", "auto")
	frame := sPane.Render(markOK()+"skill · "+gWarn+" "+gMarked+" "+gCursor+" "+bar(0.5, 10)+" … ≥ ◐ ★ ▸ ⇄ ✦ ▣ ◆") + "\nmenú de diseño"
	for _, set := range []string{"safe", "ascii"} {
		Setup(set, "mono")
		out := finish(sPane.Render(markOK()+"skill · "+gWarn+" "+gMarked+" "+gCursor+" "+bar(0.5, 10)+" … ≥ ◐ ★ ▸ ⇄ ✦ ▣ ◆") + "\nmenú de diseño")
		inLines, outLines := strings.Split(frame, "\n"), strings.Split(out, "\n")
		for i := range outLines {
			if lipgloss.Width(outLines[i]) != lipgloss.Width(ansiRe.ReplaceAllString(inLines[i], "")) {
				t.Errorf("%s: line %d changed width:\n%q\n%q", set, i, inLines[i], outLines[i])
			}
		}
		if set == "ascii" {
			for _, r := range out {
				if r > 0x7e && r != '\n' {
					t.Fatalf("ascii frame contains %q", r)
				}
			}
		}
	}
}

func TestCleanRemovesEmojiAndInvisible(t *testing.T) {
	in := "Plan 🎮 the level ✅ now​‮evil️ 👨‍👩‍👧 done — ñandú"
	got := clean(in)
	if got != "Plan the level nowevil done — ñandú" { // zero-width and bidi characters vanish
		t.Fatalf("%q", got)
	}
	if clean("plain text") != "plain text" {
		t.Fatal("plain text must not change")
	}
}

// Caps Lock must not break shortcuts.
func TestUpperCaseShortcuts(t *testing.T) {
	m := model{cursor: map[tab]int{}, marked: map[string]bool{}, rc: &rowCache{}, tab: tProfiles, w: 120, h: 30}
	m, _ = key(m, "N")
	if m.wiz == nil {
		t.Fatal("N must work like n (new profile)")
	}
	m.wiz = nil
	m, _ = key(m, "3")
	if m.tab != tLive {
		t.Fatalf("3 must open the third tab, got %v", m.tab)
	}
	m.tab = tSessions
	if !m.upperBound("R") || m.upperBound("R") != (m.level >= 1) {
		// level 0: R is free, so it falls back to r (reload)
	}
	m.level = 1
	if !m.upperBound("R") || !m.upperBound("D") {
		t.Error("R/D keep their own meaning in a session list")
	}
}

// The preset gallery opens from Profiles with p, shows the monogram, sources and the disclaimer,
// and asks for a second enter before downloading anything.
func TestPresetGallery(t *testing.T) {
	m := model{cursor: map[tab]int{}, marked: map[string]bool{}, rc: &rowCache{}, tab: tProfiles, w: 150, h: 60}
	m, _ = key(m, "p")
	if !m.gal.open || len(m.gal.list) < 10 {
		t.Fatalf("gallery: open=%v presets=%d %s", m.gal.open, len(m.gal.list), m.gal.err)
	}
	v := m.View()
	for _, want := range []string{"Presets", "Process", "Based on what", "█"} {
		if !strings.Contains(v, want) {
			t.Errorf("gallery view misses %q", want)
		}
	}
	m, cmd := key(m, "enter")
	if cmd != nil || m.gal.armed == "" {
		t.Fatal("the first enter must only arm the download")
	}
	m, _ = key(m, "esc")
	if m.gal.open {
		t.Fatal("esc closes the gallery")
	}
}

// Text from files must never be able to drive the terminal: only colour sequences survive.
func TestSanitizeTerminalBlocksEscapeInjection(t *testing.T) {
	esc := "\x1b"
	attacks := []string{
		esc + "]0;owned" + "\a",                       // set the window title (OSC 0)
		esc + "]52;c;ZWNobyBwd25lZA==" + "\a",         // write to the clipboard (OSC 52)
		esc + "]8;;https://evil.example" + esc + "\\", // hyperlink (OSC 8)
		esc + "[2J" + esc + "[H",                      // clear the screen, move the cursor
		esc + "[?1049l",                               // leave the alternate screen
		esc + "[8m" + "hidden" + esc + "[0m",          // conceal text
		"\u009b" + "2J",                               // C1 CSI
		"a\rb\x08c\x00d\x7f",                          // carriage return, backspace, NUL, DEL
	}
	for _, a := range attacks {
		got := SanitizeTerminal("name"+a+"end", true)
		if strings.ContainsAny(got, "\x1b\a\r\x08\x00\x7f\u009b") && !strings.Contains(got, esc+"[0m") {
			t.Errorf("control bytes survived %q → %q", a, got)
		}
		if strings.Contains(got, esc+"]") || strings.Contains(got, esc+"[2J") || strings.Contains(got, esc+"[?") || strings.Contains(got, esc+"[8m") {
			t.Errorf("escape sequence survived %q → %q", a, got)
		}
	}
	styled := sOK.Bold(true).Render("ok") + " " + sSel.Render("selected")
	if SanitizeTerminal(styled, true) != styled {
		t.Errorf("our own colours must pass untouched: %q", styled)
	}
	if got := SanitizeTerminal("a\tb", true); got != "a b" {
		t.Errorf("tabs in a frame become spaces: %q", got)
	}
	if got := SanitizeTerminal("a\tb\r\n", false); got != "a\tb\r\n" {
		t.Errorf("command-line output keeps tabs and CRLF: %q", got)
	}
	// the whole pipeline: a hostile skill name rendered in the list
	m := model{cursor: map[tab]int{}, marked: map[string]bool{}, rc: &rowCache{}, tab: tSkills, w: 100, h: 20, loaded: true}
	m.inv.Items = append(m.inv.Items, inventoryItem("evil"+esc+"]0;pwned\a"+esc+"[2J"))
	if v := m.View(); strings.Contains(v, esc+"]0") || strings.Contains(v, esc+"[2J") || strings.Contains(v, "\a") {
		t.Fatalf("hostile name reached the terminal: %q", v)
	}
}

// Long lists show a scrollbar with arrows when there is more above or below.
func TestScrollbar(t *testing.T) {
	if bar := scrollbar(5, 0, 10); strings.Contains(strings.Join(bar, ""), "█") {
		t.Error("no bar when everything fits")
	}
	bar := scrollbar(100, 0, 10)
	if !strings.Contains(bar[0], "█") || !strings.Contains(bar[9], "▼") {
		t.Errorf("top of a long list: %q", bar)
	}
	bar = scrollbar(100, 90, 10)
	if !strings.Contains(bar[0], "▲") || !strings.Contains(bar[9], "█") {
		t.Errorf("bottom of a long list: %q", bar)
	}
	bar = scrollbar(100, 45, 10)
	if !strings.Contains(bar[0], "▲") || !strings.Contains(bar[9], "▼") || !strings.Contains(strings.Join(bar[1:9], ""), "█") {
		t.Errorf("middle: %q", bar)
	}
	if got := position(10, 20, 230); !strings.Contains(got, "11–30 of 230") {
		t.Errorf("position: %q", got)
	}
}
