// Package storage shows what Claude Code, Claude Desktop and their scratchpads keep on disk, how much
// space it takes, what is safe to delete, and when Claude Code will delete transcripts by itself.
//
// Safety model: only categories marked Safe can be cleaned, cleaning only removes files older than
// MinAge (so a running session is never touched), credentials and configuration are never listed as
// safe, and nothing outside the listed folders is ever removed.
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/catude/catude/internal/paths"
)

// MinAge protects files that may belong to a running session.
const MinAge = time.Hour

// Entry is one place on disk.
type Entry struct {
	Key    string
	Group  string // Claude Code | Scratchpads | Claude Desktop | catude
	Label  string
	Path   string
	Safe   bool   // can be cleaned from catude
	Note   string // what it is / what you lose
	IsFile bool

	// filled by Measure
	Bytes    int64
	Files    int
	Newest   time.Time
	Measured bool
	Partial  bool // measurement stopped at a limit
	Err      string
}

type def struct {
	key, label string
	safe       bool
	note       string
}

// Claude Code's folders, from https://code.claude.com/docs/en/claude-directory and observation.
var codeDirs = []def{
	{"projects", "Session transcripts", false, "your conversations (/resume). Claude Code deletes them after cleanupPeriodDays"},
	{"file-history", "Checkpoints (file history)", false, "lets /rewind restore files; cleaned by Claude Code with the transcripts"},
	{"plans", "Plans", true, "plan-mode documents of past sessions"},
	{"debug", "Debug logs", true, "diagnostic logs; nothing is lost"},
	{"shell-snapshots", "Shell snapshots", true, "copies of your shell environment per session; recreated when needed"},
	{"session-env", "Session environments", true, "per-session environment data; recreated when needed"},
	{"paste-cache", "Paste cache", true, "large pasted texts of past sessions"},
	{"image-cache", "Image cache", true, "legacy image cache"},
	{"todos", "Todo lists (legacy)", true, "task lists of past sessions; no longer written by recent versions"},
	{"statsig", "Feature-flag cache (legacy)", true, "no longer written by recent versions"},
	{"logs", "Logs (legacy)", true, "no longer written by recent versions"},
	{"cache", "Cache", true, "changelog and other caches; downloaded again when needed"},
	{"tasks", "Background tasks", false, "output of background tasks; cleaned by Claude Code"},
	{"usage-data", "Usage data", false, "local usage records used by /usage"},
	{"backups", "Backups of ~/.claude.json", false, "keep: used to recover a corrupted ~/.claude.json"},
	{"plugins", "Plugins and marketplaces", false, "installed plugins; remove them with claude plugin uninstall"},
	{"ide", "IDE connections", false, "lock files of connected editors"},
	{"agent-memory", "Agent memory", false, "memory of your agents: keep"},
	{"skills", "Your skills", false, "keep"},
	{"agents", "Your agents", false, "keep"},
	{"commands", "Your commands", false, "keep"},
}

var neverList = map[string]bool{".credentials.json": true, "settings.json": true, "settings.local.json": true, "CLAUDE.md": true, "keybindings.json": true}

// Catalog lists every known location for this machine (sizes are filled by Measure).
func Catalog(env paths.Env) []Entry {
	var out []Entry
	known := map[string]bool{}
	for _, d := range codeDirs {
		p := filepath.Join(env.ClaudeDir, d.key)
		known[d.key] = true
		if _, err := os.Stat(p); err == nil {
			out = append(out, Entry{Key: "code:" + d.key, Group: "Claude Code", Label: d.label, Path: p, Safe: d.safe, Note: d.note})
		}
	}
	// unknown folders are shown (never cleanable) so nothing is hidden
	if ents, err := os.ReadDir(env.ClaudeDir); err == nil {
		for _, e := range ents {
			if e.IsDir() && !known[e.Name()] && !strings.HasPrefix(e.Name(), ".") {
				out = append(out, Entry{Key: "code:" + e.Name(), Group: "Claude Code", Label: e.Name(), Path: filepath.Join(env.ClaudeDir, e.Name()), Note: "not documented: kept as is"})
			}
		}
	}
	for _, f := range []struct{ name, label, note string }{
		{"history.jsonl", "Prompt history", "the ↑ history of your prompts"},
	} {
		p := filepath.Join(env.ClaudeDir, f.name)
		if _, err := os.Stat(p); err == nil {
			out = append(out, Entry{Key: "code:" + f.name, Group: "Claude Code", Label: f.label, Path: p, Note: f.note, IsFile: true})
		}
	}
	if _, err := os.Stat(env.ClaudeJSON); err == nil {
		out = append(out, Entry{Key: "code:claude.json", Group: "Claude Code", Label: "~/.claude.json (account, MCP, per-project state)", Path: env.ClaudeJSON, IsFile: true,
			Note: "grows with every project; above ~8 MiB Claude Code slows down. Never delete it"})
	}
	for _, s := range env.StaleCopies() {
		out = append(out, Entry{Key: "stale:" + s, Group: "Claude Code", Label: "Stale copy " + filepath.Base(s), Path: s, IsFile: true, Safe: true,
			Note: "leftover copy of ~/.claude.json; may hold old tokens"})
	}
	for _, t := range ScratchRoots() {
		out = append(out, Entry{Key: "tmp:" + t, Group: "Scratchpads", Label: "Claude Code temporary files and scratchpads", Path: t, Safe: true,
			Note: "per-session scratchpads and task outputs; only folders untouched for an hour are removed"})
	}
	for _, d := range desktopDirs(env) {
		out = append(out, d)
	}
	if c, err := os.UserCacheDir(); err == nil {
		p := filepath.Join(c, "catude")
		if _, err := os.Stat(p); err == nil {
			out = append(out, Entry{Key: "catude:cache", Group: "catude", Label: "catude cache and edit backups", Path: p, Note: "backups taken before every edit; trash of removed sessions"})
		}
	}
	return out
}

