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

	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

// Custom scheme

type ListResult struct {
	Items      []interface{} `json:"items"`
	TotalItems int           `json:"totalItems"`
}

type ComponentMetaList []v1.MetaResource

func (p ComponentMetaList) Sort() {
	sort.SliceStable(p, func(i, j int) bool {
		if p[i].Type < p[j].Type {
			return true
		}
		if p[i].Type == p[j].Type {
			if p[i].Name < p[j].Name {
				return true
			}
			if p[i].Name == p[j].Name {
				return p[i].Version < p[j].Version
			}
			return false
		}
		return false
	})
}

func (p *ComponentMetaList) Delete(name, version, arch string) error {
	indexMap := make(map[string]int)
	for index, item := range *p {
		indexMap[fmt.Sprintf("%s%s%s", item.Name, item.Version, item.Arch)] = index
	}
	if index, ok := indexMap[fmt.Sprintf("%s%s%s", name, version, arch)]; ok {
		tmp := *p
		if len(tmp) == index+1 {
			*p = tmp[0:index]
		} else {
			*p = append(tmp[0:index], tmp[index+1:]...)
		}
	}

	return nil
}

func (p *ComponentMetaList) ReadFile(path string, sort bool) error {
	file := filepath.Join(path, "metadata.json")
	metadata, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(metadata, p)
	if err != nil {
		return err
	}

	if sort {
		p.Sort()
	}

	return nil
}

func (p *ComponentMetaList) ReadFromCloud(sort bool) error {
	c := http.DefaultClient
	resp, err := c.Get(fmt.Sprintf("%s/metadata.json", downloader.CloudStaticServer))
	if err != nil {
		return err
	}
	if err := json.NewDecoder(resp.Body).Decode(p); err != nil {
		return err
	}

	if sort {
		p.Sort()
	}

	return nil
}

func (p *ComponentMetaList) WriteFile(path string, sort bool) error {
	if sort {
		p.Sort()
	}

	metaBytes, err := json.MarshalIndent(&p, "", "  ")
	if err != nil {
		return err
	}
	file := filepath.Join(path, "metadata.json")

	return ioutil.WriteFile(file, metaBytes, 0755)
}

func (p *ComponentMetaList) Exist(name, version, arch string) bool {
	metaMap := make(map[string]struct{}, len(*p))
	for _, v := range *p {
		key := fmt.Sprintf("%s-%s-%s", v.Name, v.Version, v.Arch)
		metaMap[key] = struct{}{}
	}
	if _, ok := metaMap[fmt.Sprintf("%s-%s-%s", name, version, arch)]; ok {
		return true
	}
	return false
}

func (p *ComponentMetaList) AppendOnly(typeName, name, version, arch string) {
	if !p.Exist(name, version, arch) {
		*p = append(*p, v1.MetaResource{Type: typeName, Name: name, Version: version, Arch: arch})
	}
}
