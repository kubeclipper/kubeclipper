package proxy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/proxy/config"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

func parseTunnel(tunnelsStr []string) ([]tunnel, error) {
	list := make([]tunnel, 0, len(tunnelsStr))
	for _, str := range tunnelsStr {
		split := strings.Split(str, ":")
		if len(split) != 3 {
			return nil, fmt.Errorf("invalid tunnel %s", str)
		}
		localPort, err := strconv.Atoi(split[0])
		if err != nil {
			return nil, err
		}
		t := tunnel{
			AgentIP:    split[1],
			LocalPort:  localPort,
			RemotePort: split[2],
		}
		list = append(list, t)
	}
	return list, nil
}

type tunnel struct {
	AgentIP    string
	LocalPort  int
	RemotePort string
}

func (d *ProxyOptions) generateProxyConfig() *config.Config {
	c := config.New()
	c.ServerIP = d.deployConfig.ServerIPs[0]
	c.DefaultMQPort = d.deployConfig.MQ.Port
	c.DefaultStaticServerPort = d.deployConfig.StaticServerPort

	m := make(map[string][]tunnel)
	for _, t := range d.tunnels {
		m[t.AgentIP] = append(m[t.AgentIP], t)
	}

	for agentIP, tunnels := range m {
		hostname, _ := sshutils.GetRemoteHostName(d.deployConfig.SSHConfig, agentIP)
		agentConfig := config.AgentConfig{
			Name:    hostname,
			Tunnels: make([]config.Tunnel, 0, len(tunnels)),
		}
		for _, v := range tunnels {
			t := config.Tunnel{
				LocalPort:     v.LocalPort,
				RemoteAddress: fmt.Sprintf("%s:%s", v.AgentIP, v.RemotePort),
			}
			agentConfig.Tunnels = append(agentConfig.Tunnels, t)
		}
		c.Config = append(c.Config, agentConfig)
	}
	return c
}
