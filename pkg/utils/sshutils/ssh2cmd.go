package sshutils

import (
	"os/exec"
)

// SSHToCmd if caller don't provide enough config for run ssh cmd，change to run cmd by os.exec on localhost.
// for aio deploy now.
func SSHToCmd(sshConfig *SSH, host string) bool {

	ret := host == "" ||
		sshConfig == nil ||
		sshConfig != nil && sshConfig.User == "" || (sshConfig.Password == "" && sshConfig.PkFile == "" && sshConfig.PrivateKey == "")
	return ret
}

func ExtraExitCode(err error) (bool, int) {
	if exiterr, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0
		return true, exiterr.ExitCode()
	}
	return false, -1
}
