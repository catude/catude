package ctxcomp

import (
	"errors"
	"testing"
)

// Shape of `claude -p /context` output (Claude Code 2.1.x).
const sample = `## Context Usage

**Model:** claude-opus-5-5
**Tokens:** 24.9k / 1m (2%)

### Estimated usage by category

| Category | Tokens | Percentage |
|----------|--------|------------|
| System prompt | 2.5k | 0.3% |
| System tools | 10k | 1.0% |
| Skills | 6.7k | 0.7% |

### MCP Tools

| Tool | Server | Tokens |
|------|--------|--------|
| mcp__a__x | a | 1.1k |
| mcp__a__y | a | 461 |
| mcp__b__z | b | 300 |

### Memory Files

| Type | Path | Tokens |
|------|------|--------|
| User | C:\Users\u\.claude\CLAUDE.md | 241 |

### Skills

| Skill | Source | Tokens |
|-------|--------|--------|
| tdd | User | ~50 |
| dataviz | Built-in | ~480 |
`

func TestParse(t *testing.T) {
	c, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	if c.Model != "claude-opus-5-5" || c.Used != 24900 || c.Max != 1_000_000 {
		t.Fatalf("header: %+v", c)
	}
	if len(c.Categories) != 3 || c.Categories[1].Tokens != 10000 {
		t.Fatalf("categories: %+v", c.Categories)
	}
	if len(c.Skills) != 2 || c.Skills[1].Tokens != 480 || c.Skills[1].Source != "Built-in" {
		t.Fatalf("skills: %+v", c.Skills)
	}
	if len(c.Memory) != 1 || c.Memory[0].Tokens != 241 {
		t.Fatalf("memory: %+v", c.Memory)
	}
	by := c.MCPByServer()
	if len(by) != 2 || by[0].Name != "a" || by[0].Tokens != 1561 {
		t.Fatalf("mcp by server: %+v", by)
	}
}

func TestUnknownFormatIsAnError(t *testing.T) {
	if _, err := Parse("something else entirely"); !errors.Is(err, ErrFormat) {
		t.Fatalf("expected ErrFormat, got %v", err)
	}
}

func TestNumber(t *testing.T) {
	for in, want := range map[string]int{"6.7k": 6700, "~70": 70, "1m": 1_000_000, "24,900": 24900, "1.2M": 1_200_000, "": 0} {
		if got := Number(in); got != want {
			t.Errorf("Number(%q) = %d, want %d", in, got, want)
		}
	}
}
