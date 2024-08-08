//go:build !linux
// +build !linux

package cni

const (
	defaultVTEPDeviceName = "vxlan.calico"
	ipipKernelModuleName  = "ipip"
)

// clearCalicoNICs cleans up all the NICs created by calico
func clearCalicoNICs(mode string) {
	return
}
