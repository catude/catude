// Package tools finds the command-line tools a Claude setup depends on — runtimes that MCP servers start
// (npx, uvx, docker, node, python…), tools that hooks call (rtk…) and companion tools (claude-swap,
// ccusage, gitleaks…) — with their versions, and which MCP servers or hooks would break without them.
package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/catude/catude/internal/inventory"
)

// Tool is one command.
type Tool struct {
	Name     string   `json:"name"`
	Bins     []string `json:"-"`
	Category string   `json:"category"` // runtime | hook & context | security | accounts & usage | dev
	Purpose  string   `json:"purpose"`
	Install  string   `json:"install,omitempty"`
	Path     string   `json:"path,omitempty"`
	Version  string   `json:"version,omitempty"`
	Found    bool     `json:"found"`
	Stub     bool     `json:"stub,omitempty"` // Windows "App execution alias" that opens the Microsoft Store
	UsedBy   []string `json:"used_by,omitempty"`
}

func win(w, other string) string {
	if runtime.GOOS == "windows" {
		return w
	}
	return other
}

// Known lists what catude looks for. Install hints are shown, never run.
func Known() []Tool {
	return []Tool{
		{Name: "claude", Bins: []string{"claude"}, Category: "runtime", Purpose: "Claude Code itself", Install: "npm install -g @anthropic-ai/claude-code"},
		{Name: "node", Bins: []string{"node"}, Category: "runtime", Purpose: "runs JavaScript MCP servers and hooks", Install: win("winget install OpenJS.NodeJS.LTS", "https://nodejs.org (or your package manager)")},
		{Name: "npx", Bins: []string{"npx"}, Category: "runtime", Purpose: "starts npm MCP servers (npx -y …)", Install: "comes with node"},
		{Name: "uv / uvx", Bins: []string{"uvx", "uv"}, Category: "runtime", Purpose: "starts Python MCP servers (uvx …) and installs Python tools", Install: win("winget install astral-sh.uv", "curl -LsSf https://astral.sh/uv/install.sh | sh")},
		{Name: "python", Bins: []string{"python3", "python", "py"}, Category: "runtime", Purpose: "runs Python MCP servers, hooks and skill scripts", Install: win("winget install Python.Python.3.13", "your package manager")},
		{Name: "docker", Bins: []string{"docker"}, Category: "runtime", Purpose: "runs containerised MCP servers", Install: "https://docs.docker.com/get-docker/"},
		{Name: "bun / bunx", Bins: []string{"bunx", "bun"}, Category: "runtime", Purpose: "starts MCP servers with bunx"},
		{Name: "deno", Bins: []string{"deno"}, Category: "runtime", Purpose: "runs Deno MCP servers"},
		{Name: "git", Bins: []string{"git"}, Category: "dev", Purpose: "worktrees, checkpoints and plugin marketplaces", Install: win("winget install Git.Git", "your package manager")},
		{Name: "gh", Bins: []string{"gh"}, Category: "dev", Purpose: "GitHub from Claude Code (PRs, issues)", Install: win("winget install GitHub.cli", "https://cli.github.com")},
		{Name: "ripgrep", Bins: []string{"rg"}, Category: "dev", Purpose: "fast search (Claude Code ships its own)"},
		{Name: "jq", Bins: []string{"jq"}, Category: "dev", Purpose: "JSON in hooks and status lines"},
		{Name: "rtk", Bins: []string{"rtk"}, Category: "hook & context", Purpose: "condenses command output to save context (hook)", Install: "https://github.com/rtk-ai/rtk"},
		{Name: "ccusage", Bins: []string{"ccusage"}, Category: "accounts & usage", Purpose: "cost and token reports", Install: "npx ccusage@latest"},
		{Name: "claude-swap", Bins: []string{"cswap", "claude-swap"}, Category: "accounts & usage", Purpose: "several Claude accounts (catude Profiles tab)", Install: "uv tool install claude-swap==0.29.0"},
		{Name: "gitleaks", Bins: []string{"gitleaks"}, Category: "security", Purpose: "finds secrets before you commit", Install: win("winget install Gitleaks.Gitleaks", "brew install gitleaks")},
		{Name: "semgrep", Bins: []string{"semgrep"}, Category: "security", Purpose: "static analysis", Install: "uv tool install semgrep"},
		{Name: "osv-scanner", Bins: []string{"osv-scanner"}, Category: "security", Purpose: "vulnerable dependencies", Install: "https://google.github.io/osv-scanner/"},
		{Name: "ast-grep", Bins: []string{"ast-grep", "sg"}, Category: "dev", Purpose: "structural code search and rewrite"},
	}
}

