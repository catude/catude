// Package profiles manages catude work profiles: named sets of skills, agents, commands, MCP servers, plugins
// and safe settings ("game design", "SWE", "data") that Claude Code loads on top of — or instead of — your
// normal setup, without ever modifying ~/.claude.
//
// A profile is described by a Rigcard (profile.json) and materialised as:
//
//	<root>/<name>/profile.json            the card (source of truth, no secrets)
//	<root>/<name>/plugin/                 skills/, agents/, commands/ packaged as a session-only plugin
//	<root>/<name>/mcp.json                MCP servers; secrets are ${ENV_VAR} references
//	<root>/<name>/settings.json           allow-listed settings (model, effort, deny/ask rules, enabled plugins)
//	<root>/<name>/home/                   only for --isolated: a separate CLAUDE_CONFIG_DIR (own login)
//
// and launched with `claude --plugin-dir … --mcp-config … --settings …`. Hooks are never installed.
package profiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/catude/catude/internal/rigcard"
)

const cardFile = "profile.json"

// Root is where profiles live: $CATUDE_HOME/profiles or <user config dir>/catude/profiles.
func Root() string {
	if d := os.Getenv("CATUDE_HOME"); d != "" {
		return filepath.Join(d, "profiles")
	}
	d, err := os.UserConfigDir()
	if err != nil { // never a shared temporary folder: another user could plant profiles there
		home, herr := os.UserHomeDir()
		if herr != nil {
			return filepath.Join(".catude", "profiles")
		}
		return filepath.Join(home, ".catude", "profiles")
	}
	return filepath.Join(d, "catude", "profiles")
}

// Profile is an installed work profile.
type Profile struct {
	Name     string
	Dir      string
	Card     rigcard.Card
	Skills   []string
	Agents   []string
	Commands []string
	MCP      []string
	Plugins  []string
	Options  Options
	Err      string // set when profile.json cannot be read
}

func (p Profile) PluginDir() string    { return filepath.Join(p.Dir, "plugin") }
func (p Profile) MCPFile() string      { return filepath.Join(p.Dir, "mcp.json") }
func (p Profile) SettingsFile() string { return filepath.Join(p.Dir, "settings.json") }
func (p Profile) HomeDir() string      { return filepath.Join(p.Dir, "home") }

// Summary is a one-line description of what the profile brings.
func (p Profile) Summary() string {
	var parts []string
	for _, x := range []struct {
		n    int
		what string
	}{{len(p.Skills), "skills"}, {len(p.Agents), "agents"}, {len(p.Commands), "commands"}, {len(p.MCP), "MCP"}, {len(p.Plugins), "plugins"}} {
		if x.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", x.n, x.what))
		}
	}
	if len(parts) == 0 {
		return "empty"
	}
	return strings.Join(parts, " · ")
}

