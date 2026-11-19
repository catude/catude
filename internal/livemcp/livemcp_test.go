package livemcp

import (
	"strings"
	"testing"
)

const sample = `Checking MCP server health…

claude.ai Gmail: https://gmailmcp.googleapis.com/mcp/v1 - ✔ Connected
claude.ai Example CRM: https://example.org/crm/mcp - ! Needs authentication
claude.ai Broken: https://x.example/mcp?token=abcdefghijklmnopqrstuvwxyz012345 - ✗ Failed to connect
portainer: uvx --from mcp-portainer==2.45.3 mcp-portainer --api-key ptr_SECRETSECRETSECRETSECRET - ✔ Connected
`

func TestParse(t *testing.T) {
	s := Parse(sample)
	if len(s) != 4 {
		t.Fatalf("got %d servers, want 4: %+v", len(s), s)
	}
	if !s[0].ClaudeAI || s[0].DisplayName() != "Gmail" || s[0].Status != Connected {
		t.Errorf("gmail: %+v", s[0])
	}
	if s[1].Status != NeedsAuth || s[1].DisplayName() != "Example CRM" {
		t.Errorf("crm: %+v", s[1])
	}
	if s[2].Status != Failed || strings.Contains(s[2].Target, "abcdefghij") {
		t.Errorf("broken (secret must be masked): %+v", s[2])
	}
	if s[3].ClaudeAI || strings.Contains(s[3].Target, "ptr_SECRET") || !strings.Contains(s[3].Target, "mcp-portainer==2.45.3") {
		t.Errorf("portainer: %+v", s[3])
	}
}
