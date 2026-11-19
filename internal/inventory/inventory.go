// Package inventory loads everything that shapes Claude's behaviour on this machine:
// skills, subagents, slash commands, hooks, MCP servers (Claude Code and Claude Desktop),
// installed plugins and the plugin catalogs of every known marketplace.
// Loading never fails as a whole: a broken file becomes a Problem on its item or an Error entry.
package inventory

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/catude/catude/internal/frontmatter"
	"github.com/catude/catude/internal/livemcp"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/redact"
)

// Kind of item.
type Kind string

const (
	Skill       Kind = "skill"
	Agent       Kind = "agent"
	Command     Kind = "command"
	Hook        Kind = "hook"
	MCP         Kind = "mcp"
	Plugin      Kind = "plugin"
	Marketplace Kind = "marketplace"
	Available   Kind = "available" // plugin offered by a marketplace
	Memory      Kind = "memory"
	Settings    Kind = "settings"
)

// Item is one thing catude can show, edit or audit.
type Item struct {
	Kind        Kind
	Name        string
	Scope       paths.Scope
	Origin      string // e.g. plugin id, Desktop install, settings file label
	Path        string // file to open/edit ("" when not a file)
	Description string
	Invocable   bool // skill: Claude can load it by itself (not disable-model-invocation)
	Enabled     bool
	Problems    []string
	Detail      []string // already redacted lines for the detail pane
	Content     string   // raw text for auditing (never shown unredacted)
	Editable    bool     // a user-owned file catude may open in $EDITOR
	Status      string   // live MCP status from `claude mcp list` ("" = not checked)
}

// Inventory is the full picture.
type Inventory struct {
	Env      paths.Env
	Items    []Item
	Errors   []string
	Surfaces []paths.File
	LoadedAt time.Time
	LiveMCP  bool // claude mcp list merged
}

// Count returns the number of items of a kind.
func (inv Inventory) Count(k Kind) int {
	n := 0
	for _, it := range inv.Items {
		if it.Kind == k {
			n++
		}
	}
	return n
}

