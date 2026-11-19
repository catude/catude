// Command demo makes the README screenshots and GIFs from an invented machine — user "alex", invented
// projects, skills, MCP servers and sessions — so no real data ever appears in them.
//
// Steps (scripts/demo/README.md runs them):
//
//	demo fixture -models FILE     build the invented home in $HOME (run with HOME=/home/alex in a sandbox)
//	demo frames  -out DIR         run catude on it with scripted keys, write DIR/NNN.ansi + frames.json
//	demo svg     -in DIR          turn every frame into DIR/NNN.svg (a terminal window)
//	demo gif     -in DIR -set S -out FILE.gif   assemble the PNGs of set S (made from the SVGs) into a GIF
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: demo fixture|frames|svg|gif ...")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "fixture":
		fs := flag.NewFlagSet("fixture", flag.ExitOnError)
		models := fs.String("models", "", "a models-epoch.json cache to copy (public Epoch AI data)")
		fs.Parse(os.Args[2:])
		err = fixture(*models)
	case "frames":
		fs := flag.NewFlagSet("frames", flag.ExitOnError)
		out := fs.String("out", "frames", "output folder")
		fs.Parse(os.Args[2:])
		err = frames(*out)
	case "svg":
		fs := flag.NewFlagSet("svg", flag.ExitOnError)
		in := fs.String("in", "frames", "frames folder")
		fs.Parse(os.Args[2:])
		err = svgAll(*in)
	case "gif":
		fs := flag.NewFlagSet("gif", flag.ExitOnError)
		in := fs.String("in", "frames", "frames folder")
		set := fs.String("set", "hero", "which frames")
		out := fs.String("out", "demo.gif", "output GIF")
		fs.Parse(os.Args[2:])
		err = makeGIF(*in, *set, *out)
	default:
		err = fmt.Errorf("unknown step %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "demo:", err)
		os.Exit(1)
	}
}

// ---- the invented machine ----

const home = "/home/alex"

type project struct {
	dir, slug string
}

var projects = []project{
	{home + "/code/storefront", "-home-alex-code-storefront"},
	{home + "/code/payments-api", "-home-alex-code-payments-api"},
	{home + "/code/infra", "-home-alex-code-infra"},
	{home + "/notes", "-home-alex-notes"},
}

func write(p, s string) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(s), 0o644)
}

func skill(name, desc string) string {
	return fmt.Sprintf("---\nname: %s\ndescription: %q\n---\n\n# %s\n\n%s\n", name, desc, name, desc)
}

