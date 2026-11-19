package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/catude/catude/internal/models"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/sessions"
)

// The Models tab shows what you use, advice, a ranking for the chosen kind of work and a verdict.
// Data here is invented.
func TestModelsView(t *testing.T) {
	m := newModel(paths.Env{ClaudeDir: t.TempDir()}, "test")
	m.w, m.h = 130, 40
	mk := func(id, group, effort string, arc, lmca, cursor float64) models.Model {
		return models.Model{ID: id, Base: strings.Split(id, "_")[0], Group: group, Effort: effort, Released: "2026-10-01",
			Scores:  map[string]float64{"ARC-AGI-2": arc, "LMCA": lmca, "CursorBench": cursor},
			Raw:     map[string]float64{},
			PriceIn: 4, PriceOut: 20}
	}
	m.mo.data = models.Data{Fetched: time.Now(), Updated: "2026-11-14",
		Benches: []models.Bench{{Name: "ARC-AGI-2", Category: "reasoning"}, {Name: "LMCA", Category: "agents"}, {Name: "CursorBench", Category: "coding"}},
		Models: []models.Model{
			mk("claude-alpha-9-1_medium", "Claude Alpha 9.1", "medium", 96, 100, 91),
			mk("claude-opus-5-5_high", "Claude Opus 5.5", "high", 100, 94, 93),
			mk("claude-opus-5-5_medium", "Claude Opus 5.5", "medium", 80, 85, 87),
		}}
	m.mo.data.Models[0].PriceIn, m.mo.data.Models[0].PriceOut = 8, 40
	m.mo.loaded = true
	m.usage.rep = &sessions.UsageReport{Models: map[string]int{"claude-opus-5-5-20260922": 100}}
	out := stripANSI(m.modelsView(36))
	if os.Getenv("SHOW") != "" {
		t.Log("\n" + out)
	}
	for _, want := range []string{"no account, no key", "Epoch AI", "You use Opus 5.5 · medium", "effortLevel not set", "for harder work"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// t changes the kind of work: on coding only CursorBench counts
	m.mo.cat = 1
	if out := stripANSI(m.modelsView(36)); !strings.Contains(out, "score (coding)") || !strings.Contains(out, "on CursorBench") {
		t.Errorf("coding view:\n%s", out)
	}
}

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }
