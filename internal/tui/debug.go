package tui

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/profiles"
)

// lastRender is the duration of the last frame (shown in the debug panel).
var lastRender atomic.Int64

// claudeVersion is the version of the claude found on PATH (first install), for the header corner.
func (m model) claudeVersion() string {
	for _, in := range m.instances {
		if in.Version != "" {
			return in.Version
		}
	}
	return ""
}

// DebugInfo lists what is useful in a bug report. It never includes secrets or file contents.
func DebugInfo(m model) [][2]string {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	prof := map[termenv.Profile]string{termenv.TrueColor: "truecolor", termenv.ANSI256: "256 colours", termenv.ANSI: "16 colours", termenv.Ascii: "no colour"}[lipgloss.ColorProfile()]
	term := strings.TrimSpace(strings.Join([]string{os.Getenv("TERM"), os.Getenv("TERM_PROGRAM"), map[bool]string{true: "Windows Terminal", false: ""}[os.Getenv("WT_SESSION") != ""]}, " "))
	jobs := []string{}
	for _, j := range []struct {
		on   bool
		name string
	}{{m.ctx.running, "context"}, {m.st.measuring > 0, fmt.Sprintf("storage(%d)", m.st.measuring)}, {m.search.running, "search"}, {m.liveRunning, "mcp health"}, {m.tools.running, "tools"}, {m.usage.running, "usage"}} {
		if j.on {
			jobs = append(jobs, j.name)
		}
	}
	return [][2]string{
		{"catude", m.version + " · " + runtime.Version() + " · " + runtime.GOOS + "/" + runtime.GOARCH},
		{"Claude Code", firstNonEmpty(m.claudeVersion(), "not found") + fmt.Sprintf(" · %d installs found", len(m.instances))},
		{"terminal", firstNonEmpty(term, "unknown") + fmt.Sprintf(" · %dx%d · %s · dark background %v", m.w, m.h, prof, lipgloss.HasDarkBackground())},
		{"glyphs / theme", Glyphs + " / " + Theme + map[bool]string{true: " · NO_COLOR set", false: ""}[os.Getenv("NO_COLOR") != ""]},
		{"last frame", time.Duration(lastRender.Load()).Round(10 * time.Microsecond).String()},
		{"inventory", fmt.Sprintf("%d items · %d findings · %d load errors · fingerprint %.8s", len(m.inv.Items), len(m.findings), len(m.inv.Errors), m.fp)},
		{"background jobs", firstNonEmpty(strings.Join(jobs, ", "), "none")},
		{"memory", fmt.Sprintf("%.1f MB in use · %d goroutines", float64(ms.Alloc)/(1<<20), runtime.NumGoroutine())},
		{"configuration", m.env.ClaudeDir},
		{"project", m.env.ProjectDir},
		{"profiles", profiles.Root() + " · active: " + firstNonEmpty(m.active, "none")},
		{"cache", firstNonEmpty(cacheDir(), "unavailable")},
		{"live MCP", fmt.Sprintf("%d servers · checked %s", len(m.liveMCP), ago(m.liveAt))},
	}
}

func (m model) debugView(h int) string {
	var b strings.Builder
	b.WriteString(sTitle.Render(T("Debug")) + sMuted.Render("  F12 or ctrl+o closes · catude debug prints the same for bug reports (no secrets, no file contents)") + "\n\n")
	for _, kv := range DebugInfo(m) {
		b.WriteString(fmt.Sprintf("  %-16s %s\n", kv[0], clipEnd(kv[1], max(20, m.w-26))))
	}
	for i, e := range m.inv.Errors {
		if i == 8 {
			break
		}
		b.WriteString(sWarn.Render("  error "+clipEnd(e, m.w-14)) + "\n")
	}
	return b.String()
}

type verifiedMsg struct {
	name string
	c    profiles.Check
	err  error
}

// verifyProfile asks Claude Code (in the background) whether every item of the profile loads.
func (m model) verifyProfile(p profiles.Profile) tea.Cmd {
	dir, _ := m.env.RunDir()
	return func() tea.Msg {
		c, err := profiles.Verify(contextBG(), p, profiles.LaunchOptions{Exclusive: p.Options.Exclusive}, dir)
		return verifiedMsg{p.Name, c, err}
	}
}

// DebugReport is DebugInfo for the command line (loads what the TUI would).
func DebugReport(env paths.Env, version string) [][2]string {
	m := newModel(env, version)
	m.inv = inventory.Load(env)
	m.instances, m.cfgDirs = env.CodeInstances(), env.ConfigDirs()
	return DebugInfo(m)
}
