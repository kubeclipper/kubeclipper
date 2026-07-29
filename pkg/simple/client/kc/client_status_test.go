/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package kc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
)

func TestFromConfigWithoutValidationRejectsIncompleteConfig(t *testing.T) {
	_, err := FromConfigWithoutValidation(config.Config{CurrentContext: "missing"})
	if err == nil || !strings.Contains(err.Error(), "current context") {
		t.Fatalf("expected current context error, got %v", err)
	}
}

func TestPlatformStatusRejectsInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != platformStatusPath {
			t.Fatalf("request path = %q, want %q", request.URL.Path, platformStatusPath)
		}
		writer.Header().Set("Content-Type", "application/json")
		if _, err := writer.Write([]byte(`{}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewClientWithOpts(WithEndpoint(server.URL), WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.PlatformStatus(context.Background()); err == nil || !strings.Contains(err.Error(), "invalid platform status response") {
		t.Fatalf("expected invalid response error, got %v", err)
	}
}
