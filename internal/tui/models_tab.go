package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/catude/catude/internal/models"
	"github.com/catude/catude/internal/sessions"
)

// Models tab: which Claude model and effort level is stronger, for the kind of work you choose, from
// public benchmark data (Epoch AI) and prices (models.dev). No account, no API key.

type modelsState struct {
	data    models.Data
	loaded  bool
	running bool
	err     error
	sel     int
	pin     string // ID compared against; "" = the one you use
	cat     int    // index in models.Categories
}

type modelsMsg struct {
	d   models.Data
	err error
}

func (m *model) ensureModels(refresh bool) tea.Cmd {
	if m.mo.running || (m.mo.loaded && !refresh && !m.mo.data.Stale()) {
		return nil
	}
	if !m.mo.loaded {
		m.mo.data = models.ReadCache(cacheDir()) // instant; refreshed below when older than a day
		m.mo.loaded = true
		if !refresh && !m.mo.data.Stale() {
			return nil
		}
	}
	m.mo.running = true
	dir := cacheDir()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		d, err := models.Load(ctx, dir, refresh)
		return modelsMsg{d, err}
	}
}

func (m *model) onModels(msg modelsMsg) {
	m.mo.running, m.mo.err = false, msg.err
	if len(msg.d.Models) > 0 {
		m.mo.data = msg.d
	}
}

// configuredEffort reads effortLevel from the user's Claude Code settings ("" when not set).
func configuredEffort(claudeDir string) string {
	b, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil || len(b) > 4<<20 {
		return ""
	}
	var s struct {
		Effort string `json:"effortLevel"`
	}
	if json.Unmarshal(b, &s) != nil {
		return ""
	}
	for _, e := range models.Efforts {
		if strings.EqualFold(s.Effort, e) {
			return e
		}
	}
	return ""
}

func (m model) moCat() string { return models.Categories[m.mo.cat%len(models.Categories)] }

// moRows are the recent Claude versions ranked for the chosen kind of work, filtered by "/".
func (m model) moRows() ([]models.Model, []string) {
	ranked, core := models.Ranked(m.mo.data, models.Recent(m.mo.data.Models), m.moCat())
	if m.filter == "" {
		return ranked, core
	}
	f := strings.ToLower(m.filter)
	var out []models.Model
	for _, x := range ranked {
		if strings.Contains(strings.ToLower(x.Label()+" "+x.ID), f) {
			out = append(out, x)
		}
	}
	return out, core
}

// yours is the model you use most (from Usage) at your configured effort level (medium when unset).
func (m model) yours() (models.Model, bool) {
	if m.usage.rep == nil {
		return models.Model{}, false
	}
	used := models.FromUsage(models.Recent(m.mo.data.Models), m.usage.rep.Models, configuredEffort(m.env.ClaudeDir))
	if len(used) == 0 {
		return models.Model{}, false
	}
	return used[0], true
}

func (m model) modelsKey(k tea.KeyMsg) (model, tea.Cmd, bool) {
	rows, _ := m.moRows()
	switch k.String() {
	case "up", "k":
		m.mo.sel = clamp(m.mo.sel-1, 0, len(rows)-1)
	case "down", "j":
		m.mo.sel = clamp(m.mo.sel+1, 0, len(rows)-1)
	case "pgup":
		m.mo.sel = clamp(m.mo.sel-10, 0, len(rows)-1)
	case "pgdown":
		m.mo.sel = clamp(m.mo.sel+10, 0, len(rows)-1)
	case "t", "T":
		d := 1
		if k.String() == "T" {
			d = len(models.Categories) - 1
		}
		m.mo.cat = (m.mo.cat + d) % len(models.Categories)
		m.status = "comparing for: " + m.moCat()
	case " ", "enter":
		if m.mo.sel < len(rows) {
			if id := rows[m.mo.sel].ID; m.mo.pin == id {
				m.mo.pin = ""
				m.status = "comparing with the model you use"
			} else {
				m.mo.pin = id
				m.status = "comparing everything with " + rows[m.mo.sel].Label() + " (space again: back to yours)"
			}
		}
	case "r":
		m.status = "downloading the latest results from epoch.ai and prices from models.dev…"
		return m, tea.Batch(m.ensureModels(true), spin()), true
	case "o":
		if err := openFolder(models.SiteURL); err != nil {
			m.status = err.Error()
		}
	default:
		return m, nil, false
	}
	return m, nil, true
}

