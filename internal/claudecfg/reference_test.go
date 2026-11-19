package claudecfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReferenceCoversEveryApp(t *testing.T) {
	for app, min := range map[string]int{"code-settings": 250, "code-env": 400, "code-global": 10, "desktop": 1, "desktop-policy": 100} {
		if n := len(Refs(app)); n < min {
			t.Errorf("%s: %d documented settings, want at least %d", app, n, min)
		}
	}
	r := lookup(Target{ID: "code-user"}, "effortLevel")
	if r == nil || r.Type != "enum" || len(r.Values) == 0 || r.Desc == "" {
		t.Errorf("effortLevel: %+v", r)
	}
	if lookup(Target{ID: "desktop:x"}, "mcpServers.github.command") == nil {
		t.Error("placeholder keys (mcpServers.<name>.command) must match real names")
	}
}

func TestDocumentedListsEverythingNotSet(t *testing.T) {
	tg := target(t)
	es, _ := Read(tg)
	docs := Documented(tg, es)
	if len(docs) < 600 {
		t.Fatalf("every documented setting and variable should be listed: %d", len(docs))
	}
	if find(docs, "model") != nil {
		t.Error("model is set in the file: not listed again")
	}
	e := find(docs, "env.DISABLE_TELEMETRY")
	if e == nil || !e.Env || e.Type != "yes/no" || !e.Editable || e.Value != NotSet {
		t.Fatalf("env var: %+v", e)
	}
	if k := find(docs, "env.ANTHROPIC_API_KEY"); k == nil || k.Editable {
		t.Errorf("credential variables are listed but never typed in here: %+v", k)
	}
	for _, d := range docs {
		if strings.EqualFold(d.Scope, "managed") {
			t.Errorf("%s is only read from managed settings: not offered in the user file", d.Key)
		}
	}
	// a yes/no variable is stored as "1" and turned off by removing it
	if _, err := Set(tg, *e, "true"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(tg.Path)
	if !strings.Contains(string(b), `"DISABLE_TELEMETRY": "1"`) {
		t.Fatalf("env bool:\n%s", b)
	}
	es, _ = Read(tg)
	on := find(es, "env.DISABLE_TELEMETRY")
	if on == nil || on.Type != "yes/no" || !on.Set {
		t.Fatalf("read back: %+v", on)
	}
	if _, err := Set(tg, *on, "false"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(tg.Path); strings.Contains(string(b), "DISABLE_TELEMETRY") {
		t.Error("off removes the variable (some treat any value as on)")
	}
}

func TestUnsetGoesBackToDefault(t *testing.T) {
	tg := target(t)
	es, _ := Read(tg)
	if _, err := Unset(tg, *find(es, "alwaysThinkingEnabled")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(tg.Path)
	if strings.Contains(string(b), "alwaysThinkingEnabled") || !strings.Contains(string(b), `"alpha"`) {
		t.Fatalf("unset:\n%s", b)
	}
	if _, err := Unset(tg, Entry{Key: "x", Path: []string{"x"}}); err == nil {
		t.Error("a setting that is not in the file cannot be removed")
	}
}

func TestStateFileOnlyWhileClosed(t *testing.T) {
	d := t.TempDir()
	t.Setenv("LOCALAPPDATA", d)
	t.Setenv("XDG_CACHE_HOME", d)
	t.Setenv("HOME", d)
	p := filepath.Join(d, ".claude.json")
	os.WriteFile(p, []byte(`{"numStartups": 3, "copyOnSelect": true, "oauthAccount": {"emailAddress": "x@example.com"}, "projects": {"/a": {}}}`), 0o600)
	tg := Target{ID: "code-state", Path: p, State: true, RefApp: "code-global"}
	es, err := Read(tg)
	if err != nil {
		t.Fatal(err)
	}
	if find(es, "numStartups") != nil || find(es, "oauthAccount.emailAddress") != nil || find(es, "projects") != nil {
		t.Errorf("only documented options are listed from ~/.claude.json: %+v", es)
	}
	c := find(es, "copyOnSelect")
	if c == nil || !c.Editable {
		t.Fatalf("copyOnSelect: %+v", c)
	}
	old := CodeRunning
	defer func() { CodeRunning = old }()
	CodeRunning = func() int { return 2 }
	if _, err := Set(tg, *c, "false"); err == nil || !strings.Contains(err.Error(), "close Claude Code") {
		t.Errorf("must refuse while Claude Code runs: %v", err)
	}
	CodeRunning = func() int { return 0 }
	if _, err := Set(tg, *c, "false"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), `"copyOnSelect": false`) || !strings.Contains(string(b), "x@example.com") {
		t.Errorf("everything else must survive:\n%s", b)
	}
}

func TestDesktopPolicyIsAReadOnlyReference(t *testing.T) {
	es, _ := Read(Target{ID: "desktop-policy", ReadOnly: true, RefApp: "desktop-policy"})
	if len(es) < 100 {
		t.Fatalf("policy keys: %d", len(es))
	}
	for _, e := range es {
		if e.Editable {
			t.Fatalf("policies are never editable here: %s", e.Key)
		}
	}
}
