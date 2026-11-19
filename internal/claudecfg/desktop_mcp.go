package claudecfg

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

// Copying MCP servers into Claude Desktop (claude_desktop_config.json, key mcpServers). Only local
// servers that start a program fit there; remote servers are added in the app as connectors. Servers that
// use ${VAR} references are left out: Claude Code fills those from the environment, and catude never
// writes the secret itself into a file.

// MCPChange is what copying one server would do.
type MCPChange struct {
	Name   string
	Action string // add | replace | same | keep | skip
	Reason string // for keep and skip
}

// Writes says whether the change modifies the file.
func (c MCPChange) Writes() bool { return c.Action == "add" || c.Action == "replace" }

func desktopTarget(t Target) error {
	if !strings.HasPrefix(t.ID, "desktop:") || t.ReadOnly {
		return errors.New("MCP servers are copied into a Claude Desktop configuration file")
	}
	return nil
}

// PlanDesktopMCP lists what copying the servers into a Claude Desktop config would do, without writing.
func PlanDesktopMCP(t Target, servers map[string]json.RawMessage, replace bool) ([]MCPChange, error) {
	if err := desktopTarget(t); err != nil {
		return nil, err
	}
	root, err := loadRoot(t)
	if err != nil {
		return nil, err
	}
	if root.kind != 'o' {
		return nil, errors.New("the Claude Desktop config is not a JSON object")
	}
	names := make([]string, 0, len(servers))
	for n := range servers {
		names = append(names, n)
	}
	sort.Strings(names)
	var out []MCPChange
	for _, n := range names {
		c := MCPChange{Name: n}
		srv, err := parse(servers[n])
		switch {
		case err != nil || srv.kind != 'o':
			c.Action, c.Reason = "skip", "not a valid server definition"
		case srv.get("url") != nil:
			c.Action, c.Reason = "skip", "remote server: add it in Claude Desktop under Settings, Connectors"
		case srv.get("command") == nil:
			c.Action, c.Reason = "skip", "no command to start"
		case strings.Contains(string(servers[n]), "${"):
			c.Action, c.Reason = "skip", "it uses ${VAR} references that Claude Desktop would receive as plain text; add it by hand with the value from your password manager"
		default:
			cur := root.at([]string{"mcpServers", n})
			switch {
			case cur == nil:
				c.Action = "add"
			case compact(cur) == compact(srv):
				c.Action = "same"
			case replace:
				c.Action = "replace"
			default:
				c.Action, c.Reason = "keep", "Claude Desktop already has a different server with this name (--replace overwrites it)"
			}
		}
		out = append(out, c)
	}
	return out, nil
}

// CopyToDesktopMCP writes the servers that PlanDesktopMCP marks as add or replace. The file is backed up
// first; the other keys and their order stay as they were. It returns the backup path ("" for a new file).
func CopyToDesktopMCP(t Target, servers map[string]json.RawMessage, replace bool) ([]MCPChange, string, error) {
	plan, err := PlanDesktopMCP(t, servers, replace)
	if err != nil {
		return nil, "", err
	}
	writes := false
	for _, c := range plan {
		writes = writes || c.Writes()
	}
	if !writes {
		return plan, "", nil
	}
	backup, err := write(t, func(root *node) error {
		for _, c := range plan {
			if !c.Writes() {
				continue
			}
			srv, err := parse(servers[c.Name])
			if err != nil {
				return err
			}
			if err := root.set([]string{"mcpServers", c.Name}, srv); err != nil {
				return err
			}
		}
		return nil
	})
	return plan, backup, err
}
