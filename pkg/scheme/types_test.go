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
	"reflect"
	"testing"
)

var (
	metadata = PackageMetadata{
		Addons: []MetaResource{
			{Name: "k8s", Version: "v1.23.6", Arch: "amd64", Type: "k8s"},
			{Name: "containerd", Version: "1.6.4", Arch: "amd64", Type: "cri"},
			{Name: "docker", Version: "19.03.12", Arch: "amd64", Type: "cri"},
			{Name: "calico", Version: "v3.22.4", Arch: "amd64", Type: "cni"},
			{Name: "nfs", Version: "v4.0.2", Arch: "amd64", Type: "csi"},
		},
		KcVersions: MetaKcVersions{
			{
				Version:       "v1.1.0",
				LatestVersion: "",
				Charts: []MetaChart{
					{
						Name:         "k8s",
						MinorVersion: "v1.23",
						VersionControl: []MetaChartVersionControl{
							{Name: "containerd", Type: "cri", RecommendVersion: "1.6.3", MinVersion: "1.6.2"},
							{Name: "docker", Type: "cri", RecommendVersion: "19.03.12", MinVersion: "", MaxVersion: ""},
							{Name: "calico", Type: "cni", RecommendVersion: "v3.22.4", MinVersion: "v3.21.2", MaxVersion: "v3.22.4"},
							{Name: "nfs", Type: "csi", RecommendVersion: "v4.0.2", MinVersion: "", MaxVersion: ""},
						},
					},
				},
				AddonManifests: []MetaAddonManifest{
					{Name: "containerd", Type: "cri", Versions: []string{"1.6.3", "1.6.4"}},
					{Name: "docker", Type: "cri", Versions: []string{"19.03.12", ""}},
					{Name: "calico", Type: "cni", Versions: []string{"v3.16.10", "v3.22.4"}},
					{Name: "nfs", Type: "csi", Versions: []string{"v4.0.2"}},
				},
			},
			{
				Version:       "v1.8.0",
				LatestVersion: "",
				Charts: []MetaChart{
					{
						Name:         "k8s",
						MinorVersion: "v1.23",
						VersionControl: []MetaChartVersionControl{
							{Name: "containerd", Type: "cri", RecommendVersion: "1.6.3", MinVersion: "1.6.2"},
							{Name: "docker", Type: "cri", RecommendVersion: "19.03.12", MinVersion: "", MaxVersion: ""},
							{Name: "calico", Type: "cni", RecommendVersion: "v3.22.4", MinVersion: "v3.21.2", MaxVersion: "v3.22.4"},
							{Name: "nfs", Type: "csi", RecommendVersion: "v4.0.2", MinVersion: "", MaxVersion: ""},
						},
					},
				},
				AddonManifests: []MetaAddonManifest{
					{Name: "containerd", Type: "cri", Versions: []string{"1.6.5", "1.6.6"}},
					{Name: "docker", Type: "cri", Versions: []string{"19.03.13"}},
					{Name: "calico", Type: "cni", Versions: []string{"v3.16.11", "v3.22.5"}},
					{Name: "nfs", Type: "csi", Versions: []string{"v4.0.3"}},
				},
			},
			{
				Version: "v2.2.1",
				Charts: []MetaChart{
					{
						Name:         "k8s",
						MinorVersion: "v1.23",
						VersionControl: []MetaChartVersionControl{
							{Name: "containerd", Type: "cri", RecommendVersion: "1.6.3", MinVersion: "1.6.2"},
							{Name: "docker", Type: "cri", RecommendVersion: "19.03.12", MinVersion: "", MaxVersion: ""},
							{Name: "calico", Type: "cni", RecommendVersion: "v3.22.4", MinVersion: "v3.21.2", MaxVersion: "v3.22.4"},
							{Name: "nfs", Type: "csi", RecommendVersion: "v4.0.2", MinVersion: "", MaxVersion: ""},
						},
					},
				},
				AddonManifests: []MetaAddonManifest{
					{Name: "containerd", Type: "cri", Versions: []string{"1.6.7", "1.6.8"}},
					{Name: "docker", Type: "cri", Versions: []string{"19.03.14"}},
					{Name: "calico", Type: "cni", Versions: []string{"v3.16.12", "v3.22.6"}},
					{Name: "nfs", Type: "csi", Versions: []string{"v4.0.4"}},
				},
			},
			{
				Version:       "v2.3.2",
				LatestVersion: "v2.3.1",
				Charts: []MetaChart{
					{
						Name:         "k8s",
						MinorVersion: "v1.23",
						VersionControl: []MetaChartVersionControl{
							{Name: "containerd", Type: "cri", RecommendVersion: "1.6.3", MinVersion: "1.6.2"},
							{Name: "docker", Type: "cri", RecommendVersion: "19.03.12", MinVersion: "", MaxVersion: ""},
							{Name: "calico", Type: "cni", RecommendVersion: "v3.22.4", MinVersion: "v3.21.2", MaxVersion: "v3.22.4"},
							{Name: "nfs", Type: "csi", RecommendVersion: "v4.0.2", MinVersion: "", MaxVersion: ""},
						},
					},
				},
				AddonManifests: []MetaAddonManifest{
					{Name: "containerd", Type: "cri", Versions: []string{"1.6.7", "1.6.8"}},
					{Name: "docker", Type: "cri", Versions: []string{"19.03.14"}},
					{Name: "calico", Type: "cni", Versions: []string{"v3.16.12", "v3.22.6"}},
					{Name: "nfs", Type: "csi", Versions: []string{"v4.0.4"}},
				},
			},
		},
	}
	metaKcVersion = MetaKcVersion{
		Version: "v2.3.2",
		//LatestVersion: "v2.3.1",
		Charts: []MetaChart{
			{
				Name:         "k8s",
				MinorVersion: "v1.23",
				VersionControl: []MetaChartVersionControl{
					{Name: "containerd", Type: "cri", RecommendVersion: "1.6.3", MinVersion: "1.6.2"},
					{Name: "docker", Type: "cri", RecommendVersion: "19.03.12", MinVersion: "", MaxVersion: ""},
					{Name: "calico", Type: "cni", RecommendVersion: "v3.22.4", MinVersion: "v3.11.0", MaxVersion: "v3.22.4"},
					{Name: "nfs", Type: "csi", RecommendVersion: "v4.0.2", MinVersion: "", MaxVersion: ""},
				},
			},
		},
		AddonManifests: []MetaAddonManifest{
			{Name: "containerd", Type: "cri", Versions: []string{"1.6.7", "1.6.8"}},
			{Name: "docker", Type: "cri", Versions: []string{"19.03.14"}},
			{Name: "calico", Type: "cni", Versions: []string{"v3.16.10", "v3.22.4"}},
			{Name: "nfs", Type: "csi", Versions: []string{"v4.0.4"}},
		},
	}

	sortedAddons = []MetaResource{
		{Name: "calico", Version: "v3.22.4", Arch: "amd64", Type: "cni"},
		{Name: "containerd", Version: "1.6.4", Arch: "amd64", Type: "cri"},
		{Name: "docker", Version: "19.03.12", Arch: "amd64", Type: "cri"},
		{Name: "nfs", Version: "v4.0.2", Arch: "amd64", Type: "csi"},
		{Name: "k8s", Version: "v1.23.6", Arch: "amd64", Type: "k8s"},
	}
)

