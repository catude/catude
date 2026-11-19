package claudecfg

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// Configuration presets: small groups of settings for one goal (privacy, protecting secrets, lower cost,
// notifications…). Every preset comes from a public source that is quoted word for word: Anthropic's
// documentation and example settings, or a configuration an organisation published with its reasons.
// Every key is a documented setting (tests check it against reference.json).
//
//go:embed config-presets.json
var presetsJSON []byte

// Source is where a preset comes from, with the exact text that supports it.
type Source struct {
	URL       string `json:"url"`
	Quote     string `json:"quote"`
	Retrieved string `json:"retrieved"`
}

// Preset is a named group of settings.
type Preset struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Summary    string          `json:"summary"`
	Tags       []string        `json:"tags"`
	App        string          `json:"app"`
	Scope      string          `json:"scope"` // user | project: where it is meant to go
	Settings   json.RawMessage `json:"settings"`
	Risk       string          `json:"risk"` // low | medium | high
	RiskReason string          `json:"riskReason,omitempty"`
	By         string          `json:"by"`
	Note       string          `json:"note,omitempty"`
	Sources    []Source        `json:"sources"`
}

// Unsupported says why a preset does not fit this system ("" when it does).
func (p Preset) Unsupported() string {
	has := func(t string) bool {
		for _, x := range p.Tags {
			if x == t {
				return true
			}
		}
		return false
	}
	switch {
	case has("sandbox") && runtime.GOOS == "windows":
		return "the Claude Code sandbox does not run on native Windows (use it in WSL)"
	case has("windows") && runtime.GOOS != "windows":
		return "for Windows"
	case p.ID == "notify-desktop-macos" && runtime.GOOS != "darwin":
		return "for macOS"
	case p.ID == "notify-desktop-linux" && runtime.GOOS != "linux":
		return "for Linux"
	}
	return ""
}

var (
	presetsOnce sync.Once
	presets     []Preset
	presetsErr  error
)

// Presets returns the configuration presets, sorted by tag then name.
func Presets() ([]Preset, error) {
	presetsOnce.Do(func() {
		var doc struct {
			Presets []Preset `json:"presets"`
		}
		presetsErr = json.Unmarshal(presetsJSON, &doc)
		presets = doc.Presets
		sort.SliceStable(presets, func(i, j int) bool {
			a, b := firstTag(presets[i]), firstTag(presets[j])
			if a != b {
				return a < b
			}
			return presets[i].Name < presets[j].Name
		})
	})
	return presets, presetsErr
}

func firstTag(p Preset) string {
	if len(p.Tags) == 0 {
		return "~"
	}
	return p.Tags[0]
}

// Change is one effect of a preset on a file.
type Change struct {
	Key    string
	Before string // "" when not set
	After  string
	Same   bool // already so in the file
}

// Plan lists what applying a preset would change in a target, without writing anything.
func Plan(t Target, p Preset) ([]Change, error) {
	root, err := loadRoot(t)
	if err != nil {
		return nil, err
	}
	add, err := parse(p.Settings)
	if err != nil {
		return nil, fmt.Errorf("preset %s: %w", p.ID, err)
	}
	var out []Change
	var walk func(path []string, n *node)
	walk = func(path []string, n *node) {
		if n.kind == 'o' && !(len(path) > 0 && isLeafObject(path)) {
			for i, k := range n.keys {
				walk(append(append([]string{}, path...), k), n.vals[i])
			}
			return
		}
		cur := root.at(path)
		c := Change{Key: strings.Join(path, ".")}
		if cur != nil {
			c.Before = compact(cur)
		}
		merged := mergeNode(clone(cur), n)
		c.After = compact(merged)
		c.Same = cur != nil && c.Before == c.After
		out = append(out, c)
	}
	walk(nil, add)
	return out, nil
}

// isLeafObject marks objects that are one setting as a whole (a hook list, statusLine…), not a group.
func isLeafObject(path []string) bool {
	// modelPicker is one ordered value that never merges (settings docs, "Lists merge instead of overriding")
	return path[0] == "hooks" || path[0] == "statusLine" || path[0] == "modelPicker"
}

// ApplyPreset merges a preset into a target: objects are merged key by key, lists keep their items and
// get the new ones, other values are replaced. The file is backed up first. It returns the backup path.
func ApplyPreset(t Target, p Preset) (string, error) {
	if t.ReadOnly || t.State || !strings.HasPrefix(t.refApp(), "code-settings") {
		return "", errors.New("presets go into a Claude Code settings file (yours, the project's or local)")
	}
	add, err := parse(p.Settings)
	if err != nil {
		return "", fmt.Errorf("preset %s: %w", p.ID, err)
	}
	return write(t, func(root *node) error {
		merged := mergeNode(root, add)
		*root = *merged
		return nil
	})
}

func loadRoot(t Target) (*node, error) {
	b, err := readTarget(t)
	if err != nil || len(bytes.TrimSpace(b)) == 0 {
		return &node{kind: 'o'}, nil
	}
	return parse(b)
}

// mergeNode merges b into a (a may be nil) and returns the result.
func mergeNode(a, b *node) *node {
	if a == nil {
		return clone(b)
	}
	switch {
	case a.kind == 'o' && b.kind == 'o':
		for i, k := range b.keys {
			if cur := a.get(k); cur != nil {
				for j, ak := range a.keys {
					if ak == k {
						a.vals[j] = mergeNode(cur, b.vals[i])
					}
				}
			} else {
				a.keys, a.vals = append(a.keys, k), append(a.vals, clone(b.vals[i]))
			}
		}
		return a
	case a.kind == 'a' && b.kind == 'a':
		seen := map[string]bool{}
		for _, v := range a.vals {
			seen[compact(v)] = true
		}
		for _, v := range b.vals {
			if !seen[compact(v)] {
				a.vals = append(a.vals, clone(v))
			}
		}
		return a
	}
	return clone(b)
}

func clone(n *node) *node {
	if n == nil {
		return nil
	}
	c := *n
	c.keys = append([]string(nil), n.keys...)
	c.vals = make([]*node, len(n.vals))
	for i, v := range n.vals {
		c.vals[i] = clone(v)
	}
	return &c
}

// compact is a one-line JSON form of a value, for showing and comparing.
func compact(n *node) string {
	if n == nil {
		return ""
	}
	var b bytes.Buffer
	if err := json.Compact(&b, n.encode()); err != nil {
		return strings.TrimSpace(string(n.encode()))
	}
	return b.String()
}

// readTarget reads a settings file; a missing file is empty.
func readTarget(t Target) ([]byte, error) {
	fi, err := os.Stat(t.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() || fi.Size() > 32<<20 {
		return nil, errors.New("not a regular file, or larger than 32 MiB")
	}
	return os.ReadFile(t.Path)
}
