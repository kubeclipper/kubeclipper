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

package cni

import "testing"

func TestIsHighCalicoVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "empty version",
			version: "",
			want:    false,
		},
		{
			name:    "v3.29.5 - before threshold",
			version: "v3.29.5",
			want:    false,
		},
		{
			name:    "v3.29.6 - exact threshold",
			version: "v3.29.6",
			want:    true,
		},
		{
			name:    "v3.29.7 - after threshold",
			version: "v3.29.7",
			want:    true,
		},
		{
			name:    "v3.30.0 - higher minor version",
			version: "v3.30.0",
			want:    true,
		},
		{
			name:    "v4.0.0 - higher major version",
			version: "v4.0.0",
			want:    true,
		},
		{
			name:    "v3.26.1 - older version",
			version: "v3.26.1",
			want:    false,
		},
		{
			name:    "3.29.6 - without v prefix",
			version: "3.29.6",
			want:    true,
		},
		{
			name:    "3.29.5 - without v prefix",
			version: "3.29.5",
			want:    false,
		},
		{
			name:    "invalid version - missing patch",
			version: "v3.29",
			want:    false,
		},
		{
			name:    "invalid version - non-numeric",
			version: "v3.29.x",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsHighCalicoVersion(tt.version)
			if got != tt.want {
				t.Errorf("IsHighCalicoVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestUseCalicoOperator(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "empty version",
			version: "",
			want:    false,
		},
		{
			name:    "v3.24.5 - before v3.26",
			version: "v3.24.5",
			want:    false,
		},
		{
			name:    "v3.25.0 - before v3.26",
			version: "v3.25.0",
			want:    false,
		},
		{
			name:    "v3.26.0 - exact threshold",
			version: "v3.26.0",
			want:    true,
		},
		{
			name:    "v3.26.1 - at threshold",
			version: "v3.26.1",
			want:    true,
		},
		{
			name:    "v3.29.6 - after threshold",
			version: "v3.29.6",
			want:    true,
		},
		{
			name:    "v3.30.0 - higher minor version",
			version: "v3.30.0",
			want:    true,
		},
		{
			name:    "v4.0.0 - higher major version",
			version: "v4.0.0",
			want:    true,
		},
		{
			name:    "3.26.1 - without v prefix",
			version: "3.26.1",
			want:    true,
		},
		{
			name:    "3.25.0 - without v prefix",
			version: "3.25.0",
			want:    false,
		},
		{
			name:    "v3.11.2 - old version",
			version: "v3.11.2",
			want:    false,
		},
		{
			name:    "v3.16.10 - old version",
			version: "v3.16.10",
			want:    false,
		},
		{
			name:    "v3.21.2 - old version",
			version: "v3.21.2",
			want:    false,
		},
		{
			name:    "v3.22.4 - old version",
			version: "v3.22.4",
			want:    false,
		},
		{
			name:    "invalid version - missing minor",
			version: "v3",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UseCalicoOperator(tt.version)
			if got != tt.want {
				t.Errorf("UseCalicoOperator(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