// ScratchRoots are the temporary folders Claude Code uses for scratchpads and task output.
func ScratchRoots() []string {
	var out []string
	add := func(p string) {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			out = append(out, p)
		}
	}
	if d := os.Getenv("CLAUDE_CODE_TMPDIR"); d != "" {
		add(d)
	}
	// documented locations (code.claude.com/docs/en/claude-directory): %TEMP%\claude on Windows,
	// /tmp/claude-<uid> on Linux, /private/tmp/claude-<uid> on macOS, or the same under $TMPDIR
	if runtime.GOOS == "windows" {
		add(filepath.Join(os.TempDir(), "claude"))
		return out
	}
	name := "claude-" + strconv.Itoa(os.Getuid())
	seen := map[string]bool{}
	for _, base := range []string{os.Getenv("TMPDIR"), "/tmp", "/private/tmp"} {
		if base == "" {
			continue
		}
		p := filepath.Join(base, name)
		if real, err := filepath.EvalSymlinks(p); err == nil && !seen[real] {
			seen[real] = true
			add(p)
		}
	}
	return out
}

func desktopDirs(env paths.Env) []Entry {
	var out []Entry
	seen := map[string]bool{}
	for _, f := range env.DesktopConfigs() {
		base := filepath.Dir(f.Path)
		if seen[base] {
			continue
		}
		seen[base] = true
		tag := f.Label
		for _, d := range []def{
			{"logs", "Claude Desktop logs", true, "mcp*.log and main.log; useful to debug MCP servers"},
			{"Cache", "Claude Desktop web cache", true, "Electron cache; rebuilt when needed (close Claude Desktop first)"},
			{"Code Cache", "Claude Desktop code cache", true, "Electron cache; rebuilt when needed"},
			{"GPUCache", "Claude Desktop GPU cache", true, "Electron cache; rebuilt when needed"},
			{"Claude Extensions", "Claude Desktop extensions", false, "installed .mcpb/.dxt extensions: manage them in Settings → Extensions"},
			{"vm_bundles", "Cowork virtual machine", false, "can take many GB; remove it from Claude Desktop, not by hand"},
		} {
			p := filepath.Join(base, d.key)
			if _, err := os.Stat(p); err == nil {
				out = append(out, Entry{Key: "desktop:" + p, Group: "Claude Desktop", Label: d.label + " · " + tag, Path: p, Safe: d.safe, Note: d.note})
			}
		}
	}
	// Desktop logs on macOS live elsewhere
	if env.GOOS == "darwin" {
		p := filepath.Join(env.Home, "Library", "Logs", "Claude")
		if _, err := os.Stat(p); err == nil {
			out = append(out, Entry{Key: "desktop:" + p, Group: "Claude Desktop", Label: "Claude Desktop logs", Path: p, Safe: true, Note: "mcp*.log and main.log"})
		}
	}
	return out
}

// Limits keep measuring fast on huge folders.
var (
	MaxFiles   = 2_000_000
	MaxMeasure = 90 * time.Second
)

// Measure fills size, file count and last change of an entry.
func Measure(ctx context.Context, e Entry) Entry {
	e.Measured = true
	if e.IsFile {
		fi, err := os.Stat(e.Path)
		if err != nil {
			e.Err = err.Error()
			return e
		}
		e.Bytes, e.Files, e.Newest = fi.Size(), 1, fi.ModTime()
		return e
	}
	ctx, cancel := context.WithTimeout(ctx, MaxMeasure)
	defer cancel()
	stop := errors.New("stop")
	err := filepath.WalkDir(e.Path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entries are skipped, not fatal
		}
		if ctx.Err() != nil || e.Files >= MaxFiles {
			e.Partial = true
			return stop
		}
		if d.Type()&fs.ModeSymlink != 0 || d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		e.Bytes += fi.Size()
		e.Files++
		if fi.ModTime().After(e.Newest) {
			e.Newest = fi.ModTime()
		}
		return nil
	})
	if err != nil && err != stop {
		e.Err = err.Error()
	}
	return e
}

