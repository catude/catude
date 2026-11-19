// Package catalog is catude's curated, offline list of well-known Claude ecosystem resources:
// marketplaces, plugins, skill collections, MCP servers, tools and sourced best practices.
// It is embedded in the binary (no network) and only ever proposes official `claude` commands.
package catalog

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

//go:embed catalog.json
var raw []byte

type Marketplace struct {
	Name        string   `json:"name"`
	Repo        string   `json:"repo"`
	Add         string   `json:"add"`
	Stars       *int     `json:"stars"`
	License     string   `json:"license"`
	Description string   `json:"description"`
	Official    bool     `json:"official"`
	URL         string   `json:"url"`
	Tags        []string `json:"tags"`
	Pushed      string   `json:"pushed,omitempty"`   // last push to the repository (scripts/refresh-catalog.mjs)
	Archived    bool     `json:"archived,omitempty"` // the repository is archived
}

type Plugin struct {
	Name        string   `json:"name"`
	Marketplace string   `json:"marketplace"`
	Install     string   `json:"install"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	URL         string   `json:"url"`
	Tags        []string `json:"tags"`
	Stars       *int     `json:"stars"`
	License     string   `json:"license"`
}

type Skill struct {
	Name        string   `json:"name"`
	Repo        string   `json:"repo"`
	Path        string   `json:"path"`
	Stars       *int     `json:"stars"`
	License     string   `json:"license"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	URL         string   `json:"url"`
	Tags        []string `json:"tags"`
	Install     string   `json:"install"`            // official plugin command when the skill ships as a plugin
	Pushed      string   `json:"pushed,omitempty"`   // last push to the repository (scripts/refresh-catalog.mjs)
	Archived    bool     `json:"archived,omitempty"` // the repository is archived
}

type MCP struct {
	Name        string   `json:"name"`
	Add         string   `json:"add"`
	Description string   `json:"description"`
	Official    bool     `json:"official"`
	URL         string   `json:"url"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

type Tool struct {
	Name        string `json:"name"`
	Repo        string `json:"repo"`
	Stars       *int   `json:"stars"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Pushed      string `json:"pushed,omitempty"`
	Archived    bool   `json:"archived,omitempty"`
}

type Practice struct {
	Rule   string `json:"rule"`
	Why    string `json:"why"`
	Source string `json:"source"`
}

// Catalog is the whole embedded list.
type Catalog struct {
	Generated    string        `json:"generated"`
	Tags         []TagInfo     `json:"tags"`
	Marketplaces []Marketplace `json:"marketplaces"`
	Skills       []Skill       `json:"skills"`
	Plugins      []Plugin      `json:"plugins"`
	MCP          []MCP         `json:"mcp"`
	Tools        []Tool        `json:"tools"`
	Practices    []Practice    `json:"practices"`
}

// Load parses the embedded catalog.
func Load() (Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, fmt.Errorf("embedded catalog is invalid: %w", err)
	}
	return c, nil
}

// Entry is a flat, displayable catalog row.
type Entry struct {
	Section     string // marketplace | plugin | skill | mcp | tool | practice
	Name        string
	Description string
	URL         string
	Command     string // official command, "" if none
	Stars       int
	Official    bool
	Category    string
	Key         string   // id used to check if it is already installed
	Tags        []string // what it is for: frontend, testing, security…
	License     string
	Pushed      string // last push to its repository, when known
	Archived    bool
}

// Quiet says the repository is archived or had no push in the year before the catalog was generated.
func (e Entry) Quiet(generated string) bool {
	if e.Archived {
		return true
	}
	pushed, err1 := time.Parse(time.DateOnly, e.Pushed)
	gen, err2 := time.Parse(time.DateOnly, generated)
	return err1 == nil && err2 == nil && pushed.Before(gen.AddDate(-1, 0, 0))
}

// TagInfo is one tag of the controlled list.
type TagInfo struct {
	Tag     string `json:"tag"`
	Meaning string `json:"meaning"`
}

