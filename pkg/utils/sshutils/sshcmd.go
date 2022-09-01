package sshutils

import (
	"fmt"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"
)

func IsFloatIP(sshConfig *SSH, ip string) (bool, error) {
	// 1.extra all ip
	result, err := SSHCmd(sshConfig, ip, `ifconfig|grep inet|awk {'print $2'}`)
	if err != nil {
		return false, err
	}
	if result.ExitCode != 0 {
		return false, fmt.Errorf("%s stderr:%s", result.Short(), result.Stderr)
	}
	ips := strings.Split(result.Stdout, "\n")
	ips = sliceutil.RemoveString(ips, func(item string) bool {
		return item == ""
	})
	// 2.check
	return !sliceutil.HasString(ips, ip), nil
}

func GetRemoteHostName(sshConfig *SSH, hostIP string) (string, error) {
	result, err := SSHCmd(sshConfig, hostIP, "hostname")
	if err != nil {
		return "", err
	}
	return strings.ToLower(result.StdoutToString("")), nil
}
