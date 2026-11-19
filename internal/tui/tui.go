// Package tui is the catude terminal UI (Bubble Tea). Slow work runs in commands so the UI never blocks;
// every error ends up in the status bar instead of crashing.
package tui

import (
	"context"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/accounts"
	"github.com/catude/catude/internal/audit"
	"github.com/catude/catude/internal/catalog"
	"github.com/catude/catude/internal/claudecfg"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/livemcp"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/prefs"
	"github.com/catude/catude/internal/profiles"
	"github.com/catude/catude/internal/sessions"
)

type tab int

const (
	tOverview tab = iota
	tConfig
	tLive
	tContext
	tUsage
	tModels
	tProfiles
	tSessions
	tGraph
	tSkills
	tAgents
	tCommands
	tMCP
	tPlugins
	tHooks
	tCatalog
	tDiscover
	tTools
	tStorage
	tAudit
)

var tabNames = []string{"Overview", "Config", "Live", "Context", "Usage", "Models", "Profiles", "Sessions", "Graph", "Skills", "Agents", "Commands", "MCP", "Plugins", "Hooks", "Available", "Discover", "Tools", "Storage", "Audit"}

var kindOf = map[tab]inventory.Kind{tSkills: inventory.Skill, tAgents: inventory.Agent, tCommands: inventory.Command,
	tHooks: inventory.Hook, tMCP: inventory.MCP, tPlugins: inventory.Plugin, tCatalog: inventory.Available}

// ---- messages ----
type loadedMsg struct {
	inv      inventory.Inventory
	findings []audit.Finding
	fp       string
	stale    []string
}
type projectsMsg struct {
	ps  []sessions.Project
	err error
}
type sessionListMsg struct{ ss []sessions.Session }
type sessionMsg struct{ s sessions.Session }
type editDoneMsg struct {
	path string
	err  error
}
type tickMsg time.Time
type spinMsg struct{}
type liveMsg struct {
	servers []livemcp.Server
	err     error
}

type model struct {
	env     paths.Env
	version string

	inv      inventory.Inventory
	findings []audit.Finding
	loaded   bool

	tab       tab
	cursor    map[tab]int
	filter    string
	filtering bool
	w, h      int
	status    string
	busy      bool

	cache    *sessions.Cache
	projects []sessions.Project
	level    int // 0 projects, 1 sessions, 2 live session
	projSel  int
	sessList []sessions.Session
	sessSel  int
	live     sessions.Session

	liveMCP     []livemcp.Server
	liveRunning bool
	liveAt      time.Time

	instances []paths.Instance
	cfgDirs   []paths.ConfigDir
	profSel   int

	workProfiles []profiles.Profile
	wpErr        string
	marked       map[string]bool // "kind:name" items chosen for a new work profile
	naming       bool
	nameInput    string

	switcher     *accounts.Switcher
	accts        []accounts.Account
	acctErr      string
	installArmed bool
	armed        string // catalog command waiting for a second press

	ctx        ctxState
	gal        galleryState
	tools      toolsState
	usage      usageState
	st         stState
	search     searchState
	projCWD    map[string]string
	latest     *sessions.Session
	trashArmed string
	delArmed   string
	active     string // active work profile
	wiz        *wizard
	help       bool
	debug      bool
	verifying  string // profile being checked with claude /context
	spin       int
	spinning   bool
	fp         string   // fingerprint of the last inventory (audit is reused when unchanged)
	stale      []string // leftover ~/.claude.json copies
	rc         *rowCache
	gen        int // bumps whenever rows may change

	opts  Options
	prefs prefs.Prefs
	lite  bool // lite view: only the tabs chosen for it
	pv    prefsState
	cfg   cfgState
	gr    graphState
	mo    modelsState
	lv    liveState
	wt    watchState // notices sessions that finish their turn, in any tab
	ag    agentsState
	off   claudecfg.Off // servers and skills switched off for this project
	ren   renameState
	desk  map[string]sessions.DesktopSession // conversations of Claude Desktop's Code tab, by session id
}

// Run starts the TUI in the alternate screen.
func Run(env paths.Env, version string, opts Options) error {
	m := newModel(env, version)
	m.opts, m.prefs = opts, opts.Prefs
	m.lite = opts.Lite || opts.Prefs.Lite
	m.applyPrefs()
	_, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

func newModel(env paths.Env, version string) model {
	m := model{env: env, version: version, cursor: map[tab]int{}, cache: &sessions.Cache{}, status: "loading…",
		marked: map[string]bool{}, rc: &rowCache{}, projCWD: map[string]string{}, active: profiles.Active()}
	m.loadCtxCache()
	m.loadLiveCache()
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.load(), m.loadProjects(), m.loadLive(), loadAccounts(), loadWorkProfiles(), tick())
}

type accountsMsg struct {
	sw    *accounts.Switcher
	accts []accounts.Account
	err   error
}
type switchedMsg struct{ err error }

// loadAccounts asks claude-swap (if installed) for the managed accounts and their usage.
func loadAccounts() tea.Cmd {
	return func() tea.Msg {
		sw, ok := accounts.Detect(context.Background())
		if !ok {
			return accountsMsg{}
		}
		a, err := sw.List(context.Background())
		return accountsMsg{&sw, a, err}
	}
}

// loadLive runs `claude mcp list` (slow: health checks) to see claude.ai connectors and live status.
func (m model) loadLive() tea.Cmd {
	dir, _ := m.env.RunDir()
	return func() tea.Msg {
		s, err := livemcp.List(context.Background(), "", dir) // dir is env.RunDir(): never an untrusted project
		return liveMsg{s, err}
	}
}

func tick() tea.Cmd { return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }) }

func (m model) load() tea.Cmd {
	env, prevFP, prevFindings := m.env, m.fp, m.findings
	return func() (msg tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				msg = loadedMsg{inv: inventory.Inventory{Env: env, Errors: []string{fmt.Sprint("internal error while loading: ", r)}}}
			}
		}()
		inv := inventory.Load(env)
		fp := fingerprint(inv)
		findings := prevFindings
		if fp != prevFP || prevFindings == nil {
			findings = audit.Run(inv) // the audit is the slow part: only when something changed
		}
		return loadedMsg{inv, findings, fp, env.StaleCopies()}
	}
}

func (m model) loadProjects() tea.Cmd {
	dir := m.env.ProjectsDir()
	return func() tea.Msg {
		ps, err := sessions.ListProjects(dir)
		return projectsMsg{ps, err}
	}
}

func (m model) loadSessions(p sessions.Project) tea.Cmd {
	c := m.cache
	return func() tea.Msg {
		var out []sessions.Session
		for i, f := range p.Sessions {
			if i >= 60 {
				break
			}
			if s, err := c.Get(f); err == nil {
				out = append(out, s)
			}
		}
		return sessionListMsg{out}
	}
}

