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

func TestAddOrRemoveInsecureRegistryToCRI(t *testing.T) {
	type args struct {
		ctx      context.Context
		criType  string
		registry string
		add      bool
		dryRun   bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "add an insecure registry to docker",
			args: args{
				ctx:      context.TODO(),
				criType:  "docker",
				registry: "127.0.0.1:5000",
				add:      true,
				dryRun:   true,
			},
		},
		{
			name: "remove an insecure registry from docker",
			args: args{
				ctx:      context.TODO(),
				criType:  "docker",
				registry: "127.0.0.1:5000",
				add:      false,
				dryRun:   true,
			},
		},
		{
			name: "add an insecure registry to containerd",
			args: args{
				ctx:      context.TODO(),
				criType:  "containerd",
				registry: "127.0.0.1:5000",
				add:      true,
				dryRun:   true,
			},
		},
		{
			name: "remove an insecure registry from containerd",
			args: args{
				ctx:      context.TODO(),
				criType:  "containerd",
				registry: "127.0.0.1:5000",
				add:      false,
				dryRun:   true,
			},
		},
		{
			name: "test with a wrong Cri",
			args: args{
				ctx:      context.TODO(),
				criType:  "docker1",
				registry: "127.0.0.1:5000",
				add:      true,
				dryRun:   true,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddOrRemoveInsecureRegistryToCRI(tt.args.ctx, tt.args.criType, tt.args.registry, tt.args.add, tt.args.dryRun); (err != nil) != tt.wantErr {
				t.Errorf("AddOrRemoveInsecureRegistryToCRI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