// Of returns the items of a kind, sorted by scope then name.
func (inv Inventory) Of(k Kind) []Item {
	var out []Item
	for _, it := range inv.Items {
		if it.Kind == k {
			out = append(out, it)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Scope != out[j].Scope {
			return scopeRank(out[i].Scope) < scopeRank(out[j].Scope)
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func scopeRank(s paths.Scope) int {
	for i, x := range []paths.Scope{paths.ScopeManaged, paths.ScopeLocal, paths.ScopeProject, paths.ScopeUser, paths.ScopePlugin, paths.ScopeSynced, paths.ScopeDesktop} {
		if x == s {
			return i
		}
	}
	return 99
}

type loader struct {
	inv *Inventory
}

func (l *loader) errf(format string, a ...any) {
	l.inv.Errors = append(l.inv.Errors, fmt.Sprintf(format, a...))
}

// Load builds the inventory. It only reads files.
func Load(env paths.Env) Inventory {
	inv := Inventory{Env: env, LoadedAt: time.Now()}
	l := &loader{inv: &inv}

	inv.Surfaces = append(inv.Surfaces, env.SettingsFiles()...)
	inv.Surfaces = append(inv.Surfaces, env.MemoryFiles()...)
	inv.Surfaces = append(inv.Surfaces, paths.File{Scope: paths.ScopeUser, Label: "user MCP & state (~/.claude.json)", Path: env.ClaudeJSON, Exists: fileExists(env.ClaudeJSON)})
	inv.Surfaces = append(inv.Surfaces, env.DesktopConfigs()...)

	for _, f := range env.MemoryFiles() {
		if f.Exists {
			inv.Items = append(inv.Items, Item{Kind: Memory, Name: f.Label, Scope: f.Scope, Path: f.Path, Editable: f.Scope != paths.ScopeManaged, Content: readText(f.Path)})
		}
	}

	// Skills, agents and commands from the user and project folders.
	l.skills(filepath.Join(env.ClaudeDir, "skills"), paths.ScopeUser, "", true)
	l.syncedSkills(filepath.Join(env.ClaudeDir, "skills", "synced"))
	l.markdown(filepath.Join(env.ClaudeDir, "agents"), Agent, paths.ScopeUser, "", true)
	l.markdown(filepath.Join(env.ClaudeDir, "commands"), Command, paths.ScopeUser, "", true)
	if env.ProjectDir != "" {
		pd := filepath.Join(env.ProjectDir, ".claude")
		l.skills(filepath.Join(pd, "skills"), paths.ScopeProject, "", true)
		l.markdown(filepath.Join(pd, "agents"), Agent, paths.ScopeProject, "", true)
		l.markdown(filepath.Join(pd, "commands"), Command, paths.ScopeProject, "", true)
	}

	// Settings: hooks, enabled plugins, permissions.
	enabled := map[string]bool{}
	for _, f := range env.SettingsFiles() {
		if !f.Exists {
			continue
		}
		var s settingsFile
		if err := readJSON(f.Path, &s); err != nil {
			l.errf("%s: %v", f.Path, err)
			inv.Items = append(inv.Items, Item{Kind: Settings, Name: f.Label, Scope: f.Scope, Path: f.Path, Problems: []string{err.Error()}, Editable: f.Scope != paths.ScopeManaged})
			continue
		}
		for id, on := range s.EnabledPlugins {
			enabled[id] = on
		}
		inv.Items = append(inv.Items, Item{Kind: Settings, Name: f.Label, Scope: f.Scope, Path: f.Path, Content: readText(f.Path), Editable: f.Scope != paths.ScopeManaged, Detail: settingsDetail(s)})
		l.hooks(s.Hooks, f.Scope, f.Label, f.Path)
	}

	// Plugins (and what they bring), marketplaces and their catalogs.
	l.plugins(env, enabled)
	l.marketplaces(env)

	// MCP servers.
	l.claudeJSON(env)
	if env.ProjectDir != "" {
		// MCP servers your administrator deployed (documented: managed-mcp.json next to managed settings)
		if b, err := os.ReadFile(env.ManagedMCP()); err == nil {
			var m struct {
				MCPServers map[string]mcpServer `json:"mcpServers"`
			}
			if json.Unmarshal(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")), &m) == nil {
				l.mcpServers(m.MCPServers, paths.ScopeManaged, "managed-mcp.json", env.ManagedMCP())
			}
		}
		p := filepath.Join(env.ProjectDir, ".mcp.json")
		if fileExists(p) {
			var m mcpFile
			if err := readJSON(p, &m); err != nil {
				l.errf("%s: %v", p, err)
			} else {
				l.mcpServers(m.MCPServers, paths.ScopeProject, ".mcp.json", p)
			}
		}
	}
	for _, f := range env.DesktopConfigs() {
		if !f.Exists {
			continue
		}
		var m mcpFile
		if err := readJSON(f.Path, &m); err != nil {
			l.errf("%s: %v", f.Path, err)
			continue
		}
		l.mcpServers(m.MCPServers, paths.ScopeDesktop, f.Label, f.Path)
	}
	return inv
}

// ---- skills, agents, commands ----

func (l *loader) skills(dir string, scope paths.Scope, origin string, editable bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			l.errf("%s: %v", dir, err)
		}
		return
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "synced" {
			continue
		}
		p := filepath.Join(dir, e.Name(), "SKILL.md")
		if !fileExists(p) {
			continue
		}
		l.inv.Items = append(l.inv.Items, skillItem(p, e.Name(), scope, origin, editable))
	}
}

func (l *loader) syncedSkills(dir string) {
	orgs, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	seen := map[string]bool{}
	for _, o := range orgs {
		if !o.IsDir() {
			continue
		}
		entries, _ := os.ReadDir(filepath.Join(dir, o.Name()))
		for _, e := range entries {
			p := filepath.Join(dir, o.Name(), e.Name(), "SKILL.md")
			if !e.IsDir() || !fileExists(p) || seen[e.Name()] {
				continue
			}
			seen[e.Name()] = true // the same skill synced for several accounts is listed once
			l.inv.Items = append(l.inv.Items, skillItem(p, e.Name(), paths.ScopeSynced, "claude.ai", false))
		}
	}
}

func skillItem(p, folder string, scope paths.Scope, origin string, editable bool) Item {
	raw, rerr := readCapped(p)
	doc, perr := frontmatter.Parse(raw)
	if rerr != nil {
		perr = rerr
	}
	it := Item{Kind: Skill, Name: firstNonEmpty(doc.String("name"), folder), Scope: scope, Origin: origin, Path: p,
		Description: doc.String("description"), Invocable: !doc.Bool("disable-model-invocation"), Enabled: true,
		Content: string(raw), Editable: editable}
	it.Problems = frontmatter.ValidateSkill(doc, perr, folder)
	files, _ := os.ReadDir(filepath.Dir(p))
	var names []string
	for _, f := range files {
		names = append(names, f.Name())
	}
	it.Detail = append(it.Detail, "files: "+strings.Join(names, ", "))
	if !it.Invocable {
		it.Detail = append(it.Detail, "manual only (/"+it.Name+"): costs no context until invoked")
	}
	return it
}

func (l *loader) markdown(dir string, kind Kind, scope paths.Scope, origin string, editable bool) {
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				l.errf("%s: %v", p, err)
			}
			return nil
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		raw, rerr := readCapped(p)
		doc, perr := frontmatter.Parse(raw)
		if rerr != nil {
			perr = rerr
		}
		name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		if rel, err := filepath.Rel(dir, p); err == nil && kind == Command {
			name = strings.ReplaceAll(strings.TrimSuffix(rel, filepath.Ext(rel)), string(filepath.Separator), ":")
		}
		it := Item{Kind: kind, Name: firstNonEmpty(doc.String("name"), name), Scope: scope, Origin: origin, Path: p,
			Description: doc.String("description"), Enabled: true, Content: string(raw), Editable: editable}
		if kind == Agent {
			it.Problems = frontmatter.ValidateAgent(doc, perr)
			if m := doc.String("model"); m != "" {
				it.Detail = append(it.Detail, "model: "+m)
			}
			if t := doc.String("tools"); t != "" {
				it.Detail = append(it.Detail, "tools: "+t)
			}
		} else {
			it.Problems = frontmatter.ValidateCommand(doc, perr)
			if h := doc.String("argument-hint"); h != "" {
				it.Detail = append(it.Detail, "arguments: "+h)
			}
		}
		l.inv.Items = append(l.inv.Items, it)
		return nil
	})
}

