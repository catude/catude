package claudecfg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every preset is sourced and quoted, and uses only documented settings with values of the right type.
func TestPresetsAreDocumentedAndSourced(t *testing.T) {
	ps, err := Presets()
	if err != nil || len(ps) < 20 {
		t.Fatalf("presets: %d, %v", len(ps), err)
	}
	ids := map[string]bool{}
	for _, p := range ps {
		if ids[p.ID] {
			t.Errorf("duplicate preset %s", p.ID)
		}
		ids[p.ID] = true
		if p.Name == "" || p.Summary == "" || len(p.Tags) == 0 || p.By == "" {
			t.Errorf("%s: name, summary, tags and author are required", p.ID)
		}
		if len(p.Sources) == 0 {
			t.Errorf("%s: no source", p.ID)
		}
		for _, s := range p.Sources {
			if !strings.HasPrefix(s.URL, "https://") || len(s.Quote) < 15 {
				t.Errorf("%s: a source needs an https URL and the exact quote: %+v", p.ID, s)
			}
		}
		if p.Risk != "low" && p.RiskReason == "" {
			t.Errorf("%s: risk %s without a reason", p.ID, p.Risk)
		}
		if strings.Contains(p.Summary+p.Name+p.Note, "—") {
			t.Errorf("%s: plain punctuation please (no em dash)", p.ID)
		}
		var v map[string]any
		if err := json.Unmarshal(p.Settings, &v); err != nil {
			t.Fatalf("%s: settings: %v", p.ID, err)
		}
		tg := Target{ID: "code-user", RefApp: "code-settings"}
		var check func(prefix string, m map[string]any)
		check = func(prefix string, m map[string]any) {
			for k, val := range m {
				key := k
				if prefix != "" {
					key = prefix + "." + k
				}
				if prefix == "env" {
					if r := lookup(tg, key); r == nil {
						t.Errorf("%s: env var %s is not documented", p.ID, k)
					}
					if _, ok := val.(string); !ok {
						t.Errorf("%s: env values are text: %s", p.ID, k)
					}
					continue
				}
				if sub, ok := val.(map[string]any); ok && (key == "env" || hasChildren(tg, key)) {
					check(key, sub)
					continue
				}
				if lookup(tg, key) == nil {
					t.Errorf("%s: %s is not a documented setting", p.ID, key)
				}
			}
		}
		check("", v)
	}
}

// Applying a preset keeps the file's own values and list items, adds the new ones, and plans honestly.
func TestApplyPresetMerges(t *testing.T) {
	d := t.TempDir()
	t.Setenv("LOCALAPPDATA", d)
	t.Setenv("XDG_CACHE_HOME", d)
	t.Setenv("HOME", d)
	p := filepath.Join(d, "settings.json")
	os.WriteFile(p, []byte(`{"model": "opus", "permissions": {"deny": ["Bash(rm -rf:*)"]}, "env": {"FOO_BAR": "1"}}`), 0o644)
	tg := Target{ID: "code-project", RefApp: "code-settings", Path: p}
	var preset Preset
	ps, _ := Presets()
	for _, x := range ps {
		if x.ID == "protect-secrets-project" {
			preset = x
		}
	}
	if preset.ID == "" {
		t.Fatal("protect-secrets-project missing")
	}
	plan, err := Plan(tg, preset)
	if err != nil || len(plan) != 1 || plan[0].Key != "permissions.deny" || !strings.Contains(plan[0].After, "Bash(rm -rf:*)") || !strings.Contains(plan[0].After, "Read(./.env)") {
		t.Fatalf("plan: %+v %v", plan, err)
	}
	if _, err := ApplyPreset(tg, preset); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	s := string(b)
	for _, want := range []string{`"model": "opus"`, `"Bash(rm -rf:*)"`, `"Read(./.env)"`, `"Read(./secrets/**)"`, `"FOO_BAR": "1"`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s after apply:\n%s", want, s)
		}
	}
	// a second time changes nothing
	plan, _ = Plan(tg, preset)
	if len(plan) != 1 || !plan[0].Same {
		t.Errorf("already applied: %+v", plan)
	}
	if _, err := ApplyPreset(Target{ID: "code-state", State: true, RefApp: "code-global", Path: p}, preset); err == nil {
		t.Error("presets never go into ~/.claude.json")
	}
}
