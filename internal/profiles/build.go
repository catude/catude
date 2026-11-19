package profiles

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/catude/catude/internal/audit"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/paths"
	"github.com/catude/catude/internal/rigcard"
)

// Limits for everything a profile brings in.
const (
	MaxFile      = 2 << 20  // one file
	MaxArtifact  = 10 << 20 // one skill folder
	MaxDownload  = 64 << 20 // one repository tarball
	FetchTimeout = 2 * time.Minute
)

// More limits for downloaded archives.
const (
	MaxUnpacked = 1 << 30 // bytes read from a tarball after decompression
	MaxEntries  = 200_000 // entries walked in a tarball
)

var reservedWin = map[string]bool{"con": true, "prn": true, "aux": true, "nul": true, "com1": true, "com2": true, "com3": true, "com4": true,
	"com5": true, "com6": true, "com7": true, "com8": true, "com9": true, "lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true,
	"lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true, "conin$": true, "conout$": true}

// SafeRelPath checks a slash-separated relative path from an archive or a card so that it cannot leave
// its folder or mean something special on any operating system: no absolute paths, no "..", no
// backslashes or drive letters, no Windows alternate data streams ("a:b"), no reserved device names
// (CON, NUL, COM1…), no names ending in a dot or space (Windows strips them), no control characters.
func SafeRelPath(rel string) error {
	if rel == "" || len(rel) > 400 || strings.HasPrefix(rel, "/") {
		return errors.New("empty, absolute or too long")
	}
	for _, seg := range strings.Split(rel, "/") {
		switch {
		case seg == "" || seg == "." || seg == "..":
			return errors.New("empty or dot segment")
		case strings.ContainsAny(seg, "\\:<>\"|?*"):
			return errors.New("character not allowed in a file name")
		case strings.HasSuffix(seg, ".") || strings.HasSuffix(seg, " "):
			return errors.New("name ends in a dot or a space")
		}
		for _, r := range seg {
			if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) || (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) {
				return errors.New("control or bidi character in a file name")
			}
		}
		base := strings.ToLower(seg)
		if i := strings.IndexByte(base, '.'); i >= 0 {
			base = base[:i]
		}
		if reservedWin[base] {
			return errors.New("reserved device name on Windows")
		}
	}
	return nil
}

// ErrRejected means the card failed validation and nothing was installed.
var ErrRejected = errors.New("the profile card was rejected")

// Fetcher downloads an https URL (rigcard.Fetch in production, a fake in tests).
type Fetcher func(ctx context.Context, url string, max int64, timeout time.Duration) ([]byte, error)

// BuildOptions control how a card becomes a profile.
type BuildOptions struct {
	Raw           []byte            // the card as downloaded, for validation (nil for local profiles)
	Local         map[string]string // "skill:name" → folder, "agent:name" / "command:name" → file, copied as is
	AllowUnpinned bool              // accept sources not pinned to a commit and without sha256
	Fetch         Fetcher
	Inventory     *inventory.Inventory // this machine, to tell which plugins are missing
	Origin        rigcard.Origin       // where the card came from: bundled files are read next to it
}

// Staged is a profile built in a temporary folder, waiting for the user's consent.
type Staged struct {
	Name     string
	Dir      string
	Card     rigcard.Card
	Problems []rigcard.Problem
	Findings []audit.Finding
	Notes    []string // what was skipped and what the user must do
	Scripts  []string // files that can be executed (relative paths)
	Secrets  []string // environment variables the user must set
	Missing  []string // official commands for plugins/marketplaces not installed here
}

// HighFindings counts high-severity audit findings.
func (s *Staged) HighFindings() int {
	n := 0
	for _, f := range s.Findings {
		if f.Severity == audit.High {
			n++
		}
	}
	return n
}