func (m model) modelsView(h int) string {
	var b strings.Builder
	state := ""
	switch {
	case m.mo.running:
		state = m.spinner() + " downloading public results…"
	case !m.mo.data.Fetched.IsZero():
		state = "downloaded " + ago(m.mo.data.Fetched)
		if m.mo.data.Updated != "" {
			state += " · newest result " + m.mo.data.Updated
		}
	}
	b.WriteString(sHeading.Render(T("Models")) + sMuted.Render("  which Claude model and effort is stronger · no account, no key · "+state) + "\n")
	tabs := ""
	for i, c := range models.Categories {
		if i == m.mo.cat {
			tabs += sSel.Render(" "+T(c)+" ") + " "
		} else {
			tabs += sMuted.Render(" "+T(c)+" ") + " "
		}
	}
	b.WriteString(sMuted.Render("for ") + tabs + sMuted.Render("(t changes)") + "\n")

	if len(m.mo.data.Models) == 0 {
		b.WriteString("\n")
		if m.mo.running {
			b.WriteString(m.spinner() + " downloading the Epoch AI Benchmarking Hub (about 2 MB, once a day)…")
		} else if m.mo.err != nil {
			b.WriteString(sWarn.Render(gWarn+" "+m.mo.err.Error()) + "\n" + sMuted.Render("Check the connection and press r to try again."))
		}
		return b.String()
	}
	if m.mo.err != nil {
		b.WriteString(sWarn.Render(gWarn+" "+m.mo.err.Error()+" (showing the last download)") + "\n")
	}
	rows, core := m.moRows()
	all := models.Recent(m.mo.data.Models)
	cat := m.moCat()
	mine, haveMine := m.yours()
	ref, haveRef := mine, haveMine
	if m.mo.pin != "" {
		for _, x := range all {
			if x.ID == m.mo.pin {
				ref, haveRef = x, true
			}
		}
	}

	b.WriteString("\n")
	if haveMine {
		how := "effortLevel not set: shown at medium"
		if e := configuredEffort(m.env.ClaudeDir); e != "" {
			how = "effortLevel " + e
		}
		b.WriteString(sHeading.Render(T("You use ")) + sTitle.Render(mine.Label()) + sMuted.Render("  (most tokens in your transcripts · "+how+")") + "\n")
		for _, a := range models.Advice(m.mo.data, all, mine, cat) {
			b.WriteString("  " + gCursor + " " + a + "\n")
		}
	} else if m.usage.running || m.usage.rep == nil {
		b.WriteString(sMuted.Render(m.spinner()+" reading your transcripts to see which model you use…") + "\n")
	} else {
		b.WriteString(sMuted.Render("No recent Claude model found in your transcripts.") + "\n")
	}

	b.WriteString("\n" + sMuted.Render(fmt.Sprintf("    %-22s %-27s %5s %8s", "model · effort", "score ("+cat+")", "ECI", "$/1M")) + "\n")
	head := strings.Count(b.String(), "\n")
	room := max(3, h-head-6)
	sel := clamp(m.mo.sel, 0, max(0, len(rows)-1))
	start := clamp(sel-room+2, 0, max(0, len(rows)-room))
	var lines []string
	for i := start; i < len(rows) && i < start+room; i++ {
		x := rows[i]
		mark := "  "
		switch {
		case haveMine && x.ID == mine.ID:
			mark = sTitle.Render(gOK) + " "
		case m.mo.pin == x.ID:
			mark = sTitle.Render(gMarked) + " "
		}
		sc, n := models.Score(x, core)
		score := sMuted.Render(strings.Repeat("·", 20)) + "      —"
		if sc > 0 {
			score = meter(sc/100, 20) + fmt.Sprintf(" %5.1f", sc)
			if n < len(core) {
				score += sMuted.Render("*")
			} else {
				score += " "
			}
		}
		eci, price := "—", "—"
		if x.ECI > 0 {
			eci = fmt.Sprintf("%.0f", x.ECI)
		}
		if x.Blended() > 0 {
			price = fmt.Sprintf("%.2f", x.Blended())
		}
		t := fmt.Sprintf("%s%-22s %s %5s %8s", mark, clipEnd(x.Label(), 22), score, eci, price)
		if i == sel {
			t = sSel.Render(gCursor + " " + stripStyles(t, m.w-10))
		} else {
			t = "  " + t
		}
		lines = append(lines, t)
	}
	if len(rows) == 0 {
		lines = append(lines, sMuted.Render("  nothing matches"))
	}
	b.WriteString(withScrollbar(lines, len(rows), start, min(room, max(1, len(rows))), m.w-6) + "\n")
	if p := position(start, room, len(rows)); p != "" {
		b.WriteString(p + "\n")
	}

	if sel < len(rows) && haveRef {
		x := rows[sel]
		if x.ID == ref.ID {
			b.WriteString(sMuted.Render("Move to another model to compare it with "+ref.Label()+" · space compares everything with the selected one") + "\n")
		} else {
			v := models.Compare(m.mo.data, ref, x, cat)
			b.WriteString(sTitle.Render(gCursor+" "+v.Summary) + "\n")
			if len(v.Notes) > 0 {
				b.WriteString(sMuted.Render("  "+strings.Join(v.Notes, " · ")) + "\n")
			}
		}
	}
	b.WriteString(sMuted.Render(fmt.Sprintf("score: average %% of the best result on %s (* = measured on fewer) · ECI: Epoch Capabilities Index of the model · %s", strings.Join(core, ", "), models.Attribution)))
	return b.String()
}

