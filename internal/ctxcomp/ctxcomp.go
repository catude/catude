// Package ctxcomp gets the exact composition of the context window from Claude Code itself
// (`claude -p /context`): system prompt, tools, MCP, memory files and skills. It is the only exact
// source; catude never estimates tokens by dividing text length.
package ctxcomp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Row is one line of a /context table.
type Row struct {
	Name   string
	Source string // skill source, MCP server or memory type
	Tokens int
}

// Composition is the parsed /context output.
type Composition struct {
	Model      string
	Used, Max  int
	Categories []Row
	Memory     []Row
	Skills     []Row
	MCPTools   []Row
}

// Timeout bounds the claude process.
const Timeout = 3 * time.Minute

// Run executes `claude -p /context` in dir. The command does not call the model.
func Run(ctx context.Context, claudeBin, dir string, extra ...string) (Composition, error) {
	if claudeBin == "" {
		p, err := exec.LookPath("claude")
		if err != nil {
			return Composition{}, errors.New("the claude CLI is not in PATH")
		}
		claudeBin = p
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, claudeBin, append([]string{"-p", "/context", "--no-session-persistence"}, extra...)...)
	cmd.Dir = dir
	cmd.Stdin = nil // claude -p waits for stdin if it is a pipe
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return Composition{}, fmt.Errorf("claude /context timed out after %s", Timeout)
	}
	if err != nil && len(out) == 0 {
		return Composition{}, fmt.Errorf("claude /context failed: %w", err)
	}
	return Parse(string(out))
}

var (
	reTokens = regexp.MustCompile(`\*\*Tokens:\*\*\s*([\d.,]+\s*[kKmM]?)\s*/\s*([\d.,]+\s*[kKmM]?)`)
	reModel  = regexp.MustCompile(`\*\*Model:\*\*\s*(\S+)`)
	reNum    = regexp.MustCompile(`^~?\s*([\d.,]+)\s*([kKmM]?)`)
)

// ErrFormat means Claude Code changed the /context layout; catude refuses to guess.
var ErrFormat = errors.New("unrecognised /context output (Claude Code may have changed its format)")

// Parse reads the Markdown tables printed by /context.
func Parse(text string) (Composition, error) {
	var c Composition
	if m := reModel.FindStringSubmatch(text); m != nil {
		c.Model = m[1]
	}
	if m := reTokens.FindStringSubmatch(text); m != nil {
		c.Used, c.Max = Number(m[1]), Number(m[2])
	}
	section := ""
	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(ln, "### ") {
			section = strings.ToLower(strings.TrimSpace(ln[4:]))
			continue
		}
		if !strings.HasPrefix(ln, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(ln, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if len(cells) < 2 || strings.Trim(cells[0], "-: ") == "" || isHeader(cells[0]) {
			continue
		}
		switch {
		case strings.HasPrefix(section, "estimated usage") && len(cells) >= 2:
			c.Categories = append(c.Categories, Row{Name: cells[0], Tokens: Number(cells[1])})
		case strings.HasPrefix(section, "memory") && len(cells) >= 3:
			c.Memory = append(c.Memory, Row{Name: cells[1], Source: cells[0], Tokens: Number(cells[2])})
		case strings.HasPrefix(section, "skills") && len(cells) >= 3:
			c.Skills = append(c.Skills, Row{Name: cells[0], Source: cells[1], Tokens: Number(cells[2])})
		case strings.HasPrefix(section, "mcp tools") && len(cells) >= 3:
			c.MCPTools = append(c.MCPTools, Row{Name: cells[0], Source: cells[1], Tokens: Number(cells[2])})
		}
	}
	if c.Max == 0 && len(c.Categories) == 0 {
		return c, ErrFormat
	}
	return c, nil
}

func isHeader(s string) bool {
	switch strings.ToLower(s) {
	case "category", "tool", "type", "skill", "agent", "command":
		return true
	}
	return false
}

// Number parses "6.7k", "~70", "1m", "24,900".
func Number(s string) int {
	m := reNum.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64)
	if err != nil {
		return 0
	}
	switch strings.ToLower(m[2]) {
	case "k":
		n *= 1e3
	case "m":
		n *= 1e6
	}
	return int(n + 0.5)
}

// MCPByServer sums MCP tool definition tokens per server (deferred: loaded only when used).
func (c Composition) MCPByServer() []Row {
	idx := map[string]int{}
	var out []Row
	for _, r := range c.MCPTools {
		if i, ok := idx[r.Source]; ok {
			out[i].Tokens += r.Tokens
			continue
		}
		idx[r.Source] = len(out)
		out = append(out, Row{Name: r.Source, Tokens: r.Tokens})
	}
	return out
}
