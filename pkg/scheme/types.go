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
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
)

// Custom scheme

type ListResult struct {
	Items      []interface{} `json:"items"`
	TotalItems int           `json:"totalItems"`
}

type PackageMetadata struct {
	Addons     []MetaResource `json:"addons"`      // kubernetes addons
	KcVersions MetaKcVersions `json:"kc_versions"` // kc release tag record, only tags with addon version updates are recorded
}

type MetaResource struct {
	Type    string `json:"type"` // addon type cni / csi / cri / app
	Name    string `json:"name"`
	Version string `json:"version"`
	Arch    string `json:"arch"` // amd64 / arm64
}

type MetaKcVersion struct {
	Version        string              `json:"version"`         // kc release tag
	LatestVersion  string              `json:"latest_version"`  // kc last latest tag, for the debug
	Charts         []MetaChart         `json:"charts"`          // kubernetes addons version control chart
	AddonManifests []MetaAddonManifest `json:"addon_manifests"` // a list of all addons versions currently supported by kc
}

type MetaChart struct {
	Name           string                    `json:"name"`            // k8s
	MinorVersion   string                    `json:"minor_version"`   // kubernetes minor version, for example: v1.23 v.24
	VersionControl []MetaChartVersionControl `json:"version_control"` // the current kubernetes`s addons version control chart
}

type MetaAddonManifest struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Versions []string `json:"versions"` // the currently supported component version of kc
}

type MetaChartVersionControl struct {
	Name             string `json:"name"`
	Type             string `json:"type"`              // cni csi cri
	RecommendVersion string `json:"recommend_version"` // for some addons, the recommended version is used when no version is specified
	MinVersion       string `json:"min_version"`       // kubernetes minor version the minimum installed version of this addon allowed
	MaxVersion       string `json:"max_version"`       // kubernetes minor version the maximum installed version of this addon allowed
}

// MetaKcVersions kc versions involved in updating kc support addons will be recorded.
// Means that the recorded versions will not be consecutive release tags.
type MetaKcVersions []MetaKcVersion

func (x MetaKcVersions) Len() int           { return len(x) }
func (x MetaKcVersions) Less(i, j int) bool { return x[i].Version < x[j].Version }
func (x MetaKcVersions) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

