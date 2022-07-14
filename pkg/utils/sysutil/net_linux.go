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

package sysutil

import (
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/vishvananda/netlink"
)

const (
	deviceType = "device"
)

func netInfo() ([]Net, error) {
	list, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	nets := make([]Net, 0)
	for _, v := range list {
		if v.Type() == deviceType && v.Attrs().Name != Loopback {
			iaddrs := make([]InterfaceAddr, 0)
			addrs, err := netlink.AddrList(v, 0)
			if err != nil {
				return nil, err
			}
			family := "ipv6"
			for _, addr := range addrs {
				if strings.Count(addr.IP.String(), ":") < 2 {
					family = "ipv4"
				}

				iaddrs = append(iaddrs, InterfaceAddr{family, addr.IP.String()})
			}
			nets = append(nets, Net{
				Index:        v.Attrs().Index,
				Name:         v.Attrs().Name,
				MTU:          v.Attrs().MTU,
				HardwareAddr: v.Attrs().HardwareAddr.String(),
				Addrs:        iaddrs,
			})
		}
	}
	return nets, nil
}

func CUPInfo() (CPU, error) {
	c := CPU{}
	infos, err := cpu.Info()
	if err != nil {
		return c, err
	}

	c.CacheSize = infos[0].CacheSize
	c.Family = infos[0].Family
	c.Mhz = infos[0].Mhz
	c.ModelName = infos[0].ModelName

	for _, val := range infos {
		c.Cores = c.Cores + val.Cores
	}
	return c, nil
}
