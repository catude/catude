// Package prefs stores how the user wants catude to look: theme, glyphs, accent colour, which tabs are
// visible in the full view and in the lite view ("ctlite"), and small behaviour switches.
// The file is <user config dir>/catude/config.json (or $CATUDE_HOME/config.json).
package prefs

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Prefs is the saved configuration. Empty values mean "default".
type Prefs struct {
	Theme        string   `json:"theme,omitempty"`    // auto | dark | light | high-contrast | mono
	Glyphs       string   `json:"glyphs,omitempty"`   // auto | unicode | safe | ascii
	Accent       string   `json:"accent,omitempty"`   // clay (default) | amber | blue | green | magenta | cyan | red
	Tabs         []string `json:"tabs,omitempty"`     // tabs shown in the full view (ids); empty = DefaultTabs
	LiteTabs     []string `json:"liteTabs,omitempty"` // tabs shown in the lite view; empty = DefaultLite
	Lite         bool     `json:"lite,omitempty"`     // open in the lite view
	ReduceMotion bool     `json:"reduceMotion,omitempty"`
	HideHints    bool     `json:"hideHints,omitempty"` // hide the key hints in the bottom bar
	// Bell: ring the terminal bell when a Claude Code session finishes its turn and waits for you
	Bell bool `json:"bell,omitempty"`
	// Notify: also show a system notification for it
	Notify   bool   `json:"notify,omitempty"`
	Language string `json:"language,omitempty"` // interface language (i18n code); empty = auto
}

// DefaultTabs are the tabs of the full view until the user chooses: the ones most people need. The others
// (skills, agents, commands, plugins, hooks, available, tools, storage, audit) are switched on in preferences.
var DefaultTabs = []string{"overview", "config", "live", "context", "usage", "models", "profiles", "sessions", "graph", "mcp", "discover"}

// DefaultLite is what ctlite shows until the user chooses otherwise.
var DefaultLite = []string{"overview", "profiles", "context", "usage", "sessions"}

// Path is where preferences are stored.
func Path() string {
	if d := os.Getenv("CATUDE_HOME"); d != "" {
		return filepath.Join(d, "config.json")
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "catude", "config.json")
}

// Load reads the preferences; a missing or broken file gives the defaults.
func Load() Prefs {
	var p Prefs
	if b, err := os.ReadFile(Path()); err == nil {
		json.Unmarshal(b, &p)
	}
	return p
}

// Save writes the preferences atomically.
func Save(p Prefs) error {
	path := Path()
	if path == "" {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
