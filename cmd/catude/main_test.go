package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolated points HOME/USERPROFILE/APPDATA at an empty folder so tests never read the real machine.
func isolated(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	for _, k := range []string{"HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME"} {
		t.Setenv(k, d)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CATUDE_NO_WSL", "1") // under WSL catude also reads the Windows side: not in tests
	t.Setenv("PATH", "")
	return d
}

func call(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestUnknownCommandAndHelp(t *testing.T) {
	isolated(t)
	if c, _, e := call("nope"); c != 2 || !strings.Contains(e, "unknown command") {
		t.Fatalf("%d %s", c, e)
	}
	if c, o, _ := call("help"); c != 0 || !strings.Contains(o, "rig export") {
		t.Fatalf("help: %d", c)
	}
}

func TestDiscoverFilters(t *testing.T) {
	isolated(t)
	c, o, _ := call("discover", "plugins", "security")
	if c != 0 || !strings.Contains(o, "claude plugin install") || strings.Contains(o, " marketplace ") {
		t.Fatalf("discover: %d\n%s", c, o)
	}
}

func TestRigExportShowRoundTrip(t *testing.T) {
	home := isolated(t)
	skill := filepath.Join(home, ".claude", "skills", "tdd", "SKILL.md")
	os.MkdirAll(filepath.Dir(skill), 0o755)
	os.WriteFile(skill, []byte("---\nname: tdd\ndescription: Test first. Use when coding.\n---\nbody\n"), 0o644)
	card := filepath.Join(home, "rigcard.json")
	if c, _, e := call("rig", "export", "-o", card, "--summary", "test rig"); c != 0 {
		t.Fatalf("export: %d %s", c, e)
	}
	c, o, e := call("rig", "show", card)
	if c != 0 || !strings.Contains(o, "test rig") || !strings.Contains(o, "tdd") {
		t.Fatalf("show: %d\n%s\n%s", c, o, e)
	}
	bad := filepath.Join(home, "bad.json")
	os.WriteFile(bad, []byte(`{"rigcard":"9.9"}`), 0o644)
	if c, o, _ := call("rig", "show", bad); c != 1 || !strings.Contains(o, "REJECTED") {
		t.Fatalf("a card with an unknown version must be rejected: %d %s", c, o)
	}
}

func TestProfileNewLoadUseFlow(t *testing.T) {
	home := isolated(t)
	t.Setenv("CATUDE_HOME", filepath.Join(home, "ct"))
	skill := filepath.Join(home, ".claude", "skills", "tdd", "SKILL.md")
	os.MkdirAll(filepath.Dir(skill), 0o755)
	os.WriteFile(skill, []byte("---\nname: tdd\ndescription: Test first. Use when coding.\n---\nbody\n"), 0o644)
	if c, o, _ := call("profile", "list"); c != 0 || !strings.Contains(o, "no profiles yet") {
		t.Fatalf("empty list: %d %s", c, o)
	}
	stdin = strings.NewReader("n\n")
	if c, o, _ := call("profile", "new", "swe", "--skills", "tdd"); c != 1 || !strings.Contains(o, "nothing installed") {
		t.Fatalf("declining must install nothing: %d %s", c, o)
	}
	stdin = strings.NewReader("y\n")
	if c, o, e := call("profile", "new", "swe", "--skills", "tdd", "--summary", "software"); c != 0 || !strings.Contains(o, "installed swe") {
		t.Fatalf("new: %d %s %s", c, o, e)
	}
	c, o, _ := call("profile", "show", "swe")
	if c != 0 || !strings.Contains(o, "tdd") || !strings.Contains(o, "--plugin-dir") {
		t.Fatalf("show: %d %s", c, o)
	}
	// share it as a card and load it back under another name: local sources cannot be downloaded
	card := filepath.Join(home, "swe.json")
	os.WriteFile(card, []byte(`{"rigcard":"0.1","summary":"x","skills":[{"name":"tdd","source":{"type":"local"}}]}`), 0o644)
	if c, o, e := call("profile", "load", card, "--name", "copy", "--yes"); c != 0 || !strings.Contains(o, "not downloadable") {
		t.Fatalf("load: %d %s %s", c, o, e)
	}
	if c, _, e := call("profile", "use", "swe"); c != 1 || !strings.Contains(e, "claude is not on PATH") {
		t.Fatalf("use without claude: %d %s", c, e)
	}
	stdin = strings.NewReader("y\n")
	if c, _, _ := call("profile", "rm", "swe"); c != 0 {
		t.Fatal("rm")
	}
}

func TestInstallSaveAndSwitch(t *testing.T) {
	home := isolated(t)
	t.Setenv("CATUDE_HOME", filepath.Join(home, "ct"))
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "local"))
	var pathDir string
	addToPathFn = func(dir string, _ io.Writer) error { pathDir = dir; return nil }
	defer func() { addToPathFn = addToPath }()

	stdin = strings.NewReader("n\n")
	if c, o, _ := call("install"); c != 0 || !strings.Contains(o, "nothing installed") || pathDir != "" {
		t.Fatalf("declining must install nothing: %d %s", c, o)
	}
	stdin = strings.NewReader("\n\n\n") // enter = pre-selected answers: install yes, ct yes, shell no
	c, o, e := call("install")
	if c != 0 || pathDir != binDir() || !fileExists(filepath.Join(binDir(), exeName("catude"))) || !fileExists(filepath.Join(binDir(), exeName("ct"))) {
		t.Fatalf("install: %d %s %s (path %q)", c, o, e, pathDir)
	}

	skill := filepath.Join(home, ".claude", "skills", "tdd", "SKILL.md")
	os.MkdirAll(filepath.Dir(skill), 0o755)
	os.WriteFile(skill, []byte("---\nname: tdd\ndescription: Test first. Use when coding.\n---\nbody\n"), 0o644)
	if c, o, e := call("profile", "save", "mine", "--yes"); c != 0 || !strings.Contains(o, "installed mine") {
		t.Fatalf("save current setup: %d %s %s", c, o, e)
	}
	stdin = strings.NewReader("1\n")
	if c, o, _ := call("switch"); c != 0 || !strings.Contains(o, "active profile: mine") {
		t.Fatalf("switch picker: %d %s", c, o)
	}
	if c, o, _ := call("profile", "current"); c != 0 || strings.TrimSpace(o) != "mine" {
		t.Fatalf("current: %q", o)
	}
	if c, o, _ := call("switch", "none"); c != 0 || !strings.Contains(o, "no active profile") {
		t.Fatalf("switch none: %s", o)
	}
	if c, o, _ := call("--glyphs", "ascii", "profile", "list"); c != 0 || !strings.Contains(o, "mine") {
		t.Fatalf("global flags anywhere: %d %s", c, o)
	}
}

