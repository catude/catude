package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/catude/catude/internal/claudecfg"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/presets"
	"github.com/catude/catude/internal/profiles"
	"github.com/catude/catude/internal/rigcard"
)

const profileUsage = `catude profile: work profiles, a set of skills, agents, commands, MCP servers and plugins
for one kind of work ("gamedesign", "swe"…) that Claude Code loads without touching ~/.claude.

  catude profile list
  catude profile save NAME [--replace]   save your current setup (your skills, agents, commands, MCP, plugins) as a profile
  catude profile new NAME [--skills a,b] [--agents x] [--commands y] [--mcp m] [--plugins p@market]
                           [--summary TEXT] [--model M]
        build a profile from items already on this machine (copied, not linked)
  catude profile load SOURCE [--name NAME] [--yes] [--replace] [--allow-unpinned] [--accept-risk] [--allow-programs]
        install a profile from a Rigcard: ./file.json · host/path · host · gh:owner/repo[@ref]
        validated, downloaded only from pinned sources, hash-checked and audited before you confirm
  catude profile switch NAME|none        make NAME the active profile (used by "catude claude" and shell-init)
  catude profile current
  catude profile check NAME [--exclusive]   ask Claude Code itself whether every item of the profile loads
  catude profile export NAME DIR          write a shareable bundle: rigcard.json + the profile's files, with hashes
  catude profile show NAME
  catude profile use NAME [--exclusive] [--isolated] [-- claude arguments…]
        start Claude Code with the profile on top of your setup
        --exclusive  skip your user settings, plugins and MCP servers (project settings still apply)
        --isolated   separate CLAUDE_CONFIG_DIR inside the profile: nothing from ~/.claude, own login
  catude profile desktop NAME [--config FILE] [--replace] [--yes]
        copy the profile's local MCP servers into Claude Desktop (preview, confirmation and a backup first)
  catude profile rm NAME
  catude profile dir                      where profiles are stored

Profiles never install hooks. MCP secrets are ${ENV_VAR} references you set yourself.
`

