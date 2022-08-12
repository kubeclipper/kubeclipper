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

package scheme

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"
)

// Custom scheme

type ListResult struct {
	Items      []interface{} `json:"items"`
	TotalItems int           `json:"totalItems"`
}

type PackageMetadata struct {
	Addons []MetaResource `json:"addons"` // kubernetes addons
	Charts []MetaChart    `json:"charts"` // kubernetes addons version control chart
}

type MetaResource struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
}

type MetaChart struct {
	Name           string                    `json:"name"`            // k8s
	MinorVersion   string                    `json:"minor_version"`   // kubernetes minor version, for example: v1.23 v.24
	VersionControl []MetaChartVersionControl `json:"version_control"` // the current kubernetes`s addons version control chart
}

type MetaChartVersionControl struct {
	Name             string `json:"name"`
	Type             string `json:"type"`              // cni csi cri
	RecommendVersion string `json:"recommend_version"` // for some addons, the recommended version is used when no version is specified
	MinVersion       string `json:"min_version"`       // kubernetes minor version the minimum installed version of this addon allowed
	MaxVersion       string `json:"max_version"`       // kubernetes minor version the maximum installed version of this adon allowed
}

func (m *PackageMetadata) ReadMetadata(online bool, path string) error {
	if online {
		c := http.DefaultClient
		resp, err := c.Get(fmt.Sprintf("%s/metadata.json", downloader.CloudStaticServer))
		if err != nil {
			return err
		}
		if err = json.NewDecoder(resp.Body).Decode(m); err != nil {
			return err
		}
	} else {
		file := filepath.Join(path, "metadata.json")
		metadata, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		err = json.Unmarshal(metadata, m)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m PackageMetadata) AddonsSort() {
	sort.SliceStable(m.Addons, func(i, j int) bool {
		if m.Addons[i].Type < m.Addons[j].Type {
			return true
		}
		if m.Addons[i].Type == m.Addons[j].Type {
			if m.Addons[i].Name < m.Addons[j].Name {
				return true
			}
			if m.Addons[i].Name == m.Addons[j].Name {
				return m.Addons[i].Version < m.Addons[j].Version
			}
			return false
		}
		return false
	})
}

func (m *PackageMetadata) AddonsDelete(name, version, arch string) error {
	indexMap := make(map[string]int)
	for index, item := range m.Addons {
		indexMap[fmt.Sprintf("%s%s%s", item.Name, item.Version, item.Arch)] = index
	}
	if index, ok := indexMap[fmt.Sprintf("%s%s%s", name, version, arch)]; ok {
		tmp := m.Addons
		if len(tmp) == index+1 {
			m.Addons = tmp[0:index]
		} else {
			m.Addons = append(tmp[0:index], tmp[index+1:]...)
		}
	}

	return nil
}

func (m *PackageMetadata) WriteFile(path string, sort bool) error {
	if sort {
		m.AddonsSort()
	}

	metaBytes, err := json.MarshalIndent(&m, "", "  ")
	if err != nil {
		return err
	}
	file := filepath.Join(path, "metadata.json")

	return ioutil.WriteFile(file, metaBytes, 0755)
}

func (m *PackageMetadata) AddonsExist(name, version, arch string) bool {
	metaMap := make(map[string]struct{}, len(m.Addons))
	for _, v := range m.Addons {
		key := fmt.Sprintf("%s-%s-%s", v.Name, v.Version, v.Arch)
		metaMap[key] = struct{}{}
	}
	if _, ok := metaMap[fmt.Sprintf("%s-%s-%s", name, version, arch)]; ok {
		return true
	}
	return false
}

func (m *PackageMetadata) AddonsAppendOnly(typeName, name, version, arch string) {
	if !m.AddonsExist(name, version, arch) {
		m.Addons = append(m.Addons, MetaResource{Type: typeName, Name: name, Version: version, Arch: arch})
	}
}

func (m *PackageMetadata) SupportK8sMinorVersion(version string) bool {
	for _, v := range m.Charts {
		if strings.Contains(version, v.MinorVersion) {
			return true
		}
	}
	return false
}

func (m *PackageMetadata) FindK8sVersionControl(k8sVersion string) MetaChart {
	for _, v := range m.Charts {
		if strings.Contains(k8sVersion, v.MinorVersion) {
			return v
		}
	}
	return MetaChart{}
}

func (m *PackageMetadata) CheckAddonVersion(k8sVersion, name, version string) bool {
	versionRoller := func(vs string) string {
		vs = strings.ReplaceAll(vs, "v", "")
		vs = strings.ReplaceAll(vs, ".", "")
		return vs
	}
	for _, v := range m.FindK8sVersionControl(k8sVersion).VersionControl {
		if v.Name == name {
			if v.MinVersion != "" && versionRoller(version) < versionRoller(v.MinVersion) {
				return false
			}
			if v.MaxVersion != "" && versionRoller(version) > versionRoller(v.MaxVersion) {
				return false
			}
		}
	}
	return true
}

func (m *PackageMetadata) FindK8sMatchCniVersion(k8sVersion, cniName string) string {
	for _, v := range m.FindK8sVersionControl(k8sVersion).VersionControl {
		if v.Name == cniName {
			return v.RecommendVersion
		}
	}
	return ""
}
