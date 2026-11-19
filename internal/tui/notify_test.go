package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/live"
)

// A session that goes from busy to idle is announced once, in any tab; the bell rings only when asked.
func TestWatchAnnouncesFinishedSessions(t *testing.T) {
	m, _ := testModel(t)
	rang := 0
	old := ringBell
	ringBell = func() tea.Msg { rang++; return nil }
	defer func() { ringBell = old }()
	busy := []live.Session{{ID: "a", Name: "storefront-01", Status: "busy"}, {ID: "b", Cwd: "/w/api", Status: "idle"}}
	if cmd := m.onWatch(busy); cmd != nil || m.status != "" {
		t.Fatal("the first read only learns the current state")
	}
	idle := []live.Session{{ID: "a", Name: "storefront-01", Status: "idle"}, {ID: "b", Cwd: "/w/api", Status: "idle"}}
	if cmd := m.onWatch(idle); cmd != nil || !strings.Contains(m.status, "storefront-01") || strings.Contains(m.status, "api") {
		t.Fatalf("busy → idle is announced, without the bell by default: %q", m.status)
	}
	m.status = ""
	if m.onWatch(idle); m.status != "" {
		t.Fatal("announced once")
	}
	m.prefs.Bell = true
	m.onWatch(busy)
	if cmd := m.onWatch(idle); cmd == nil {
		t.Fatal("the bell rings when the preference is on")
	} else {
		cmd()
	}
	if rang != 1 {
		t.Fatalf("bell rang %d times", rang)
	}
}

// With the Notification preference on, a finished session also shows a system notification.
func TestWatchDesktopNotification(t *testing.T) {
	m, _ := testModel(t)
	var got string
	old := desktopNotify
	desktopNotify = func(title, body string) tea.Cmd { got = body; return nil }
	defer func() { desktopNotify = old }()
	m.prefs.Notify = true
	m.onWatch([]live.Session{{ID: "a", Name: "api-02", Status: "busy"}})
	m.onWatch([]live.Session{{ID: "a", Name: "api-02", Status: "idle"}})
	if !strings.Contains(got, "api-02") {
		t.Fatalf("notification body: %q", got)
	}
}
