// catude: one terminal UI for everything that shapes Claude on your machine.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/catude/catude/internal/i18n"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"text/tabwriter"

	"github.com/catude/catude/internal/audit"
	"github.com/catude/catude/internal/ctxcomp"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/livemcp"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/prefs"
	"github.com/catude/catude/internal/sessions"
	"github.com/catude/catude/internal/tui"
)

var version = "0.1.0-dev"

const usage = `catude (ct): manage Claude Code and Claude Desktop from one terminal UI

Usage:
  catude [--project DIR] [--lite]        open the TUI (ctlite = catude --lite: only the tabs you chose)
  catude doctor [--project DIR]          where Claude reads configuration on this machine
  catude ls KIND [--json]                skills | agents | commands | hooks | mcp | plugins | marketplaces | available | settings | memory
  catude audit [--json]                  security findings (exit code 1 if any is high)
  catude sessions [--limit N] [--json]   exact context of recent sessions
  catude context [DIR] [--json]          exact composition of the context window (runs claude /context)
  catude why [--days N]                   where your tokens went: subagents, cache rebuilt after pauses, claude -p, start-up context
  catude instructions [--fix]             which CLAUDE.md, CLAUDE.local.md and AGENTS.md files Claude Code reads here
  catude health [--days N]                tool calls, failures, compactions and tokens per response, by day and by Claude Code version
  catude agents [--days N] [--json]       agent map: every subagent, how often it ran, who launched it, tools and tokens
  catude instances                       every Claude Code binary, profile folder and leftover config copy
  catude discover [SECTION] [QUERY]      curated marketplaces, plugins, MCP, skills, tools, best practices
  catude profile list|new|load|use|switch|current|show|rm   work profiles (catude profile help)
  catude switch [NAME|none]              choose the active work profile (no NAME: pick from a list)
  catude install · uninstall             put catude and ct on your PATH (asked on first run) · remove them
  catude claude [ARGS…]                  start Claude Code with the active work profile (if any)
  catude shell-init [zsh|bash|fish|powershell]   make plain "claude" use the active profile
  catude export [FILE]                   your setup in one file (no secrets), to share or to move to another machine
  catude import FILE|URL [--name N]      check and install a setup someone exported, as a work profile
  catude rig export|show ...             share your setup / load someone else's (Rigcard: catude rig help)
  catude card [--svg FILE]               a shareable summary of your setup (numbers only: no names, no secrets)
  catude presets                         work profiles based on the public setups of well-known practitioners
  catude models [compare A B] [--for coding]  which Claude model and effort is stronger (public data, no key)
  catude graph [mermaid|html|dot] [--json] everything your Claude uses and how it connects (files, hooks, MCP, tools, processes)
  catude debug                           versions, terminal, paths and state for bug reports (no secrets)
  catude snapshot [--live]               render every tab as text (CI, screenshots)
  catude version

Flags: --lang en|es|ja|zh-Hans|zh-Hant|ko|de|fr|pt-BR|ru (interface language, default: your system's) · --project DIR · --json · --live (ls mcp, snapshot: claude.ai connectors and health via claude mcp list)
       --glyphs unicode|safe|ascii · --theme auto|dark|light|high-contrast|mono (or CATUDE_GLYPHS, CATUDE_THEME)
       Inside the TUI, press , for preferences: visible tabs, theme, accent colour, glyphs, lite view.
`

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// stdin answers confirmation prompts (replaced in tests).
var stdin io.Reader = os.Stdin