// ---- settings & hooks ----

type hookEntry struct {
	Matcher string `json:"matcher"`
	Hooks   []struct {
		Type    string `json:"type"`
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
	} `json:"hooks"`
}

type settingsFile struct {
	Model          string                 `json:"model"`
	Hooks          map[string][]hookEntry `json:"hooks"`
	EnabledPlugins map[string]bool        `json:"enabledPlugins"`
	Permissions    struct {
		Allow       []string `json:"allow"`
		Deny        []string `json:"deny"`
		Ask         []string `json:"ask"`
		DefaultMode string   `json:"defaultMode"`
	} `json:"permissions"`
	Env map[string]string `json:"env"`
}

func settingsDetail(s settingsFile) []string {
	var d []string
	if s.Model != "" {
		d = append(d, "model: "+s.Model)
	}
	if s.Permissions.DefaultMode != "" {
		d = append(d, "permissions.defaultMode: "+s.Permissions.DefaultMode)
	}
	d = append(d, fmt.Sprintf("permissions: %d allow · %d ask · %d deny", len(s.Permissions.Allow), len(s.Permissions.Ask), len(s.Permissions.Deny)))
	for k, v := range s.Env {
		d = append(d, "env "+k+"="+redact.Value(k, v))
	}
	n := 0
	for _, hs := range s.Hooks {
		n += len(hs)
	}
	d = append(d, fmt.Sprintf("hooks: %d · plugins enabled/disabled: %d", n, len(s.EnabledPlugins)))
	return d
}

func (l *loader) hooks(h map[string][]hookEntry, scope paths.Scope, origin, file string) {
	events := make([]string, 0, len(h))
	for ev := range h {
		events = append(events, ev)
	}
	sort.Strings(events)
	for _, ev := range events {
		for _, e := range h[ev] {
			for _, x := range e.Hooks {
				name := ev
				if e.Matcher != "" {
					name += " [" + e.Matcher + "]"
				}
				l.inv.Items = append(l.inv.Items, Item{Kind: Hook, Name: name, Scope: scope, Origin: origin, Path: file,
					Description: redact.Text(x.Command), Enabled: true, Content: x.Command,
					Detail: []string{"type: " + x.Type, "command: " + redact.Text(x.Command), fmt.Sprintf("timeout: %ds", x.Timeout)}})
			}
		}
	}
}

// ---- plugins & marketplaces ----

type installedFile struct {
	Plugins map[string][]struct {
		Scope       string `json:"scope"`
		InstallPath string `json:"installPath"`
		Version     string `json:"version"`
		ProjectPath string `json:"projectPath"`
		LastUpdated string `json:"lastUpdated"`
	} `json:"plugins"`
}

