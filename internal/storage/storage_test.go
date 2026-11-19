package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/catude/catude/internal/paths"
)

func touch(t *testing.T, p string, age time.Duration) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := time.Now().Add(-age)
	os.Chtimes(p, ts, ts)
}

func TestCatalogMeasureAndSafeClean(t *testing.T) {
	r := t.TempDir()
	env := paths.Env{GOOS: "linux", Home: r, ClaudeDir: filepath.Join(r, ".claude"), ClaudeJSON: filepath.Join(r, ".claude.json")}
	touch(t, filepath.Join(env.ClaudeDir, "debug", "old.txt"), 48*time.Hour)
	touch(t, filepath.Join(env.ClaudeDir, "debug", "now.txt"), 0)
	touch(t, filepath.Join(env.ClaudeDir, "projects", "p", "s.jsonl"), 40*24*time.Hour)
	touch(t, filepath.Join(env.ClaudeDir, ".credentials.json"), 48*time.Hour)
	touch(t, filepath.Join(env.ClaudeDir, "mystery", "x"), 48*time.Hour)
	touch(t, env.ClaudeJSON, 0)
	es := Catalog(env)
	byKey := map[string]Entry{}
	for _, e := range es {
		byKey[e.Key] = e
		if filepath.Base(e.Path) == ".credentials.json" {
			t.Fatal("credentials must never be listed")
		}
	}
	if !byKey["code:debug"].Safe || byKey["code:projects"].Safe || byKey["code:mystery"].Safe || byKey["code:claude.json"].Safe {
		t.Fatalf("safety flags wrong: %+v", byKey)
	}
	d := Measure(context.Background(), byKey["code:debug"])
	if d.Files != 2 || d.Bytes != 20 {
		t.Fatalf("measure: %+v", d)
	}
	if _, err := Clean(byKey["code:projects"]); err == nil {
		t.Fatal("transcripts must not be cleanable")
	}
	res, err := Clean(byKey["code:debug"])
	if err != nil || res.Removed != 1 || res.Kept != 1 {
		t.Fatalf("clean: %+v %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(env.ClaudeDir, "debug", "now.txt")); err != nil {
		t.Error("a recent file (maybe a running session) was removed")
	}
	if _, err := os.Stat(filepath.Join(env.ClaudeDir, ".credentials.json")); err != nil {
		t.Error("credentials touched")
	}
}

func TestScratchCleanKeepsRunningSession(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "abc", "old-session", "scratchpad", "a.txt"), 5*time.Hour)
	touch(t, filepath.Join(root, "abc", "this-session", "scratchpad", "b.txt"), 0)
	touch(t, filepath.Join(root, "def", "gone", "x.txt"), 30*time.Hour)
	res, err := Clean(Entry{Key: "tmp:" + root, Path: root, Safe: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "abc", "this-session", "scratchpad", "b.txt")); err != nil {
		t.Fatal("the running session's scratchpad was removed")
	}
	if _, err := os.Stat(filepath.Join(root, "abc", "old-session")); err == nil {
		t.Error("an old session folder was kept")
	}
	if _, err := os.Stat(filepath.Join(root, "def")); err == nil {
		t.Error("an emptied project folder was kept")
	}
	if res.Removed != 2 {
		t.Errorf("removed %d", res.Removed)
	}
}

func TestRetention(t *testing.T) {
	r := t.TempDir()
	env := paths.Env{GOOS: "linux", Home: r, ClaudeDir: filepath.Join(r, ".claude")}
	touch(t, filepath.Join(env.ClaudeDir, "projects", "p", "old.jsonl"), 25*24*time.Hour)
	touch(t, filepath.Join(env.ClaudeDir, "projects", "p", "new.jsonl"), time.Hour)
	ret := ReadRetention(env)
	if ret.Days != 30 || ret.Explicit || ret.Expiring != 1 || ret.Transcrip != 2 {
		t.Fatalf("default retention: %+v", ret)
	}
	os.WriteFile(filepath.Join(env.ClaudeDir, "settings.json"), []byte(`{"cleanupPeriodDays": 365}`), 0o644)
	if ret := ReadRetention(env); ret.Days != 365 || !ret.Explicit || ret.Expiring != 0 {
		t.Fatalf("explicit retention: %+v", ret)
	}
}
