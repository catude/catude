package sessions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DesktopSession is what Claude Desktop records about a conversation of its Code tab: its own title and
// whether you archived it. Claude Desktop keeps one small JSON file per conversation
// (claude-code-sessions/<account>/<org>/local_<id>.json); cliSessionId is the Claude Code session whose
// transcript is in ~/.claude/projects. Read-only: catude never writes there.
type DesktopSession struct {
	Title    string
	Archived bool
}

// DesktopSessions maps Claude Code session ids to what Claude Desktop knows about them. dataDirs are
// Claude Desktop's data folders (where its claude-code-sessions folder lives).
func DesktopSessions(dataDirs []string) map[string]DesktopSession {
	out := map[string]DesktopSession{}
	for _, d := range dataDirs {
		files, _ := filepath.Glob(filepath.Join(d, "claude-code-sessions", "*", "*", "local_*.json"))
		for _, f := range files {
			b, err := readCappedFile(f, 1<<20)
			if err != nil {
				continue
			}
			var s struct {
				CliSessionID string `json:"cliSessionId"`
				Title        string `json:"title"`
				IsArchived   bool   `json:"isArchived"`
			}
			if json.Unmarshal(b, &s) != nil || !reSessionID.MatchString(s.CliSessionID) {
				continue
			}
			out[s.CliSessionID] = DesktopSession{Title: strings.Join(strings.Fields(s.Title), " "), Archived: s.IsArchived}
		}
	}
	return out
}

func readCappedFile(p string, max int64) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b := make([]byte, max)
	n, _ := f.Read(b)
	return b[:n], nil
}
