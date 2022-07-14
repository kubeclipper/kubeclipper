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

package registry

import (
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
)

type Image struct {
	Name string   `json:"name" yaml:"name"`
	Tags []string `json:"tags" yaml:"tags"`
}

func (i *Image) JSONPrint() ([]byte, error) {
	return printer.JSONPrinter(i)
}

func (i *Image) YAMLPrint() ([]byte, error) {
	return printer.YAMLPrinter(i)
}

func (i *Image) TablePrint() ([]string, [][]string) {
	headers := []string{"name", "tags"}
	var data [][]string
	for index, v := range i.Tags {
		if index == 0 {
			data = append(data, []string{i.Name, v})
		} else {
			data = append(data, []string{"", v})
		}
	}
	return headers, data
}

type Repositories struct {
	Repositories []string `json:"repositories" yaml:"repositories"`
}

func (i *Repositories) JSONPrint() ([]byte, error) {
	return printer.JSONPrinter(i)
}

func (i *Repositories) YAMLPrint() ([]byte, error) {
	return printer.YAMLPrinter(i)
}

func (i *Repositories) TablePrint() ([]string, [][]string) {
	headers := []string{"repositories"}
	var data [][]string
	for _, v := range i.Repositories {
		data = append(data, []string{v})
	}
	return headers, data
}
