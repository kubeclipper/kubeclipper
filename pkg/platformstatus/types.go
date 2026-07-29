/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package platformstatus

import (
	"context"
	"fmt"
	"time"
)

const (
	APIVersion = "config.kubeclipper.io/v1"
	Kind       = "PlatformStatus"
)

type Status string

const (
	Healthy   Status = "Healthy"
	Degraded  Status = "Degraded"
	Unhealthy Status = "Unhealthy"
	Unknown   Status = "Unknown"
	Skipped   Status = "Skipped"
)

type Check struct {
	Name           string `json:"name" yaml:"name"`
	Status         Status `json:"status" yaml:"status"`
	Message        string `json:"message" yaml:"message"`
	DurationMillis int64  `json:"durationMillis,omitempty" yaml:"durationMillis,omitempty"`
}

type Component struct {
	Name           string  `json:"name" yaml:"name"`
	Status         Status  `json:"status" yaml:"status"`
	Message        string  `json:"message" yaml:"message"`
	DurationMillis int64   `json:"durationMillis,omitempty" yaml:"durationMillis,omitempty"`
	Checks         []Check `json:"checks,omitempty" yaml:"checks,omitempty"`
}

type PlatformStatus struct {
	APIVersion     string      `json:"apiVersion" yaml:"apiVersion"`
	Kind           string      `json:"kind" yaml:"kind"`
	Status         Status      `json:"status" yaml:"status"`
	CheckedAt      time.Time   `json:"checkedAt" yaml:"checkedAt"`
	DurationMillis int64       `json:"durationMillis" yaml:"durationMillis"`
	Components     []Component `json:"components" yaml:"components"`
}

type Provider interface {
	PlatformStatus(ctx context.Context) *PlatformStatus
}

func New(components []Component, checkedAt time.Time, duration time.Duration) *PlatformStatus {
	return &PlatformStatus{
		APIVersion:     APIVersion,
		Kind:           Kind,
		Status:         Aggregate(components),
		CheckedAt:      checkedAt.UTC(),
		DurationMillis: duration.Milliseconds(),
		Components:     components,
	}
}

func Aggregate(components []Component) Status {
	result := Healthy
	for _, component := range components {
		switch component.Status {
		case Unhealthy:
			return Unhealthy
		case Degraded:
			result = Degraded
		case Unknown:
			if result == Healthy {
				result = Unknown
			}
		}
	}
	return result
}

func (s *PlatformStatus) Validate() error {
	if s.APIVersion != APIVersion {
		return fmt.Errorf("unexpected apiVersion %q", s.APIVersion)
	}
	if s.Kind != Kind {
		return fmt.Errorf("unexpected kind %q", s.Kind)
	}
	if s.CheckedAt.IsZero() {
		return fmt.Errorf("checkedAt is required")
	}
	if !isAggregateStatus(s.Status) {
		return fmt.Errorf("invalid platform status %q", s.Status)
	}
	expectedComponents := []string{"kc-server", "kc-etcd", "kc-agent"}
	if len(s.Components) != len(expectedComponents) {
		return fmt.Errorf("expected %d components, got %d", len(expectedComponents), len(s.Components))
	}
	for i := range s.Components {
		component := &s.Components[i]
		if component.Name != expectedComponents[i] {
			return fmt.Errorf("component %d is %q, expected %q", i, component.Name, expectedComponents[i])
		}
		if !isCheckStatus(component.Status) {
			return fmt.Errorf("component %q has invalid status %q", component.Name, component.Status)
		}
		for _, check := range component.Checks {
			if !isCheckStatus(check.Status) {
				return fmt.Errorf("check %q has invalid status %q", check.Name, check.Status)
			}
		}
	}
	if aggregated := Aggregate(s.Components); s.Status != aggregated {
		return fmt.Errorf("platform status %q does not match aggregate %q", s.Status, aggregated)
	}
	return nil
}

func isAggregateStatus(status Status) bool {
	return status == Healthy || status == Degraded || status == Unhealthy || status == Unknown
}

func isCheckStatus(status Status) bool {
	return isAggregateStatus(status) || status == Skipped
}
