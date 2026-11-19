package claudecfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catude/catude/internal/paths"
)

const settings = "\xef\xbb\xbf" + `{
  "zeta": 1,
  "model": "sonnet",
  "permissions": {
    "defaultMode": "default",
    "deny": ["Read(./.env)", "Bash(rm -rf *)"]
  },
  "env": {"GITHUB_TOKEN": "ghp_` + "Zq9XZq9XZq9XZq9XZq9XZq9XZq9XZq9XZq9XZq9X" + `", "REF": "${MY_TOKEN}", "HOST": "localhost"},
  "alwaysThinkingEnabled": true,
  "hooks": {"Stop": [{"hooks": [{"type": "command", "command": "echo hi"}]}]},
  "alpha": "last"
}
`

func target(t *testing.T) Target {
	t.Helper()
	d := t.TempDir()
	t.Setenv("LOCALAPPDATA", d)
	t.Setenv("XDG_CACHE_HOME", d)
	t.Setenv("HOME", d)
	p := filepath.Join(d, "settings.json")
	if err := os.WriteFile(p, []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	return Target{ID: "code-user", App: "Claude Code", Label: "user", Path: p}
}

func find(es []Entry, key string) *Entry {
	for i := range es {
		if es[i].Key == key {
			return &es[i]
		}
	}
	return nil
}

func TestReadMasksSecretsAndMarksEditable(t *testing.T) {
	tg := target(t)
	es, err := Read(tg)
	if err != nil {
		t.Fatal(err)
	}
	if e := find(es, "env.GITHUB_TOKEN"); e == nil || !e.Secret || strings.Contains(e.Value, "ghp_") || e.Editable {
		t.Fatalf("token must be masked and not editable: %+v", e)
	}
	if e := find(es, "env.REF"); e == nil || e.Secret || e.Value != "${MY_TOKEN}" {
		t.Errorf("an env reference is not a secret: %+v", e)
	}
	if e := find(es, "env.HOST"); e == nil || !e.Editable || !e.Env || e.Category != "Environment variables" {
		t.Errorf("plain env values are editable inline: %+v", e)
	}
	if e := find(es, "model"); e == nil || !e.Editable || e.Help == "" {
		t.Errorf("model: %+v", e)
	}
	if e := find(es, "permissions.defaultMode"); e == nil || len(e.Choices) == 0 {
		t.Errorf("defaultMode must offer its documented choices: %+v", e)
	}
	if e := find(es, "permissions.deny"); e == nil || e.Type != "list" || e.Editable {
		t.Errorf("lists are shown, not edited inline: %+v", e)
	}
	for _, e := range es {
		if strings.HasPrefix(e.Key, "hooks.") && e.Editable {
			t.Errorf("hooks must never be editable inline: %s", e.Key)
		}
	}
}

func TestSetKeepsOrderBacksUpAndValidates(t *testing.T) {
	tg := target(t)
	es, _ := Read(tg)
	backup, err := Set(tg, *find(es, "model"), "opus")
	if err != nil || backup == "" {
		t.Fatalf("set model: %v (backup %q)", err, backup)
	}
	if b, _ := os.ReadFile(backup); !strings.Contains(string(b), `"sonnet"`) {
		t.Error("the backup must hold the previous content")
	}
	out, _ := os.ReadFile(tg.Path)
	s := string(out)
	if !strings.Contains(s, `"model": "opus"`) {
		t.Fatalf("not written: %s", s)
	}
	if !(strings.Index(s, `"zeta"`) < strings.Index(s, `"model"`) && strings.Index(s, `"model"`) < strings.Index(s, `"permissions"`) && strings.Index(s, `"hooks"`) < strings.Index(s, `"alpha"`)) {
		t.Fatalf("key order changed:\n%s", s)
	}
	if !strings.Contains(s, "ghp_") || !strings.Contains(s, `"Bash(rm -rf *)"`) {
		t.Fatal("untouched values must survive byte for byte")
	}
	es, _ = Read(tg)
	if _, err := Set(tg, *find(es, "permissions.defaultMode"), "bypassPermissions"); err == nil {
		t.Error("bypassPermissions must not be settable inline")
	}
	if _, err := Set(tg, *find(es, "permissions.defaultMode"), "plan"); err != nil {
		t.Error(err)
	}
	if _, err := Set(tg, *find(es, "alwaysThinkingEnabled"), "maybe"); err == nil {
		t.Error("a yes/no value must be true or false")
	}
	if _, err := Set(tg, *find(es, "model"), "sk-ant-api03-"+strings.Repeat("A", 90)); err == nil {
		t.Error("a secret-shaped value must be refused")
	}
	if _, err := Set(tg, *find(es, "permissions.deny"), "x"); err == nil {
		t.Error("non-editable entries must be refused")
	}
	// a documented setting that is not in the file yet is added at the end
	var cleanup Entry
	for _, sgg := range Suggestions(tg, es) {
		if sgg.Key == "cleanupPeriodDays" {
			cleanup = sgg
		}
	}
	if _, err := Set(tg, cleanup, "0"); err == nil {
		t.Error("cleanupPeriodDays must be at least 1")
	}
	if _, err := Set(tg, cleanup, "365"); err != nil {
		t.Fatal(err)
	}
	out, _ = os.ReadFile(tg.Path)
	if !strings.Contains(string(out), `"cleanupPeriodDays": 365`) || strings.Index(string(out), `"alpha"`) > strings.Index(string(out), `"cleanupPeriodDays"`) {
		t.Fatalf("new key must be appended:\n%s", out)
	}
}

func TestReadOnlyNewFileAndBrokenJSON(t *testing.T) {
	tg := target(t)
	ro := tg
	ro.ReadOnly = true
	es, _ := Read(ro)
	if e := find(es, "model"); e.Editable {
		t.Error("read-only targets have no editable entries")
	}
	if _, err := Set(ro, Entry{Path: []string{"model"}, Key: "model", Type: "text", Editable: true}, "opus"); err == nil {
		t.Error("read-only targets must refuse writes")
	}
	fresh := Target{ID: "code-local", App: "Claude Code", Path: filepath.Join(filepath.Dir(tg.Path), "sub", "settings.local.json")}
	if es, err := Read(fresh); err != nil || len(es) != 0 {
		t.Fatalf("a missing file is empty, not an error: %v %v", es, err)
	}
	if _, err := Set(fresh, *find(Documented(fresh, nil), "model"), "opus"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(fresh.Path); strings.TrimSpace(string(b)) != "{\n  \"model\": \"opus\"\n}" {
		t.Fatalf("new file: %q", b)
	}
	os.WriteFile(tg.Path, []byte(`{"model": `), 0o644)
	if _, err := Read(tg); err == nil {
		t.Error("broken JSON must be reported")
	}
	if _, err := Set(tg, Entry{Path: []string{"model"}, Key: "model", Type: "text", Editable: true}, "opus"); err == nil {
		t.Error("a broken file must not be overwritten")
	}
}

func TestTargetsCoverCodeAndDesktop(t *testing.T) {
	r := t.TempDir()
	env := paths.Env{GOOS: "windows", Home: r, ClaudeDir: filepath.Join(r, ".claude"), ClaudeJSON: filepath.Join(r, ".claude.json"), ProjectDir: filepath.Join(r, "proj"),
		AppData: filepath.Join(r, "AppData", "Roaming"), LocalAppData: filepath.Join(r, "AppData", "Local")}
	msix := filepath.Join(env.LocalAppData, "Packages", "Claude_abc", "LocalCache", "Roaming", "Claude")
	os.MkdirAll(msix, 0o755)
	os.WriteFile(filepath.Join(msix, "claude_desktop_config.json"), []byte(`{"mcpServers":{}}`), 0o644)
	ts := Targets(env)
	apps := map[string]int{}
	var store *Target
	for i, tg := range ts {
		apps[tg.App]++
		if strings.Contains(tg.Path, "Packages") {
			store = &ts[i]
		}
		if tg.ID == "code-state" && (!tg.State || tg.RefApp != "code-global") {
			t.Error("~/.claude.json is a state file with its documented global options")
		}
	}
	if apps["Claude Code"] < 4 || apps["Claude Desktop"] < 2 {
		t.Fatalf("targets: %v", apps)
	}
	if store == nil || !store.Exists || !strings.Contains(store.Note, "Microsoft Store") {
		t.Fatalf("the Microsoft Store copy must be listed and explained: %+v", store)
	}
}
