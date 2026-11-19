package presets

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/rigcard"
)

func TestMonogramRowsHaveEqualWidth(t *testing.T) {
	for _, in := range []string{"LT", "BC", "SW", "M", "XYZ"} {
		rows := Monogram(in)
		for _, r := range rows {
			if lipgloss.Width(r) != lipgloss.Width(rows[0]) {
				t.Fatalf("%s: uneven rows %q", in, rows)
			}
		}
	}
	if (Preset{Name: "Lauren Tan"}).Initials() != "LT" || (Preset{Name: "Jesse \"obra\" Vincent"}).Initials() != "JV" {
		t.Error("initials")
	}
}

func TestCardKeepsOnlyPinnedAndSafe(t *testing.T) {
	sha := strings.Repeat("a", 40)
	p := Preset{ID: "x", Name: "Ada Lovelace", Philosophy: "plan first",
		Skills: []Artifact{{Name: "tdd", Repo: "ada/skills", Path: "skills/tdd", Commit: sha, License: "MIT"}, {Name: "floating", Repo: "ada/skills", Path: "skills/f", License: "MIT"},
			{Name: "closed", Repo: "ada/skills", Path: "skills/c", Commit: sha, License: "No license file"}},
		Plugins: []PluginRef{{ID: "superpowers@claude-plugins-official"}, {ID: "no-marketplace"}},
		MCP: []MCP{{Name: "linear", Add: "claude mcp add --transport http linear https://mcp.linear.app/mcp"},
			{Name: "local", Add: "claude mcp add local -- npx -y something"}},
		Settings: Settings{Model: "opus"}}
	c, notes := p.Card()
	if len(c.Skills) != 1 || c.Skills[0].Source.Commit != sha {
		t.Fatalf("only pinned skills: %+v", c.Skills)
	}
	if len(c.Plugins) != 0 || len(c.MCPServers) != 1 || c.MCPServers[0].URL != "https://mcp.linear.app/mcp" {
		t.Fatalf("plugins must not be enabled next to their own files; plain remote MCP only: %+v %+v", c.Plugins, c.MCPServers)
	}
	if c.Settings != nil {
		t.Error("model and effort must never be applied by a preset")
	}
	if len(notes) != 5 {
		t.Errorf("unpinned, unlicensed, plugin, local MCP and model must become notes: %v", notes)
	}
	only := Preset{ID: "y", Name: "B", Plugins: []PluginRef{{ID: "ralph-loop@claude-plugins-official"}}}
	if c2, _ := only.Card(); len(c2.Plugins) != 1 {
		t.Error("a preset without files enables its plugin")
	}
	if rigcard.Rejected(rigcard.Validate(c, nil)) {
		t.Fatalf("preset card must validate: %v", rigcard.Validate(c, nil))
	}
}

// Every embedded preset must validate and cite sources.
func TestEmbeddedPresets(t *testing.T) {
	ps, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		if p.ID == "" || p.Name == "" || len(p.Sources) == 0 {
			t.Errorf("preset %q needs an id, a name and sources", p.ID)
		}
		for _, u := range p.UsersWant {
			if !strings.HasPrefix(u.URL, "https://") || u.Point == "" {
				t.Errorf("%s: user request without an https link: %+v", p.ID, u)
			}
		}
		for _, s := range p.Sources {
			if !strings.HasPrefix(s.URL, "https://") {
				t.Errorf("%s: source %q is not https", p.ID, s.URL)
			}
		}
		c, _ := p.Card()
		if ps := rigcard.Validate(c, nil); rigcard.Rejected(ps) {
			t.Errorf("%s: %v", p.ID, ps)
		}
	}
}
