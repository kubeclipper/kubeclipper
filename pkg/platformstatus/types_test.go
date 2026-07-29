/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package platformstatus

import (
	"testing"
	"time"
)

func TestAggregate(t *testing.T) {
	tests := []struct {
		name       string
		components []Component
		want       Status
	}{
		{name: "healthy", components: []Component{{Status: Healthy}, {Status: Skipped}}, want: Healthy},
		{name: "unknown", components: []Component{{Status: Healthy}, {Status: Unknown}}, want: Unknown},
		{name: "degraded", components: []Component{{Status: Unknown}, {Status: Degraded}}, want: Degraded},
		{name: "unhealthy", components: []Component{{Status: Degraded}, {Status: Unhealthy}}, want: Unhealthy},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Aggregate(test.components); got != test.want {
				t.Fatalf("Aggregate() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPlatformStatusValidate(t *testing.T) {
	valid := &PlatformStatus{
		APIVersion: APIVersion,
		Kind:       Kind,
		Status:     Healthy,
		CheckedAt:  time.Now(),
		Components: []Component{
			{Name: "kc-server", Status: Healthy},
			{Name: "kc-etcd", Status: Healthy},
			{Name: "kc-agent", Status: Skipped},
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid status rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*PlatformStatus)
	}{
		{name: "missing metadata", mutate: func(status *PlatformStatus) { status.APIVersion = "" }},
		{name: "invalid status", mutate: func(status *PlatformStatus) { status.Status = Skipped }},
		{name: "missing component", mutate: func(status *PlatformStatus) { status.Components = status.Components[:2] }},
		{name: "wrong order", mutate: func(status *PlatformStatus) {
			status.Components[0], status.Components[1] = status.Components[1], status.Components[0]
		}},
		{name: "aggregate mismatch", mutate: func(status *PlatformStatus) { status.Components[1].Status = Unhealthy }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := *valid
			status.Components = append([]Component(nil), valid.Components...)
			test.mutate(&status)
			if err := status.Validate(); err == nil {
				t.Fatal("invalid status was accepted")
			}
		})
	}
}