func TestPackageMetadata_AddonsSort(t *testing.T) {
	metadata.AddonsSort()
	if !reflect.DeepEqual(metadata.Addons, sortedAddons) {
		t.Fatalf("the sorting results are not as expectedg")
	}
}

func TestPackageMetadata_AddonsExist(t *testing.T) {
	for _, v := range []struct {
		name    string
		version string
		arch    string
		result  bool
	}{
		{name: "calico", version: "v3.11.2", arch: "amd64", result: false},
		{name: "calico", version: "v3.22.4", arch: "amd64", result: true},
	} {
		t.Run("AddonsExist", func(t *testing.T) {
			if res := metadata.AddonsExist(v.name, v.version, v.arch); res != v.result {
				t.Fatalf("AddonsExist() = %v, want = %v", res, v.result)
			}
		})
	}
}

func TestPackageMetadata_AddonsDelete(t *testing.T) {
	for _, v := range []struct {
		name    string
		version string
		arch    string
	}{
		{name: "nfs", version: "v4.0.2", arch: "amd64"},
	} {
		t.Run("AddonsDelete", func(t *testing.T) {
			if err := metadata.AddonsDelete(v.name, v.version, v.arch); err != nil {
				t.Fatalf("AddonsDelete() error: %+v", err)
			}
			for _, add := range metadata.Addons {
				if add.Name == v.name && add.Name == v.version && add.Arch == v.arch {
					t.Fatalf("an unspecified addon was removed")
				}
			}
		})
	}
}

