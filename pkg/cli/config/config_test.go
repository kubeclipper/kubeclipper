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
