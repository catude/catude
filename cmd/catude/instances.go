package main

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/catude/catude/internal/paths"
)

func instancesCmd(env paths.Env, out io.Writer) int {
	tw := tabwriter.NewWriter(out, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "CLAUDE CODE\tVERSION\tPATH")
	for _, in := range env.CodeInstances() {
		v := in.Version
		if v == "" {
			v = "?"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", in.Source, v, in.Path)
	}
	fmt.Fprintln(tw, "\t\t")
	fmt.Fprintln(tw, "PROFILE FOLDER\tSTATE\tPATH")
	for _, d := range env.ConfigDirs() {
		state := "profile"
		switch {
		case d.Active:
			state = "active"
		case d.Backup:
			state = "backup (never read)"
		case !d.HasData:
			state = "empty"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", d.Label, state, d.Path)
	}
	fmt.Fprintln(tw, "\t\t")
	fmt.Fprintln(tw, "CLAUDE DESKTOP CONFIG\tEXISTS\tPATH")
	for _, f := range env.DesktopConfigs() {
		ok := "no"
		if f.Exists {
			ok = "yes"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", f.Label, ok, f.Path)
	}
	tw.Flush()
	if s := env.StaleCopies(); len(s) > 0 {
		fmt.Fprintf(out, "\n%d leftover copies of ~/.claude.json (may hold old secrets; contents not read):\n", len(s))
		for _, p := range s {
			fmt.Fprintln(out, "  "+p)
		}
	}
	return 0
}
