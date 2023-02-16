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
// the code is mainly from:
// 	   https://github.com/projectcalico/calico
// thanks to the related developer

package autodetection

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

const (
	MethodFirst     = "first-found"
	MethodInterface = "interface="
	MethodCidr      = "cidr="
	MethodCanReach  = "can-reach="
)

const (
	IPv4 = 4
	IPv6 = 6
)

func CheckMethod(method string) bool {
	if method == "" || method == MethodFirst {
		return true
	}

	return strings.HasPrefix(method, MethodInterface) || strings.HasPrefix(method, MethodCidr)
}

func CheckCalicoMethod(method string) bool {
	if method == "" || method == MethodFirst {
		return true
	}

	return strings.HasPrefix(method, MethodInterface) || strings.HasPrefix(method, MethodCanReach)
}

// AutoDetectCIDR auto-detects the IP and Network using the requested detection method.
func AutoDetectCIDR(method string, version int) (*net.IPNet, error) {
	if method == "" || method == MethodFirst {
		// Autodetect the IP by enumerating all interfaces (excluding
		// known internal interfaces).
		return autoDetectCIDRFirstFound(version)
	} else if strings.HasPrefix(method, MethodInterface) {
		// Autodetect the IP from the specified interface.
		ifStr := strings.TrimPrefix(method, MethodInterface)
		// Regexes are passed in as a string separated by ","
		ifRegexes := regexp.MustCompile(`\s*,\s*`).Split(ifStr, -1)
		return autoDetectCIDRByInterface(ifRegexes, version)
	} else if strings.HasPrefix(method, MethodCidr) {
		// Autodetect the IP by filtering interface by its address.
		cidrStr := strings.TrimPrefix(method, MethodCidr)
		// CIDRs are passed in as a string separated by ","
		var matches []net.IPNet
		for _, r := range regexp.MustCompile(`\s*,\s*`).Split(cidrStr, -1) {
			_, cidr, err := parseCIDR(r)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q for IP autodetection method: %s", r, method)
			}
			matches = append(matches, *cidr)
		}
		return autoDetectCIDRByCIDR(matches, version)
	}
	return nil, fmt.Errorf("invalid IP autodetection method: %s", method)
}

// autoDetectCIDRFirstFound auto-detects the first valid Network it finds across
// all interfaces (excluding common known internal interface names).
func autoDetectCIDRFirstFound(version int) (*net.IPNet, error) {
	iface, cidr, err := FilteredEnumeration(nil, DefaultInterfacesToExclude, nil, version)
	if err != nil {
		return nil, fmt.Errorf("unable to auto-detect an IPv%d address: %s", version, err)
	}

	logger.Infof("Using auto detected IPv%d address on interface %s: %s", version, iface.Name, cidr.String())

	return cidr, nil
}

// autoDetectCIDRByInterface auto-detects the first valid Network on the interfaces
// matching the supplied interface regex.
func autoDetectCIDRByInterface(ifaceRegexes []string, version int) (*net.IPNet, error) {
	iface, cidr, err := FilteredEnumeration(ifaceRegexes, nil, nil, version)
	if err != nil {
		return nil, fmt.Errorf("unable to auto-detect an IPv%d address using interface regexes %v: %s", version, ifaceRegexes, err)
	}

	logger.Infof("Using autodetected IPv%d address %s on matching interface %s", version, cidr.String(), iface.Name)

	return cidr, nil
}

// autoDetectCIDRByCIDR auto-detects the first valid Network on the interfaces
// matching the supplied cidr.
func autoDetectCIDRByCIDR(matches []net.IPNet, version int) (*net.IPNet, error) {
	iface, cidr, err := FilteredEnumeration(nil, nil, matches, version)
	if err != nil {
		return nil, fmt.Errorf("unable to auto-detect an IPv%d address using interface cidr %s: %s", version, matches, err)
	}

	logger.Infof("Using autodetected IPv%d address %s on interface %s matching cidrs %+v", version, cidr.String(), iface.Name, matches)
	return cidr, nil
}
