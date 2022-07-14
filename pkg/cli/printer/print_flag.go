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

package printer

import (
	"encoding/json"
	"io"

	"sigs.k8s.io/yaml"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type PrintFlags struct {
	format string
}

func (p *PrintFlags) AllowedFormats() []string {
	if p == nil {
		return []string{}
	}
	return []string{"json", "yaml", "table"}
}

// TODO
func (p *PrintFlags) Print(pr ResourcePrinter, w io.Writer) error {
	switch p.format {
	case "json":
		data, err := pr.JSONPrint()
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte("\n"))
		return err
	case "yaml":
		data, err := pr.YAMLPrint()
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	case "table":
		fallthrough
	default:
		table := tablewriter.NewWriter(w)
		headers, data := pr.TablePrint()
		table.SetHeader(headers)
		for _, v := range data {
			table.Append(v)
		}
		table.Render()
		return nil
	}
}

func (p *PrintFlags) AddFlags(c *cobra.Command) {
	if p == nil {
		return
	}
	c.Flags().StringVarP(&p.format, "output", "o", p.format, "Output format either: json,yaml,table")
}

func NewPrintFlags() *PrintFlags {
	return &PrintFlags{format: "table"}
}

func JSONPrinter(data interface{}) ([]byte, error) {
	return json.MarshalIndent(data, "", "\t")
}

func YAMLPrinter(data interface{}) ([]byte, error) {
	b, err := JSONPrinter(data)
	if err != nil {
		return nil, err
	}
	return yaml.JSONToYAML(b)
}
