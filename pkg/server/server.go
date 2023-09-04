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

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	authx509 "github.com/kubeclipper/kubeclipper/pkg/authentication/request/x509"

	"github.com/emicklei/go-restful"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/sets"
	unionauth "k8s.io/apiserver/pkg/authentication/request/union"
	etcdRESTOptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/component-base/version"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	auditingv1 "github.com/kubeclipper/kubeclipper/pkg/apis/auditing/v1"
	configv1 "github.com/kubeclipper/kubeclipper/pkg/apis/config/v1"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/apis/core/v1"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/apis/iam/v1"
	"github.com/kubeclipper/kubeclipper/pkg/apis/oauth"
	"github.com/kubeclipper/kubeclipper/pkg/apis/proxy"
	"github.com/kubeclipper/kubeclipper/pkg/auditing"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/auth"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/mfa"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/request/anonymous"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/request/bearertoken"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/request/internaltoken"
	authnpath "github.com/kubeclipper/kubeclipper/pkg/authentication/request/path"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/request/wstoken"
	"github.com/kubeclipper/kubeclipper/pkg/authorization/authorizer"
	"github.com/kubeclipper/kubeclipper/pkg/authorization/rbac"
	"github.com/kubeclipper/kubeclipper/pkg/client/clientrest"
	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	"github.com/kubeclipper/kubeclipper/pkg/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller/backupcontroller"
	"github.com/kubeclipper/kubeclipper/pkg/controller/cloudprovidercontroller"
	"github.com/kubeclipper/kubeclipper/pkg/controller/clustercontroller"
	"github.com/kubeclipper/kubeclipper/pkg/controller/cronbackupcontroller"
	"github.com/kubeclipper/kubeclipper/pkg/controller/dnscontroller"
	"github.com/kubeclipper/kubeclipper/pkg/controller/operationcontroller"
	"github.com/kubeclipper/kubeclipper/pkg/controller/regioncontroller"
	"github.com/kubeclipper/kubeclipper/pkg/controller/tokencontroller"
	"github.com/kubeclipper/kubeclipper/pkg/healthz"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/models/core"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
	"github.com/kubeclipper/kubeclipper/pkg/models/lease"
	"github.com/kubeclipper/kubeclipper/pkg/models/operation"
	"github.com/kubeclipper/kubeclipper/pkg/models/platform"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
	"github.com/kubeclipper/kubeclipper/pkg/server/config"
	"github.com/kubeclipper/kubeclipper/pkg/server/filters"
	"github.com/kubeclipper/kubeclipper/pkg/server/registry"
	"github.com/kubeclipper/kubeclipper/pkg/server/request"
	"github.com/kubeclipper/kubeclipper/pkg/service"
	"github.com/kubeclipper/kubeclipper/pkg/service/delivery"
	"github.com/kubeclipper/kubeclipper/pkg/service/staticresource"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/cache"
	"github.com/kubeclipper/kubeclipper/pkg/utils/hashutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/metrics"
)

type APIServer struct {
	Server                *http.Server
	container             *restful.Container
	Config                *config.Config
	Services              []service.Interface
	cache                 cache.Interface
	RESTOptionsGetter     *etcdRESTOptions.StorageFactoryRestOptionsFactory
	storageFactory        registry.SharedStorageFactory
	rbacAuthorizer        authorizer.Authorizer
	databaseAuditBackend  auditing.Backend
	internalInformerUser  string
	InternalInformerToken string
	terminationChan       chan struct{}
}

