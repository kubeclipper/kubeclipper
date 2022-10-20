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

package deploy

import (
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"

	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
)

func BuildAgentRegion(agentRegions []string, defaultRegion string) (options.AgentRegions, error) {
	m := make(options.AgentRegions)
	for _, agentRegion := range agentRegions {
		var region, agentStr string
		// region:agent
		// agent like: agent1,agent2,
		// 	1. parse region and agentStr from input
		region, agentStr, err := parseAgentRegion(agentRegion, defaultRegion)
		if err != nil {
			return nil, err
		}

		// 	2.1 parse agents,spilt by ,
		agents := strings.Split(agentStr, ",")
		for _, v := range agents {
			// 2.2 parse network segment,spilt by -
			ips, err := parseAgent(v)
			if err != nil {
				return nil, err
			}
			m[region] = append(m[region], ips...)
		}
	}
	return m, nil
}

// BuildFIP parse ip and fip.
func BuildFIP(fips []string) (options.FIPs, error) {
	m := make(options.FIPs)

	// ip:fip
	for _, str := range fips {
		// spilt by :
		split := strings.Split(str, ":")
		if len(split) != 2 {
			return nil, fmt.Errorf("invalid fip %s", str)
		}
		ipStr := split[0]
		fip := split[1]
		if ip := net.ParseIP(ipStr); ip == nil {
			return nil, fmt.Errorf("invalid ip %s", ipStr)
		}
		if ip := net.ParseIP(fip); ip == nil {
			return nil, fmt.Errorf("invalid fip %s", fip)
		}
		if oldFip, ok := m[ipStr]; ok {
			logger.Warnf("config %s repeat, %s already bind to %s", str, oldFip, ipStr)
		}
		m[ipStr] = fip
	}
	return m, nil
}

func BuildAgent(agentRegions, fips []string, defaultRegion string) (options.Agents, error) {
	ipRegions, err := BuildAgentRegion(agentRegions, defaultRegion)
	if err != nil {
		return nil, errors.WithMessage(err, "parse region")
	}
	ipFIPs, err := BuildFIP(fips)
	if err != nil {
		return nil, errors.WithMessage(err, "parse fip")
	}

	var m = make(options.Agents)
	for region, ips := range ipRegions {
		for _, ip := range ips {
			agent := options.Metadata{Region: region}
			fip, ok := ipFIPs[ip]
			if ok {
				agent.FloatIP = fip
			}
			m[ip] = agent
		}
	}
	return m, nil
}

func parseAgent(agent string) ([]string, error) {
	set := sets.NewString()
	split := strings.Split(agent, "-")
	if len(split) == 1 { // basic ip
		if !netutil.IsValidIP(split[0]) {
			return nil, errors.New("invalid agent")
		}
		set.Insert(split[0])
		return set.List(), nil
	}

	if len(split) == 2 { // network segment
		if !netutil.IsValidIP(split[0]) || !netutil.IsValidIP(split[1]) {
			return nil, errors.New("invalid agent")
		}
		start := netutil.InetAtoN(split[0])
		end := netutil.InetAtoN(split[1])
		for ; start <= end; start++ {
			set.Insert(netutil.InetNtoA(start))
		}
		return set.List(), nil
	}

	return nil, errors.New("invalid agent")
}

func parseAgentRegion(agentRegion, defaultRegion string) (region, agentStr string, err error) {
	split := strings.Split(agentRegion, ":")
	if len(split) == 1 { // not specify region
		region = defaultRegion
		agentStr = split[0]
	} else if len(split) == 2 { // specified region
		region = split[0]
		agentStr = split[1]
	} else { // invalid
		return "", "", fmt.Errorf("invalid agent %s", agentRegion)
	}
	return region, agentStr, nil
}
