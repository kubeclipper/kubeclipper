//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package cmdutil

import "os/exec"

func configureProcessGroup(_ *exec.Cmd, _ <-chan struct{}) {}
