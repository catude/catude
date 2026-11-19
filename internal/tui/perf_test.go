package tui

import (
	"os"
	"testing"
	"time"

	"github.com/catude/catude/internal/audit"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/sessions"
)

func TestPerfRealMachine(t *testing.T) {
	if os.Getenv("LR_PERF") == "" {
		t.Skip()
	}
	wd, _ := os.Getwd()
	env := paths.Detect(wd)
	t0 := time.Now()
	inv := inventory.Load(env)
	t.Logf("inventory.Load %v (%d items)", time.Since(t0), len(inv.Items))
	t0 = time.Now()
	f := audit.Run(inv)
	t.Logf("audit %v", time.Since(t0))
	t0 = time.Now()
	ps, _ := sessions.ListProjects(env.ProjectsDir())
	t.Logf("ListProjects %v (%d)", time.Since(t0), len(ps))
	t0 = time.Now()
	ins, cd := env.CodeInstances(), env.ConfigDirs()
	t.Logf("instances+configdirs %v", time.Since(t0))
	m := newModel(env, "perf")
	m.w, m.h, m.inv, m.findings, m.loaded, m.projects, m.instances, m.cfgDirs = 160, 45, inv, f, true, ps, ins, cd
	if msg, ok := m.loadProjInfo(ps)().(projInfoMsg); ok {
		m.projCWD, m.latest = msg.cwd, msg.latest
	}
	t.Logf("project info (background) %v", time.Since(t0))
	for i, n := range tabNames {
		m.tab = tab(i)
		t0 = time.Now()
		m.View() // first frame builds caches
		t0 = time.Now()
		for j := 0; j < 5; j++ {
			m.View()
		}
		t.Logf("View %-10s %v", n, time.Since(t0)/5)
	}
}
