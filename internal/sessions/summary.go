package sessions

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/catude/catude/internal/redact"
)

// promptText is the text you typed in a user message ("" for tool results, command wrappers and
// messages Claude Code adds itself).
func promptText(content json.RawMessage) string {
	var s string
	if json.Unmarshal(content, &s) != nil {
		var blocks []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(content, &blocks) != nil {
			return ""
		}
		for _, b := range blocks {
			if b.Type == "text" {
				s = b.Text
				break
			}
		}
	}
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "<") || strings.HasPrefix(s, "Caveat:") || strings.HasPrefix(s, "[Request interrupted") {
		return ""
	}
	s = strings.Join(strings.Fields(redact.Text(s)), " ")
	if r := []rune(s); len(r) > 200 {
		s = string(r[:199]) + "…"
	}
	return s
}

// countTools adds the tool calls of an assistant message, and the files it edits or writes.
func countTools(s *Session, content json.RawMessage) {
	var blocks []struct {
		Type  string `json:"type"`
		Name  string `json:"name"`
		Input struct {
			FilePath     string `json:"file_path"`
			NotebookPath string `json:"notebook_path"`
		} `json:"input"`
	}
	if json.Unmarshal(content, &blocks) != nil {
		return
	}
	for _, b := range blocks {
		if b.Type != "tool_use" || b.Name == "" {
			continue
		}
		if s.Tools == nil {
			s.Tools = map[string]int{}
		}
		s.Tools[b.Name]++
		switch b.Name {
		case "Edit", "Write", "MultiEdit", "NotebookEdit":
			p := b.Input.FilePath
			if p == "" {
				p = b.Input.NotebookPath
			}
			if p != "" {
				if s.Files == nil {
					s.Files = map[string]int{}
				}
				s.Files[p]++
			}
		}
	}
}

// Rename gives a conversation a name, with the same line Claude Code appends for /rename. Rename a
// conversation only while no Claude Code session has it open: that session would keep its own name.
func Rename(path, title string) error {
	title = strings.Join(strings.Fields(title), " ")
	if title == "" {
		return errors.New("the name is empty")
	}
	if len([]rune(title)) > 120 {
		return errors.New("the name is longer than 120 characters")
	}
	id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	if !reSessionID.MatchString(id) {
		return errors.New("not a Claude Code conversation file")
	}
	b, err := json.Marshal(map[string]string{"type": "custom-title", "customTitle": title, "sessionId": id})
	if err != nil {
		return err
	}
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return errors.New("the conversation file is a link or a special file")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	// a file that does not end in a newline gets one first, so the line stays a line of its own
	if fi.Size() > 0 {
		r, err := os.Open(path)
		if err == nil {
			last := make([]byte, 1)
			r.ReadAt(last, fi.Size()-1)
			r.Close()
			if last[0] != '\n' {
				b = append([]byte("\n"), b...)
			}
		}
	}
	_, err = f.Write(append(b, '\n'))
	return err
}
