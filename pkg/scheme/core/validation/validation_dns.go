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

package validation

import (
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateDomain(c *corev1.Domain) field.ErrorList {
	return ValidateObjectMeta(&c.ObjectMeta, false, ValidateNodeName, field.NewPath("metadata"))
}