func run(args []string, out, errw io.Writer) (code int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(errw, "catude: internal error: %v\n%s", r, debug.Stack())
			code = 2
		}
	}()
	// Windows looks for programs in the current folder before PATH: a repository must not be able to
	// replace node, git or claude for the commands catude starts
	os.Setenv("NoDefaultCurrentDirectoryInExePath", "1")
	migrateLegacy()
	glyphs, theme, lite, args := globalFlags(args)
	pf := prefs.Load()
	tui.Accent = pf.Accent
	tui.Setup(firstNonEmpty(glyphs, pf.Glyphs), firstNonEmpty(theme, pf.Theme))
	if v := os.Getenv("CATUDE_LANG"); v != "" {
		i18n.Set(v)
	} else {
		i18n.Set(pf.Language) // empty or "auto": the system language
	}
	out, errw = tui.Writer(out), tui.Writer(errw)
	cmd := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "claude":
		return claudeCmd(args, errw)
	case "models":
		wd, _ := os.Getwd()
		return modelsCmd(paths.Detect(wd), args, out, errw)
	case "card":
		wd, _ := os.Getwd()
		return cardCmd(paths.Detect(wd), args, out, errw)
	case "export": // your setup in one file, without secrets: catude export [FILE]
		wd, _ := os.Getwd()
		file := "claude-setup.json"
		var rest []string
		for _, a := range args {
			if !strings.HasPrefix(a, "-") && file == "claude-setup.json" && len(rest) == 0 {
				file = a
				continue
			}
			rest = append(rest, a)
		}
		code := rigCmd(paths.Detect(wd), append([]string{"export", "-o", file}, rest...), out, errw)
		if code == 0 {
			fmt.Fprintf(out, "Your setup is in %s (no secrets). On another machine: catude import %s\n", file, file)
		}
		return code
	case "import": // a file or URL made by catude export: checked, shown, then installed as a work profile
		wd, _ := os.Getwd()
		return profileCmd(paths.Detect(wd), append([]string{"load"}, args...), stdin, out, errw)
	case "install", "setup":
		return installCmd(args, stdin, out, errw)
	case "uninstall":
		return uninstallCmd(args, stdin, out, errw)
	case "switch", "use":
		wd, _ := os.Getwd()
		if cmd == "switch" && len(args) == 0 {
			return pickProfile(stdin, out, errw)
		}
		return profileCmd(paths.Detect(wd), append([]string{cmd}, args...), stdin, out, errw)
	case "":
		firstRun(stdin, out, errw)
	case "shell-init":
		return shellInit(args, out, errw)
	}
	if cmd == "rig" || cmd == "profile" || cmd == "profiles" {
		wd, _ := os.Getwd()
		if cmd == "rig" {
			return rigCmd(paths.Detect(wd), args, out, errw)
		}
		return profileCmd(paths.Detect(wd), args, stdin, out, errw)
	}
	fs := flag.NewFlagSet("catude", flag.ContinueOnError)
	fs.SetOutput(errw)
	project := fs.String("project", "", "project directory (default: current directory)")
	asJSON := fs.Bool("json", false, "machine-readable output")
	limit := fs.Int("limit", 15, "number of sessions")
	days := fs.Int("days", 30, "days of history (agents)")
	fix := fs.Bool("fix", false, "instructions: make Claude Code read what it skips (asks first)")
	yes := fs.Bool("yes", false, "do not ask for confirmation")
	live := fs.Bool("live", false, "also ask Claude Code (claude mcp list): claude.ai connectors and health")
	fs.Usage = func() { fmt.Fprint(errw, usage) }
	var pos []string
	for len(args) > 0 { // allow flags after positional arguments
		if err := fs.Parse(args); err != nil {
			return 2
		}
		args = fs.Args()
		if len(args) > 0 {
			pos, args = append(pos, args[0]), args[1:]
		}
	}
	if *project == "" {
		*project, _ = os.Getwd()
	}
	env := paths.Detect(*project)

	switch cmd {
	case "", "tui":
		if ok, why := consoleSupportsVT(); !ok {
			fmt.Fprintln(errw, "catude: "+why+".")
			fmt.Fprintln(errw, "  Open it in Windows Terminal (free in the Microsoft Store), or in this window: right-click the title bar →")
			fmt.Fprintln(errw, "  Properties → untick \"Use legacy console\", then open a new window.")
			fmt.Fprintln(errw, "  Every command still works here as plain text: catude doctor, ls, audit, sessions, profile list…")
			return 1
		}
		if err := tui.Run(env, version, tui.Options{Lite: lite || invokedAsLite(), Glyphs: glyphs, Theme: theme, Prefs: pf}); err != nil {
			fmt.Fprintln(errw, "catude:", err)
			return 1
		}
	case "snapshot":
		fmt.Fprint(out, tui.Snapshot(env, version, 140, 34, false, *live))
	case "why":
		return whyCmd(env, *days, out, errw)
	case "instructions":
		return instructionsCmd(env, *fix, *yes, stdin, out)
	case "health":
		return healthCmd(env, *days, out, errw)
	case "agents":
		return agentsCmd(env, *days, *asJSON, out)
	case "instances":
		return instancesCmd(env, out)
	case "presets":
		return presetsCmd(out)
	case "graph":
		return graphCmd(env, pos, *asJSON, out, errw)
	case "debug":
		return debugCmd(env, buildInfo(), out)
	case "discover":
		sec, q := "", ""
		if len(pos) > 0 {
			sec = pos[0]
		}
		if len(pos) > 1 {
			q = strings.Join(pos[1:], " ")
		}
		return discoverCmd(env, sec, q, *asJSON, out, errw)
	case "version", "--version":
		fmt.Fprintln(out, "catude", buildInfo()+platformNote())
	case "help", "-h", "--help":
		fmt.Fprint(out, usage)
	case "doctor":
		return doctor(env, out)
	case "ls":
		if len(pos) == 0 {
			fmt.Fprint(errw, usage)
			return 2
		}
		return ls(env, pos[0], *asJSON, *live, out, errw)
	case "audit":
		return auditCmd(env, *asJSON, out)
	case "sessions":
		return sessionsCmd(env, *limit, *asJSON, out, errw)
	case "context":
		dir := env.ProjectDir
		if len(pos) > 0 {
			dir = pos[0]
		}
		return contextCmd(dir, *asJSON, out, errw)
	default:
		fmt.Fprintf(errw, "catude: unknown command %q\n\n%s", cmd, usage)
		return 2
	}
	return 0
}

