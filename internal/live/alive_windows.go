//go:build windows

package live

import "golang.org/x/sys/windows"

const stillActive = 259 // STILL_ACTIVE

// processAlive asks Windows whether the process exists and has not exited (no process listing: fast).
func processAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if windows.GetExitCodeProcess(h, &code) != nil {
		return false
	}
	return code == stillActive
}
