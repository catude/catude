//go:build windows

package claudecfg

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Claude Desktop policies on Windows live in the registry (support article 12622667):
// HKLM or HKCU \SOFTWARE\Policies\Claude, DWORD for booleans and numbers.
const policyKey = `SOFTWARE\Policies\Claude`

func desktopPolicyPath() string  { return `HKLM\` + policyKey }
func desktopPolicyWhere() string { return `registry HKLM/HKCU\` + policyKey }

// readDesktopPolicy returns the policy values set on this machine (machine-wide wins over the user's).
func readDesktopPolicy() map[string]string {
	out := map[string]string{}
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		k, err := registry.OpenKey(root, policyKey, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		names, _ := k.ReadValueNames(0)
		for i, n := range names {
			if i >= 500 {
				break
			}
			if s, _, err := k.GetStringValue(n); err == nil {
				out[n] = s
			} else if v, _, err := k.GetIntegerValue(n); err == nil {
				out[n] = fmt.Sprint(v)
			} else if ss, _, err := k.GetStringsValue(n); err == nil {
				out[n] = strings.Join(ss, ", ")
			}
		}
		k.Close()
	}
	return out
}
