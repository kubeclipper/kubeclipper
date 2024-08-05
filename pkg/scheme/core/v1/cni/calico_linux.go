//go:build linux
// +build linux

package cni

import (
	"strings"

	"golang.org/x/sys/unix"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
	"github.com/vishvananda/netlink"
)

const (
	defaultVTEPDeviceName = "vxlan.calico"
	ipipKernelModuleName  = "ipip"
)

// clearCalicoNICs cleans up all the NICs created by calico
func clearCalicoNICs(mode string) {
	switch mode {
	case CalicoNetworkIPIPAll, CalicoNetworkIPIPSubnet:
		logger.Info("disable IPIP kernel module")
		if err := unix.DeleteModule(ipipKernelModuleName, 0); err != nil {
			logger.Errorf("failed to disable IPIP kernel module: %v", err)
		}
	case CalicoNetworkVXLANAll, CalicoNetworkVXLANSubnet:
		logger.Infof("remove VTEP device: %s", defaultVTEPDeviceName)
		if err := netutil.DeleteLink(defaultVTEPDeviceName); err != nil {
			logger.Errorf("failed to remove VTEP device: %v", err)
		}
	}

	links, err := netlink.LinkList()
	if err != nil {
		logger.Errorf("failed to list links: %v", err)
		return
	}
	// remove veth pairs created by calico
	for _, link := range links {
		name := link.Attrs().Name
		if !(strings.Contains(name, "cali") && netutil.IsLinkVeth(link)) {
			continue
		}
		if err := netutil.DeleteLink(name); err != nil {
			logger.Errorf("failed to remove link %s: %v", name, err)
		}
	}
}
