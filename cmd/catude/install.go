package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/charmbracelet/x/term"
)

// The installer makes catude (and the short name ct) available in every terminal from the first run:
// it copies the binary to a per-user folder, adds that folder to the user PATH and, if asked, sets up
// the shell integration. Defaults are pre-selected; nothing needs administrator rights.

func binDir() string {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base, _ = os.UserConfigDir()
		}
		return filepath.Join(base, "Programs", "catude")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin")
}

func exeName(n string) string {
	if runtime.GOOS == "windows" {
		return n + ".exe"
	}
	return n
}

func stateFile(name string) string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "catude", name)
}

func onPath(dir string) bool {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if strings.EqualFold(filepath.Clean(p), filepath.Clean(dir)) {
			return true
		}
	}
	return false
}

// installed reports whether catude can be started by name from a new terminal.
func installed() bool {
	if p, err := exec.LookPath("catude"); err == nil && p != "" {
		return true
	}
	return onPath(binDir()) && fileExists(filepath.Join(binDir(), exeName("catude")))
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// ask prints a question with a pre-selected default and reads y/n.
func ask(in *bufio.Reader, out io.Writer, q string, def bool) bool {
	hint := "[Y/n]"
	if !def {
		hint = "[y/N]"
	}
	fmt.Fprintf(out, "%s %s ", q, hint)
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		fmt.Fprintln(out)
		return def
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "":
		return def
	case "y", "yes", "s", "si", "sí":
		return true
	}
	return false
}

func installCmd(args []string, in io.Reader, out, errw io.Writer) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(errw)
	yes := fs.Bool("yes", false, "accept the pre-selected answers")
	noLR := fs.Bool("no-ct", false, "do not create the short command ct")
	shell := fs.Bool("shell", false, "also make plain `claude` start the active work profile")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	r := bufio.NewReader(in)
	q := func(text string, def bool) bool {
		if *yes {
			fmt.Fprintf(out, "%s %s\n", text, map[bool]string{true: "yes", false: "no"}[def])
			return def
		}
		return ask(r, out, text, def)
	}
	dir := binDir()
	fmt.Fprintf(out, "catude setup\n\n")
	if !q(fmt.Sprintf("1. Install catude in %s and add it to your PATH, so it runs from any terminal (cmd, PowerShell, bash)?", dir), true) {
		fmt.Fprintln(out, "nothing installed. You can run this again any time: catude install")
		markDeclined()
		return 0
	}
	wantLR := !*noLR && q("2. Also create the short commands ct (= catude) and ctlite (= catude in lite mode)?", true)
	wantShell := *shell || q("3. Make plain `claude` start your active work profile (shell integration; change it with ct switch)?", false)

	self, err := os.Executable()
	if err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	self, _ = filepath.EvalSymlinks(self)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(errw, "catude:", err)
		return 1
	}
	target := filepath.Join(dir, exeName("catude"))
	if !sameFile(self, target) {
		if err := copyExe(self, target); err != nil {
			fmt.Fprintln(errw, "catude: copying the program:", err)
			return 1
		}
	}
	fmt.Fprintln(out, "  ✓ installed "+target)
	if wantLR {
		for _, a := range aliases {
			link := filepath.Join(dir, exeName(a))
			if p, err := exec.LookPath(a); err == nil && !sameFile(p, link) && !sameFile(p, target) && filepath.Dir(p) != dir {
				fmt.Fprintf(out, "  ! %s is already another program (%s): not replaced; use catude\n", a, p)
			} else if err := linkOrCopy(target, link); err != nil {
				fmt.Fprintf(out, "  ! could not create %s: %v\n", a, err)
			} else {
				fmt.Fprintf(out, "  ✓ %s → catude\n", a)
			}
		}
	}
	if err := addToPathFn(dir, out); err != nil {
		fmt.Fprintln(errw, "catude: adding to PATH:", err)
		return 1
	}
	if wantShell {
		setupShell(out)
	}
	os.Remove(stateFile("install-declined"))
	fmt.Fprintln(out, "\nDone. Open a new terminal and type catude (or ct). Switch work profiles with: ct switch NAME")
	return 0
}

func markDeclined() {
	if p := stateFile("install-declined"); p != "" {
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("1\n"), 0o644)
	}
}

func sameFile(a, b string) bool {
	fa, err1 := os.Stat(a)
	fb, err2 := os.Stat(b)
	return err1 == nil && err2 == nil && os.SameFile(fa, fb)
}

func copyExe(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmp := dst + ".new"
	if err := os.WriteFile(tmp, b, 0o755); err != nil {
		return err
	}
	if runtime.GOOS == "windows" && fileExists(dst) {
		old := dst + ".old"
		os.Remove(old)
		os.Rename(dst, old) // a running exe cannot be overwritten on Windows, but it can be renamed
	}
	return os.Rename(tmp, dst)
}

