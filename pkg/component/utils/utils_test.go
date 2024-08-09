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

package utils

import (
	"context"
	"testing"
)

func TestLoadImage(t *testing.T) {

	type args struct {
		ctx     context.Context
		dryRun  bool
		file    string
		criType string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "load a docker image",
			args: args{
				ctx:     context.TODO(),
				dryRun:  true,
				file:    "/demoPath",
				criType: "docker",
			},
		},
		{
			name: "load a containerd image",
			args: args{
				ctx:     context.TODO(),
				dryRun:  true,
				file:    "/demoPath",
				criType: "containerd",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := LoadImage(tt.args.ctx, tt.args.dryRun, tt.args.file, tt.args.criType); err != nil {
				t.Errorf("LoadImage() error = %v", err)
			}
		})
	}
}