func (s *APIServer) PrepareRun(stopCh <-chan struct{}) error {
	s.internalInformerUser = "system:kc-server"
	s.InternalInformerToken = uuid.New().String()
	s.storageFactory = registry.NewSharedStorageFactory(s.RESTOptionsGetter)
	s.terminationChan = make(chan struct{})

	var err error
	switch s.Config.CacheOptions.CacheProvider {
	case cache.ProviderMemory:
		s.cache, err = cache.NewMemory()
	case cache.ProviderEtcd:
		iamOperator := iam.NewOperator(s.storageFactory.Users(), s.storageFactory.GlobalRoles(),
			s.storageFactory.GlobalRoleBindings(), s.storageFactory.Tokens(), s.storageFactory.LoginRecords())
		s.cache, err = cache.NewEtcd(iamOperator, iamOperator)
	case cache.ProviderRedis:
		s.cache, err = cache.NewRedis(s.Config.CacheOptions.RedisOptions)
	default:
		return fmt.Errorf("not support cache provider:%s", err)
	}
	if err != nil {
		return err
	}
	if err := mfa.SetupWithOptions(s.cache, s.Config.AuthenticationOptions.MFAOptions); err != nil {
		return err
	}

	s.container = restful.NewContainer()
	s.container.DoNotRecover(false)
	s.container.Filter(filters.LogRequestAndResponse)
	s.container.Router(restful.CurlyRouter{})
	s.container.RecoverHandler(func(panicReason interface{}, httpWriter http.ResponseWriter) {
		filters.LogStackOnRecover(panicReason, httpWriter)
	})
	if err := s.installAPIs(stopCh); err != nil {
		return err
	}
	healthz.InstallRootHealthz(s.container)
	s.installMetricsAPI()
	s.installVersionAPI()
	s.container.Filter(monitorRequest)

	if err := s.buildHandlerChain(stopCh); err != nil {
		return err
	}

	s.Server.Handler = s.container

	for _, svc := range s.Services {
		if err := svc.PrepareRun(stopCh); err != nil {
			return err
		}
	}

	return s.migrate()
}

func (s *APIServer) Run(stopCh <-chan struct{}) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-stopCh
		_ = s.Server.Shutdown(ctx)
	}()

	for _, svc := range s.Services {
		if err := svc.Run(stopCh); err != nil {
			return err
		}
	}

	logger.Info("Server start", zap.String("addr", s.Server.Addr), zap.String("version", version.Get().String()))

	if s.Server.TLSConfig != nil {
		err = s.Server.ListenAndServeTLS("", "")
	} else {
		err = s.Server.ListenAndServe()
	}

	return err
}

func (s *APIServer) buildHandlerChain(stopCh <-chan struct{}) error {
	infoFactory := &request.InfoFactory{
		APIPrefixes: sets.New("api", "cluster"),
	}
	s.container.Filter(filters.WithRequestInfo(infoFactory))

	iamOperator := iam.NewOperator(s.storageFactory.Users(), s.storageFactory.GlobalRoles(),
		s.storageFactory.GlobalRoleBindings(), s.storageFactory.Tokens(), s.storageFactory.LoginRecords())
	tokenOperator := auth.NewTokenOperator(iamOperator, s.Config.AuthenticationOptions)

	authnPathAuthenticator, err := authnpath.NewAuthenticator([]string{"/oauth/login", "/version", "/metrics", "/healthz"})
	if err != nil {
		return err
	}

	opts, err := authx509.NewStaticVerifierFromFile(s.Config.GenericServerRunOptions.CACertFile)
	if err != nil {
		return err
	}
	x509auth := authx509.NewDynamic(opts, authx509.CommonNameUserConversion)

	tokenAuthn := auth.NewTokenAuthenticator(iamOperator, tokenOperator)

	s.container.Filter(filters.WithAuthentication(
		unionauth.New(
			authnPathAuthenticator,
			internaltoken.New(s.internalInformerUser, s.InternalInformerToken),
			x509auth,
			anonymous.NewAuthenticator(),
			bearertoken.New(tokenAuthn),
			wstoken.New(tokenAuthn)),
	))

	s.container.Filter(filters.WithAuthorization(s.rbacAuthorizer))

	a := auditing.NewAuditing(s.Config.AuditOptions)
	a.AddBackend(auditing.ConsoleBackend{})
	if s.databaseAuditBackend != nil {
		a.AddBackend(s.databaseAuditBackend)
	}
	s.container.Filter(filters.WithAudit(a))
	return nil
}

func (s *APIServer) installMetricsAPI() {
	registerMetrics()
	metrics.Defaults.Install(s.container)
}

func (s *APIServer) installVersionAPI() {
	s.container.HandleWithFilter("/version", http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		v := version.Get()
		vByte, err := json.Marshal(v)
		if err != nil {
			http.Error(writer, fmt.Sprintf("marshal version failed due to %s", err.Error()), 500)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(vByte)
	}))
}