func profileCmd(env paths.Env, args []string, in io.Reader, out, errw io.Writer) int {
	if len(args) == 0 {
		args = []string{"list"}
	}
	sub, args := args[0], args[1:]
	var passthrough []string
	for i, a := range args {
		if a == "--" {
			args, passthrough = args[:i], args[i+1:]
			break
		}
	}
	fs := flag.NewFlagSet("profile", flag.ContinueOnError)
	fs.SetOutput(errw)
	fs.Usage = func() { fmt.Fprint(errw, profileUsage) }
	var sel struct{ skills, agents, commands, mcp, plugins string }
	fs.StringVar(&sel.skills, "skills", "", "comma-separated skill names")
	fs.StringVar(&sel.agents, "agents", "", "comma-separated agent names")
	fs.StringVar(&sel.commands, "commands", "", "comma-separated command names")
	fs.StringVar(&sel.mcp, "mcp", "", "comma-separated MCP server names")
	fs.StringVar(&sel.plugins, "plugins", "", "comma-separated plugin ids")
	summary := fs.String("summary", "", "one line about the profile")
	model := fs.String("model", "", "model for this profile (e.g. opus, sonnet)")
	name := fs.String("name", "", "profile name (load)")
	yes := fs.Bool("yes", false, "do not ask for confirmation")
	replace := fs.Bool("replace", false, "replace an existing profile with the same name")
	unpinned := fs.Bool("allow-unpinned", false, "accept sources not pinned to a commit")
	risk := fs.Bool("accept-risk", false, "install even with high-severity audit findings")
	programs := fs.Bool("allow-programs", false, "accept the programs (stdio MCP servers) a loaded card starts")
	exclusive := fs.Bool("exclusive", false, "use only this profile's plugins and MCP servers")
	layered := fs.Bool("layered", false, "load the profile on top of your setup even if it defaults to exclusive")
	isolated := fs.Bool("isolated", false, "separate CLAUDE_CONFIG_DIR (own login)")
	desktopCfg := fs.String("config", "", "Claude Desktop config file, when there are several (desktop)")
	var pos []string
	for len(args) > 0 {
		if err := fs.Parse(args); err != nil {
			return 2
		}
		args = fs.Args()
		if len(args) > 0 {
			pos, args = append(pos, args[0]), args[1:]
		}
	}
	need := func(n int) bool {
		if len(pos) != n {
			fmt.Fprint(errw, profileUsage)
			return false
		}
		return true
	}
	fail := func(err error) int { fmt.Fprintln(errw, "catude:", err); return 1 }
	ctx := context.Background()

	switch sub {
	case "list", "ls":
		ps, err := profiles.List()
		if err != nil {
			return fail(err)
		}
		if len(ps) == 0 {
			fmt.Fprintln(out, "no profiles yet · create one with: catude profile new NAME --skills a,b   or   catude profile load SOURCE")
			return 0
		}
		tw := tabwriter.NewWriter(out, 2, 4, 2, ' ', 0)
		for _, p := range ps {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", p.Name, firstNonEmpty(p.Err, p.Summary()), clip(p.Card.Summary, 60))
		}
		tw.Flush()
	case "switch":
		if len(pos) == 0 {
			return pickProfile(in, out, errw)
		}
		if !need(1) {
			return 2
		}
		if err := profiles.SetActive(pos[0]); err != nil {
			return fail(err)
		}
		if pos[0] == "none" {
			fmt.Fprintln(out, "no active profile: `catude claude` starts your normal setup")
		} else {
			fmt.Fprintln(out, "active profile: "+pos[0]+" · start it with `catude claude` (or plain `claude` after `catude shell-init`)")
		}
	case "current":
		fmt.Fprintln(out, firstNonEmpty(profiles.Active(), "none"))
	case "export":
		if len(pos) != 2 {
			fmt.Fprint(errw, profileUsage)
			return 2
		}
		p, err := profiles.Get(pos[0])
		if err != nil {
			return fail(err)
		}
		res, err := profiles.ExportBundle(p, pos[1], *summary)
		if err != nil {
			for _, f := range res.Blocked {
				fmt.Fprintf(out, "  [%s] %s · %s: %s\n", f.Severity, f.Rule, f.Item, f.Detail)
			}
			return fail(err)
		}
		fmt.Fprintf(out, "bundle written to %s: rigcard.json + %d files (%d skills, %d agents, %d commands)\n", res.Dir, res.Files, len(p.Skills), len(p.Agents), len(p.Commands))
		fmt.Fprintln(out, "Review it, then publish the folder as a GitHub repository. Anyone can load it with:")
		fmt.Fprintln(out, "  catude profile load gh:YOUR-USER/YOUR-REPO --name "+p.Name)
		fmt.Fprintln(out, "or straight from the folder (a shared drive, a zip you sent):")
		fmt.Fprintln(out, "  catude profile load "+filepath.Join(res.Dir, "rigcard.json"))
	case "check", "verify":
		if !need(1) {
			return 2
		}
		p, err := profiles.Get(pos[0])
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(out, "Asking Claude Code what profile %s loads (claude /context, 15–40 s)…\n", p.Name)
		runDir, _ := env.RunDir()
		c, err := profiles.Verify(ctx, p, profiles.LaunchOptions{Exclusive: (*exclusive || p.Options.Exclusive) && !*layered}, runDir)
		if err != nil {
			return fail(err)
		}
		for _, it := range c.Items {
			mark, note := "✓", it.As
			switch {
			case it.Manual:
				mark, note = "○", it.As+"  (manual only: type it to use it; costs no context)"
			case !it.Loaded:
				mark, note = "✗", it.Problem
			}
			fmt.Fprintf(out, "  %s %-8s %-30s %s\n", mark, it.Kind, it.Name, note)
		}
		fmt.Fprintf(out, "  model: %s · checked in %s\n", firstNonEmpty(c.Model, "?"), c.Elapsed.Round(time.Second))
		if !c.OK() {
			fmt.Fprintln(out, "Some items did not load: run catude audit and check that their files are valid (Skills/Agents tabs show the problem).")
			return 1
		}
		fmt.Fprintln(out, "Everything in the profile is available in Claude Code (✓ loaded at start · ○ manual, loaded when you type it).")
	case "dir":
		fmt.Fprintln(out, profiles.Root())
	case "show":
		if !need(1) {
			return 2
		}
		p, err := profiles.Get(pos[0])
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(out, "%s · %s\n%s\n\n", p.Name, p.Summary(), p.Dir)
		for _, x := range []struct {
			t string
			v []string
		}{{"skills", p.Skills}, {"agents", p.Agents}, {"commands", p.Commands}, {"MCP", p.MCP}, {"plugins", p.Plugins}} {
			if len(x.v) > 0 {
				fmt.Fprintf(out, "%-9s %s\n", x.t, strings.Join(x.v, ", "))
			}
		}
		a, e := profiles.Args(p, profiles.LaunchOptions{})
		fmt.Fprintf(out, "\nstarts: %s claude %s\n", strings.Join(e, " "), strings.Join(a, " "))
	case "desktop":
		if !need(1) {
			return 2
		}
		return profileToDesktop(env, pos[0], *desktopCfg, *replace, *yes, in, out, errw)
	case "rm", "remove":
		if !need(1) {
			return 2
		}
		if !*yes && !confirm(in, out, "Delete profile "+pos[0]+" (and its isolated login, if any)?") {
			return 1
		}
		if err := profiles.Remove(pos[0]); err != nil {
			return fail(err)
		}
		fmt.Fprintln(out, "deleted", pos[0])
	case "save":
		if !need(1) {
			return 2
		}
		sel := profiles.CurrentSelection(inventory.Load(env))
		sel.Summary = firstNonEmpty(*summary, "my setup as of "+time.Now().Format("2006-01-02"))
		st, err := profiles.FromLocal(ctx, pos[0], inventory.Load(env), sel)
		if err != nil {
			return fail(err)
		}
		return finish(st, *replace, *yes, *risk, in, out, errw)
	case "new":
		if !need(1) {
			return 2
		}
		s := profiles.Selection{Skills: csv(sel.skills), Agents: csv(sel.agents), Commands: csv(sel.commands), MCP: csv(sel.mcp), Plugins: csv(sel.plugins), Summary: *summary, Model: *model}
		if s.Empty() {
			fmt.Fprintln(errw, "catude: say what goes in the profile, e.g. --skills tdd,grilling --agents reviewer (see catude ls skills)")
			return 2
		}
		st, err := profiles.FromLocal(ctx, pos[0], inventory.Load(env), s)
		if err != nil {
			return fail(err)
		}
		return finish(st, *replace, *yes, *risk, in, out, errw)
	case "load", "install":
		if !need(1) {
			return 2
		}
		var (
			c      rigcard.Card
			raw    []byte
			from   string
			notes  []string
			origin rigcard.Origin
		)
		if id, ok := strings.CutPrefix(pos[0], "preset:"); ok {
			pr, err := presets.Get(id)
			if err != nil {
				return fail(err)
			}
			c, notes = pr.Card()
			from = "preset " + id + " (based on what " + pr.Name + " has published; not affiliated or endorsed)"
			if *name == "" {
				*name = id
			}
		} else {
			var err error
			c, raw, from, err = rigcard.Load(ctx, pos[0])
			if err != nil {
				return fail(err)
			}
			origin = rigcard.OriginOf(pos[0], from)
		}
		for _, n := range notes {
			fmt.Fprintln(out, "· "+n)
		}
		n := *name
		if n == "" {
			n = guessName(c, from)
		}
		fmt.Fprintf(out, "Loading %s as profile %q…\n", from, n)
		inv := inventory.Load(env)
		st, err := profiles.Build(ctx, n, c, profiles.BuildOptions{Raw: raw, AllowUnpinned: *unpinned, Inventory: &inv, Origin: origin})
		if st != nil && len(st.Problems) > 0 {
			for _, p := range st.Problems {
				fmt.Fprintf(out, "  %-6s %s: %s\n", strings.ToUpper(p.Severity), p.Where, p.Msg)
			}
		}
		if err != nil {
			return fail(err)
		}
		return finish(st, *replace, *yes, *risk, in, out, errw, consent{remote: true, programs: *programs})
	case "use", "run":
		if !need(1) {
			return 2
		}
		p, err := profiles.Get(pos[0])
		if err != nil {
			return fail(err)
		}
		cmd, err := profiles.Command(p, profiles.LaunchOptions{Exclusive: (*exclusive || p.Options.Exclusive) && !*layered, Isolated: *isolated}, passthrough...)
		if err != nil {
			return fail(err)
		}
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		fmt.Fprintf(errw, "catude: starting Claude Code with profile %s (%s)\n", p.Name, p.Summary())
		if err := cmd.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return ee.ExitCode()
			}
			return fail(err)
		}
	case "help", "-h", "--help":
		fmt.Fprint(out, profileUsage)
	default:
		fmt.Fprintf(errw, "catude profile: unknown subcommand %q\n\n%s", sub, profileUsage)
		return 2
	}
	return 0
}

