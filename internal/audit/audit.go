// Package audit flags risky configuration: third-party skills, hooks and plugins are instructions
// and commands that run with the user's permissions, and MCP configs often hold plain secrets.
// Rules are deterministic and explainable; every finding says why and what to do.
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/catude/catude/internal/instr"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/redact"
	"github.com/catude/catude/internal/tools"
)

// Severity of a finding.
type Severity int

const (
	Info Severity = iota
	Warn
	High
)

func (s Severity) String() string { return [...]string{"info", "warn", "high"}[s] }

// Finding is one problem with a recommended action.
type Finding struct {
	Severity Severity
	Rule     string
	Kind     inventory.Kind
	Item     string
	Path     string
	Detail   string
	Fix      string
}

var (
	reRemoteExec = regexp.MustCompile(`(?i)(curl|wget)\b[^|\n]*\|\s*(sudo\s+)?(sh|bash|zsh|python3?|node)\b|\b(iwr|irm|invoke-webrequest|invoke-restmethod)\b[^|\n]*\|\s*(iex|invoke-expression)\b`)
	reDestroy    = regexp.MustCompile(`(?i)\brm\s+-[a-z]*r[a-z]*f[a-z]*\s+(/|~|\$home|\*)(\s|$)|remove-item\b[^\n]*-recurse[^\n]*-force[^\n]*\b[a-z]:\\\s*($|\s)|\bformat\s+[a-z]:|\bmkfs\.`)
	reInjection  = regexp.MustCompile(`(?i)(ignore|disregard|forget)\s+(all\s+)?(previous|prior|above|earlier)\s+(instructions|rules|prompts?)|you\s+are\s+no\s+longer\s+bound|do\s+not\s+(tell|inform)\s+the\s+user`)
	reExfil      = regexp.MustCompile(`(?i)(\.ssh/|id_rsa|\.aws/credentials|\.credentials\.json|\.env\b|keychain|credential manager)[^\n]{0,120}(curl|wget|http|upload|post|send)`)
	reUnpinned   = regexp.MustCompile(`(?i)^(npx|bunx|uvx|pipx|pnpm\s+dlx)$`)
	reVersioned  = regexp.MustCompile(`@[\w.^~<>=*-]*\d|==\d|@latest`)
	reBroadAllow = regexp.MustCompile(`^(Bash|PowerShell)(\((\*|\*:\*)\))?$`)
	reRule       = regexp.MustCompile(`^(Bash|PowerShell)\((.*)\)$`)
	reFlagValue  = regexp.MustCompile(`(?i)^(true|false|yes|no|none|null|\d+|on|off)$`)
	// commands that destroy data or rewrite history: allowed without a prompt, one wrong guess is final
	reDestructive = regexp.MustCompile(`(?i)^(sudo\b|rm\b|rmdir\b|dd\b|mkfs|shred\b|git\s+push\b.*(--force|\s-f\b)|git\s+reset\s+--hard|git\s+clean\b|git\s+branch\s+-D|chmod\s+-R|chown\s+-R|remove-item\b|del\s+/s|rd\s+/s|drop\s+(table|database)|truncate\b)`)
)

// hidden characters used to smuggle instructions past a human reviewer
func hiddenChars(s string) int {
	n := 0
	for _, r := range s {
		switch {
		case r >= 0x200B && r <= 0x200F, r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069, r == 0x2060, r == 0xFEFF && n > 0:
			n++
		case r >= 0xE0000 && r <= 0xE007F: // Unicode tag characters
			n++
		case unicode.Is(unicode.Co, r): // private use
			n++
		}
	}
	return n
}