func (l *loader) plugins(env paths.Env, enabled map[string]bool) {
	p := filepath.Join(env.PluginsDir(), "installed_plugins.json")
	if !fileExists(p) {
		return
	}
	var f installedFile
	if err := readJSON(p, &f); err != nil {
		l.errf("%s: %v", p, err)
		return
	}
	ids := make([]string, 0, len(f.Plugins))
	for id := range f.Plugins {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		for _, inst := range f.Plugins[id] {
			on, known := enabled[id]
			it := Item{Kind: Plugin, Name: id, Scope: paths.ScopePlugin, Origin: inst.Scope, Path: inst.InstallPath,
				Enabled: on || !known, Description: "v" + inst.Version}
			it.Detail = []string{"installed for: " + firstNonEmpty(inst.ProjectPath, inst.Scope), "path: " + inst.InstallPath, "updated: " + inst.LastUpdated}
			if known && !on {
				it.Detail = append(it.Detail, "disabled in settings (enabledPlugins)")
			}
			if !fileExists(inst.InstallPath) {
				it.Problems = append(it.Problems, "install path does not exist")
			}
			l.inv.Items = append(l.inv.Items, it)
			if !it.Enabled || !fileExists(inst.InstallPath) || inst.Scope == "project" && inst.ProjectPath != "" && !samePath(inst.ProjectPath, env.ProjectDir) {
				continue // only what Claude would load here contributes skills/agents/commands
			}
			origin := id
			l.skills(filepath.Join(inst.InstallPath, "skills"), paths.ScopePlugin, origin, false)
			l.markdown(filepath.Join(inst.InstallPath, "agents"), Agent, paths.ScopePlugin, origin, false)
			l.markdown(filepath.Join(inst.InstallPath, "commands"), Command, paths.ScopePlugin, origin, false)
			hp := filepath.Join(inst.InstallPath, "hooks", "hooks.json")
			if fileExists(hp) {
				var hf struct {
					Hooks map[string][]hookEntry `json:"hooks"`
				}
				if err := readJSON(hp, &hf); err != nil {
					l.errf("%s: %v", hp, err)
				} else {
					l.hooks(hf.Hooks, paths.ScopePlugin, origin, hp)
				}
			}
		}
	}
}

type marketplaceManifest struct {
	Name    string `json:"name"`
	Owner   any    `json:"owner"`
	Plugins []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Homepage    string `json:"homepage"`
		Version     string `json:"version"`
		Source      any    `json:"source"`
	} `json:"plugins"`
}

func (l *loader) marketplaces(env paths.Env) {
	p := filepath.Join(env.PluginsDir(), "known_marketplaces.json")
	if !fileExists(p) {
		return
	}
	var known map[string]struct {
		Source struct {
			Source string `json:"source"`
			Repo   string `json:"repo"`
			URL    string `json:"url"`
		} `json:"source"`
		InstallLocation string `json:"installLocation"`
		LastUpdated     string `json:"lastUpdated"`
	}
	if err := readJSON(p, &known); err != nil {
		l.errf("%s: %v", p, err)
		return
	}
	installed := map[string]bool{}
	for _, it := range l.inv.Items {
		if it.Kind == Plugin {
			installed[it.Name] = true
		}
	}
	names := make([]string, 0, len(known))
	for n := range known {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		m := known[name]
		src := firstNonEmpty(m.Source.Repo, redact.URL(m.Source.URL))
		it := Item{Kind: Marketplace, Name: name, Origin: m.Source.Source, Path: m.InstallLocation, Description: src, Enabled: true,
			Detail: []string{"source: " + m.Source.Source + " " + src, "updated: " + m.LastUpdated}}
		mf := filepath.Join(m.InstallLocation, ".claude-plugin", "marketplace.json")
		var man marketplaceManifest
		if err := readJSON(mf, &man); err != nil {
			it.Problems = append(it.Problems, "catalog not readable: "+err.Error())
		} else {
			it.Detail = append(it.Detail, fmt.Sprintf("plugins in catalog: %d", len(man.Plugins)))
			for _, pl := range man.Plugins {
				id := pl.Name + "@" + name
				desc := pl.Description
				av := Item{Kind: Available, Name: pl.Name, Origin: name, Description: desc, Enabled: installed[id],
					Detail: []string{"marketplace: " + name, "category: " + pl.Category, "install: claude plugin install " + id}}
				if pl.Homepage != "" {
					av.Detail = append(av.Detail, "homepage: "+pl.Homepage)
				}
				l.inv.Items = append(l.inv.Items, av)
			}
		}
		l.inv.Items = append(l.inv.Items, it)
	}
}

// ---- MCP ----

