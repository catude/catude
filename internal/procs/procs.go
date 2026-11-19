// Package procs lists the processes that belong to Claude right now: Claude Code sessions, Claude
// Desktop, and everything they started (MCP servers, hooks, shells). Command lines are redacted, because
// tokens are often passed as arguments.
package procs

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/catude/catude/internal/redact"
)

// Proc is one running process.
type Proc struct {
	PID  int    `json:"pid"`
	PPID int    `json:"ppid"`
	Name string `json:"name"`
	Cmd  string `json:"cmd,omitempty"`  // redacted and shortened
	Cwd  string `json:"cwd,omitempty"`  // only where the OS lets us read it (Linux)
	Root bool   `json:"root,omitempty"` // a Claude process itself (not a child)
}

// Timeout bounds the listing.
const Timeout = 20 * time.Second

// List returns every process the current user can see.
func List(ctx context.Context) ([]Proc, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	switch runtime.GOOS {
	case "windows":
		return listWindows(ctx)
	case "linux":
		if ps, err := listProcFS("/proc"); err == nil && len(ps) > 0 {
			return ps, nil
		}
	}
	return listPS(ctx)
}

func listWindows(ctx context.Context) ([]Proc, error) {
	script := `Get-CimInstance Win32_Process | Select-Object ProcessId,ParentProcessId,Name,CommandLine | ConvertTo-Json -Compress`
	out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return nil, err
	}
	var raw []struct {
		ProcessId       int
		ParentProcessId int
		Name            string
		CommandLine     *string
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	ps := make([]Proc, 0, len(raw))
	for _, r := range raw {
		cmd := ""
		if r.CommandLine != nil {
			cmd = *r.CommandLine
		}
		ps = append(ps, Proc{PID: r.ProcessId, PPID: r.ParentProcessId, Name: r.Name, Cmd: Redact(cmd)})
	}
	return ps, nil
}

func listProcFS(root string) ([]Proc, error) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var ps []Proc
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		stat, err := os.ReadFile(filepath.Join(root, e.Name(), "stat"))
		if err != nil {
			continue
		}
		// "pid (comm) state ppid …": comm may contain spaces and parentheses, so cut at the last ')'
		s := string(stat)
		open, end := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
		if open < 0 || end < open {
			continue
		}
		fields := strings.Fields(s[end+1:])
		if len(fields) < 2 {
			continue
		}
		ppid, _ := strconv.Atoi(fields[1])
		p := Proc{PID: pid, PPID: ppid, Name: s[open+1 : end]}
		if b, err := os.ReadFile(filepath.Join(root, e.Name(), "cmdline")); err == nil && len(b) > 0 {
			p.Cmd = Redact(strings.Join(strings.Split(strings.TrimRight(string(b), "\x00"), "\x00"), " "))
		}
		if cwd, err := os.Readlink(filepath.Join(root, e.Name(), "cwd")); err == nil {
			p.Cwd = cwd
		}
		ps = append(ps, p)
	}
	return ps, nil
}

func listPS(ctx context.Context) ([]Proc, error) {
	out, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,ppid=,args=").Output()
	if err != nil {
		return nil, err
	}
	return parsePS(string(out)), nil
}

func parsePS(out string) []Proc {
	var ps []Proc
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		if err1 != nil || err2 != nil {
			continue
		}
		args := strings.Join(f[2:], " ")
		ps = append(ps, Proc{PID: pid, PPID: ppid, Name: filepath.Base(f[2]), Cmd: Redact(args)})
	}
	return ps
}

// Redact masks secrets in a command line and shortens it.
func Redact(cmd string) string {
	cmd = strings.Join(redact.Args(strings.Fields(cmd)), " ")
	cmd = redact.Text(cmd)
	if r := []rune(cmd); len(r) > 220 {
		cmd = string(r[:219]) + "…"
	}
	return cmd
}

// IsClaude reports whether a process is Claude Code or Claude Desktop itself.
func IsClaude(p Proc) bool {
	name := strings.TrimSuffix(strings.ToLower(p.Name), ".exe")
	if name == "claude" || name == "claude-desktop" || strings.HasPrefix(name, "claude helper") {
		return true
	}
	cmd := strings.ToLower(p.Cmd)
	if (name == "node" || name == "bun") && (strings.Contains(cmd, "@anthropic-ai/claude-code") || strings.Contains(cmd, "claude-code/cli.js")) {
		return true
	}
	return false
}

// ClaudeTree keeps the Claude processes and everything they started, parents first. The process with
// pid self (catude) and its children are left out.
func ClaudeTree(all []Proc, self int) []Proc {
	byPID := map[int]Proc{}
	children := map[int][]int{}
	for _, p := range all {
		if strings.EqualFold(p.Name, "conhost.exe") {
			continue // console host windows: noise
		}
		byPID[p.PID] = p
		children[p.PPID] = append(children[p.PPID], p.PID)
	}
	seen := map[int]bool{}
	var out []Proc
	var walk func(pid int, root bool, depth int)
	walk = func(pid int, root bool, depth int) {
		if seen[pid] || pid == self || depth > 12 {
			return
		}
		seen[pid] = true
		p := byPID[pid]
		p.Root = root
		out = append(out, p)
		kids := children[pid]
		sort.Ints(kids)
		for _, k := range kids {
			if k != pid {
				walk(k, false, depth+1)
			}
		}
	}
	var roots []int
	for _, p := range all {
		if !IsClaude(p) {
			continue
		}
		// a Claude process whose parent is also Claude is not a root (Desktop helpers, sub-sessions)
		if parent, ok := byPID[p.PPID]; ok && IsClaude(parent) && parent.PID != p.PID {
			continue
		}
		roots = append(roots, p.PID)
	}
	sort.Ints(roots)
	for _, r := range roots {
		walk(r, true, 0)
	}
	return out
}
