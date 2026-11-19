package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/catude/catude/internal/models"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/sessions"
	"github.com/catude/catude/internal/tui"
)

// modelsCmd says which Claude model and effort level is stronger, from public data (no key):
//
//	catude models [--for coding]                          ranking and advice on the model you use
//	catude models compare "opus 5.5 high" "fable 5.2 medium" [--for coding]
//	catude models refresh                                 download again (once a day is automatic)
func modelsCmd(env paths.Env, pos []string, out, errw io.Writer) int {
	cat := "all"
	var args []string
	for i := 0; i < len(pos); i++ {
		if (pos[i] == "--for" || pos[i] == "-for") && i+1 < len(pos) {
			cat = strings.ToLower(strings.TrimSpace(pos[i+1]))
			if cat == "" { // the /catude:models command passes its argument even when empty
				cat = "all"
			}
			i++
			continue
		}
		args = append(args, pos[i])
	}
	okCat := false
	for _, c := range models.Categories {
		okCat = okCat || c == cat
	}
	if !okCat {
		fmt.Fprintf(errw, "--for must be one of: %q\n", models.Categories)
		return 2
	}
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "", "list", "refresh", "compare", "vs":
	default:
		fmt.Fprintln(errw, `usage: catude models [compare A B | refresh] [--for coding|reasoning|"math & science"|agents|knowledge]`)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	d, err := models.Load(ctx, tui.StateDir(), sub == "refresh")
	if len(d.Models) == 0 {
		fmt.Fprintln(errw, "models:", err)
		return 1
	}
	if err != nil {
		fmt.Fprintln(errw, "note:", err, "(showing the last download)")
	}
	all := models.Recent(d.Models)
	defer fmt.Fprintf(out, "\n%s · downloaded %s\n", models.Attribution, d.Fetched.Local().Format("2006-01-02 15:04"))

	if sub == "compare" || sub == "vs" {
		if len(args) < 3 {
			fmt.Fprintln(errw, `usage: catude models compare "opus 5.5 high" "fable 5.2 medium" [--for coding]`)
			return 2
		}
		var ms [2]models.Model
		for i := 0; i < 2; i++ {
			m, ok := models.Find(d.Models, args[1+i])
			if m.ID == "" {
				fmt.Fprintf(errw, "no Claude model matches %q (see catude models)\n", args[1+i])
				return 1
			}
			if !ok {
				fmt.Fprintf(out, "note: %q has no results at that effort; using %s\n", args[1+i], m.Label())
			}
			ms[i] = m
		}
		v := models.Compare(d, ms[0], ms[1], cat)
		fmt.Fprintln(out, v.Summary)
		for _, n := range v.Notes {
			fmt.Fprintln(out, "  "+n)
		}
		return 0
	}
	if ps, err := sessions.ListProjects(env.ProjectsDir()); err == nil {
		if rep, err := (&sessions.UsageCache{}).Usage(ctx, ps, 14); err == nil {
			if used := models.FromUsage(all, rep.Models, tui.ConfiguredEffort(env.ClaudeDir)); len(used) > 0 {
				fmt.Fprintf(out, "You use %s most (last 14 days).\n", used[0].Label())
				for _, a := range models.Advice(d, all, used[0], cat) {
					fmt.Fprintln(out, "  - "+a)
				}
				fmt.Fprintln(out)
			}
		}
	}
	ranked, core := models.Ranked(d, all, cat)
	fmt.Fprintf(out, "%-22s %7s %5s %8s   (for %s; score = average %% of the best result on %v)\n", "model · effort", "score", "ECI", "$/1M", cat, core)
	for _, m := range ranked {
		s, _ := models.Score(m, core)
		fmt.Fprintf(out, "%-22s %7s %5s %8s\n", m.Label(), num(s, "%.1f"), num(m.ECI, "%.0f"), num(m.Blended(), "%.2f"))
	}
	return 0
}

func num(v float64, f string) string {
	if v <= 0 {
		return "—"
	}
	return fmt.Sprintf(f, v)
}