// Build validates a card, fetches and verifies its artifacts into a staging folder and audits the result.
// Nothing is installed until Commit.
func Build(ctx context.Context, name string, c rigcard.Card, o BuildOptions) (*Staged, error) {
	if !rigcard.SafeName(name) || strings.ContainsAny(name, " :*?\"<>|") {
		return nil, fmt.Errorf("invalid profile name %q: use letters, digits, - and _", name)
	}
	s := &Staged{Name: name, Card: c, Problems: rigcard.Validate(c, o.Raw)}
	if rigcard.Rejected(s.Problems) {
		return s, ErrRejected
	}
	if o.Fetch == nil {
		o.Fetch = rigcard.Fetch
	}
	if err := os.MkdirAll(Root(), 0o755); err != nil {
		return s, err
	}
	dir, err := os.MkdirTemp(Root(), ".staging-"+name+"-")
	if err != nil {
		return s, err
	}
	s.Dir = dir
	fail := func(err error) (*Staged, error) { s.Discard(); return s, err }

	plugin := filepath.Join(dir, "plugin")
	manifest := map[string]string{"name": "profile-" + strings.ToLower(name), "description": "catude work profile " + name, "version": "0.0.0"}
	if err := writeJSON(filepath.Join(plugin, ".claude-plugin", "plugin.json"), manifest); err != nil {
		return fail(err)
	}
	tarballs := map[string][]byte{}
	for _, group := range []struct {
		kind  string
		items []rigcard.Artifact
		dir   string
		isDir bool
	}{{"skill", c.Skills, "skills", true}, {"agent", c.Agents, "agents", false}, {"command", c.Commands, "commands", false}} {
		for _, a := range group.items {
			if !rigcard.SafeName(a.Name) {
				return fail(fmt.Errorf("%s %q: unsafe name", group.kind, a.Name))
			}
			dst := filepath.Join(plugin, group.dir, a.Name)
			if !group.isDir {
				dst += ".md"
			}
			done, err := s.materialize(ctx, o, tarballs, group.kind, a, dst, group.isDir)
			if err != nil {
				return fail(err)
			}
			if !done {
				continue
			}
			if a.SHA256 != "" {
				got, err := hashOf(dst, group.isDir)
				if err != nil {
					return fail(err)
				}
				if !strings.EqualFold(got, a.SHA256) {
					return fail(fmt.Errorf("%s %s: sha256 mismatch (card %s…, downloaded %s…): the source changed after the card was written", group.kind, a.Name, short(a.SHA256), short(got)))
				}
			}
		}
	}
	if err := s.writeMCP(filepath.Join(dir, "mcp.json")); err != nil {
		return fail(err)
	}
	if err := s.writeSettings(filepath.Join(dir, "settings.json")); err != nil {
		return fail(err)
	}
	if len(c.Hooks) > 0 {
		s.Notes = append(s.Notes, fmt.Sprintf("%d hooks described in the card were not installed: profiles never install hooks", len(c.Hooks)))
	}
	if c.Desktop != nil && len(c.Desktop.Connectors) > 0 {
		s.Notes = append(s.Notes, "claude.ai connectors to add in your account (Settings → Connectors): "+strings.Join(c.Desktop.Connectors, ", "))
	}
	if o.Inventory != nil {
		for _, st := range rigcard.Plan(rigcard.Card{Rigcard: c.Rigcard, Marketplaces: c.Marketplaces, Plugins: c.Plugins}, *o.Inventory) {
			if st.Cmd != "" && st.State != "have" {
				s.Missing = append(s.Missing, st.Cmd)
			}
		}
	}
	if err := writeJSON(filepath.Join(dir, cardFile), c); err != nil {
		return fail(err)
	}
	s.Scripts = scripts(plugin)
	s.Findings = auditDir(dir)
	return s, nil
}

// materialize copies or downloads one artifact. It returns false when the artifact was skipped (with a note).
func (s *Staged) materialize(ctx context.Context, o BuildOptions, tarballs map[string][]byte, kind string, a rigcard.Artifact, dst string, isDir bool) (bool, error) {
	if src, ok := o.Local[kind+":"+a.Name]; ok {
		if isDir {
			return true, copyTree(src, dst)
		}
		return true, copyFile(src, dst)
	}
	if a.Source == nil || a.Source.Type == "local" {
		s.Notes = append(s.Notes, fmt.Sprintf("%s %s skipped: it is the author's local copy, not downloadable", kind, a.Name))
		return false, nil
	}
	src := *a.Source
	if src.Type == "bundle" {
		return s.fromBundle(ctx, o, tarballs, kind, a, dst, isDir)
	}
	if src.URL != "" && src.Repo == "" && a.SHA256 == "" {
		return false, fmt.Errorf("%s %s comes from a URL without a sha256 in the card: nothing ties the file to what was reviewed", kind, a.Name)
	}
	pinned := src.Pinned() || a.SHA256 != ""
	if !pinned && !o.AllowUnpinned {
		return false, fmt.Errorf("%s %s is not pinned to a commit and has no sha256: its content could change after you review it (use --allow-unpinned to accept)", kind, a.Name)
	}
	switch {
	case src.Repo != "" && (src.Type == "" || src.Type == "github" || src.Source == "github"):
		ref := firstNonEmpty(src.Revision(), "HEAD")
		return true, s.fromGitHub(ctx, o.Fetch, tarballs, src.Repo, ref, src.Path, dst, isDir)
	case strings.HasPrefix(src.URL, "https://") && !isDir:
		b, err := o.Fetch(ctx, src.URL, MaxFile, FetchTimeout)
		if err != nil {
			return false, fmt.Errorf("%s %s: %w", kind, a.Name, err)
		}
		return true, writeFile(dst, b)
	}
	s.Notes = append(s.Notes, fmt.Sprintf("%s %s skipped: unsupported source (use a GitHub repo + path + commit)", kind, a.Name))
	return false, nil
}