var reVersion = regexp.MustCompile(`\d+\.\d+(\.\d+)?([-+.][0-9A-Za-z.]+)?`)

// Detect finds every known tool (in parallel, 4 s per tool) and links them to MCP servers and hooks.
func Detect(ctx context.Context, inv inventory.Inventory) []Tool { return detect(ctx, inv, true) }

// Locate is Detect without asking each tool for its version: no process is started, so it takes
// milliseconds instead of seconds (starting npx, python or docker just to read a version is slow).
func Locate(inv inventory.Inventory) []Tool { return detect(context.Background(), inv, false) }

func detect(ctx context.Context, inv inventory.Inventory, versions bool) []Tool {
	ts := Known()
	var wg sync.WaitGroup
	for i := range ts {
		wg.Add(1)
		go func(t *Tool) {
			defer wg.Done()
			for _, b := range t.Bins {
				p, err := exec.LookPath(b)
				if err != nil {
					continue
				}
				t.Path, t.Found = p, true
				if isStoreStub(p) {
					t.Stub, t.Found = true, false
					return
				}
				if versions {
					t.Version = version(ctx, p)
				}
				return
			}
		}(&ts[i])
	}
	wg.Wait()
	link(ts, inv)
	sort.SliceStable(ts, func(i, j int) bool {
		if (len(ts[i].UsedBy) > 0) != (len(ts[j].UsedBy) > 0) {
			return len(ts[i].UsedBy) > 0
		}
		return false
	})
	return ts
}

func version(ctx context.Context, p string) string {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, p, "--version")
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return ""
	}
	return reVersion.FindString(string(out))
}

// isStoreStub detects the zero-byte python/python3 aliases Windows installs to open the Microsoft Store.
func isStoreStub(p string) bool {
	if runtime.GOOS != "windows" || !strings.Contains(strings.ToLower(p), `\microsoft\windowsapps\`) {
		return false
	}
	base := strings.ToLower(filepath.Base(p))
	if !strings.HasPrefix(base, "python") {
		return false
	}
	fi, err := os.Lstat(p)
	return err == nil && fi.Size() == 0
}

// link records which MCP servers and hooks need each tool.
func link(ts []Tool, inv inventory.Inventory) {
	byBin := map[string]*Tool{}
	for i := range ts {
		for _, b := range ts[i].Bins {
			byBin[b] = &ts[i]
		}
	}
	add := func(t *Tool, who string) {
		for _, u := range t.UsedBy {
			if u == who {
				return
			}
		}
		t.UsedBy = append(t.UsedBy, who)
	}
	for _, it := range inv.Of(inventory.MCP) {
		if cmd := MCPCommand(it); cmd != "" {
			if t := byBin[cmdBase(cmd)]; t != nil {
				add(t, "MCP "+it.Name)
			}
		}
	}
	for _, it := range inv.Of(inventory.Hook) {
		for _, w := range strings.Fields(it.Content) {
			if t := byBin[cmdBase(strings.Trim(w, `"'`))]; t != nil {
				add(t, "hook "+it.Name)
				break
			}
		}
	}
}

// MCPCommand returns the executable of a stdio MCP server ("" for remote servers).
func MCPCommand(it inventory.Item) string {
	var s struct {
		Command string `json:"command"`
		URL     string `json:"url"`
	}
	if json.Unmarshal([]byte(it.Content), &s) != nil || s.URL != "" {
		return ""
	}
	return s.Command
}

func cmdBase(c string) string {
	b := strings.ToLower(filepath.Base(strings.ReplaceAll(c, `\`, "/")))
	for _, ext := range []string{".exe", ".cmd", ".bat", ".ps1"} {
		b = strings.TrimSuffix(b, ext)
	}
	return b
}

// Resolvable reports whether an MCP command can be started from this environment.
func Resolvable(cmd string) bool {
	if cmd == "" {
		return true
	}
	// a UNC or device path (\\host\share\x.exe) is never probed: on Windows even checking whether it
	// exists sends your login to that host
	if strings.HasPrefix(cmd, `\\`) || strings.HasPrefix(cmd, "//") {
		return false
	}
	if filepath.IsAbs(cmd) {
		_, err := os.Stat(cmd)
		return err == nil
	}
	p, err := exec.LookPath(cmd)
	return err == nil && !isStoreStub(p)
}
