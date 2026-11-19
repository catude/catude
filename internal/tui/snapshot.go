package tui

import (
	"time"

	"context"
	livesess "github.com/catude/catude/internal/live"
	"github.com/catude/catude/internal/models"
	"strings"

	"github.com/catude/catude/internal/audit"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/livemcp"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/profiles"
	"github.com/catude/catude/internal/sessions"
)

// Snapshot renders every tab to plain text without a terminal: used for tests, CI and README screenshots.
// live also asks Claude Code for the claude.ai connectors and MCP health (slow).
func Snapshot(env paths.Env, version string, width, height int, color, live bool) string {
	inv := inventory.Load(env)
	var servers []livemcp.Server
	if live {
		dir, _ := env.RunDir()
		if s, err := livemcp.List(context.Background(), "", dir); err == nil {
			servers = s
			inventory.MergeLive(&inv, s)
		}
	}
	m := model{env: env, version: version, cursor: map[tab]int{}, cache: &sessions.Cache{}, w: width, h: height,
		inv: inv, findings: audit.Run(inv), loaded: true, status: "snapshot", liveMCP: servers,
		instances: env.CodeInstances(), cfgDirs: env.ConfigDirs(), marked: map[string]bool{}, rc: &rowCache{}, projCWD: map[string]string{}, stale: env.StaleCopies()}
	m.workProfiles, _ = profiles.List()
	m.projects, _ = sessions.ListProjects(env.ProjectsDir())
	if live {
		if msg, ok := loadAccounts()().(accountsMsg); ok {
			m.switcher, m.accts = msg.sw, msg.accts
			if msg.err != nil {
				m.acctErr = msg.err.Error()
			}
		}
	}
	for i := range tabNames { // a snapshot shows every tab, also the ones hidden by default
		m.prefs.Tabs = append(m.prefs.Tabs, tabID(tab(i)))
	}
	m.ensureConfig()
	m.lv.reader = &livesess.Reader{ClaudeDir: env.ClaudeDir}
	m.lv.list, m.lv.at = m.lv.reader.Sessions(), time.Now()
	m.mo.data, m.mo.loaded = models.ReadCache(cacheDir()), true // cached results only: a snapshot never downloads
	var b strings.Builder
	for i := range tabNames {
		m.tab = tab(i)
		v := m.View()
		if !color {
			v = ansiRe.ReplaceAllString(v, "")
		}
		b.WriteString(v)
		b.WriteString("\n" + strings.Repeat("═", width) + "\n")
	}
	return b.String()
}