func monitorRequest(r *restful.Request, response *restful.Response, chain *restful.FilterChain) {
	// start := time.Now()
	chain.ProcessFilter(r, response)
	// TODO: Add metrics
}

func (s *APIServer) installAPIs(stopCh <-chan struct{}) error {
	clusterOperator := cluster.NewClusterOperator(s.storageFactory.Clusters(),
		s.storageFactory.Nodes(),
		s.storageFactory.Regions(),
		s.storageFactory.Backups(),
		s.storageFactory.Recoveries(),
		s.storageFactory.BackupPoints(),
		s.storageFactory.CronBackups(),
		s.storageFactory.DNSDomains(),
		s.storageFactory.Template(),
		s.storageFactory.CloudProvider(),
		s.storageFactory.Registry(),
	)
	coreOperator := core.NewOperator(s.storageFactory.ConfigMaps())
	leaseOperator := lease.NewLeaseOperator(s.storageFactory.Leases())
	opOperator := operation.NewOperationOperator(s.storageFactory.Operations())
	iamOperator := iam.NewOperator(s.storageFactory.Users(), s.storageFactory.GlobalRoles(),
		s.storageFactory.GlobalRoleBindings(), s.storageFactory.Tokens(), s.storageFactory.LoginRecords())
	s.rbacAuthorizer = rbac.NewAuthorizer(iamOperator, clusterOperator)

	deliverySvc := delivery.NewService(s.Config.MQOptions, clusterOperator, leaseOperator, opOperator, &s.terminationChan)
	s.Services = append(s.Services, deliverySvc)

	platformOperator := platform.NewPlatformOperator(s.storageFactory.PlatformSettings(), s.storageFactory.Events())
	if err := configv1.AddToContainer(s.container, platformOperator, s.Config); err != nil {
		return err
	}

	s.databaseAuditBackend = auditing.NewDatabaseBackend(platformOperator, stopCh)

	tokenOperator := auth.NewTokenOperator(iamOperator, s.Config.AuthenticationOptions)

	if err := iamv1.AddToContainer(s.container, iamOperator, s.rbacAuthorizer, tokenOperator); err != nil {
		return err
	}

	if err := auditingv1.AddToContainer(s.container, platformOperator); err != nil {
		return err
	}

	if err := oauth.AddToContainer(s.container, iamOperator, tokenOperator,
		auth.NewPasswordAuthenticator(iamOperator, s.Config.AuthenticationOptions),
		auth.NewOauthAuthenticator(iamOperator, s.Config.AuthenticationOptions),
		auth.NewMFAAuthenticator(iamOperator, s.cache, s.Config.AuthenticationOptions.MFAOptions),
		s.Config.AuthenticationOptions, s.cache); err != nil {
		return err
	}
	scheme := "http"
	host := s.Config.GenericServerRunOptions.BindAddress
	if host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	port := s.Config.GenericServerRunOptions.InsecurePort
	if s.Config.GenericServerRunOptions.SecurePort != 0 {
		scheme = "https"
		port = s.Config.GenericServerRunOptions.SecurePort
	}
	addr := fmt.Sprintf("%s://%s:%d", scheme, host, port)

	rc := clientrest.InternalRestConfig(addr, s.internalInformerUser, s.InternalInformerToken)
	if scheme == "https" {
		rc.TLSClientConfig.ServerName = options.KCServerAltName
		rc.TLSClientConfig.CAFile = s.Config.GenericServerRunOptions.CACertFile
		if rc.TLSClientConfig.CAFile == "" {
			// self-signed certificate
			rc.TLSClientConfig.CAFile = s.Config.GenericServerRunOptions.TLSCertFile
		}
	}

	ctrl, err := manager.NewControllerManager(rc, s.storageFactory, deliverySvc, &s.terminationChan, s.SetupController)
	if err != nil {
		return err
	}
	s.Services = append(s.Services, ctrl)

	if err = corev1.AddToContainer(s.container, clusterOperator, opOperator, platformOperator,
		leaseOperator, coreOperator, deliverySvc, tokenOperator, s.Config.GenericServerRunOptions, &s.terminationChan); err != nil {
		return err
	}
	if err = proxy.AddToContainer(s.container, clusterOperator); err != nil {
		return err
	}
	staticResourceSvc, err := staticresource.NewService(s.Config.StaticServerOptions)
	if err != nil {
		return err
	}
	s.Services = append(s.Services, staticResourceSvc)
	return nil
}

