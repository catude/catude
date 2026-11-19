package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/profiles"
)

// wizard creates or edits a work profile in two steps:
//
//	1 pick   fzf-like checklist of every skill, agent, command, MCP server and plugin on this machine:
//	         type to filter, space to toggle, tab to toggle every visible item, enter to continue
//	2 name   name, model and launch mode, then save (built, audited and installed in the background)
type wizard struct {
	step    int // 0 pick, 1 name
	editing string
	items   []wizItem
	sel     map[string]bool
	query   string
	cursor  int
	name    string
	model   int // index in wizModels
	excl    bool
	field   int  // step 1: 0 name, 1 model, 2 mode
	saveAll bool // started from "save my current setup"
	err     string
}

type wizItem struct {
	kind  inventory.Kind
	name  string
	desc  string
	scope string
	cost  int // ≈ tokens always loaded (skills: name + description)
}

var wizModels = []string{"(keep yours)", "opus", "sonnet", "haiku"}

var wizKinds = []inventory.Kind{inventory.Skill, inventory.Agent, inventory.Command, inventory.MCP, inventory.Plugin}

func newWizard(inv inventory.Inventory, preset map[string]bool) *wizard {
	w := &wizard{sel: map[string]bool{}}
	seen := map[string]bool{}
	for _, k := range wizKinds {
		for _, it := range inv.Of(k) {
			if it.Path == "" && k != inventory.Plugin { // claude.ai connectors belong to the account
				continue
			}
			key := markKey(k, it.Name)
			if seen[key] {
				continue
			}
			seen[key] = true
			cost := 0
			if k == inventory.Skill && it.Invocable {
				cost = (len(it.Name) + len(it.Description) + 20) / 4
			}
			w.items = append(w.items, wizItem{k, it.Name, clean(it.Description), string(it.Scope), cost})
		}
	}
	sort.SliceStable(w.items, func(i, j int) bool {
		if w.items[i].kind != w.items[j].kind {
			return kindOrder(w.items[i].kind) < kindOrder(w.items[j].kind)
		}
		return strings.ToLower(w.items[i].name) < strings.ToLower(w.items[j].name)
	})
	for k := range preset {
		w.sel[k] = true
	}
	return w
}

func kindOrder(k inventory.Kind) int {
	for i, x := range wizKinds {
		if x == k {
			return i
		}
	}
	return 99
}

// editWizard preselects what an existing profile contains.
func editWizard(inv inventory.Inventory, p profiles.Profile) *wizard {
	pre := map[string]bool{}
	for _, n := range p.Skills {
		pre[markKey(inventory.Skill, n)] = true
	}
	for _, n := range p.Agents {
		pre[markKey(inventory.Agent, n)] = true
	}
	for _, n := range p.Commands {
		pre[markKey(inventory.Command, n)] = true
	}
	for _, n := range p.MCP {
		pre[markKey(inventory.MCP, n)] = true
	}
	for _, n := range p.Plugins {
		pre[markKey(inventory.Plugin, n)] = true
	}
	w := newWizard(inv, pre)
	w.editing, w.name, w.excl = p.Name, p.Name, p.Options.Exclusive
	if p.Card.Settings != nil {
		for i, m := range wizModels {
			if m == p.Card.Settings.Model {
				w.model = i
			}
		}
	}
	return w
}

func (w *wizard) visible() []int {
	q := strings.ToLower(strings.TrimSpace(w.query))
	var out []int
	for i, it := range w.items {
		if q == "" || fuzzy(strings.ToLower(string(it.kind)+" "+it.name+" "+it.desc), q) {
			out = append(out, i)
		}
	}
	return out
}

// fuzzy matches every word of q, each as a subsequence (like fzf).
func fuzzy(s, q string) bool {
	for _, word := range strings.Fields(q) {
		if strings.Contains(s, word) {
			continue
		}
		j := 0
		for _, r := range s {
			if j < len(word) && r == rune(word[j]) {
				j++
			}
		}
		if j < len(word) {
			return false
		}
	}
	return true
}

func (w *wizard) selection() profiles.Selection {
	var s profiles.Selection
	for _, it := range w.items {
		if !w.sel[markKey(it.kind, it.name)] {
			continue
		}
		switch it.kind {
		case inventory.Skill:
			s.Skills = append(s.Skills, it.name)
		case inventory.Agent:
			s.Agents = append(s.Agents, it.name)
		case inventory.Command:
			s.Commands = append(s.Commands, it.name)
		case inventory.MCP:
			s.MCP = append(s.MCP, it.name)
		case inventory.Plugin:
			s.Plugins = append(s.Plugins, it.name)
		}
	}
	if w.model > 0 {
		s.Model = wizModels[w.model]
	}
	return s
}

