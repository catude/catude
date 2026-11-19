package tui

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/catude/catude/internal/audit"
	"github.com/catude/catude/internal/inventory"
	"github.com/catude/catude/internal/livemcp"
)

// fingerprint summarises an inventory so unchanged reloads skip the audit and keep cached rows.
func fingerprint(inv inventory.Inventory) string {
	h := sha1.New()
	for _, it := range inv.Items {
		fmt.Fprintf(h, "%s\x00%s\x00%s\x00%s\x00%d\x00%v\x00%s\x00%d\n", it.Kind, it.Name, it.Scope, it.Path, len(it.Content), it.Enabled, it.Description, len(it.Problems))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// cleanInventory removes emoji and invisible characters from what is displayed (see clean).
func cleanInventory(inv *inventory.Inventory, fs []audit.Finding) {
	for i := range inv.Items {
		it := &inv.Items[i]
		it.Name, it.Description, it.Origin = clean(it.Name), clean(it.Description), clean(it.Origin)
		for j := range it.Problems {
			it.Problems[j] = clean(it.Problems[j])
		}
		for j := range it.Detail {
			it.Detail[j] = clean(it.Detail[j])
		}
	}
	for i := range fs {
		fs[i].Item, fs[i].Detail = clean(fs[i].Item), clean(fs[i].Detail)
	}
}

// rowCache keeps the rows of the current list between frames (building them was the main cost of a
// key press on large inventories).
type rowCache struct {
	key  string
	rows []row
}

type liveCached struct {
	Servers []livemcp.Server
	At      time.Time
}

func (m model) liveCacheName() string { return "mcp-" + cacheKey(m.env.ProjectDir, m.env.ClaudeDir) }

// loadLiveCache shows the last known claude.ai connectors and MCP health instantly.
func (m *model) loadLiveCache() {
	var c liveCached
	if readCache(m.liveCacheName(), &c) && time.Since(c.At) < 24*time.Hour {
		m.liveMCP, m.liveAt = c.Servers, c.At
	}
}

func (m model) saveLiveCache() { writeCache(m.liveCacheName(), liveCached{m.liveMCP, m.liveAt}) }

// inventoryItem builds a skill item (used by tests).
func inventoryItem(name string) inventory.Item {
	return inventory.Item{Kind: inventory.Skill, Name: name, Scope: "user", Enabled: true, Invocable: true}
}
