package claudecfg

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/catude/catude/internal/paths"
)

// Switching MCP servers and skills off for one project, with the mechanisms Anthropic documents:
//
//   - a server from the project's .mcp.json: disabledMcpjsonServers in .claude/settings.local.json
//     (settings reference, "disabledMcpjsonServers");
//   - any other server (yours, a plugin's, the organisation's, a claude.ai connector): disabledMcpServers
//     under the project in ~/.claude.json, what the /mcp panel writes (MCP docs, "Disable a server
//     without removing it"). Claude Code rewrites that file while it runs, so it is changed only while
//     Claude Code is closed;
//   - a skill: skillOverrides in .claude/settings.local.json ("off", "name-only", "user-invocable-only";
//     settings reference, "skillOverrides"). Plugin skills are managed through /plugin instead.
//
// Only you see these changes: settings.local.json is git-ignored and ~/.claude.json is yours.

// Off is what is switched off in the current project.
type Off struct {
	MCP      map[string]bool   // disabledMcpServers (~/.claude.json)
	MCPJSON  map[string]bool   // disabledMcpjsonServers (settings.local.json)
	Skills   map[string]string // skillOverrides (settings.local.json)
	Project  string
	Readable bool // a project is open
}

func localTarget(env paths.Env) Target {
	return Target{ID: "code-local", App: "Claude Code", Label: "This project (only you)", Path: filepath.Join(env.ProjectDir, ".claude", "settings.local.json"), RefApp: "code-settings"}
}

func stateTarget(env paths.Env) Target {
	return Target{ID: "code-state", App: "Claude Code", Path: env.ClaudeJSON, State: true, RefApp: "code-global"}
}

// ProjectOff reads what is switched off for the project of env.
func ProjectOff(env paths.Env) Off {
	o := Off{MCP: map[string]bool{}, MCPJSON: map[string]bool{}, Skills: map[string]string{}, Project: env.ProjectDir}
	if env.ProjectDir == "" {
		return o
	}
	o.Readable = true
	if root, err := loadRoot(localTarget(env)); err == nil {
		for _, v := range strings1(root.get("disabledMcpjsonServers")) {
			o.MCPJSON[v] = true
		}
		if so := root.get("skillOverrides"); so != nil && so.kind == 'o' {
			for i, k := range so.keys {
				if so.vals[i].kind == 's' {
					o.Skills[k] = so.vals[i].str
				}
			}
		}
	}
	if root, err := loadRoot(stateTarget(env)); err == nil {
		if key := projectKey(root, env.ProjectDir); key != "" {
			for _, v := range strings1(root.at([]string{"projects", key, "disabledMcpServers"})) {
				o.MCP[v] = true
			}
		}
	}
	return o
}

// SetMCPOff switches a server off (or back on) for the project. fromMCPJSON says the server is defined
// in the project's .mcp.json. It returns the backup of the file it changed.
func SetMCPOff(env paths.Env, name string, fromMCPJSON, off bool) (string, error) {
	if env.ProjectDir == "" {
		return "", errors.New("open catude in a project folder to switch servers off for it")
	}
	if fromMCPJSON {
		return write(localTarget(env), func(root *node) error {
			return setInList(root, []string{"disabledMcpjsonServers"}, name, off)
		})
	}
	t := stateTarget(env)
	if err := stateClosed(t); err != nil {
		return "", err
	}
	return write(t, func(root *node) error {
		key := projectKey(root, env.ProjectDir)
		if key == "" {
			key = filepath.Clean(env.ProjectDir)
		}
		return setInList(root, []string{"projects", key, "disabledMcpServers"}, name, off)
	})
}

// SkillValues are the documented values of skillOverrides, in the order a key cycles through them.
var SkillValues = []string{"", "name-only", "user-invocable-only", "off"}

// SetSkillOverride sets how a skill is shown to Claude in this project ("" removes the override).
func SetSkillOverride(env paths.Env, name, value string) (string, error) {
	if env.ProjectDir == "" {
		return "", errors.New("open catude in a project folder to change skills for it")
	}
	ok := false
	for _, v := range SkillValues {
		ok = ok || v == value
	}
	if !ok {
		return "", errors.New("skillOverrides takes on, name-only, user-invocable-only or off")
	}
	return write(localTarget(env), func(root *node) error {
		if value == "" {
			root.del([]string{"skillOverrides", name})
			if so := root.get("skillOverrides"); so != nil && so.kind == 'o' && len(so.keys) == 0 {
				root.del([]string{"skillOverrides"})
			}
			return nil
		}
		v, _ := parse([]byte(`"` + value + `"`))
		return root.set([]string{"skillOverrides", name}, v)
	})
}

// projectKey finds the key of dir in ~/.claude.json's projects (paths are compared as the OS does).
func projectKey(root *node, dir string) string {
	ps := root.get("projects")
	if ps == nil || ps.kind != 'o' {
		return ""
	}
	want := filepath.Clean(dir)
	for _, k := range ps.keys {
		if c := filepath.Clean(k); c == want || runtime.GOOS == "windows" && strings.EqualFold(c, want) {
			return k
		}
	}
	return ""
}

func strings1(n *node) []string {
	if n == nil || n.kind != 'a' {
		return nil
	}
	var out []string
	for _, v := range n.vals {
		if v.kind == 's' {
			out = append(out, v.str)
		}
	}
	return out
}

// setInList adds name to (or removes it from) the string list at path, creating it when needed.
func setInList(root *node, path []string, name string, add bool) error {
	cur := strings1(root.at(path))
	var next []string
	for _, v := range cur {
		if v != name {
			next = append(next, v)
		}
	}
	if add {
		next = append(next, name)
	}
	if len(next) == 0 {
		root.del(path)
		return nil
	}
	b, _ := json.Marshal(next)
	v, err := parse(b)
	if err != nil {
		return err
	}
	return root.set(path, v)
}