// wizardKey handles a key while the wizard is open. done=true closes it.
func (m model) wizardKey(k tea.KeyMsg) (model, tea.Cmd) {
	w := m.wiz
	s := k.String()
	if s == "ctrl+c" {
		m.wiz = nil
		return m, nil
	}
	if w.step == 0 {
		vis := w.visible()
		switch s {
		case "esc":
			if w.query != "" {
				w.query, w.cursor = "", 0
			} else {
				m.wiz, m.status = nil, "profile not saved"
			}
		case "up", "ctrl+p", "ctrl+k":
			w.cursor = clamp(w.cursor-1, 0, len(vis)-1)
		case "down", "ctrl+n", "ctrl+j":
			w.cursor = clamp(w.cursor+1, 0, len(vis)-1)
		case "pgup":
			w.cursor = clamp(w.cursor-10, 0, len(vis)-1)
		case "pgdown":
			w.cursor = clamp(w.cursor+10, 0, len(vis)-1)
		case " ":
			if w.cursor < len(vis) {
				key := markKey(w.items[vis[w.cursor]].kind, w.items[vis[w.cursor]].name)
				w.sel[key] = !w.sel[key]
				if !w.sel[key] {
					delete(w.sel, key)
				}
				w.cursor = clamp(w.cursor+1, 0, len(vis)-1)
			}
		case "ctrl+a": // preselect the current setup
			w.sel = currentSetup(m.inv)
		case "tab": // toggle every visible item
			all := true
			for _, i := range vis {
				all = all && w.sel[markKey(w.items[i].kind, w.items[i].name)]
			}
			for _, i := range vis {
				key := markKey(w.items[i].kind, w.items[i].name)
				if all {
					delete(w.sel, key)
				} else {
					w.sel[key] = true
				}
			}
		case "enter":
			if len(w.sel) == 0 {
				w.err = "choose at least one item (space)"
				return m, nil
			}
			w.step, w.err, w.field = 1, "", 0
		case "backspace":
			if r := []rune(w.query); len(r) > 0 {
				w.query, w.cursor = string(r[:len(r)-1]), 0
			}
		default:
			if len(k.Runes) > 0 && k.Type == tea.KeyRunes {
				w.query += string(k.Runes)
				w.cursor = 0
			}
		}
		return m, nil
	}
	switch s {
	case "esc":
		w.step = 0
	case "tab", "down":
		w.field = (w.field + 1) % 3
	case "shift+tab", "up":
		w.field = (w.field + 2) % 3
	case "left", "right", " ":
		d := 1
		if s == "left" {
			d = -1
		}
		switch w.field {
		case 1:
			w.model = (w.model + d + len(wizModels)) % len(wizModels)
		case 2:
			w.excl = !w.excl
		default:
			if s == " " {
				break
			}
		}
	case "backspace":
		if w.field == 0 && w.editing == "" {
			if r := []rune(w.name); len(r) > 0 {
				w.name = string(r[:len(r)-1])
			}
		}
	case "enter":
		name := strings.TrimSpace(w.name)
		if name == "" {
			w.err, w.field = "give the profile a name", 0
			return m, nil
		}
		m.wiz = nil
		m.status = m.spinner() + " building profile " + name + " (copying, hashing, auditing)…"
		return m, saveWizard(m.inv, m.env, name, w.selection(), w.excl, w.editing != "")
	default:
		if w.field == 0 && w.editing == "" {
			for _, r := range k.Runes {
				if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
					w.name += string(r)
				}
			}
		}
	}
	return m, nil
}

func saveWizard(inv inventory.Inventory, env paths.Env, name string, sel profiles.Selection, excl, replace bool) tea.Cmd {
	return func() tea.Msg {
		st, err := profiles.FromLocal(context.Background(), name, inv, sel)
		if err != nil {
			return profileSavedMsg{err: err}
		}
		defer st.Discard()
		if n := st.HighFindings(); n > 0 {
			return profileSavedMsg{err: fmt.Errorf("not saved: %d high-severity audit findings in the selected items (see Audit)", n)}
		}
		p, err := st.Commit(replace)
		if err != nil {
			return profileSavedMsg{err: err}
		}
		if excl {
			profiles.SetOptions(p.Name, profiles.Options{Exclusive: true})
			p.Options.Exclusive = true
		}
		notes := st.Notes
		if len(st.Secrets) > 0 {
			notes = append(notes, "set "+strings.Join(st.Secrets, ", "))
		}
		return profileSavedMsg{p, notes, nil}
	}
}

