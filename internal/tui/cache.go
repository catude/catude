package tui

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// Small on-disk cache so slow results (claude /context, claude mcp list, folder sizes) show instantly
// on the next start and are refreshed in the background ("stale while revalidate").
// It lives in <user cache dir>/catude/state and never holds secrets (MCP output is already redacted
// by livemcp; compositions and sizes contain no values).

func cacheDir() string {
	d, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "catude", "state")
}

func cacheKey(parts ...string) string {
	h := sha1.New()
	for _, p := range parts {
		h.Write([]byte(p + "\x00"))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func readCache(name string, v any) bool {
	d := cacheDir()
	if d == "" {
		return false
	}
	b, err := os.ReadFile(filepath.Join(d, name+".json"))
	if err != nil {
		return false
	}
	return json.Unmarshal(b, v) == nil
}

func writeCache(name string, v any) {
	d := cacheDir()
	if d == "" {
		return
	}
	if os.MkdirAll(d, 0o700) != nil {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	tmp := filepath.Join(d, name+".json.tmp")
	if os.WriteFile(tmp, b, 0o600) == nil {
		os.Rename(tmp, filepath.Join(d, name+".json"))
	}
}

func contextBG() context.Context { return context.Background() }