// The shareable card carries numbers only: never the name of a skill, server or project.
func TestCardHasNoNames(t *testing.T) {
	home := isolated(t)
	t.Setenv("CATUDE_HOME", filepath.Join(home, "ct"))
	skill := filepath.Join(home, ".claude", "skills", "secret-client-acme", "SKILL.md")
	os.MkdirAll(filepath.Dir(skill), 0o755)
	os.WriteFile(skill, []byte("---\nname: secret-client-acme\ndescription: Private. Use for Acme.\n---\nbody\n"), 0o644)
	os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"acme-internal-db":{"command":"npx","args":["-y","x@1.0.0"]}}}`), 0o644)
	svg := filepath.Join(home, "card.svg")
	c, o, e := call("card", "--svg", svg)
	if c != 0 || !strings.Contains(o, "1    skills") || !strings.Contains(o, "1    MCP servers") {
		t.Fatalf("card: %d\n%s\n%s", c, o, e)
	}
	img, _ := os.ReadFile(svg)
	for _, private := range []string{"acme", "secret-client", home} {
		if strings.Contains(strings.ToLower(o), private) || strings.Contains(strings.ToLower(string(img)), strings.ToLower(private)) {
			t.Fatalf("the card leaked %q", private)
		}
	}
	if !strings.HasPrefix(string(img), "<svg") {
		t.Error("svg not written")
	}
	if c, o, _ := call("--lite", "version"); c != 0 || !strings.Contains(o, "catude") {
		t.Fatalf("--lite is accepted anywhere: %d %s", c, o)
	}
}

// profile export → profile load from the folder: the documented way to share a profile.
func TestProfileExportAndLoadBundle(t *testing.T) {
	home := isolated(t)
	t.Setenv("CATUDE_HOME", filepath.Join(home, "ct"))
	skill := filepath.Join(home, ".claude", "skills", "tdd", "SKILL.md")
	os.MkdirAll(filepath.Dir(skill), 0o755)
	os.WriteFile(skill, []byte("---\nname: tdd\ndescription: Test first. Use when coding.\n---\nbody\n"), 0o644)
	if c, _, e := call("profile", "new", "swe", "--skills", "tdd", "--yes"); c != 0 {
		t.Fatal(e)
	}
	dir := filepath.Join(home, "share")
	c, o, e := call("profile", "export", "swe", dir)
	if c != 0 || !strings.Contains(o, "gh:YOUR-USER/YOUR-REPO") || !fileExists(filepath.Join(dir, "rigcard.json")) || !fileExists(filepath.Join(dir, "skills", "tdd", "SKILL.md")) {
		t.Fatalf("export: %d\n%s\n%s", c, o, e)
	}
	if c, _, _ := call("profile", "export", "swe", dir); c == 0 {
		t.Error("exporting into a non-empty folder must be refused")
	}
	c, o, e = call("profile", "load", filepath.Join(dir, "rigcard.json"), "--name", "swe-copy", "--yes")
	if c != 0 || !strings.Contains(o, "installed swe-copy") {
		t.Fatalf("load bundle: %d\n%s\n%s", c, o, e)
	}
	if c, o, _ := call("profile", "show", "swe-copy"); c != 0 || !strings.Contains(o, "tdd") {
		t.Fatalf("loaded profile: %s", o)
	}
}

// uninstall --help shows help and removes nothing; without --yes it asks first.
func TestUninstallAsksAndHelps(t *testing.T) {
	home := isolated(t)
	t.Setenv("CATUDE_HOME", filepath.Join(home, "ct"))
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "local"))
	addToPathFn = func(string, io.Writer) error { return nil }
	defer func() { addToPathFn = addToPath }()
	stdin = strings.NewReader("\n\n\n")
	if c, _, e := call("install"); c != 0 {
		t.Fatalf("install: %s", e)
	}
	exe := filepath.Join(binDir(), exeName("catude"))
	if c, o, _ := call("uninstall", "--help"); c != 0 || !strings.Contains(o, "--yes") || !fileExists(exe) {
		t.Fatalf("--help must only show help: %d %s", c, o)
	}
	stdin = strings.NewReader("n\n")
	if c, _, _ := call("uninstall"); c != 1 || !fileExists(exe) {
		t.Fatal("declining keeps catude installed")
	}
}

// catude instructions explains and, with --fix, makes Claude Code read the AGENTS.md it skips.
func TestInstructionsFix(t *testing.T) {
	home := isolated(t)
	proj := filepath.Join(home, "repo")
	os.MkdirAll(proj, 0o755)
	os.WriteFile(filepath.Join(proj, "AGENTS.md"), []byte("rules"), 0o644)
	os.WriteFile(filepath.Join(proj, "CLAUDE.local.md"), []byte("mine"), 0o644)
	if c, o, _ := call("instructions", "--project", proj); c != 0 || !strings.Contains(o, "--fix would") {
		t.Fatalf("explains: %d %s", c, o)
	}
	if c, o, _ := call("instructions", "--project", proj, "--fix", "--yes"); c != 0 || !strings.Contains(o, "done") {
		t.Fatalf("fix: %d %s", c, o)
	}
	if b, _ := os.ReadFile(filepath.Join(proj, "CLAUDE.local.md")); !strings.Contains(string(b), "@AGENTS.md") {
		t.Fatalf("import added: %s", b)
	}
}
