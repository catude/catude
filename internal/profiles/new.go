package profiles

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/rigcard"
)

// Selection names the items of this machine that go into a new profile.
type Selection struct {
	Skills, Agents, Commands, MCP, Plugins []string
	Summary, Model                         string
}

// Empty reports whether nothing was selected.
func (s Selection) Empty() bool {
	return len(s.Skills)+len(s.Agents)+len(s.Commands)+len(s.MCP)+len(s.Plugins) == 0
}

// FromLocal builds (but does not install) a profile from items already on this machine. Files are copied,
// so later edits to the originals do not change the profile. MCP secrets become ${ENV_VAR} references.
func FromLocal(ctx context.Context, name string, inv inventory.Inventory, sel Selection) (*Staged, error) {
	full, err := rigcard.Export(inv, rigcard.ExportOptions{Summary: sel.Summary, KeepValues: true})
	if err != nil {
		return nil, err
	}
	c := rigcard.Card{Schema: full.Schema, Rigcard: full.Rigcard, Updated: full.Updated, Summary: sel.Summary, Harnesses: full.Harnesses}
	local := map[string]string{}
	var missing []string
	pick := func(kind string, k inventory.Kind, names []string) []rigcard.Artifact {
		var out []rigcard.Artifact
		for _, n := range names {
			it := find(inv, k, n)
			if it == nil {
				missing = append(missing, kind+" "+n)
				continue
			}
			a := rigcard.Artifact{Name: it.Name, Description: it.Description, ManualOnly: !it.Invocable, Source: &rigcard.Source{Type: "local"}}
			src := it.Path
			if k == inventory.Skill {
				src = filepath.Dir(it.Path)
			}
			// the hash is verified again after copying, so a file changed mid-copy is caught
			if h, err := hashOf(src, k == inventory.Skill); err == nil {
				a.SHA256 = h
			}
			local[kind+":"+a.Name] = src
			out = append(out, a)
		}
		return out
	}
	c.Skills = pick("skill", inventory.Skill, sel.Skills)
	c.Agents = pick("agent", inventory.Agent, sel.Agents)
	c.Commands = pick("command", inventory.Command, sel.Commands)
	for _, n := range sel.MCP {
		found := false
		for _, m := range full.MCPServers {
			if strings.EqualFold(m.Name, n) {
				c.MCPServers, found = append(c.MCPServers, m), true
				break
			}
		}
		if !found {
			missing = append(missing, "mcp "+n+" (claude.ai connectors are per account and cannot go in a profile)")
		}
	}
	usedMarkets := map[string]bool{}
	for _, n := range sel.Plugins {
		found := false
		for _, p := range full.Plugins {
			if strings.EqualFold(p.ID, n) || strings.EqualFold(strings.Split(p.ID, "@")[0], n) {
				p.Enabled = true
				c.Plugins, found = append(c.Plugins, p), true
				if i := strings.LastIndex(p.ID, "@"); i > 0 {
					usedMarkets[p.ID[i+1:]] = true
				}
				break
			}
		}
		if !found {
			missing = append(missing, "plugin "+n)
		}
	}
	for _, m := range full.Marketplaces {
		if usedMarkets[m.Name] {
			c.Marketplaces = append(c.Marketplaces, m)
		}
	}
	if sel.Model != "" {
		c.Settings = &rigcard.Settings{Model: sel.Model}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("not found on this machine: %s", strings.Join(missing, ", "))
	}
	return Build(ctx, name, c, BuildOptions{Local: local, Inventory: &inv})
}

func find(inv inventory.Inventory, k inventory.Kind, name string) *inventory.Item {
	items := inv.Of(k)
	for i := range items {
		if strings.EqualFold(items[i].Name, name) && items[i].Path != "" {
			return &items[i]
		}
	}
	return nil
}

// CurrentSelection is "your setup as it is now": your own skills, agents and commands (not the ones that
// come with plugins or claude.ai), your MCP servers (claude.ai connectors belong to the account) and your
// enabled plugins. Hooks are never part of a profile.
func CurrentSelection(inv inventory.Inventory) Selection {
	var s Selection
	seen := map[string]bool{}
	own := func(k inventory.Kind) []string {
		var out []string
		for _, it := range inv.Of(k) {
			key := string(k) + ":" + strings.ToLower(it.Name)
			if it.Path == "" || seen[key] || it.Scope == paths.ScopePlugin || it.Scope == paths.ScopeSynced || it.Scope == paths.ScopeClaudeAI || it.Scope == paths.ScopeManaged {
				continue
			}
			seen[key] = true
			out = append(out, it.Name)
		}
		return out
	}
	s.Skills, s.Agents, s.Commands = own(inventory.Skill), own(inventory.Agent), own(inventory.Command)
	for _, it := range inv.Of(inventory.MCP) {
		key := "mcp:" + strings.ToLower(it.Name)
		if it.Path == "" || seen[key] || it.Scope == paths.ScopeClaudeAI || it.Scope == paths.ScopeDesktop || it.Scope == paths.ScopePlugin {
			continue
		}
		seen[key] = true
		s.MCP = append(s.MCP, it.Name)
	}
	for _, it := range inv.Of(inventory.Plugin) {
		if it.Enabled {
			s.Plugins = append(s.Plugins, it.Name)
		}
	}
	return s
}
