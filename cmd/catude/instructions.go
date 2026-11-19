package main

import (
	"fmt"
	"io"

	"github.com/catude/catude/internal/claudecfg"
	"github.com/catude/catude/internal/instr"
	"github.com/catude/catude/internal/paths"
)

// instructionsCmd says which CLAUDE.md, CLAUDE.local.md and AGENTS.md files Claude Code reads here.
func instructionsCmd(env paths.Env, fix, yes bool, in io.Reader, out io.Writer) int {
	r := instr.Check(env)
	from := "default"
	if r.ModeFrom != "" {
		from = r.ModeFrom
	}
	fmt.Fprintf(out, "Project instructions: %s (%s)\n\n", r.Mode, from)
	if len(r.Files) == 0 {
		fmt.Fprintln(out, "No CLAUDE.md, CLAUDE.local.md or AGENTS.md in this folder or above it.")
	}
	for _, f := range r.Files {
		mark := "✗"
		if f.Loaded {
			mark = "✓"
		}
		fmt.Fprintf(out, "%s %s\n    %s\n", mark, f.Path, f.Why)
	}
	if len(r.AgentSkills) > 0 {
		fmt.Fprintf(out, "\n.agents/skills has %d skills that Claude Code does not read; copy or link them into .claude/skills.\n", len(r.AgentSkills))
	}
	if mem, ok := instr.AutoMemory(env); ok {
		fmt.Fprintf(out, "\nAuto memory (notes Claude keeps for this project): %s\n  MEMORY.md %d lines, %d topic files", mem.Dir, mem.IndexLines, len(mem.Files))
		if mem.Truncated {
			fmt.Fprint(out, " · only the first 200 lines or 25 KB load: ask Claude to tidy it")
		}
		fmt.Fprintln(out)
	}
	fixes := instr.Fixes(env, claudecfg.Backup)
	if len(fixes) == 0 {
		return 0
	}
	if !fix {
		fmt.Fprintln(out, "\ncatude instructions --fix would:")
		for _, f := range fixes {
			fmt.Fprintln(out, "  · "+f.What)
		}
		fmt.Fprintln(out, "Or set Project instructions to claude-md-and-agents-md in /config to read CLAUDE.md and AGENTS.md together.")
		return 0
	}
	for _, f := range fixes {
		if !yes && !confirm(in, out, "\n"+f.What+"?") {
			continue
		}
		if err := f.Apply(); err != nil {
			fmt.Fprintln(out, "  failed: "+err.Error())
			continue
		}
		fmt.Fprintln(out, "  done")
	}
	return 0
}
