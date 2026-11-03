//go:build !windows

package update

import (
	"os"
	"syscall"
)

// Restart replaces this process with the updated binary.
func Restart(exe string) error {
	return syscall.Exec(exe, os.Args, os.Environ())
}
