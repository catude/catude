package instr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catude/catude/internal/paths"
)

func touch(t *testing.T, p, s string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte(s), 0o644)
}

// The documented table: AGENTS.md alone is read; a CLAUDE.local.md silently stops it; an import or the
// claude-md-and-agents-md mode brings it back; .agents/skills is never read.
func TestCheck(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "code", "shop")
	env := paths.Env{GOOS: "linux", Home: home, ClaudeDir: filepath.Join(home, ".claude"), ProjectDir: proj}
	touch(t, filepath.Join(proj, "AGENTS.md"), "# rules")
	touch(t, filepath.Join(proj, ".agents", "skills", "deploy", "SKILL.md"), "---\nname: deploy\n---\n")
	touch(t, filepath.Join(proj, "AGENTS.override.md"), "x")

	r := Check(env)
	if len(r.Ignored()) != 0 || r.Mode != "claude-md-or-agents-md" {
		t.Fatalf("AGENTS.md alone is read: %+v", r)
	}
	if len(r.AgentSkills) != 1 {
		t.Errorf(".agents/skills found: %v", r.AgentSkills)
	}
	touch(t, filepath.Join(proj, "CLAUDE.local.md"), "my notes")
	if r = Check(env); len(r.Ignored()) != 1 {
		t.Fatalf("a CLAUDE.local.md stops AGENTS.md: %+v", r.Files)
	}
	touch(t, filepath.Join(env.ClaudeDir, "settings.json"), `{"pluginConfigs": {"cc-plugin-agents-md@builtin": {"options": {"instructionFiles": "claude-md-and-agents-md"}}}}`)
	if r = Check(env); len(r.Ignored()) != 0 || r.ModeFrom == "" {
		t.Fatalf("claude-md-and-agents-md reads both: %+v", r)
	}
	os.Remove(filepath.Join(env.ClaudeDir, "settings.json"))
	touch(t, filepath.Join(proj, "CLAUDE.md"), "See @AGENTS.md for the rules")
	if r = Check(env); len(r.Ignored()) != 0 {
		t.Fatalf("an import counts as read: %+v", r.Files)
	}
	ignored := 0
	for _, f := range r.Files {
		if f.Kind == "ignored" {
			ignored++
		}
	}
	if ignored != 1 {
		t.Errorf("AGENTS.override.md is listed as never read: %+v", r.Files)
	}
}

// Your ~/.claude/CLAUDE.md is not a project file: it does not stop AGENTS.md.
func TestUserFileDoesNotCount(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "repo")
	touch(t, filepath.Join(home, ".claude", "CLAUDE.md"), "mine")
	touch(t, filepath.Join(proj, "AGENTS.md"), "rules")
	r := Check(paths.Env{Home: home, ClaudeDir: filepath.Join(home, ".claude"), ProjectDir: proj})
	if len(r.Ignored()) != 0 || len(r.Files) != 1 {
		t.Fatalf("%+v", r.Files)
	}
}

// Fixes: the import goes into the CLAUDE.local.md that stops AGENTS.md, and .agents/skills is copied.
func TestFixes(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "repo")
	env := paths.Env{Home: home, ClaudeDir: filepath.Join(home, ".claude"), ProjectDir: proj}
	touch(t, filepath.Join(proj, "AGENTS.md"), "rules")
	touch(t, filepath.Join(proj, "CLAUDE.local.md"), "mine")
	touch(t, filepath.Join(proj, ".agents", "skills", "deploy", "SKILL.md"), "---\nname: deploy\n---\n")
	backed := ""
	fs := Fixes(env, func(p string) (string, error) { backed = p; return p + ".bak", nil })
	if len(fs) != 2 {
		t.Fatalf("fixes: %+v", fs)
	}
	for _, f := range fs {
		if err := f.Apply(); err != nil {
			t.Fatal(err)
		}
	}
	if r := Check(env); len(r.Ignored()) != 0 || !strings.HasSuffix(backed, "CLAUDE.local.md") {
		t.Fatalf("after the fix AGENTS.md is read: %+v (backup of %s)", r.Files, backed)
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude", "skills", "deploy", "SKILL.md")); err != nil {
		t.Fatal("the skill is copied")
	}
	if len(Fixes(env, func(p string) (string, error) { return "", nil })) != 0 {
		t.Error("nothing left to fix")
	}
}

// Auto memory is found under the project's slug, and an index over 200 lines is flagged.
func TestAutoMemory(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "my repo")
	os.MkdirAll(filepath.Join(proj, ".git"), 0o755)
	cd := filepath.Join(home, ".claude")
	slug := reNonAlnum.ReplaceAllString(filepath.Clean(proj), "-")
	touch(t, filepath.Join(cd, "projects", slug, "memory", "MEMORY.md"), strings.Repeat("- a note\n", 210))
	touch(t, filepath.Join(cd, "projects", slug, "memory", "testing.md"), "use go test")
	m, ok := AutoMemory(paths.Env{ClaudeDir: cd, ProjectDir: filepath.Join(proj, "sub")})
	if !ok || len(m.Files) != 1 || m.IndexLines != 210 || !m.Truncated {
		t.Fatalf("%+v %v", m, ok)
	}
}