func (s *APIServer) migrate() error {
	operator := iam.NewOperator(s.storageFactory.Users(), s.storageFactory.GlobalRoles(),
		s.storageFactory.GlobalRoleBindings(), s.storageFactory.Tokens(), s.storageFactory.LoginRecords())
	if err := s.migrateRole(operator); err != nil {
		return err
	}
	if err := s.migrateRoleBinding(operator); err != nil {
		return err
	}
	return s.migrateUser(operator)
}

func (s *APIServer) migrateRole(operator iam.Operator) error {
	result, err := operator.ListRoles(context.TODO(), query.New())
	if err != nil {
		return err
	}
	roles := Roles.Diff(result)

	for _, val := range roles {
		if _, err := operator.CreateRole(context.TODO(), &val); err != nil {
			return err
		}
	}
	return nil
}

func (s *APIServer) migrateRoleBinding(operator iam.Operator) error {
	result, err := operator.ListRoleBindings(context.TODO(), query.New())
	if err != nil {
		return err
	}
	roleBindings := RoleBindings.Diff(result)

	for _, val := range roleBindings {
		if _, err := operator.CreateRoleBinding(context.TODO(), &val); err != nil {
			return err
		}
	}
	return nil
}

func (s *APIServer) migrateUser(operator iam.Operator) error {
	result, err := operator.ListUsers(context.TODO(), query.New())
	if err != nil {
		return err
	}

	userList := GetInternalUser(s.Config.AuthenticationOptions.InitialPassword)
	users := userList.Diff(result)

	for _, val := range users {
		encPass, err := hashutil.EncryptPassword(val.Spec.EncryptedPassword)
		if err != nil {
			return err
		}
		val.Spec.EncryptedPassword = encPass
		state := v1.UserActive
		val.Status.State = &state
		if _, err := operator.CreateUser(context.TODO(), &val); err != nil {
			return err
		}
	}
	return nil
}