func (m model) wizardView(h int) string {
	w := m.wiz
	var b strings.Builder
	title := "New work profile"
	if w.saveAll {
		title = "Save your current setup as a profile"
	}
	if w.editing != "" {
		title = "Edit work profile " + w.editing
	}
	counts := map[inventory.Kind]int{}
	cost := 0
	for _, it := range w.items {
		if w.sel[markKey(it.kind, it.name)] {
			counts[it.kind]++
			cost += it.cost
		}
	}
	var cs []string
	for _, k := range wizKinds {
		if counts[k] > 0 {
			cs = append(cs, fmt.Sprintf("%d %s", counts[k], plural(string(k))))
		}
	}
	summary := "nothing chosen yet"
	if len(cs) > 0 {
		summary = strings.Join(cs, " · ") + fmt.Sprintf(" · skills ≈ %s tokens at start (estimate)", human(cost))
	}
	b.WriteString(sTitle.Render(title) + sMuted.Render("  step "+fmt.Sprint(w.step+1)+" of 2") + "\n")
	b.WriteString(sMuted.Render(summary) + "\n\n")
	if w.step == 0 {
		b.WriteString(sHeading.Render(T("Choose what goes in it")) + sMuted.Render("  type to search · space choose · tab choose all shown · ctrl+a your current setup · enter next · esc cancel") + "\n")
		b.WriteString("search: " + w.query + "▏\n\n")
		vis := w.visible()
		room := max(3, h-8)
		start := clamp(w.cursor-room+3, 0, max(0, len(vis)-room))
		for j := start; j < len(vis) && j < start+room; j++ {
			it := w.items[vis[j]]
			box := "[ ]"
			if w.sel[markKey(it.kind, it.name)] {
				box = sOK.Bold(true).Render("[x]")
			}
			line := fmt.Sprintf("%s %-8s %-34s %s", box, it.kind, clipEnd(it.name, 34), sMuted.Render(clipEnd(it.desc, max(10, m.w-60))))
			if j == w.cursor {
				line = sSel.Render(gCursor + " " + stripStyles(line, m.w-10))
			} else {
				line = "  " + line
			}
			b.WriteString(line + "\n")
		}
		if len(vis) == 0 {
			b.WriteString(sMuted.Render("  nothing matches “"+w.query+"”") + "\n")
		}
	} else {
		field := func(i int, label, value string) {
			l := fmt.Sprintf("%-10s %s", label, value)
			if w.field == i {
				l = sSel.Render(gCursor + " " + l)
			} else {
				l = "  " + l
			}
			b.WriteString(l + "\n")
		}
		name := w.name
		if w.editing == "" {
			name += "▏"
		}
		mode := "your setup + this profile"
		if w.excl {
			mode = "only this profile (exclusive: your other skills, plugins and MCP servers are not loaded)"
		}
		field(0, "Name", name)
		field(1, "Model", "‹ "+wizModels[w.model]+" ›")
		field(2, "Mode", "‹ "+mode+" ›")
		b.WriteString("\n" + sMuted.Render("tab/↑↓ move · ←→ change · enter save · esc back") + "\n")
		b.WriteString(sMuted.Render("Files are copied into the profile, audited, and nothing in ~/.claude changes. Hooks are never included.") + "\n")
	}
	if w.err != "" {
		b.WriteString("\n" + sWarn.Render(gWarn+" "+w.err))
	}
	return b.String()
}

func plural(s string) string {
	switch {
	case strings.HasSuffix(s, "ss"):
		return s + "es" // process → processes
	case strings.HasSuffix(s, "s"):
		return s
	}
	return s + "s"
}

// currentSetup preselects "your setup as it is now" (see profiles.CurrentSelection).
func currentSetup(inv inventory.Inventory) map[string]bool {
	s := profiles.CurrentSelection(inv)
	out := map[string]bool{}
	for _, x := range []struct {
		k inventory.Kind
		v []string
	}{{inventory.Skill, s.Skills}, {inventory.Agent, s.Agents}, {inventory.Command, s.Commands}, {inventory.MCP, s.MCP}, {inventory.Plugin, s.Plugins}} {
		for _, n := range x.v {
			out[markKey(x.k, n)] = true
		}
	}
	return out
}
