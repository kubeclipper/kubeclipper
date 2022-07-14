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

var (
	_agentSteps = defaultAgentStepHandler()
)

var (
	ErrStepExist     = errors.New("component already exist")
	ErrStepKeyFormat = errors.New("component key must be name/version")
)

const OfflinePackagesKeyFormat = "%s-%s-%s"

type agentStep struct {
	steps map[string]StepRunnable
}

func defaultAgentStepHandler() agentStep {
	return agentStep{steps: map[string]StepRunnable{}}
}

// RegisterAgentStep KV must format at componentName/version/stepName
func RegisterAgentStep(kv string, p StepRunnable) error {
	if !checkAgentStepKey(kv) {
		return ErrStepKeyFormat
	}
	return _agentSteps.registerAgentStep(kv, p)
}

func LoadAgentStep(kv string) (StepRunnable, bool) {
	return _agentSteps.load(kv)
}

func (h *agentStep) load(kv string) (StepRunnable, bool) {
	c, exist := h.steps[kv]
	return c, exist
}

func (h *agentStep) registerAgentStep(kv string, p StepRunnable) error {
	_, exist := h.steps[kv]
	if exist {
		return ErrStepExist
	}
	h.steps[kv] = p
	return nil
}

func checkAgentStepKey(kv string) bool {
	parts := strings.Split(kv, "/")
	return len(parts) == 3
}
