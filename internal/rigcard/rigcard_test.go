package rigcard

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
)

func TestCandidates(t *testing.T) {
	cases := map[string][]string{
		"example.dev/rig":       {"https://example.dev/rig", "https://example.dev/rig.json"},
		"https://example.dev/":  {"https://example.dev/.well-known/rigcard.json"},
		"example.dev/ada.json":  {"https://example.dev/ada.json"},
		"gh:ada/dotclaude@v1.2": {"https://raw.githubusercontent.com/ada/dotclaude/v1.2/rigcard.json"},
	}
	for in, want := range cases {
		got, err := Candidates(in)
		if err != nil || strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("%s → %v (%v), want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"http://example.dev/rig", "gh:../../x", "gh:a/b@$(rm)"} {
		if _, err := Candidates(bad); err == nil {
			t.Errorf("%s must be refused", bad)
		}
	}
}

func TestPublicIP(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.1.2.3", "192.168.1.10", "169.254.169.254", "100.64.0.1", "::1", "fe80::1", "0.0.0.0"} {
		if PublicIP(net.ParseIP(s)) {
			t.Errorf("%s must not be considered public", s)
		}
	}
	if !PublicIP(net.ParseIP("140.82.112.3")) {
		t.Error("a public address was refused")
	}
}

func TestValidate(t *testing.T) {
	good := Card{Rigcard: Version, Marketplaces: []Marketplace{{Name: "tools", Source: Source{Source: "github", Repo: "ada/tools", Ref: strings.Repeat("a", 40)}}}}
	if Rejected(Validate(good, nil)) {
		t.Fatalf("good card rejected: %v", Validate(good, nil))
	}
	secret := []byte(`{"rigcard":"0.1","summary":"key ghp_` + strings.Repeat("Ab3x", 10) + `"}`)
	c, _ := Decode(secret)
	if !Rejected(Validate(c, secret)) {
		t.Error("card with a secret must be rejected")
	}
	bidi := Card{Rigcard: Version, Skills: []Artifact{{Name: "safe" + string(rune(0x202E)) + "evil"}}}
	if !Rejected(Validate(bidi, nil)) {
		t.Error("bidi name must be rejected")
	}
	fake := Card{Rigcard: Version, Marketplaces: []Marketplace{{Name: "claude-plugins-official", Source: Source{Repo: "mallory/official"}}}}
	if !Rejected(Validate(fake, nil)) {
		t.Error("impersonation of the official marketplace must be rejected")
	}
	if !Rejected(Validate(Card{Rigcard: "9.9"}, nil)) {
		t.Error("unknown version must be rejected")
	}
	unpinned := Card{Rigcard: Version, MCPServers: []MCPServer{{Name: "x", Transport: "stdio", Command: "npx", Args: []string{"-y", "some-mcp"}}}}
	if ps := Validate(unpinned, nil); len(ps) == 0 || Rejected(ps) {
		t.Errorf("unpinned npx must warn, not reject: %v", ps)
	}
}