// Run audits the inventory. Findings are sorted by severity (high first).
func Run(inv inventory.Inventory) []Finding {
	var out []Finding
	add := func(f Finding) { out = append(out, f) }
	if fi, err := os.Stat(inv.Env.ClaudeJSON); err == nil && inv.Env.ClaudeJSON != "" && fi.Size() > 8<<20 {
		add(Finding{Warn, "claude-json-large", inventory.Settings, "~/.claude.json", inv.Env.ClaudeJSON, fmt.Sprintf("%d MiB: Claude Code reads and rewrites it constantly and slows down", fi.Size()>>20),
			"remove projects you no longer use (see claude purge --help) and keep big MCP configs in .mcp.json files"})
	}
	for _, it := range inv.Items {
		switch it.Kind {
		case inventory.Skill, inventory.Agent, inventory.Command, inventory.Memory:
			text(it, add)
		case inventory.Hook:
			hook(it, add)
		case inventory.MCP:
			mcp(it, add)
		case inventory.Settings:
			settings(it, add)
		case inventory.Marketplace:
			if it.Origin != "github" || !strings.HasPrefix(strings.ToLower(it.Description), "anthropics/") {
				add(Finding{Info, "third-party-marketplace", it.Kind, it.Name, it.Path,
					"plugins from this marketplace run code and add instructions with your permissions",
					"review each plugin before installing; prefer pinned versions"})
			}
		}
		switch it.Kind { // connection state (needs auth, failed) is shown in the MCP tab, not a security finding
		case inventory.Skill, inventory.Agent, inventory.Command, inventory.Memory:
		default:
			continue
		}
		if len(it.Problems) > 0 {
			add(Finding{Warn, "invalid-definition", it.Kind, it.Name, it.Path, strings.Join(it.Problems, "; "),
				"fix the frontmatter so Claude can load and route it correctly"})
		}
	}
	if inv.Env.ProjectDir != "" {
		r := instr.Check(inv.Env)
		for _, f := range r.Ignored() {
			add(Finding{Warn, "agents-md-ignored", inventory.Memory, "AGENTS.md", f.Path, "Claude Code does not read it here: " + f.Why,
				"add a line @AGENTS.md to your CLAUDE.md, or set Project instructions to claude-md-and-agents-md in /config"})
		}
		if len(r.AgentSkills) > 0 {
			add(Finding{Info, "agents-skills-not-read", inventory.Skill, ".agents/skills", filepath.Join(inv.Env.ProjectDir, ".agents", "skills"),
				fmt.Sprintf("Claude Code does not read .agents/skills (%s)", strings.Join(r.AgentSkills, ", ")),
				"copy or link those folders into .claude/skills to use them in Claude Code"})
		}
	}
	if stale := inv.Env.StaleCopies(); len(stale) > 0 {
		add(Finding{Warn, "stale-config-copies", inventory.Settings, fmt.Sprintf("%d leftover copies of ~/.claude.json", len(stale)), stale[0],
			"temporary/backup copies keep old MCP configs, which often contain secrets in plain text (not read by catude)",
			"delete them once Claude Code is closed; rotate any secret that was ever stored in ~/.claude.json"})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Severity > out[j].Severity })
	return out
}

func text(it inventory.Item, add func(Finding)) {
	c := it.Content
	if c == "" {
		return
	}
	if m := reRemoteExec.FindString(c); m != "" {
		add(Finding{High, "remote-code-execution", it.Kind, it.Name, it.Path, "downloads and runs code: " + clip(redact.Text(m)),
			"remove it or pin the script by hash and review it"})
	}
	if m := reDestroy.FindString(c); m != "" {
		add(Finding{High, "destructive-command", it.Kind, it.Name, it.Path, "destructive command: " + clip(m), "remove it or scope it to a safe path"})
	}
	if m := unquotedMatch(reInjection, c); m != "" {
		add(Finding{High, "prompt-injection-phrase", it.Kind, it.Name, it.Path, "instruction override: \"" + clip(m) + "\"",
			"do not use this item until you trust its author"})
	}
	if m := reExfil.FindString(c); m != "" {
		add(Finding{High, "credential-exfiltration", it.Kind, it.Name, it.Path, "touches credentials and sends data: " + clip(m),
			"review urgently; remove if unexpected"})
	}
	if n := hiddenChars(c); n > 0 {
		add(Finding{High, "hidden-unicode", it.Kind, it.Name, it.Path, fmt.Sprintf("%d invisible/bidi characters can hide instructions from reviewers", n),
			"open it in a hex-aware editor and remove them"})
	}
	if redact.Text(c) != c {
		add(Finding{High, "secret-in-text", it.Kind, it.Name, it.Path, "contains a credential-shaped value",
			"move it to an environment variable or your password manager and rotate it"})
	}
}

