package live

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// LR_LIVE=1 go test -run Real -v ./internal/live (this machine's running sessions)
func TestRealSessions(t *testing.T) {
	if os.Getenv("LR_LIVE") == "" {
		t.Skip()
	}
	home, _ := os.UserHomeDir()
	r := &Reader{ClaudeDir: filepath.Join(home, ".claude")}
	t0 := time.Now()
	ss := r.Sessions()
	t.Logf("first read %v", time.Since(t0))
	t0 = time.Now()
	ss = r.Sessions()
	t.Logf("refresh %v", time.Since(t0))
	for _, s := range ss {
		t.Logf("%d %s %q %s %s ctx=%d model=%s effort=%s todo=%+v working=%v", s.PID, s.Name, s.Title, s.Entrypoint, s.Status, s.Context, s.Model, s.Effort, s.Todo, s.Working())
		for _, st := range s.Steps {
			t.Logf("   %v %-12s %s done=%v err=%v sub=%d", st.At.Format("15:04:05"), st.Tool, st.Detail, st.Done, st.Err, len(st.Sub))
		}
		t.Logf("   last: %s", s.LastText)
	}
}