// finish shows what a staged profile brings and installs it after consent.
// consent says where a profile comes from and what the user already agreed to on the command line.
type consent struct {
	remote   bool // the card came from a file or URL someone else wrote
	programs bool // --allow-programs
}

func quoteArgv(cmd string, args []string) string {
	parts := []string{cmd}
	for _, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\"'") {
			a = strconv.Quote(a)
		}
		parts = append(parts, a)
	}
	return strings.Join(parts, " ")
}

func finish(st *profiles.Staged, replace, yes, risk bool, in io.Reader, out, errw io.Writer, cs ...consent) int {
	defer st.Discard()
	var who consent
	if len(cs) > 0 {
		who = cs[0]
	}
	c := st.Card
	fmt.Fprintf(out, "\nProfile %s: %d skills · %d agents · %d commands · %d MCP · %d plugins\n", st.Name, len(c.Skills), len(c.Agents), len(c.Commands), len(c.MCPServers), len(c.Plugins))
	var runs, remotes []string
	for _, m := range c.MCPServers {
		var needs []string
		for _, v := range append(append([]rigcard.Variable{}, m.Env...), m.Headers...) {
			needs = append(needs, v.Name)
		}
		line := m.Name + ": "
		if m.Command != "" {
			line += quoteArgv(m.Command, m.Args)
		} else {
			line += "connects to " + m.URL
		}
		if len(needs) > 0 {
			line += "   (reads from your environment: " + strings.Join(needs, ", ") + ")"
		}
		if m.Command != "" {
			runs = append(runs, line)
		} else if m.URL != "" {
			remotes = append(remotes, line)
		}
	}
	if len(runs) > 0 {
		fmt.Fprintf(out, "\n! Starting this profile will RUN these programs on your machine (its MCP servers), shown in full:\n")
		for _, r := range runs {
			fmt.Fprintln(out, "    "+r)
		}
	}
	if len(remotes) > 0 {
		fmt.Fprintf(out, "\n! Starting this profile will CONNECT to these servers and send them the listed variables:\n")
		for _, r := range remotes {
			fmt.Fprintln(out, "    "+r)
		}
	}
	if len(st.Scripts) > 0 {
		fmt.Fprintf(out, "\n! %d executable files come with it. Claude may run them (it asks first unless you allowed it):\n", len(st.Scripts))
		for _, s := range st.Scripts {
			fmt.Fprintln(out, "    "+s)
		}
	}
	if len(st.Findings) > 0 {
		fmt.Fprintln(out, "\nSecurity audit:")
		for _, f := range st.Findings {
			fmt.Fprintf(out, "  [%s] %s · %s %s: %s\n", f.Severity, f.Rule, f.Kind, f.Item, f.Detail)
		}
	} else {
		fmt.Fprintln(out, "\nSecurity audit: no findings.")
	}
	for _, n := range st.Notes {
		fmt.Fprintln(out, "· "+n)
	}
	if len(st.Secrets) > 0 {
		fmt.Fprintln(out, "· set these environment variables before using it: "+strings.Join(st.Secrets, ", "))
	}
	if len(st.Missing) > 0 {
		fmt.Fprintln(out, "· plugins not installed here (run these yourself if you want them):")
		for _, m := range st.Missing {
			fmt.Fprintln(out, "    "+m)
		}
	}
	if st.HighFindings() > 0 && !risk {
		fmt.Fprintf(errw, "\ncatude: not installed: %d high-severity findings. Review them; to install anyway use --accept-risk\n", st.HighFindings())
		return 1
	}
	if who.remote && len(runs) > 0 && !who.programs {
		if yes {
			fmt.Fprintf(errw, "\ncatude: not installed: this profile starts %d programs. Read them above; to accept use --allow-programs\n", len(runs))
			return 1
		}
		if !confirm(in, out, fmt.Sprintf("\nThis card was written by someone else. Allow the %d programs above to run every time the profile starts?", len(runs))) {
			fmt.Fprintln(out, "nothing installed")
			return 1
		}
	}
	if !yes && !confirm(in, out, "\nInstall profile "+st.Name+"?") {
		fmt.Fprintln(out, "nothing installed")
		return 1
	}
	p, err := st.Commit(replace)
	if err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	fmt.Fprintf(out, "installed %s → %s\nstart it with: catude profile use %s\n", p.Name, p.Dir, p.Name)
	return 0
}

