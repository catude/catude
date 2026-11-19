package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/storage"
)

// stState is the Storage tab: transcripts, checkpoints, caches, scratchpads and Claude Desktop data,
// with sizes measured in the background and a safe clean for the categories that allow it.
type stState struct {
	entries   []storage.Entry
	loaded    bool
	measuring int // entries still being measured
	sel       int
	armed     string // key of the entry waiting for a second "c"
	ret       *storage.Retention
	at        time.Time
}

type stCatalogMsg struct{ es []storage.Entry }
type stMeasuredMsg struct{ e storage.Entry }
type stRetentionMsg struct{ r storage.Retention }
type stCleanedMsg struct {
	key string
	r   storage.CleanResult
	err error
}

type stCached struct {
	Entries []storage.Entry
	At      time.Time
}

func (m model) stCacheName() string { return "storage-" + cacheKey(m.env.ClaudeDir) }

// ensureStorage lists the locations (instant), shows cached sizes and measures again in the background.
func (m *model) ensureStorage() tea.Cmd {
	if m.st.loaded || m.st.measuring > 0 {
		return nil
	}
	env := m.env
	var c stCached
	if readCache(m.stCacheName(), &c) {
		m.st.entries, m.st.at = c.Entries, c.At
	}
	return tea.Batch(
		func() tea.Msg { return stCatalogMsg{storage.Catalog(env)} },
		func() tea.Msg { return stRetentionMsg{storage.ReadRetention(env)} },
	)
}

func (m *model) onStCatalog(es []storage.Entry) tea.Cmd {
	old := map[string]storage.Entry{}
	for _, e := range m.st.entries {
		old[e.Key] = e
	}
	for i, e := range es {
		if o, ok := old[e.Key]; ok && o.Measured { // keep the cached size until the new one arrives
			es[i].Bytes, es[i].Files, es[i].Newest = o.Bytes, o.Files, o.Newest
		}
	}
	m.st.entries, m.st.loaded, m.st.measuring = es, true, len(es)
	cmds := make([]tea.Cmd, 0, len(es))
	for _, e := range es {
		e := e
		cmds = append(cmds, func() tea.Msg { return stMeasuredMsg{storage.Measure(context.Background(), e)} })
	}
	return tea.Batch(cmds...)
}

func (m *model) onStMeasured(e storage.Entry) {
	for i := range m.st.entries {
		if m.st.entries[i].Key == e.Key {
			m.st.entries[i] = e
		}
	}
	m.st.measuring--
	if m.st.measuring <= 0 {
		m.st.measuring, m.st.at = 0, time.Now()
		storage.Sort(m.st.entries)
		writeCache(m.stCacheName(), stCached{m.st.entries, m.st.at})
	}
}

func (m *model) stSelected() *storage.Entry {
	if m.st.sel < 0 || m.st.sel >= len(m.st.entries) {
		return nil
	}
	return &m.st.entries[m.st.sel]
}

// stClean asks for a second press, then cleans the selected safe entry in the background.
func (m *model) stClean() tea.Cmd {
	e := m.stSelected()
	if e == nil {
		return nil
	}
	if !e.Safe {
		m.status = e.Label + ": not safe to clean from catude (" + e.Note + ")"
		return nil
	}
	if m.st.armed != e.Key {
		m.st.armed = e.Key
		m.status = fmt.Sprintf("press c again to delete what is older than 1 hour in %s (%s)", e.Label, humanBytes(e.Bytes))
		return nil
	}
	m.st.armed = ""
	m.status = "cleaning " + e.Label + "…"
	ent := *e
	return func() tea.Msg {
		r, err := storage.Clean(ent)
		return stCleanedMsg{ent.Key, r, err}
	}
}

func (m *model) onStCleaned(msg stCleanedMsg) tea.Cmd {
	if msg.err != nil {
		m.status = "clean: " + msg.err.Error()
		return nil
	}
	m.status = fmt.Sprintf("freed %s · %d files removed · %d kept (recent or in use)", humanBytes(msg.r.Freed), msg.r.Removed, msg.r.Kept)
	for _, e := range m.st.entries {
		if e.Key == msg.key {
			e := e
			m.st.measuring++
			return func() tea.Msg { return stMeasuredMsg{storage.Measure(context.Background(), e)} }
		}
	}
	return nil
}

