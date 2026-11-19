package paths

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ProjectTrusted reports whether the user has accepted Claude Code's trust dialog for the project folder
// (projects[<folder>].hasTrustDialogAccepted in ~/.claude.json).
//
// Why it matters: catude asks Claude Code for facts with headless commands (claude -p /context,
// claude mcp list), and headless mode skips the trust dialog. In a folder you have not trusted — a
// repository you just cloned — that would start the hooks and MCP servers the repository defines.
func (e Env) ProjectTrusted() bool {
	if e.ProjectDir == "" || e.ClaudeJSON == "" {
		return false
	}
	fi, err := os.Stat(e.ClaudeJSON)
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > 64<<20 {
		return false
	}
	b, err := os.ReadFile(e.ClaudeJSON)
	if err != nil {
		return false
	}
	var cj struct {
		Projects map[string]struct {
			Trusted bool `json:"hasTrustDialogAccepted"`
		} `json:"projects"`
	}
	if json.Unmarshal(b, &cj) != nil {
		return false
	}
	want := normPath(e.ProjectDir)
	for dir, p := range cj.Projects {
		if p.Trusted && normPath(dir) == want {
			return true
		}
	}
	return false
}

func normPath(p string) string {
	p = strings.TrimRight(filepath.ToSlash(filepath.Clean(p)), "/")
	if runtime.GOOS == "windows" || (len(p) > 1 && p[1] == ':') {
		p = strings.ToLower(p)
	}
	return p
}

// RunDir is where catude may run headless claude commands: the project folder when you have trusted it
// in Claude Code, otherwise an empty neutral folder (so only your user-level setup is started).
func (e Env) RunDir() (dir string, trusted bool) {
	if e.ProjectTrusted() {
		return e.ProjectDir, true
	}
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	dir = filepath.Join(base, "catude", "neutral")
	if os.MkdirAll(dir, 0o700) != nil {
		return os.TempDir(), false
	}
	return dir, false
}
