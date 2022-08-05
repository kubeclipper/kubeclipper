//go:build linux
// +build linux

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

package netutil

import (
	"errors"
	"net"

	"github.com/vishvananda/netlink"

	"github.com/kubeclipper/kubeclipper/pkg/utils/autodetection"
)

func GetDefaultIP(ipv4 bool, method string) (net.IP, error) {
	version := autodetection.IPv4
	if !ipv4 {
		version = autodetection.IPv4
	}
	ipNet, err := autodetection.AutoDetectCIDR(method, version)
	if err != nil {
		return nil, err
	}
	return ipNet.IP, nil
}

func GetDefaultGateway(ipv4 bool) (ip net.IP, err error) {
	family := netlink.FAMILY_V4
	if !ipv4 {
		family = netlink.FAMILY_V6
	}
	rl, err := netlink.RouteList(nil, family)
	if err != nil {
		return nil, err
	}
	for _, r := range rl {
		if r.Gw != nil && r.Dst == nil {
			return r.Gw, nil
		}
	}
	return nil, errors.New("no default gateway")
}
