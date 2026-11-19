//go:build !windows

package live

import (
	"errors"
	"syscall"
)

// processAlive sends signal 0: it checks that the process exists without touching it.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