func TestPackageMetadata_AddonsAppendOnly(t *testing.T) {
	for _, v := range []struct {
		typeName string
		name     string
		version  string
		arch     string
	}{
		{typeName: "csi", name: "cinder", version: "v1.23", arch: "amd64"},
	} {
		t.Run("AddonsAppendOnly", func(t *testing.T) {
			metadata.AddonsAppendOnly(v.typeName, v.name, v.version, v.arch)
			var suc bool
			for _, add := range metadata.Addons {
				if add.Type == v.typeName && add.Name == v.name && add.Version == v.version && add.Arch == v.arch {
					suc = true
				}
			}
			if !suc {
				t.Fatalf("append addon failed")
			}
		})
	}
}

func TestPackageMetadata_GetFrontAndBackKcVersion(t *testing.T) {
	for _, v := range []struct {
		Version               string
		WantPrecedingVersion  string
		WantSucceedingVersion string
	}{
		{Version: "v1.0.0", WantPrecedingVersion: "", WantSucceedingVersion: "v1.1.0"},
		{Version: "v1.1.0", WantPrecedingVersion: "", WantSucceedingVersion: "v1.8.0"},
		{Version: "v2.3.1", WantPrecedingVersion: "v2.2.1", WantSucceedingVersion: "v2.3.2"},
		{Version: "v2.3.2", WantPrecedingVersion: "v2.2.1", WantSucceedingVersion: ""},
	} {
		t.Run("GetPrecedingAndSucceedingKcVersion", func(t *testing.T) {
			preceding, succeeding := metadata.GetPrecedingAndSucceedingKcVersion(v.Version)
			if preceding.Version != v.WantPrecedingVersion || succeeding.Version != v.WantSucceedingVersion {
				t.Fatalf("GetPrecedingAndSucceedingKcVersion() = %v, %v, want = %v, %v", preceding.Version, succeeding.Version, v.WantPrecedingVersion, v.WantSucceedingVersion)
			}
		})
	}
}

func TestPackageMetadata_MatchKcVersion(t *testing.T) {
	for _, v := range []struct {
		Version     string
		WantVersion string
	}{
		{Version: "v1.0.0", WantVersion: ""},
		{Version: "v1.1.0", WantVersion: "v1.1.0"},
		{Version: "v2.3.2", WantVersion: "v2.3.2"},
		{Version: "v2.3.2-3-g47a2fd39", WantVersion: "v2.3.2"},
		{Version: "v3.3.2-3-g47a2fd39", WantVersion: "v2.3.2"},
		{Version: "v2.3.1-3-g47a2fd39", WantVersion: "v2.3.2"},
		{Version: "v2.2.1-3-g47a2fd39", WantVersion: "v2.2.1"},
		{Version: "v1.0.0-3-g47a2fd39", WantVersion: ""},
	} {
		t.Run("MatchKcVersion", func(t *testing.T) {
			matchVersion := metadata.MatchKcVersion(v.Version)
			if matchVersion.Version != v.WantVersion {
				t.Fatalf("MatchKcVersion() = %v, want = %v", matchVersion.Version, v.WantVersion)
			}
		})
	}
}

func TestPackageMetadata_FindAddons(t *testing.T) {
	for _, v := range []struct {
		typeName string
		nums     int
	}{
		{typeName: "k8s", nums: 1},
		{typeName: "app", nums: 0},
	} {
		t.Run("FindAddons", func(t *testing.T) {
			if res := metadata.FindAddons(v.typeName); len(res) != v.nums {
				t.Fatalf("FindAddons() = %v, want = %v", len(res), v.nums)
			}
		})
	}
}

