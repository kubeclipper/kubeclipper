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

package metrics

import (
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	compbasemetrics "k8s.io/component-base/metrics"
)

type DefaultMetrics struct{}

var (
	Defaults        DefaultMetrics
	defaultRegistry compbasemetrics.KubeRegistry
	// MustRegister registers registerable metrics but uses the defaultRegistry, panic upon the first registration that causes an error
	MustRegister func(...compbasemetrics.Registerable)
	// Register registers a collectable metric but uses the defaultRegistry
	Register        func(compbasemetrics.Registerable) error
	RawMustRegister func(...prometheus.Collector)
)

func init() {
	defaultRegistry = compbasemetrics.NewKubeRegistry()
	MustRegister = defaultRegistry.MustRegister
	Register = defaultRegistry.Register
	RawMustRegister = defaultRegistry.RawMustRegister
}

// Install adds the DefaultMetrics handler
func (m DefaultMetrics) Install(c *restful.Container) {
	RawMustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	RawMustRegister(collectors.NewGoCollector())

	c.HandleWithFilter("/metrics", Handler())
}

func Handler() http.Handler {
	return promhttp.InstrumentMetricHandler(prometheus.NewRegistry(), promhttp.HandlerFor(defaultRegistry, promhttp.HandlerOpts{}))
}
