package sessions

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// QuickInfo reads only the beginning and the end of a transcript: enough for the working directory and
// the title, without parsing the whole file (transcripts can be hundreds of MB).
func QuickInfo(path string) (cwd, title string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()
	head := make([]byte, 256<<10)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	cwd = jsonString(head, `"cwd":"`)
	title = lastJSONString(head, `"aiTitle":"`)
	if st, err := f.Stat(); err == nil && st.Size() > int64(n) {
		tail := make([]byte, 256<<10)
		off := st.Size() - int64(len(tail))
		if off < int64(n) {
			off = int64(n)
		}
		m, _ := f.ReadAt(tail, off)
		if t := lastJSONString(tail[:m], `"aiTitle":"`); t != "" {
			title = t
		}
		if cwd == "" {
			cwd = jsonString(tail[:m], `"cwd":"`)
		}
	}
	return cwd, title
}

func jsonString(b []byte, key string) string {
	i := bytes.Index(b, []byte(key))
	if i < 0 {
		return ""
	}
	return decodeAt(b[i+len(key)-1:])
}

func lastJSONString(b []byte, key string) string {
	i := bytes.LastIndex(b, []byte(key))
	if i < 0 {
		return ""
	}
	return decodeAt(b[i+len(key)-1:])
}

// decodeAt decodes the JSON string starting at b[0] == '"'.
func decodeAt(b []byte) string {
	for j := 1; j < len(b); j++ {
		if b[j] == '\\' {
			j++
			continue
		}
		if b[j] == '"' {
			var s string
			if json.Unmarshal(b[:j+1], &s) == nil {
				return s
			}
			return ""
		}
	}
	return ""
}

// ---- messages ----

// Message is one turn of a conversation, reduced to what a person reads.
type Message struct {
	Role string // user | assistant | tool
	Time time.Time
	Text string
}

type rawLine struct {
	Type        string          `json:"type"`
	IsSidechain bool            `json:"isSidechain"`
	IsMeta      bool            `json:"isMeta"`
	UUID        string          `json:"uuid"`
	Timestamp   string          `json:"timestamp"`
	Message     json.RawMessage `json:"message"`
}

type rawMessage struct {
	ID      string          `json:"id"`
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type block struct {
	Type    string          `json:"type"`
	Text    string          `json:"text"`
	Name    string          `json:"name"`
	Input   json.RawMessage `json:"input"`
	Content json.RawMessage `json:"content"`
}

// Messages reads the main thread of a transcript (subagents and meta lines excluded). Tool output is
// summarised to its first lines when withTools is false.
func Messages(path string, withTools bool) ([]Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	br := bufio.NewReaderSize(f, 1<<20)
	var out []Message
	seen := map[string]bool{}
	for {
		raw, err := readLine(br)
		if len(raw) > 0 {
			if m, ok := toMessage(raw, withTools, seen); ok {
				out = append(out, m...)
			}
		}
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil && !errors.Is(err, errLineTooLong) {
			return out, err
		}
	}
}

func toMessage(raw []byte, withTools bool, seen map[string]bool) ([]Message, bool) {
	var l rawLine
	if json.Unmarshal(raw, &l) != nil || l.IsSidechain || l.IsMeta || (l.Type != "user" && l.Type != "assistant") {
		return nil, false
	}
	if l.UUID != "" {
		if seen[l.UUID] {
			return nil, false
		}
		seen[l.UUID] = true
	}
	var m rawMessage
	if json.Unmarshal(l.Message, &m) != nil {
		return nil, false
	}
	ts, _ := time.Parse(time.RFC3339Nano, l.Timestamp)
	var s string
	if json.Unmarshal(m.Content, &s) == nil {
		if strings.HasPrefix(s, "<command-") || strings.HasPrefix(s, "<local-command") {
			return nil, false
		}
		return []Message{{Role: l.Type, Time: ts, Text: s}}, true
	}
	var blocks []block
	if json.Unmarshal(m.Content, &blocks) != nil {
		return nil, false
	}
	var out []Message
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if strings.TrimSpace(b.Text) != "" {
				out = append(out, Message{Role: l.Type, Time: ts, Text: b.Text})
			}
		case "tool_use":
			in := string(b.Input)
			if !withTools && len(in) > 200 {
				in = in[:200] + "…"
			}
			out = append(out, Message{Role: "tool", Time: ts, Text: "→ " + b.Name + " " + in})
		case "tool_result":
			if !withTools {
				continue
			}
			var txt string
			if json.Unmarshal(b.Content, &txt) != nil {
				var parts []block
				if json.Unmarshal(b.Content, &parts) == nil {
					for _, p := range parts {
						txt += p.Text
					}
				}
			}
			out = append(out, Message{Role: "tool", Time: ts, Text: txt})
		}
	}
	return out, len(out) > 0
}

