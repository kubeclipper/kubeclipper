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
)

func GetDefaultIP(ipv4 bool) (net.IP, error) {
	family := netlink.FAMILY_V4
	if !ipv4 {
		family = netlink.FAMILY_V6
	}
	rl, err := netlink.RouteList(nil, family)
	if err != nil {
		return nil, err
	}
	for _, r := range rl {
		if r.Gw == nil {
			continue
		}
		link, err := netlink.LinkByIndex(r.LinkIndex)
		if err != nil {
			return nil, err
		}
		addrs, err := netlink.AddrList(link, family)
		if err != nil {
			return nil, err
		}
		return addrs[0].IP, nil
	}
	return nil, errors.New("no default IP address")
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
		if r.Gw == nil {
			continue
		}
		return r.Gw, nil
	}
	return nil, errors.New("no default gateway")
}