func (m model) refreshLive() tea.Cmd {
	c, path := m.cache, m.live.Path
	return func() tea.Msg {
		s, err := c.Get(path)
		if err != nil {
			return nil
		}
		return sessionMsg{s}
	}
}

// ---- update ----

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.MouseMsg:
		return m.mouse(msg)
	case loadedMsg:
		if len(m.liveMCP) > 0 {
			inventory.MergeLive(&msg.inv, m.liveMCP)
		}
		cleanInventory(&msg.inv, msg.findings)
		changed := msg.fp != m.fp
		m.inv, m.findings, m.loaded, m.fp, m.stale = msg.inv, msg.findings, true, msg.fp, msg.stale
		m.off = claudecfg.ProjectOff(m.env)
		m.desk = sessions.DesktopSessions(m.env.DesktopDataDirs())
		m.instances, m.cfgDirs = m.env.CodeInstances(), m.env.ConfigDirs()
		if changed {
			m.gen++
		}
		if m.status == "loading…" || strings.HasPrefix(m.status, "reloaded") {
			m.status = fmt.Sprintf("reloaded %s · %d items · %d findings", time.Now().Format("15:04:05"), len(m.inv.Items), len(m.findings))
		}
	case workProfilesMsg:
		m.workProfiles, m.wpErr, m.active = msg.ps, "", profiles.Active()
		if msg.err != nil {
			m.wpErr = msg.err.Error()
		}
	case profileSavedMsg:
		m.gal.busy = ""
		if msg.err != nil {
			m.status = "profile: " + msg.err.Error()
			return m, nil
		}
		m.marked = map[string]bool{}
		m.gen++
		m.status = gOK + " profile " + msg.p.Name + " saved (" + msg.p.Summary() + ") · enter starts Claude with it · s makes it the active profile"
		if len(msg.notes) > 0 {
			m.status += " · " + strings.Join(msg.notes, " · ")
		}
		m.tab = tProfiles
		return m, loadWorkProfiles()
	case profileDoneMsg:
		m.status = "back from Claude Code (profile " + msg.name + ")"
		if msg.err != nil {
			m.status += ": " + msg.err.Error()
		}
		return m, tea.Batch(m.load(), m.loadProjects())
	case accountsMsg:
		m.switcher, m.accts, m.acctErr = msg.sw, msg.accts, ""
		if msg.err != nil {
			m.acctErr = msg.err.Error()
		}
	case installedMsg:
		if msg.err != nil {
			m.status = msg.what + ": " + msg.err.Error()
		} else {
			m.status = msg.what + " installed · restart Claude Code to load it"
		}
		return m, m.load()
	case switchedMsg:
		if msg.err != nil {
			m.status = "account switch: " + msg.err.Error()
		} else {
			m.status = "account switched · restart open Claude sessions to use it"
		}
		return m, loadAccounts()
	case liveMsg:
		m.liveRunning = false
		if msg.err != nil {
			m.status = "live MCP: " + msg.err.Error()
		} else {
			m.liveMCP, m.liveAt = msg.servers, time.Now()
			inventory.MergeLive(&m.inv, m.liveMCP)
			m.saveLiveCache()
			m.gen++
			m.status = fmt.Sprintf("MCP health checked · %d servers seen by Claude Code", len(m.liveMCP))
		}
	case projectsMsg:
		if msg.err != nil {
			m.status = "sessions: " + msg.err.Error()
		}
		m.projects = msg.ps
		return m, m.loadProjInfo(msg.ps)
	case projInfoMsg:
		for k, v := range msg.cwd {
			m.projCWD[k] = clean(v)
		}
		if msg.latest != nil {
			m.latest = msg.latest
		}
	case sessionListMsg:
		m.sessList, m.busy = msg.ss, false
	case sessionMsg:
		m.live = msg.s
	case ctxMsg:
		m.onCtxMsg(msg)
	case verifiedMsg:
		m.verifying = ""
		if msg.err != nil {
			m.status = "check " + msg.name + ": " + msg.err.Error()
			return m, nil
		}
		var bad []string
		manual := 0
		for _, it := range msg.c.Items {
			switch {
			case it.Manual:
				manual++
			case !it.Loaded:
				bad = append(bad, it.Kind+" "+it.Name)
			}
		}
		if len(bad) == 0 {
			m.status = fmt.Sprintf("%s profile %s: %d items loaded at start, %d manual (typed as /name, no context cost) · model %s · checked in %s", gOK, msg.name, len(msg.c.Items)-manual, manual, firstNonEmpty(msg.c.Model, "?"), msg.c.Elapsed.Round(time.Second))
		} else {
			m.status = fmt.Sprintf("%s profile %s: not loaded: %s (see Audit and the Skills/Agents tabs)", gFail, msg.name, strings.Join(bad, ", "))
		}
	case graphMsg:
		g := msg.g
		m.gr.g, m.gr.running, m.gr.at, m.gr.rows = &g, msg.partial, time.Now(), nil
		m.graphRows() // computed here once, so drawing never has to
		if msg.partial {
			return m, m.graphProcs()
		}
	case toolsMsg:
		m.tools.list, m.tools.running, m.tools.at = msg.ts, false, time.Now()
	case liveTickMsg:
		return m, m.onLiveTick()
	case liveListMsg:
		m.onLiveList(msg.ss)
	case agentsMsg:
		m.onAgents(msg.m)
	case watchMsg:
		return m, m.onWatch(msg.ss)
	case modelsMsg:
		m.onModels(msg)
	case usageMsg:
		m.usage.running = false
		if msg.err != nil {
			m.status = "usage: " + msg.err.Error()
		} else {
			r := msg.rep
			m.usage.rep, m.usage.at = &r, time.Now()
		}
	case stCatalogMsg:
		return m, m.onStCatalog(msg.es)
	case stMeasuredMsg:
		m.onStMeasured(msg.e)
	case stRetentionMsg:
		r := msg.r
		m.st.ret = &r
	case stCleanedMsg:
		return m, m.onStCleaned(msg)
	case searchMsg:
		if msg.query == m.search.query {
			m.search.running, m.search.hits = false, msg.hits
			if msg.err != nil {
				m.search.err = msg.err.Error()
			}
			m.status = fmt.Sprintf("%d conversations contain “%s” · enter open · R resume · x export", len(msg.hits), msg.query)
		}
	case exportedMsg:
		if msg.err != nil {
			m.status = "export: " + msg.err.Error()
		} else {
			m.status = gOK + " exported to " + msg.path + " (review it before sharing: it contains the whole conversation)"
		}
	case trashedMsg:
		if msg.err != nil {
			m.status = "remove session: " + msg.err.Error()
		} else {
			m.status = gOK + " moved to " + msg.dir + " · see RESTORE.txt there to undo"
			m.level = max(0, min(m.level, 1))
			if len(m.projects) > m.projSel {
				return m, tea.Batch(m.loadProjects(), m.loadSessions(m.projects[m.projSel]))
			}
			return m, m.loadProjects()
		}
	case resumedMsg:
		m.status = "back from Claude Code"
		if msg.err != nil {
			m.status += ": " + msg.err.Error()
		}
		return m, m.loadProjects()
	case editDoneMsg:
		if m.tab == tConfig {
			m.loadCfgEntries(true)
		}
		if msg.err != nil {
			m.status = "editor: " + msg.err.Error()
		} else {
			m.status = "saved " + filepath.Base(msg.path) + " · validating…"
		}
		return m, m.load()
	case spinMsg:
		busy := m.busy2() || strings.Contains(m.status, "building profile")
		if !busy {
			m.spinning = false
			return m, nil
		}
		m.spin++
		return m, spin()
	case tickMsg:
		cmds := []tea.Cmd{tick(), m.watchSessions()}
		if m.tab == tSessions && m.level == 2 && m.live.Path != "" {
			cmds = append(cmds, m.refreshLive())
		}
		if time.Time(msg).Second()%10 < 2 && m.loaded { // light auto-reload every ~10 s (audit reused if nothing changed)
			cmds = append(cmds, m.load())
		}
		if !m.spinning && m.busy2() {
			m.spinning = true
			cmds = append(cmds, spin())
		}
		return m, tea.Batch(cmds...)
	case tea.KeyMsg:
		nm, cmd := m.key(msg)
		mm := nm.(model)
		if !mm.spinning && mm.busy2() {
			mm.spinning = true
			cmd = tea.Batch(cmd, spin())
		}
		return mm, cmd
	}
	return m, nil
}

