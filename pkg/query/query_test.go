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
	"net/http"
	"reflect"
	"testing"

	"github.com/emicklei/go-restful"
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

func TestPagination_GetValidPagination(t *testing.T) {
	type fields struct {
		Limit  int
		Offset int
	}
	type args struct {
		total int
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStartIndex int
		wantEndIndex   int
	}{
		{
			name: "base",
			fields: fields{
				Limit:  DefaultLimit,
				Offset: 3,
			},
			args: args{
				total: 10,
			},
			wantStartIndex: 3,
			wantEndIndex:   10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pagination{
				Limit:  tt.fields.Limit,
				Offset: tt.fields.Offset,
			}
			gotStartIndex, gotEndIndex := p.GetValidPagination(tt.args.total)
			if gotStartIndex != tt.wantStartIndex {
				t.Errorf("GetValidPagination() gotStartIndex = %v, want %v", gotStartIndex, tt.wantStartIndex)
			}
			if gotEndIndex != tt.wantEndIndex {
				t.Errorf("GetValidPagination() gotEndIndex = %v, want %v", gotEndIndex, tt.wantEndIndex)
			}
		})
	}
}

func TestQuery_HasLabelSelector(t *testing.T) {
	type fields struct {
		Pagination           *Pagination
		TimeoutSeconds       *int64
		ResourceVersion      string
		Watch                bool
		LabelSelector        string
		FieldSelector        string
		Continue             string
		Limit                int64
		Reverse              bool
		OrderBy              string
		DryRun               string
		AllowWatchBookmarks  bool
		ResourceVersionMatch string
		FuzzySearch          map[string]string
	}
	type args struct {
		selector string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "base",
			fields: fields{
				LabelSelector: "test",
			},
			args: args{
				selector: "test",
			},
			want: true,
		},
		{
			name: "not exist",
			fields: fields{
				LabelSelector: "",
			},
			args: args{
				selector: "test",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &Query{
				Pagination:           tt.fields.Pagination,
				TimeoutSeconds:       tt.fields.TimeoutSeconds,
				ResourceVersion:      tt.fields.ResourceVersion,
				Watch:                tt.fields.Watch,
				LabelSelector:        tt.fields.LabelSelector,
				FieldSelector:        tt.fields.FieldSelector,
				Continue:             tt.fields.Continue,
				Limit:                tt.fields.Limit,
				Reverse:              tt.fields.Reverse,
				OrderBy:              tt.fields.OrderBy,
				DryRun:               tt.fields.DryRun,
				AllowWatchBookmarks:  tt.fields.AllowWatchBookmarks,
				ResourceVersionMatch: tt.fields.ResourceVersionMatch,
				FuzzySearch:          tt.fields.FuzzySearch,
			}
			if got := q.HasLabelSelector(tt.args.selector); got != tt.want {
				t.Errorf("HasLabelSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsePaging(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://example.com", nil)
	request := restful.NewRequest(r)
	type args struct {
		req *restful.Request
	}
	tests := []struct {
		name       string
		args       args
		wantLimit  int
		wantOffset int
	}{
		{
			name: "base",
			args: args{
				req: request,
			},
			wantLimit:  DefaultLimit,
			wantOffset: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotOffset := ParsePaging(tt.args.req)
			if gotLimit != tt.wantLimit {
				t.Errorf("ParsePaging() gotLimit = %v, want %v", gotLimit, tt.wantLimit)
			}
			if gotOffset != tt.wantOffset {
				t.Errorf("ParsePaging() gotOffset = %v, want %v", gotOffset, tt.wantOffset)
			}
		})
	}
}

func TestGetBoolValueWithDefault(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://example.com", nil)
	request := restful.NewRequest(r)
	type args struct {
		req  *restful.Request
		name string
		dv   bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "default value is false",
			args: args{
				req:  request,
				name: ParameterWatch,
			},
			want: false,
		},
		{
			name: "default value is true",
			args: args{
				req:  request,
				name: ParameterWatch,
				dv:   true,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBoolValueWithDefault(tt.args.req, tt.args.name, tt.args.dv); got != tt.want {
				t.Errorf("GetBoolValueWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStringValueWithDefault(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://example.com", nil)
	request := restful.NewRequest(r)
	type args struct {
		req  *restful.Request
		name string
		dv   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "base",
			args: args{
				req:  request,
				name: ParameterName,
				dv:   "Jack",
			},
			want: "Jack",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetStringValueWithDefault(tt.args.req, tt.args.name, tt.args.dv); got != tt.want {
				t.Errorf("GetStringValueWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetIntValueWithDefault(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://example.com", nil)
	request := restful.NewRequest(r)
	type args struct {
		req  *restful.Request
		name string
		dv   int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "base",
			args: args{
				req:  request,
				name: ParameterName,
				dv:   2,
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetIntValueWithDefault(tt.args.req, tt.args.name, tt.args.dv); got != tt.want {
				t.Errorf("GetIntValueWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInt64ValueWithDefault(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://example.com", nil)
	request := restful.NewRequest(r)
	type args struct {
		req  *restful.Request
		name string
		dv   int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "base",
			args: args{
				req:  request,
				name: ParameterName,
				dv:   2,
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetInt64ValueWithDefault(tt.args.req, tt.args.name, tt.args.dv); got != tt.want {
				t.Errorf("GetInt64ValueWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInformerRawQuery(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://example.com", nil)
	request := restful.NewRequest(r)
	type args struct {
		req *restful.Request
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "base",
			args: args{
				req: request,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInformerRawQuery(tt.args.req); got != tt.want {
				t.Errorf("IsInformerRawQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