func confirm(in io.Reader, out io.Writer, q string) bool {
	fmt.Fprint(out, q+" [y/N] ")
	line, _ := bufio.NewReader(in).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes", "s", "si", "sí":
		return true
	}
	return false
}

func csv(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func guessName(c rigcard.Card, from string) string {
	cand := strings.TrimSuffix(filepath.Base(strings.TrimRight(firstNonEmpty(c.ID, from), "/")), ".json")
	if cand == "rigcard" || cand == "rig" || cand == "profile" || cand == "" {
		if c.Owner != nil && c.Owner.Handle != "" {
			cand = strings.TrimPrefix(c.Owner.Handle, "@")
		} else {
			cand = "imported"
		}
	}
	var b strings.Builder
	for _, r := range strings.ToLower(cand) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return firstNonEmpty(b.String(), "imported")
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// profileToDesktop copies a profile's MCP servers into Claude Desktop's config: preview, confirmation,
// backup, then a reminder to restart the app.
func profileToDesktop(env paths.Env, name, cfg string, replace, yes bool, in io.Reader, out, errw io.Writer) int {
	fail := func(err error) int { fmt.Fprintln(errw, "catude:", err); return 1 }
	p, err := profiles.Get(name)
	if err != nil {
		return fail(err)
	}
	var servers map[string]json.RawMessage
	if b, err := os.ReadFile(p.MCPFile()); err == nil {
		var f struct {
			MCPServers map[string]json.RawMessage `json:"mcpServers"`
		}
		if err := json.Unmarshal(b, &f); err != nil {
			return fail(fmt.Errorf("%s: %w", p.MCPFile(), err))
		}
		servers = f.MCPServers
	}
	if len(servers) == 0 {
		fmt.Fprintln(out, name+" has no MCP servers to copy (claude.ai connectors already reach Claude Desktop through your account)")
		return 0
	}
	var found []claudecfg.Target
	for _, t := range claudecfg.Targets(env) {
		if strings.HasPrefix(t.ID, "desktop:") && (t.Exists || cfg != "") && (cfg == "" || samePath(t.Path, cfg)) {
			found = append(found, t)
		}
	}
	switch {
	case len(found) == 0 && cfg != "":
		return fail(fmt.Errorf("%s is not a Claude Desktop config catude knows (see catude doctor)", cfg))
	case len(found) == 0:
		return fail(errors.New("Claude Desktop's config file was not found: open Claude Desktop once (Settings, Developer, Edit Config creates it)"))
	case len(found) > 1:
		fmt.Fprintln(errw, "catude: there are several Claude Desktop configs; choose one with --config:")
		for _, t := range found {
			fmt.Fprintf(errw, "  %s  (%s)\n", t.Path, t.Label)
		}
		return 2
	}
	t := found[0]
	plan, err := claudecfg.PlanDesktopMCP(t, servers, replace)
	if err != nil {
		return fail(err)
	}
	fmt.Fprintf(out, "Claude Desktop · %s\n%s\n\n", t.Label, t.Path)
	writes := 0
	for _, c := range plan {
		line := fmt.Sprintf("  %-8s %s", c.Action, c.Name)
		if c.Reason != "" {
			line += "  (" + c.Reason + ")"
		}
		fmt.Fprintln(out, line)
		if c.Writes() {
			writes++
		}
	}
	if writes == 0 {
		fmt.Fprintln(out, "\nnothing to change")
		return 0
	}
	fmt.Fprintln(out, "\nThese programs will start every time Claude Desktop opens.")
	if !yes && !confirm(in, out, fmt.Sprintf("Write %d server(s) into Claude Desktop's config (a backup is made first)?", writes)) {
		return 1
	}
	_, backup, err := claudecfg.CopyToDesktopMCP(t, servers, replace)
	if err != nil {
		return fail(err)
	}
	if backup != "" {
		fmt.Fprintln(out, "backup:", backup)
	}
	fmt.Fprintln(out, "done · quit Claude Desktop completely and open it again to load them")
	return 0
}

func samePath(a, b string) bool {
	return a != "" && b != "" && strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