// busy2 reports whether any background job is running (drives the spinner).
func (m model) busy2() bool {
	return m.ag.running || m.ctx.running || m.st.measuring > 0 || m.search.running || m.liveRunning || m.tools.running || m.usage.running || m.verifying != "" || m.gr.running || m.mo.running
}

func spin() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return spinMsg{} })
}

func (m model) key(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := k.String()
	if m.filtering {
		switch s {
		case "esc":
			m.filtering, m.filter = false, ""
		case "enter":
			m.filtering = false
		case "backspace":
			if r := []rune(m.filter); len(r) > 0 {
				m.filter = string(r[:len(r)-1])
			}
		default:
			if len(k.Runes) > 0 {
				m.filter += string(k.Runes)
			}
		}
		m.cursor[m.tab] = 0
		return m, nil
	}
	if m.naming {
		switch s {
		case "esc":
			m.naming, m.nameInput, m.status = false, "", "profile not saved"
		case "enter":
			name := strings.TrimSpace(m.nameInput)
			m.naming, m.nameInput = false, ""
			if name == "" {
				m.status = "profile not saved: empty name"
				return m, nil
			}
			m.status = "building profile " + name + " (copying, hashing, auditing)…"
			return m, m.saveProfile(name)
		case "backspace":
			if r := []rune(m.nameInput); len(r) > 0 {
				m.nameInput = string(r[:len(r)-1])
			}
		default:
			for _, r := range k.Runes {
				if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
					m.nameInput += string(r)
				}
			}
		}
		return m, nil
	}
	if m.help {
		m.help = false
		return m, nil
	}
	if m.debug && (s == "esc" || s == "f12" || s == "ctrl+o" || s == "q") {
		m.debug = false
		return m, nil
	}
	if m.wiz != nil {
		return m.wizardKey(k)
	}
	if m.gal.open {
		return m.galleryKey(k)
	}
	if m.pv.open {
		return m.prefsKey(k)
	}
	if m.tab == tGraph && !m.filtering {
		if nm, cmd, handled := m.graphKey(k); handled {
			return nm, cmd
		}
	}
	if m.tab == tLive && !m.filtering {
		if nm, cmd, handled := m.liveKey(k); handled {
			return nm, cmd
		}
	}
	if m.tab == tModels && !m.filtering {
		if nm, cmd, handled := m.modelsKey(k); handled {
			return nm, cmd
		}
	}
	if m.tab == tConfig && !m.filtering {
		if nm, cmd, handled := m.cfgKey(k); handled {
			return nm, cmd
		}
	}
	if m.ren.active {
		return m.renameKey(k)
	}
	if m.search.typing {
		switch s {
		case "esc":
			m.search.typing = false
		case "enter":
			m.search.typing = false
			if strings.TrimSpace(m.search.query) != "" {
				return m, m.runSearch()
			}
		case "backspace":
			if r := []rune(m.search.query); len(r) > 0 {
				m.search.query = string(r[:len(r)-1])
			}
		default:
			if k.Type == tea.KeyRunes || s == " " {
				m.search.query += string(k.Runes)
			}
		}
		return m, nil
	}
	// Caps Lock or Shift must not break shortcuts: an upper-case letter acts like the lower-case one
	// unless it has its own meaning here (G end, P profile from marks, R/D in a session list).
	if k.Type == tea.KeyRunes && len(k.Runes) == 1 && k.Runes[0] >= 'A' && k.Runes[0] <= 'Z' && !m.upperBound(s) {
		s = strings.ToLower(s)
	}
	nWP := len(m.workProfiles)
	if len(s) == 1 && s[0] >= '0' && s[0] <= '9' && k.Type == tea.KeyRunes {
		n := int(s[0] - '1')
		if s == "0" {
			n = 9
		}
		if ts := m.activeTabs(); n < len(ts) {
			return m.gotoTab(ts[n])
		}
	}
	switch s {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "?":
		m.help = true
	case ",", "f2":
		m.pv = prefsState{open: true}
	case "f12", "ctrl+o":
		m.debug = !m.debug
	case "v":
		if m.tab == tProfiles && m.profSel < nWP && m.verifying == "" {
			p := m.workProfiles[m.profSel]
			m.verifying = p.Name
			m.status = "asking Claude Code whether every item of " + p.Name + " loads (claude /context)…"
			return m, m.verifyProfile(p)
		}
	case " ":
		m.toggleMark()
	case "P":
		if len(m.marked) == 0 {
			m.status = "mark skills, agents, commands, MCP servers or plugins with space first (or press n in Profiles)"
			return m, nil
		}
		m.wiz = newWizard(m.inv, m.marked)
	case "n":
		if m.tab == tProfiles {
			m.wiz = newWizard(m.inv, m.marked)
		}
		if m.tab == tSessions && m.level == 1 {
			return m.startRename()
		}
	case "a":
		if m.tab == tProfiles {
			m.wiz = newWizard(m.inv, currentSetup(m.inv))
			m.wiz.name = "my-setup-" + time.Now().Format("20060102")
			m.wiz.saveAll = true
		}
	case "[", "]":
		if m.tab == tUsage {
			steps := []int{7, 14, 30}
			i := 1
			for j, d := range steps {
				if d == m.usage.days {
					i = j
				}
			}
			if s == "[" {
				i = max(0, i-1)
			} else {
				i = min(len(steps)-1, i+1)
			}
			m.usage.days, m.usage.at = steps[i], time.Time{}
			return m, m.ensureUsage()
		}
	case "x":
		switch m.tab {
		case tMCP, tSkills:
			return m.toggleHere()
		case tContext:
			return m.ctxToggle()
		case tProfiles:
			if m.profSel < nWP {
				p := m.workProfiles[m.profSel]
				return m.startProfile(p, !p.Options.Exclusive)
			}
		case tSessions:
			if cmd := m.exportSession(); cmd != nil {
				m.status = m.spinner() + " exporting…"
				return m, cmd
			}
		}
	case "s":
		if m.tab == tProfiles && m.profSel < nWP {
			name := m.workProfiles[m.profSel].Name
			if m.active == name {
				name = ""
			}
			if err := profiles.SetActive(name); err != nil {
				m.status = err.Error()
				return m, nil
			}
			m.active = name
			if name == "" {
				m.status = "no active profile: catude claude starts your normal setup"
			} else {
				m.status = gOK + " active profile: " + name + " · `catude claude` starts it (run `catude shell-init` once to make plain `claude` use it)"
			}
		}
	case "d":
		if m.tab == tProfiles && m.profSel < nWP {
			name := m.workProfiles[m.profSel].Name
			if m.delArmed != name {
				m.delArmed = name
				m.status = "press d again to delete work profile " + name + " (your skills in ~/.claude are not touched)"
				return m, nil
			}
			m.delArmed = ""
			if err := profiles.Remove(name); err != nil {
				m.status = err.Error()
				return m, nil
			}
			if m.active == name {
				profiles.SetActive("")
			}
			m.status = "deleted work profile " + name
			return m, loadWorkProfiles()
		}
	case "D":
		if m.tab == tSessions && m.level >= 1 {
			return m, m.trashSession()
		}
	case "R":
		if m.tab == tSessions && m.level >= 1 {
			return m.resumeSession()
		}
	case "f":
		if m.tab == tSessions {
			m.search.typing, m.search.query = true, ""
		}
	case "o":
		if m.tab == tStorage {
			if e := m.stSelected(); e != nil {
				p := e.Path
				if e.IsFile {
					p = filepath.Dir(p)
				}
				if err := openFolder(p); err != nil {
					m.status = "open: " + err.Error()
				}
			}
		}
	case "p":
		if m.tab == tContext {
			return m, m.cycleCtxProfile()
		}
		if m.tab == tProfiles {
			m.openGallery()
			return m, nil
		}
	case "tab", "right", "l":
		return m.gotoTab(m.stepTab(1))
	case "shift+tab", "left", "h":
		return m.gotoTab(m.stepTab(-1))
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "pgup":
		m.move(-10)
	case "pgdown":
		m.move(10)
	case "home", "g":
		m.move(-1 << 20)
	case "end", "G":
		m.move(1 << 20)
	case "/":
		m.filtering, m.filter = true, ""
	case "esc", "backspace":
		if m.tab == tSessions && m.level > 0 {
			if m.level == 3 {
				m.level = 0
			} else {
				m.level--
			}
		} else {
			m.filter = ""
		}
	case "r":
		switch m.tab {
		case tContext:
			return m, m.refreshCtx()
		case tStorage:
			m.st.loaded = false
			return m, m.ensureStorage()
		case tTools:
			m.tools.at = time.Time{}
			return m, m.ensureTools()
		case tAgents:
			m.ag.at = time.Time{}
			return m, m.ensureAgents()
		case tUsage:
			m.usage.at = time.Time{}
			return m, m.ensureUsage()
		}
		m.status = "reloaded…"
		return m, tea.Batch(m.load(), m.loadProjects(), loadWorkProfiles())
	case "c":
		if m.tab == tStorage {
			return m, m.stClean()
		}
		if m.tab == tSessions && m.level >= 1 { // continue this session in Claude Code (same as R)
			return m.resumeSession()
		}
		nm, cmd := m.gotoTab(tContext)
		mm := nm.(model)
		return mm, tea.Batch(cmd, mm.refreshCtx())
	case "e":
		if m.tab == tProfiles && m.profSel < nWP {
			m.wiz = editWizard(m.inv, m.workProfiles[m.profSel])
			return m, nil
		}
		return m.edit()
	case "i":
		if m.tab == tDiscover {
			return m.installSelected()
		}
		if m.tab != tProfiles {
			return m, nil
		}
		if !m.installArmed {
			m.installArmed = true
			m.status = "press i again to run: uv tool install --force claude-swap==" + accounts.PinnedVersion
			return m, nil
		}
		m.installArmed = false
		cmd, err := accounts.InstallCmd()
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return switchedMsg{err} })
	case "m":
		if !m.liveRunning {
			m.liveRunning = true
			m.status = "checking MCP health with claude mcp list…"
			return m, m.loadLive()
		}
	case "enter":
		if m.tab == tProfiles && m.profSel < nWP {
			p := m.workProfiles[m.profSel]
			return m.startProfile(p, p.Options.Exclusive)
		}
		if m.tab == tProfiles && m.profSel-nWP >= len(m.cfgDirs) && m.switcher != nil {
			i := m.profSel - nWP - len(m.cfgDirs)
			if i < len(m.accts) {
				a := m.accts[i]
				if a.Active {
					m.status = "already the active account"
					return m, nil
				}
				m.status = "switching to account " + fmt.Sprint(a.Number) + " with claude-swap…"
				return m, tea.ExecProcess(m.switcher.SwitchCmd(a.Number), func(err error) tea.Msg { return switchedMsg{err} })
			}
		}
		if m.tab == tProfiles && m.profSel-nWP < len(m.cfgDirs) {
			d := m.cfgDirs[m.profSel-nWP]
			if d.Backup {
				m.status = "not opening " + d.Label + ": looks like a backup/credential store"
				return m, nil
			}
			m.env.ClaudeDir = d.Path
			m.liveMCP, m.status = nil, "inspecting configuration folder "+d.Label+" …"
			return m, tea.Batch(m.load(), m.loadProjects())
		}
		if m.tab == tStorage {
			if e := m.stSelected(); e != nil && e.Key == "code:projects" {
				return m.gotoTab(tSessions)
			}
		}
		if m.tab == tSessions {
			switch {
			case m.level == 0 && len(m.projects) > 0:
				m.level, m.sessSel, m.busy, m.sessList = 1, 0, true, nil
				return m, m.loadSessions(m.projects[m.projSel])
			case m.level == 1 && len(m.sessList) > 0:
				m.level, m.live = 2, m.sessList[m.sessSel]
				return m, m.refreshLive()
			case m.level == 3 && m.search.sel < len(m.search.hits):
				c, path := m.cache, m.search.hits[m.search.sel].Path
				m.level = 2
				return m, func() tea.Msg {
					s, err := c.Get(path)
					if err != nil {
						return nil
					}
					return sessionMsg{s}
				}
			}
		}
	}
	return m, nil
}