// fromBundle reads a file that travels with the card: from the folder of a local card, or from the
// same GitHub repository and revision the card was downloaded from. The card's sha256 is mandatory and
// is verified by Build after copying, so a bundle cannot be altered without the card noticing.
func (s *Staged) fromBundle(ctx context.Context, o BuildOptions, tarballs map[string][]byte, kind string, a rigcard.Artifact, dst string, isDir bool) (bool, error) {
	rel := strings.Trim(a.Source.Path, "/")
	if err := SafeRelPath(rel); err != nil {
		return false, fmt.Errorf("%s %s: unsafe bundle path %q: %v", kind, a.Name, a.Source.Path, err)
	}
	if a.SHA256 == "" {
		return false, fmt.Errorf("%s %s: a bundled file needs a sha256 in the card", kind, a.Name)
	}
	switch o.Origin.Kind {
	case "file":
		src := filepath.Join(o.Origin.Dir, filepath.FromSlash(rel))
		if isDir {
			return true, copyTree(src, dst)
		}
		return true, copyFile(src, dst)
	case "github":
		p := rel
		if o.Origin.Base != "" {
			p = o.Origin.Base + "/" + rel
		}
		return true, s.fromGitHub(ctx, o.Fetch, tarballs, o.Origin.Repo, firstNonEmpty(o.Origin.Ref, "HEAD"), p, dst, isDir)
	}
	s.Notes = append(s.Notes, fmt.Sprintf("%s %s skipped: bundled files can be loaded from a folder or from a GitHub repository (gh:owner/repo), not from a plain web page", kind, a.Name))
	return false, nil
}

var (
	reRepo = regexp.MustCompile(`^[\w.-]+/[\w.-]+$`)
	reRef  = regexp.MustCompile(`^[\w./-]+$`)
)

