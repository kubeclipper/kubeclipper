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

package request

import (
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	"k8s.io/apimachinery/pkg/api/validation/path"

	"github.com/kubeclipper/kubeclipper/pkg/query"
)

type Resolver interface {
	NewRequestInfo(req *http.Request) (*Info, error)
}

type Info struct {
	// IsResourceRequest indicates whether or not the request is for an API resource or subresource
	IsResourceRequest bool
	// Path is the URL path of the request
	Path string
	// Verb is the kube verb associated with the request for API requests, not the http verb.  This includes things like list and watch.
	// for non-resource requests, this is the lowercase http verb
	Verb string

	APIPrefix  string
	APIGroup   string
	APIVersion string
	// Resource is the name of the resource being requested.  This is not the kind.  For example: pods
	Resource string
	// Subresource is the name of the subresource being requested.  This is a different resource, scoped to the parent resource, but it may have a different kind.
	// For instance, /pods has the resource "pods" and the kind "Pod", while /pods/foo/status has the resource "pods", the sub resource "status", and the kind "Pod"
	// (because status operates on pods). The binding resource for a pod though may be /pods/foo/binding, which has resource "pods", subresource "binding", and kind "Binding".
	Subresource string
	// Name is empty for some verbs, but if the request directly indicates a name (not in body content) then this field is filled in.
	Name string
	// Parts are the path parts for the request, always starting with /{resource}/{name}
	// Parts []string
}

type InfoFactory struct {
	APIPrefixes sets.Set[string]
}

/*
Resource paths
/api/{api-group}/{version}/{resource}
eg: /api/core.kubeclipper.io/v1/nodes, /api/core.kubeclipper.io/v1/clusters
/api/{api-group}/{version}/{resource}/{resourceName}
eg: /api/core.kubeclipper.io/v1/nodes/node1


Kubernetes proxy paths
/cluster/{cluster-name}/{k8s-resource}
eg: /cluster/test-cluster/api/v1/namespaces

NonResource paths
/healthz
/metrics
/oauth/login
/oauth/token
*/

func (i *InfoFactory) NewRequestInfo(req *http.Request) (*Info, error) {
	// start with a non-resource request until proven otherwise
	requestInfo := Info{
		IsResourceRequest: false,
		Path:              req.URL.Path,
		Verb:              strings.ToLower(req.Method),
	}

	currentParts := splitPath(req.URL.Path)
	if len(currentParts) < 3 {
		// return a non-resource request
		return &requestInfo, nil
	}

	if !i.APIPrefixes.Has(currentParts[0]) {
		return &requestInfo, nil
	}

	switch req.Method {
	case "POST":
		requestInfo.Verb = "create"
	case "GET", "HEAD":
		requestInfo.Verb = "get"
	case "PUT":
		requestInfo.Verb = "update"
	case "PATCH":
		requestInfo.Verb = "patch"
	case "DELETE":
		requestInfo.Verb = "delete"
	default:
		requestInfo.Verb = ""
	}

	requestInfo.APIPrefix = currentParts[0]
	switch requestInfo.APIPrefix {
	case "api":
		requestInfo.APIGroup = currentParts[1]
		requestInfo.APIVersion = currentParts[2]
		requestInfo.IsResourceRequest = true
		currentParts = currentParts[3:]
	case "cluster":
		requestInfo.Name = currentParts[1]
		requestInfo.APIGroup = "core.kubeclipper.io"
		requestInfo.IsResourceRequest = true
		requestInfo.Resource = "clusters"
		requestInfo.Subresource = "proxy"
		return &requestInfo, nil
	}
	// parsing successful, so we now know the proper value for .Parts
	// requestInfo.Parts = currentParts

	// parts look like: resource/resourceName/subresource/other/stuff/we/don't/interpret
	switch {
	case len(currentParts) >= 3:
		requestInfo.Subresource = currentParts[2]
		fallthrough
	case len(currentParts) >= 2:
		requestInfo.Name = currentParts[1]
		fallthrough
	case len(currentParts) >= 1:
		requestInfo.Resource = currentParts[0]
	}

	// if there's no name on the request and we thought it was a get before, then the actual verb is a list or a watch
	if len(requestInfo.Name) == 0 && requestInfo.Verb == "get" {
		opts := query.NewFromRawQuery(req.URL.Query())
		if opts.Watch {
			requestInfo.Verb = "watch"
		} else {
			requestInfo.Verb = "list"
		}
		if opts.FieldSelector != "" {
			if name, ok := opts.GetFieldSelector().RequiresExactMatch("metadata.name"); ok {
				if len(path.IsValidPathSegmentName(name)) == 0 {
					requestInfo.Name = name
				}
			}
		}
	}
	return &requestInfo, nil
}

// splitPath returns the segments for a URL path.
func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}
