package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/sessions"
)

// whyCmd explains where the tokens of the last days went: subagents, cache rebuilt after pauses,
// headless runs, the context loaded before the first message, the most expensive conversations.
func whyCmd(env paths.Env, days int, out, errw io.Writer) int {
	ps, err := sessions.ListProjects(filepath.Join(env.ClaudeDir, "projects"))
	if err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	rep, err := (&sessions.UsageCache{}).Usage(ctx, ps, days)
	if err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	d := rep.Drain
	fmt.Fprintf(out, "Where your tokens went · last %d days · %s fresh tokens (new input, cache writes, output) · %s read from cache\n\n",
		days, human(d.Fresh), human(sumCacheRead(rep)))
	fs := d.Explain(human)
	if len(fs) == 0 {
		fmt.Fprintln(out, "Nothing stands out: no subagent, pause, headless or start-up cost above the thresholds.")
	}
	for _, f := range fs {
		fmt.Fprintln(out, "• "+f.Text)
		fmt.Fprintln(out, "  "+f.Hint)
	}
	if len(d.Top) > 0 {
		fmt.Fprintln(out, "\nMost expensive conversations (subagents included):")
		for i, s := range d.Top {
			if i == 5 {
				break
			}
			fmt.Fprintf(out, "  %s  %8s  %-12s %s\n", s.Start.Local().Format("02 Jan 15:04"), human(s.Fresh), s.Entry, filepath.Base(s.Project))
		}
	}
	fmt.Fprintln(out, "\nMeasured from the transcripts on this machine. Plan limits are computed by Anthropic and also count other devices.")
	return 0
}

func sumCacheRead(r sessions.UsageReport) int {
	n := 0
	for _, d := range r.Days {
		n += d.CacheR
	}
	return n
}