// fromGitHub extracts one path of a repository at a revision, from the codeload tarball.
func (s *Staged) fromGitHub(ctx context.Context, fetch Fetcher, cache map[string][]byte, repo, ref, sub, dst string, isDir bool) error {
	sub = strings.Trim(path.Clean("/"+strings.ReplaceAll(sub, `\`, "/")), "/")
	if !reRepo.MatchString(repo) || !reRef.MatchString(ref) || strings.Contains(ref, "..") {
		return fmt.Errorf("invalid GitHub source %s@%s", repo, ref)
	}
	key := repo + "@" + ref
	tgz, ok := cache[key]
	if !ok {
		var err error
		tgz, err = fetch(ctx, "https://codeload.github.com/"+repo+"/tar.gz/"+ref, MaxDownload, FetchTimeout)
		if err != nil {
			return fmt.Errorf("downloading %s: %w", key, err)
		}
		cache[key] = tgz
	}
	gz, err := gzip.NewReader(bytes.NewReader(tgz))
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	// a small download can unpack to a huge stream (decompression bomb): cap what is read and how many
	// entries are walked, whatever the archive claims
	tr := tar.NewReader(io.LimitReader(gz, MaxUnpacked))
	var total int64
	found, skipped, entries := false, 0, 0
	seen := map[string]string{} // lower-cased path → real path: two names that differ only in case would overwrite each other on Windows and macOS
	for {
		h, err := tr.Next()
		if entries++; entries > MaxEntries {
			return fmt.Errorf("%s: more than %d entries in the archive", key, MaxEntries)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		name := h.Name
		if i := strings.IndexByte(name, '/'); i >= 0 {
			name = name[i+1:] // drop "<repo>-<sha>/"
		} else {
			continue
		}
		var rel string
		switch {
		case !isDir && name == sub:
			rel = ""
		case isDir && (sub == "" || strings.HasPrefix(name, sub+"/")):
			rel = strings.TrimPrefix(strings.TrimPrefix(name, sub), "/")
		default:
			continue
		}
		if rel != "" {
			if err := SafeRelPath(rel); err != nil {
				return fmt.Errorf("%s: unsafe path %q in archive: %v", key, h.Name, err)
			}
			if prev, dup := seen[strings.ToLower(rel)]; dup && prev != rel {
				return fmt.Errorf("%s: unsafe path %q in archive: collides with %q on case-insensitive file systems", key, h.Name, prev)
			}
			seen[strings.ToLower(rel)] = rel
		}
		switch h.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
		default:
			skipped++ // symlinks, devices…: never extracted
			continue
		}
		if h.Size > MaxFile {
			return fmt.Errorf("%s: %s is larger than %d MiB", key, name, MaxFile>>20)
		}
		total += h.Size
		if total > MaxArtifact {
			return fmt.Errorf("%s/%s is larger than %d MiB", key, sub, MaxArtifact>>20)
		}
		b, err := io.ReadAll(io.LimitReader(tr, MaxFile+1))
		if err != nil {
			return err
		}
		out := dst
		if isDir {
			out = filepath.Join(dst, filepath.FromSlash(rel))
		}
		if err := writeFile(out, b); err != nil {
			return err
		}
		found = true
	}
	if !found {
		return fmt.Errorf("%s: path %q not found in the repository", key, sub)
	}
	if skipped > 0 {
		s.Notes = append(s.Notes, fmt.Sprintf("%s/%s: %d links or special files were not extracted", repo, sub, skipped))
	}
	return nil
}

func (s *Staged) writeMCP(p string) error {
	servers := map[string]any{}
	for _, m := range s.Card.MCPServers {
		if !rigcard.SafeName(m.Name) {
			return fmt.Errorf("mcp %q: unsafe name", m.Name)
		}
		if m.Connector {
			continue
		}
		val := func(v rigcard.Variable) string {
			if v.Default != "" && !v.IsSecret && !strings.Contains(v.Default, "$") {
				return v.Default
			}
			if v.IsSecret || v.IsRequired {
				s.Secrets = append(s.Secrets, envName(v.Name))
			}
			return "${" + envName(v.Name) + "}"
		}
		if m.Transport != "stdio" && m.URL != "" {
			if !strings.HasPrefix(m.URL, "https://") && !strings.Contains(m.URL, "localhost") && !strings.Contains(m.URL, "127.0.0.1") {
				s.Notes = append(s.Notes, "mcp "+m.Name+" uses plain http")
			}
			srv := map[string]any{"type": m.Transport, "url": m.URL}
			if len(m.Headers) > 0 {
				h := map[string]string{}
				for _, v := range m.Headers {
					v.Default = "" // a header is always a reference to a variable you set
					h[v.Name] = val(v)
				}
				srv["headers"] = h
			}
			servers[m.Name] = srv
			continue
		}
		if m.Command == "" {
			s.Notes = append(s.Notes, "mcp "+m.Name+" skipped: no command or url")
			continue
		}
		srv := map[string]any{"command": m.Command, "args": m.Args}
		if len(m.Env) > 0 {
			e := map[string]string{}
			for _, v := range m.Env {
				e[v.Name] = val(v)
			}
			srv["env"] = e
		}
		servers[m.Name] = srv
	}
	sort.Strings(s.Secrets)
	s.Secrets = uniq(s.Secrets)
	if len(servers) == 0 {
		return nil
	}
	return writeJSON(p, map[string]any{"mcpServers": servers})
}

// writeSettings keeps only the allow-listed settings; "allow" rules, env, hooks and helpers are never written.
func (s *Staged) writeSettings(p string) error {
	out := map[string]any{}
	if st := s.Card.Settings; st != nil {
		if st.Model != "" {
			out["model"] = st.Model
		}
		if st.EffortLevel != "" {
			out["effortLevel"] = st.EffortLevel
		}
		if st.OutputStyle != "" {
			out["outputStyle"] = st.OutputStyle
		}
		if st.Permissions != nil && len(st.Permissions.Deny)+len(st.Permissions.Ask) > 0 {
			perm := map[string][]string{}
			if len(st.Permissions.Deny) > 0 {
				perm["deny"] = st.Permissions.Deny
			}
			if len(st.Permissions.Ask) > 0 {
				perm["ask"] = st.Permissions.Ask
			}
			out["permissions"] = perm
		}
	}
	if len(s.Card.Plugins) > 0 {
		en := map[string]bool{}
		for _, pl := range s.Card.Plugins {
			en[pl.ID] = pl.Enabled
		}
		out["enabledPlugins"] = en
	}
	if len(out) == 0 {
		return nil
	}
	return writeJSON(p, out)
}

// Commit installs the staged profile. With replace, an existing profile of the same name is replaced
// (its isolated login folder is kept).
func (s *Staged) Commit(replace bool) (Profile, error) {
	final := filepath.Join(Root(), s.Name)
	if _, err := os.Stat(final); err == nil {
		if !replace {
			return Profile{}, fmt.Errorf("a profile named %q already exists (use --replace)", s.Name)
		}
		old := final + ".old-" + time.Now().Format("20060102150405")
		if err := os.Rename(final, old); err != nil {
			return Profile{}, err
		}
		if err := os.Rename(s.Dir, final); err != nil {
			os.Rename(old, final)
			return Profile{}, err
		}
		keepOld := false
		if _, err := os.Stat(filepath.Join(old, "home")); err == nil {
			// the isolated login must survive: if it cannot be moved, the old folder is kept
			keepOld = os.Rename(filepath.Join(old, "home"), filepath.Join(final, "home")) != nil
		}
		if !keepOld {
			os.RemoveAll(old)
		}
	} else if err := os.Rename(s.Dir, final); err != nil {
		return Profile{}, err
	}
	s.Dir = ""
	return Get(s.Name)
}

// Discard deletes the staging folder.
func (s *Staged) Discard() {
	if s != nil && s.Dir != "" && strings.Contains(filepath.Base(s.Dir), ".staging-") {
		os.RemoveAll(s.Dir)
		s.Dir = ""
	}
}

// auditDir runs the security audit over the staged files only.
func auditDir(dir string) []audit.Finding {
	env := paths.Env{GOOS: "linux", Home: dir, ClaudeDir: filepath.Join(dir, "plugin"), ClaudeJSON: filepath.Join(dir, "mcp.json"), ProjectDir: filepath.Join(dir, "plugin")}
	var out []audit.Finding
	for _, f := range audit.Run(inventory.Load(env)) {
		if f.Path == "" || strings.HasPrefix(filepath.Clean(f.Path), filepath.Clean(dir)) {
			out = append(out, f)
		}
	}
	return out
}

var scriptExt = map[string]bool{".sh": true, ".bash": true, ".zsh": true, ".ps1": true, ".psm1": true, ".bat": true, ".cmd": true, ".py": true, ".js": true, ".mjs": true, ".cjs": true, ".ts": true, ".rb": true, ".pl": true, ".exe": true, ".dll": true, ".so": true, ".dylib": true, ".jar": true}

func scripts(root string) []string {
	var out []string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && scriptExt[strings.ToLower(filepath.Ext(p))] {
			r, _ := filepath.Rel(root, p)
			out = append(out, filepath.ToSlash(r))
		}
		return nil
	})
	return out
}

func hashOf(p string, isDir bool) (string, error) {
	if isDir {
		return rigcard.TreeHash(p)
	}
	return rigcard.FileHash(p)
}

func copyTree(src, dst string) error {
	var total int64
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil // never follow links out of the folder
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		if total += fi.Size(); fi.Size() > MaxFile || total > MaxArtifact {
			return fmt.Errorf("%s is too large for a profile", src)
		}
		r, _ := filepath.Rel(src, p)
		return copyFile(p, filepath.Join(dst, r))
	})
}

func copyFile(src, dst string) error {
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() || fi.Size() > MaxFile {
		return fmt.Errorf("%s: not a regular file or too large", src)
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return writeFile(dst, b)
}

// writeFile writes a non-executable file.
func writeFile(p string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func writeJSON(p string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(p, append(b, '\n'))
}

func envName(h string) string {
	return strings.ToUpper(strings.NewReplacer("-", "_", " ", "_", ".", "_").Replace(h))
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

func uniq(s []string) []string {
	var out []string
	for i, v := range s {
		if i == 0 || v != s[i-1] {
			out = append(out, v)
		}
	}
	return out
}

func jsonIndent(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