func (m *PackageMetadata) ReadMetadata(online bool, path string) error {
	if online {
		c := http.DefaultClient
		resp, err := c.Get(fmt.Sprintf("%s/metadata.json", downloader.CloudStaticServer))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if err = json.NewDecoder(resp.Body).Decode(m); err != nil {
			return err
		}
	} else {
		file := filepath.Join(path, "metadata.json")
		metadata, err := os.ReadFile(file)
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

func (m *PackageMetadata) AddonsSort() {
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
		if index < len(m.Addons) {
			m.Addons = append(m.Addons[:index], m.Addons[index+1:]...)
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

	return os.WriteFile(file, metaBytes, 0755)
}

func (m PackageMetadata) AddonsExist(name, version, arch string) bool {
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

// MatchKcVersion find the best version control for the incoming version
func (m PackageMetadata) MatchKcVersion(kcVersion string) MetaKcVersion {
	// `git describe` command, look at https://git-scm.com/docs/git-describe
	tag, newCommit := strutil.ParseGitDescribeInfo(kcVersion)
	var (
		exist   bool
		existKv MetaKcVersion
	)
	for _, v := range m.KcVersions {
		if v.Version == tag {
			exist = true
			existKv = v
		}
	}
	// no have new commit, it could be a branch or a tag.
	// Q: the master or release branch cannot get the latest version control until the branch produces a new commit ?
	// A: in a real world scenario, we would not be able to type a new tag without any commits record, so we can accept this situation.
	if !newCommit {
		if exist {
			return existKv
		}
		// no matching version information was found, use its previous version information
		preceding, _ := m.GetPrecedingAndSucceedingKcVersion(tag)
		if preceding.Version == "" {
			return preceding
		}
		return preceding
	}
	// have new commit, it must be a branch.
	// Even if we find equivalent version information, we will try to find information that is newer than the version.
	// Because the current branch may be in the debug phase of an upgrade addon, we want to make sure it gets the version information marked debug.
	if exist {
		_, succeeding := m.GetPrecedingAndSucceedingKcVersion(tag)
		if succeeding.Version != "" && succeeding.LatestVersion == tag {
			return succeeding
		}
		return existKv
	}
	// no matching version information was found
	// if the next version of it is marked debug, use it. not marked, the previous version is used.
	preceding, succeeding := m.GetPrecedingAndSucceedingKcVersion(tag)
	if succeeding.Version != "" && succeeding.LatestVersion == tag {
		return succeeding
	}
	if preceding.Version == "" {
		return preceding
	}
	return preceding
}

func (m PackageMetadata) GetPrecedingAndSucceedingKcVersion(version string) (MetaKcVersion, MetaKcVersion) {
	// shallow copy, easy to calculate
	versions := m.KcVersions

	// finds the preceding and succeeding elements at the specified position in the slice
	findEle := func(vs MetaKcVersions, version string) (MetaKcVersion, MetaKcVersion, bool) {
		var preceding, succeeding = MetaKcVersion{}, MetaKcVersion{}
		// ascending order
		sort.Sort(vs)
		for k, v := range vs {
			if v.Version == version {
				p := k - 1 // preceding element index
				s := k + 1 // succeeding element index
				if p >= 0 && p < len(vs) {
					preceding = vs[p]
				}
				if s >= 0 && s < len(vs) {
					succeeding = vs[s]
				}
				return preceding, succeeding, true
			}
		}
		return preceding, succeeding, false
	}
	// if the version is found in the slice the first time, it is returned as the final result
	if pv, sv, got := findEle(versions, version); got {
		return pv, sv
	}
	// ensure that this version is not in slice, append this version to slice
	versions = append(versions, MetaKcVersion{
		Version: version,
	})
	// find the elements front and back the specified version
	pv, sv, _ := findEle(versions, version)
	return pv, sv
}

func (m PackageMetadata) FindAddons(typeName string) []MetaResource {
	var addons []MetaResource
	for _, v := range m.Addons {
		if v.Type == typeName {
			addons = append(addons, v)
		}
	}
	return addons
}

// GetK8sVersionControlRules get k8s version control rules data
func (m PackageMetadata) GetK8sVersionControlRules(kcVersion string) []map[string]interface{} {
	metaKcVersion := m.MatchKcVersion(kcVersion)

	data := make([]map[string]interface{}, 0)
	for _, v := range m.FindAddons("k8s") {
		el := make(map[string]interface{})
		el["name"] = v.Name
		el["version"] = v.Version
		el["type"] = v.Type
		el["arch"] = v.Arch
		vs := make(map[string]interface{})
		for _, t := range metaKcVersion.FindAddonTypes(v.Version) {
			// skip k8s type
			if t == "k8s" {
				continue
			}
			var validAddons []map[string]interface{}
			vc := metaKcVersion.FindK8sVersionControl(v.Version).VersionControl
			for _, addon := range m.FindAddons(t) {
				// only a valid component version can be appended
				if valid, isDefault := metaKcVersion.CheckAddonVersion(addon.Version, addon.Name, addon.Version, vc); valid {
					validAddons = append(validAddons, map[string]interface{}{
						"name":    addon.Name,
						"version": addon.Version,
						"type":    addon.Type,
						"arch":    addon.Arch,
						"default": isDefault,
					})
				}
			}
			vs[t] = validAddons
		}
		el["version_control"] = vs
		data = append(data, el)
	}
	return data
}

func (kv MetaKcVersion) FindK8sVersionControl(k8sVersion string) MetaChart {
	for _, v := range kv.Charts {
		if strings.Contains(k8sVersion, v.MinorVersion) {
			return v
		}
	}
	return MetaChart{}
}

// CheckAddonVersion check whether the component is valid and default
// 1. `RecommendVersion` value indicates the default version of the control addon
// 2. `MinVersion\MaxVersion` value controls the valid range of the addon
// 3. `MetaAddonManifest.Versions` value controls addon versions currently supported by kc
// 4. The intersection of `MinVersion\MaxVersion` and `MetaAddonManifest.Versions` is the final effective version
func (kv MetaKcVersion) CheckAddonVersion(k8sVersion, name, version string, vc []MetaChartVersionControl) (bool, bool) {
	versionRoller := func(vs string) string {
		vs = strings.ReplaceAll(vs, "v", "")
		vs = strings.ReplaceAll(vs, ".", "")
		return vs
	}
	if vc == nil {
		vc = kv.FindK8sVersionControl(k8sVersion).VersionControl
	}
	if vc == nil {
		return false, false
	}
	var isDefault bool
	for _, v := range vc {
		// check whether the addon version is the default version
		if v.RecommendVersion == version {
			isDefault = true
		}
		if v.Name == name {
			if v.MinVersion != "" && versionRoller(version) < versionRoller(v.MinVersion) {
				return false, isDefault
			}
			if v.MaxVersion != "" && versionRoller(version) > versionRoller(v.MaxVersion) {
				return false, isDefault
			}
		}
	}
	manifest := kv.GetMetaAddonManifest(name)
	if manifest.Name == "" {
		return false, isDefault
	}
	for _, v := range manifest.Versions {
		if v == version {
			return true, isDefault
		}
	}
	return false, isDefault
}

func (kv MetaKcVersion) FindK8sMatchCniVersion(k8sVersion, cniName string) string {
	for _, v := range kv.FindK8sVersionControl(k8sVersion).VersionControl {
		if v.Name == cniName {
			return v.RecommendVersion
		}
	}
	return ""
}

func (kv MetaKcVersion) FindAddonTypes(k8sVersion string) []string {
	dict := make(map[string]int)
	for _, v := range kv.FindK8sVersionControl(k8sVersion).VersionControl {
		times := dict[v.Type]
		times++
		dict[v.Type] = times
	}
	var types []string
	for k := range dict {
		types = append(types, k)
	}
	return types
}

func (kv MetaKcVersion) GetMetaAddonManifest(name string) MetaAddonManifest {
	for _, v := range kv.AddonManifests {
		if v.Name == name {
			return v
		}
	}
	return MetaAddonManifest{}
}