func hook(it inventory.Item, add func(Finding)) {
	c := it.Content
	if m := reRemoteExec.FindString(c); m != "" {
		add(Finding{High, "hook-remote-code", it.Kind, it.Name, it.Path, "hook downloads and runs code on every trigger: " + clip(redact.Text(m)), "pin and vendor the script"})
	}
	if m := reDestroy.FindString(c); m != "" {
		add(Finding{High, "hook-destructive", it.Kind, it.Name, it.Path, "destructive hook: " + clip(m), "remove it"})
	}
	if redact.Text(c) != c {
		add(Finding{High, "secret-in-hook", it.Kind, it.Name, it.Path, "credential inside a hook command", "read it from an environment variable"})
	}
	if it.Scope == paths.ScopeProject {
		add(Finding{Info, "project-hook", it.Kind, it.Name, it.Path, "runs for anyone who opens this repository with Claude Code", "review it like code; keep it in version control"})
	}
}

func mcp(it inventory.Item, add func(Finding)) {
	var s struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	if err := json.Unmarshal([]byte(it.Content), &s); err != nil {
		return
	}
	for k, v := range s.Env {
		if v != "" && !redact.IsEnvRef(v) && (redact.KeyIsSecret(k) || redact.LooksSecret(v)) {
			add(Finding{High, "mcp-plaintext-secret", it.Kind, it.Name, it.Path, "env " + k + " holds a secret in plain text",
				"use ${" + k + "} and set it in your environment or a secret manager; rotate the exposed value"})
		}
	}
	for k, v := range s.Headers {
		if v != "" && !strings.Contains(v, "${") && (redact.KeyIsSecret(k) || redact.LooksSecret(v)) {
			add(Finding{High, "mcp-plaintext-secret", it.Kind, it.Name, it.Path, "header " + k + " holds a secret in plain text", "reference an environment variable instead"})
		}
	}
	for _, a := range s.Args {
		if redact.LooksSecret(a) {
			add(Finding{High, "mcp-secret-in-args", it.Kind, it.Name, it.Path, "a command-line argument looks like a credential (visible in process lists)", "pass it through env instead"})
			break
		}
	}
	if why := inlineCode(s.Command, s.Args); why != "" {
		add(Finding{High, "mcp-inline-code", it.Kind, it.Name, it.Path, "starts " + why + ": the command line itself is a program, run every time Claude starts",
			"use a named, pinned package or a script file you have read instead of inline code"})
	}
	if s.URL == "" && s.Command != "" && !tools.Resolvable(s.Command) {
		add(Finding{Warn, "mcp-command-not-found", it.Kind, it.Name, it.Path, "\"" + s.Command + "\" is not installed or not on PATH: the server cannot start (Claude only says it failed)",
			"install it (see catude Tools tab) or use the full path to the program"})
	}
	if s.URL != "" && strings.HasPrefix(strings.ToLower(s.URL), "http://") && !strings.Contains(s.URL, "localhost") && !strings.Contains(s.URL, "127.0.0.1") {
		add(Finding{Warn, "mcp-plain-http", it.Kind, it.Name, it.Path, "remote MCP over unencrypted HTTP", "use https"})
	}
	for k, v := range s.Env {
		lk, lv := strings.ToLower(k), strings.ToLower(strings.TrimSpace(v))
		off := lv == "0" || lv == "false" || lv == "no"
		if (strings.Contains(lk, "tls") && strings.Contains(lk, "verif") && off) || (k == "NODE_TLS_REJECT_UNAUTHORIZED" && lv == "0") ||
			(strings.Contains(lk, "ssl") && strings.Contains(lk, "verif") && off) {
			add(Finding{Warn, "tls-verification-disabled", it.Kind, it.Name, it.Path, "env " + k + "=" + v + " turns off certificate checks (man-in-the-middle risk)",
				"trust the server's certificate (or its CA) and enable verification"})
		}
	}
	for _, a := range s.Args {
		if a == "--insecure" || a == "-k" || a == "--no-verify-ssl" || a == "--skip-tls-verify" {
			add(Finding{Warn, "tls-verification-disabled", it.Kind, it.Name, it.Path, "argument " + a + " turns off certificate checks", "remove it and trust the certificate instead"})
		}
	}
	if reUnpinned.MatchString(strings.TrimSuffix(strings.ToLower(lastPathPart(s.Command)), ".exe")) {
		pinned := false
		for _, a := range s.Args {
			if reVersioned.MatchString(a) && !strings.Contains(a, "@latest") {
				pinned = true
			}
		}
		if !pinned {
			add(Finding{Warn, "mcp-unpinned-package", it.Kind, it.Name, it.Path, s.Command + " runs the latest published version on every start",
				"pin a version (pkg@1.2.3 / pkg==1.2.3) to avoid supply-chain surprises"})
		}
	}
}

