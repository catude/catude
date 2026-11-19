package inventory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catude/catude/internal/livemcp"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/redact"
)

// rig builds a small but complete Claude setup in a temporary folder: every surface the inventory reads.
func rig(t *testing.T) paths.Env {
	t.Helper()
	r := t.TempDir()
	env := paths.Env{GOOS: "windows", Home: r, ClaudeDir: filepath.Join(r, ".claude"), ClaudeJSON: filepath.Join(r, ".claude.json"),
		ProjectDir: filepath.Join(r, "proj"), AppData: filepath.Join(r, "AppData", "Roaming"), LocalAppData: filepath.Join(r, "AppData", "Local"),
		ProgramFiles: filepath.Join(r, "Program Files")}
	put := func(rel, s string) string {
		p := filepath.Join(r, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	slash := filepath.ToSlash
	plug := slash(filepath.Join(r, ".claude", "plugins", "cache", "tools"))
	other := slash(filepath.Join(r, ".claude", "plugins", "cache", "elsewhere"))
	// memory, skills, agents, commands
	put(".claude/CLAUDE.md", "# me")
	put("proj/CLAUDE.md", "# project")
	put(".claude/skills/tdd/SKILL.md", "---\nname: tdd\ndescription: Test first. Use when writing tests.\n---\nbody")
	put(".claude/skills/manual/SKILL.md", "---\nname: manual\ndescription: Only by hand.\ndisable-model-invocation: true\n---\n")
	put(".claude/skills/not-a-skill/README.md", "no SKILL.md here")
	put(".claude/skills/synced/org1/brand/SKILL.md", "---\nname: brand\ndescription: Brand rules.\n---\n")
	put(".claude/skills/synced/org2/brand/SKILL.md", "---\nname: brand\ndescription: Brand rules.\n---\n")
	put("proj/.claude/skills/deploy/SKILL.md", "---\nname: deploy\ndescription: Deploy the app.\n---\n")
	put(".claude/agents/reviewer.md", "---\nname: reviewer\ndescription: Reviews code.\nmodel: sonnet\ntools: Read, Grep\n---\n")
	put(".claude/commands/git/sync.md", "---\ndescription: Sync.\nargument-hint: <branch>\n---\n")
	put(".claude/commands/notes.txt", "not markdown")
	// settings: user (with a BOM, as Notepad writes it), project with a hook, plugins on and off
	put(".claude/settings.json", "\xef\xbb\xbf"+`{"model": "opus", "env": {"GITHUB_TOKEN": "ghp_`+strings.Repeat("A", 36)+`", "LOG": "1"},
		"permissions": {"allow": ["Bash(go test:*)"], "deny": ["Read(./.env)"]},
		"enabledPlugins": {"tools@market": true, "off@market": false}}`)
	put("proj/.claude/settings.json", `{"hooks": {"PostToolUse": [{"matcher": "Edit", "hooks": [{"type": "command", "command": "gofmt -w .", "timeout": 30}]}]}}`)
	put("proj/.claude/settings.local.json", `{not json`)
	// plugins: one enabled with a skill and a hook, one disabled, one installed for another project
	put(".claude/plugins/installed_plugins.json", `{"plugins": {
		"tools@market": [{"scope": "user", "installPath": "`+plug+`", "version": "1.2.0"}],
		"off@market": [{"scope": "user", "installPath": "`+plug+`", "version": "0.1.0"}],
		"far@market": [{"scope": "project", "installPath": "`+other+`", "version": "2.0.0", "projectPath": "/some/other/project"}],
		"gone@market": [{"scope": "user", "installPath": "/does/not/exist", "version": "1.0.0"}]}}`)
	put(".claude/plugins/cache/tools/skills/lint/SKILL.md", "---\nname: lint\ndescription: Lint the code.\n---\n")
	put(".claude/plugins/cache/tools/hooks/hooks.json", `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "echo done"}]}]}}`)
	put(".claude/plugins/cache/elsewhere/skills/far-skill/SKILL.md", "---\nname: far-skill\ndescription: Elsewhere.\n---\n")
	mk := slash(filepath.Join(r, ".claude", "plugins", "marketplaces", "market"))
	put(".claude/plugins/known_marketplaces.json", `{"market": {"source": {"source": "github", "repo": "acme/market"}, "installLocation": "`+mk+`", "lastUpdated": "2026-11-01"},
		"broken": {"source": {"source": "url", "url": "https://user:pass@example.com/m.json"}, "installLocation": "/nowhere"}}`)
	put(".claude/plugins/marketplaces/market/.claude-plugin/marketplace.json", `{"name": "market", "plugins": [
		{"name": "tools", "description": "Dev tools", "category": "dev"}, {"name": "new", "description": "Not installed", "homepage": "https://example.com"}]}`)
	// MCP: user, this project (local), another project, .mcp.json, managed, Claude Desktop
	put(".claude.json", `{"mcpServers": {"github": {"command": "npx", "args": ["-y", "server-github"], "env": {"GITHUB_TOKEN": "ghp_`+strings.Repeat("B", 36)+`"}}},
		"projects": {"`+slash(env.ProjectDir)+`": {"mcpServers": {"db": {"type": "http", "url": "https://db.example.com/mcp", "headers": {"Authorization": "Bearer abcdefghijklmnopqrstuvwxyz123456"}}}},
			"/other": {"mcpServers": {"ignored": {"command": "x"}}}}}`)
	put("proj/.mcp.json", `{"mcpServers": {"docs": {"url": "https://docs.example.com/mcp"}}}`)
	put("Program Files/ClaudeCode/managed-mcp.json", "\xef\xbb\xbf"+`{"mcpServers": {"corp": {"command": "corp-mcp"}}}`)
	put("AppData/Roaming/Claude/claude_desktop_config.json", `{"mcpServers": {"files": {"command": "node", "args": ["files.js"]}}}`)
	return env
}

func find(inv Inventory, k Kind, name string) *Item {
	for i := range inv.Items {
		if inv.Items[i].Kind == k && inv.Items[i].Name == name {
			return &inv.Items[i]
		}
	}
	return nil
}

func TestLoadFindsEverySurface(t *testing.T) {
	env := rig(t)
	inv := Load(env)

	// skills: user, project, synced once, plugin; folders without SKILL.md are not skills
	for _, want := range []struct {
		name  string
		scope paths.Scope
	}{{"tdd", paths.ScopeUser}, {"deploy", paths.ScopeProject}, {"brand", paths.ScopeSynced}, {"lint", paths.ScopePlugin}} {
		if it := find(inv, Skill, want.name); it == nil || it.Scope != want.scope {
			t.Errorf("skill %s (%s): %+v", want.name, want.scope, it)
		}
	}
	if n := inv.Count(Skill); n != 5 {
		t.Errorf("skills: %d, want 5 (tdd, manual, brand once, deploy, lint): %v", n, names(inv.Of(Skill)))
	}
	if it := find(inv, Skill, "manual"); it == nil || it.Invocable || !strings.Contains(strings.Join(it.Detail, " "), "manual only") {
		t.Errorf("disable-model-invocation makes a skill manual: %+v", it)
	}
	if it := find(inv, Skill, "brand"); it == nil || it.Editable {
		t.Error("synced skills come from claude.ai and are not edited here")
	}
	if find(inv, Skill, "far-skill") != nil {
		t.Error("a plugin installed for another project brings nothing here")
	}
	// agents and commands
	if it := find(inv, Agent, "reviewer"); it == nil || strings.Join(it.Detail, "|") != "model: sonnet|tools: Read, Grep" {
		t.Errorf("agent: %+v", it)
	}
	if it := find(inv, Command, "git:sync"); it == nil || it.Detail[0] != "arguments: <branch>" {
		t.Errorf("commands in folders are named folder:name: %+v", it)
	}
	if inv.Count(Command) != 1 {
		t.Error("only .md files are commands")
	}
	// memory
	if inv.Count(Memory) != 2 {
		t.Errorf("memory files: %v", names(inv.Of(Memory)))
	}
	// settings: the BOM file is read, the broken one is reported, secrets never reach the detail
	us := find(inv, Settings, "user settings")
	if us == nil || len(us.Problems) > 0 {
		t.Fatalf("a settings file saved with a BOM must be read: %+v", us)
	}
	d := strings.Join(us.Detail, "\n")
	if !strings.Contains(d, "model: opus") || !strings.Contains(d, "1 allow · 0 ask · 1 deny") || strings.Contains(d, "ghp_") || !strings.Contains(d, "env LOG=1") {
		t.Errorf("settings detail:\n%s", d)
	}
	if ls := find(inv, Settings, "local settings"); ls == nil || len(ls.Problems) == 0 || len(inv.Errors) == 0 {
		t.Error("invalid JSON is reported, not fatal")
	}
	// hooks from settings and from plugins
	if it := find(inv, Hook, "PostToolUse [Edit]"); it == nil || it.Scope != paths.ScopeProject || !strings.Contains(strings.Join(it.Detail, " "), "timeout: 30s") {
		t.Errorf("project hook: %+v", it)
	}
	if it := find(inv, Hook, "Stop"); it == nil || it.Origin != "tools@market" {
		t.Errorf("plugin hook: %+v", it)
	}
	// plugins
	if it := find(inv, Plugin, "tools@market"); it == nil || !it.Enabled || it.Description != "v1.2.0" {
		t.Errorf("enabled plugin: %+v", it)
	}
	if it := find(inv, Plugin, "off@market"); it == nil || it.Enabled {
		t.Errorf("enabledPlugins false disables a plugin: %+v", it)
	}
	if it := find(inv, Plugin, "gone@market"); it == nil || len(it.Problems) == 0 {
		t.Errorf("a missing install folder is a problem: %+v", it)
	}
	// marketplaces and what they offer
	if it := find(inv, Marketplace, "market"); it == nil || it.Description != "acme/market" || len(it.Problems) > 0 {
		t.Errorf("marketplace: %+v", it)
	}
	if it := find(inv, Marketplace, "broken"); it == nil || len(it.Problems) == 0 || strings.Contains(it.Description, "pass") {
		t.Errorf("an unreadable catalog is reported and its URL credentials hidden: %+v", it)
	}
	if it := find(inv, Available, "tools"); it == nil || !it.Enabled {
		t.Error("an installed plugin is marked as installed in its catalog")
	}
	if it := find(inv, Available, "new"); it == nil || it.Enabled || !strings.Contains(strings.Join(it.Detail, " "), "claude plugin install new@market") {
		t.Errorf("available plugin: %+v", it)
	}
	// MCP servers from every file, with secrets masked
	for _, want := range []struct {
		name  string
		scope paths.Scope
	}{{"github", paths.ScopeUser}, {"db", paths.ScopeLocal}, {"docs", paths.ScopeProject}, {"corp", paths.ScopeManaged}, {"files", paths.ScopeDesktop}} {
		if it := find(inv, MCP, want.name); it == nil || it.Scope != want.scope {
			t.Errorf("MCP %s (%s): %+v", want.name, want.scope, it)
		}
	}
	if find(inv, MCP, "ignored") != nil {
		t.Error("servers of another project are not listed")
	}
	gh := find(inv, MCP, "github")
	if d := strings.Join(gh.Detail, "\n"); strings.Contains(d, "ghp_") || !strings.Contains(d, redact.Mask) {
		t.Errorf("MCP env secrets are masked:\n%s", d)
	}
	if d := strings.Join(find(inv, MCP, "db").Detail, "\n"); strings.Contains(d, "abcdefghij") {
		t.Errorf("MCP headers are masked:\n%s", d)
	}
	if find(inv, MCP, "docs").Description != "http https://docs.example.com/mcp" {
		t.Error("a server with a url and no type is http")
	}
}

func names(items []Item) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Name+"("+string(it.Scope)+")")
	}
	return out
}

