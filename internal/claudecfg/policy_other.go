//go:build !windows

package claudecfg

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
)

// Claude Desktop policies (support article 12622667): Linux /etc/claude-desktop/managed-settings.json,
// macOS the managed preferences of com.anthropic.claudefordesktop (a property list catude does not parse:
// only the documented keys are listed there).
func desktopPolicyPath() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Managed Preferences/com.anthropic.claudefordesktop.plist"
	}
	return "/etc/claude-desktop/managed-settings.json"
}

func desktopPolicyWhere() string {
	if runtime.GOOS == "darwin" {
		return "managed preferences com.anthropic.claudefordesktop"
	}
	return "/etc/claude-desktop/managed-settings.json"
}

func readDesktopPolicy() map[string]string {
	out := map[string]string{}
	if runtime.GOOS == "darwin" {
		return out
	}
	b, err := os.ReadFile(desktopPolicyPath())
	if err != nil || len(b) > 4<<20 {
		return out
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return out
	}
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
		} else {
			bb, _ := json.Marshal(v)
			out[k] = fmt.Sprint(string(bb))
		}
	}
	return out
}
