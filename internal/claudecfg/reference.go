package claudecfg

import (
	"context"
	_ "embed"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/catude/catude/internal/procs"
)

// Every documented setting of the Claude apps, extracted from Anthropic's documentation by
// scripts/build-settings.mjs (see that file for the sources). Apps:
//
//	code-settings   keys of Claude Code's settings.json files
//	code-global     keys of ~/.claude.json that people may change
//	code-env        environment variables Claude Code reads (also settable in settings.json "env")
//	desktop         keys of claude_desktop_config.json
//	desktop-policy  Claude Desktop policies set by an organisation (registry, plist, /etc)
//
//go:embed reference.json
var referenceJSON []byte

// Ref documents one setting.
type Ref struct {
	App          string   `json:"app"`
	Key          string   `json:"key"`
	Type         string   `json:"type"` // bool | number | string | enum | array | object
	Values       []string `json:"values,omitempty"`
	Default      string   `json:"default,omitempty"`
	Desc         string   `json:"desc"`
	Category     string   `json:"category"`
	Scope        string   `json:"scope,omitempty"` // which files may hold it (Claude Code)
	Deprecated   bool     `json:"deprecated,omitempty"`
	Undocumented bool     `json:"undocumented,omitempty"` // only in the published JSON schema
	pattern      *regexp.Regexp
}

var (
	refOnce    sync.Once
	refs       []Ref
	refByKey   map[string]*Ref // app + "\x00" + key
	refPattern []*Ref          // keys with <name> or [] parts
	refParent  map[string]bool // app + "\x00" + a key that has documented keys below it
	refUpdated string
)

func loadRefs() {
	refOnce.Do(func() {
		var doc struct {
			Updated  string `json:"updated"`
			Settings []Ref  `json:"settings"`
		}
		_ = json.Unmarshal(referenceJSON, &doc)
		refs, refUpdated = doc.Settings, doc.Updated
		refByKey, refParent = map[string]*Ref{}, map[string]bool{}
		for i := range refs {
			r := &refs[i]
			parts := strings.Split(r.Key, ".")
			for j := 1; j < len(parts); j++ {
				refParent[r.App+"\x00"+strings.Join(parts[:j], ".")] = true
			}
			if strings.ContainsAny(r.Key, "<[") {
				q := regexp.QuoteMeta(r.Key)
				q = regexp.MustCompile(`<[a-zA-Z_]+>`).ReplaceAllString(q, `[^.]+`)
				q = strings.ReplaceAll(q, `\[\]`, `(\.\d+)?`)
				r.pattern = regexp.MustCompile("^" + q + "$")
				refPattern = append(refPattern, r)
				continue
			}
			refByKey[r.App+"\x00"+r.Key] = r
		}
	})
}

// Refs returns the documented settings of one app, in category order.
func Refs(app string) []Ref {
	loadRefs()
	var out []Ref
	for _, r := range refs {
		if r.App == app {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// lookup finds what the documentation says about a key of a target's file.
func lookup(t Target, key string) *Ref {
	loadRefs()
	t.RefApp = t.refApp()
	if t.RefApp == "" {
		return nil
	}
	if t.RefApp == "code-settings" && strings.HasPrefix(key, "env.") {
		return refByKey["code-env\x00"+strings.TrimPrefix(key, "env.")]
	}
	if r := refByKey[t.RefApp+"\x00"+key]; r != nil {
		return r
	}
	for _, r := range refPattern {
		if r.App == t.RefApp && r.pattern.MatchString(key) {
			return r
		}
	}
	return nil
}

// allowedIn says whether a setting may be written in a target, from its documented scope.
func allowedIn(r Ref, t Target) bool {
	s := strings.ToLower(r.Scope)
	switch {
	case s == "" || strings.HasPrefix(s, "any") || s == "global config":
		return true
	case t.ID == "code-managed":
		return strings.Contains(s, "managed")
	case s == "managed":
		return false
	case strings.HasPrefix(t.ID, "code-user") || strings.HasPrefix(t.ID, "code-dir:"):
		return strings.Contains(s, "user")
	case t.ID == "code-local":
		return strings.Contains(s, "local")
	case t.ID == "code-project":
		return strings.Contains(s, "project")
	}
	return true
}

// envSecret says whether an environment variable holds a credential: those are never typed into
// settings files from catude.
func envSecret(name string) bool {
	n := strings.ToUpper(name)
	for _, w := range []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "CREDENTIAL", "AUTH_HEADER", "_AUTH", "COOKIE", "HEADERS"} {
		if strings.Contains(n, w) {
			return true
		}
	}
	return false
}

func entryType(r Ref) string {
	switch r.Type {
	case "bool":
		return "yes/no"
	case "number":
		return "number"
	case "array":
		return "list"
	case "object":
		return "object"
	}
	return "text"
}

// refApp is the documented app of a target, inferred from its id when not set.
func (t Target) refApp() string {
	switch {
	case t.RefApp != "":
		return t.RefApp
	case t.ID == "code-state":
		return "code-global"
	case t.ID == "desktop-policy":
		return "desktop-policy"
	case strings.HasPrefix(t.ID, "code-"):
		return "code-settings"
	case strings.HasPrefix(t.ID, "desktop:"):
		return "desktop"
	}
	return ""
}

// CodeRunning counts the running Claude Code and Claude Desktop processes: while any runs, it may
// rewrite ~/.claude.json and undo a change made underneath it.
var CodeRunning = func() int {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ps, err := procs.List(ctx)
	if err != nil {
		return 0
	}
	n := 0
	for _, p := range ps {
		if procs.IsClaude(p) {
			n++
		}
	}
	return n
}

// hasChildren says whether the documentation lists keys below this one (so its object is walked).
func hasChildren(t Target, key string) bool {
	loadRefs()
	return refParent[t.refApp()+"\x00"+key]
}

// RefUpdated is when the documented settings were last extracted from the docs.
func RefUpdated() string {
	loadRefs()
	return refUpdated
}