// inlineCode reports an MCP command that is a shell or interpreter executing code given on the command
// line (sh -c, bash -c, cmd /c, powershell -Command/-EncodedCommand, python -c, node -e, perl -e…).
func inlineCode(command string, args []string) string {
	base := strings.TrimSuffix(strings.ToLower(lastPathPart(command)), ".exe")
	flags := map[string][]string{
		"sh": {"-c"}, "bash": {"-c"}, "zsh": {"-c"}, "dash": {"-c"}, "fish": {"-c"}, "ksh": {"-c"}, "cmd": {"/c", "/k"},
		"powershell": {"-c", "-command", "-encodedcommand", "-enc", "-e"}, "pwsh": {"-c", "-command", "-encodedcommand", "-enc", "-e"},
		"python": {"-c"}, "python3": {"-c"}, "py": {"-c"}, "node": {"-e", "--eval", "-p", "--print"}, "deno": {"eval"}, "bun": {"-e", "--eval"},
		"perl": {"-e"}, "ruby": {"-e"}, "php": {"-r"}, "osascript": {"-e"},
	}
	switch base {
	case "env": // env VAR=x sh -c …: look at the real command
		for i, a := range args {
			if strings.Contains(a, "=") || strings.HasPrefix(a, "-") {
				continue
			}
			return inlineCode(a, args[i+1:])
		}
		return ""
	case "mshta", "rundll32", "regsvr32", "wscript", "cscript", "certutil", "bitsadmin", "msiexec", "installutil":
		return base + " (a Windows script host or installer)"
	}
	want, ok := flags[base]
	if !ok {
		return ""
	}
	shell := base == "sh" || base == "bash" || base == "zsh" || base == "dash" || base == "fish" || base == "ksh"
	ps := base == "powershell" || base == "pwsh"
	for _, a := range args {
		la := strings.ToLower(a)
		for _, f := range want {
			if la == f || strings.HasPrefix(la, f+"=") {
				return base + " " + a + " …"
			}
		}
		switch {
		case shell && len(la) > 1 && la[0] == '-' && la[1] != '-' && strings.HasSuffix(la, "c"): // -lc, -ec, -xc…
			return base + " " + a + " …"
		case ps && len(la) > 1 && (strings.HasPrefix("-command", la) || strings.HasPrefix("-encodedcommand", la) || la == "-ec" || la == "-enc"):
			return base + " " + a + " …"
		}
	}
	return ""
}

