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
	"context"
	"os"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

// JSON represents any valid JSON value.
// These types are supported: bool, int64, float64, string, []interface{}, map[string]interface{} and nil.
type JSON interface{}

const (
	JSONSchemaTypeObject = "object"
	JSONSchemaTypeBool   = "boolean"
	JSONSchemaTypeString = "string"
	JSONSchemaTypeArray  = "array"
	JSONSchemaTypeInt    = "number"
)

const (
	InternalCategoryNodes      = "nodes"
	InternalCategoryKubernetes = "kubernetes"
	InternalCategoryStorage    = "storage"
	InternalCategoryPAAS       = "PAAS"
	InternalCategoryLB         = "LB"
)

type Props struct {
	Min int  `json:"min"`
	Max *int `json:"max,omitempty"`
}

type JSONSchemaProps struct {
	Title        string                     `json:"title"`
	Properties   map[string]JSONSchemaProps `json:"properties,omitempty"`
	Type         string                     `json:"type,omitempty"`
	Mask         bool                       `json:"mask,omitempty"`
	Default      JSON                       `json:"default,omitempty"`
	Description  string                     `json:"description,omitempty"`
	Required     []string                   `json:"required,omitempty"`
	Items        *JSONSchemaProps           `json:"items,omitempty"`
	Enum         []JSON                     `json:"enum,omitempty"`
	EnumNames    []string                   `json:"enumNames,omitempty"`
	Dependencies []string                   `json:"dependencies,omitempty"`
	Priority     int                        `json:"priority,omitempty"`
	Props        *Props                     `json:"props,omitempty"`
}

type Meta struct {
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	Icon           string           `json:"icon"`
	Unique         bool             `json:"unique"`
	Template       bool             `json:"template"`
	Category       string           `json:"category"`
	Deprecated     bool             `json:"deprecated"`
	Name           string           `json:"name"`
	Version        string           `json:"version"`
	Dependence     []string         `json:"dependence"`
	TimeoutSeconds int              `json:"timeoutSeconds"`
	Priority       int              `json:"priority,omitempty"`
	Schema         *JSONSchemaProps `json:"schema"`
}

func PropsMax(i int) *int {
	max := i
	return &max
}

type ObjectMeta interface {
	NewInstance() ObjectMeta
}

type HealthCheck interface {
	Ns() string
	Svc() string
	RequestPath() string
	Supported() bool
}

type Interface interface {
	ObjectMeta
	HealthCheck
	GetInstanceName() string
	// Object metadata
	GetComponentMeta(lang Lang) Meta
	// Component method
	GetDependence() []string
	RequireExtraCluster() []string
	CompleteWithExtraCluster(extra map[string]ExtraMetadata) error
	Validate() error
	InitSteps(ctx context.Context) error
	GetInstallSteps() []v1.Step
	GetUninstallSteps() []v1.Step
	GetUpgradeSteps() []v1.Step
	GetImageRepoMirror() string
}

// OfflinePackages key must format as version-osVendor-osArch
// value is packages
// eg for docker 19.03, docker-19.03-centos7-x86_64
type OfflinePackages map[string][]string

type Options struct {
	DryRun bool
}

type StepRunnable interface {
	Runnable
	ObjectMeta
}

type TemplateRender interface {
	Render(ctx context.Context, opts Options) error
	ObjectMeta
}

type Runnable interface {
	Install(ctx context.Context, opts Options) ([]byte, error)
	Uninstall(ctx context.Context, opts Options) ([]byte, error)
}

type FuncIndex func() (key, value []byte, err error)

type OperationLogFile interface {
	GetRootDir() string
	CreateOperationDir(opID string) error
	GetOperationDir(opID string) (path string, err error)
	CreateStepLogFile(opID, stepID string) (file *os.File, err error)
	GetStepLogFile(opID, stepID string) (path string, err error)
	GetStepLogContent(opID, stepID string, offset int64, length int) (content []byte, deliverySize int64, logSize int64, err error)
	CreateStepLogFileAndAppend(opID, stepID string, data []byte) error
	TruncateStepLogFile(opID, stepID string) error
}
