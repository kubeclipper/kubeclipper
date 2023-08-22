package common

import (
	"strconv"
	"strings"
)

func IsKubeVersionGreater(kubeVersion string, baseVersion int) bool {
	if kubeVersion == "" {
		return false
	}
	kubeVersion = strings.ReplaceAll(kubeVersion, "v", "")
	kubeVersion = strings.ReplaceAll(kubeVersion, ".", "")

	kubeVersion = strings.Join(strings.Split(kubeVersion, "")[0:3], "")

	if v, _ := strconv.Atoi(kubeVersion); v >= baseVersion {
		return true
	}
	return false
}
