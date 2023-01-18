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

package platform

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func Test_platformOperator_eventFilter(t *testing.T) {
	type args struct {
		obj runtime.Object
		in1 *query.Query
	}
	tests := []struct {
		name string
		args args
		want []runtime.Object
	}{
		{
			name: "eventFilter test",
			args: args{
				obj: &v1.EventList{
					Items: []v1.Event{
						{
							AuditID: "test1",
						},
						{
							AuditID: "test2",
						},
						{
							AuditID: "test3",
						},
					},
				},
				in1: query.New(),
			},
			want: []runtime.Object{
				&v1.Event{
					AuditID: "test1",
				},
				&v1.Event{
					AuditID: "test2",
				},
				&v1.Event{
					AuditID: "test3",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &platformOperator{
				storage:      nil,
				eventStorage: nil,
			}
			if got := p.eventFilter(tt.args.obj, tt.args.in1); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("eventFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_eventCustomFilter(t *testing.T) {
	type args struct {
		ev    *v1.Event
		key   string
		value string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "base",
			args: args{
				ev: &v1.Event{
					Username: "Tom",
				},
				key:   "name",
				value: "Tom",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eventCustomFilter(tt.args.ev, tt.args.key, tt.args.value); got != tt.want {
				t.Errorf("eventCustomFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}
