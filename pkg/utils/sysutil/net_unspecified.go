//go:build !linux
// +build !linux

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

	"github.com/shirou/gopsutil/v3/net"
)

func netInfo() ([]Net, error) {
	inters, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	nets := make([]Net, 0)

	for _, v := range inters {
		if v.Name == Loopback {
			continue
		}
		nets = append(nets, Net{
			Index:        v.Index,
			Name:         v.Name,
			MTU:          v.MTU,
			HardwareAddr: v.HardwareAddr,
			Addrs:        AddrsConvert(v.Addrs),
		})
	}
	return nets, err
}

func AddrsConvert(list net.InterfaceAddrList) []InterfaceAddr {
	inters := make([]InterfaceAddr, len(list))
	for i, item := range list {
		inters[i].Addr = item.Addr
		if strings.Count(item.Addr, ":") < 2 {
			inters[i].Family = "ipv4"
		} else {
			inters[i].Family = "ipv6"
		}
	}
	return inters
}

func CUPInfo() (CPU, error) {
	return CPU{}, nil
}