func doctor(env paths.Env, out io.Writer) int {
	if n := platformNote(); n != "" {
		fmt.Fprintln(out, strings.TrimSpace(n)+" (see docs/linux.md)")
		fmt.Fprintln(out)
	}
	inv := inventory.Load(env)
	tw := tabwriter.NewWriter(out, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "SCOPE\tSOURCE\tEXISTS\tPATH")
	for _, f := range inv.Surfaces {
		ok := "no"
		if f.Exists {
			ok = "yes"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", f.Scope, f.Label, ok, f.Path)
	}
	tw.Flush()
	fmt.Fprintln(out)
	for _, k := range []inventory.Kind{inventory.Skill, inventory.Agent, inventory.Command, inventory.Hook, inventory.MCP, inventory.Plugin, inventory.Marketplace, inventory.Available} {
		fmt.Fprintf(out, "%-12s %d\n", k, inv.Count(k))
	}
	for _, e := range inv.Errors {
		fmt.Fprintln(out, "error:", e)
	}
	return 0
}

type itemJSON struct {
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Scope       string   `json:"scope"`
	Origin      string   `json:"origin,omitempty"`
	Path        string   `json:"path,omitempty"`
	Description string   `json:"description,omitempty"`
	Invocable   bool     `json:"model_invocable"`
	Enabled     bool     `json:"enabled"`
	Problems    []string `json:"problems,omitempty"`
	Detail      []string `json:"detail,omitempty"`
	Status      string   `json:"status,omitempty"`
}

func ls(env paths.Env, kind string, asJSON, live bool, out, errw io.Writer) int {
	k := inventory.Kind(strings.TrimSuffix(kind, "s"))
	switch kind {
	case "mcp":
		k = inventory.MCP
	case "memory":
		k = inventory.Memory
	case "settings":
		k = inventory.Settings
	case "available":
		k = inventory.Available
	}
	inv := inventory.Load(env)
	if live {
		dir, trusted := env.RunDir()
		if !trusted {
			fmt.Fprintln(errw, "catude: this folder is not trusted in Claude Code yet: showing your user-level servers only (its own MCP servers are not started)")
		}
		servers, err := livemcp.List(context.Background(), "", dir)
		if err != nil {
			fmt.Fprintln(errw, "catude: live MCP:", err)
		} else {
			inventory.MergeLive(&inv, servers)
		}
	}
	items := inv.Of(k)
	if asJSON {
		js := make([]itemJSON, 0, len(items)) // Content (raw text) is never exported
		for _, it := range items {
			js = append(js, itemJSON{string(it.Kind), it.Name, string(it.Scope), it.Origin, it.Path, it.Description, it.Invocable, it.Enabled, it.Problems, it.Detail, it.Status})
		}
		return writeJSON(out, js)
	}
	if len(items) == 0 {
		fmt.Fprintf(errw, "no %s found\n", kind)
		return 0
	}
	tw := tabwriter.NewWriter(out, 2, 4, 2, ' ', 0)
	for _, it := range items {
		flag := ""
		if len(it.Problems) > 0 {
			flag = "! " + it.Problems[0]
		}
		name := it.Name
		if it.Status != "" {
			name = livemcp.Status(it.Status).Mark() + " " + name
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", it.Scope, name, clip(it.Description, 70), flag)
	}
	tw.Flush()
	return 0
}

func auditCmd(env paths.Env, asJSON bool, out io.Writer) int {
	fs := audit.Run(inventory.Load(env))
	high := 0
	for _, f := range fs {
		if f.Severity == audit.High {
			high++
		}
	}
	if asJSON {
		type fj struct{ Severity, Rule, Kind, Item, Path, Detail, Fix string }
		js := make([]fj, 0, len(fs))
		for _, f := range fs {
			js = append(js, fj{f.Severity.String(), f.Rule, string(f.Kind), f.Item, f.Path, f.Detail, f.Fix})
		}
		writeJSON(out, js)
	} else {
		for _, f := range fs {
			fmt.Fprintf(out, "[%s] %s · %s %s\n    %s\n    %s\n    → %s\n", f.Severity, f.Rule, f.Kind, f.Item, f.Detail, f.Path, f.Fix)
		}
		fmt.Fprintf(out, "\n%d findings (%d high)\n", len(fs), high)
	}
	if high > 0 {
		return 1
	}
	return 0
}

func sessionsCmd(env paths.Env, limit int, asJSON bool, out, errw io.Writer) int {
	ps, err := sessions.ListProjects(env.ProjectsDir())
	if err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	var c sessions.Cache
	type sj struct {
		Project, Title, Model string
		ContextTokens, Window int
		ContextPct            float64
		Responses, Compact    int
		CacheHitPct           float64
		Ended                 string
	}
	var rows []sj
	for _, p := range ps {
		if len(rows) >= limit {
			break
		}
		s, err := c.Get(p.Sessions[0])
		if err != nil {
			continue
		}
		rows = append(rows, sj{c.ProjectCWD(p), s.Title, s.Model, s.ContextTokens(), s.Window(), 100 * s.ContextRatio(), s.Responses, s.Compactions, 100 * s.CacheHitRatio(), s.End.Format("2006-01-02 15:04")})
	}
	if asJSON {
		return writeJSON(out, rows)
	}
	tw := tabwriter.NewWriter(out, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ENDED\tCONTEXT\t%\tRESP\tCOMPACT\tCACHE\tTITLE\tPROJECT")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%.1f%%\t%d\t%d\t%.0f%%\t%s\t%s\n", r.Ended, human(r.ContextTokens), r.ContextPct, r.Responses, r.Compact, r.CacheHitPct, clip(r.Title, 34), clip(r.Project, 40))
	}
	tw.Flush()
	return 0
}

func contextCmd(dir string, asJSON bool, out, errw io.Writer) int {
	if e := paths.Detect(dir); !e.ProjectTrusted() {
		fmt.Fprintln(errw, "catude: "+dir+" is not trusted in Claude Code yet: measuring your user-level setup only (its hooks and MCP servers are not started)")
		dir, _ = e.RunDir()
	}
	c, err := ctxcomp.Run(context.Background(), "", dir)
	if err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	if asJSON {
		return writeJSON(out, c)
	}
	fmt.Fprintf(out, "%s · %s of %s at start (source: claude /context)\n\n", c.Model, human(c.Used), human(c.Max))
	for _, sec := range []struct {
		t string
		r []ctxcomp.Row
	}{{"CATEGORIES", c.Categories}, {"MEMORY FILES", c.Memory}, {"SKILLS (always loaded: name + description)", c.Skills}, {"MCP TOOL DEFINITIONS BY SERVER (deferred)", c.MCPByServer()}} {
		if len(sec.r) == 0 {
			continue
		}
		fmt.Fprintln(out, sec.t)
		for _, r := range sec.r {
			fmt.Fprintf(out, "  %-56s %8s\n", clip(r.Name, 56), human(r.Tokens))
		}
		fmt.Fprintln(out)
	}
	return 0
}

func writeJSON(w io.Writer, v any) int {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return 1
	}
	return 0
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

func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}
