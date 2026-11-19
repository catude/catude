// Package livemcp asks Claude Code which MCP servers it actually sees (`claude mcp list`), including the
// connectors attached to the user's claude.ai account (Gmail, Drive, company connectors…), which live in
// the account and not in any local file. The command health-checks every server, so it is slow (~20 s):
// callers run it in the background.
package livemcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/catude/catude/internal/redact"
)

// Status of a server as reported by Claude Code.
type Status string

const (
	Connected Status = "connected"
	NeedsAuth Status = "needs-auth"
	Failed    Status = "failed"
	Unknown   Status = "unknown"
)

// Server is one line of `claude mcp list`.
type Server struct {
	Name     string // as shown by Claude Code, e.g. "claude.ai Gmail" or "portainer"
	Target   string // URL or command line, redacted
	Status   Status
	ClaudeAI bool // connector from the claude.ai account
}

// Timeout bounds the health check.
const Timeout = 90 * time.Second

// List runs `claude mcp list` in dir.
func List(ctx context.Context, claudeBin, dir string) ([]Server, error) {
	if claudeBin == "" {
		p, err := exec.LookPath("claude")
		if err != nil {
			return nil, errors.New("the claude CLI is not in PATH")
		}
		claudeBin = p
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, claudeBin, "mcp", "list")
	cmd.Dir = dir
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("claude mcp list timed out after %s", Timeout)
	}
	servers := Parse(string(out))
	if err != nil && len(servers) == 0 {
		return nil, fmt.Errorf("claude mcp list failed: %w", err)
	}
	return servers, nil
}

// Parse reads lines like "claude.ai Gmail: https://… - ✔ Connected".
func Parse(text string) []Server {
	var out []Server
	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		i := strings.Index(ln, ": ")
		j := strings.LastIndex(ln, " - ")
		if i <= 0 || j <= i {
			continue // header ("Checking MCP server health…") or anything unexpected
		}
		name, target, st := ln[:i], strings.TrimSpace(ln[i+2:j]), strings.ToLower(ln[j+3:])
		s := Server{Name: name, ClaudeAI: strings.HasPrefix(name, "claude.ai "), Status: Unknown}
		switch {
		case strings.Contains(st, "connected") && !strings.Contains(st, "not"):
			s.Status = Connected
		case strings.Contains(st, "auth"):
			s.Status = NeedsAuth
		case strings.Contains(st, "fail") || strings.Contains(st, "error") || strings.Contains(st, "✗"):
			s.Status = Failed
		}
		if strings.Contains(target, "://") && !strings.Contains(target, " ") {
			s.Target = redact.URL(target)
		} else {
			s.Target = strings.Join(redact.Args(strings.Fields(target)), " ")
		}
		out = append(out, s)
	}
	return out
}

// DisplayName strips the "claude.ai " prefix.
func (s Server) DisplayName() string { return strings.TrimPrefix(s.Name, "claude.ai ") }

// Mark is a one-character status for lists.
func (s Status) Mark() string {
	switch s {
	case Connected:
		return "✓"
	case NeedsAuth:
		return "!"
	case Failed:
		return "✗"
	}
	return "?"
}