// gotoTab switches tab and starts loading whatever that tab needs (never blocking).
func (m model) gotoTab(t tab) (tea.Model, tea.Cmd) {
	m.tab = t
	m.filter, m.armed, m.installArmed, m.delArmed, m.trashArmed, m.st.armed = "", "", false, "", "", ""
	var cmd tea.Cmd
	switch t {
	case tContext:
		cmd = m.ensureCtx()
	case tConfig:
		m.ensureConfig()
	case tGraph:
		cmd = m.ensureGraph()
	case tStorage:
		cmd = m.ensureStorage()
	case tTools:
		cmd = m.ensureTools()
	case tAgents:
		cmd = m.ensureAgents()
	case tUsage:
		cmd = m.ensureUsage()
	case tModels:
		cmd = tea.Batch(m.ensureModels(false), m.ensureUsage())
	case tLive:
		cmd = m.ensureLive()
	}
	if !m.spinning && m.busy2() {
		m.spinning = true
		cmd = tea.Batch(cmd, spin())
	}
	return m, cmd
}

func (m *model) move(d int) {
	switch m.tab {
	case tContext:
		if m.ctx.comp != nil {
			m.ctx.sel = clamp(m.ctx.sel+d, 0, len(ctxBig(m.ctx.comp))-1)
		}
		m.ctx.scroll = max(0, m.ctx.scroll+d)
		return
	case tStorage:
		m.st.sel = clamp(m.st.sel+d, 0, len(m.st.entries)-1)
		m.st.armed = ""
		return
	case tTools:
		m.tools.sel = clamp(m.tools.sel+d, 0, len(m.tools.list)-1)
		return
	case tAgents:
		m.ag.sel = clamp(m.ag.sel+d, 0, len(m.ag.m.Agents)-1)
		return
	case tUsage:
		return
	}
	if m.tab == tSessions && m.level == 3 {
		m.search.sel = clamp(m.search.sel+d, 0, len(m.search.hits)-1)
		return
	}
	if m.tab == tProfiles {
		m.profSel = clamp(m.profSel+d, 0, len(m.workProfiles)+len(m.cfgDirs)+len(m.accts)-1)
		return
	}
	if m.tab == tSessions {
		switch m.level {
		case 0:
			m.projSel = clamp(m.projSel+d, 0, len(m.projects)-1)
		case 1:
			m.sessSel = clamp(m.sessSel+d, 0, len(m.sessList)-1)
		}
		return
	}
	n := len(m.rows())
	m.cursor[m.tab] = clamp(m.cursor[m.tab]+d, 0, n-1)
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// edit opens the selected user-owned file in $VISUAL/$EDITOR after backing it up.
func (m model) edit() (tea.Model, tea.Cmd) {
	it := m.selectedItem()
	if it == nil {
		return m, nil
	}
	if !it.Editable || it.Path == "" {
		m.status = "read-only: " + string(it.Scope) + " items are managed elsewhere (plugins, claude.ai sync, policy)"
		return m, nil
	}
	backup, err := backupFile(it.Path)
	if err != nil {
		m.status = "backup failed, not editing: " + err.Error()
		return m, nil
	}
	cmd, err := editorCmd(it.Path)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.status = "editing " + it.Path + " (backup: " + backup + ")"
	path := it.Path
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return editDoneMsg{path, err} })
}