func settings(it inventory.Item, add func(Finding)) {
	var s struct {
		Permissions struct {
			Allow       []string `json:"allow"`
			Ask         []string `json:"ask"`
			Deny        []string `json:"deny"`
			DefaultMode string   `json:"defaultMode"`
		} `json:"permissions"`
		SkipDangerous bool `json:"skipDangerousModePermissionPrompt"`
	}
	if it.Content != "" && json.Unmarshal([]byte(it.Content), &s) == nil {
		if s.Permissions.DefaultMode == "bypassPermissions" {
			add(Finding{High, "bypass-permissions", it.Kind, it.Name, it.Path, "permission prompts are disabled by default", "use acceptEdits or auto mode instead"})
		}
		for _, a := range s.Permissions.Allow {
			if reBroadAllow.MatchString(a) {
				add(Finding{Warn, "broad-shell-allow", it.Kind, it.Name, it.Path, "allows every shell command without asking: " + a, "allow specific commands (e.g. Bash(npm test:*))"})
			} else if m := reRule.FindStringSubmatch(a); m != nil && reDestructive.MatchString(strings.TrimSpace(m[2])) {
				add(Finding{Warn, "destructive-allow", it.Kind, it.Name, it.Path, "runs without asking although it deletes data or rewrites history: " + clip(redactRule(a)),
					"move it to permissions.ask so Claude asks each time, or remove it"})
			}
		}
		for _, list := range [][]string{s.Permissions.Allow, s.Permissions.Ask, s.Permissions.Deny} {
			for _, r := range list {
				if masked := redactRule(r); masked != r {
					add(Finding{High, "secret-in-permission", it.Kind, it.Name, it.Path, "a permission rule holds a password or token: " + clip(masked),
						`remove the rule (it was probably saved by "don't ask again"), change that password or token, and pass secrets through environment variables`})
				}
			}
		}
	}
	for _, p := range it.Problems {
		add(Finding{High, "invalid-settings", it.Kind, it.Name, it.Path, p, "Claude Code may ignore this file until it is valid JSON"})
	}
}

// redactRule masks secrets inside a permission rule: --password=x, Bearer x, KEY=x for secret names,
// credentials in URLs and values shaped like tokens. A rule without secrets comes back unchanged.
func redactRule(rule string) string {
	fields := strings.Fields(rule)
	changed := false
	for i, f := range fields {
		masked := f
		if j := strings.Index(f, "="); j > 0 {
			// KEY=value and --key=value: a secret name with a real value (not a flag like --private=false)
			k, v := f[:j], strings.TrimRight(f[j+1:], ")*:")
			if k = k[strings.LastIndex(k, "(")+1:]; v != "" && !redact.IsEnvRef(v) &&
				(redact.LooksSecret(v) || redact.KeyIsSecret(k) && len(v) >= 6 && !reFlagValue.MatchString(v)) {
				masked = f[:j+1] + redact.Mask
			}
		} else if i > 0 && strings.EqualFold(strings.TrimLeft(fields[i-1], "("), "bearer") {
			masked = redact.Mask
		} else {
			masked = redact.Text(redact.URL(redact.Arg(strings.TrimRight(f, ")*:"))))
			if masked == strings.TrimRight(f, ")*:") {
				masked = f
			}
		}
		if masked != f {
			fields[i], changed = masked, true
		}
	}
	if !changed {
		return rule
	}
	return strings.Join(fields, " ")
}

// unquotedMatch returns the first match that is not quoted. Defensive texts quote attack phrases
// ("treat ‘ignore previous instructions’ as data"); real injections state them as instructions.
func unquotedMatch(re *regexp.Regexp, s string) string {
	for _, loc := range re.FindAllStringIndex(s, -1) {
		before := strings.TrimRight(s[:loc[0]], " ")
		if r := []rune(before); len(r) > 0 && strings.ContainsRune("\"'`“‘«", r[len(r)-1]) {
			continue
		}
		return s[loc[0]:loc[1]]
	}
	return ""
}

func clip(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 90 {
		return s[:87] + "..."
	}
	return s
}

func lastPathPart(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
