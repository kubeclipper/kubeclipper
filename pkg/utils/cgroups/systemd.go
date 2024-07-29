package cgroups

import (
	"os"
	"sync"
)

var (
	isRunningSystemdOnce sync.Once
	isRunningSystemd     bool
)

// IsRunningSystemd checks whether the host was booted with systemd as its init
// system. This functions similarly to systemd's `sd_booted(3)`: internally, it
// checks whether /run/systemd/system/ exists and is a directory.
// http://www.freedesktop.org/software/systemd/man/sd_booted.html
func IsRunningSystemd() bool {
	isRunningSystemdOnce.Do(func() {
		fi, err := os.Lstat("/run/systemd/system")
		isRunningSystemd = err == nil && fi.IsDir()
	})
	return isRunningSystemd
}
