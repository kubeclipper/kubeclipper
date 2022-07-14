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
	"fmt"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type ByteSize float64

const (
	MinByteSize          = iota
	KB          ByteSize = 1 << (10 * iota)
	MB
	GB
	TB
	MaxByteSize
)

const (
	Loopback = "lo"
)

type SysInfo struct {
	CPU    CPU
	Memory Memory
	Disk   Disk
	Net    []Net
	Host   Host
}

type CPU struct {
	Family    string  `json:"family"` // ex: Intel(R) Core(TM)
	Cores     int32   `json:"cores"`
	ModelName string  `json:"modelName"` // ex: i7-8750H
	Mhz       float64 `json:"mhz"`       // ex: 2.20GHz
	CacheSize int32   `json:"cacheSize"`
}

type Memory struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
	Free        uint64  `json:"free"`
}

type Net struct {
	Index        int             `json:"index"`
	Name         string          `json:"name"`         // e.g., "en0", "lo0", "eth0.100"
	MTU          int             `json:"mtu"`          // maximum transmission unit
	HardwareAddr string          `json:"hardwareAddr"` // IEEE MAC-48, EUI-48 and EUI-64 form
	Flags        string          `json:"flags"`        // e.g., FlagUp, FlagLoopback, FlagMulticast
	Addrs        []InterfaceAddr `json:"addrs"`
}

// NetInterfaceAddr is designed for represent interface addresses
type InterfaceAddr struct {
	Family string `json:"family"` //ex: ipv4/ipv6
	Addr   string `json:"addr"`
}

type Disk struct {
	DiskDevices []DiskDevice
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
}

type DiskDevice struct {
	Device      string  `json:"device"`     // ex: /dev/sda
	Mountpoint  string  `json:"mountpoint"` // ex: /boot
	Fstype      string  `json:"fstype"`     // ex: xfs
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
}

type Host struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`              // ex: freebsd, linux
	Platform        string `json:"platform"`        // ex: ubuntu, linuxmint
	PlatformFamily  string `json:"platformFamily"`  // ex: debian, rhel
	PlatformVersion string `json:"platformVersion"` // version of the complete OS
	KernelVersion   string `json:"kernelVersion"`   // version of the OS kernel (if available)
	KernelArch      string `json:"kernelArch"`      // native cpu architecture queried at runtime, as returned by `uname -m` or empty string in case of error
	HostID          string `json:"hostId"`          // ex: uuid
}

func GetSysInfo() (*SysInfo, error) {
	cpuInfo, err := CUPInfo()
	if err != nil {
		return nil, err
	}
	memoryInfo, err := MemoryInfo(MB)
	if err != nil {
		return nil, err
	}
	diskInfo, err := DiskInfo(GB)
	if err != nil {
		return nil, err
	}
	netInfo, err := NetInfo()
	if err != nil {
		return nil, err
	}
	hostInfo, err := HostInfo()
	if err != nil {
		return nil, err
	}

	return &SysInfo{
		CPU:    cpuInfo,
		Memory: memoryInfo,
		Disk:   diskInfo,
		Net:    netInfo,
		Host:   hostInfo,
	}, nil
}

func MemoryInfo(byteSize ByteSize) (Memory, error) {
	if byteSize <= MinByteSize {
		byteSize = KB
	} else if byteSize >= MaxByteSize {
		byteSize = TB
	}

	info, err := mem.VirtualMemory()
	if err != nil {
		return Memory{}, err
	}
	return Memory{
		Total:       info.Total / uint64(byteSize),
		Available:   info.Available / uint64(byteSize),
		Used:        info.Used / uint64(byteSize),
		UsedPercent: info.UsedPercent,
		Free:        info.Free / uint64(byteSize),
	}, err
}

func DiskInfo(byteSize ByteSize) (Disk, error) {
	if byteSize <= MinByteSize {
		byteSize = KB
	} else if byteSize >= MaxByteSize {
		byteSize = TB
	}
	d := Disk{}
	parts, err := disk.Partitions(false)
	if err != nil {
		return d, err
	}
	for _, part := range parts {
		usage, err := disk.Usage(part.Mountpoint)
		if err != nil {
			return d, err
		}

		d.DiskDevices = append(d.DiskDevices,
			DiskDevice{
				Device:      part.Device,
				Mountpoint:  part.Mountpoint,
				Fstype:      part.Fstype,
				Total:       usage.Total,
				Free:        usage.Free,
				Used:        usage.Used,
				UsedPercent: usage.UsedPercent,
			})
		d.Total = usage.Total / uint64(byteSize)
		d.Free = usage.Free / uint64(byteSize)
		d.Used = usage.Used / uint64(byteSize)
		d.UsedPercent, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", usage.UsedPercent), 64)
	}
	return d, nil
}

func NetInfo() ([]Net, error) {
	return netInfo()
}

func HostInfo() (Host, error) {
	info, err := host.Info()
	if err != nil {
		return Host{}, err
	}
	return Host{
		Hostname:        info.Hostname,
		OS:              info.OS,
		Platform:        info.Platform,
		PlatformFamily:  info.PlatformFamily,
		PlatformVersion: strings.Split(info.PlatformVersion, ".")[0],
		KernelVersion:   info.KernelVersion,
		KernelArch:      info.KernelArch,
		HostID:          info.HostID,
	}, nil
}
