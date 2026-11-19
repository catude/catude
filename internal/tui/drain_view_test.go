package tui

import (
	"strings"
	"testing"

	"github.com/catude/catude/internal/sessions"
)

// Every finding the analysis can produce has its sentence in the interface, with all its values.
func TestDrainTextsCoverEveryFinding(t *testing.T) {
	for id, en := range sessions.Texts {
		n := strings.Count(en[0], "%s")
		args := make([]string, n)
		for i := range args {
			args[i] = "V" + string(rune('A'+i))
		}
		text, hint := drainText(sessions.Finding{ID: id, Args: args})
		if hint == "" || strings.Contains(text, "%!") {
			t.Errorf("%s: %q / %q", id, text, hint)
		}
		for _, a := range args {
			if !strings.Contains(text, a) {
				t.Errorf("%s: %q misses %s", id, text, a)
			}
		}
	}
}