// Entries flattens the catalog in display order.
func (c Catalog) Entries() []Entry {
	var out []Entry
	star := func(p *int) int {
		if p == nil {
			return 0
		}
		return *p
	}
	for _, m := range c.Marketplaces {
		out = append(out, Entry{Section: "marketplace", Name: m.Name, Description: m.Description, URL: m.URL, Command: m.Add, Stars: star(m.Stars), Official: m.Official, Key: m.Name, Tags: m.Tags, License: m.License, Pushed: m.Pushed, Archived: m.Archived})
	}
	for _, p := range c.Plugins {
		out = append(out, Entry{Section: "plugin", Name: p.Name + "@" + p.Marketplace, Description: p.Description, URL: p.URL, Command: p.Install, Stars: star(p.Stars), Official: strings.HasPrefix(p.Marketplace, "claude-plugins"), Category: p.Category, Key: p.Name + "@" + p.Marketplace, Tags: p.Tags, License: p.License})
	}
	for _, m := range c.MCP {
		out = append(out, Entry{Section: "mcp", Name: m.Name, Description: m.Description, URL: m.URL, Command: m.Add, Official: m.Official, Category: m.Category, Key: m.Name, Tags: m.Tags})
	}
	for _, s := range c.Skills {
		out = append(out, Entry{Section: "skill", Name: s.Name, Description: s.Description, URL: s.URL, Command: s.Install, Stars: star(s.Stars), Official: strings.HasPrefix(s.Repo, "anthropics/"), Category: s.Category, Key: s.Name, Tags: s.Tags, License: s.License, Pushed: s.Pushed, Archived: s.Archived})
	}
	for _, t := range c.Tools {
		out = append(out, Entry{Section: "tool", Name: t.Name, Description: t.Description, URL: t.URL, Stars: star(t.Stars), Category: t.Kind, Key: t.Name, Pushed: t.Pushed, Archived: t.Archived})
	}
	for _, p := range c.Practices {
		out = append(out, Entry{Section: "practice", Name: p.Rule, Description: p.Why, URL: p.Source, Official: strings.Contains(p.Source, "anthropic.com") || strings.Contains(p.Source, "claude.com")})
	}
	return out
}

var (
	allowedPrefixes = []string{"claude plugin marketplace add ", "claude plugin install ", "claude mcp add "}
	placeholder     = regexp.MustCompile(`(?i)YOUR_|<[^>]+>|\$\{[^}]+\}|\bxxx+\b|REPLACE|CHANGE_?ME`)
	shellMeta       = regexp.MustCompile("[;&|`$<>(){}]")
)

// ErrNeedsInput means the command has a placeholder (usually a secret) the user must fill in themselves.
var ErrNeedsInput = errors.New("the command contains a placeholder (usually a secret): run it yourself after filling it in")

// Args validates an entry's command and returns argv for exec (no shell). Only official `claude plugin`
// and `claude mcp add` commands are allowed, without shell metacharacters or placeholders.
func Args(cmd string) ([]string, error) {
	cmd = strings.TrimSpace(cmd)
	ok := false
	for _, p := range allowedPrefixes {
		if strings.HasPrefix(cmd, p) {
			ok = true
		}
	}
	if !ok {
		return nil, fmt.Errorf("only official claude plugin/mcp commands can be run from catude")
	}
	if placeholder.MatchString(cmd) {
		return nil, ErrNeedsInput
	}
	args, err := split(cmd)
	if err != nil {
		return nil, err
	}
	for _, a := range args {
		if shellMeta.MatchString(a) {
			return nil, fmt.Errorf("refusing argument with shell metacharacters: %q", a)
		}
	}
	return args, nil
}

// split is a minimal shell-like splitter supporting double quotes (no expansion).
func split(s string) ([]string, error) {
	var out []string
	var cur strings.Builder
	inQ, has := false, false
	for _, r := range s {
		switch {
		case r == '"':
			inQ, has = !inQ, true
		case r == ' ' && !inQ:
			if has {
				out = append(out, cur.String())
				cur.Reset()
				has = false
			}
		default:
			cur.WriteRune(r)
			has = true
		}
	}
	if inQ {
		return nil, errors.New("unbalanced quotes")
	}
	if has {
		out = append(out, cur.String())
	}
	return out, nil
}