func TestPackageMetadata_FindAddonTypes(t *testing.T) {
	for _, v := range []struct {
		k8sVersion string
		want       int
	}{
		{k8sVersion: "v1.18.20", want: 0},
		{k8sVersion: "v1.23.6", want: 3},
	} {
		t.Run("FindAddonTypes", func(t *testing.T) {
			res := metaKcVersion.FindAddonTypes(v.k8sVersion)
			if len(res) != v.want {
				t.Fatalf("FindAddonTypes() = %v, want = %v", v.want, res)
			}
		})
	}
}

func TestPackageMetadata_GetK8sVersionControlData(t *testing.T) {
	testMetadata := PackageMetadata{
		Addons: []MetaResource{
			{Name: "k8s", Version: "v1.23.6", Arch: "amd64", Type: "k8s"},
			{Name: "docker", Version: "19.03.12", Arch: "amd64", Type: "cri"},
			{Name: "calico", Version: "v3.22.4", Arch: "amd64", Type: "cni"},
			{Name: "nfs", Version: "v4.0.2", Arch: "amd64", Type: "csi"},
		},
	}
	t.Run("GetK8sVersionControlRules", func(t *testing.T) {
		res := testMetadata.GetK8sVersionControlRules("")
		if len(res) != 1 {
			t.Fatalf("GetK8sVersionControlRules() return slice length is %v, want = %v", len(res), 1)
		}
		if res[0]["name"] != "k8s" && res[0]["version"] != "v1.23.6" {
			t.Fatalf("GetK8sVersionControlRules() return k8s version is %v, want = %v", res[0]["name"], "v1.23.6")
		}
	})
}

func TestMetaKcVersion_FindK8sMatchCniVersion(t *testing.T) {
	for _, v := range []struct {
		k8sVersion string
		cniName    string
		want       string
	}{
		{k8sVersion: "v1.18.20", cniName: "calico", want: ""},
		{k8sVersion: "v1.23.6", cniName: "calico", want: "v3.22.4"},
	} {
		t.Run("FindK8sMatchCniVersion", func(t *testing.T) {
			if cni := metaKcVersion.FindK8sMatchCniVersion(v.k8sVersion, v.cniName); cni != v.want {
				t.Fatalf("FindK8sMatchCniVersion() = %v, want = %v", cni, v.want)
			}
		})
	}
}

func TestMetaKcVersion_FindK8sVersionControl(t *testing.T) {
	for _, v := range []struct {
		k8sVersion string
		want       string
	}{
		{k8sVersion: "v1.18.20", want: ""},
		{k8sVersion: "v1.23.6", want: "v1.23"},
	} {
		t.Run("FindK8sVersionControl", func(t *testing.T) {
			if chart := metaKcVersion.FindK8sVersionControl(v.k8sVersion); chart.MinorVersion != v.want {
				t.Fatalf("FindK8sVersionControl() = %v, want = %v", chart.MinorVersion, v.want)
			}
		})
	}
}

func TestMetaKcVersion_CheckAddonVersion(t *testing.T) {
	for _, v := range []struct {
		k8sVersion string
		name       string
		version    string
		result     bool
		isDefault  bool
		vc         []MetaChartVersionControl
	}{
		{k8sVersion: "v1.18.20", name: "calico", version: "v3.22.4", result: false, isDefault: false, vc: nil},
		{k8sVersion: "v1.23.6", name: "calico", version: "v3.11.2", result: false, isDefault: false, vc: metaKcVersion.Charts[0].VersionControl},
		{k8sVersion: "v1.23.6", name: "calico", version: "v3.16.10", result: true, isDefault: false, vc: metaKcVersion.Charts[0].VersionControl},
		{k8sVersion: "v1.23.6", name: "calico", version: "v3.11.2", result: false, isDefault: false, vc: nil},
		{k8sVersion: "v1.23.6", name: "calico", version: "v3.22.4", result: true, isDefault: true, vc: nil},
	} {
		t.Run("CheckAddonVersion", func(t *testing.T) {
			if res, isDefault := metaKcVersion.CheckAddonVersion(v.k8sVersion, v.name, v.version, v.vc); res != v.result || isDefault != v.isDefault {
				t.Fatalf("CheckAddonVersion() = %v, %v, want = %v, %v", res, isDefault, v.result, v.isDefault)
			}
		})
	}
}
