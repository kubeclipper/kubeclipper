/*
 *
 *  * Copyright 2021 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package autodetection

import (
	"net"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

// Interface contains details about an interface on the host.
type Interface struct {
	Name  string
	Cidrs []net.IPNet
}

// GetInterfaces returns a list of all interfaces, skipping any interfaces whose
// name matches any of the exclusion list regexes, and including those on the
// inclusion list.
func GetInterfaces(includeRegexes []string, excludeRegexes []string, version int) ([]Interface, error) {
	netIfaces, err := net.Interfaces()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to enumerate interfaces")
	}

	var (
		filteredIfaces []Interface
		includeRegexp  *regexp.Regexp
		excludeRegexp  *regexp.Regexp
	)

	// Create single include and exclude regexes to perform the interface check.
	if len(includeRegexes) > 0 {
		if includeRegexp, err = regexp.Compile("(" + strings.Join(includeRegexes, ")|(") + ")"); err != nil {
			// log.WithError(err).Warnf("Invalid interface regex")
			return nil, err
		}
	}
	if len(excludeRegexes) > 0 {
		if excludeRegexp, err = regexp.Compile("(" + strings.Join(excludeRegexes, ")|(") + ")"); err != nil {
			// log.WithError(err).Warnf("Invalid interface regex")
			return nil, err
		}
	}

	// Loop through interfaces filtering on the regexes.  Loop in reverse
	// order to maintain behavior with older versions.
	for idx := len(netIfaces) - 1; idx >= 0; idx-- {
		iface := netIfaces[idx]
		include := (includeRegexp == nil) || includeRegexp.MatchString(iface.Name)
		exclude := (excludeRegexp != nil) && excludeRegexp.MatchString(iface.Name)
		if include && !exclude {
			if i, err := convertInterface(&iface, version); err == nil {
				filteredIfaces = append(filteredIfaces, *i)
			}
		}
	}
	return filteredIfaces, nil
}

// convertInterface converts a net.Interface to our Interface type (which has converted address types).
func convertInterface(i *net.Interface, version int) (*Interface, error) {
	addrs, err := i.Addrs()
	if err != nil {
		return nil, errors.WithMessagef(err, "Cannot get interface(%s) address", i.Name)
	}

	iface := &Interface{Name: i.Name}
	for _, addr := range addrs {
		addrStr := addr.String()
		ip, ipNet, err := parseCIDR(addrStr)
		if err != nil {
			logger.Warnf("Failed to parse CIDR(%s)", addrStr)
			continue
		}

		if ipVersion(*ip) == version {
			// Switch out the IP address in the network with the
			// interface IP to get the CIDR (IP + network).
			ipNet.IP = *ip
			logger.Debugf("Found valid IP address and network %v", ipNet)
			iface.Cidrs = append(iface.Cidrs, *ipNet)
		}
	}

	return iface, nil
}

func ipVersion(ip net.IP) int {
	if ip.To4() != nil {
		return IPv4
	} else if len(ip) == net.IPv6len {
		return IPv6
	}
	return 0
}

func parseCIDR(c string) (*net.IP, *net.IPNet, error) {
	netIP, netIPNet, e := net.ParseCIDR(c)
	if netIPNet == nil || e != nil {
		return nil, nil, e
	}

	// The base golang net library always uses a 4-byte IPv4 address in an
	// IPv4 IPNet, so for uniformity in the returned types, make sure the
	// IP address is also 4-bytes - this allows the user to safely assume
	// all IP addresses returned by this function use the same encoding
	// mechanism (not strictly required but better for testing and debugging).
	if ip4 := netIP.To4(); ip4 != nil {
		return &ip4, netIPNet, nil
	}

	return &netIP, netIPNet, nil
}