func linkOrCopy(target, link string) error {
	if runtime.GOOS != "windows" {
		os.Remove(link)
		return os.Symlink(target, link)
	}
	if fileExists(link) {
		os.Rename(link, link+".old")
	}
	if err := os.Link(target, link); err == nil {
		return nil
	}
	return copyExe(target, link)
}

// addToPathFn is replaced in tests so they never touch the real PATH.
var addToPathFn = addToPath

// addToPath adds dir to the user PATH: the Windows user environment (no admin), or the shell startup
// files on macOS and Linux.
func addToPath(dir string, out io.Writer) error {
	if runtime.GOOS == "windows" {
		ps := `$d = $env:CATUDE_DIR; $p = [Environment]::GetEnvironmentVariable('Path','User'); ` +
			`if (-not $p) { $p = '' }; ` +
			`if (($p -split ';') -notcontains $d) { [Environment]::SetEnvironmentVariable('Path', (($p.TrimEnd(';') + ';' + $d).TrimStart(';')), 'User'); 'added' } else { 'present' }`
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
		cmd.Env = append(os.Environ(), "CATUDE_DIR="+dir)
		res, err := cmd.Output()
		if err != nil {
			return err
		}
		if strings.Contains(string(res), "added") {
			fmt.Fprintln(out, "  ✓ added to your user PATH (new terminals pick it up)")
		} else {
			fmt.Fprintln(out, "  ✓ already on your PATH")
		}
		return nil
	}
	if onPath(dir) {
		fmt.Fprintln(out, "  ✓ already on your PATH")
		return nil
	}
	line := `export PATH="` + dir + `:$PATH" # added by catude`
	return appendRC(line, out)
}

func appendRC(line string, out io.Writer) error {
	home, _ := os.UserHomeDir()
	done := false
	for _, rc := range []string{".zshrc", ".bashrc", ".profile"} {
		p := filepath.Join(home, rc)
		b, err := os.ReadFile(p)
		if err != nil && rc != ".profile" {
			continue
		}
		if strings.Contains(string(b), line) {
			done = true
			continue
		}
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		fmt.Fprintln(f, "\n"+line)
		f.Close()
		fmt.Fprintln(out, "  ✓ updated ~/"+rc)
		done = true
		if rc != ".profile" {
			break
		}
	}
	if !done {
		return errors.New("no shell startup file found")
	}
	return nil
}

func setupShell(out io.Writer) {
	if runtime.GOOS == "windows" {
		ps := `$f = $PROFILE.CurrentUserAllHosts; $l = 'Invoke-Expression (& catude shell-init powershell | Out-String) # added by catude'; ` +
			`New-Item -ItemType File -Force -Path $f -ErrorAction SilentlyContinue | Out-Null; if (-not (Select-String -Path $f -SimpleMatch $l -Quiet)) { Add-Content -Path $f -Value $l }; $f`
		res, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Output()
		if err != nil {
			fmt.Fprintln(out, "  ! PowerShell integration:", err)
		} else {
			fmt.Fprintln(out, "  ✓ PowerShell: "+strings.TrimSpace(string(res)))
		}
		fmt.Fprintln(out, "  · cmd.exe has no functions: there, start Claude with your profile using catude claude (or ct claude)")
		return
	}
	shell := filepath.Base(os.Getenv("SHELL"))
	if shell != "zsh" && shell != "bash" {
		fmt.Fprintln(out, "  · add this to your shell startup file: eval \"$(catude shell-init "+shell+")\"")
		return
	}
	if err := appendRC(`eval "$(catude shell-init `+shell+`)" # added by catude`, out); err != nil {
		fmt.Fprintln(out, "  !", err)
	}
}

// firstRun offers the installer when catude is started for the first time outside the PATH (for example
// double-clicked from Downloads). It asks once; the answer is remembered.
func firstRun(in io.Reader, out, errw io.Writer) {
	if os.Getenv("CATUDE_NO_SETUP") != "" || !term.IsTerminal(os.Stdin.Fd()) {
		return
	}
	if installed() {
		offerUpdate(in, out)
		return
	}
	if p := stateFile("install-declined"); p != "" && fileExists(p) {
		return
	}
	fmt.Fprintln(out, "catude is not installed yet: you are running it from "+filepath.Dir(must(os.Executable()))+".")
	installCmd(nil, in, out, errw)
	fmt.Fprint(out, "\nPress enter to open catude… ")
	bufio.NewReader(in).ReadString('\n')
}

func must(s string, _ error) string { return s }

