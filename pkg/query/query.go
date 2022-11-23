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
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/client/clientrest"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	PagingParam                   = "paging"
	ParameterName                 = "name"
	ParameterLabelSelector        = "labelSelector"
	ParameterFieldSelector        = "fieldSelector"
	ParameterContinue             = "continue"
	ParameterLimit                = "limit"
	ParameterWatch                = "watch"
	ParameterAllowWatchBookMark   = "allowWatchBookmarks"
	ParameterResourceVersion      = "resourceVersion"
	ParameterResourceVersionMatch = "resourceVersionMatch"
	ParameterTimeoutSeconds       = "timeoutSeconds"
	ParameterNode                 = "node"
	ParameterOperation            = "operation"
	ParameterStep                 = "step"
	ParameterOffset               = "offset"
	OrderByParam                  = "orderBy"
	ParamReverse                  = "reverse"
	ParamDryRun                   = "dryRun"
	ParamRole                     = "role"
	ParamOffline                  = "offline"
	ParameterSubDomain            = "subdomain"
	ParameterFuzzySearch          = "fuzzy"
	ParameterForce                = "force"
)

const (
	DefaultLimit      = 10
	DefaultPage       = 1
	MinTimeoutSeconds = 1800
)

const (
	ResourceVersionMatchNotOlderThan = "NotOlderThan"
	ResourceVersionMatchExact        = "Exact"
)

// Query represents api search terms
type Query struct {
	Pagination      *Pagination
	TimeoutSeconds  *int64
	ResourceVersion string
	Watch           bool
	LabelSelector   string
	FieldSelector   string
	Continue        string
	Limit           int64
	Reverse         bool
	// TODO: add orderby
	OrderBy              string
	DryRun               string
	AllowWatchBookmarks  bool
	ResourceVersionMatch string
	FuzzySearch          map[string]string
}

var NoPagination = func() *Pagination {
	return newPagination(-1, 0)
}

var noPagination = newPagination(-1, 0)

type Pagination struct {
	Limit  int
	Offset int
}

func (p *Pagination) GetValidPagination(total int) (startIndex, endIndex int) {

	// no pagination
	if p.Limit == noPagination.Limit {
		return 0, total
	}

	// out of range
	if p.Limit < 0 || p.Offset < 0 || p.Offset > total {
		return 0, 0
	}

	startIndex = p.Offset
	endIndex = startIndex + p.Limit

	if endIndex > total {
		endIndex = total
	}

	return startIndex, endIndex
}

// make sure that pagination is valid
func newPagination(limit int, offset int) *Pagination {
	return &Pagination{
		Limit:  limit,
		Offset: offset,
	}
}

func (q *Query) GetLabelSelector() labels.Selector {
	var (
		selector labels.Selector
		err      error
	)
	if selector, err = labels.Parse(q.LabelSelector); err != nil {
		return labels.Everything()
	}
	return selector
}

func (q *Query) GetFieldSelector() fields.Selector {
	var (
		selector fields.Selector
		err      error
	)
	if selector, err = fields.ParseSelector(q.FieldSelector); err != nil {
		return fields.Everything()
	}
	return selector
}

// AddLabelSelector add labelSelector to query.
func (q *Query) AddLabelSelector(selectors []string) {
	if q.LabelSelector == "" {
		q.LabelSelector = strings.Join(selectors, ",")
		return
	}
	for _, selector := range selectors {
		if q.HasLabelSelector(selector) {
			continue
		}
		q.LabelSelector = fmt.Sprintf("%s,%s", q.LabelSelector, selector)
	}
}

// HasLabelSelector check label is exist
func (q *Query) HasLabelSelector(selector string) bool {
	split := strings.Split(q.LabelSelector, ",")
	for _, v := range split {
		if v == selector {
			return true
		}
	}
	return false
}

// DeepCopy copy query.
func (q *Query) DeepCopy() *Query {
	out := *q
	if q.Pagination != nil {
		out.Pagination = newPagination(q.Pagination.Limit, q.Pagination.Offset)
	}
	if q.TimeoutSeconds != nil {
		newTimeoutSeconds := *q.TimeoutSeconds
		out.TimeoutSeconds = &newTimeoutSeconds
	}
	if q.FuzzySearch != nil {
		newFuzzySearch := make(map[string]string)
		for k, v := range q.FuzzySearch {
			newFuzzySearch[k] = v
		}
		out.FuzzySearch = q.FuzzySearch
	}
	return &out
}

func New() *Query {
	return &Query{
		Pagination: NoPagination(),
		Watch:      false,
	}
}