func editorCmd(path string) (*exec.Cmd, error) {
	ed := strings.TrimSpace(os.Getenv("VISUAL"))
	if ed == "" {
		ed = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if ed == "" {
		if runtime.GOOS == "windows" {
			ed = "notepad"
		} else {
			ed = "vi"
		}
	}
	parts := strings.Fields(ed)
	bin, err := exec.LookPath(parts[0])
	if err != nil {
		return nil, fmt.Errorf("editor %q not found (set $EDITOR)", parts[0])
	}
	return exec.Command(bin, append(parts[1:], path)...), nil
}

func backupFile(p string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "catude", "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	dst := filepath.Join(dir, time.Now().Format("20060102-150405")+"-"+strings.ReplaceAll(filepath.Base(filepath.Dir(p)), string(filepath.Separator), "_")+"-"+filepath.Base(p))
	in, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", err
	}
	return dst, out.Close()
}

// ---- rows ----

type row struct {
	text  string
	item  *inventory.Item
	find  *audit.Finding
	entry *catalog.Entry
	style lipgloss.Style
}

func (m model) rows() []row {
	if m.rc == nil {
		return m.buildRows()
	}
	key := fmt.Sprintf("%d|%s|%d|%d", m.tab, m.filter, m.gen, len(m.marked))
	if m.rc.key != key {
		m.rc.key, m.rc.rows = key, m.buildRows()
	}
	return m.rc.rows
}

func (m model) buildRows() []row {
	var out []row
	f := strings.ToLower(m.filter)
	match := func(s ...string) bool {
		if f == "" {
			return true
		}
		return strings.Contains(strings.ToLower(strings.Join(s, " ")), f)
	}
	if k, ok := kindOf[m.tab]; ok {
		items := m.inv.Of(k)
		for i := range items {
			it := items[i]
			if !match(it.Name, it.Description, string(it.Scope), it.Origin) {
				continue
			}
			mark, st := "  ", lipgloss.NewStyle()
			switch {
			case len(it.Problems) > 0:
				mark = markWarn()
			case k == inventory.Available && it.Enabled:
				mark = markOK()
			case !it.Enabled:
				mark, st = markOff(), sMuted
			}
			switch it.Status {
			case string(livemcp.Connected):
				mark = markOK()
			case string(livemcp.NeedsAuth):
				mark = markWarn()
			case string(livemcp.Failed):
				mark = markFail()
			}
			label := it.Name
			if k == inventory.Available {
				label += sMuted.Render(" @" + it.Origin)
			} else {
				label += sMuted.Render(" · " + string(it.Scope))
			}
			if n := m.contextCost(it); n > 0 {
				label += sMuted.Render(" · " + F("%s tokens in context", human(n)))
			}
			if o := m.offLabel(it); o != "" {
				label += sWarn.Render(" · " + o)
				st = sMuted
			}
			if m.marked[markKey(k, it.Name)] {
				label = sTitle.Render(gMarked+" ") + label
			}
			out = append(out, row{text: mark + label, item: &it, style: st})
		}
	}
	if m.tab == tDiscover {
		return m.discoverRows(match)
	}
	if m.tab == tAudit {
		for i := range m.findings {
			fd := m.findings[i]
			if !match(fd.Rule, fd.Item, fd.Detail) {
				continue
			}
			st := sMuted
			switch fd.Severity {
			case audit.High:
				st = sHigh
			case audit.Warn:
				st = sWarn
			}
			out = append(out, row{text: st.Render(fmt.Sprintf("%-4s", fd.Severity)) + " " + fd.Rule + sMuted.Render(" · "+fd.Item), find: &fd})
		}
	}
	return out
}

