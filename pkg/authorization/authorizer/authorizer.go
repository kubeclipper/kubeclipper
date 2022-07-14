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

package authorizer

import "k8s.io/apiserver/pkg/authentication/user"

var (
	_ Attributes = (*AttributesRecord)(nil)
)

// AttributesRecord implements Attributes interface.
type AttributesRecord struct {
	User            user.Info
	Verb            string
	APIGroup        string
	APIVersion      string
	Resource        string
	Subresource     string
	Name            string
	ResourceRequest bool
	Path            string
}

func (a *AttributesRecord) GetVerb() string {
	return a.Verb
}

func (a *AttributesRecord) GetResource() string {
	return a.Resource
}

func (a *AttributesRecord) GetSubresource() string {
	return a.Subresource
}

func (a *AttributesRecord) GetName() string {
	return a.Name
}

func (a *AttributesRecord) GetAPIGroup() string {
	return a.APIGroup
}

func (a *AttributesRecord) GetAPIVersion() string {
	return a.APIVersion
}

func (a *AttributesRecord) IsResourceRequest() bool {
	return a.ResourceRequest
}

func (a *AttributesRecord) GetPath() string {
	return a.Path
}

func (a *AttributesRecord) GetUser() user.Info {
	return a.User
}