// CleanResult says what a clean removed.
type CleanResult struct {
	Removed int
	Freed   int64
	Kept    int // too recent or in use
}

// Clean removes the contents of a safe entry that are older than MinAge. Folders directly under a
// scratchpad root are removed as a whole only when nothing inside changed for MinAge.
func Clean(e Entry) (CleanResult, error) {
	var r CleanResult
	if !e.Safe {
		return r, fmt.Errorf("%s is not safe to clean", e.Label)
	}
	if base := filepath.Base(e.Path); neverList[base] {
		return r, fmt.Errorf("refusing to remove %s", base)
	}
	// the entry itself must be a real folder or file: a link or junction could point anywhere
	if fi, err := os.Lstat(e.Path); err != nil {
		return r, err
	} else if fi.Mode()&(fs.ModeSymlink|fs.ModeIrregular|fs.ModeDevice|fs.ModeNamedPipe|fs.ModeSocket) != 0 {
		return r, fmt.Errorf("%s is a link or a special file: not cleaning through it", e.Path)
	}
	cut := time.Now().Add(-MinAge)
	if e.IsFile {
		fi, err := os.Stat(e.Path)
		if err != nil {
			return r, err
		}
		if fi.ModTime().After(cut) {
			r.Kept++
			return r, nil
		}
		if err := os.Remove(e.Path); err != nil {
			return r, err
		}
		r.Removed, r.Freed = 1, fi.Size()
		return r, nil
	}
	ents, err := os.ReadDir(e.Path)
	if err != nil {
		return r, err
	}
	if strings.HasPrefix(e.Key, "tmp:") {
		// <root>/<project hash>/<session>/…: judge each session folder on its own, so the folders
		// of old sessions go even when a running session shares the project folder.
		for _, c := range ents {
			if !c.IsDir() || c.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
				continue
			}
			sub := Entry{Key: "dir", Label: e.Label, Path: filepath.Join(e.Path, c.Name()), Safe: true}
			x, _ := Clean(sub)
			r.Removed, r.Freed, r.Kept = r.Removed+x.Removed, r.Freed+x.Freed, r.Kept+x.Kept
			os.Remove(sub.Path) // only succeeds when it became empty
		}
		return r, nil
	}
	for _, c := range ents {
		p := filepath.Join(e.Path, c.Name())
		// links, Windows junctions (reported as irregular), devices and pipes are never followed or deleted
		if c.Type()&(fs.ModeSymlink|fs.ModeIrregular|fs.ModeDevice|fs.ModeNamedPipe|fs.ModeSocket) != 0 || neverList[c.Name()] {
			r.Kept++
			continue
		}
		size, newest, n := treeInfo(p)
		if newest.After(cut) {
			r.Kept++
			continue
		}
		if err := os.RemoveAll(p); err != nil {
			r.Kept++
			continue
		}
		r.Removed += n
		r.Freed += size
	}
	return r, nil
}

func treeInfo(p string) (size int64, newest time.Time, n int) {
	filepath.WalkDir(p, func(q string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil // folder times change whenever a file is added: only files count
		}
		if fi, err := d.Info(); err == nil {
			if fi.ModTime().After(newest) {
				newest = fi.ModTime()
			}
			size += fi.Size()
			n++
		}
		return nil
	})
	return
}

// Retention reads cleanupPeriodDays (default 30) and counts transcripts that Claude Code will delete
// within the next week.
type Retention struct {
	Days      int
	Explicit  bool // set in settings
	Expiring  int  // transcripts that expire within 7 days
	Oldest    time.Time
	Transcrip int
}

func ReadRetention(env paths.Env) Retention {
	r := Retention{Days: 30}
	for _, f := range env.SettingsFiles() {
		if f.Scope != paths.ScopeUser {
			continue
		}
		b, err := os.ReadFile(f.Path)
		if err != nil {
			continue
		}
		var s struct {
			CleanupPeriodDays *int `json:"cleanupPeriodDays"`
		}
		if json.Unmarshal(b, &s) == nil && s.CleanupPeriodDays != nil {
			r.Days, r.Explicit = *s.CleanupPeriodDays, true
		}
	}
	cut := time.Now().AddDate(0, 0, -(r.Days - 7))
	filepath.WalkDir(env.ProjectsDir(), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".jsonl") || strings.Contains(filepath.ToSlash(p), "/subagents/") {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		r.Transcrip++
		if fi.ModTime().Before(cut) {
			r.Expiring++
		}
		if r.Oldest.IsZero() || fi.ModTime().Before(r.Oldest) {
			r.Oldest = fi.ModTime()
		}
		return nil
	})
	return r
}

// Sort orders entries by group then size.
func Sort(es []Entry) {
	order := map[string]int{"Claude Code": 0, "Scratchpads": 1, "Claude Desktop": 2, "catude": 3}
	sort.SliceStable(es, func(i, j int) bool {
		if order[es[i].Group] != order[es[j].Group] {
			return order[es[i].Group] < order[es[j].Group]
		}
		return es[i].Bytes > es[j].Bytes
	})
}
