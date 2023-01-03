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
	"errors"
	"net/url"
	"regexp"
)

var (
	ErrInvalidSCName        = errors.New("invalid name of storage class")
	ErrInvalidNamespace     = errors.New("invalid namespace")
	ErrInvalidReclaimPolicy = errors.New("invalid reclaim policy")
	ErrInvalidLBMode        = errors.New("invalid load balancer mode")
)

const (
	dns1123LabelFmt   string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"             // DNS-1123 label
	hostnameRFC952Fmt string = `[a-zA-Z]([a-zA-Z0-9\-]+[\.]?)*[a-zA-Z0-9]` // RFC 952
	linuxFilePathFmt  string = `(/[^/ ]*)+/?`
)

var (
	namespaceReg           = regexp.MustCompile("^" + dns1123LabelFmt + "$")
	storageClassReg        = regexp.MustCompile("^" + dns1123LabelFmt + "$")
	hostnameRegexRFC952Reg = regexp.MustCompile("^" + hostnameRFC952Fmt + "$")
	linuxFilePathReg       = regexp.MustCompile("^" + linuxFilePathFmt + "$")
)

func MatchKubernetesNamespace(namespace string) bool {
	return namespaceReg.MatchString(namespace)
}

func MatchKubernetesStorageClass(storageClass string) bool {
	return storageClassReg.MatchString(storageClass)
}

func IsHostNameRFC952(hostname string) bool {
	return hostnameRegexRFC952Reg.MatchString(hostname)
}

func MatchLinuxFilePath(path string) bool {
	return linuxFilePathReg.MatchString(path)
}

func IsURL(raw string) bool {
	_, err := url.ParseRequestURI(raw)
	return err == nil
}

func MatchKubernetesReclaimPolicy(policy string) error {
	if policy == "Retain" || policy == "Delete" {
		return nil
	}
	return ErrInvalidReclaimPolicy
}

func MatchLoadBalancerMode(mode string) error {
	if mode == "L2" || mode == "BGP" {
		return nil
	}
	return ErrInvalidLBMode
}
