package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/profiles"
	"github.com/catude/catude/internal/sessions"
)

// Every tab must render on an empty machine and on a tiny terminal without panicking.
func TestSnapshotEmptyMachine(t *testing.T) {
	r := t.TempDir()
	env := paths.Env{GOOS: "linux", Home: r, ClaudeDir: filepath.Join(r, ".claude"), ClaudeJSON: filepath.Join(r, ".claude.json"), ProjectDir: r}
	for _, size := range [][2]int{{140, 34}, {40, 8}} {
		out := Snapshot(env, "test", size[0], size[1], false, false)
		for _, n := range tabNames {
			if !strings.Contains(out, n) && size[0] > 100 {
				t.Errorf("tab %s missing at %dx%d", n, size[0], size[1])
			}
		}
	}
}

func key(m model, s string) (model, tea.Cmd) {
	var k tea.KeyMsg
	switch s {
	case "enter":
		k = tea.KeyMsg{Type: tea.KeyEnter}
	case " ":
		k = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	default:
		k = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	nm, cmd := m.Update(k)
	return nm.(model), cmd
}

// Mark two skills with space, press P, type a name, enter: a work profile is built, audited and installed.
func TestMarkAndSaveWorkProfile(t *testing.T) {
	t.Setenv("CATUDE_HOME", t.TempDir())
	r := t.TempDir()
	env := paths.Env{GOOS: "linux", Home: r, ClaudeDir: filepath.Join(r, ".claude"), ClaudeJSON: filepath.Join(r, ".claude.json"), ProjectDir: r}
	for _, n := range []string{"level-design", "playtest"} {
		p := filepath.Join(env.ClaudeDir, "skills", n, "SKILL.md")
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("---\nname: "+n+"\ndescription: Game design helper. Use when designing.\n---\nx\n"), 0o644)
	}
	inv := inventory.Load(env)
	m := model{env: env, cursor: map[tab]int{}, cache: &sessions.Cache{}, w: 140, h: 34, inv: inv, loaded: true, marked: map[string]bool{}, tab: tSkills}
	m, _ = key(m, " ")
	m, _ = key(m, " ")
	if len(m.marked) != 2 {
		t.Fatalf("marked: %v", m.marked)
	}
	if !strings.Contains(m.View(), gMarked) {
		t.Error("marked items must be visible in the list")
	}
	m, _ = key(m, "P")
	if m.wiz == nil || len(m.wiz.sel) != 2 {
		t.Fatalf("P must open the profile wizard with the marked items: %+v", m.wiz)
	}
	m, _ = key(m, "enter")                                     // step 2: name and options
	for _, s := range []string{"g", "a", "m", "e", "/", "d"} { // "/" is not a valid name character
		m, _ = key(m, s)
	}
	if m.wiz.name != "gamed" {
		t.Fatalf("name input: %q", m.wiz.name)
	}
	m, cmd := key(m, "enter")
	if cmd == nil {
		t.Fatal("enter must start building the profile")
	}
	msg := cmd()
	saved, ok := msg.(profileSavedMsg)
	if !ok || saved.err != nil {
		t.Fatalf("save: %#v", msg)
	}
	nm, _ := m.Update(saved)
	m = nm.(model)
	if len(m.marked) != 0 || !strings.Contains(m.status, "gamed saved") {
		t.Fatalf("after save: %v %q", m.marked, m.status)
	}
	ps, _ := profiles.List()
	if len(ps) != 1 || len(ps[0].Skills) != 2 {
		t.Fatalf("installed: %+v", ps)
	}
	m.workProfiles, m.tab = ps, tProfiles
	if v := m.View(); !strings.Contains(v, "gamed") || !strings.Contains(v, "2 skills") {
		t.Errorf("Profiles tab must list the work profile:\n%s", v)
	}
}