func fixture(modelsCache string) error {
	h, _ := os.UserHomeDir()
	if h != home {
		return fmt.Errorf("run with HOME=%s inside a sandbox (now %s)", home, h)
	}
	c := filepath.Join(home, ".claude")
	files := map[string]string{
		c + "/CLAUDE.md": "# Alex's preferences\n\n- Small commits, conventional messages.\n- Run the tests before saying something works.\n- TypeScript strict, Go with table tests.\n",
		c + "/settings.json": `{
  "model": "opus",
  "effortLevel": "high",
  "cleanupPeriodDays": 60,
  "permissions": {
    "defaultMode": "acceptEdits",
    "allow": ["Bash(npm test:*)", "Bash(go test:*)", "Read(~/code/**)"],
    "deny": ["Read(./.env)", "Bash(rm -rf:*)"]
  },
  "env": { "DISABLE_TELEMETRY": "1" },
  "hooks": {
    "PostToolUse": [{ "matcher": "Edit|Write", "hooks": [{ "type": "command", "command": "npx prettier --write \"$CLAUDE_FILE_PATHS\"" }] }],
    "Stop": [{ "hooks": [{ "type": "command", "command": "notify-send 'Claude Code' 'Done'" }] }]
  },
  "enabledPlugins": { "commit-commands@acme-tools": true, "pr-review@acme-tools": true }
}
`,
		home + "/.claude.json": `{
  "numStartups": 214,
  "theme": "dark",
  "mcpServers": {
    "github": { "type": "stdio", "command": "npx", "args": ["-y", "@modelcontextprotocol/server-github"], "env": { "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}" } },
    "playwright": { "type": "stdio", "command": "npx", "args": ["@playwright/mcp@latest"] },
    "postgres": { "type": "stdio", "command": "uvx", "args": ["mcp-server-postgres", "postgresql://localhost/shop"] },
    "linear": { "type": "http", "url": "https://mcp.linear.app/mcp" }
  },
  "projects": {
    "/home/alex/code/storefront": { "hasTrustDialogAccepted": true },
    "/home/alex/code/payments-api": { "hasTrustDialogAccepted": true },
    "/home/alex/code/infra": { "hasTrustDialogAccepted": true }
  }
}
`,
		home + "/code/storefront/.claude/settings.json": `{
  "effortLevel": "xhigh",
  "permissions": { "allow": ["Bash(pnpm test:*)", "Bash(go test:*)"] }
}
`,
		home + "/code/storefront/.claude/settings.local.json": `{
  "model": "sonnet"
}
`,
		home + "/.config/Claude/claude_desktop_config.json": `{
  "mcpServers": {
    "filesystem": { "command": "npx", "args": ["-y", "@modelcontextprotocol/server-filesystem", "/home/alex/notes"] }
  }
}
`,
		c + "/agents/reviewer.md":                                   "---\nname: reviewer\ndescription: Reviews a diff for bugs, missing tests and unclear names before you open a PR\ntools: Read, Grep, Glob\nmodel: opus\n---\n\nYou review code changes.\n",
		c + "/agents/test-runner.md":                                "---\nname: test-runner\ndescription: Runs the test suite, reads the failures and proposes the smallest fix\ntools: Bash, Read\n---\n\nYou run tests.\n",
		c + "/agents/explorer.md":                                   "---\nname: explorer\ndescription: Finds where something is defined and who calls it, without editing\ntools: Read, Grep, Glob\nmodel: haiku\n---\n\nYou explore code.\n",
		c + "/agents/docs-writer.md":                                "---\nname: docs-writer\ndescription: Writes and updates READMEs, guides and changelogs from the code\n---\n\nYou write docs.\n",
		c + "/commands/ship.md":                                     "---\ndescription: Run checks, write the changelog entry and open a PR\n---\n\nShip the current branch.\n",
		c + "/commands/standup.md":                                  "---\ndescription: Summarise yesterday's commits and open PRs for the stand-up\n---\n\nWrite my stand-up.\n",
		c + "/commands/changelog.md":                                "---\ndescription: Draft CHANGELOG.md entries from the commits since the last tag\n---\n\nDraft the changelog.\n",
		projects[0].dir + "/.claude/skills/checkout-flows/SKILL.md": skill("checkout-flows", "How carts, discounts and payments fit together in this storefront"),
		projects[0].dir + "/CLAUDE.md":                              "# Storefront\n\nNext.js 15, pnpm, Playwright for e2e.\n",
		projects[1].dir + "/CLAUDE.md":                              "# Payments API\n\nGo 1.24, Postgres, table tests.\n",
	}
	for name, desc := range map[string]string{
		"tdd":                 "Write the failing test first, then the smallest code that passes, then refactor",
		"code-review":         "Review a change for correctness, security and readability, with concrete suggestions",
		"frontend-design":     "Build distinctive, accessible interfaces with a clear visual hierarchy",
		"release-notes":       "Turn merged PRs into release notes users understand",
		"db-migrations":       "Write reversible database migrations and check them against production-sized data",
		"api-design":          "Design HTTP APIs: resources, errors, pagination, versioning",
		"security-review":     "Look for injection, auth and secret-handling problems in a change",
		"writing-docs":        "Write clear guides: task first, one idea per section, runnable examples",
		"perf-profiling":      "Profile before optimising: measure, find the hot path, verify the gain",
		"incident-postmortem": "Write a blameless post-mortem: timeline, impact, causes, actions",
	} {
		files[c+"/skills/"+name+"/SKILL.md"] = skill(name, desc)
	}
	// plugins from an invented marketplace
	plug := c + "/plugins"
	files[plug+"/installed_plugins.json"] = `{"version":2,"plugins":{
 "commit-commands@acme-tools":[{"scope":"user","installPath":"` + plug + `/cache/acme-tools/commit-commands/1.2.0","version":"1.2.0","lastUpdated":"2026-11-02T10:00:00Z"}],
 "pr-review@acme-tools":[{"scope":"user","installPath":"` + plug + `/cache/acme-tools/pr-review/0.9.1","version":"0.9.1","lastUpdated":"2026-10-20T10:00:00Z"}]}}`
	files[plug+"/known_marketplaces.json"] = `{"acme-tools":{"source":{"source":"github","repo":"acme/claude-tools"},"installLocation":"` + plug + `/marketplaces/acme-tools","lastUpdated":"2026-11-02T10:00:00Z"}}`
	files[plug+"/cache/acme-tools/commit-commands/1.2.0/.claude-plugin/plugin.json"] = `{"name":"commit-commands","description":"Commit, push and open a PR in one go","version":"1.2.0"}`
	files[plug+"/cache/acme-tools/commit-commands/1.2.0/commands/commit-push-pr.md"] = "---\ndescription: Commit, push and open a pull request\n---\n\nDo it.\n"
	files[plug+"/cache/acme-tools/pr-review/0.9.1/.claude-plugin/plugin.json"] = `{"name":"pr-review","description":"Specialised review agents for pull requests","version":"0.9.1"}`
	files[plug+"/cache/acme-tools/pr-review/0.9.1/agents/silent-failure-hunter.md"] = "---\nname: silent-failure-hunter\ndescription: Finds errors that are swallowed or logged and ignored\n---\n\nHunt.\n"
	for p, s := range files {
		if err := write(p, s); err != nil {
			return err
		}
	}
	if err := history(c); err != nil {
		return err
	}
	if modelsCache != "" {
		b, err := os.ReadFile(modelsCache)
		if err != nil {
			return err
		}
		var d map[string]any
		if err := json.Unmarshal(b, &d); err != nil {
			return err
		}
		d["fetched"] = time.Now().Format(time.RFC3339) // fresh: the demo never downloads
		nb, _ := json.Marshal(d)
		if err := write(filepath.Join(home, ".cache/catude/state/models-epoch.json"), string(nb)); err != nil {
			return err
		}
	}
	return nil
}

