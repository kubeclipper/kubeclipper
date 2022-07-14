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

package template

import (
	"bytes"
	"errors"
	"io"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type AdvancedTemplate struct {
	*template.Template
	funcMap template.FuncMap
}

const (
	SplitPipelineKey       = "split"
	StrArrIndexPipelineKey = "index"
)

func funcMap() template.FuncMap {
	f := sprig.TxtFuncMap()

	// add extra functionality
	extra := template.FuncMap{}

	for k, v := range extra {
		f[k] = v
	}
	return f
}

// New news an AdvancedTemplate instance.
func New() *AdvancedTemplate {
	fm := funcMap()
	return &AdvancedTemplate{
		Template: template.New("gotmpl").Funcs(fm),
		funcMap:  fm,
	}
}

// RegisterFunc register a customized functionality with key to AdvancedTemplate.
// If the functionality key is already existed, the relative function will be override.
func (at *AdvancedTemplate) RegisterFunc(key string, customFunc interface{}) error {
	if key == "" {
		return errors.New("empty functionality key")
	}
	if customFunc == nil {
		return errors.New("empty function")
	}
	at.funcMap[key] = customFunc
	at.Funcs(at.funcMap)
	return nil
}

// Render fills the source template string with variables struct like helm.
func (at *AdvancedTemplate) Render(tmpl string, vars interface{}) (string, error) {
	t, err := at.Parse(tmpl)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	if err := t.Execute(buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderTo fills the source template string with variables struct
// and writes content to io.Writer.
func (at *AdvancedTemplate) RenderTo(w io.Writer, tmpl string, vars interface{}) (int, error) {
	out, err := at.Render(tmpl, vars)
	if err != nil {
		return 0, err
	}
	return w.Write([]byte(out))
}
