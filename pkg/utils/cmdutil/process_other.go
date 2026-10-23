//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package cmdutil

import (
	"errors"
	"os"
	"os/exec"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

func configureProcessGroup(_ *exec.Cmd, _ <-chan struct{}, _ time.Duration) {}

func killProcessGroup(command *exec.Cmd) {
	if command.Process != nil {
		if err := command.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			logger.Warnf("terminate diagnostic process: %v", err)
		}
	}
}