// Markdown renders a conversation for reading or sharing. Review it before sharing: it contains
// everything that was said, including file contents Claude read.
func Markdown(s Session, msgs []Message) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", firstNonEmpty(s.Title, s.ID))
	fmt.Fprintf(&b, "- Session: `%s`\n- Folder: `%s`\n- Model: %s\n- From %s to %s\n\n---\n\n", s.ID, s.CWD, firstNonEmpty(s.Model, "?"), s.Start.Local().Format("2006-01-02 15:04"), s.End.Local().Format("2006-01-02 15:04"))
	for _, m := range msgs {
		switch m.Role {
		case "user":
			b.WriteString("## You\n\n" + m.Text + "\n\n")
		case "assistant":
			b.WriteString("## Claude\n\n" + m.Text + "\n\n")
		default:
			b.WriteString("```text\n" + strings.TrimSpace(m.Text) + "\n```\n\n")
		}
	}
	return b.String()
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// ---- search ----

// Hit is a search match.
type Hit struct {
	Path    string
	Project string // working directory
	Title   string
	Snippet string
	Time    time.Time
}

// Search looks for text in every transcript, newest first, case-insensitively, and stops at limit.
// It reads raw lines and only parses the ones that match, so it is fast on large histories.
func Search(ctx context.Context, projects []Project, query string, limit int) ([]Hit, error) {
	q := bytes.ToLower([]byte(strings.TrimSpace(query)))
	if len(q) < 2 {
		return nil, errors.New("type at least 2 characters")
	}
	type file struct {
		path string
		mod  time.Time
	}
	var files []file
	for _, p := range projects {
		for _, f := range p.Sessions {
			if st, err := os.Stat(f); err == nil {
				files = append(files, file{f, st.ModTime()})
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
	var hits []Hit
	for _, f := range files {
		if ctx.Err() != nil {
			return hits, ctx.Err()
		}
		h, ok := searchFile(f.path, q)
		if !ok {
			continue
		}
		h.Time = f.mod
		h.Project, h.Title = QuickInfo(f.path)
		hits = append(hits, h)
		if len(hits) >= limit {
			break
		}
	}
	return hits, nil
}

func searchFile(path string, q []byte) (Hit, bool) {
	fh, err := os.Open(path)
	if err != nil {
		return Hit{}, false
	}
	defer fh.Close()
	br := bufio.NewReaderSize(fh, 1<<20)
	seen := map[string]bool{}
	for {
		raw, err := readLine(br)
		if len(raw) > 0 && bytes.Contains(bytes.ToLower(raw), q) {
			if ms, ok := toMessage(raw, false, seen); ok {
				for _, m := range ms {
					if m.Role == "tool" {
						continue
					}
					if i := strings.Index(strings.ToLower(m.Text), string(q)); i >= 0 {
						return Hit{Path: path, Snippet: snippet(m.Text, i, len(q))}, true
					}
				}
			}
		}
		if err != nil && !errors.Is(err, errLineTooLong) {
			return Hit{}, false
		}
	}
}

func snippet(s string, i, n int) string {
	start, end := max(0, i-60), min(len(s), i+n+80)
	for start > 0 && !utf8.RuneStart(s[start]) {
		start--
	}
	for end < len(s) && !utf8.RuneStart(s[end]) {
		end++
	}
	out := strings.Join(strings.Fields(s[start:end]), " ")
	if start > 0 {
		out = "…" + out
	}
	if end < len(s) {
		out += "…"
	}
	return out
}

// ---- trash ----

var reSessionID = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z_-]{0,63}$`)

// ValidID reports whether a transcript name is a session id (so it can never be read as a command-line
// option such as --settings=… or as a path).
func ValidID(id string) bool { return reSessionID.MatchString(id) }

// LocalPath reports whether p is an ordinary local path: not a UNC or device path, which on Windows
// would make the machine authenticate against whoever owns the host name.
func LocalPath(p string) bool {
	return p != "" && !strings.HasPrefix(p, `\\`) && !strings.HasPrefix(p, "//")
}

// Trash moves a session (transcript, its subagent folder and its checkpoints) to a trash folder, so
// it disappears from /resume but can be restored by moving the files back. It returns the trash folder.
func Trash(claudeDir, transcript, trashRoot string) (string, error) {
	id := strings.TrimSuffix(filepath.Base(transcript), ".jsonl")
	if !ValidID(id) {
		return "", fmt.Errorf("invalid session %q", transcript)
	}
	dst := filepath.Join(trashRoot, time.Now().Format("20060102-150405")+"-"+id)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return "", err
	}
	moves := [][2]string{
		{transcript, filepath.Join(dst, "transcript", filepath.Base(filepath.Dir(transcript)), filepath.Base(transcript))},
		{strings.TrimSuffix(transcript, ".jsonl"), filepath.Join(dst, "transcript", filepath.Base(filepath.Dir(transcript)), id)},
		{filepath.Join(claudeDir, "file-history", id), filepath.Join(dst, "file-history", id)},
	}
	for _, mv := range moves {
		if _, err := os.Stat(mv[0]); err != nil {
			continue
		}
		os.MkdirAll(filepath.Dir(mv[1]), 0o755)
		if err := os.Rename(mv[0], mv[1]); err != nil {
			return dst, err
		}
	}
	readme := "Moved by catude on " + time.Now().Format(time.RFC3339) + ".\nTo restore, move transcript/<project>/* back to " +
		filepath.Dir(transcript) + " and file-history/* back to " + filepath.Join(claudeDir, "file-history") + ".\n"
	os.WriteFile(filepath.Join(dst, "RESTORE.txt"), []byte(readme), 0o644)
	return dst, nil
}
