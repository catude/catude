// Package claudecfg lists the configuration file of every Claude app found on the machine — Claude Code
// (user, project, local, managed, extra configuration folders, ~/.claude.json) and Claude Desktop (every
// install, including the Microsoft Store one) — shows what each contains and changes single values safely.
//
// How the apps relate (Claude Code docs, desktop.md, mcp.md and settings-reference.md):
//
//   - Claude Code in a terminal reads its own files only. It does not read Claude Desktop's file.
//   - The "Code" tab inside Claude Desktop uses Claude Code's files (settings.json, CLAUDE.md, skills,
//     ~/.claude.json, .mcp.json) AND the MCP servers of Claude Desktop's file.
//   - Connectors, skills and plugins enabled on claude.ai reach both through the account, not files.
//
// Safety: values that look like secrets are masked and cannot be changed inline; managed (policy) files
// and ~/.claude.json are read-only here; every write is preceded by a backup, keeps the key order of the
// file, and is validated by parsing the result before it replaces the original.
package claudecfg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/redact"
)

// Target is one configuration file of one Claude app.
type Target struct {
	ID       string
	App      string // Claude Code | Claude Desktop
	Label    string
	Path     string
	Exists   bool
	ReadOnly bool   // cannot be changed from catude (policy file, app state)
	Note     string // who reads it / what to know before editing
	Restart  string // what to restart for a change to apply
	RefApp   string // which documented settings apply (see reference.go)
	State    bool   // an app state file the app rewrites while running (~/.claude.json)
}

