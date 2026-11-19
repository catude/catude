package redact

import (
	"strings"
	"testing"
)

// fake builds credential-shaped strings at run time so no literal secret lives in the repository.
func fake(prefix string, n int) string { return prefix + strings.Repeat("aB3dE5", n/6+1)[:n] }

func TestEnvReferencesAreNotSecrets(t *testing.T) {
	for _, v := range []string{"${API_KEY}", "$TOKEN", "%PORTAINER_API_KEY%"} {
		if Value("API_KEY", v) != v {
			t.Errorf("env reference %q was masked", v)
		}
	}
}

func TestMaskByKeyAndShape(t *testing.T) {
	if Value("PORTAINER_API_KEY", "anything") != Mask {
		t.Error("secret-named variable not masked")
	}
	if Value("URL", fake("sk-ant-", 30)) != Mask {
		t.Error("anthropic-shaped value not masked")
	}
	if Value("REGION", "eu-west-1") != "eu-west-1" {
		t.Error("ordinary value masked")
	}
}

func TestArgsAndURLs(t *testing.T) {
	got := Args([]string{"--from", "pkg==1.0", "--api-key", "abc", "--token=" + fake("ghp_", 30), "Bearer xyz"})
	joined := strings.Join(got, " ")
	if strings.Contains(joined, "abc") || strings.Contains(joined, "ghp_") || strings.Contains(joined, "xyz") {
		t.Fatalf("secret leaked in args: %v", got)
	}
	if !strings.Contains(joined, "pkg==1.0") {
		t.Fatal("normal argument was masked")
	}
	u := URL("https://user:pa55@example.com/mcp?token=" + fake("x", 30) + "&page=2")
	if strings.Contains(u, "pa55") || strings.Contains(u, "xaB") || !strings.Contains(u, "page=2") {
		t.Fatalf("url not redacted correctly: %s", u)
	}
}

func TestText(t *testing.T) {
	s := "export X=" + fake("sk-ant-", 30) + " && run"
	if strings.Contains(Text(s), "sk-ant-") {
		t.Fatal("secret survived Text")
	}
}