// StateDir is where catude keeps its private cache (also used by the CLI).
func StateDir() string { return cacheDir() }

// ConfiguredEffort is the effortLevel set in Claude Code's user settings ("" when not set).
func ConfiguredEffort(claudeDir string) string { return configuredEffort(claudeDir) }

// prices read from the models cache when the Models tab has not loaded yet (no download from here)
var cachedPrices struct {
	once sync.Once
	d    models.Data
}

// sessionCost estimates what a session's main thread would cost at list prices (false when the model has
// no known price).
func (m model) sessionCost(s sessions.Session) (float64, bool) {
	d := m.mo.data
	if !m.mo.loaded {
		cachedPrices.once.Do(func() { cachedPrices.d = models.ReadCache(cacheDir()) })
		d = cachedPrices.d
	}
	p, ok := d.PriceFor(s.Model)
	if !ok {
		return 0, false
	}
	u := s.Total
	return p.CostUSD(u.Input, u.CacheCreate, u.CacheRead, u.Output), true
}

func usd(v float64) string {
	switch {
	case v < 0.01:
		return "<$0.01"
	case v < 100:
		return fmt.Sprintf("$%.2f", v)
	}
	return fmt.Sprintf("$%.0f", v)
}

// costLine is the cost estimate of the session detail, said to be an estimate.
func (m model) costLine(s sessions.Session) string {
	c, ok := m.sessionCost(s)
	if !ok {
		return sMuted.Render("  " + T("estimated cost: no price known for this model yet (the Models tab downloads prices)"))
	}
	return "  " + F("estimated cost ~%s at API list prices (models.dev, subagents excluded); on a Pro or Max plan you pay the subscription, not this", usd(c))
}
