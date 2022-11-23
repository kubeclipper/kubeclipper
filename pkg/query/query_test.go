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

package query

import (
	"reflect"
	"testing"
)

func Test_parseFuzzy(t *testing.T) {
	type args struct {
		conditionsStr string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "fuzzy-search",
			args: args{conditionsStr: "foo~bar,bar~baz"},
			want: map[string]string{
				"foo": "bar",
				"bar": "baz",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseFuzzy(tt.args.conditionsStr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseFuzzy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQuery_AddLabelSelector(t *testing.T) {
	type args struct {
		selectors []string
	}
	tests := []struct {
		name string
		old  args
		add  args
		want string
	}{
		{name: "empty", old: args{selectors: []string{""}}, add: args{selectors: []string{""}}, want: ""},
		{name: "normal", old: args{selectors: []string{""}}, add: args{selectors: []string{"a=b"}}, want: "a=b"},
		{name: "normal2", old: args{selectors: []string{"a=b"}}, add: args{selectors: []string{"c=d"}}, want: "a=b,c=d"},
		{name: "normal3", old: args{selectors: []string{"a=b", "c=d"}}, add: args{selectors: []string{"e=f"}}, want: "a=b,c=d,e=f"},
		{name: "override", old: args{selectors: []string{"a=b", "c=d"}}, add: args{selectors: []string{"c=d"}}, want: "a=b,c=d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := New()
			q.AddLabelSelector(tt.old.selectors)
			q.AddLabelSelector(tt.add.selectors)
			if q.LabelSelector != tt.want {
				t.Fatalf("want:%s got:%s", tt.want, q.LabelSelector)
			}
		})
	}
}
