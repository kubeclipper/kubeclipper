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

package etcd

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

type Options struct {
	// storage transport config
	ServerList []string `json:"serverList" yaml:"serverList"`
	// storage transport TLS credentials
	KeyFile       string `json:"keyFile" yaml:"keyFile"`
	CertFile      string `json:"certFile" yaml:"certFile"`
	TrustedCAFile string `json:"trustedCAFile" yaml:"trustedCAFile"`

	Prefix string `json:"prefix" yaml:"prefix"`
	Paging bool   `json:"paging" yaml:"paging"`
	// CompactionInterval is an interval of requesting compaction from apiserver.
	// If the value is 0, no compaction will be issued.
	CompactionInterval time.Duration `json:"compactionInterval" yaml:"compactionInterval"`
	// CountMetricPollPeriod specifies how often should count metric be updated
	CountMetricPollPeriod time.Duration `json:"countMetricPollPeriod" yaml:"countMetricPollPeriod"`

	// To enable protobuf as storage format, it is enough
	// to set it to "application/vnd.kubernetes.protobuf".
	DefaultStorageMediaType string `json:"defaultStorageMediaType" yaml:"defaultStorageMediaType"`
	DeleteCollectionWorkers int    `json:"deleteCollectionWorkers" yaml:"deleteCollectionWorkers"`
	EnableGarbageCollection bool   `json:"enableGarbageCollection" yaml:"enableGarbageCollection"`

	// Set EnableWatchCache to false to disable all watch caches
	EnableWatchCache bool `json:"enableWatchCache" yaml:"enableWatchCache"`
	// Set DefaultWatchCacheSize to zero to disable watch caches for those resources that have no explicit cache size set
	DefaultWatchCacheSize int `json:"defaultWatchCacheSize" yaml:"defaultWatchCacheSize"`
	// WatchCacheSizes represents override to a given resource
	WatchCacheSizes []string `json:"watchCacheSizes" yaml:"watchCacheSizes"`
}

func NewEtcdOptions() *Options {
	return &Options{
		Prefix:                  "/registry/kubeclipper-server",
		Paging:                  true,
		CompactionInterval:      5 * time.Minute,
		CountMetricPollPeriod:   1 * time.Minute,
		DefaultStorageMediaType: "application/json",
		DeleteCollectionWorkers: 1,
		EnableGarbageCollection: true,
		EnableWatchCache:        true,
		DefaultWatchCacheSize:   100,
	}
}

func (s *Options) Validate() []error {
	if s == nil {
		return nil
	}

	var allErrors []error
	if len(s.ServerList) == 0 {
		allErrors = append(allErrors, fmt.Errorf("--etcd-servers must be specified"))
	}

	return allErrors
}

// AddEtcdFlags adds flags related to etcd storage for a specific APIServer to the specified FlagSet
func (s *Options) AddFlags(fs *pflag.FlagSet) {
	if s == nil {
		return
	}

	fs.StringVar(&s.DefaultStorageMediaType, "storage-media-type", s.DefaultStorageMediaType, ""+
		"The media type to use to store objects in storage. "+
		"Some resources or storage backends may only support a specific media type and will ignore this setting.")
	fs.IntVar(&s.DeleteCollectionWorkers, "delete-collection-workers", s.DeleteCollectionWorkers,
		"Number of workers spawned for DeleteCollection call. These are used to speed up namespace cleanup.")

	fs.BoolVar(&s.EnableGarbageCollection, "enable-garbage-collector", s.EnableGarbageCollection, ""+
		"Enables the generic garbage collector. MUST be synced with the corresponding flag "+
		"of the kube-controller-manager.")

	fs.BoolVar(&s.EnableWatchCache, "watch-cache", s.EnableWatchCache,
		"Enable watch caching in the apiserver")

	fs.IntVar(&s.DefaultWatchCacheSize, "default-watch-cache-size", s.DefaultWatchCacheSize,
		"Default watch cache size. If zero, watch cache will be disabled for resources that do not have a default watch size set.")

	fs.StringSliceVar(&s.WatchCacheSizes, "watch-cache-sizes", s.WatchCacheSizes, ""+
		"Watch cache size settings for some resources (pods, nodes, etc.), comma separated. "+
		"The individual setting format: resource[.group]#size, where resource is lowercase plural (no version), "+
		"group is omitted for resources of apiVersion v1 (the legacy core API) and included for others, "+
		"and size is a number. It takes effect when watch-cache is enabled. "+
		"Some resources (replicationcontrollers, endpoints, nodes, pods, services, apiservices.apiregistration.k8s.io) "+
		"have system defaults set by heuristics, others default to default-watch-cache-size")

	fs.StringSliceVar(&s.ServerList, "etcd-servers", s.ServerList,
		"List of etcd servers to connect with (scheme://ip:port), comma separated.")

	fs.StringVar(&s.Prefix, "etcd-prefix", s.Prefix,
		"The prefix to prepend to all resource paths in etcd.")

	fs.StringVar(&s.KeyFile, "etcd-keyfile", s.KeyFile,
		"SSL key file used to secure etcd communication.")

	fs.StringVar(&s.CertFile, "etcd-certfile", s.CertFile,
		"SSL certification file used to secure etcd communication.")

	fs.StringVar(&s.TrustedCAFile, "etcd-cafile", s.TrustedCAFile,
		"SSL Certificate Authority file used to secure etcd communication.")

	fs.DurationVar(&s.CompactionInterval, "etcd-compaction-interval", s.CompactionInterval,
		"The interval of compaction requests. If 0, the compaction request from apiserver is disabled.")

	fs.DurationVar(&s.CountMetricPollPeriod, "etcd-count-metric-poll-period", s.CountMetricPollPeriod, ""+
		"Frequency of polling etcd for number of resources per type. 0 disables the metric collection.")
}