// openFolder shows a location in the system file manager (or opens a file with its default app).
// A variable so tests do not open windows.
var openFolder = func(p string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		c = exec.Command("explorer", p)
	case "darwin":
		c = exec.Command("open", p)
	default:
		c = exec.Command("xdg-open", p)
	}
	return c.Start()
}

func (m model) storageView(h int) string {
	var b strings.Builder
	var total int64
	for _, e := range m.st.entries {
		total += e.Bytes
	}
	state := "measured " + ago(m.st.at)
	if m.st.measuring > 0 {
		state = fmt.Sprintf("%s measuring %d folders in the background…", m.spinner(), m.st.measuring)
	}
	b.WriteString(sHeading.Render(T("Storage")) + sMuted.Render(fmt.Sprintf("  %s on disk · %s · o open · c c clean (safe items only) · r measure again", humanBytes(total), state)) + "\n")
	if r := m.st.ret; r != nil {
		line := fmt.Sprintf("Claude Code deletes transcripts after %d days (cleanupPeriodDays%s).", r.Days, map[bool]string{true: "", false: ", default"}[r.Explicit])
		if r.Expiring > 0 {
			b.WriteString(sWarn.Render(fmt.Sprintf("%s %s %d of %d sessions will be deleted within 7 days. Export them from Sessions (x) or raise cleanupPeriodDays in ~/.claude/settings.json.", gWarn, line, r.Expiring, r.Transcrip)) + "\n")
		} else {
			b.WriteString(sMuted.Render(line+fmt.Sprintf(" %d sessions, none expiring this week.", r.Transcrip)) + "\n")
		}
	}
	if len(m.st.entries) == 0 {
		b.WriteString("\n" + m.spinner() + " listing…")
		return b.String()
	}
	type line struct {
		text string
		idx  int
	}
	var lines []line
	group := ""
	for i, e := range m.st.entries {
		if e.Group != group {
			group = e.Group
			lines = append(lines, line{"", -1}, line{sHeading.Render(group), -1})
		}
		size := humanBytes(e.Bytes)
		switch {
		case !e.Measured && e.Bytes == 0:
			size = "…"
		case e.Partial:
			size = "≥" + size
		}
		mark := markOff()
		if e.Safe {
			mark = markOK()
		}
		t := fmt.Sprintf("%s%-52s %9s %8s  %s", mark, clipEnd(e.Label, 52), size, fmt.Sprintf("%d files", e.Files), sMuted.Render(clipEnd(e.Note, max(10, m.w-100))))
		if i == m.st.sel {
			t = sSel.Render(gCursor + " " + stripStyles(t, m.w-10))
		} else {
			t = "  " + t
		}
		lines = append(lines, line{t, i})
	}
	cur := 0
	for j, l := range lines {
		if l.idx == m.st.sel {
			cur = j
		}
	}
	head := strings.Count(b.String(), "\n")
	room := max(3, h-head-3)
	start := clamp(cur-room+2, 0, max(0, len(lines)-room))
	var shown []string
	for j := start; j < len(lines) && j < start+room; j++ {
		shown = append(shown, lines[j].text)
	}
	b.WriteString(withScrollbar(shown, len(lines), start, min(room, len(lines)), m.w-6) + "\n")
	if e := m.stSelected(); e != nil {
		b.WriteString("\n" + sMuted.Render(shorten(e.Path, m.w-8)))
		if !e.Newest.IsZero() {
			b.WriteString(sMuted.Render(" · last change " + ago(e.Newest)))
		}
		if e.Safe {
			b.WriteString("\n" + sOK.Render(gOK+" safe to clean: only files untouched for 1 hour are removed"))
		} else {
			b.WriteString("\n" + sMuted.Render(gOff+" kept: catude never deletes this"))
		}
	}
	return b.String()
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}
