package claudecfg

import (
	"sort"
	"strings"

	"github.com/catude/catude/internal/redact"
)

// Effective settings: for every key set in any Claude Code settings file of this project, the value Claude
// Code uses and where it comes from, following the documented rules (code.claude.com/docs/en/settings,
// "Settings precedence" and "Lists merge instead of overriding"):
//
//   - a key set in several files takes the value of the highest level: managed, project local, shared
//     project, user (the command line, --settings, sits between managed and local but is per session);
//   - list keys such as permissions.allow combine the lists of every file;
//   - fallbackModel never merges: the whole value comes from the highest file that sets it;
//   - modelPicker never merges and is read only from managed and user settings;
//   - availableModels from managed settings applies as-is; otherwise lists merge as usual.
//
// Objects are followed key by key (env, permissions, sandbox…), so env.FOO is resolved on its own.

// EffectiveID is the id of the virtual "effective settings" target.
const EffectiveID = "code-effective"

// EffectiveTarget is the read-only target that shows the effective settings.
func EffectiveTarget() Target {
	return Target{ID: EffectiveID, App: "Claude Code", Label: "Effective settings (what wins)", ReadOnly: true,
		Exists: true, RefApp: "code-settings",
		Note: "the value Claude Code uses for each key set in any of the settings files above, and which file it comes from. Higher levels win: organisation policy, then this project only you, then this project shared, then your settings; lists combine"}
}

// level orders the settings files, highest precedence first.
func level(t Target) (int, string) {
	switch t.ID {
	case "code-managed":
		return 0, "policy"
	case "code-local":
		return 1, "project (only you)"
	case "code-project":
		return 2, "project (shared)"
	case "code-user":
		return 3, "your settings"
	}
	return -1, ""
}

type leaf struct {
	scope string
	node  *node
}

// Effective resolves the settings of the given targets (other targets are ignored).
func Effective(targets []Target) []Entry {
	type src struct {
		lvl   int
		scope string
		root  *node
	}
	var srcs []src
	for _, t := range targets {
		lv, scope := level(t)
		if lv < 0 {
			continue
		}
		b, err := readTarget(t)
		if err != nil || len(strings.TrimSpace(string(b))) == 0 {
			continue
		}
		root, err := parse(b)
		if err != nil || root.kind != 'o' {
			continue
		}
		srcs = append(srcs, src{lv, scope, root})
	}
	sort.SliceStable(srcs, func(i, j int) bool { return srcs[i].lvl < srcs[j].lvl })
	// every leaf key, with its value in each file (highest first)
	perKey := map[string][]leaf{}
	var order []string
	ref := Target{ID: "code-user", RefApp: "code-settings"}
	for _, s := range srcs {
		var walk func(path []string, n *node)
		walk = func(path []string, n *node) {
			key := strings.Join(path, ".")
			if n.kind == 'o' && (len(path) == 0 || !isLeafObject(path) && (path[0] == "env" || hasChildren(ref, key) || lookup(ref, key) == nil)) && len(path) < 6 {
				for i, k := range n.keys {
					walk(append(append([]string{}, path...), k), n.vals[i])
				}
				return
			}
			if _, seen := perKey[key]; !seen {
				order = append(order, key)
			}
			perKey[key] = append(perKey[key], leaf{s.scope, n})
		}
		walk(nil, s.root)
	}
	var out []Entry
	for _, key := range order {
		ls := perKey[key]
		e := Entry{Path: strings.Split(key, "."), Key: key, Set: true, Category: "Other"}
		e.document(lookup(ref, key))
		if strings.HasPrefix(key, "env.") {
			e.Category = "Environment variables"
		}
		top := ls[0]
		e.Type = top.node.typeName()
		isList := top.node.kind == 'a'
		switch {
		case key == "modelPicker":
			var kept []leaf
			for _, l := range ls {
				if l.scope == "policy" || l.scope == "your settings" {
					kept = append(kept, l)
				}
			}
			if len(kept) == 0 {
				e.Value, e.From = "(ignored here: only read from policy or your settings)", ls[0].scope
				out = append(out, e)
				continue
			}
			ls = kept
			fallthrough
		case !isList || key == "fallbackModel" || (key == "availableModels" && ls[0].scope == "policy"):
			e.Value, e.From = display(ls[0].node, key), ls[0].scope
			for _, l := range ls[1:] {
				e.Overrides = append(e.Overrides, l.scope+" ("+display(l.node, key)+")")
			}
		default: // lists combine
			seen := map[string]bool{}
			var items, from []string
			for _, l := range ls {
				from = append(from, l.scope)
				for _, it := range l.node.vals {
					c := compact(it)
					if !seen[c] {
						seen[c] = true
						items = append(items, strings.Trim(c, `"`))
					}
				}
			}
			e.Value = "[" + strings.Join(items, ", ") + "]"
			e.From = strings.Join(from, " + ")
			e.Merged = len(ls) > 1
		}
		e.Secret = e.Value == redact.Mask
		out = append(out, e)
	}
	return out
}

// WithEffective places the effective-settings view among Claude Code's files, after the settings files
// it summarises (before the global config, which is not part of the settings stack).
func WithEffective(ts []Target) []Target {
	pos := -1
	for i, t := range ts {
		if lv, _ := level(t); lv >= 0 || strings.HasPrefix(t.ID, "code-dir:") {
			pos = i + 1
		}
	}
	if pos < 0 {
		return ts
	}
	out := append(append(append([]Target{}, ts[:pos]...), EffectiveTarget()), ts[pos:]...)
	return out
}

func display(n *node, key string) string {
	leafName := key[strings.LastIndex(key, ".")+1:]
	if n.kind == 's' && !redact.IsEnvRef(n.str) && (redact.KeyIsSecret(leafName) || redact.LooksSecret(n.str) || (strings.HasPrefix(key, "env.") && envSecret(leafName))) {
		return redact.Mask
	}
	if n.kind == 's' {
		return redact.Text(n.str)
	}
	return redact.Text(compact(n))
}
