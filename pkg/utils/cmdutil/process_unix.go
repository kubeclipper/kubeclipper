//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package cmdutil

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const commandTerminationGrace = 30 * time.Second

func configureProcessGroup(command *exec.Cmd, done <-chan struct{}) {
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
			timer := time.NewTimer(commandTerminationGrace)
			defer timer.Stop()
			select {
			case <-done:
			case <-timer.C:
				_ = syscall.Kill(-pid, syscall.SIGKILL)
			}
		}()
		return nil
	}
}