func (m model) selectedItem() *inventory.Item {
	if m.tab == tAgents {
		return m.agentItem()
	}
	rs := m.rows()
	i := m.cursor[m.tab]
	if i < 0 || i >= len(rs) {
		return nil
	}
	return rs[i].item
}

// ---- view ----

func (m model) View() string {
	start := time.Now()
	s := finish(m.view())
	lastRender.Store(int64(time.Since(start)))
	return s
}

func (m model) tabLabel(t tab) string {
	n := T(tabNames[t])
	label := n
	if pos := m.tabPos(t); pos >= 0 && pos < 10 {
		label = fmt.Sprintf("%d %s", (pos+1)%10, n)
	}
	if k, ok := kindOf[t]; ok && m.loaded {
		label += fmt.Sprintf(" %d", m.inv.Count(k))
	}
	switch t {
	case tDiscover:
		label += fmt.Sprintf(" %d", len(discoverEntries))
	case tAudit:
		if m.loaded {
			label += fmt.Sprintf(" %d", len(m.findings))
		}
	case tContext:
		if m.ctx.running {
			label += " " + m.spinner()
		} else if m.ctx.comp != nil {
			label += " " + human(m.ctx.comp.Used)
		}
	case tStorage:
		if m.st.measuring > 0 {
			label += " " + m.spinner()
		}
	}
	return label
}

// visibleTabs returns the tabs that fit, always including the active one (the bar scrolls with it).
func (m model) visibleTabs() []int {
	var all []int
	cur := 0
	for i, t := range m.activeTabs() {
		all = append(all, int(t))
		if t == m.tab {
			cur = i
		}
	}
	width := func(ts []int) int {
		w := 0
		for _, i := range ts {
			w += lipgloss.Width(sTab.Render(m.tabLabel(tab(i))))
		}
		return w
	}
	if m.w <= 0 || width(all) <= m.w-2 {
		return all
	}
	start, end := cur, cur+1
	for {
		grew := false
		if end < len(all) && width(all[start:end+1]) <= m.w-4 {
			end++
			grew = true
		}
		if start > 0 && width(all[start-1:end]) <= m.w-4 {
			start--
			grew = true
		}
		if !grew {
			break
		}
	}
	return all[start:end]
}

func (m model) view() string {
	if m.w == 0 {
		return "loading…"
	}
	var b strings.Builder
	name := "catude"
	if m.lite {
		name = "ctlite"
	}
	head := sTitle.Render(gLogo+" "+name) + sMuted.Render(" "+m.version+" · "+shorten(m.env.ProjectDir, 40))
	if m.active != "" {
		head += sMuted.Render(" · profile ") + sTitle.Render(m.active)
	}
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		head += sWarn.Render(" · experimental build")
	}
	head += sMuted.Render(" · " + T("? help"))
	b.WriteString(lipgloss.NewStyle().MaxWidth(m.w).Render(head) + "\n")
	vis := m.visibleTabs()
	var tabs []string
	if len(vis) > 0 && vis[0] != int(m.activeTabs()[0]) {
		tabs = append(tabs, sMuted.Render("‹"))
	}
	for _, i := range vis {
		if tab(i) == m.tab {
			tabs = append(tabs, sTabOn.Render(m.tabLabel(tab(i))))
		} else {
			tabs = append(tabs, sTab.Render(m.tabLabel(tab(i))))
		}
	}
	if at := m.activeTabs(); len(vis) > 0 && vis[len(vis)-1] != int(at[len(at)-1]) {
		tabs = append(tabs, sMuted.Render("›"))
	} else if n := m.hiddenTabs(); n > 0 {
		tabs = append(tabs, sMuted.Render("  "+F("+%d tabs in ,", n)))
	}
	b.WriteString(lipgloss.NewStyle().MaxWidth(m.w).Render(strings.Join(tabs, "")) + "\n")
	bodyH := m.h - 5
	if bodyH < 5 {
		bodyH = 5
	}
	pane := func(s string) string { return sPane.Width(m.w - 4).Height(bodyH - 2).MaxHeight(bodyH).Render(s) }
	var body string
	switch {
	case m.debug:
		body = pane(m.debugView(bodyH - 2))
	case m.help:
		body = pane(m.helpView(bodyH - 2))
	case m.pv.open:
		body = pane(m.prefsView(bodyH - 2))
	case m.gal.open:
		body = pane(m.galleryView(bodyH - 2))
	case m.wiz != nil:
		body = pane(m.wizardView(bodyH - 2))
	case m.tab == tOverview:
		body = pane(m.overview(bodyH - 2))
	case m.tab == tContext:
		body = pane(m.contextView(bodyH - 2))
	case m.tab == tSessions:
		body = m.sessionsView(bodyH)
	case m.tab == tProfiles:
		body = pane(m.profilesView(bodyH - 2))
	case m.tab == tConfig:
		body = pane(m.configView(bodyH - 2))
	case m.tab == tGraph:
		body = pane(m.graphView(bodyH - 2))
	case m.tab == tStorage:
		body = pane(m.storageView(bodyH - 2))
	case m.tab == tTools:
		body = pane(m.toolsView(bodyH - 2))
	case m.tab == tAgents:
		body = pane(m.agentsView(bodyH - 2))
	case m.tab == tUsage:
		body = pane(m.usageView(bodyH - 2))
	case m.tab == tModels:
		body = pane(m.modelsView(bodyH - 2))
	case m.tab == tLive:
		body = pane(m.liveView(bodyH - 2))
	default:
		body = m.listView(bodyH)
	}
	b.WriteString(body + "\n")
	keys := T("←→ tabs · ↑↓ move · / filter · e edit · space mark · P profile · , preferences · ? help · q quit")
	switch m.tab {
	case tContext:
		keys = T("↑↓ choose · x off in this project · r measure again · p compare with a work profile · ? help")
	case tSessions:
		keys = T("enter open · n name · f search all · R/c resume in Claude · x export · D remove · esc back · ? help")
	case tProfiles:
		keys = T("n new · a save current setup · p presets · e edit · s switch · v verify · enter start · x other mode · d delete · ? help")
	case tStorage:
		keys = T("↑↓ move · o open folder · c c clean safe item · r measure · ? help")
	case tTools:
		keys = T("↑↓ move · r check again · ? help")
	case tAgents:
		keys = T("↑↓ move · e edit the definition · space mark for a profile · r read again · ? help")
	case tUsage:
		keys = T("r refresh · [ ] 7 / 14 / 30 days · ? help")
	case tLive:
		keys = T("↑↓ choose a session · enter open its conversation · o open its folder · ? help")
	case tModels:
		keys = T("↑↓ compare with yours · t kind of work · space compare with the selected · / filter · r download again · ? help")
	case tDiscover:
		keys = T("↑↓ move · / search, #tag (#testing, #frontend…) · i i install · ? help")
	case tMCP:
		keys = T("↑↓ move · / filter · x off in this project · m check health · space mark · ? help")
	case tSkills:
		keys = T("↑↓ move · / filter · x off / name only in this project · e edit · space mark · ? help")
	}
	if m.wiz != nil || m.help {
		keys = ""
	}
	if m.tab == tGraph && !m.wiz2() {
		keys = T("↑↓ move · enter/+ open · - close/parent · E/C all · / find · w diagram in browser · r rebuild · ←→ tabs · ? help")
	}
	if m.tab == tConfig && !m.wiz2() && m.cfg.ps.open {
		keys = presetsHint()
	} else if m.tab == tConfig && !m.wiz2() {
		keys = T("↑↓ choose app file · enter open its values · e edit file in your editor · r read again · , preferences · ? help")
		if m.cfg.focus == 1 {
			keys = T("↑↓ move · enter change · space toggle / next value · d remove (default) · a only set / all · / search · e editor · esc back")
			if t, _ := m.cfgSelected(); t.ID == claudecfg.EffectiveID {
				keys = T("↑↓ move · / search · esc back (read-only: change a value in the file it comes from)")
			}
		}
		if m.cfg.editing {
			_, e := m.cfgSelected()
			key := ""
			if e != nil {
				key = e.Key
			}
			keys = F("new value for %s:", key) + " " + m.cfg.input + "▏ (enter save · esc cancel)"
		}
	}
	if m.pv.open {
		keys = ""
	}
	if m.gal.open {
		keys = T("↑↓ choose · enter enter load as a work profile (download pinned files, verify, audit) · pgup pgdn scroll · esc close")
	}
	if m.debug {
		keys = T("F12 close")
	}
	if m.ren.active {
		keys = T("name: ") + m.ren.input + "▏ " + T("(enter save · esc cancel)")
	}
	if m.search.typing {
		keys = T("search every conversation: ") + m.search.query + "▏ (enter search · esc cancel)"
	}
	if m.filtering {
		keys = T("filter: ") + m.filter + "▏ (enter keep · esc clear)"
	} else if m.filter != "" {
		keys = T("filter: ") + m.filter + " · " + keys
	}
	status := m.status
	if m.liveRunning {
		status = m.spinner() + " checking MCP health… · " + status
	}
	if m.prefs.HideHints && !m.filtering && !m.search.typing && !m.cfg.editing && !m.gal.open {
		keys = T("? help · , preferences")
	}
	b.WriteString(sStatus.MaxWidth(m.w).Render(keys + "  │  " + status))
	return b.String()
}

