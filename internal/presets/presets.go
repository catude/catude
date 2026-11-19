// Package presets ships ready-made work profiles based on the publicly documented agentic coding setups
// of well-known practitioners. Each preset cites its sources and is labelled "based on public material";
// catude is not affiliated with or endorsed by the people it credits. Skills, agents and commands are
// pinned to a commit and downloaded only when the user loads a preset (with the usual hash checks and audit).
package presets

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/catude/catude/internal/rigcard"
)

//go:embed presets.json
var raw []byte

type Source struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Date  string `json:"date"`
}

type MCP struct {
	Name string `json:"name"`
	Add  string `json:"add"`
	URL  string `json:"url"`
}

type Repo struct {
	Name    string `json:"name"`
	Repo    string `json:"repo"`
	Commit  string `json:"commit"`
	License string `json:"license"`
}

type PluginRef struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Artifact struct {
	Name        string `json:"name"`
	Repo        string `json:"repo"`
	Path        string `json:"path"`
	Commit      string `json:"commit"`
	License     string `json:"license"`
	Description string `json:"description"`
}

type Settings struct {
	Model       string `json:"model"`
	EffortLevel string `json:"effortLevel"`
	Notes       string `json:"notes"`
}

// Ask is something users of a setup ask for, with the link where they ask.
type Ask struct {
	Point string `json:"point"`
	URL   string `json:"url"`
}

// Preset is one creator's setup as published by them.
type Preset struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Handle       string      `json:"handle"`
	Role         string      `json:"role"`
	Sources      []Source    `json:"sources"`
	Philosophy   string      `json:"philosophy"`
	Process      []string    `json:"process"`
	Tools        []string    `json:"tools"`
	MCP          []MCP       `json:"mcp"`
	Marketplaces []Repo      `json:"marketplaces"`
	Plugins      []PluginRef `json:"plugins"`
	Skills       []Artifact  `json:"skills"`
	Agents       []Artifact  `json:"agents"`
	Commands     []Artifact  `json:"commands"`
	Settings     Settings    `json:"settings"`
	Unverified   []string    `json:"unverified"`
	UsersWant    []Ask       `json:"users_want"`
	FactChecked  string      `json:"factchecked"`     // date of the last adversarial check against the sources
	ASCII        string      `json:"ascii,omitempty"` // only if the person provided it
}

// Load returns every embedded preset, sorted by name.
func Load() ([]Preset, error) {
	var ps []Preset
	if err := json.Unmarshal(raw, &ps); err != nil {
		return nil, fmt.Errorf("embedded presets are invalid: %w", err)
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].Name < ps[j].Name })
	return ps, nil
}

// Get finds a preset by id.
func Get(id string) (Preset, error) {
	ps, err := Load()
	if err != nil {
		return Preset{}, err
	}
	for _, p := range ps {
		if p.ID == id {
			return p, nil
		}
	}
	return Preset{}, fmt.Errorf("no preset %q (see catude presets)", id)
}

var reSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

// OpenLicense reports whether an artifact may be downloaded: only clearly open licenses qualify.
// "No license" means all rights reserved, so such files are linked, never copied.
func OpenLicense(l string) bool {
	l = strings.ToLower(strings.TrimSpace(l))
	for _, ok := range []string{"mit", "apache-2.0", "apache 2.0", "bsd", "isc", "cc0", "unlicense", "mpl-2.0"} {
		if strings.HasPrefix(l, ok) {
			return true
		}
	}
	return false
}

// Installable reports whether catude can download the artifact: pinned to a commit and openly licensed.
func (a Artifact) Installable() (bool, string) {
	switch {
	case !reSHA.MatchString(a.Commit) || a.Repo == "" || a.Path == "":
		return false, "not pinned to a commit"
	case !OpenLicense(a.License):
		return false, "no open license: see the author's repository"
	}
	return true, ""
}

