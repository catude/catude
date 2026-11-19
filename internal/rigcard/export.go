package rigcard

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/redact"
)

// ExportOptions controls what goes into an exported card.
type ExportOptions struct {
	Owner   Owner
	Summary string
	ID      string // canonical URL where it will be published
	// KeepValues keeps non-secret environment values (a host name, a flag) as defaults. Only for profiles
	// that stay on this machine: shared cards never carry values.
	KeepValues bool
}

// Export builds a card from the local inventory. Secrets are reduced to variable names; local skills,
// agents and commands carry a sha256 so readers can verify what they later fetch from your repo.
func Export(inv inventory.Inventory, o ExportOptions) (Card, error) {
	c := Card{Schema: "https://catude.dev/schema/rigcard/0.1.json", Rigcard: Version, ID: o.ID, Summary: o.Summary,
		Updated: time.Now().UTC().Format(time.RFC3339), Harnesses: []Harness{{ID: "claude-code", Primary: true}}}
	if o.Owner != (Owner{}) {
		owner := o.Owner
		c.Owner = &owner
	}
	for _, it := range inv.Of(inventory.Marketplace) {
		src := Source{Source: it.Origin}
		if strings.Contains(it.Description, "/") && !strings.Contains(it.Description, "://") {
			src.Repo = it.Description
		} else {
			src.URL = it.Description
		}
		c.Marketplaces = append(c.Marketplaces, Marketplace{Name: it.Name, Source: src})
	}
	for _, it := range inv.Of(inventory.Plugin) {
		c.Plugins = append(c.Plugins, Plugin{ID: it.Name, Version: strings.TrimPrefix(it.Description, "v"), Enabled: it.Enabled})
	}
	for _, it := range inv.Of(inventory.Skill) {
		if it.Scope == paths.ScopePlugin || it.Scope == paths.ScopeSynced {
			continue // they come with their plugin / with claude.ai
		}
		a := Artifact{Name: it.Name, Description: it.Description, Scope: string(it.Scope), ManualOnly: !it.Invocable, Source: &Source{Type: "local"}}
		if h, err := TreeHash(filepath.Dir(it.Path)); err == nil {
			a.SHA256 = h
		}
		c.Skills = append(c.Skills, a)
	}
	for _, k := range []inventory.Kind{inventory.Agent, inventory.Command} {
		for _, it := range inv.Of(k) {
			if it.Scope == paths.ScopePlugin {
				continue
			}
			a := Artifact{Name: it.Name, Description: it.Description, Scope: string(it.Scope), Source: &Source{Type: "local"}}
			if h, err := FileHash(it.Path); err == nil {
				a.SHA256 = h
			}
			if k == inventory.Agent {
				c.Agents = append(c.Agents, a)
			} else {
				c.Commands = append(c.Commands, a)
			}
		}
	}
	for _, it := range inv.Of(inventory.Hook) {
		if it.Scope == paths.ScopePlugin {
			continue
		}
		ev, matcher := it.Name, ""
		if i := strings.Index(ev, " ["); i > 0 {
			ev, matcher = it.Name[:i], strings.Trim(it.Name[i+1:], "[]")
		}
		c.Hooks = append(c.Hooks, HookRef{Event: ev, Matcher: matcher, ExecutesCode: true, Summary: "command hook (" + string(it.Scope) + ")"})
	}
	seen := map[string]bool{}
	for _, it := range inv.Of(inventory.MCP) {
		if seen[it.Name] {
			continue // the same server copied to several Desktop configs
		}
		seen[it.Name] = true
		if it.Scope == paths.ScopeClaudeAI {
			if c.Desktop == nil {
				c.Desktop = &Desktop{}
			}
			c.Desktop.Connectors = append(c.Desktop.Connectors, it.Name)
			continue
		}
		var s struct {
			Type    string            `json:"type"`
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		}
		if err := json.Unmarshal([]byte(it.Content), &s); err != nil {
			continue
		}
		m := MCPServer{Name: it.Name, Transport: firstNonEmpty(s.Type, map[bool]string{true: "http", false: "stdio"}[s.URL != ""]), URL: redact.URL(s.URL), Command: s.Command, Args: redact.Args(s.Args)}
		for _, k := range sortedKeys(s.Env) {
			v := Variable{Name: k, IsSecret: redact.KeyIsSecret(k) || redact.LooksSecret(s.Env[k]), IsRequired: true}
			// values stay on this machine unless the caller asks (local profiles only) and the value cannot
			// be a credential: short passwords and URLs with user:password do not look like tokens
			if o.KeepValues && !v.IsSecret && !redact.IsEnvRef(s.Env[k]) && !maybeSecret(k, s.Env[k]) {
				v.Default = s.Env[k]
			} else if !v.IsSecret && maybeSecret(k, s.Env[k]) {
				v.IsSecret = true
			}
			m.Env = append(m.Env, v)
		}
		for _, k := range sortedKeys(s.Headers) {
			m.Headers = append(m.Headers, Variable{Name: k, IsSecret: true, IsRequired: true})
		}
		c.MCPServers = append(c.MCPServers, m)
	}
	if c.Desktop != nil {
		sort.Strings(c.Desktop.Connectors)
	}
	for _, it := range inv.Of(inventory.Settings) {
		if it.Scope != paths.ScopeUser {
			continue
		}
		var s struct {
			Model       string `json:"model"`
			EffortLevel string `json:"effortLevel"`
			OutputStyle string `json:"outputStyle"`
			Permissions struct {
				Deny []string `json:"deny"`
				Ask  []string `json:"ask"`
			} `json:"permissions"`
		}
		if json.Unmarshal([]byte(it.Content), &s) == nil {
			c.Settings = &Settings{Model: s.Model, EffortLevel: s.EffortLevel, OutputStyle: s.OutputStyle}
			if len(s.Permissions.Deny)+len(s.Permissions.Ask) > 0 {
				c.Settings.Permissions = &Permissions{Deny: s.Permissions.Deny, Ask: s.Permissions.Ask}
			}
		}
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return c, err
	}
	if redact.Text(string(raw)) != string(raw) {
		return c, errors.New("refusing to export: the card would contain a credential-shaped value")
	}
	return c, nil
}

var reSecretKey = regexp.MustCompile(`(?i)pass|pwd|secret|token|key|auth|cred|cookie|session|dsn|pat$|signature|private`)

// maybeSecret is deliberately broad: when in doubt a value is not exported.
func maybeSecret(key, value string) bool {
	if reSecretKey.MatchString(key) {
		return true
	}
	if i := strings.Index(value, "://"); i >= 0 && strings.Contains(value[i+3:], "@") {
		return true // scheme://user:password@host
	}
	return strings.Contains(value, "$")
}

func sortedKeys(m map[string]string) []string {
	k := make([]string, 0, len(m))
	for x := range m {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}
