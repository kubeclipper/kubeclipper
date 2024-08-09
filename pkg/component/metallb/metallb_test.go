/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package metallb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var lb = &MetalLB{
	ImageRepoMirror: "192.168.10.10:5000",
	ManifestsDir:    "/tmp/.metallb",
	Mode:            "BGP",
	Addresses:       []string{"192.168.20.20-192.168.20.30"},
	Version:         "v0.13.7",
}

func TestRenderTo(t *testing.T) {
	sb := &strings.Builder{}
	if err := lb.renderTo(sb); err != nil {
		assert.FailNow(t, "deploy template render failed, err: %v", err)
	}
	// BGP mode should be deployed FRR
	if !strings.Contains(sb.String(), "frr-startup") {
		t.Error("BGP mode should be deployed FRR")
	}

	nlb := *lb
	nlb.Mode = "L2"
	sb1 := &strings.Builder{}
	if err := nlb.renderTo(sb1); err != nil {
		assert.FailNow(t, "deploy template render failed, err: %v", err)
	}
	// L2 mode should not be deployed FRR
	if strings.Contains(sb1.String(), "frr-startup") {
		t.Error("L2 mode should not be deployed FRR")
	}
}

func TestRenderIPAddressPool(t *testing.T) {
	expected := `
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: first-pool
  namespace: metallb-system
spec:
  addresses:
  - 192.168.20.20-192.168.20.30
`
	sb := &strings.Builder{}
	if err := lb.renderIPAddressPool(sb); err != nil {
		assert.FailNow(t, "ip address pool template render failed, err: %v", err)
	}
	if !assert.Equal(t, expected, sb.String()) {
		t.Errorf("expected is not the same as actual")
	}
}

func TestRenderAdvertisement(t *testing.T) {
	expected := `
apiVersion: metallb.io/v1beta1
kind: BGPAdvertisement
metadata:
  name: local
  namespace: metallb-system
`
	sb := &strings.Builder{}
	if err := lb.renderAdvertisement(sb); err != nil {
		assert.FailNow(t, "advertisement template render failed, err: %v", err)
	}
	if !assert.Equal(t, expected, sb.String()) {
		t.Errorf("expected is not the same as actual")
	}
}
