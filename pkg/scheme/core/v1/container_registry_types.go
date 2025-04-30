/*
 *
 *  * Copyright 2022 KubeClipper Authors.
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

package v1

import (
	"strconv"

	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=false

type Registry struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	RegistrySpec      `json:",inline"`
}

type RegistrySpec struct {
	Scheme       string        `json:"scheme,omitempty"`
	Host         string        `json:"host,omitempty"`
	SkipVerify   bool          `json:"skipVerify,omitempty"`
	CA           string        `json:"ca,omitempty"`
	RegistryAuth *RegistryAuth `json:"auth,omitempty"`
}

type RegistryAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// RegistryList is a resource containing a list of RegistryList objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RegistryList struct {
	metav1.TypeMeta
	// +optional
	metav1.ListMeta

	// Items is the list of registry.
	Items []Registry
}

func (n *RegistryList) YAMLPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.YAMLPrinter(n.Items[0])
	}
	return printer.YAMLPrinter(n)
}

func (n *RegistryList) TablePrint() ([]string, [][]string) {
	headers := []string{"name", "scheme", "host", "skip-tls-verify", "ca", "auth"}
	var data [][]string
	for _, registry := range n.Items {
		data = append(data, []string{registry.Name,
			registry.Scheme,
			registry.Host,
			strconv.FormatBool(registry.SkipVerify),
			strconv.FormatBool(registry.CA != ""),
			strconv.FormatBool(registry.RegistryAuth != nil),
		})
	}
	return headers, data
}

func (n *RegistryList) JSONPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.JSONPrinter(n.Items[0])
	}
	return printer.JSONPrinter(n)
}