func (m model) listView(h int) string {
	rs := m.rows()
	listW := m.w * 42 / 100
	if m.w < 90 {
		listW = m.w - 4
	}
	cur := m.cursor[m.tab]
	room := h - 2
	start := 0
	if cur >= room {
		start = cur - (room - 1)
	}
	inner := listW - 1 // the last column holds the scrollbar
	var lines []string
	for i := start; i < len(rs) && len(lines) < room; i++ {
		t := "  " + lipgloss.NewStyle().MaxWidth(inner-4).Render(rs[i].text)
		if i == cur {
			t = sSel.Width(inner - 2).Render(gCursor + " " + stripStyles(rs[i].text, inner-4))
		} else if rs[i].item != nil {
			t = "  " + rs[i].style.Render(lipgloss.NewStyle().MaxWidth(inner-4).Render(rs[i].text))
		}
		lines = append(lines, t)
	}
	if len(rs) == 0 {
		lines = append(lines, sMuted.Render("nothing here"+map[bool]string{true: " matching " + m.filter, false: ""}[m.filter != ""]))
	}
	left := sPane.Width(listW).Height(h - 2).Render(withScrollbar(lines, len(rs), start, room, listW-2))
	if m.w < 90 {
		return left
	}
	right := sPane.Width(m.w - listW - 6).Height(h - 2).Render(whatIsBlock(m.tab, m.w-listW-8) + m.detail(rs, cur, m.w-listW-8))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m model) detail(rs []row, cur, w int) string {
	if cur < 0 || cur >= len(rs) {
		return sMuted.Render("select an item")
	}
	wrap := lipgloss.NewStyle().Width(w)
	if f := rs[cur].find; f != nil {
		return wrap.Render(sHeading.Render(f.Rule) + "\n\n" + f.Detail + "\n\n" + sOK.Render("→ "+f.Fix) + "\n\n" + sMuted.Render(string(f.Kind)+" "+f.Item+"\n"+f.Path))
	}
	if e := rs[cur].entry; e != nil {
		return m.discoverDetail(e, w)
	}
	it := rs[cur].item
	if it == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(sHeading.Render(it.Name) + "\n")
	b.WriteString(sMuted.Render(string(it.Kind)+" · "+string(it.Scope)) + "\n\n")
	if it.Description != "" {
		b.WriteString(it.Description + "\n\n")
	}
	for _, p := range it.Problems {
		b.WriteString(sWarn.Render(gWarn+" "+p) + "\n")
	}
	for _, d := range it.Detail {
		b.WriteString(sMuted.Render("• ") + d + "\n")
	}
	if it.Origin != "" {
		b.WriteString(sMuted.Render("• origin: ") + it.Origin + "\n")
	}
	if it.Path != "" {
		b.WriteString(sMuted.Render("• "+it.Path) + "\n")
	}
	if it.Editable {
		b.WriteString("\n" + sOK.Render("e  edit (a backup is made first, then it is re-validated)"))
	}
	return wrap.Render(b.String())
}

func (m model) overview(h int) string {
	if !m.loaded {
		return "loading…"
	}
	var b strings.Builder
	b.WriteString(sHeading.Render(T("Where Claude reads configuration on this machine")) + "\n\n")
	for _, f := range m.inv.Surfaces {
		mark := sMuted.Render("·")
		if f.Exists {
			mark = sOK.Render("●")
		}
		b.WriteString(fmt.Sprintf("%s %-9s %-34s %s\n", mark, f.Scope, clipEnd(f.Label, 34), sMuted.Render(shorten(f.Path, m.w-56))))
	}
	high := 0
	for _, f := range m.findings {
		if f.Severity == audit.High {
			high++
		}
	}
	b.WriteString("\n" + sHeading.Render(T("Inventory")) + "\n")
	for _, k := range []inventory.Kind{inventory.Skill, inventory.Agent, inventory.Command, inventory.Hook, inventory.MCP, inventory.Plugin, inventory.Marketplace, inventory.Available} {
		b.WriteString(fmt.Sprintf("  %-12s %d\n", k, m.inv.Count(k)))
	}
	sev := sOK.Render("no high findings")
	if high > 0 {
		sev = sHigh.Render(fmt.Sprintf("%d high findings → Audit tab", high))
	}
	b.WriteString("\n" + sHeading.Render(T("Security")) + "  " + sev + "\n")
	if m.latest != nil {
		if s := *m.latest; true {
			b.WriteString("\n" + sHeading.Render(T("Latest session")) + "  " + clipEnd(s.Title, 40) + "\n  " + ctxBar(s, 40) + "\n")
		}
	}
	for i, e := range m.inv.Errors {
		if i == 5 {
			break
		}
		b.WriteString(sWarn.Render("error: "+clipEnd(e, m.w-12)) + "\n")
	}
	return b.String()
}