// List returns every installed profile, sorted by name.
func List() ([]Profile, error) {
	ents, err := os.ReadDir(Root())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Profile
	for _, e := range ents {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p, err := Get(e.Name())
		if err != nil {
			p = Profile{Name: e.Name(), Dir: filepath.Join(Root(), e.Name()), Err: err.Error()}
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Get reads one installed profile.
func Get(name string) (Profile, error) {
	if !rigcard.SafeName(name) {
		return Profile{}, fmt.Errorf("invalid profile name %q", name)
	}
	p := Profile{Name: name, Dir: filepath.Join(Root(), name)}
	b, err := os.ReadFile(filepath.Join(p.Dir, cardFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return p, fmt.Errorf("no profile named %q (see catude profile list)", name)
		}
		return p, err
	}
	if err := json.Unmarshal(b, &p.Card); err != nil {
		return p, fmt.Errorf("%s: %w", cardFile, err)
	}
	p.Skills = dirNames(filepath.Join(p.PluginDir(), "skills"), true)
	p.Agents = dirNames(filepath.Join(p.PluginDir(), "agents"), false)
	p.Commands = dirNames(filepath.Join(p.PluginDir(), "commands"), false)
	var mcp struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if b, err := os.ReadFile(p.MCPFile()); err == nil && json.Unmarshal(b, &mcp) == nil {
		for k := range mcp.MCPServers {
			p.MCP = append(p.MCP, k)
		}
		sort.Strings(p.MCP)
	}
	for _, pl := range p.Card.Plugins {
		p.Plugins = append(p.Plugins, pl.ID)
	}
	if b, err := os.ReadFile(optionsFile(p.Dir)); err == nil {
		json.Unmarshal(b, &p.Options)
	}
	return p, nil
}

func dirNames(dir string, dirs bool) []string {
	ents, _ := os.ReadDir(dir)
	var out []string
	for _, e := range ents {
		if e.IsDir() == dirs {
			out = append(out, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	return out
}

// Remove deletes an installed profile (including its isolated login, if any).
func Remove(name string) error {
	p, err := Get(name)
	if err != nil {
		return err
	}
	return os.RemoveAll(p.Dir)
}

// LaunchOptions choose how a profile combines with your normal setup.
type LaunchOptions struct {
	// Exclusive skips your user settings (and so your user plugins, hooks and permissions) and every MCP server
	// that is not in the profile. Project settings still apply.
	Exclusive bool
	// Isolated runs Claude Code with its own CLAUDE_CONFIG_DIR inside the profile: nothing from ~/.claude
	// is loaded, and you log in once for that profile.
	Isolated bool
}

// Args returns the claude arguments and extra environment that load the profile.
func Args(p Profile, o LaunchOptions) (args []string, env []string) {
	if len(p.Skills)+len(p.Agents)+len(p.Commands) > 0 {
		args = append(args, "--plugin-dir", p.PluginDir())
	}
	if len(p.MCP) > 0 {
		args = append(args, "--mcp-config", p.MCPFile())
	}
	if fi, err := os.Stat(p.SettingsFile()); err == nil && fi.Size() > 2 {
		args = append(args, "--settings", p.SettingsFile())
	}
	if o.Exclusive {
		args = append(args, "--strict-mcp-config", "--setting-sources", "project,local")
	}
	if o.Isolated {
		env = append(env, "CLAUDE_CONFIG_DIR="+p.HomeDir())
	}
	return args, env
}

// Command builds the claude process that starts a profile; extra arguments are passed through.
func Command(p Profile, o LaunchOptions, extra ...string) (*exec.Cmd, error) {
	bin, err := exec.LookPath("claude")
	if err != nil {
		return nil, errors.New("claude is not on PATH")
	}
	args, env := Args(p, o)
	cmd := exec.Command(bin, append(args, extra...)...)
	cmd.Env = append(os.Environ(), env...)
	return cmd, nil
}

// Options are per-profile launch preferences, stored next to the card (options.json).
type Options struct {
	Exclusive bool `json:"exclusive,omitempty"`
}

func optionsFile(dir string) string { return filepath.Join(dir, "options.json") }

// SetOptions saves the launch preferences of an installed profile.
func SetOptions(name string, o Options) error {
	p, err := Get(name)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(o, "", "  ")
	return os.WriteFile(optionsFile(p.Dir), append(b, '\n'), 0o644)
}

func activeFile() string { return filepath.Join(filepath.Dir(Root()), "active-profile") }

// Active is the profile `catude claude` (and the shell integration) starts by default; "" for none.
func Active() string {
	b, err := os.ReadFile(activeFile())
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(b))
	if _, err := Get(name); err != nil {
		return ""
	}
	return name
}

// SetActive switches the default profile; "" or "none" clears it.
func SetActive(name string) error {
	if name == "" || name == "none" {
		err := os.Remove(activeFile())
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if _, err := Get(name); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(activeFile()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(activeFile(), []byte(name+"\n"), 0o644)
}