// ---- transcripts: two weeks of sessions, and three running now ----

type tl struct {
	lines []string
	entry string // how Claude Code was started (cli, claude-vscode, sdk-cli for claude -p)
}

func (t *tl) add(v map[string]any) { b, _ := json.Marshal(v); t.lines = append(t.lines, string(b)) }

func stamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

var reqN int

func (t *tl) answer(at time.Time, cwd, model string, in, cacheR, out int, content []map[string]any) {
	reqN++
	t.answerW(at, cwd, model, in, in/3, cacheR, out, content)
}

// answerW is answer with an explicit cache write (a pause makes the next answer write the context again).
func (t *tl) answerW(at time.Time, cwd, model string, in, cacheW, cacheR, out int, content []map[string]any) {
	reqN++
	entry := t.entry
	if entry == "" {
		entry = "cli"
	}
	t.add(map[string]any{"type": "assistant", "timestamp": stamp(at), "cwd": cwd, "entrypoint": entry, "version": "2.1.326", "effort": effortOf(model), "requestId": fmt.Sprintf("req_%05d", reqN), "gitBranch": "main",
		"message": map[string]any{"id": fmt.Sprintf("msg_%05d", reqN), "model": model, "role": "assistant", "content": content,
			"usage": map[string]any{"input_tokens": in, "cache_creation_input_tokens": cacheW, "cache_read_input_tokens": cacheR, "output_tokens": out}}})
}

// effortOf is the effort level of the invented sessions (Opus at high, the rest at medium).
func effortOf(model string) string {
	if strings.Contains(model, "opus") {
		return "high"
	}
	return "medium"
}

func toolUse(id, name string, input map[string]any) map[string]any {
	return map[string]any{"type": "tool_use", "id": id, "name": name, "input": input}
}

func (t *tl) result(at time.Time, id string) {
	t.add(map[string]any{"type": "user", "timestamp": stamp(at), "message": map[string]any{"role": "user",
		"content": []map[string]any{{"type": "tool_result", "tool_use_id": id, "content": "ok"}}}})
}

func uuid(n int) string { return fmt.Sprintf("%08x-4e1c-4d2a-9b7f-%012x", 0x5ea51000+n, 0xa11ce0000+n) }