// Targets lists every configuration file for this machine, existing or not (so it can be created).
func Targets(env paths.Env) []Target {
	var out []Target
	add := func(t Target) {
		_, err := os.Stat(t.Path)
		t.Exists = err == nil
		out = append(out, t)
	}
	codeNote := "Claude Code in the terminal, the IDE extensions and the Code tab of Claude Desktop"
	add(Target{ID: "code-user", App: "Claude Code", Label: "Your settings (all projects)", Path: filepath.Join(env.ClaudeDir, "settings.json"),
		Note: "read by " + codeNote, Restart: "new Claude Code sessions", RefApp: "code-settings"})
	if env.ProjectDir != "" {
		add(Target{ID: "code-project", App: "Claude Code", Label: "This project (shared with the team)", Path: filepath.Join(env.ProjectDir, ".claude", "settings.json"),
			Note: "checked into the repository: applies to everyone who opens this project", Restart: "new Claude Code sessions in this project", RefApp: "code-settings"})
		add(Target{ID: "code-local", App: "Claude Code", Label: "This project (only you)", Path: filepath.Join(env.ProjectDir, ".claude", "settings.local.json"),
			Note: "your private overrides for this project (git-ignored)", Restart: "new Claude Code sessions in this project", RefApp: "code-settings"})
	}
	for _, d := range env.ConfigDirs() {
		if d.Backup || filepath.Clean(d.Path) == filepath.Clean(env.ClaudeDir) {
			continue
		}
		add(Target{ID: "code-dir:" + d.Path, App: "Claude Code", Label: "Configuration folder " + d.Label, Path: filepath.Join(d.Path, "settings.json"),
			Note: "a separate CLAUDE_CONFIG_DIR profile with its own login", Restart: "new sessions started with that folder", RefApp: "code-settings"})
	}
	if p := env.ManagedSettings(); p != "" {
		add(Target{ID: "code-managed", App: "Claude Code", Label: "Organisation policy (managed)", Path: p, ReadOnly: true,
			Note: "set by your administrator: it wins over every other file and cannot be changed here", RefApp: "code-settings"})
	}
	add(Target{ID: "code-state", App: "Claude Code", Label: "Global config ~/.claude.json (IDE, copy, editor…)", Path: env.ClaudeJSON, State: true, RefApp: "code-global", Restart: "new Claude Code sessions",
		Note: "Claude Code rewrites this file while it runs, so catude changes its documented options only while Claude Code is closed. MCP servers: claude mcp add/remove"})
	seen := map[string]bool{}
	for _, f := range env.DesktopConfigs() {
		if seen[f.Path] {
			continue
		}
		seen[f.Path] = true
		note := "read by the Claude Desktop chat app and by its Code tab; Claude Code in the terminal does NOT read it"
		// the Store copy is recognised by its path with either separator (WSL reads it through /mnt/c)
		if strings.Contains(strings.ToLower(filepath.ToSlash(f.Path)), "/packages/") || strings.Contains(strings.ToLower(f.Path), `\packages\`) {
			note = "the copy the Microsoft Store app really reads (its %APPDATA% is redirected here) · " + note
		}
		add(Target{ID: "desktop:" + f.Path, App: "Claude Desktop", Label: f.Label, Path: f.Path, Note: note, Restart: "Claude Desktop (quit it completely and open it again)", RefApp: "desktop"})
	}
	if p := desktopPolicyPath(); p != "" {
		t := Target{ID: "desktop-policy", App: "Claude Desktop", Label: "Organisation policy (all documented keys)", Path: p, ReadOnly: true, RefApp: "desktop-policy",
			Note: "what an administrator can set for Claude Desktop (" + desktopPolicyWhere() + "). Read-only here: other Claude Desktop options live in the app, under Settings"}
		_, err := os.Stat(t.Path)
		t.Exists = err == nil || strings.HasPrefix(t.Path, `HK`)
		out = append(out, t)
	}
	return out
}

// Entry is one setting of a target: a value in the file, or a documented setting not set yet.
type Entry struct {
	Path       []string
	Key        string // dotted path
	Type       string // text | number | yes/no | empty | list | object
	Value      string // display value (masked when secret); "(not set)" when absent
	Set        bool   // present in the file
	Secret     bool
	Editable   bool     // can be changed inline
	Choices    []string // documented allowed values
	Help       string   // what the setting does (documentation)
	Category   string
	Default    string // documented default, when there is one
	Scope      string // which files may hold it
	Deprecated bool
	Documented bool
	Env        bool // an environment variable inside "env"
	// effective settings only: the file the value comes from, the files it overrides, and whether a
	// list was combined from several files
	From      string
	Overrides []string
	Merged    bool
}

// MaxEntries bounds what is listed from very large files.
const MaxEntries = 600

// NotSet is the value shown for documented settings absent from the file.
const NotSet = "(not set)"

func (e *Entry) document(r *Ref) {
	if r == nil {
		return
	}
	e.Documented, e.Help, e.Category, e.Default, e.Scope, e.Deprecated = true, r.Desc, r.Category, r.Default, r.Scope, r.Deprecated
	if r.Type == "enum" {
		e.Choices = r.Values
	}
}

// Read lists the values of a target. A missing file gives no entries and no error.
func Read(t Target) ([]Entry, error) {
	if t.ID == "desktop-policy" {
		return readPolicy(t), nil
	}
	fi, err := os.Stat(t.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() || fi.Size() > 32<<20 { // checked before reading: a link to a device never ends
		return nil, errors.New("not a regular file, or larger than 32 MiB")
	}
	b, err := os.ReadFile(t.Path)
	if err != nil {
		return nil, err
	}
	root, err := parse(b)
	if err != nil {
		return nil, fmt.Errorf("not valid JSON: %w", err)
	}
	var out []Entry
	var walk func(n *node, path []string, depth int)
	walk = func(n *node, path []string, depth int) {
		if len(out) >= MaxEntries {
			return
		}
		key := strings.Join(path, ".")
		r := lookup(t, key)
		// ~/.claude.json: only its documented options and MCP servers are worth listing
		if t.State && depth == 1 && path[0] != "mcpServers" && r == nil {
			return
		}
		descend := n.kind == 'o' && depth < 6
		if t.State && depth >= 1 && path[0] != "mcpServers" {
			descend = false
		}
		if depth >= 1 && r != nil && r.Type == "object" && path[0] != "env" && !hasChildren(t, key) {
			descend = false // a documented object (hooks.PreToolUse, statusLine…) is shown as one setting
		}
		if descend {
			for i, k := range n.keys {
				walk(n.vals[i], append(append([]string{}, path...), k), depth+1)
			}
			if len(n.keys) > 0 || len(path) == 0 {
				return
			}
		}
		if len(path) == 0 {
			return
		}
		e := Entry{Path: path, Key: key, Type: n.typeName(), Value: n.text(), Set: true, Category: "Other"}
		t.RefApp = t.refApp()
		leaf := path[len(path)-1]
		envArea := containsAny(path, "env")
		inSecretArea := containsAny(path, "headers", "apiKeyHelper", "oauthAccount", "customApiKeyResponses") ||
			(path[0] == "env" && envSecret(leaf)) || (envArea && path[0] != "env")
		if n.kind == 's' && !redact.IsEnvRef(n.str) && n.str != "" && (redact.KeyIsSecret(leaf) || redact.LooksSecret(n.str) || inSecretArea) {
			e.Secret, e.Value = true, redact.Mask
		} else if n.kind == 's' {
			e.Value = redact.Text(n.str)
		} else if n.kind == 'a' {
			var parts []string
			for _, item := range n.vals {
				if item.kind == 's' {
					parts = append(parts, item.str)
				}
			}
			if len(parts) == len(n.vals) && len(parts) > 0 {
				e.Value = "[" + strings.Join(redact.Args(parts), ", ") + "]"
			}
			e.Value = redact.Text(e.Value)
		}
		e.document(r)
		e.Env = t.RefApp == "code-settings" && path[0] == "env" && len(path) == 2
		if e.Env {
			e.Category = "Environment variables"
		}
		scalar := n.kind == 's' || n.kind == 'n' || n.kind == 'b'
		e.Editable = scalar && !t.ReadOnly && !e.Secret && !inSecretArea && !containsAny(path, "hooks", "statusLine")
		if t.State { // only documented options of ~/.claude.json
			e.Editable = e.Editable && r != nil && r.App == "code-global"
		}
		if e.Env && r != nil && r.Type == "bool" {
			e.Type = "yes/no"
		}
		out = append(out, e)
	}
	walk(root, nil, 0)
	return out, nil
}

func readPolicy(t Target) []Entry {
	set := readDesktopPolicy()
	var out []Entry
	for _, r := range Refs("desktop-policy") {
		r := r
		e := Entry{Path: []string{r.Key}, Key: r.Key, Type: entryType(r), Value: NotSet}
		e.document(&r)
		if v, ok := set[r.Key]; ok {
			e.Value, e.Set = redact.Text(v), true
			if redact.KeyIsSecret(r.Key) || redact.LooksSecret(v) {
				e.Value, e.Secret = redact.Mask, true
			}
		}
		out = append(out, e)
	}
	return out
}

func containsAny(path []string, names ...string) bool {
	for _, p := range path {
		for _, n := range names {
			if strings.EqualFold(p, n) {
				return true
			}
		}
	}
	return false
}

// Documented lists every documented setting of the target's app that is not in the file yet, so it
// can be seen and set. For Claude Code settings files this includes every environment variable
// (settable under "env").
func Documented(t Target, have []Entry) []Entry {
	t.RefApp = t.refApp()
	if t.RefApp == "" || t.ID == "desktop-policy" {
		return nil
	}
	present := map[string]bool{}
	for _, e := range have {
		present[e.Key] = true
		for i := 1; i < len(e.Path); i++ { // a parent object that exists in the file
			present[strings.Join(e.Path[:i], ".")] = true
		}
	}
	var out []Entry
	add := func(r Ref, key string, env bool) {
		if present[key] || strings.ContainsAny(r.Key, "<[") || !allowedIn(r, t) {
			return
		}
		e := Entry{Path: strings.Split(key, "."), Key: key, Type: entryType(r), Value: NotSet, Env: env}
		e.document(&r)
		scalar := r.Type == "bool" || r.Type == "number" || r.Type == "string" || r.Type == "enum"
		e.Editable = scalar && !t.ReadOnly && !(env && envSecret(r.Key)) && !strings.HasPrefix(key, "hooks") && !strings.HasPrefix(key, "statusLine")
		if env {
			e.Path, e.Category = []string{"env", r.Key}, "Environment variables"
		}
		out = append(out, e)
	}
	for _, r := range Refs(t.RefApp) {
		add(r, r.Key, false)
	}
	if t.RefApp == "code-settings" {
		for _, r := range Refs("code-env") {
			add(r, "env."+r.Key, true)
		}
	}
	return out
}

// Suggestions is the former name of Documented.
func Suggestions(t Target, have []Entry) []Entry { return Documented(t, have) }

// Set changes one value and returns the backup path. The new value is typed after the entry:
// yes/no accepts true/false/yes/no, number accepts digits, anything else is text. Environment
// variables are always stored as text; a yes/no variable is turned off by removing it, because some
// variables treat any value (even "0") as on.
func Set(t Target, e Entry, value string) (backup string, err error) {
	if t.ReadOnly {
		return "", errors.New(t.Label + " cannot be changed from catude")
	}
	if !e.Editable {
		return "", errors.New(e.Key + " cannot be changed inline: press e to open the file in your editor")
	}
	value = strings.TrimSpace(value)
	if e.Key == "permissions.defaultMode" && value == "bypassPermissions" {
		return "", errors.New("bypassPermissions turns every permission prompt off: set it in your editor (e) if you really mean it")
	}
	if err := stateClosed(t); err != nil {
		return "", err
	}
	var nv *node
	switch e.Type {
	case "yes/no":
		on := false
		switch strings.ToLower(value) {
		case "true", "yes", "y", "on", "1", "si", "sí":
			on = true
		case "false", "no", "n", "off", "0":
		default:
			return "", errors.New("type true or false")
		}
		if e.Env {
			if !on {
				if !e.Set {
					return "", nil
				}
				return Unset(t, e)
			}
			nv = &node{kind: 's', str: "1"}
		} else {
			nv = &node{kind: 'b', b: on}
		}
	case "number":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return "", errors.New("type a number")
		}
		if e.Env {
			nv = &node{kind: 's', str: value}
		} else {
			nv = &node{kind: 'n', num: json.Number(value)}
		}
	default:
		if redact.LooksSecret(value) {
			return "", errors.New("that looks like a secret: keep secrets in environment variables or your password manager, not in settings files")
		}
		nv = &node{kind: 's', str: value}
	}
	if len(e.Choices) > 0 {
		ok := false
		for _, c := range e.Choices {
			ok = ok || c == value
			if i := strings.Index(c, "<"); i > 0 && strings.HasPrefix(value, c[:i]) && len(value) > i {
				ok = true // e.g. theme custom:<slug>
			}
		}
		if !ok {
			return "", fmt.Errorf("%s must be one of: %s", e.Key, strings.Join(e.Choices, ", "))
		}
	}
	if e.Key == "cleanupPeriodDays" || e.Key == "desktopSessionCleanupPeriodDays" {
		if n, err := strconv.Atoi(value); err != nil || n < 1 {
			return "", errors.New("must be a whole number of days, at least 1")
		}
	}
	return write(t, func(root *node) error { return root.set(e.Path, nv) })
}

// stateClosed refuses to change an app state file while the app runs.
func stateClosed(t Target) error {
	if !t.State {
		return nil
	}
	if n := CodeRunning(); n > 0 {
		return fmt.Errorf("close Claude Code and Claude Desktop first (%d running): they rewrite ~/.claude.json and would undo the change. Or change it with /config inside Claude Code", n)
	}
	return nil
}

// Unset removes a setting from the file, so the app goes back to its default.
func Unset(t Target, e Entry) (string, error) {
	if t.ReadOnly {
		return "", errors.New(t.Label + " cannot be changed from catude")
	}
	if !e.Set {
		return "", errors.New(e.Key + " is not set in this file")
	}
	if err := stateClosed(t); err != nil {
		return "", err
	}
	if !e.Editable && !(e.Documented && (e.Type == "list" || e.Type == "object") && !t.State) {
		return "", errors.New(e.Key + " cannot be removed here: press e to open the file in your editor")
	}
	return write(t, func(root *node) error {
		if root.at(e.Path) == nil {
			return errors.New(e.Key + " is no longer in the file")
		}
		root.del(e.Path)
		return nil
	})
}

// write applies a change to the file: read, change, validate, back up, replace.
func write(t Target, change func(*node) error) (string, error) {
	b, err := os.ReadFile(t.Path)
	root := &node{kind: 'o'}
	existed := err == nil
	if existed && len(strings.TrimSpace(string(b))) > 0 {
		root, err = parse(b)
		if err != nil {
			return "", fmt.Errorf("%s is not valid JSON, fix it in your editor first: %w", filepath.Base(t.Path), err)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := change(root); err != nil {
		return "", err
	}
	out := root.encode()
	if _, err := parse(out); err != nil {
		return "", fmt.Errorf("internal error: the result would not be valid JSON: %w", err)
	}
	backup := ""
	if existed {
		backup, err = Backup(t.Path)
		if err != nil {
			return "", fmt.Errorf("could not back up the file, nothing was changed: %w", err)
		}
	} else if err := os.MkdirAll(filepath.Dir(t.Path), 0o755); err != nil {
		return "", err
	}
	// the temporary file gets a random name (a repository cannot plant a link under a known name) and the
	// permissions of the file it replaces; a new settings file is private
	mode := os.FileMode(0o600)
	if fi, err := os.Lstat(t.Path); err == nil {
		if !fi.Mode().IsRegular() {
			return backup, fmt.Errorf("%s is a link or a special file: not changing it", filepath.Base(t.Path))
		}
		mode = fi.Mode().Perm()
	}
	f, err := os.CreateTemp(filepath.Dir(t.Path), ".catude-*.tmp")
	if err != nil {
		return backup, err
	}
	tmp := f.Name()
	_, werr := f.Write(out)
	cerr := f.Close()
	if werr != nil || cerr != nil {
		os.Remove(tmp)
		return backup, firstErr(werr, cerr)
	}
	os.Chmod(tmp, mode)
	if err := os.Rename(tmp, t.Path); err != nil {
		os.Remove(tmp)
		return backup, err
	}
	return backup, nil
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// Backup copies a file to <user cache dir>/catude/backups and returns the copy's path.
func Backup(p string) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "catude", "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	dst := filepath.Join(dir, time.Now().Format("20060102-150405")+"-"+filepath.Base(p))
	return dst, os.WriteFile(dst, b, 0o600)
}
