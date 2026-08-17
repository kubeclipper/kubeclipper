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

	"sigs.k8s.io/yaml"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
)

func TestDeployOptions_GenDefaultConfig(t *testing.T) {
	omitempty, err := options.Omitempty([]byte(configTemplate))
	if err != nil {
		t.Fatal(err)
	}
	d := options.NewDeployOptions()
	if d.TempDir != "/tmp" {
		t.Fatalf("default tempDir = %q, want /tmp", d.TempDir)
	}
	err = yaml.Unmarshal(omitempty, d)
	if err != nil {
		return
	}
	marshal, err := yaml.Marshal(d)
	if err != nil {
		return
	}
	t.Log(string(marshal))
}

func TestDeployOptionsConfigTempDir(t *testing.T) {
	d := options.NewDeployOptions()
	if err := yaml.Unmarshal([]byte("tempDir: /var/lib/kubeclipper/tmp\n"), d); err != nil {
		t.Fatalf("unmarshal deploy config: %v", err)
	}
	if d.TempDir != "/var/lib/kubeclipper/tmp" {
		t.Fatalf("tempDir = %q", d.TempDir)
	}
}
