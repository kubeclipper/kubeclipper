/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package staticresource

import (
	"context"
	"testing"

	"github.com/kubeclipper/kubeclipper/pkg/simple/staticserver"
)

func TestHealth(t *testing.T) {
	service, err := NewService(&staticserver.Options{
		BindAddress:  "127.0.0.1",
		InsecurePort: 0,
		Path:         t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	stopCh := make(chan struct{})
	if err := service.PrepareRun(stopCh); err != nil {
		t.Fatal(err)
	}
	if err := service.Run(stopCh); err != nil {
		t.Fatal(err)
	}
	defer close(stopCh)
	if err := service.Health(context.Background()); err != nil {
		t.Fatalf("health check failed: %v", err)
	}
}
