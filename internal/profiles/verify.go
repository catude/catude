package profiles

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/catude/catude/internal/frontmatter"
)

// Check is the result of asking Claude Code itself what a profile loads.
type Check struct {
	Model   string
	Items   []CheckItem
	Raw     string
	Elapsed time.Duration
}

// CheckItem is one thing the profile should bring.
type CheckItem struct {
	Kind, Name string
	Loaded     bool
	Manual     bool   // skill with disable-model-invocation: valid and usable as /name, but /context never lists it
	Problem    string // why it will not load (invalid frontmatter…)
	As         string // how Claude Code shows it, e.g. /profile-swe:review or mcp__db__query
}

// OK reports whether everything loaded.
func (c Check) OK() bool {
	for _, it := range c.Items {
		if !it.Loaded && !it.Manual {
			return false
		}
	}
	return true
}

var reModel = regexp.MustCompile(`\*\*Model:\*\*\s*(\S+)`)

// Verify starts Claude Code headless with the profile (claude -p /context, nothing is sent to the model)
// and checks that every skill, agent, command and MCP server of the profile is really loaded.
func Verify(ctx context.Context, p Profile, o LaunchOptions, dir string) (Check, error) {
	bin, err := exec.LookPath("claude")
	if err != nil {
		return Check{}, errors.New("claude is not on PATH")
	}
	args, env := Args(p, o)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	start := time.Now()
	cmd := exec.CommandContext(ctx, bin, append([]string{"-p", "/context", "--no-session-persistence"}, args...)...)
	cmd.Dir, cmd.Env = dir, append(cmd.Environ(), env...)
	out, err := cmd.Output()
	c := Check{Raw: string(out), Elapsed: time.Since(start)}
	if ctx.Err() != nil {
		return c, errors.New("claude /context timed out")
	}
	if err != nil && len(out) == 0 {
		return c, fmt.Errorf("claude /context failed: %w", err)
	}
	text := string(out)
	if m := reModel.FindStringSubmatch(text); m != nil {
		c.Model = m[1]
	}
	plugin := "profile-" + strings.ToLower(p.Name) + ":"
	for _, x := range []struct {
		kind  string
		names []string
	}{{"skill", p.Skills}, {"agent", p.Agents}, {"command", p.Commands}} {
		for _, n := range x.names {
			as := plugin + n
			shown := "/" + as
			if x.kind == "agent" {
				shown = as + " (subagent type)"
			}
			it := CheckItem{Kind: x.kind, Name: n, Loaded: strings.Contains(text, as), As: shown}
			if !it.Loaded && x.kind == "skill" {
				// skills with disable-model-invocation cost no context, so /context does not list them:
				// check the file itself instead
				b, err := os.ReadFile(filepath.Join(p.PluginDir(), "skills", n, "SKILL.md"))
				doc, perr := frontmatter.Parse(b)
				switch {
				case err != nil:
					it.Problem = "SKILL.md is missing"
				case len(frontmatter.ValidateSkill(doc, perr, n)) > 0:
					it.Problem = strings.Join(frontmatter.ValidateSkill(doc, perr, n), "; ")
				case doc.Bool("disable-model-invocation"):
					it.Manual = true
				default:
					it.Problem = "valid file, but Claude Code did not list it"
				}
			}
			if !it.Loaded && x.kind == "command" {
				// a command without a description (or with disable-model-invocation) can only be typed:
				// the model cannot call it, so /context does not list it
				b, err := os.ReadFile(filepath.Join(p.PluginDir(), "commands", n+".md"))
				doc, _ := frontmatter.Parse(b)
				switch {
				case err != nil:
					it.Problem = "the command file is missing"
				case len(strings.TrimSpace(string(b))) == 0:
					it.Problem = "the command file is empty"
				case doc.String("description") == "" || doc.Bool("disable-model-invocation"):
					it.Manual = true
				default:
					it.Problem = "valid file, but Claude Code did not list it"
				}
			}
			if !it.Loaded && x.kind == "agent" {
				it.Problem = "Claude Code did not list this agent: check its frontmatter (name, description)"
			}
			c.Items = append(c.Items, it)
		}
	}
	for _, n := range p.MCP {
		as := "mcp__" + n + "__"
		it := CheckItem{Kind: "mcp", Name: n, Loaded: strings.Contains(text, as), As: as + "…"}
		if !it.Loaded {
			it.Problem = "the server did not start or needs authentication (see the MCP tab, key m)"
		}
		c.Items = append(c.Items, it)
	}
	return c, nil
}
