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

package component

import (
	"errors"
	"strings"
)

var _tmpl = defaultTmpl()

var (
	ErrTemplateExist     = errors.New("component template already exist")
	ErrTemplateKeyFormat = errors.New("component template key must be name/version/templateName")
)

const (
	RegisterTemplateKeyFormat = "%s/%s/%s"
	RegisterStepKeyFormat     = "%s/%s/%s"
)

type tmpl struct {
	template map[string]TemplateRender
}

func defaultTmpl() tmpl {
	return tmpl{template: map[string]TemplateRender{}}
}

func RegisterTemplate(kv string, t TemplateRender) error {
	if !checkTemplateKey(kv) {
		return ErrTemplateKeyFormat
	}
	return _tmpl.registerTemplate(kv, t)
}

func LoadTemplate(kv string) (TemplateRender, bool) {
	return _tmpl.load(kv)
}

func (h *tmpl) load(kv string) (TemplateRender, bool) {
	c, exist := h.template[kv]
	return c, exist
}

func (h *tmpl) registerTemplate(kv string, p TemplateRender) error {
	_, exist := h.template[kv]
	if exist {
		return ErrTemplateExist
	}
	h.template[kv] = p
	return nil
}

func checkTemplateKey(kv string) bool {
	parts := strings.Split(kv, "/")
	return len(parts) == 3
}
