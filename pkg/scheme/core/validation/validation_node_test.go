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
	"testing"

	"github.com/google/uuid"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateNode(t *testing.T) {
	// TODO: test failed

	/*type args struct {
		c *corev1.Node
	}
	t1 := getTestNode()

	tests := []struct {
		name string
		args args
		want field.ErrorList
	}{
		// TODO: Add test cases.
		{
			name: "node name is uuid",
			args: args{
				c: t1,
			},
			want: field.ErrorList{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateNode(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateNode() = %v, want %v", got, tt.want)
			}
		})
	}*/
}

func getTestNode() *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: uuid.New().String(),
		},
		ProxyIpv4CIDR: "",
		Status:        corev1.NodeStatus{},
	}
}
