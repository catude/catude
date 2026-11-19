// Package frontmatter parses and validates the YAML header of skills, agents and commands.
package frontmatter

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// MaxFileBytes bounds how much of a Markdown file is read, so a huge or binary file cannot hang the UI.
const MaxFileBytes = 2 << 20

// Doc is a parsed Markdown file with optional YAML frontmatter.
type Doc struct {
	Meta map[string]any
	Body string
}

// ErrNoFrontmatter is returned when the file does not start with a '---' block.
var ErrNoFrontmatter = errors.New("no YAML frontmatter")

// Parse splits '---\n<yaml>\n---\n<body>'. It accepts CRLF and a UTF-8 BOM.
func Parse(content []byte) (Doc, error) {
	c := bytes.TrimPrefix(content, []byte("\xef\xbb\xbf"))
	c = bytes.ReplaceAll(c, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(c, []byte("---\n")) {
		return Doc{Meta: map[string]any{}, Body: string(c)}, ErrNoFrontmatter
	}
	rest := c[4:]
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return Doc{Meta: map[string]any{}, Body: string(c)}, errors.New("frontmatter is not closed with '---'")
	}
	meta := map[string]any{}
	if err := yaml.Unmarshal(rest[:end], &meta); err != nil {
		return Doc{Meta: map[string]any{}, Body: string(c)}, fmt.Errorf("invalid YAML in frontmatter: %w", err)
	}
	if meta == nil {
		meta = map[string]any{}
	}
	body := rest[end+4:]
	body = bytes.TrimPrefix(body, []byte("\n"))
	return Doc{Meta: meta, Body: string(body)}, nil
}

// String returns a string field, or "".
func (d Doc) String(key string) string {
	if v, ok := d.Meta[key]; ok {
		switch t := v.(type) {
		case string:
			return t
		case fmt.Stringer:
			return t.String()
		}
	}
	return ""
}

// Bool returns a boolean field, accepting true/"true".
func (d Doc) Bool(key string) bool {
	switch t := d.Meta[key].(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(t, "true")
	}
	return false
}

var skillName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// ValidateSkill applies the Agent Skills rules: name (lowercase, digits, hyphens, ≤64) matching the
// folder, and a description (≤1024 chars) that says what the skill does and when to use it.
func ValidateSkill(d Doc, parseErr error, folder string) []string {
	var p []string
	if parseErr != nil {
		return []string{parseErr.Error()}
	}
	name, desc := d.String("name"), d.String("description")
	switch {
	case name == "":
		p = append(p, "missing 'name'")
	case !skillName.MatchString(name):
		p = append(p, "'name' must be lowercase letters, digits and hyphens (max 64)")
	case folder != "" && name != folder:
		p = append(p, fmt.Sprintf("'name' (%s) differs from its folder (%s)", name, folder))
	}
	switch {
	case strings.TrimSpace(desc) == "":
		p = append(p, "missing 'description' (Claude uses it to decide when to load the skill)")
	case len([]rune(desc)) > 1024:
		p = append(p, fmt.Sprintf("'description' is %d characters (max 1024)", len([]rune(desc))))
	}
	if strings.TrimSpace(d.Body) == "" {
		p = append(p, "empty body")
	}
	return p
}

// ValidateAgent checks a subagent definition (name + description required).
func ValidateAgent(d Doc, parseErr error) []string {
	if parseErr != nil {
		return []string{parseErr.Error()}
	}
	var p []string
	if d.String("name") == "" {
		p = append(p, "missing 'name'")
	}
	if strings.TrimSpace(d.String("description")) == "" {
		p = append(p, "missing 'description' (Claude uses it to decide when to delegate)")
	}
	if strings.TrimSpace(d.Body) == "" {
		p = append(p, "empty system prompt")
	}
	return p
}

// ValidateCommand checks a slash command: frontmatter is optional, the body is required.
func ValidateCommand(d Doc, parseErr error) []string {
	if parseErr != nil && !errors.Is(parseErr, ErrNoFrontmatter) {
		return []string{parseErr.Error()}
	}
	if strings.TrimSpace(d.Body) == "" {
		return []string{"empty command"}
	}
	return nil
}