func (s *APIServer) SetupController(mgr manager.Manager, informerFactory informers.SharedInformerFactory, storageFactory registry.SharedStorageFactory) error {
	var err error
	clusterOperator := cluster.NewClusterOperator(storageFactory.Clusters(),
		storageFactory.Nodes(),
		storageFactory.Regions(),
		storageFactory.Backups(),
		storageFactory.Recoveries(),
		storageFactory.BackupPoints(),
		storageFactory.CronBackups(),
		storageFactory.DNSDomains(),
		storageFactory.Template(),
		storageFactory.CloudProvider(),
		storageFactory.Registry(),
	)
	coreOperator := core.NewOperator(storageFactory.ConfigMaps())
	opOperator := operation.NewOperationOperator(storageFactory.Operations())
	iamOperator := iam.NewOperator(storageFactory.Users(), storageFactory.GlobalRoles(), storageFactory.GlobalRoleBindings(),
		storageFactory.Tokens(), storageFactory.LoginRecords())
	if err = (&regioncontroller.RegionReconciler{
		NodeLister:   informerFactory.Core().V1().Nodes().Lister(),
		RegionLister: informerFactory.Core().V1().Regions().Lister(),
		RegionWriter: clusterOperator,
	}).SetupWithManager(mgr, informerFactory); err != nil {
		return err
	}
	if err = (&backupcontroller.BackupReconciler{
		ClusterLister:   informerFactory.Core().V1().Clusters().Lister(),
		BackupLister:    informerFactory.Core().V1().Backups().Lister(),
		OperationLister: informerFactory.Core().V1().Operations().Lister(),
		BackupWriter:    clusterOperator,
	}).SetupWithManager(mgr, informerFactory); err != nil {
		return err
	}
	if err = (&clustercontroller.ClusterReconciler{
		CmdDelivery:         mgr.GetCmdDelivery(),
		ClusterLister:       informerFactory.Core().V1().Clusters().Lister(),
		ClusterWriter:       clusterOperator,
		ClusterOperator:     clusterOperator,
		RegistryLister:      informerFactory.Core().V1().Registries().Lister(),
		NodeLister:          informerFactory.Core().V1().Nodes().Lister(),
		NodeWriter:          clusterOperator,
		OperationOperator:   opOperator,
		OperationWriter:     opOperator,
		CronBackupWriter:    clusterOperator,
		CloudProviderLister: informerFactory.Core().V1().CloudProviders().Lister(),
	}).SetupWithManager(mgr, informerFactory); err != nil {
		return err
	}
	if err = (&cronbackupcontroller.CronBackupReconciler{
		CmdDelivery:       mgr.GetCmdDelivery(),
		ClusterLister:     informerFactory.Core().V1().Clusters().Lister(),
		NodeLister:        informerFactory.Core().V1().Nodes().Lister(),
		BackupLister:      informerFactory.Core().V1().Backups().Lister(),
		CronBackupLister:  informerFactory.Core().V1().CronBackups().Lister(),
		BackupPointLister: informerFactory.Core().V1().BackupPoints().Lister(),
		OperationWriter:   opOperator,
		ClusterWriter:     clusterOperator,
		CronBackupWriter:  clusterOperator,
		BackupWriter:      clusterOperator,
		Now:               time.Now,
	}).SetupWithManager(mgr, informerFactory); err != nil {
		return err
	}
	if err = (&dnscontroller.DNSReconciler{
		DomainLister:  informerFactory.Core().V1().Domains().Lister(),
		DomainWriter:  clusterOperator,
		ClusterLister: informerFactory.Core().V1().Clusters().Lister(),
	}).SetupWithManager(mgr, informerFactory); err != nil {
		return err
	}
	if err = (&operationcontroller.OperationReconciler{
		CmdDelivery:       mgr.GetCmdDelivery(),
		ClusterLister:     informerFactory.Core().V1().Clusters().Lister(),
		ClusterOperator:   clusterOperator,
		OperationLister:   informerFactory.Core().V1().Operations().Lister(),
		OperationWriter:   opOperator,
		OperationOperator: opOperator,
	}).SetupWithManager(mgr, informerFactory); err != nil {
		return err
	}
	if err = (&tokencontroller.TokenReconciler{
		TokenLister: informerFactory.Iam().V1().Tokens().Lister(),
		TokenWriter: iamOperator,
	}).SetupWithManager(mgr, informerFactory); err != nil {
		return err
	}
	if err = (&cloudprovidercontroller.CloudProviderReconciler{
		ClusterLister:       informerFactory.Core().V1().Clusters().Lister(),
		ClusterWriter:       clusterOperator,
		CloudProviderLister: informerFactory.Core().V1().CloudProviders().Lister(),
		CloudProviderWriter: clusterOperator,
		NodeLister:          informerFactory.Core().V1().Nodes().Lister(),
		NodeWriter:          clusterOperator,
		ConfigmapLister:     informerFactory.Core().V1().ConfigMaps().Lister(),
		ConfigmapWriter:     coreOperator,
	}).SetupWithManager(mgr, informerFactory); err != nil {
		return err
	}
	(&controller.ClusterStatusMon{
		ClusterWriter:       clusterOperator,
		ClusterLister:       informerFactory.Core().V1().Clusters().Lister(),
		NodeLister:          informerFactory.Core().V1().Nodes().Lister(),
		CmdDelivery:         mgr.GetCmdDelivery(),
		CloudProviderLister: informerFactory.Core().V1().CloudProviders().Lister(),
	}).SetupWithManager(mgr)
	(&controller.NodeStatusMon{
		NodeLister:  informerFactory.Core().V1().Nodes().Lister(),
		LeaseLister: informerFactory.Core().V1().Leases().Lister(),
		NodeWriter:  clusterOperator,
	}).SetupWithManager(mgr)
	(&controller.AuditStatusMon{
		AuditOperator: platform.NewPlatformOperator(storageFactory.Operations(), storageFactory.Events()),
		AuditOptions:  s.Config.AuditOptions,
	}).SetupWithManager(mgr)
	(&controller.IAMStatusMon{
		IAMOperator: iam.NewOperator(storageFactory.Users(), storageFactory.GlobalRoles(),
			storageFactory.GlobalRoleBindings(), storageFactory.Tokens(), storageFactory.LoginRecords()),
		AuthenticationOpts: s.Config.AuthenticationOptions,
	}).SetupWithManager(mgr)
	return nil
}