func uninstallCmd(args []string, in io.Reader, out, errw io.Writer) int {
	yes := false
	for _, a := range args {
		switch a {
		case "--yes", "-yes", "-y":
			yes = true
		case "-h", "--help", "-help", "help":
			fmt.Fprintln(out, "catude uninstall [--yes]: remove catude, ct and ctlite from your PATH (profiles and settings stay)")
			return 0
		default:
			fmt.Fprintln(errw, "catude uninstall: unknown argument "+a+" (only --yes)")
			return 2
		}
	}
	if !yes && !confirm(in, out, "Remove catude, ct and ctlite from your PATH? Profiles and settings stay") {
		return 1
	}
	dir := binDir()
	for _, n := range append([]string{"catude"}, aliases...) {
		p := filepath.Join(dir, exeName(n))
		if err := os.Remove(p); err == nil {
			fmt.Fprintln(out, "removed "+p)
		} else if runtime.GOOS == "windows" && fileExists(p) {
			os.Rename(p, p+".old") // the running exe: renamed now, deleted on the next install
			fmt.Fprintln(out, "will be removed when catude closes: "+p)
		}
	}
	if runtime.GOOS == "windows" {
		ps := `$d = $env:CATUDE_DIR; $p = [Environment]::GetEnvironmentVariable('Path','User'); ` +
			`[Environment]::SetEnvironmentVariable('Path', (($p -split ';' | Where-Object { $_ -and $_ -ne $d }) -join ';'), 'User')`
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
		cmd.Env = append(os.Environ(), "CATUDE_DIR="+dir)
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(errw, "catude: PATH:", err)
		} else {
			fmt.Fprintln(out, "removed "+dir+" from your user PATH")
		}
	}
	fmt.Fprintln(out, "Lines marked “# added by catude” in your shell startup files can be deleted by hand. Profiles stay in: catude profile dir")
	return 0
}

// aliases are the extra command names installed next to catude.
var aliases = []string{"ct", "ctlite"}

func fileSHA(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// staleInstall returns the installed catude when the running program is a different, newer build —
// the usual way to end up with an old copy on PATH after downloading or building a new version.
func staleInstall() (target string, stale bool) {
	self, err := os.Executable()
	if err != nil {
		return "", false
	}
	self, _ = filepath.EvalSymlinks(self)
	target = filepath.Join(binDir(), exeName("catude"))
	if !fileExists(target) || sameFile(self, target) {
		return target, false
	}
	for _, a := range aliases {
		if sameFile(self, filepath.Join(binDir(), exeName(a))) {
			return target, false
		}
	}
	si, err1 := os.Stat(self)
	ti, err2 := os.Stat(target)
	if err1 != nil || err2 != nil || !si.ModTime().After(ti.ModTime()) {
		return target, false // never offer a downgrade
	}
	return target, fileSHA(self) != fileSHA(target)
}

// updateInstalled replaces the installed copy (and its aliases) with the running program.
func updateInstalled(out io.Writer) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self, _ = filepath.EvalSymlinks(self)
	target := filepath.Join(binDir(), exeName("catude"))
	if err := copyExe(self, target); err != nil {
		return err
	}
	for _, a := range aliases {
		p := filepath.Join(binDir(), exeName(a))
		if fileExists(p) || a == "ctlite" {
			if err := linkOrCopy(target, p); err != nil {
				fmt.Fprintf(out, "  ! could not update %s: %v\n", a, err)
			}
		}
	}
	fmt.Fprintln(out, "  ✓ updated "+target+" (and its aliases)")
	return nil
}

// offerUpdate asks once per build whether to replace an older installed copy.
func offerUpdate(in io.Reader, out io.Writer) {
	target, stale := staleInstall()
	if !stale {
		return
	}
	self, _ := os.Executable()
	sum := fileSHA(self)
	state := stateFile("update-declined")
	if b, err := os.ReadFile(state); err == nil && strings.TrimSpace(string(b)) == sum {
		return
	}
	ti, _ := os.Stat(target)
	si, _ := os.Stat(self)
	fmt.Fprintf(out, "The catude on your PATH is older than this one:\n  installed  %s  (%s)\n  this one   %s  (%s)\n",
		target, ti.ModTime().Format("2006-01-02 15:04"), self, si.ModTime().Format("2006-01-02 15:04"))
	if ask(bufio.NewReader(in), out, "Update the installed copy, so catude and ct open this version?", true) {
		if err := updateInstalled(out); err != nil {
			fmt.Fprintln(out, "  ! update failed:", err)
		}
		return
	}
	if state != "" {
		os.MkdirAll(filepath.Dir(state), 0o755)
		os.WriteFile(state, []byte(sum+"\n"), 0o644)
	}
}

// buildInfo describes this binary: version, commit and build time when Go recorded them.
func buildInfo() string {
	s := version
	if bi, ok := debug.ReadBuildInfo(); ok {
		var rev, when, dirty string
		for _, kv := range bi.Settings {
			switch kv.Key {
			case "vcs.revision":
				rev = kv.Value
			case "vcs.time":
				when = kv.Value
			case "vcs.modified":
				if kv.Value == "true" {
					dirty = "+changes"
				}
			}
		}
		if rev != "" {
			if len(rev) > 7 {
				rev = rev[:7]
			}
			s += " (" + rev + dirty + " " + when + ")"
		}
	}
	return s
}
