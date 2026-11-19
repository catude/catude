package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/sessions"
)

// healthCmd shows how your conversations went day by day: tool calls, how many failed, compactions and
// fresh tokens per response, and the same per Claude Code version. Your own evidence when something
// feels worse.
func healthCmd(env paths.Env, days int, out, errw io.Writer) int {
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
	fmt.Fprintf(out, "Health of your conversations · last %d days (subagents included)\n\n", days)
	tw := tabwriter.NewWriter(out, 2, 4, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tw, "DAY\tRESPONSES\tTOOL CALLS\tFAILED\tAPI RETRIES\tCOMPACTIONS\tCACHE HIT\tFRESH / RESPONSE\t")
	for _, h := range rep.Health {
		if h.Responses == 0 {
			continue
		}
		fmt.Fprintf(tw, "%s\t%d\t%d\t%.1f%%\t%d\t%d\t%.0f%%\t%s\t\n", h.Day.Format("Mon 02 Jan"), h.Responses, h.ToolCalls, 100*h.ErrorRate(),
			h.APIRetries, h.Compactions, 100*h.CacheHit(), human(h.Fresh/h.Responses))
	}
	tw.Flush()
	if v := rep.Drain.ByVersion; len(v) > 0 {
		fmt.Fprintln(out, "\nBy Claude Code version (main conversations):")
		for i, x := range v {
			if i == 6 {
				break
			}
			fmt.Fprintf(out, "  %-12s %6d responses  %8s fresh tokens per response\n", x.Version, x.Responses, human(x.PerResp))
		}
	}
	fmt.Fprintln(out, "\nA failed tool call is a tool result Claude Code marked as an error (a command that failed, a file not found, a permission denied). An API retry is an error from the API (overloaded, rate limited, network) that Claude Code retried.")
	return 0
}