var titles = []string{"Fix flaky checkout e2e test", "Add idempotency keys to refunds", "Upgrade to Next.js 15", "Rate-limit the login endpoint",
	"Write the Q4 roadmap notes", "Terraform: move staging to eu-west", "Speed up product search", "Refactor webhook signatures",
	"Dark mode for the dashboard", "Migrate orders table to bigint ids", "Draft release notes 3.4", "Investigate memory spike"}

func history(c string) error {
	now := time.Now()
	n := 0
	for day := 13; day >= 1; day-- {
		for k := 0; k < 1+(day*7)%3; k++ {
			p := projects[(day+k)%len(projects)]
			at := time.Date(now.Year(), now.Month(), now.Day(), 9+(k*3)%8, 10*k, 0, 0, time.Local).AddDate(0, 0, -day)
			var t tl
			t.add(map[string]any{"type": "ai-title", "aiTitle": titles[(day*3+k)%len(titles)]})
			model := "claude-opus-5-5"
			if (day+k)%4 == 0 {
				model = "claude-sonnet-5-5"
			}
			if (day+k)%5 == 0 {
				t.entry = "sdk-cli" // started by a script with claude -p
			}
			turns := 6 + (day*5+k)%14
			gap := time.Duration(0)
			for i := 0; i < turns; i++ {
				if i > 0 && i%8 == 0 { // a coffee break: the cache expires and the next answer writes the context again
					gap += 14 * time.Minute
					t.answerW(at.Add(time.Duration(i)*90*time.Second+gap), p.dir, model, 2000+i*300, 22000+i*2000, 0, 600+i*40, []map[string]any{{"type": "text", "text": "Done."}})
					continue
				}
				t.answer(at.Add(time.Duration(i)*90*time.Second+gap), p.dir, model, 2000+i*300, 40000+i*9000, 600+i*40, []map[string]any{{"type": "text", "text": "Done."}})
			}
			id := uuid(n)
			n++
			tp := filepath.Join(c, "projects", p.slug, id+".jsonl")
			if err := write(tp, strings.Join(t.lines, "\n")+"\n"); err != nil {
				return err
			}
			end := at.Add(time.Duration(turns)*90*time.Second + gap) // the file changed with its last answer
			if err := os.Chtimes(tp, end, end); err != nil {
				return err
			}
		}
	}
	return nil
}

// Live sessions are written by the frames step (it starts the processes they belong to).

func liveSession(c string, pid int, p project, id, name, entry, status string, t *tl) error {
	updated := time.Now()
	if status == "idle" { // a session that has been waiting for a while
		updated = updated.Add(-9 * time.Minute)
	}
	s := fmt.Sprintf(`{"pid":%d,"sessionId":%q,"cwd":%q,"startedAt":%d,"version":"2.1.326","kind":"interactive","entrypoint":%q,"name":%q,"status":%q,"updatedAt":%d}`,
		pid, id, p.dir, time.Now().Add(-40*time.Minute).UnixMilli(), entry, name, status, updated.UnixMilli())
	if err := write(filepath.Join(c, "sessions", fmt.Sprintf("%d.json", pid)), s); err != nil {
		return err
	}
	tp := filepath.Join(c, "projects", p.slug, id+".jsonl")
	if err := write(tp, strings.Join(t.lines, "\n")+"\n"); err != nil {
		return err
	}
	return os.Chtimes(tp, updated, updated)
}

func appendLines(p string, t *tl) {
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	for _, l := range t.lines {
		f.WriteString(l + "\n")
	}
}

// ---- frames ----

type frameInfo struct {
	File string `json:"file"`
	Set  string `json:"set"`
	Name string `json:"name"`
	Hold int    `json:"hold"`
}

const termW, termH = 132, 38