func (m model) sessionsView(h int) string {
	var lines []string
	title := ""
	switch m.level {
	case 0:
		title = "Projects (" + fmt.Sprint(len(m.projects)) + ")"
		room := h - 5
		start := clamp(m.projSel-(room-1), 0, max(0, len(m.projects)-room))
		var shown []string
		for i := start; i < len(m.projects) && len(shown) < room; i++ {
			p := m.projects[i]
			l := fmt.Sprintf("%s  %3d sessions  %8s  %s", p.Latest.Format("02 Jan 15:04"), len(p.Sessions), human(int(p.Bytes))+"B", shorten(m.projectCWD(p), m.w-50))
			if i == m.projSel {
				l = sSel.Render(gCursor + " " + stripStyles(l, m.w-10))
			} else {
				l = "  " + l
			}
			shown = append(shown, l)
		}
		lines = append(lines, withScrollbar(shown, len(m.projects), start, min(room, len(m.projects)), m.w-6), position(start, room, len(m.projects)))
	case 1:
		title = "Sessions · " + shorten(m.projectCWD(m.projects[m.projSel]), m.w-20)
		if m.busy {
			lines = append(lines, sMuted.Render("measuring…"))
		}
		room := h - 5 - len(lines)
		start := clamp(m.sessSel-(room-1), 0, max(0, len(m.sessList)-room))
		var shown []string
		for i := start; i < len(m.sessList) && len(shown) < room; i++ {
			s := m.sessList[i]
			title, origin := s.Title, ""
			if d, ok := m.desk[s.ID]; ok {
				origin = " " + T("Desktop")
				if d.Archived {
					origin += " · " + T("archived")
				}
				if !s.Named && d.Title != "" {
					title = d.Title
				}
			}
			l := fmt.Sprintf("%s  %-34s %s  %4d resp  %d compact", s.End.Format("02 Jan 15:04"), clipEnd(clean(firstNonEmpty(title, s.ID[:min(8, len(s.ID))])), 34), ctxBar(s, 16), s.Responses, s.Compactions)
			if origin != "" {
				l += sMuted.Render(" ·" + origin)
			}
			if c, ok := m.sessionCost(s); ok {
				l += sMuted.Render("  ~" + usd(c))
			}
			if i == m.sessSel {
				l = sSel.Render(gCursor + " " + stripStyles(l, m.w-10))
			} else {
				l = "  " + l
			}
			shown = append(shown, l)
		}
		lines = append(lines, withScrollbar(shown, len(m.sessList), start, min(room, len(m.sessList)), m.w-6), position(start, room, len(m.sessList)))
	case 3:
		title = "Search · " + m.search.query
		lines = m.searchView(h)
	case 2:
		s := m.live
		title = "Live session (refreshes every 2 s)"
		lines = append(lines,
			sHeading.Render(firstNonEmpty(s.Title, s.ID)),
			sMuted.Render(fmt.Sprintf("%s · window %s · %s · branch %s", s.Model, human(s.Window()), s.CWD, firstNonEmpty(s.Branch, "-"))),
			"",
			"CONTEXT NOW  "+ctxBar(s, 40),
			fmt.Sprintf("  last response: input %d · cache write %d · cache read %d · output %d", s.Last.Input, s.Last.CacheCreate, s.Last.CacheRead, s.Last.Output),
			fmt.Sprintf("  free: %s · compactions: %d", human(max(0, s.Window()-s.ContextTokens())), s.Compactions),
			"",
			fmt.Sprintf("WHOLE SESSION  %d unique responses (subagents excluded)", s.Responses),
			fmt.Sprintf("  processed %s = input %s + cache write %s + cache read %s", human(s.Total.Context()), human(s.Total.Input), human(s.Total.CacheCreate), human(s.Total.CacheRead)),
			fmt.Sprintf("  generated %s · cache hit %.0f%%", human(s.Total.Output), 100*s.CacheHitRatio()),
			m.costLine(s),
		)
		if sum := summaryLines(s, m.w-8); len(sum) > 0 {
			lines = append(lines, "")
			lines = append(lines, sum...)
		}
		lines = append(lines,
			"",
			sMuted.Render("Exact: API usage of the last response (= context), responses deduplicated by message.id+requestId."))
		if s.Malformed > 0 {
			lines = append(lines, sWarn.Render(fmt.Sprintf("%d lines could not be read", s.Malformed)))
		}
	}
	return sPane.Width(m.w - 4).Height(h - 2).Render(sHeading.Render(title) + "\n\n" + strings.Join(lines, "\n"))
}

// ---- small helpers ----

func ctxBar(s sessions.Session, w int) string {
	r := s.ContextRatio()
	return bar(r, w) + fmt.Sprintf(" %s %4.1f%%", human(s.ContextTokens()), 100*r)
}

func bar(r float64, w int) string {
	if r < 0 {
		r = 0
	}
	if r > 1 {
		r = 1
	}
	n := int(r*float64(w) + 0.5)
	st := sOK
	if r >= 0.8 {
		st = sHigh
	} else if r >= 0.5 {
		st = sWarn
	}
	return st.Render(strings.Repeat("█", n)) + sMuted.Render(strings.Repeat("·", w-n))
}

// meter is a bar where more is better (a score, a finished to-do list): always in the success colour,
// unlike bar, which warns as it fills (context, usage).
func meter(r float64, w int) string {
	r = max(0, min(1, r))
	n := int(r*float64(w) + 0.5)
	return sOK.Render(strings.Repeat("█", n)) + sMuted.Render(strings.Repeat("·", w-n))
}

func human(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.2fM", float64(n)/1e6)
	case n >= 1000:
		return fmt.Sprintf("%.1fk", float64(n)/1e3)
	}
	return fmt.Sprint(n)
}

// shorten keeps the end of paths, cut to n terminal cells.
func shorten(s string, n int) string {
	if n < 4 {
		n = 4
	}
	w := ansiWidth(s)
	if w <= n {
		return s
	}
	return ansi.TruncateLeft(s, w-n+1, "…")
}

func stripStyles(s string, w int) string {
	plain := ansiRe.ReplaceAllString(s, "")
	if w > 0 && ansiWidth(plain) > w {
		return ansi.Truncate(plain, w, "")
	}
	return plain
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// clipEnd keeps the beginning of names, titles and e-mails (paths use shorten, which keeps the end).
func clipEnd(s string, n int) string {
	if n < 4 {
		n = 4
	}
	return truncate(s, n)
}

// upperBound reports whether an upper-case key has its own action in the current context.
func (m model) upperBound(s string) bool {
	switch s {
	case "G":
		return true
	case "P":
		return len(m.marked) > 0
	case "R", "D":
		return m.tab == tSessions && m.level >= 1
	}
	return false
}
