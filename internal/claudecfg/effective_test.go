package claudecfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The documented examples: a team value beats yours, your local file beats the team, lists combine,
// fallbackModel never merges, modelPicker is ignored in project files, secrets stay masked.
func TestEffectiveFollowsTheDocumentedPrecedence(t *testing.T) {
	d := t.TempDir()
	write := func(name, s string) string {
		p := filepath.Join(d, name)
		os.WriteFile(p, []byte(s), 0o600)
		return p
	}
	targets := []Target{
		{ID: "code-user", Path: write("user.json", `{"spinnerTipsEnabled": false, "model": "opus", "permissions": {"allow": ["Bash(npm test:*)"]},
			"fallbackModel": ["sonnet"], "modelPicker": {"replaceBuiltInOptions": true}, "env": {"GITHUB_TOKEN": "ghp_`+strings.Repeat("A", 36)+`", "FOO": "1"}}`)},
		{ID: "code-project", Path: write("project.json", `{"spinnerTipsEnabled": true, "permissions": {"allow": ["Read(./src/**)"], "deny": ["Read(./.env)"]},
			"fallbackModel": ["haiku"], "modelPicker": {"replaceBuiltInOptions": false}}`)},
		{ID: "code-local", Path: write("local.json", `{"spinnerTipsEnabled": false, "permissions": {"allow": ["Bash(npm test:*)", "Bash(go test:*)"]}}`)},
		{ID: "desktop:x", Path: write("desktop.json", `{"mcpServers": {}}`)}, // not part of the stack
	}
	es := Effective(targets)
	get := func(k string) *Entry {
		for i := range es {
			if es[i].Key == k {
				return &es[i]
			}
		}
		t.Fatalf("%s missing from %+v", k, es)
		return nil
	}
	if e := get("spinnerTipsEnabled"); e.Value != "false" || e.From != "project (only you)" || len(e.Overrides) != 2 {
		t.Errorf("local beats shared project and user: %+v", e)
	}
	if e := get("model"); e.Value != "opus" || e.From != "your settings" || len(e.Overrides) != 0 {
		t.Errorf("a key set once comes from its file: %+v", e)
	}
	allow := get("permissions.allow")
	if !allow.Merged || allow.Value != "[Bash(npm test:*), Bash(go test:*), Read(./src/**)]" {
		t.Errorf("lists combine without duplicates: %+v", allow)
	}
	if e := get("fallbackModel"); e.Merged || e.Value != `["haiku"]` || e.From != "project (shared)" {
		t.Errorf("fallbackModel comes whole from the highest file: %+v", e)
	}
	if e := get("modelPicker"); e.From != "your settings" {
		t.Errorf("modelPicker is read only from policy and user settings: %+v", e)
	}
	if e := get("env.GITHUB_TOKEN"); strings.Contains(e.Value, "ghp_") {
		t.Errorf("secrets stay masked: %+v", e)
	}
	if e := get("env.FOO"); e.Value != "1" || e.Category != "Environment variables" {
		t.Errorf("env keys are resolved one by one: %+v", e)
	}
	for _, e := range es {
		if strings.HasPrefix(e.Key, "mcpServers") {
			t.Error("Claude Desktop is not part of Claude Code's settings stack")
		}
	}
}
