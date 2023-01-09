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

package sliceutil

import (
	"reflect"
	"testing"
)

func TestMergeSlice(t *testing.T) {
	type args struct {
		s1 []string
		s2 []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "merge null slice",
			args: args{
				s1: nil,
				s2: nil,
			},
			want: []string{},
		},
		{
			name: "merge valid slice",
			args: args{
				s1: []string{"1"},
				s2: []string{"2", "3"},
			},
			want: []string{"1", "2", "3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeSlice(tt.args.s1, tt.args.s2); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringMask(t *testing.T) {
	type args struct {
		s        string
		start    int
		end      int
		maskChar rune
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "phoneMask",
			args: args{
				s:        "13588888888",
				start:    3,
				end:      8,
				maskChar: '*',
			},
			want: "135******88",
		},
		{
			name: "utf8MaskChar",
			args: args{
				s:        "ABC",
				start:    1,
				end:      1,
				maskChar: '密',
			},
			want: "A密C",
		},
		{
			name: "nameMask",
			args: args{
				s:        "赵小明",
				start:    1,
				end:      1,
				maskChar: '*',
			},
			want: "赵*明",
		},
		{
			name: "overflow",
			args: args{
				s:        "ABC",
				start:    1,
				end:      10,
				maskChar: '*',
			},
			want: "A**",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringMask(tt.args.s, tt.args.start, tt.args.end, tt.args.maskChar); got != tt.want {
				t.Errorf("StringMask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasString(t *testing.T) {
	type args struct {
		slice []string
		str   string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "slice has string",
			args: args{
				slice: []string{
					"a", "b", "c",
				},
				str: "a",
			},
			want: true,
		},
		{
			name: "slice has string",
			args: args{
				slice: []string{
					"a", "b", "c",
				},
				str: "d",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasString(tt.args.slice, tt.args.str); got != tt.want {
				t.Errorf("HasString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveString(t *testing.T) {
	type args struct {
		slice  []string
		remove func(item string) bool
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "remove string",
			args: args{
				slice: []string{
					"a", "b", "c",
				},
				remove: func(item string) bool {
					return item == "c"
				},
			},
			want: []string{
				"a", "b",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveString(tt.args.slice, tt.args.remove); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveString() = %v, want %v", got, tt.want)
			}
		})
	}
}