func TestTreeHashIsStable(t *testing.T) {
	d := t.TempDir()
	os.MkdirAll(filepath.Join(d, "sub"), 0o755)
	os.WriteFile(filepath.Join(d, "SKILL.md"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(d, "sub", "a.md"), []byte("y"), 0o644)
	h1, err := TreeHash(d)
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := TreeHash(d)
	os.WriteFile(filepath.Join(d, "sub", "a.md"), []byte("z"), 0o644)
	h3, _ := TreeHash(d)
	if h1 != h2 || h1 == h3 || len(h1) != 64 {
		t.Fatalf("hash not stable/sensitive: %s %s %s", h1, h2, h3)
	}
}

func writeFile(t *testing.T, p, s string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExportNeverLeaksSecretsAndPlan(t *testing.T) {
	r := t.TempDir()
	env := paths.Env{GOOS: "linux", Home: r, ClaudeDir: filepath.Join(r, ".claude"), ClaudeJSON: filepath.Join(r, ".claude.json")}
	writeFile(t, filepath.Join(env.ClaudeDir, "skills", "tdd", "SKILL.md"), "---\nname: tdd\ndescription: Test first. Use when coding.\n---\nRed, green.\n")
	secret := "ghp_" + strings.Repeat("Zq9X", 10)
	writeFile(t, env.ClaudeJSON, `{"mcpServers":{"gh":{"type":"http","url":"https://api.example.com/mcp","headers":{"Authorization":"Bearer `+secret+`"}},"db":{"command":"uvx","args":["pg-mcp==1.2.0"],"env":{"DB_PASSWORD":"`+secret+`","DB_HOST":"localhost"}}}}`)
	writeFile(t, filepath.Join(env.ClaudeDir, "settings.json"), `{"model":"opus","permissions":{"allow":["Bash"],"deny":["Read(./.env)"]},"env":{"TOKEN":"`+secret+`"}}`)
	inv := inventory.Load(env)
	c, err := Export(inv, ExportOptions{Summary: "test"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(c)
	if strings.Contains(string(raw), secret) || strings.Contains(string(raw), "Bearer") {
		t.Fatalf("secret leaked into the card: %s", raw)
	}
	if strings.Contains(string(raw), `"allow"`) || strings.Contains(string(raw), `"TOKEN"`) {
		t.Fatal("forbidden settings exported (permissions.allow / env)")
	}
	if len(c.Skills) != 1 || len(c.Skills[0].SHA256) != 64 || len(c.MCPServers) != 2 {
		t.Fatalf("export content: %+v", c)
	}
	if Rejected(Validate(c, raw)) {
		t.Fatalf("own export must validate: %v", Validate(c, raw))
	}
	// someone else's card against this machine
	other := Card{Rigcard: Version, Plugins: []Plugin{{ID: "superpowers@claude-plugins-official", Version: "7.0.2"}},
		Skills:     []Artifact{{Name: "tdd"}, {Name: "grilling"}},
		MCPServers: []MCPServer{{Name: "linear", Transport: "http", URL: "https://mcp.linear.app/mcp", Headers: []Variable{{Name: "Authorization", IsSecret: true}}}}}
	steps := Plan(other, inv)
	got := map[string]string{}
	for _, s := range steps {
		got[s.Kind+":"+s.Name] = s.State + "|" + s.Cmd
	}
	if !strings.HasPrefix(got["skill:tdd"], "have") || !strings.HasPrefix(got["skill:grilling"], "missing") {
		t.Errorf("skills plan: %v", got)
	}
	if !strings.Contains(got["plugin:superpowers@claude-plugins-official"], "claude plugin install superpowers@claude-plugins-official") {
		t.Errorf("plugin plan: %v", got)
	}
	if !strings.Contains(got["mcp:linear"], "--transport http") || !strings.Contains(got["mcp:linear"], "${AUTHORIZATION}") {
		t.Errorf("mcp plan must reference an env var, never a value: %v", got["mcp:linear"])
	}
}

func TestOwnerArtIsBounded(t *testing.T) {
	ok := Card{Rigcard: Version, Owner: &Owner{Name: "Ada", Art: " /-_-/\n( o.o )\n > ^ <"}}
	if Rejected(Validate(ok, nil)) {
		t.Fatalf("small ASCII art must be accepted: %v", Validate(ok, nil))
	}
	for _, bad := range []string{strings.Repeat("x\n", 13), strings.Repeat("x", 41), "a\x1b[31mb", "a" + string(rune(0x202E)) + "b"} {
		if !Rejected(Validate(Card{Rigcard: Version, Owner: &Owner{Art: bad}}, nil)) {
			t.Errorf("art %q must be rejected", bad)
		}
	}
}

func TestPublicIPReservedRanges(t *testing.T) {
	for _, s := range []string{"192.0.0.1", "192.0.2.5", "198.18.0.1", "198.19.255.255", "198.51.100.7", "203.0.113.9", "240.0.0.1", "255.255.255.255",
		"::ffff:127.0.0.1", "::ffff:10.0.0.1", "64:ff9b::7f00:1", "64:ff9b::a00:1", "2001:db8::1", "2002:7f00:1::1", "2002:c0a8:101::1", "fc00::1", "fd12:3456::1"} {
		if PublicIP(net.ParseIP(s)) {
			t.Errorf("%s must not be considered public", s)
		}
	}
	for _, s := range []string{"8.8.8.8", "140.82.112.3", "2606:4700:4700::1111", "64:ff9b::808:808"} {
		if !PublicIP(net.ParseIP(s)) {
			t.Errorf("%s is public", s)
		}
	}
	for _, bad := range []string{"CON", "nul.json", "a:b", "x.", " x", "a|b", "a?b"} {
		if SafeName(bad) {
			t.Errorf("name %q must be refused", bad)
		}
	}
}