func frames(out string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	c := filepath.Join(home, ".claude")
	// three running sessions, each owned by a real (sleeping) process so catude sees them alive
	var pids []int
	for i := 0; i < 3; i++ {
		cmd := exec.Command("sleep", "900")
		if err := cmd.Start(); err != nil {
			return err
		}
		pids = append(pids, cmd.Process.Pid)
		defer cmd.Process.Kill()
	}
	now := time.Now()
	ago := func(s int) time.Time { return now.Add(-time.Duration(s) * time.Second) }

	var a tl // storefront: working, with a subagent
	a.add(map[string]any{"type": "ai-title", "aiTitle": "Checkout: apply discount codes"})
	a.answer(ago(300), projects[0].dir, "claude-opus-5-5", 3000, 180000, 900, []map[string]any{toolUse("a1", "Read", map[string]any{"file_path": projects[0].dir + "/src/cart/cart.ts"})})
	a.result(ago(298), "a1")
	a.answer(ago(280), projects[0].dir, "claude-opus-5-5", 3000, 190000, 700, []map[string]any{toolUse("a2", "TodoWrite", map[string]any{"todos": []map[string]any{
		{"content": "Read the cart model", "status": "completed", "activeForm": "Reading the cart model"},
		{"content": "Add discount codes to the cart", "status": "in_progress", "activeForm": "Adding discount codes to the cart"},
		{"content": "Show the discount at checkout", "status": "pending", "activeForm": "Showing the discount at checkout"},
		{"content": "Cover it with e2e tests", "status": "pending", "activeForm": "Covering it with e2e tests"}}})})
	a.result(ago(279), "a2")
	a.answer(ago(250), projects[0].dir, "claude-opus-5-5", 3000, 200000, 800, []map[string]any{toolUse("a3", "Grep", map[string]any{"pattern": "applyDiscount"})})
	a.result(ago(249), "a3")
	a.answer(ago(200), projects[0].dir, "claude-opus-5-5", 3000, 210000, 1500, []map[string]any{toolUse("a4", "Edit", map[string]any{"file_path": projects[0].dir + "/src/cart/discount.ts"})})
	a.result(ago(199), "a4")
	a.answer(ago(30), projects[0].dir, "claude-opus-5-5", 3000, 236000, 600, []map[string]any{toolUse("a5", "Agent", map[string]any{"subagent_type": "explorer", "description": "find every caller of computeTotal"})})
	aID := uuid(900)
	// the subagent's own transcript
	var sub tl
	sub.add(map[string]any{"type": "assistant", "isSidechain": true, "timestamp": stamp(ago(5)), "message": map[string]any{"model": "claude-haiku-5-5",
		"content": []map[string]any{toolUse("s1", "Grep", map[string]any{"pattern": "computeTotal("})}}})

	var b tl // payments-api: working, an MCP call running
	b.add(map[string]any{"type": "ai-title", "aiTitle": "Retry failed webhooks with backoff"})
	b.answer(ago(420), projects[1].dir, "claude-opus-5-5", 2500, 120000, 900, []map[string]any{toolUse("b1", "Read", map[string]any{"file_path": projects[1].dir + "/webhooks/deliver.go"})})
	b.result(ago(419), "b1")
	b.answer(ago(380), projects[1].dir, "claude-opus-5-5", 2500, 125000, 2100, []map[string]any{toolUse("b2", "Edit", map[string]any{"file_path": projects[1].dir + "/webhooks/deliver.go"})})
	b.result(ago(379), "b2")
	b.answer(ago(300), projects[1].dir, "claude-opus-5-5", 2500, 128000, 500, []map[string]any{toolUse("b3", "Bash", map[string]any{"command": "go test ./webhooks/...", "description": "Run the webhook tests"})})
	b.result(ago(260), "b3")
	b.answer(ago(12), projects[1].dir, "claude-opus-5-5", 2500, 131000, 400, []map[string]any{toolUse("b4", "mcp__github__create_pull_request", map[string]any{"title": "Retry failed webhooks with exponential backoff"})})
	bID := uuid(901)

	var cc tl // infra: waiting for you
	cc.add(map[string]any{"type": "ai-title", "aiTitle": "Plan the staging changes"})
	cc.answer(ago(700), projects[2].dir, "claude-sonnet-5-5", 2000, 60000, 400, []map[string]any{toolUse("c1", "Bash", map[string]any{"command": "terraform plan -out plan.tfplan", "description": "Plan the staging changes"})})
	cc.result(ago(640), "c1")
	cc.answer(ago(600), projects[2].dir, "claude-sonnet-5-5", 2000, 64000, 900, []map[string]any{{"type": "text", "text": "The plan adds 2 resources and changes 1 (the staging load balancer). Shall I apply it?"}})
	cID := uuid(902)

	if err := liveSession(c, pids[0], projects[0], aID, "storefront-07", "cli", "busy", &a); err != nil {
		return err
	}
	if err := write(filepath.Join(c, "projects", projects[0].slug, aID, "subagents", "agent-explorer-1.jsonl"), strings.Join(sub.lines, "\n")+"\n"); err != nil {
		return err
	}
	// subagent runs of the last weeks, for the agent map (meta file + transcript, as Claude Code writes them)
	type run struct {
		p                                  int
		agent, desc, model, parent         string
		builtIn                            bool
		hoursAgo, minutes, in, cw, cr, out int
		tools                              []string
	}
	runs := []run{
		{0, "explorer", "find every caller of computeTotal", "claude-haiku-5-5", "", false, 0, 1, 3000, 9000, 20000, 900, []string{"Grep", "Read", "Read"}},
		{0, "reviewer", "review the discount code changes", "claude-opus-5-5", "", false, 26, 4, 8000, 42000, 90000, 3100, []string{"Read", "Read", "Grep", "Bash"}},
		{1, "reviewer", "review the webhook retry diff", "claude-opus-5-5", "", false, 50, 3, 6000, 38000, 70000, 2600, []string{"Read", "Grep"}},
		{1, "test-runner", "run the webhook tests and fix failures", "claude-sonnet-5-5", "", false, 3, 6, 9000, 51000, 160000, 4200, []string{"Bash", "Read", "Edit", "Bash"}},
		{1, "test-runner", "run the payment tests", "claude-sonnet-5-5", "", false, 75, 5, 7000, 40000, 120000, 3000, []string{"Bash", "Bash", "Read"}},
		{0, "general-purpose", "compare three date libraries", "claude-opus-5-5", "", true, 30, 7, 12000, 64000, 210000, 5200, []string{"WebSearch", "WebFetch", "WebFetch"}},
		{0, "Explore", "map the checkout flow", "claude-haiku-5-5", "general-purpose", true, 30, 2, 4000, 15000, 40000, 1100, []string{"Glob", "Grep", "Read"}},
		{2, "general-purpose", "summarise the terraform modules", "claude-sonnet-5-5", "", true, 120, 4, 9000, 30000, 80000, 2400, []string{"Read", "Read", "Glob"}},
	}
	for i, r := range runs {
		id := fmt.Sprintf("a%03d", i)
		dir := filepath.Join(c, "projects", projects[r.p].slug, uuid(950+r.p), "subagents")
		if i == 0 {
			dir = filepath.Join(c, "projects", projects[0].slug, aID, "subagents")
			id = "explorer-1"
		}
		parent := ""
		if r.parent != "" {
			parent = "a005" // launched by the general-purpose run above
		}
		depth := 1
		if parent != "" {
			depth = 2
		}
		meta, _ := json.Marshal(map[string]any{"agentType": r.agent, "isBuiltIn": r.builtIn, "description": r.desc, "parentAgentId": parent,
			"spawnDepth": depth, "requestShape": "foreground", "finished": i != 0})
		if err := write(filepath.Join(dir, "agent-"+id+".meta.json"), string(meta)); err != nil {
			return err
		}
		if i == 0 {
			continue // its transcript is the live one written above
		}
		start := time.Now().Add(-time.Duration(r.hoursAgo) * time.Hour)
		var lines []string
		for j, t := range r.tools {
			at := start.Add(time.Duration(j*r.minutes*60/len(r.tools)) * time.Second)
			u := map[string]any{"input_tokens": 8 * r.in / len(r.tools), "cache_creation_input_tokens": 8 * r.cw / len(r.tools), "cache_read_input_tokens": r.cr / len(r.tools), "output_tokens": r.out / len(r.tools)}
			l, _ := json.Marshal(map[string]any{"type": "assistant", "isSidechain": true, "cwd": projects[r.p].dir, "timestamp": stamp(at), "requestId": fmt.Sprintf("req-%s-%d", id, j),
				"message": map[string]any{"id": fmt.Sprintf("msg-%s-%d", id, j), "model": r.model, "usage": u, "content": []map[string]any{toolUse(fmt.Sprintf("t%s%d", id, j), t, map[string]any{})}}})
			lines = append(lines, string(l))
		}
		if err := write(filepath.Join(dir, "agent-"+id+".jsonl"), strings.Join(lines, "\n")+"\n"); err != nil {
			return err
		}
	}
	// the conversations those subagents belong to, so Usage counts them like Claude Code would
	for p := 0; p < 3; p++ {
		var t tl
		t.add(map[string]any{"type": "ai-title", "aiTitle": []string{"Compare date libraries", "Webhook retries review", "Terraform module summary"}[p]})
		at := time.Now().Add(-time.Duration(30+40*p) * time.Hour)
		t.answer(at, projects[p].dir, "claude-opus-5-5", 2500, 30000, 700, []map[string]any{{"type": "text", "text": "Done."}})
		tp := filepath.Join(c, "projects", projects[p].slug, uuid(950+p)+".jsonl")
		if err := write(tp, strings.Join(t.lines, "\n")+"\n"); err != nil {
			return err
		}
		os.Chtimes(tp, at, at)
	}
	if err := liveSession(c, pids[1], projects[1], bID, "payments-api-03", "claude-vscode", "busy", &b); err != nil {
		return err
	}
	if err := liveSession(c, pids[2], projects[2], cID, "infra-01", "cli", "idle", &cc); err != nil {
		return err
	}
	transcriptA := filepath.Join(c, "projects", projects[0].slug, aID+".jsonl")
	transcriptB := filepath.Join(c, "projects", projects[1].slug, bID+".jsonl")

	env := paths.Detect(projects[0].dir)
	shot := func(name string, keys ...string) tui.DemoStep {
		return tui.DemoStep{Keys: keys, Frame: true, Hold: 1, Name: name}
	}
	hold := func(n int, keys ...string) tui.DemoStep { return tui.DemoStep{Keys: keys, Frame: true, Hold: n} }

	sets := map[string][]tui.DemoStep{
		// one screenshot per feature
		// one screenshot per tab, plus the presets gallery, preferences and help
		"shots": {
			shot("overview", "1"),
			shot("config", "2", "enter"),
			shot("config-presets", "esc", "p", "down", "down", "down", "down", "down", "enter"),
			shot("config-effective", "esc", "esc", "esc", "2", "down", "down", "down", "down", "enter"),
			shot("live", "esc", "esc", "3"),
			shot("context", "4"),
			shot("usage", "5"),
			shot("models", "6"),
			shot("models-coding", "t"),
			shot("profiles", "7"),
			shot("presets", "p"),
			shot("presets-more", "down", "down", "down"),
			shot("sessions", "esc", "8", "enter"),
			shot("graph", "esc", "esc", "9", "E"),
			shot("skills", "0"),
			shot("agents", "right"),
			shot("commands", "right"),
			shot("mcp", "right"),
			shot("plugins", "right"),
			shot("hooks", "right"),
			shot("available", "right"),
			shot("discover", "right"),
			shot("discover-tag", "/", "#", "t", "e", "s", "t", "i", "n", "g", "enter"),
			{Keys: []string{"esc"}}, // clear the tag filter before moving on
			shot("tools", "right"),
			shot("storage", "right"),
			shot("audit", "right"),
			shot("preferences", ","),
			shot("help", "esc", "?"),
		},
		// every tab, one after another
		"tour": tourSteps(hold),
		// the hero GIF: a tour
		// where the tokens went: the Usage tab over 7, 14 and 30 days
		"quota": {
			hold(5, "5"), hold(3, "["), hold(4, "]"), hold(4, "]"),
		},
		// the agent map: each subagent with its runs, who launched it and what it cost
		"agents": {
			hold(2, "0"), hold(3, "right"), hold(2, "down"), hold(2, "down"), hold(2, "down"), hold(3, "down"), hold(3, "down"),
		},
		// switch a skill off for this project, then a consumer from the Context tab
		"switch": {
			hold(2, "0"), hold(2, "down"), hold(3, "x"), hold(3, "x"), hold(3, "x"), hold(2, "x"),
			hold(3, "4"), hold(2, "down"), hold(4, "x"),
		},
		"hero": {
			hold(4, "5"),
			hold(2, "0"), hold(3, "right"), hold(2, "down"),
			hold(3, "1"),
			hold(4, "3"),
			hold(2, "2"), hold(1, "enter"), hold(1, "down"), hold(1, "down"), hold(1, "down"), hold(3, "/", "e", "f", "f", "o", "r", "t", "enter"),
			hold(1, "esc", "esc"),
			hold(3, "6"), hold(3, "t"),
			hold(3, "9", "E"), hold(1, "down", "down", "down"), hold(2, "down", "down"),
			hold(3, "0"),
			hold(3, "?"),
		},
		// the Live GIF: sessions moving on
		"live": {
			hold(3, "3"),
			{Do: func() {
				var t tl
				t.result(time.Now(), "a5")
				t.answer(time.Now(), projects[0].dir, "claude-opus-5-5", 3000, 241000, 1200, []map[string]any{toolUse("a6", "Edit", map[string]any{"file_path": projects[0].dir + "/src/checkout/Summary.tsx"})})
				appendLines(transcriptA, &t)
			}, Frame: true, Hold: 3},
			{Do: func() {
				var t tl
				t.result(time.Now(), "b4")
				t.answer(time.Now(), projects[1].dir, "claude-opus-5-5", 2500, 133000, 300, []map[string]any{{"type": "text", "text": "Opened PR #482: Retry failed webhooks with exponential backoff."}})
				appendLines(transcriptB, &t)
			}, Frame: true, Hold: 3},
			{Do: func() {
				var t tl
				t.result(time.Now(), "a6")
				t.answer(time.Now(), projects[0].dir, "claude-opus-5-5", 3000, 246000, 500, []map[string]any{toolUse("a7", "Bash", map[string]any{"command": "pnpm test checkout", "description": "Run the checkout tests"})})
				appendLines(transcriptA, &t)
			}, Frame: true, Hold: 4},
			hold(3, "down"),
		},
	}
	var info []frameInfo
	for _, set := range []string{"shots", "hero", "live", "tour", "quota", "agents", "switch"} {
		fs := tui.DemoFrames(env, "0.1.0", termW, termH, sets[set])
		for i, f := range fs {
			name := fmt.Sprintf("%s-%03d", set, i)
			if f.Name != "" {
				name = set + "-" + f.Name
			}
			if err := os.WriteFile(filepath.Join(out, name+".ansi"), []byte(f.View), 0o644); err != nil {
				return err
			}
			info = append(info, frameInfo{File: name, Set: set, Name: f.Name, Hold: f.Hold})
		}
	}
	// the command line, run for real on the invented machine: catude why, agents, health
	self := filepath.Join(filepath.Dir(os.Args[0]), "catude")
	for _, c := range []string{"why", "agents", "health"} {
		cmd := exec.Command(self, c, "--days", "14")
		cmd.Dir = projects[0].dir
		b, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("catude %s: %w", c, err)
		}
		view := "\x1b[38;5;214m$\x1b[0m catude " + c + "\n\n" + wrapText(string(b), termW-2)
		if err := os.WriteFile(filepath.Join(out, "cli-"+c+".ansi"), []byte(view), 0o644); err != nil {
			return err
		}
		info = append(info, frameInfo{File: "cli-" + c, Set: "cli", Name: c, Hold: 1})
	}
	b2, _ := json.MarshalIndent(info, "", " ")
	return os.WriteFile(filepath.Join(out, "frames.json"), b2, 0o644)
}

// tourSteps walks through every tab (20) with the arrow key, opening the presets gallery on the way.
func tourSteps(hold func(int, ...string) tui.DemoStep) []tui.DemoStep {
	steps := []tui.DemoStep{hold(2, "1")}
	for i := 1; i < 20; i++ {
		steps = append(steps, hold(2, "right"))
		if i == 6 { // Profiles: show the presets gallery
			steps = append(steps, hold(3, "p"), hold(2, "down", "down"), hold(1, "esc"))
		}
	}
	return append(steps, hold(2, ","), hold(1, "esc"))
}

// wrapText breaks long lines at spaces, keeping their indentation, so the terminal picture stays in its window.
func wrapText(s string, w int) string {
	var out []string
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
		for len([]rune(line)) > w {
			r := []rune(line)
			cut := w
			for cut > len(indent)+10 && r[cut] != ' ' {
				cut--
			}
			out = append(out, string(r[:cut]))
			line = indent + "  " + strings.TrimLeft(string(r[cut:]), " ")
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n") + "\n"
}
