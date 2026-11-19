// Package accounts integrates account switchers for Claude Code. catude never reads or writes OAuth
// tokens itself: it drives claude-swap (realiti4/claude-swap, MIT) through its JSON interface
// (`list --json`, schemaVersion 1) and runs `switch` as a child process the user sees.
package accounts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PinnedVersion is the claude-swap release catude installs (≥ 2 weeks old when chosen).
const PinnedVersion = "0.29.0"

// Window is a usage window (5 h or 7 days) as reported by claude-swap.
type Window struct {
	Pct                   float64 `json:"pct"`
	ResetsAt              string  `json:"resetsAt"`
	Countdown             string  `json:"countdown"`
	ProjectedExhaustionAt string  `json:"projectedExhaustionAt"`
	WillLastToReset       *bool   `json:"willLastToReset"`
}

// Account is one Claude account managed by claude-swap.
type Account struct {
	Number           int    `json:"number"`
	Email            string `json:"email"`
	OrganizationName string `json:"organizationName"`
	IsOrganization   bool   `json:"isOrganization"`
	Active           bool   `json:"active"`
	UsageStatus      string `json:"usageStatus"`
	LoginExpiresAt   string `json:"loginExpiresAt"`
	LastGoodUsage    *struct {
		FiveHour *Window `json:"fiveHour"`
		SevenDay *Window `json:"sevenDay"`
	} `json:"lastGoodUsage"`
}

// FiveHour and SevenDay return the last known usage windows (nil when unknown).
func (a Account) FiveHour() *Window {
	if a.LastGoodUsage == nil {
		return nil
	}
	return a.LastGoodUsage.FiveHour
}

func (a Account) SevenDay() *Window {
	if a.LastGoodUsage == nil {
		return nil
	}
	return a.LastGoodUsage.SevenDay
}

// Switcher is an installed claude-swap.
type Switcher struct {
	Bin     string
	Version string
}

var reVersion = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// Detect finds claude-swap (`cswap` or `claude-swap`) on PATH.
func Detect(ctx context.Context) (Switcher, bool) {
	for _, n := range []string{"cswap", "claude-swap"} {
		p, err := exec.LookPath(n)
		if err != nil {
			continue
		}
		s := Switcher{Bin: p}
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		out, err := exec.CommandContext(ctx, p, "--version").Output()
		cancel()
		if err == nil {
			s.Version = reVersion.FindString(string(out))
		}
		return s, true
	}
	return Switcher{}, false
}

// List returns the managed accounts. claude-swap may refresh usage from the network.
func (s Switcher) List(ctx context.Context) ([]Account, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, s.Bin, "list", "--json").Output()
	if ctx.Err() != nil {
		return nil, errors.New("claude-swap list timed out")
	}
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("claude-swap list failed: %w", err)
	}
	return Parse(out)
}

// Parse reads `claude-swap list --json`.
func Parse(b []byte) ([]Account, error) {
	var v struct {
		SchemaVersion int       `json:"schemaVersion"`
		Accounts      []Account `json:"accounts"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("unexpected claude-swap output (needs a version with --json): %w", err)
	}
	if v.SchemaVersion != 1 {
		return v.Accounts, fmt.Errorf("claude-swap schemaVersion %d is newer than catude understands (1)", v.SchemaVersion)
	}
	return v.Accounts, nil
}

// SwitchCmd returns the command that switches to account n (run it attached to the terminal).
func (s Switcher) SwitchCmd(n int) *exec.Cmd {
	return exec.Command(s.Bin, "switch", strconv.Itoa(n))
}

// InstallCmd installs or upgrades claude-swap to the pinned version with uv.
func InstallCmd() (*exec.Cmd, error) {
	uv, err := exec.LookPath("uv")
	if err != nil {
		return nil, errors.New("uv is not installed (https://docs.astral.sh/uv/): needed to install claude-swap")
	}
	return exec.Command(uv, "tool", "install", "--force", "claude-swap=="+PinnedVersion), nil
}

// Warnings explains known problems of the installed version.
func (s Switcher) Warnings() []string {
	var w []string
	if s.Version != "" && Less(s.Version, "0.30.0") {
		w = append(w, "claude-swap < 0.30: `cswap run` with CLAUDE_CONFIG_DIR set can log out every open Claude session (issue #516); `switch` is not affected")
	}
	w = append(w, "claude-swap stores every account's tokens base64-encoded, not encrypted, on Windows and Linux")
	return w
}

// Less compares dotted versions numerically.
func Less(a, b string) bool {
	pa, pb := reVersion.FindStringSubmatch(a), reVersion.FindStringSubmatch(b)
	if pa == nil || pb == nil {
		return false
	}
	for i := 1; i <= 3; i++ {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			return x < y
		}
	}
	return false
}

// MaskEmail keeps the first letter and the domain: a••••@example.com (for screenshots and exports).
func MaskEmail(e string) string {
	i := strings.Index(e, "@")
	if i <= 1 {
		return e
	}
	return e[:1] + "••••" + e[i:]
}