// Card turns a preset into a Rigcard that profiles.Build can install.
//
//   - Skills, agents and commands are included only when pinned to a commit and openly licensed.
//   - Plugins are enabled in the profile only when the preset brings no files of its own (otherwise the
//     same skills would load twice); installing the full plugin is always offered as a note.
//   - MCP servers are included only when their documented command is a plain remote "claude mcp add".
//   - Model and effort are NOT applied: they change cost, so they are shown as the author's recommendation.
//   - Hooks are never installed.
func (p Preset) Card() (rigcard.Card, []string) {
	c := rigcard.Card{Rigcard: rigcard.Version, ID: "preset:" + p.ID, Summary: p.Philosophy,
		Owner: &rigcard.Owner{Name: p.Name, Handle: p.Handle}}
	var notes []string
	art := func(as []Artifact, kind string) []rigcard.Artifact {
		var out []rigcard.Artifact
		for _, a := range as {
			if ok, why := a.Installable(); !ok {
				notes = append(notes, kind+" "+a.Name+" left out ("+why+"): https://github.com/"+a.Repo)
				continue
			}
			out = append(out, rigcard.Artifact{Name: a.Name, Description: a.Description,
				Source: &rigcard.Source{Type: "github", Repo: a.Repo, Path: a.Path, Commit: a.Commit}})
		}
		return out
	}
	c.Skills, c.Agents, c.Commands = art(p.Skills, "skill"), art(p.Agents, "agent"), art(p.Commands, "command")
	files := len(c.Skills)+len(c.Agents)+len(c.Commands) > 0
	for _, pl := range p.Plugins {
		if !strings.Contains(pl.ID, "@") {
			continue
		}
		if files {
			notes = append(notes, "the full plugin (with its hooks) is optional: claude plugin install "+pl.ID)
			continue
		}
		c.Plugins = append(c.Plugins, rigcard.Plugin{ID: pl.ID, Enabled: true})
	}
	if !files {
		for _, m := range p.Marketplaces {
			if m.Repo == "" {
				continue
			}
			src := rigcard.Source{Source: "github", Repo: m.Repo}
			if reSHA.MatchString(m.Commit) {
				src.Ref = m.Commit
			}
			c.Marketplaces = append(c.Marketplaces, rigcard.Marketplace{Name: firstNonEmpty(m.Name, m.Repo), Source: src})
		}
	}
	for _, m := range p.MCP {
		if s, ok := remoteMCP(m.Add); ok {
			c.MCPServers = append(c.MCPServers, s)
		} else if m.Add != "" {
			notes = append(notes, "MCP "+m.Name+": run it yourself if you want it: "+m.Add)
		} else if m.Name != "" {
			notes = append(notes, "MCP "+m.Name+" is mentioned without an install command: "+firstNonEmpty(m.URL, "see the sources"))
		}
	}
	if s := strings.TrimSpace(p.Settings.Model + " " + p.Settings.EffortLevel); s != "" {
		notes = append(notes, "recommended model/effort ("+s+") is not applied automatically: set it yourself if you want it")
	}
	return c, notes
}

var reRemoteAdd = regexp.MustCompile(`^claude mcp add (?:--transport (http|sse) )?(?:--scope \w+ )?([\w.-]+) (https://\S+)$`)

func remoteMCP(add string) (rigcard.MCPServer, bool) {
	m := reRemoteAdd.FindStringSubmatch(strings.TrimSpace(add))
	if m == nil {
		return rigcard.MCPServer{}, false
	}
	return rigcard.MCPServer{Name: m[2], Transport: firstNonEmpty(m[1], "http"), URL: m[3]}, true
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// Initials gives the two letters used for the monogram.
func (p Preset) Initials() string {
	var out []rune
	for _, w := range strings.Fields(p.Name) {
		r := []rune(strings.ToUpper(w))
		if len(r) > 0 && r[0] >= 'A' && r[0] <= 'Z' {
			out = append(out, r[0])
		}
	}
	if len(out) > 2 {
		out = []rune{out[0], out[len(out)-1]}
	}
	return string(out)
}