type mcpServer struct {
	Type    string            `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type mcpFile struct {
	MCPServers map[string]mcpServer `json:"mcpServers"`
}

func (l *loader) claudeJSON(env paths.Env) {
	if !fileExists(env.ClaudeJSON) {
		return
	}
	var cj struct {
		MCPServers map[string]mcpServer `json:"mcpServers"`
		Projects   map[string]struct {
			MCPServers map[string]mcpServer `json:"mcpServers"`
		} `json:"projects"`
	}
	if err := readJSON(env.ClaudeJSON, &cj); err != nil {
		l.errf("%s: %v", env.ClaudeJSON, err)
		return
	}
	l.mcpServers(cj.MCPServers, paths.ScopeUser, "~/.claude.json", env.ClaudeJSON)
	for proj, p := range cj.Projects {
		if env.ProjectDir != "" && samePath(proj, env.ProjectDir) {
			l.mcpServers(p.MCPServers, paths.ScopeLocal, "~/.claude.json (this project)", env.ClaudeJSON)
		}
	}
}

func (l *loader) mcpServers(m map[string]mcpServer, scope paths.Scope, origin, file string) {
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		s := m[n]
		kind := firstNonEmpty(s.Type, map[bool]string{true: "http", false: "stdio"}[s.URL != ""])
		it := Item{Kind: MCP, Name: n, Scope: scope, Origin: origin, Path: file, Enabled: true}
		raw, _ := json.Marshal(s)
		it.Content = string(raw)
		if s.URL != "" {
			it.Description = kind + " " + redact.URL(s.URL)
		} else {
			it.Description = kind + " " + s.Command + " " + strings.Join(redact.Args(s.Args), " ")
		}
		it.Detail = append(it.Detail, "transport: "+kind)
		if s.Command != "" {
			it.Detail = append(it.Detail, "command: "+s.Command, "args: "+strings.Join(redact.Args(s.Args), " "))
		}
		if s.URL != "" {
			it.Detail = append(it.Detail, "url: "+redact.URL(s.URL))
		}
		for _, k := range sortedKeys(s.Env) {
			it.Detail = append(it.Detail, "env "+k+"="+redact.Value(k, s.Env[k]))
		}
		for _, k := range sortedKeys(s.Headers) {
			it.Detail = append(it.Detail, "header "+k+": "+redact.Value(k, s.Headers[k]))
		}
		l.inv.Items = append(l.inv.Items, it)
	}
}

// ---- helpers ----

func readCapped(p string) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, frontmatter.MaxFileBytes+1))
	if err != nil {
		return b, err
	}
	if len(b) > frontmatter.MaxFileBytes {
		return b[:frontmatter.MaxFileBytes], fmt.Errorf("file larger than %d MB, truncated", frontmatter.MaxFileBytes>>20)
	}
	return b, nil
}

func readText(p string) string {
	b, _ := readCapped(p)
	return string(b)
}

func readJSON(p string, v any) error {
	b, err := readCapped(p)
	if err != nil {
		return err
	}
	b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")) // UTF-8 BOM written by some Windows editors
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	ca, cb := filepath.Clean(a), filepath.Clean(b)
	return strings.EqualFold(ca, cb)
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func sortedKeys(m map[string]string) []string {
	k := make([]string, 0, len(m))
	for x := range m {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}

// MergeLive adds what `claude mcp list` reports: claude.ai account connectors (not stored in any local
// file) become MCP items with scope claude.ai, and every local server gets its live status.
func MergeLive(inv *Inventory, servers []livemcp.Server) {
	status := map[string]livemcp.Status{}
	for _, s := range servers {
		status[s.Name] = s.Status
	}
	for i := range inv.Items {
		it := &inv.Items[i]
		if it.Kind == MCP && it.Scope != paths.ScopeClaudeAI && it.Scope != paths.ScopeDesktop {
			if st, ok := status[it.Name]; ok {
				it.Status = string(st)
			}
		}
	}
	// drop a previous merge, then add the account connectors
	kept := inv.Items[:0]
	for _, it := range inv.Items {
		if !(it.Kind == MCP && it.Scope == paths.ScopeClaudeAI) {
			kept = append(kept, it)
		}
	}
	inv.Items = kept
	for _, s := range servers {
		if !s.ClaudeAI {
			continue
		}
		it := Item{Kind: MCP, Name: s.DisplayName(), Scope: paths.ScopeClaudeAI, Origin: "claude.ai account", Enabled: true,
			Status: string(s.Status), Description: "http " + s.Target,
			Detail: []string{"transport: http (remote connector managed in claude.ai)", "url: " + s.Target, "status: " + string(s.Status),
				"manage: claude.ai → Settings → Connectors (and /mcp inside Claude Code to authenticate)"}}
		if s.Status == livemcp.NeedsAuth {
			it.Problems = append(it.Problems, "needs authentication in Claude Code (/mcp)")
		}
		if s.Status == livemcp.Failed {
			it.Problems = append(it.Problems, "Claude Code could not connect")
		}
		inv.Items = append(inv.Items, it)
	}
	inv.LiveMCP = true
}
