/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package config

import (
	"fmt"
	"reflect"
	"testing"
)

func TestConfig_Merge(t *testing.T) {
	cfgCert := &Config{
		Servers: map[string]*Server{
			"default-cert": {
				Server:                "localhost:8080",
				CertificateAuthority:  "/root/.kc/ca.crt",
				InsecureSkipTLSVerify: true,
			},
		},
		AuthInfos: map[string]*AuthInfo{
			"kcctl": {
				ClientCertificate: "/root/.kc/kcctl.crt",
				ClientKey:         "/root/.kc/kcctl.key",
			},
		},
		CurrentContext: fmt.Sprintf("%s@cert", "kcctl"),
		Contexts: map[string]*Context{
			fmt.Sprintf("%s@cert", "kcctl"): {
				AuthInfo: "kcctl",
				Server:   "cert",
			},
		},
	}
	cfg2 := &Config{
		Servers: map[string]*Server{
			"default": {
				Server:                "localhost:8080",
				InsecureSkipTLSVerify: true,
			},
		},
		AuthInfos: map[string]*AuthInfo{
			"admin": {
				Token: "admin's token",
			},
		},
		CurrentContext: fmt.Sprintf("%s@default", "admin"),
		Contexts: map[string]*Context{
			fmt.Sprintf("%s@default", "admin"): {
				AuthInfo: "admin",
				Server:   "default",
			},
		},
	}
	cfgCert.Merge(cfg2)

	cfgTarget := &Config{
		Servers: map[string]*Server{
			"default-cert": {
				Server:                "localhost:8080",
				CertificateAuthority:  "/root/.kc/ca.crt",
				InsecureSkipTLSVerify: true,
			},
			"default": {
				Server:                "localhost:8080",
				InsecureSkipTLSVerify: true,
			},
		},
		AuthInfos: map[string]*AuthInfo{
			"kcctl": {
				ClientCertificate: "/root/.kc/kcctl.crt",
				ClientKey:         "/root/.kc/kcctl.key",
			},
			"admin": {
				Token: "admin's token",
			},
		},
		CurrentContext: fmt.Sprintf("%s@default", "admin"),
		Contexts: map[string]*Context{
			fmt.Sprintf("%s@cert", "kcctl"): {
				AuthInfo: "kcctl",
				Server:   "cert",
			},
			fmt.Sprintf("%s@default", "admin"): {
				AuthInfo: "admin",
				Server:   "default",
			},
		},
	}

	if !reflect.DeepEqual(cfgCert, cfgTarget) {
		t.Fatalf("merge fail,want %v got %v", cfgCert, cfgTarget)
	}
}
