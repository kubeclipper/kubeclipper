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

package options

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime/schema"

	etcdRESTOptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/kubeclipper/kubeclipper/pkg/scheme"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
	"github.com/kubeclipper/kubeclipper/pkg/server"
	serverconfig "github.com/kubeclipper/kubeclipper/pkg/server/config"
)

type ServerOptions struct {
	*serverconfig.Config
}

func NewServerOptions() *ServerOptions {
	return &ServerOptions{
		Config: serverconfig.New(),
	}
}

func (s *ServerOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("generic")
	s.GenericServerRunOptions.AddFlags(fs, s.GenericServerRunOptions)
	s.EtcdOptions.AddFlags(fss.FlagSet("etcd"))
	s.CacheOptions.AddFlags(fss.FlagSet("cache"))
	s.MQOptions.AddFlags(fss.FlagSet("mq"))
	s.LogOptions.AddFlags(fss.FlagSet("log"))
	s.AuthenticationOptions.AddFlags(fss.FlagSet("authentication"))
	s.AuditOptions.AddFlags(fss.FlagSet("audit"))
	return fss
}

func (s *ServerOptions) Validate() []error {
	var errors []error
	errors = append(errors, s.GenericServerRunOptions.Validate()...)
	errors = append(errors, s.EtcdOptions.Validate()...)
	errors = append(errors, s.MQOptions.Validate()...)
	errors = append(errors, s.LogOptions.Validate()...)
	errors = append(errors, s.AuthenticationOptions.Validate()...)
	errors = append(errors, s.AuditOptions.Validate()...)
	return errors
}

func (s *ServerOptions) NewAPIServer(stopCh <-chan struct{}) (*server.APIServer, error) {
	apiServer := &server.APIServer{
		Config: s.Config,
	}

	httpSrv := &http.Server{
		Addr: fmt.Sprintf("%s:%d", s.GenericServerRunOptions.BindAddress, s.GenericServerRunOptions.InsecurePort),
	}
	if s.GenericServerRunOptions.SecurePort != 0 {
		pool := x509.NewCertPool()
		caCertPath := s.Config.GenericServerRunOptions.CACertFile

		caCrt, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, errors.WithMessagef(err, "read ca file %s failed", caCertPath)
		}
		pool.AppendCertsFromPEM(caCrt)
		certificate, err := tls.LoadX509KeyPair(s.GenericServerRunOptions.TLSCertFile, s.GenericServerRunOptions.TLSPrivateKey)
		if err != nil {
			return nil, err
		}

		if httpSrv.TLSConfig == nil {
			httpSrv.TLSConfig = new(tls.Config)
		}
		httpSrv.TLSConfig.ClientCAs = pool
		httpSrv.TLSConfig.ClientAuth = tls.VerifyClientCertIfGiven
		httpSrv.TLSConfig.Certificates = []tls.Certificate{certificate}
		httpSrv.Addr = fmt.Sprintf("%s:%d", s.GenericServerRunOptions.BindAddress, s.GenericServerRunOptions.SecurePort)
	}

	apiServer.RESTOptionsGetter = s.CompleteEtcdOptions()

	apiServer.Server = httpSrv
	return apiServer, nil
}

func (s *ServerOptions) CompleteEtcdOptions() *etcdRESTOptions.StorageFactoryRestOptionsFactory {
	// grpclog.SetLoggerV2(grpclog.NewLoggerV2(ioutil.Discard, ioutil.Discard, ioutil.Discard))
	gvks := []schema.GroupVersion{corev1.SchemeGroupVersion, iamv1.SchemeGroupVersion}
	c := storagebackend.NewDefaultConfig(s.EtcdOptions.Prefix, scheme.Codecs.CodecForVersions(scheme.Encoder, scheme.Codecs.UniversalDeserializer(), schema.GroupVersions(gvks), schema.GroupVersions(gvks)))
	c.Transport.ServerList = s.EtcdOptions.ServerList
	c.Transport.CertFile = s.EtcdOptions.CertFile
	c.Transport.KeyFile = s.EtcdOptions.KeyFile
	c.Transport.TrustedCAFile = s.EtcdOptions.TrustedCAFile
	c.Paging = s.EtcdOptions.Paging
	c.CompactionInterval = s.EtcdOptions.CompactionInterval
	c.CountMetricPollPeriod = s.EtcdOptions.CountMetricPollPeriod
	completeEtcdOptions := &etcdRESTOptions.EtcdOptions{
		StorageConfig:           *c,
		DefaultStorageMediaType: s.EtcdOptions.DefaultStorageMediaType,
		DeleteCollectionWorkers: s.EtcdOptions.DeleteCollectionWorkers,
		EnableGarbageCollection: s.EtcdOptions.EnableGarbageCollection,
		EnableWatchCache:        s.EtcdOptions.EnableWatchCache,
		DefaultWatchCacheSize:   s.EtcdOptions.DefaultWatchCacheSize,
		WatchCacheSizes:         s.EtcdOptions.WatchCacheSizes,
	}
	return &etcdRESTOptions.StorageFactoryRestOptionsFactory{
		Options:        *completeEtcdOptions,
		StorageFactory: &etcdRESTOptions.SimpleStorageFactory{StorageConfig: *c},
	}
}