func NewFromRawQuery(rawQuery url.Values) *Query {
	q := New()
	if len(rawQuery) == 0 {
		return q
	}
	q.FieldSelector = rawQuery.Get(ParameterFieldSelector)
	q.LabelSelector = rawQuery.Get(ParameterLabelSelector)
	q.Watch = getBoolWithDefault(rawQuery.Get(ParameterWatch), false)
	return q
}

func getBoolWithDefault(value string, dv bool) bool {
	if value == "" {
		return dv
	}
	if v, err := strconv.ParseBool(value); err == nil {
		return v
	}
	return dv
}

func ParsePaging(req *restful.Request) (limit, offset int) {
	paging := req.QueryParameter(PagingParam)
	limit = 10
	page := DefaultPage
	if paging != "" {
		if groups := regexp.MustCompile(`^limit=(-?\d+),page=(\d+)$`).FindStringSubmatch(paging); len(groups) == 3 {
			limit = AtoiOrDefault(groups[1], DefaultLimit)
			page = AtoiOrDefault(groups[2], DefaultPage)
		}
	} else {
		// try to parse from format ?limit=10&page=1
		limit = AtoiOrDefault(req.QueryParameter("limit"), DefaultLimit)
		page = AtoiOrDefault(req.QueryParameter("page"), DefaultPage)
	}
	offset = (page - 1) * limit
	return
}

func ParseQueryParameter(request *restful.Request) *Query {
	query := New()

	limit, offset := ParsePaging(request)
	query.Pagination = newPagination(limit, offset)
	query.LabelSelector = request.QueryParameter(ParameterLabelSelector)
	query.FieldSelector = request.QueryParameter(ParameterFieldSelector)
	query.Watch = GetBoolValueWithDefault(request, ParameterWatch, false)
	query.ResourceVersion = request.QueryParameter(ParameterResourceVersion)
	query.Reverse = GetBoolValueWithDefault(request, ParamReverse, false)
	if query.Watch {
		query.AllowWatchBookmarks = GetBoolValueWithDefault(request, ParameterAllowWatchBookMark, false)
	}
	query.Limit = GetInt64ValueWithDefault(request, ParameterLimit, 0)
	query.Continue = request.QueryParameter(ParameterContinue)
	query.ResourceVersionMatch = request.QueryParameter(ParameterResourceVersionMatch)
	if ts := request.QueryParameter(ParameterTimeoutSeconds); ts != "" {
		v, err := strconv.ParseInt(ts, 10, 64)
		if err == nil {
			query.TimeoutSeconds = &v
		}
	}
	query.FuzzySearch = parseFuzzy(request.QueryParameter(ParameterFuzzySearch))
	return query
}

func parseFuzzy(conditionsStr string) map[string]string {
	if conditionsStr == "" {
		return nil
	}
	fuzzy := make(map[string]string)
	for conditionsStr != "" {
		key := conditionsStr
		if i := strings.Index(key, ","); i >= 0 {
			key, conditionsStr = key[:i], key[i+1:]
		} else {
			conditionsStr = ""
		}
		if key == "" {
			continue
		}
		if i := strings.IndexAny(key, "~"); i >= 0 {
			k, v := key[:i], key[i+1:]
			fuzzy[k] = v
		}
	}
	return fuzzy
}

func GetBoolValueWithDefault(req *restful.Request, name string, dv bool) bool {
	reverse := req.QueryParameter(name)
	if v, err := strconv.ParseBool(reverse); err == nil {
		return v
	}
	return dv
}

func GetStringValueWithDefault(req *restful.Request, name string, dv string) string {
	v := req.QueryParameter(name)
	if v == "" {
		v = dv
	}
	return v
}

func GetWrapperBoolWithDefault(req *restful.Request, name string, dv *bool) *bool {
	reverse := req.QueryParameter(name)
	if v, err := strconv.ParseBool(reverse); err == nil {
		return &v
	}
	return dv
}

func GetIntValueWithDefault(req *restful.Request, name string, dv int) int {
	v := req.QueryParameter(name)
	if v, err := strconv.ParseInt(v, 10, 64); err == nil {
		return int(v)
	}
	return dv
}

func GetInt64ValueWithDefault(req *restful.Request, name string, dv int64) int64 {
	v := req.QueryParameter(name)
	if v, err := strconv.ParseInt(v, 10, 64); err == nil {
		return v
	}
	return dv
}

func GetInt64ValuePointerWithDefault(req *restful.Request, name string, dv int64) *int64 {
	v := GetInt64ValueWithDefault(req, name, dv)
	return &v
}

func AtoiOrDefault(str string, defVal int) int {
	if result, err := strconv.Atoi(str); err == nil {
		return result
	}
	return defVal
}

func IsInformerRawQuery(req *restful.Request) bool {
	if v := req.HeaderParameter(clientrest.QueryTypeHeader); v == clientrest.InformerQuery {
		return true
	}
	return false
}
