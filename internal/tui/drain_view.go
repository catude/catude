package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/catude/catude/internal/accounts"
	"github.com/catude/catude/internal/sessions"
)

// drainText says a finding of the drain analysis in the interface language.
func drainText(f sessions.Finding) (string, string) {
	a := make([]any, len(f.Args))
	for i, s := range f.Args {
		a[i] = s
	}
	switch f.ID {
	case "subagents":
		return F("Subagents spent %s fresh tokens (%s).", a...), T("Each subagent starts its own context and re-reads files. The Agents tab shows which ones and who launched them.")
	case "rebuilds":
		return F("The cache was rebuilt %s times after a pause of more than 5 minutes: %s tokens written again (%s).", a...),
			T("The prompt cache expires after a pause; the next message sends the whole conversation again. Long breaks in a big conversation cost the most: /compact or a new session before a break helps.")
	case "headless":
		return F("Headless runs (claude -p and the Agent SDK) spent %s (%s).", a...), T("Scripts, hooks and tools that start claude -p count against the same limits. Check what runs them.")
	case "start":
		return F("Each conversation starts with about %s tokens of context before your first message (largest %s).", a...),
			T("System prompt, tools, MCP servers, CLAUDE.md and skill descriptions. The Context tab shows what to trim.")
	case "top":
		return F("One conversation in %s (%s) spent %s (%s).", a...), T("Very long conversations re-send a big context with every message. Start a new one when the task changes.")
	case "cache":
		return F("Only %s of the input was read from the cache.", a...), T("Frequent changes to CLAUDE.md, tools or MCP servers in the middle of a conversation invalidate the cache.")
	case "version":
		return F("With Claude Code %s each response used %s fresh tokens on average, against %s with %s.", a...),
			T("Compare the changelog of that version; it can also be a different kind of work.")
	}
	return f.Text, f.Hint
}

// drainView is the "where did my tokens go" block of the Usage tab.
func drainView(d sessions.Drain, w int) string {
	fs := d.Explain(human)
	if len(fs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(sHeading.Render(T("Where your tokens went")) + "\n")
	// the share bar sits in a 7-cell column; the sentence and its hint wrap inside the column next to it
	body := lipgloss.NewStyle().Width(max(30, w-15))
	for _, f := range fs {
		text, hint := drainText(f)
		mark := strings.Repeat(" ", 7)
		if f.Share > 0 {
			mark = bar(f.Share, 6) + " "
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, mark, body.Render(text+"\n"+sMuted.Render(hint))) + "\n")
	}
	return b.String()
}

// healthLine sums up the health of the period: failed tool calls and compactions, and the worst day.
func healthLine(hs []sessions.HealthDay) string {
	var calls, errs, comp int
	var worst sessions.HealthDay
	for _, h := range hs {
		calls += h.ToolCalls
		errs += h.ToolErrors
		comp += h.Compactions
		if h.ToolCalls >= 50 && h.ErrorRate() > worst.ErrorRate() {
			worst = h
		}
	}
	if calls == 0 {
		return ""
	}
	s := F("Tool calls: %d, %.1f%% failed · %d compactions", calls, 100*float64(errs)/float64(calls), comp)
	if worst.ToolCalls > 0 {
		s += " · " + F("most failures on %s (%.1f%%)", worst.Day.Format("02 Jan"), 100*worst.ErrorRate())
	}
	return s + " · " + T("catude health shows every day and version")
}

// windowNote says when a plan window resets and whether the current pace lasts until then.
func windowNote(w *accounts.Window) string {
	s := ""
	if w.Countdown != "" {
		s += " " + F("resets in %s", w.Countdown)
	}
	if w.WillLastToReset != nil && !*w.WillLastToReset {
		s += " " + sWarn.Render(T("at this pace it runs out before the reset"))
	}
	return s
}
