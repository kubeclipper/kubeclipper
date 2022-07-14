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
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/validation"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var ValidateUserName = apimachineryvalidation.NameIsDNSSubdomain

func ValidateUser(u *corev1.User) field.ErrorList {
	allErrs := validation.ValidateObjectMeta(&u.ObjectMeta, false, ValidateUserName, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateUserSpec(&u.Spec, field.NewPath("spec"))...)
	return allErrs
}

func ValidateUserSpec(spec *corev1.UserSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	//if spec.Email == "" {
	//	allErrs = append(allErrs, field.Invalid(fldPath.Child("email"), spec.Email, "must be valid email address"))
	//}
	if spec.EncryptedPassword == "" {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("password"), spec.EncryptedPassword, "must be valid password"))
	}
	// TODO: validate user other field
	return allErrs
}
