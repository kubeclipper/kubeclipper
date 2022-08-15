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
	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
)

const (
	configLongDescription = `Print default deploy config.`
	configExample         = `
  #save default config to deploy-config.yaml
  kcctl deploy config > deploy-config.yaml
`
)

func NewCmdDeployConfig(o *DeployOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "config",
		DisableFlagsInUseLine: true,
		Short:                 "Print default deploy config.",
		Long:                  configLongDescription,
		Example:               configExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.GenDefaultConfig())
		},
		Args: cobra.NoArgs,
	}
	return cmd
}

func (d *DeployOptions) GenDefaultConfig() error {
	_, err := d.IOStreams.Out.Write([]byte(configTemplate))
	return err
}

const configTemplate = `
# This is the kubeclipper deploy configuration file.
# Commented options is default value or example,uncommented options override the default value.

# ssh config, need one of passwd or private key.
ssh:
  #user: root
  password: ""
  pkFile: ""
  pkPassword: ""

# etcd config
etcd:
  #clientPort: 2379
  #peerPort: 2380
  #metricsPort: 2381
  #dataDir: /var/lib/kc-etcd

# kubeclipper service node's ip list,must be odd.
# NOTE: must specify one server node at least.
serverIPs:
#- 192.168.10.10
#- 192.168.10.11
#- 192.168.10.12

# kubeclipper agent node's ip list.
# if you don't specify agent node,can use 'kcctl join' command to join after deploy.
# you can specify metadata for per node.
agents:
  # with full metadata
  #192.168.10.10:
     #region: us-west
     #fip: 172.20.150.199
  # with region metadata
  #192.168.10.11:
     #region: us-west
  # without metadata
  #192.168.10.12:

# run kubeclipper in debug mode.
#debug: false

# kubeclipper agent node's default region.
#defaultRegion: default

# kubeclipper backend server port
#serverPort: 8080
# kubeclipper frontend server port.
#consolePort: 80
# kubeclipper static server port.
#staticServerPort: 8081
# static package directory.
#staticServerPath: /opt/kubeclipper-server/resource
#automatic generate jwt token.
#jwtSecret: ""

# deploy resource package,support url or file absolute path.
#pkg: https://oss.kubeclipper.io/release/v1.1.0/kc-amd64.tar.gz
pkg: /tmp/kc-minimal.tar.gz

# mq config,support internal or external mq.
# use internal mq,kubeclipper will running mq with service,and automatic generate ips、user、secret and certs(if enable tls).
# use external mq,you need specify ips、user、secret and certs(if enable tls).
mq:
  #external: false
  #tls: true
  ca: ""
  clientCert: ""
  clientKey: ""
  ips: []
  #port: 9889
  #clusterPort: 9890
  #user: admin
  secret: ""

# operation log config.
opLog:
  #dir: /var/log/kc-agent
  # max operation log query threshold,default 1MB.
  #threshold: 1048576
`
