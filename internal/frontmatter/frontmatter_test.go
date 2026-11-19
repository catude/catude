package frontmatter

import (
	"strings"
	"testing"
)

func TestParseCRLFAndBOM(t *testing.T) {
	d, err := Parse([]byte("\xef\xbb\xbf---\r\nname: my-skill\r\ndescription: Does X. Use when Y.\r\n---\r\nBody\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if d.String("name") != "my-skill" || !strings.HasPrefix(d.Body, "Body") {
		t.Fatalf("bad parse: %+v", d)
	}
	if p := ValidateSkill(d, nil, "my-skill"); len(p) != 0 {
		t.Fatalf("valid skill reported problems: %v", p)
	}
}

func TestSkillProblems(t *testing.T) {
	d, err := Parse([]byte("---\nname: Bad_Name\n---\n"))
	p := ValidateSkill(d, err, "other")
	want := []string{"lowercase", "description", "empty body"}
	for _, w := range want {
		found := false
		for _, x := range p {
			if strings.Contains(x, w) {
				found = true
			}
		}
		if !found {
			t.Errorf("missing problem %q in %v", w, p)
		}
	}
	d2, _ := Parse([]byte("---\nname: ok\ndescription: " + strings.Repeat("a", 1100) + "\n---\nx"))
	if p := ValidateSkill(d2, nil, "ok"); len(p) != 1 || !strings.Contains(p[0], "max 1024") {
		t.Fatalf("long description not flagged: %v", p)
	}
}

func TestBrokenYAMLAndCommands(t *testing.T) {
	_, err := Parse([]byte("---\nname: [unclosed\n---\nx"))
	if err == nil {
		t.Fatal("invalid YAML accepted")
	}
	d, err := Parse([]byte("Just a prompt with $ARGUMENTS"))
	if p := ValidateCommand(d, err); len(p) != 0 {
		t.Fatalf("command without frontmatter must be valid: %v", p)
	}
	if _, err := Parse([]byte("---\nname: x\n")); err == nil {
		t.Fatal("unclosed frontmatter accepted")
	}
}
