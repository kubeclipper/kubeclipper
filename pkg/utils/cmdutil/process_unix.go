//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package cmdutil

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

const commandTerminationGrace = 30 * time.Second

func configureProcessGroup(command *exec.Cmd, done <-chan struct{}, grace time.Duration) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		pid := command.Process.Pid
		err := syscall.Kill(-pid, syscall.SIGTERM)
		if err != nil && !errors.Is(err, syscall.ESRCH) {
			return err
		}
		go func() {
			timer := time.NewTimer(grace)
			defer timer.Stop()
			select {
			case <-done:
			case <-timer.C:
				if killErr := syscall.Kill(-pid, syscall.SIGKILL); killErr != nil && !errors.Is(killErr, syscall.ESRCH) {
					return
				}
			}
		}()
		return nil
	}
}

// killProcessGroup also reaps descendants that kept pipes open after the leader exited.
func killProcessGroup(command *exec.Cmd) {
	if command.Process != nil {
		if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			logger.Warnf("terminate diagnostic process group: %v", err)
		}
	}
}
