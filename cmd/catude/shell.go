package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/presets"
	"github.com/catude/catude/internal/profiles"
	"github.com/catude/catude/internal/tui"
)

// globalFlags removes --glyphs/--theme/--lang (any position, "--x v" or "--x=v") so every command accepts
// them. --lang is passed on through CATUDE_LANG, which the interface language detection reads first.
func globalFlags(args []string) (glyphs, theme string, lite bool, rest []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			rest = append(rest, args[i:]...)
			break
		}
		if a == "--lite" || a == "-lite" {
			lite = true
			continue
		}
		name, val, hasVal := strings.Cut(strings.TrimLeft(a, "-"), "=")
		if strings.HasPrefix(a, "-") && (name == "glyphs" || name == "theme" || name == "lang") {
			if !hasVal && i+1 < len(args) {
				i++
				val = args[i]
			}
			if name == "lang" {
				os.Setenv("CATUDE_LANG", val)
			} else if name == "glyphs" {
				glyphs = val
			} else {
				theme = val
			}
			continue
		}
		rest = append(rest, a)
	}
	return glyphs, theme, lite, rest
}

// platformNote marks builds that are less tested than the Windows one.
func platformNote() string {
	switch runtime.GOOS {
	case "linux":
		return " · Linux build: experimental"
	case "darwin":
		return " · macOS build: experimental, not yet run on a real Mac"
	}
	return ""
}

// invokedAsLite reports whether the program was started under the name ctlite.
func invokedAsLite() bool {
	name := strings.ToLower(filepath.Base(os.Args[0]))
	return strings.HasPrefix(name, "ctlite")
}

// claudeCmd starts Claude Code with the active work profile (or plainly when there is none).
// Every argument is passed through to claude.
func claudeCmd(args []string, errw io.Writer) int {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	bin, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintln(errw, "catude: claude is not on PATH")
		return 1
	}
	cmd := exec.Command(bin, args...)
	if name := profiles.Active(); name != "" {
		p, err := profiles.Get(name)
		if err != nil {
			fmt.Fprintln(errw, "catude:", err)
			return 1
		}
		cmd, err = profiles.Command(p, profiles.LaunchOptions{Exclusive: p.Options.Exclusive}, args...)
		if err != nil {
			fmt.Fprintln(errw, "catude:", err)
			return 1
		}
		if os.Getenv("CATUDE_QUIET") == "" {
			fmt.Fprintf(errw, "catude: profile %s (%s) · catude profile switch none to stop\n", p.Name, p.Summary())
		}
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	return 0
}

// shellInit prints a function that makes `claude` start through catude (and so with the active
// profile). `command claude` / `claude.exe` still reach the real binary.
func shellInit(args []string, out, errw io.Writer) int {
	shell := ""
	if len(args) > 0 {
		shell = args[0]
	}
	if shell == "" {
		switch {
		case runtime.GOOS == "windows" && os.Getenv("SHELL") == "":
			shell = "powershell"
		default:
			shell = filepath.Base(os.Getenv("SHELL"))
		}
	}
	switch shell {
	case "zsh", "bash", "sh":
		fmt.Fprint(out, `# catude: `+"`claude`"+` starts the active work profile (catude profile switch NAME|none)
# add to ~/.`+shell+`rc:  eval "$(catude shell-init `+shell+`)"
claude() { command catude claude -- "$@"; }
`)
	case "fish":
		fmt.Fprint(out, `# catude: add to ~/.config/fish/config.fish:  catude shell-init fish | source
function claude; command catude claude -- $argv; end
`)
	case "powershell", "pwsh":
		fmt.Fprint(out, `# catude: add to $PROFILE:  Invoke-Expression (& catude shell-init powershell | Out-String)
function claude { catude claude -- @args }
`)
	default:
		fmt.Fprintf(errw, "catude: unknown shell %q (zsh, bash, fish, powershell)\n", shell)
		return 2
	}
	return 0
}

// pickProfile lists work profiles with numbers and switches to the chosen one.
func pickProfile(in io.Reader, out, errw io.Writer) int {
	ps, err := profiles.List()
	if err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	if len(ps) == 0 {
		fmt.Fprintln(out, "no work profiles yet: create one with catude (Profiles → n) or catude profile save NAME")
		return 0
	}
	cur := profiles.Active()
	fmt.Fprintln(out, "Work profiles:")
	fmt.Fprintf(out, "  0  none (your normal setup)%s\n", map[bool]string{true: "  ← active", false: ""}[cur == ""])
	for i, p := range ps {
		fmt.Fprintf(out, "  %d  %-22s %s%s\n", i+1, p.Name, p.Summary(), map[bool]string{true: "  ← active", false: ""}[p.Name == cur])
	}
	fmt.Fprint(out, "Switch to (number or name, enter to keep): ")
	line, _ := bufio.NewReader(in).ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return 0
	}
	name := line
	if n, err := strconv.Atoi(line); err == nil {
		switch {
		case n == 0:
			name = "none"
		case n >= 1 && n <= len(ps):
			name = ps[n-1].Name
		default:
			fmt.Fprintln(errw, "catude: no profile number", n)
			return 2
		}
	}
	if err := profiles.SetActive(name); err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	fmt.Fprintln(out, "active profile: "+firstNonEmpty(map[bool]string{true: "", false: name}[name == "none"], "none"))
	return 0
}

// debugCmd prints environment facts for bug reports (no secrets, no file contents).
func debugCmd(env paths.Env, version string, out io.Writer) int {
	for _, kv := range tui.DebugReport(env, version) {
		fmt.Fprintf(out, "%-16s %s\n", kv[0], kv[1])
	}
	return 0
}

// presetsCmd lists the embedded presets.
func presetsCmd(out io.Writer) int {
	ps, err := presets.Load()
	if err != nil {
		fmt.Fprintln(out, "catude:", err)
		return 1
	}
	for _, p := range ps {
		fmt.Fprintf(out, "%-22s %-24s %s\n", p.ID, p.Name, p.Role)
	}
	fmt.Fprintln(out, "\nload one with: catude profile load preset:ID   (based on public material; not affiliated with or endorsed by these people)")
	return 0
}