// Of sorts by precedence (managed first) and then by name, ignoring case.
func TestOfSortsByScopeThenName(t *testing.T) {
	inv := Inventory{Items: []Item{{Kind: MCP, Name: "b", Scope: paths.ScopeUser}, {Kind: MCP, Name: "A", Scope: paths.ScopeUser},
		{Kind: MCP, Name: "z", Scope: paths.ScopeManaged}, {Kind: Skill, Name: "s"}, {Kind: MCP, Name: "d", Scope: paths.ScopeDesktop}}}
	if got := strings.Join(names(inv.Of(MCP)), " "); got != "z(managed) A(user) b(user) d(desktop)" {
		t.Errorf("order: %s", got)
	}
}

// claude mcp list adds the account connectors and the live status of local servers; merging twice does not
// duplicate anything.
func TestMergeLive(t *testing.T) {
	inv := Inventory{Items: []Item{{Kind: MCP, Name: "github", Scope: paths.ScopeUser}, {Kind: MCP, Name: "files", Scope: paths.ScopeDesktop}}}
	servers := []livemcp.Server{{Name: "github", Status: livemcp.Connected}, {Name: "files", Status: livemcp.Failed},
		{Name: "claude.ai Gmail", Target: "https://gmail.example/mcp", Status: livemcp.NeedsAuth, ClaudeAI: true}}
	MergeLive(&inv, servers)
	MergeLive(&inv, servers)
	if !inv.LiveMCP || len(inv.Items) != 3 {
		t.Fatalf("items after two merges: %v", names(inv.Items))
	}
	if find(inv, MCP, "github").Status != "connected" {
		t.Error("local servers get their live status")
	}
	if find(inv, MCP, "files").Status != "" {
		t.Error("Claude Desktop servers are not what claude mcp list checks")
	}
	g := find(inv, MCP, "Gmail")
	if g == nil || g.Scope != paths.ScopeClaudeAI || len(g.Problems) == 0 {
		t.Errorf("account connector: %+v", g)
	}
}
