package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/catude/catude/internal/audit"
	"github.com/catude/catude/internal/ctxcomp"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/profiles"
	"github.com/catude/catude/internal/sessions"
)

// DemoStep is one moment of a scripted demo (README screenshots and GIFs, made from a fixture home
// with invented data — see scripts/demo).
type DemoStep struct {
	Do    func()   // runs first (for example, a session writes another line to its transcript)
	Keys  []string // key presses: right, left, up, down, enter, esc, tab, space, pgdown, or text
	Frame bool     // record the screen after this step
	Hold  int      // how long the frame stays, in units the caller chooses
	Name  string   // a name for the frame (screenshots)
}

// DemoFrame is the screen after a step, with colours.
type DemoFrame struct {
	Name string
	Hold int
	View string
}

// DemoFrames runs catude on env with the given keys and returns the screens. Background work of each
// step is waited for (up to two seconds); timers are not.
func DemoFrames(env paths.Env, version string, w, h int, steps []DemoStep) []DemoFrame {
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)
	Setup("unicode", "dark")
	reduceMotion = true // a still spinner glyph instead of a frame that changes between captures
	inv := inventory.Load(env)
	m := model{env: env, version: version, cursor: map[tab]int{}, cache: &sessions.Cache{}, w: w, h: h,
		inv: inv, findings: audit.Run(inv), loaded: true, instances: env.CodeInstances(), cfgDirs: env.ConfigDirs(),
		marked: map[string]bool{}, rc: &rowCache{}, projCWD: map[string]string{}}
	for i := range tabNames {
		m.prefs.Tabs = append(m.prefs.Tabs, tabID(tab(i)))
	}
	m.workProfiles, _ = profiles.List()
	m.projects, _ = sessions.ListProjects(env.ProjectsDir())
	if msg, ok := m.loadProjInfo(m.projects)().(projInfoMsg); ok {
		m.projCWD, m.latest = msg.cwd, msg.latest
	}
	// the invented machine has no Claude Code to run /context: a measurement of the same shape stands in
	writeCache(m.ctxCacheName(""), ctxCached{Comp: demoContext, At: time.Now()})
	var out []DemoFrame
	for _, st := range steps {
		if st.Do != nil {
			st.Do()
		}
		for _, k := range st.Keys {
			nm, cmd := m.Update(demoKey(k))
			m = demoSettle(nm.(model), cmd)
		}
		if len(st.Keys) == 0 && m.tab == tLive { // a pause on Live: read the sessions again
			m = demoSettle(m, m.readLive())
		}
		if st.Frame {
			m.status = ""
			out = append(out, DemoFrame{Name: st.Name, Hold: st.Hold, View: m.View()})
		}
	}
	return out
}

func demoKey(k string) tea.KeyMsg {
	switch k {
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

// demoSettle runs the commands a key started and feeds their results back, for up to two seconds.
// Timers (spinner, refresh ticks) are dropped so the demo controls time itself.
func demoSettle(m model, cmd tea.Cmd) model {
	msgs := make(chan tea.Msg, 64)
	pending := 0
	var start func(c tea.Cmd)
	start = func(c tea.Cmd) {
		if c == nil {
			return
		}
		pending++
		go func() { msgs <- c() }()
	}
	start(cmd)
	deadline := time.After(2 * time.Second)
	for pending > 0 {
		select {
		case msg := <-msgs:
			pending--
			switch v := msg.(type) {
			case nil:
			case tea.BatchMsg:
				for _, c := range v {
					start(c)
				}
			case tickMsg, liveTickMsg, spinMsg:
			default:
				nm, c := m.Update(v)
				m = nm.(model)
				start(c)
			}
		case <-deadline:
			return m
		}
	}
	return m
}

// demoContext is an invented /context result for the demo machine (its skills, MCP servers and memory).
var demoContext = ctxcomp.Composition{
	Model: "claude-opus-5-5", Used: 41800, Max: 1000000,
	Categories: []ctxcomp.Row{
		{Name: "System prompt", Tokens: 3100}, {Name: "System tools", Tokens: 11900}, {Name: "MCP tools", Tokens: 14600},
		{Name: "Custom agents", Tokens: 1200}, {Name: "Memory files", Tokens: 900}, {Name: "Skills", Tokens: 2600},
		{Name: "Messages", Tokens: 7500}, {Name: "Free space", Tokens: 958200},
	},
	Memory: []ctxcomp.Row{{Name: "~/.claude/CLAUDE.md", Source: "User", Tokens: 520}, {Name: "CLAUDE.md", Source: "Project", Tokens: 380}},
	Skills: []ctxcomp.Row{{Name: "frontend-design", Source: "user", Tokens: 410}, {Name: "code-review", Source: "user", Tokens: 330},
		{Name: "db-migrations", Source: "user", Tokens: 290}, {Name: "checkout-flows", Source: "project", Tokens: 270}, {Name: "tdd", Source: "user", Tokens: 240}},
	MCPTools: []ctxcomp.Row{{Name: "create_pull_request", Source: "github", Tokens: 1900}, {Name: "browser_snapshot", Source: "playwright", Tokens: 1500},
		{Name: "search_issues", Source: "github", Tokens: 1200}, {Name: "query", Source: "postgres", Tokens: 900}, {Name: "list_issues", Source: "linear", Tokens: 700}},
}
