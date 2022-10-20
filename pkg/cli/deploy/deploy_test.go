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
	"testing"

	"github.com/google/uuid"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
)

func TestDeployOptions_getEtcdTemplateContent(t *testing.T) {
	d := NewDeployOptions(options.IOStreams{})
	d.deployConfig.ServerIPs = []string{"192.168.234.3", "192.168.234.4", "192.168.234.5"}
	d.servers = map[string]string{
		"192.168.234.3": "master1",
		"192.168.234.4": "master2",
		"192.168.234.5": "master3",
	}

	for _, s := range d.deployConfig.ServerIPs {
		t.Log(d.getEtcdTemplateContent(s))
	}
}

func TestDeployOptions_getKcServerConfigTemplateContent(t *testing.T) {
	d := NewDeployOptions(options.IOStreams{})
	d.deployConfig.ServerIPs = []string{"192.168.234.3", "192.168.234.4", "192.168.234.5"}
	d.servers = map[string]string{
		"192.168.234.3": "master1",
		"192.168.234.4": "master2",
		"192.168.234.5": "master3",
	}

	for _, s := range d.deployConfig.ServerIPs {
		t.Log(d.deployConfig.GetKcServerConfigTemplateContent(s))
	}
}

func TestDeployOptions_getKcAgentConfigTemplateContent(t *testing.T) {
	d := NewDeployOptions(options.IOStreams{})
	d.deployConfig.ServerIPs = []string{"192.168.234.3", "192.168.234.4", "192.168.234.5"}
	d.servers = map[string]string{
		"192.168.234.3": "master1",
		"192.168.234.4": "master2",
		"192.168.234.5": "master3",
	}
	metadata := options.Metadata{
		Region:  d.deployConfig.DefaultRegion,
		FloatIP: "1.1.1.1",
	}
	for range d.deployConfig.ServerIPs {
		metadata.AgentID = uuid.New().String()
		t.Log(d.deployConfig.GetKcAgentConfigTemplateContent(metadata))
	}
}

func TestDeployOptions_getKcConsoleTemplateContent(t *testing.T) {
	d := NewDeployOptions(options.IOStreams{})
	d.deployConfig.ServerIPs = []string{"192.168.234.3", "192.168.234.4", "192.168.234.5"}
	d.servers = map[string]string{
		"192.168.234.3": "master1",
		"192.168.234.4": "master2",
		"192.168.234.5": "master3",
	}

	t.Log(d.getKcConsoleTemplateContent())
}
