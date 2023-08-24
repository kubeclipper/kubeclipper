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

package v1

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/utils/httputil"

	"k8s.io/client-go/util/retry"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeclipper/kubeclipper/pkg/controller/cronbackupcontroller"

	"github.com/emicklei/go-restful"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	apimachineryErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	r "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/auth"
	"github.com/kubeclipper/kubeclipper/pkg/client/clientrest"
	"github.com/kubeclipper/kubeclipper/pkg/clustermanage"
	"github.com/kubeclipper/kubeclipper/pkg/clusteroperation"
	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/client"
	"github.com/kubeclipper/kubeclipper/pkg/controller/cloudprovidercontroller"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/models/core"
	"github.com/kubeclipper/kubeclipper/pkg/models/lease"
	"github.com/kubeclipper/kubeclipper/pkg/models/operation"
	"github.com/kubeclipper/kubeclipper/pkg/models/platform"
	"github.com/kubeclipper/kubeclipper/pkg/oplog"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/k8s"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/validation"
	apirequest "github.com/kubeclipper/kubeclipper/pkg/server/request"
	"github.com/kubeclipper/kubeclipper/pkg/server/restplus"
	"github.com/kubeclipper/kubeclipper/pkg/service"
	bs "github.com/kubeclipper/kubeclipper/pkg/simple/backupstore"
	"github.com/kubeclipper/kubeclipper/pkg/simple/generic"
	"github.com/kubeclipper/kubeclipper/pkg/utils/certs"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
)

var (
	allowedDeleteStatus = sets.NewString(string(v1.ClusterInstallFailed), string(v1.ClusterRunning), string(v1.ClusterUpgradeFailed),
		string(v1.ClusterRestoreFailed), string(v1.ClusterTerminateFailed), string(v1.ClusterUpdateFailed))
)

type handler struct {
	genericConfig    *generic.ServerRunOptions
	clusterOperator  cluster.Operator
	leaseOperator    lease.Operator
	opOperator       operation.Operator
	platformOperator platform.Operator
	coreOperator     core.Operator
	delivery         service.IDelivery
	tokenOperator    auth.TokenManagementInterface
	terminationChan  *chan struct{}
}

const (
	ParameterMsg               = "msg"
	ParameterToken             = "token"
	ParameterCols              = "cols"
	ParameterRows              = "rows"
	resourceExistCheckerHeader = "X-CHECK-EXIST"
)

var (
	ErrNodesRegionDifferent = errors.New("nodes belongs to different region")
)

func newHandler(conf *generic.ServerRunOptions, clusterOperator cluster.Operator, op operation.Operator, leaseOperator lease.Operator,
	platform platform.Operator, coreOperator core.Operator, delivery service.IDelivery,
	tokenOperator auth.TokenManagementInterface, terminationChan *chan struct{}) *handler {
	return &handler{
		genericConfig:    conf,
		clusterOperator:  clusterOperator,
		delivery:         delivery,
		opOperator:       op,
		platformOperator: platform,
		leaseOperator:    leaseOperator,
		coreOperator:     coreOperator,
		tokenOperator:    tokenOperator,
		terminationChan:  terminationChan,
	}
}

func (h *handler) ListClusters(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	if q.Watch {
		h.watchCluster(request, response, q)
		return
	}

	if clientrest.IsInformerRawQuery(request.Request) {
		result, err := h.clusterOperator.ListClusters(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	} else {
		result, err := h.clusterOperator.ListClusterEx(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	}
}

func (h *handler) DescribeCluster(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	c, err := h.clusterOperator.GetClusterEx(request.Request.Context(), name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) AddOrRemoveNodes(request *restful.Request, response *restful.Response) {
	pn := &clusteroperation.PatchNodes{}
	if err := request.ReadEntity(pn); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	// cluster name in path
	clu := request.PathParameter("name")
	timeoutSecs := v1.DefaultOperationTimeoutSecs
	if v := request.QueryParameter("timeout"); v != "" {
		timeoutSecs = v
	}

	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	ctx := request.Request.Context()
	c, err := h.clusterOperator.GetClusterEx(ctx, clu, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	// backing up old masters and workers
	nodeSet := c.GetAllNodes()

	operationType := v1.OperationAddNodes
	if pn.Operation == clusteroperation.NodesOperationRemove {
		operationType = v1.OperationRemoveNodes
	}
	executable, err := clusteroperation.Executable(ctx, operationType, clu, h.opOperator)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	if !executable {
		restplus.HandleBadRequest(response, request, fmt.Errorf("%s does not support concurrent execution", operationType))
		return
	}

	if err = pn.MakeCompare(c); err != nil {
		if errors.Is(err, clusteroperation.ErrInvalidNodesOperation) || errors.Is(err, clusteroperation.ErrInvalidNodesRole) {
			restplus.HandleBadRequest(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	// Get workers node information must be called before pn.MakeCompare, in order to ensure pn.Nodes = extraMeta.Workers.
	// Replace cluster worker nodes with nodes to be operated.
	nodes, err := h.getNodeInfo(ctx, pn.Nodes, false)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	pn.ConvertNodes = nodes

	if len(nodes) == 0 {
		err = fmt.Errorf("nodes is already in use")
		if pn.Operation == clusteroperation.NodesOperationRemove {
			err = fmt.Errorf("nodes is already removed")
		}
		restplus.HandleBadRequest(response, request, err)
		return
	}

	for _, n := range nodes {
		switch pn.Operation {
		case clusteroperation.NodesOperationAdd:
			if n.Disable {
				restplus.HandleBadRequest(response, request, fmt.Errorf("this node(%s) is disabled", n.IPv4))
				return
			}
			if nodeSet.Has(n.ID) {
				restplus.HandleBadRequest(response, request, fmt.Errorf("this node(%s) is already in use", n.IPv4))
				return
			}
			if n.Region != c.Labels[common.LabelTopologyRegion] {
				restplus.HandleBadRequest(response, request, fmt.Errorf("the node(%s) belongs to different region", n.IPv4))
				return
			}
		case clusteroperation.NodesOperationRemove:
			if !nodeSet.Has(n.ID) {
				restplus.HandleBadRequest(response, request, fmt.Errorf("the node(%s) is not part of this cluster and cannot be removed", n.IPv4))
				return
			}
		}
	}

	if !dryRun {
		for _, node := range pn.Nodes {
			_, err = MarkToOriginNode(ctx, h.clusterOperator, node.ID)
			if err != nil {
				restplus.HandleInternalError(response, request, err)
				return
			}
		}
		pendingOperation, err := buildPendingOperation(operationType, buildOperationSponsor(h.genericConfig), timeoutSecs, c.ResourceVersion, pn)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}

		c.Status.Phase = v1.ClusterUpdating
		c.PendingOperations = append(c.PendingOperations, pendingOperation)
		if c, err = h.clusterOperator.UpdateCluster(ctx, c); err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}

	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) watchCluster(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.clusterOperator.WatchClusters(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("Cluster"), req, resp, timeout)
}

func (h *handler) DeleteCluster(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	var timeoutSecs string
	if v := request.QueryParameter("timeout"); v != "" {
		timeoutSecs = v
	} else {
		timeoutSecs = v1.DefaultOperationTimeoutSecs
	}
	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	force := query.GetBoolValueWithDefault(request, query.ParameterForce, false)
	c, err := h.clusterOperator.GetClusterEx(request.Request.Context(), name, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	if _, ok := c.Labels[common.LabelClusterProviderName]; ok {
		restplus.HandleBadRequest(response, request, errors.New("can't delete cluster which belongs to provider"))
		return
	}
	if !allowedDeleteStatus.Has(string(c.Status.Phase)) {
		restplus.HandleBadRequest(response, request, fmt.Errorf("can't delete cluster when cluster is %s", c.Status.Phase))
		return
	}

	q := query.New()
	q.LabelSelector = fmt.Sprintf("%s=%s", common.LabelClusterName, c.Name)
	backups, err := h.clusterOperator.ListBackupEx(request.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	if backups.TotalCount > 0 {
		restplus.HandleBadRequest(response, request, fmt.Errorf("before deleting the cluster, please delete the cluster backup file first"))
		return
	}

	extraMeta, err := h.getClusterMetadata(request.Request.Context(), c, force)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) || err == ErrNodesRegionDifferent {
			restplus.HandleBadRequest(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	if force && len(extraMeta.Masters) == 0 {
		logger.Warn("force delete cluster when master node not found, delete cluster directly", zap.String("cluster", name))
		if !dryRun {
			err = h.clusterOperator.DeleteCluster(request.Request.Context(), name)
			if err != nil {
				restplus.HandleInternalError(response, request, err)
				return
			}
		}
		response.WriteHeader(http.StatusOK)
		return
	}

	extraMeta.OperationType = v1.OperationDeleteCluster
	op, err := h.parseOperationFromCluster(extraMeta, c, v1.ActionUninstall)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	op.Status.Status = v1.OperationStatusRunning
	op.Labels[common.LabelTimeoutSeconds] = timeoutSecs
	op.Labels[common.LabelOperationAction] = v1.OperationDeleteCluster
	op.Labels[common.LabelOperationSponsor] = buildOperationSponsor(h.genericConfig)
	if !dryRun {
		c.Status.Phase = v1.ClusterTerminating
		_, err = h.clusterOperator.UpdateCluster(request.Request.Context(), c)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		op, err = h.opOperator.CreateOperation(request.Request.Context(), op)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}
	go func(o *v1.Operation, opts *service.Options) {
		if err := h.delivery.DeliverTaskOperation(context.TODO(), o, opts); err != nil {
			logger.Error("delivery task error", zap.Error(err))
		}
	}(op, &service.Options{DryRun: dryRun, ForceSkipError: force})
	response.WriteHeader(http.StatusOK)
}

func (h *handler) CreateClusters(request *restful.Request, response *restful.Response) {
	c := v1.Cluster{}
	if err := request.ReadEntity(&c); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	if c.Labels[common.LabelBackupPoint] != "" {
		_, err := h.clusterOperator.GetBackupPointEx(request.Request.Context(), c.Labels[common.LabelBackupPoint], "0")
		if err != nil {
			if apimachineryErrors.IsNotFound(err) {
				restplus.HandleBadRequest(response, request, err)
				return
			}
			restplus.HandleInternalError(response, request, err)
			return
		}
	}

	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	timeoutSecs := v1.DefaultOperationTimeoutSecs
	if v := request.QueryParameter("timeout"); v != "" {
		timeoutSecs = v
	}
	c.Complete()
	// validate node exist
	extraMeta, err := h.getClusterMetadata(request.Request.Context(), &c, false)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) || err == ErrNodesRegionDifferent {
			restplus.HandleBadRequest(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	if err := h.createClusterCheck(request.Request.Context(), &c); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	// TODO: This logic has been implemented in the clusterController
	c.Status.Registries, err = h.getClusterCRIRegistries(request.Request.Context(), &c)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	extraMeta.OperationType = v1.OperationCreateCluster
	op, err := h.parseOperationFromCluster(extraMeta, &c, v1.ActionInstall)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	// TODO: make dry run path to etcd
	if !dryRun {
		c.Status.Phase = v1.ClusterInstalling
		_, err = h.clusterOperator.CreateCluster(context.TODO(), &c)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}

	op.Labels[common.LabelTimeoutSeconds] = timeoutSecs
	op.Labels[common.LabelOperationAction] = v1.OperationCreateCluster
	op.Labels[common.LabelOperationSponsor] = buildOperationSponsor(h.genericConfig)
	op.Status.Status = v1.OperationStatusRunning
	if !dryRun {
		op, err = h.opOperator.CreateOperation(context.TODO(), op)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}

	go h.doOperation(context.TODO(), op, &service.Options{DryRun: dryRun})
	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) UpdateClusters(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	c := v1.Cluster{}
	if err := request.ReadEntity(&c); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	// TODO: vlidate update-cluster struct
	if ip, ok := c.Labels[common.LabelExternalIP]; ok {
		if !netutil.IsValidIP(ip) {
			restplus.HandleBadRequest(response, request, fmt.Errorf("external IP %s is valid", ip))
			return
		}
	}

	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	if !dryRun {
		clu, err := h.clusterOperator.GetCluster(context.TODO(), name)
		if err != nil {
			if apimachineryErrors.IsNotFound(err) {
				restplus.HandleNotFound(response, request, err)
				return
			}
			restplus.HandleInternalError(response, request, err)
			return
		}

		// update fields
		clu.Labels = c.Labels
		clu.Annotations = c.Annotations
		clu.ContainerRuntime.Registries = c.ContainerRuntime.Registries
		_, err = h.clusterOperator.UpdateCluster(context.TODO(), clu)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}

	response.WriteHeader(http.StatusOK)
}

func (h *handler) UpdateClusterCertification(request *restful.Request, response *restful.Response) {
	cluName := request.PathParameter(query.ParameterName)
	ctx := request.Request.Context()
	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	c, err := h.clusterOperator.GetCluster(ctx, cluName)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	extraMeta, err := h.getClusterMetadata(ctx, c, false)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	op, err := h.parseUpdateCertOperation(c, extraMeta)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	op.Name = uuid.New().String()
	op.Labels = map[string]string{
		common.LabelClusterName:      c.Name,
		common.LabelTimeoutSeconds:   v1.DefaultOperationTimeoutSecs,
		common.LabelOperationAction:  v1.OperationUpdateCertification,
		common.LabelOperationSponsor: buildOperationSponsor(h.genericConfig),
	}
	op.Status.Status = v1.OperationStatusRunning
	c.Status.Phase = v1.ClusterUpdating
	if !dryRun {
		op, err = h.opOperator.CreateOperation(ctx, op)
		if err != nil {
			restplus.HandleBadRequest(response, request, err)
			return
		}
		_, err = h.clusterOperator.UpdateCluster(ctx, c)
		if err != nil {
			restplus.HandleBadRequest(response, request, err)
			return
		}
	}

	go h.doOperation(context.TODO(), op, &service.Options{DryRun: dryRun})
	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) GetKubeConfig(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	proxyMode := strings.ToLower(request.QueryParameter("proxy")) == "true"

	ctx := request.Request.Context()
	clu, err := h.clusterOperator.GetCluster(ctx, name)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	var (
		kubeConfigData []byte
	)
	if proxyMode {
		kubeConfigData, err = h.getProxyKubeConfig(ctx, name)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	} else {
		extraMeta, err := h.getClusterMetadata(ctx, clu, false)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}

		if clu.Annotations != nil && clu.Annotations[common.AnnotationActualName] != "" {
			extraMeta.ClusterName = clu.Annotations[common.AnnotationActualName]
		}

		masters, err := extraMeta.Masters.AvailableKubeMasters()
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		var externalAddress string
		if clu.Labels != nil {
			externalAddress = clu.Labels[common.LabelExternalIP]
		}

		kubeConfig, err := k8s.GetKubeConfig(context.TODO(), extraMeta.ClusterName, masters[0], externalAddress, h.delivery)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		kubeConfigData = []byte(kubeConfig)
	}

	_, _ = response.Write(kubeConfigData)
}

func (h *handler) getProxyKubeConfig(ctx context.Context, clusterName string) ([]byte, error) {
	// TODO: get CA from kc server config
	var serverCA []byte

	scheme := "http"
	port := h.genericConfig.InsecurePort
	if h.genericConfig.SecurePort != 0 {
		scheme = "https"
		port = h.genericConfig.SecurePort
	}
	// FIXME floatIP or domain?
	host := fmt.Sprintf("%s:%d", h.genericConfig.BindAddress, port)

	serverURL := fmt.Sprintf("%s://%s/cluster/%s", scheme, host, clusterName)
	user, ok := apirequest.UserFrom(ctx)
	if !ok {
		return nil, fmt.Errorf("not get user info")
	}

	token, err := h.tokenOperator.IssueTo(user)
	if err != nil {
		return nil, fmt.Errorf("issue token:%w", err)
	}

	kubecfg := CreateWithToken(serverURL, clusterName, user.GetName(), serverCA, token.AccessToken)
	// skip all tls verify
	for _, c := range kubecfg.Clusters {
		if strings.HasPrefix(strings.ToLower(c.Server), "https") {
			c.InsecureSkipTLSVerify = true
			c.TLSServerName = options.KCServerAltName
		}
	}

	buf, err := clientcmd.Write(*kubecfg)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (h *handler) ListNodes(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	if q.Watch {
		h.watchNodes(request, response, q)
		return
	}

	if clientrest.IsInformerRawQuery(request.Request) {
		result, err := h.clusterOperator.ListNodes(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		// response.PrettyPrint(false)
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	} else {
		result, err := h.clusterOperator.ListNodesEx(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	}
}

func (h *handler) DescribeNode(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	c, err := h.clusterOperator.GetNodeEx(request.Request.Context(), name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) DisableNode(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	ctx := request.Request.Context()

	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	node, err := h.clusterOperator.GetNodeEx(ctx, name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	err = h.syncNodeDisable(node, true)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	updateNode, err := h.clusterOperator.UpdateNode(ctx, node)
	if err != nil {
		if apimachineryErrors.IsConflict(err) {
			restplus.HandleBadRequest(response, request, fmt.Errorf("the node %s has been modified; please apply "+
				"your changes to the latest version and try again", node.Name))
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, updateNode)
}

func (h *handler) EnableNode(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	ctx := request.Request.Context()
	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	node, err := h.clusterOperator.GetNodeEx(ctx, name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	err = h.syncNodeDisable(node, false)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	updateNode, err := h.clusterOperator.UpdateNode(ctx, node)
	if err != nil {
		if apimachineryErrors.IsConflict(err) {
			restplus.HandleBadRequest(response, request, fmt.Errorf("the node %s has been modified; please apply "+
				"your changes to the latest version and try again", node.Name))
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, updateNode)
}

func (h *handler) syncNodeDisable(node *v1.Node, reqDisable bool) error {
	_, nodeDisable := node.Labels[common.LabelNodeDisable]
	if reqDisable == nodeDisable {
		return fmt.Errorf("the node %s is already disable:%v", node.Name, nodeDisable)
	}
	// 1.only can disable/enable idle node.
	_, inCluster := node.Labels[common.LabelNodeRole]
	if inCluster {
		return fmt.Errorf("node %s are in used,cannot disable/enable", node.Name)
	}
	// 2. cannot enable unknown state node.
	if !reqDisable {
		_, condition := controller.GetNodeCondition(&node.Status, v1.NodeReady)
		if condition != nil && condition.Status == v1.ConditionUnknown {
			return fmt.Errorf("node %s state are %s,cannot enable", node.Name, v1.ConditionUnknown)
		}
	}
	// 3.update node label
	if reqDisable {
		node.Labels[common.LabelNodeDisable] = "true"
	} else {
		delete(node.Labels, common.LabelNodeDisable)
	}

	return nil
}

// DeleteNode delete node record from etcd,only called by kcctl now.
func (h *handler) DeleteNode(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	ctx := request.Request.Context()
	_, err := h.clusterOperator.GetNodeEx(ctx, name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	err = h.clusterOperator.DeleteNode(ctx, name)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	response.WriteHeader(http.StatusOK)
}

func (h *handler) watchNodes(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.clusterOperator.WatchNodes(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("Node"), req, resp, timeout)
}

func (h *handler) DescribeOperation(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	c, err := h.opOperator.GetOperationEx(request.Request.Context(), name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) ListOperations(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	if q.Watch {
		h.watchOperations(request, response, q)
		return
	}
	if clientrest.IsInformerRawQuery(request.Request) {
		result, err := h.opOperator.ListOperations(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	} else {
		result, err := h.opOperator.ListOperationsEx(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	}
}

func (h *handler) TerminationOperation(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()
	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	name := request.PathParameter(query.ParameterName)

	op, err := h.opOperator.GetOperationEx(ctx, name, "0")
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	if op.Status.Status != v1.OperationStatusRunning {
		restplus.HandleBadRequest(response, request, fmt.Errorf("the operation status does not support termination"))
		return
	}
	if !dryRun {
		// if the kc-server is not the initiator of the operation, the request is forwarded to the specified service
		if sponsor := op.Labels[common.LabelOperationSponsor]; sponsor != "" && !strings.Contains(sponsor, h.genericConfig.BindAddress) {
			headers := make(map[string]string)
			reqURL := fmt.Sprintf("%s%s", parseOperationSponsor(sponsor), request.Request.RequestURI)
			headers["Authorization"] = request.Request.Header.Get("Authorization")
			_, code, err := httputil.CommonRequest(reqURL, "POST", headers, nil, nil)
			if err != nil {
				restplus.HandleInternalError(response, request, err)
				return
			}
			if code != http.StatusOK {
				restplus.HandleInternalError(response, request, errors.New("forwarding request error"))
				return
			}
			_ = response.WriteHeaderAndEntity(http.StatusOK, nil)
			return
		}
		// update the data before broadcasting the message
		// write termination label
		op.Labels[common.LabelOperationIntent] = common.OperationIntent

		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, err = h.opOperator.UpdateOperation(ctx, op)
			return err
		}); err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		// close channel for broadcast termination message
		close(*h.terminationChan)
		// reset channel
		*h.terminationChan = make(chan struct{})
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, nil)
}

func (h *handler) watchOperations(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := query.MinTimeoutSeconds * time.Second
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	watcher, err := h.opOperator.WatchOperations(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("Operation"), req, resp, timeout)
}

// TODO: it will be deprecated in the future
func (h *handler) getClusterMetadata(ctx context.Context, c *v1.Cluster, skipNodeNotFound bool) (*component.ExtraMetadata, error) {
	meta := &component.ExtraMetadata{
		ClusterName:        c.Name,
		ClusterStatus:      c.Status.Phase,
		Offline:            c.Offline(),
		LocalRegistry:      c.LocalRegistry,
		CRI:                c.ContainerRuntime.Type,
		KubeVersion:        c.KubernetesVersion,
		KubeletDataDir:     c.Kubelet.RootDir,
		ControlPlaneStatus: c.Status.ControlPlaneHealth,
		CNI:                c.CNI.Type,
		CNINamespace:       c.CNI.Namespace,
	}

	if c.Annotations != nil {
		_, ok := c.Annotations[common.AnnotationOnlyInstallKubernetesComp]
		if ok {
			meta.OnlyInstallKubernetesComp = true
		}
	}

	meta.Addons = append(meta.Addons, c.Addons...)

	masters, err := h.getNodeInfo(ctx, c.Masters, skipNodeNotFound)
	if err != nil {
		return nil, err
	}
	meta.Masters = append(meta.Masters, masters...)
	workers, err := h.getNodeInfo(ctx, c.Workers, skipNodeNotFound)
	if err != nil {
		return nil, err
	}
	meta.Workers = append(meta.Workers, workers...)
	err = h.regionCheck(meta.Masters, meta.Workers)
	if err != nil {
		return nil, err
	}
	return meta, nil
}

func (h *handler) regionCheck(master, worker []component.Node) error {
	list := sets.NewString()
	for _, node := range master {
		list.Insert(node.Region)
	}
	for _, node := range worker {
		list.Insert(node.Region)
	}
	if list.Len() > 1 {
		return ErrNodesRegionDifferent
	}
	return nil
}

func (h *handler) getNodeInfo(ctx context.Context, nodes v1.WorkerNodeList, skipNodeNotFound bool) ([]component.Node, error) {
	var meta []component.Node
	for _, node := range nodes {
		n, err := h.clusterOperator.GetNodeEx(ctx, node.ID, "0")
		if err != nil {
			if skipNodeNotFound && apimachineryErrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		item := component.Node{
			ID:       n.Name,
			IPv4:     n.Status.Ipv4DefaultIP,
			NodeIPv4: n.Status.NodeIpv4DefaultIP,
			Region:   n.Labels[common.LabelTopologyRegion],
			Hostname: n.Labels[common.LabelHostname],
			Role:     n.Labels[common.LabelNodeRole],
		}
		_, item.Disable = n.Labels[common.LabelNodeDisable]
		meta = append(meta, item)
	}

	return meta, nil
}

func (h *handler) GetOperationLog(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()
	resourceVer := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	nodeName := request.QueryParameter(query.ParameterNode)
	if nodeName == "" {
		restplus.HandleBadRequest(response, request, errors.New("node name is required"))
		return
	}
	_, err := h.clusterOperator.GetNodeEx(ctx, nodeName, resourceVer)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	opID := request.QueryParameter(query.ParameterOperation)
	if opID == "" {
		restplus.HandleBadRequest(response, request, errors.New("operation ID is required"))
		return
	}
	op, err := h.opOperator.GetOperationEx(ctx, opID, resourceVer)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	stepID := request.QueryParameter(query.ParameterStep)
	if stepID == "" {
		restplus.HandleBadRequest(response, request, errors.New("step ID is required"))
		return
	}
	// validate operation contains the step
	step, ok := op.GetStep(stepID)
	if !ok {
		restplus.HandleBadRequest(response, request, errors.New("step ID is invalid"))
		return
	}
	offset, err := strconv.ParseInt(strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterOffset)), 10, 64)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	stepKey := fmt.Sprintf("%s-%s", stepID, step.Name)
	req, err := json.Marshal(oplog.LogContentRequest{
		OpID:   opID,
		StepID: stepKey,
		Offset: offset,
		Length: 0, // callers are not currently supported to set the length of the fetch data
	})
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	resp, err := h.delivery.DeliverLogRequest(ctx, &service.LogOperation{
		Op:                service.OperationStepLog,
		OperationIdentity: string(req),
		To:                nodeName,
	})
	if err != nil {
		logger.Error("request step log error", zap.Error(err))
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, StepLog{
		Content:      resp.Content,
		Node:         nodeName,
		Timeout:      step.Timeout,
		DeliverySize: resp.DeliverySize,
		LogSize:      resp.LogSize,
	})
}

func (h *handler) ListRegions(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	if q.Watch {
		h.watchRegions(request, response, q)
		return
	}
	if clientrest.IsInformerRawQuery(request.Request) {
		result, err := h.clusterOperator.ListRegions(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	} else {
		result, err := h.clusterOperator.ListRegionEx(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	}
}

func (h *handler) watchRegions(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.clusterOperator.WatchRegions(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("Region"), req, resp, timeout)
}

func (h *handler) DescribeRegion(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	c, err := h.clusterOperator.GetRegionEx(request.Request.Context(), name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

// doOperation should be called in goroutine.
func (h *handler) doOperation(ctx context.Context, op *v1.Operation, opts *service.Options) {
	if err := h.delivery.DeliverTaskOperation(ctx, op, opts); err != nil {
		logger.Error("distribute task error", zap.Error(err))
	}
}

func (h *handler) createClusterCheck(ctx context.Context, c *v1.Cluster) error {
	if c.Networking.IPFamily == v1.IPFamilyDualStack {
		if len(c.Networking.Pods.CIDRBlocks) < 2 {
			return fmt.Errorf("the cluster is enabled in dual-stack mode, requiring both ipv4 and ipv6")
		}
	}
	if len(c.Masters) == 0 {
		return fmt.Errorf("cluster must have one master node")
	}

	cluInfo, err := h.clusterOperator.GetClusterEx(ctx, c.Name, "0")
	if err != nil && !apimachineryErrors.IsNotFound(err) {
		return err
	}
	if cluInfo != nil {
		return fmt.Errorf("cluster %s already exists", c.Name)
	}

	// check if the node is already in use
	nodeList, err := h.clusterOperator.ListNodes(ctx, &query.Query{
		Pagination:           query.NoPagination(),
		ResourceVersion:      "0",
		Watch:                false,
		LabelSelector:        fmt.Sprintf("!%s,!%s", common.LabelNodeRole, common.LabelNodeDisable),
		ResourceVersionMatch: query.ResourceVersionMatchNotOlderThan,
	})
	if err != nil {
		return fmt.Errorf("failed to list node: %s", err.Error())
	}

	freeNodes := sets.NewString()
	for _, node := range nodeList.Items {
		freeNodes.Insert(node.Name)
	}
	cluNodes := sets.NewString()
	for _, node := range append(c.Masters, c.Workers...) {
		cluNodes.Insert(node.ID)
	}

	if freeNodes.HasAll(cluNodes.List()...) {
		return nil
	}

	return fmt.Errorf("some nodes in used or disabled")
}

func (h *handler) ListBackupsWithCluster(request *restful.Request, response *restful.Response) {
	// cluster name in path
	clusterName := request.PathParameter("name")
	ctx := request.Request.Context()
	cluster, err := h.clusterOperator.GetClusterEx(ctx, clusterName, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	q := query.ParseQueryParameter(request)
	labels := []string{fmt.Sprintf("%s=%s", common.LabelClusterName, cluster.Name)} // always select by cluster name
	q.LabelSelector = strings.Join(labels, ",")
	result, err := h.clusterOperator.ListBackupEx(ctx, q)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) ListBackups(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	if q.Watch {
		h.watchBackups(request, response, q)
		return
	}
	if clientrest.IsInformerRawQuery(request.Request) {
		result, err := h.clusterOperator.ListBackups(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	} else {
		result, err := h.clusterOperator.ListBackupEx(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	}
}

func (h *handler) DescribeBackup(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	c, err := h.clusterOperator.GetBackupEx(request.Request.Context(), name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) watchBackups(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.clusterOperator.WatchBackups(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("Backup"), req, resp, timeout)
}

func (h *handler) CreateBackup(request *restful.Request, response *restful.Response) {
	backup := &v1.Backup{}
	if err := request.ReadEntity(backup); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	// cluster name in path
	clusterName := request.PathParameter("name")
	ctx := request.Request.Context()
	c, err := h.clusterOperator.GetClusterEx(ctx, clusterName, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	if c.Status.Phase != v1.ClusterRunning {
		restplus.HandleInternalError(response, request, fmt.Errorf("cluster %s current is %s, can't back up",
			c.Name, c.Status.Phase))
		return
	}

	b, err := h.clusterOperator.GetBackup(ctx, clusterName, fmt.Sprintf("%s-%s", backup.Name, clusterName))
	if err != nil && !apimachineryErrors.IsNotFound(err) {
		restplus.HandleInternalError(response, request, err)
		return
	}
	if b != nil {
		restplus.HandleInternalError(response, request, fmt.Errorf("backup %s already exists", backup.Name))
		return
	}

	// check the backup point exist or not
	if c.Labels[common.LabelBackupPoint] == "" {
		restplus.HandleBadRequest(response, request, fmt.Errorf("no backup point specified"))
		return
	}

	_, err = h.clusterOperator.GetBackupPoint(ctx, c.Labels[common.LabelBackupPoint], "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleBadRequest(response, request, fmt.Errorf("backup point not found"))
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	randNum := r.String(6)
	backup.Name = fmt.Sprintf("%s-%s-%s", c.Name, backup.Name, randNum)
	backup.Status.KubernetesVersion = c.KubernetesVersion
	backup.Status.FileName = backup.Name
	backup.BackupPointName = c.Labels[common.LabelBackupPoint]
	_, ok := backup.Annotations[common.AnnotationDescription]
	if !ok {
		backup.Annotations[common.AnnotationDescription] = ""
	}

	// check preferred node in cluster
	if backup.PreferredNode == "" {
		backup.PreferredNode = c.Masters[0].ID
	}
	if len(c.Masters.Intersect(v1.WorkerNode{
		ID: backup.PreferredNode,
	})) == 0 {
		restplus.HandleBadRequest(response, request, fmt.Errorf("the node %s not a master node", backup.PreferredNode))
		return
	}

	q := query.New()
	q.LabelSelector = fmt.Sprintf("%s=%s", common.LabelClusterName, c.Name)
	nodeList, err := h.clusterOperator.ListNodes(context.TODO(), q)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	if backup.PreferredNode != "" {
		_, err := h.clusterOperator.GetNodeEx(ctx, backup.PreferredNode, "0")
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}
	// create operation
	op := &v1.Operation{}
	op.Name = uuid.New().String()
	op.Labels = make(map[string]string)
	op.Labels[common.LabelOperationAction] = v1.OperationBackupCluster
	op.Labels[common.LabelOperationSponsor] = buildOperationSponsor(h.genericConfig)
	op.Labels[common.LabelTimeoutSeconds] = strconv.Itoa(v1.DefaultBackupTimeoutSec)
	op.Labels[common.LabelClusterName] = c.Name
	op.Labels[common.LabelBackupName] = backup.Name
	op.Labels[common.LabelTopologyRegion] = c.Masters[0].Labels[common.LabelTopologyRegion]
	op.Status.Status = v1.OperationStatusRunning
	// add backup
	backup.Labels = make(map[string]string)
	backup.Labels[common.LabelClusterName] = c.Name
	backup.Labels[common.LabelOperationName] = op.Name
	backup.Labels[common.LabelTimeoutSeconds] = strconv.Itoa(v1.DefaultBackupTimeoutSec)
	backup.ClusterNodes = make(map[string]string)
	for _, node := range nodeList.Items {
		backup.ClusterNodes[node.Status.Ipv4DefaultIP] = node.Status.NodeInfo.Hostname
	}

	if !dryRun {
		op.Steps, err = h.parseActBackupSteps(c, backup, v1.ActionInstall)
		if err != nil {
			logger.Errorf("parse create backup step failed: %s", err.Error())
			restplus.HandleInternalError(response, request, err)
			return
		}
		if op, err = h.opOperator.CreateOperation(context.TODO(), op); err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		if backup, err = h.clusterOperator.CreateBackup(context.TODO(), backup); err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		// update cluster status to backing_up
		c.Status.Phase = v1.ClusterBackingUp
		if _, err = h.clusterOperator.UpdateCluster(context.TODO(), c); err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		go h.doOperation(context.TODO(), op, &service.Options{DryRun: dryRun})
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, backup)
}

func (h *handler) DeleteBackup(request *restful.Request, response *restful.Response) {
	// cluster name in path
	clusterName := request.PathParameter("cluster")
	ctx := request.Request.Context()
	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	c, err := h.clusterOperator.GetClusterEx(ctx, clusterName, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	// backup name in path
	backupName := request.PathParameter("backup")
	b, err := h.clusterOperator.GetBackupEx(ctx, clusterName, backupName)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	if b.PreferredNode != "" {
		_, err = h.clusterOperator.GetNodeEx(ctx, b.PreferredNode, "0")
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}
	// create operation
	op := &v1.Operation{}
	op.Name = uuid.New().String()
	op.Labels = make(map[string]string)
	op.Labels[common.LabelOperationAction] = v1.OperationDeleteBackup
	op.Labels[common.LabelTimeoutSeconds] = strconv.Itoa(v1.DefaultBackupTimeoutSec)
	op.Labels[common.LabelOperationSponsor] = buildOperationSponsor(h.genericConfig)
	op.Labels[common.LabelClusterName] = c.Name
	op.Labels[common.LabelBackupName] = b.Name
	op.Labels[common.LabelTopologyRegion] = c.Masters[0].Labels[common.LabelTopologyRegion]
	op.Status.Status = v1.OperationStatusRunning

	if b.Status.ClusterBackupStatus == v1.ClusterBackupRestoring || b.Status.ClusterBackupStatus == v1.ClusterBackupCreating {
		restplus.HandleBadRequest(response, request, fmt.Errorf("backup is %s now, can't delete", b.Status.ClusterBackupStatus))
		return
	}

	if !dryRun {
		c, err = h.clusterOperator.GetCluster(ctx, clusterName)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		// build the backup steps instance
		op.Steps, err = h.parseActBackupSteps(c, b, v1.ActionUninstall)
		if err != nil {
			logger.Errorf("delete backup step parse failed: %s", err.Error())
			restplus.HandleInternalError(response, request, err)
			return
		}
		if op, err = h.opOperator.CreateOperation(context.TODO(), op); err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		if err = h.clusterOperator.DeleteBackup(context.TODO(), backupName); err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		go h.doOperation(context.TODO(), op, &service.Options{DryRun: dryRun})
	}
	response.WriteHeader(http.StatusOK)
}

func (h *handler) UpdateBackup(request *restful.Request, response *restful.Response) {
	b := &v1.Backup{}
	if err := request.ReadEntity(b); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	// cluster name in path
	clusterName := request.PathParameter("cluster")
	ctx := request.Request.Context()

	// backup name in path
	backupName := request.PathParameter("backup")
	backup, err := h.clusterOperator.GetBackupEx(ctx, clusterName, backupName)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	// only description can be modified
	backup.Annotations[common.AnnotationDescription] = b.Annotations[common.AnnotationDescription]

	backup, err = h.clusterOperator.UpdateBackup(ctx, backup)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, backup)
}

func (h *handler) RetryCluster(request *restful.Request, response *restful.Response) {
	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	name := request.PathParameter(query.ParameterName)

	op, err := h.opOperator.GetOperationEx(request.Request.Context(), name, "0")
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	// backup/recovery/upgrade does not support retries
	switch op.Labels[common.LabelOperationAction] {
	case v1.OperationBackupCluster, v1.OperationRecoverCluster, v1.OperationUpgradeCluster:
		restplus.HandleBadRequest(response, request, fmt.Errorf("backup/recovery/upgrade operation-action does not support retries"))
		return
	case "":
		restplus.HandleBadRequest(response, request, fmt.Errorf("operation %s action is empty", name))
		return
	}

	// only the last retry is supported
	q := query.New()
	q.LabelSelector = fmt.Sprintf("%s=%s", common.LabelClusterName, op.Labels[common.LabelClusterName])
	q.Pagination.Offset = 0
	q.Pagination.Limit = 1
	opList, err := h.opOperator.ListOperationsEx(request.Request.Context(), q)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	if len(opList.Items) == 0 {
		restplus.HandleBadRequest(response, request, fmt.Errorf("the cluster did not query the operation"))
		return
	}

	op = opList.Items[0].(*v1.Operation)
	if op.Status.Status == v1.OperationStatusSuccessful || op.Status.Status == v1.OperationStatusRunning || op.Name != name {
		restplus.HandleBadRequest(response, request, fmt.Errorf("only the latest faild operation can do a retry"))
		return
	}

	// error step index
	failedIndex := len(op.Status.Conditions) - 1
	ctx := component.WithRetry(context.TODO(), true)

	// if there is an uninstall error, continue directly from the current step
	var continueSteps []v1.Step
	if op.Steps[0].Action == v1.ActionInstall {
		findStepNode := func(nodes []v1.StepNode, nodeID string) v1.StepNode {
			for _, v := range nodes {
				if v.ID == nodeID {
					return v
				}
			}
			return v1.StepNode{}
		}
		var failedNodes []v1.StepNode
		successStatus := make([]v1.StepStatus, 0)
		for _, status := range op.Status.Conditions[failedIndex].Status {
			if status.Status == "" || status.Status == v1.StepStatusFailed {
				// select the nodes whose execution fails
				if node := findStepNode(op.Steps[failedIndex].Nodes, status.Node); node.ID != "" {
					failedNodes = append(failedNodes, node)
				}
				continue
			}
			successStatus = append(successStatus, status)
		}
		// the step to continue
		continueSteps = op.Steps[failedIndex:]

		// the node that failed to execute the task
		continueSteps[0].Nodes = failedNodes

		if len(successStatus) != 0 {
			// failed status
			failedStepStatus := op.Status.Conditions[failedIndex]
			// retain successful status
			failedStepStatus.Status = successStatus
			// Remove the failed status and keep the successful status
			op.Status.Conditions = op.Status.Conditions[0:failedIndex]
			op.Status.Conditions = append(op.Status.Conditions, failedStepStatus)
		} else {
			op.Status.Conditions = op.Status.Conditions[0:failedIndex]
		}
		if failedIndex > 0 && op.Status.Conditions[failedIndex-1].Status[0].Response != nil {
			ctx = component.WithExtraData(ctx, op.Status.Conditions[failedIndex-1].Status[0].Response)
		}
	}

	// if there is an uninstall error, start from the beginning
	if op.Steps[0].Action == v1.ActionUninstall {
		continueSteps = op.Steps
		op.Status.Conditions = make([]v1.OperationCondition, 0)
	}

	op.Status.Status = v1.OperationStatusRunning

	if !dryRun {
		_, err = h.opOperator.UpdateOperation(context.TODO(), op)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		var c *v1.Cluster
		if c, err = h.clusterOperator.GetClusterEx(context.TODO(), op.Labels[common.LabelClusterName], "0"); err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		c.Status.Phase = v1.ClusterUpdating
		if _, err = h.clusterOperator.UpdateCluster(context.TODO(), c); err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}

	op.Steps = continueSteps

	go h.doOperation(ctx, op, &service.Options{DryRun: dryRun})
	_ = response.WriteHeaderAndEntity(http.StatusOK, nil)
}

func (h *handler) CreateRecovery(request *restful.Request, response *restful.Response) {
	r := &v1.Recovery{}
	if err := request.ReadEntity(r); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	// cluster name in path
	clusterName := request.PathParameter("cluster")
	ctx := request.Request.Context()

	c, err := h.clusterOperator.GetClusterEx(ctx, clusterName, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	q := query.New()
	q.LabelSelector = fmt.Sprintf("%s=%s", common.LabelClusterName, c.Name)
	nodeList, err := h.clusterOperator.ListNodes(context.TODO(), q)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	b, err := h.clusterOperator.GetBackup(ctx, clusterName, r.UseBackupName)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	for _, node := range nodeList.Items {
		val, ok := b.ClusterNodes[node.Status.Ipv4DefaultIP]
		if !ok {
			restplus.HandleInternalError(response, request,
				fmt.Errorf("the current node IP(%s) is not included in the backup file information(%v)",
					node.Status.Ipv4DefaultIP, b.ClusterNodes))
			return
		}
		if val != node.Status.NodeInfo.Hostname {
			restplus.HandleInternalError(response, request,
				fmt.Errorf("the current node Hostname(%s) is not included in the backup file information(%v)",
					node.Status.NodeInfo.Hostname, b.ClusterNodes))
			return
		}
	}

	switch c.Status.Phase {
	case v1.ClusterRestoring, v1.ClusterBackingUp,
		v1.ClusterTerminating, v1.ClusterUpdating, v1.ClusterInstalling:
		restplus.HandleInternalError(response, request,
			fmt.Errorf("cluster %s current is %s, can't recovery",
				c.Name, c.Status.ComponentConditions[0].Status))
		return
	}

	rName := uuid.New().String()
	oName := uuid.New().String()

	// update cluster status to recovering
	c.Status.Phase = v1.ClusterRestoring

	// create operation
	o := &v1.Operation{}
	o.Name = oName
	o.Labels = make(map[string]string)
	o.Labels[common.LabelOperationAction] = v1.OperationRecoverCluster
	o.Labels[common.LabelTimeoutSeconds] = strconv.Itoa(v1.DefaultRecoveryTimeoutSec)
	o.Labels[common.LabelOperationSponsor] = buildOperationSponsor(h.genericConfig)
	o.Labels[common.LabelClusterName] = c.Name
	o.Labels[common.LabelRecoveryName] = rName
	o.Labels[common.LabelTopologyRegion] = c.Masters[0].Labels[common.LabelTopologyRegion]
	o.Status.Status = v1.OperationStatusRunning
	restoreDir := filepath.Join("/var/lib/kube-restore", c.Name)
	steps, err := h.parseRecoverySteps(c, b, restoreDir, v1.ActionInstall)
	if err != nil {
		restplus.HandleInternalError(response, request, fmt.Errorf("recovery failed: %v", err))
		return
	}
	o.Steps = steps

	// add recovery
	r.Name = rName
	r.Labels = make(map[string]string)
	r.Labels[common.LabelClusterName] = c.Name
	r.Labels[common.LabelOperationName] = oName
	r.Labels[common.LabelTimeoutSeconds] = strconv.Itoa(v1.DefaultBackupTimeoutSec)

	if !dryRun {
		if c, err = h.clusterOperator.UpdateCluster(ctx, c); err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}

	go func(c *v1.Cluster, op *v1.Operation, r *v1.Recovery, b *v1.Backup) {
		var err error
		newOP, err := h.opOperator.CreateOperation(context.TODO(), op)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		// update backup operation label
		b.Labels[common.LabelOperationName] = oName
		if _, err = h.clusterOperator.UpdateBackup(context.TODO(), b); err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		h.doOperation(context.TODO(), newOP, &service.Options{DryRun: dryRun})
	}(c, o, r, b)

	_ = response.WriteHeaderAndEntity(http.StatusOK, r)
}

func (h *handler) InstallOrUninstallPlugins(request *restful.Request, response *restful.Response) {
	pcs := &PatchComponents{}
	if err := request.ReadEntity(pcs); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	clusterName := request.PathParameter("cluster")
	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	timeoutSecs := v1.DefaultOperationTimeoutSecs
	if v := request.QueryParameter("timeout"); v != "" {
		timeoutSecs = v
	}

	ctx := request.Request.Context()
	clu, err := h.clusterOperator.GetClusterEx(ctx, clusterName, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	// We need IP addresses of all master nodes later.
	extraMeta, err := h.getClusterMetadata(ctx, clu, false)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) || err == ErrNodesRegionDifferent {
			restplus.HandleBadRequest(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	if err = pcs.checkComponents(clu); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	action, operationAction := v1.ActionInstall, v1.OperationInstallComponents
	if pcs.Uninstall {
		action = v1.ActionUninstall
		operationAction = v1.OperationUninstallComponents
	}
	op, err := h.parseOperationFromComponent(ctx, extraMeta, pcs.Addons, clu, action)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	// The current component does not support uninstallation, so the steps will be empty
	if len(op.Steps) == 0 {
		restplus.HandleBadRequest(response, request, errors.New("the current operation steps is empty and cannot be performed"))
		return
	}

	op.Labels[common.LabelTimeoutSeconds] = timeoutSecs
	op.Labels[common.LabelOperationAction] = operationAction
	op.Labels[common.LabelOperationSponsor] = buildOperationSponsor(h.genericConfig)
	op.Status.Status = v1.OperationStatusRunning

	if !dryRun {
		// cluster.Status.Registries will be updated
		var statusRegistry []v1.RegistrySpec
		statusRegistry, err = h.getClusterCRIRegistries(ctx, clu)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		criStep, err := h.getCRIRegistriesStep(ctx, clu, statusRegistry)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		if criStep != nil {
			op.Steps = append(op.Steps[:1], op.Steps...)
			op.Steps[0] = *criStep
			clu.Status.Registries = statusRegistry
		}
		clu.Status.Phase = v1.ClusterUpdating
		_, err = h.clusterOperator.UpdateCluster(context.TODO(), clu)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}

		op, err = h.opOperator.CreateOperation(context.TODO(), op)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}

	}
	go func(o *v1.Operation, opts *service.Options, oPcs *PatchComponents) {
		if err := h.delivery.DeliverTaskOperation(context.TODO(), o, opts); err != nil {
			logger.Error("delivery task error", zap.Error(err))
			return
		}
		logger.Debugf("the install or uninstall plugins message was delivered successfully")
		// the database is not updated until the message is delivered successfully
		if !opts.DryRun {
			latestCluster, err := h.clusterOperator.GetClusterEx(ctx, clusterName, "0")
			if err != nil {
				logger.Error("get the latest cluster info error", zap.Error(err))
				return
			}
			newCluster, err := oPcs.addOrRemoveComponentFromCluster(latestCluster)
			if err != nil {
				logger.Error("add or remove component from cluster", zap.Error(err))
				return
			}
			_, err = h.clusterOperator.UpdateCluster(context.TODO(), newCluster)
			if err != nil {
				logger.Error("update cluster metadata error", zap.Error(err))
			}
		}
	}(op, &service.Options{DryRun: dryRun}, pcs)

	_ = response.WriteHeaderAndEntity(http.StatusOK, clu)
}

func (h *handler) UpgradeCluster(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	body := &ClusterUpgrade{}
	if err := request.ReadEntity(body); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	clu, err := h.clusterOperator.GetClusterEx(request.Request.Context(), name, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	// TODO: validate cluster struct
	timeoutSecs := v1.DefaultOperationTimeoutSecs
	if v := request.QueryParameter("timeout"); v != "" {
		timeoutSecs = v
	}
	extraMeta, err := h.getClusterMetadata(request.Request.Context(), clu, false)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) || err == ErrNodesRegionDifferent {
			restplus.HandleBadRequest(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	extraMeta.Offline = body.Offline
	extraMeta.KubeVersion = body.Version
	extraMeta.LocalRegistry = body.LocalRegistry
	upgradeComp := &k8s.Upgrade{}
	upgradeComp.InitStepper(extraMeta, clu)
	if err := upgradeComp.Validate(); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	if err := upgradeComp.InitSteps(component.WithExtraMetadata(context.TODO(), *extraMeta)); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	op := &v1.Operation{}
	op.Name = uuid.New().String()
	op.Labels = map[string]string{
		common.LabelClusterName:    clu.Name,
		common.LabelTopologyRegion: extraMeta.Masters[0].Region,
	}
	op.Steps = upgradeComp.GetInstallSteps()

	// TODO: make dry run path to etcd
	if !dryRun {
		clu.Status.Phase = v1.ClusterUpgrading
		_, err = h.clusterOperator.UpdateCluster(request.Request.Context(), clu)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}

	op.Labels[common.LabelTimeoutSeconds] = timeoutSecs
	op.Labels[common.LabelOperationAction] = v1.OperationUpgradeCluster
	op.Labels[common.LabelOperationSponsor] = buildOperationSponsor(h.genericConfig)
	op.Labels[common.LabelUpgradeVersion] = body.Version
	op.Status.Status = v1.OperationStatusRunning
	if !dryRun {
		op, err = h.opOperator.CreateOperation(context.TODO(), op)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}
	go h.doOperation(context.TODO(), op, &service.Options{DryRun: dryRun})
	response.WriteHeader(http.StatusOK)
}

func (h *handler) ResetClusterStatus(request *restful.Request, response *restful.Response) {
	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	cluName := request.PathParameter(query.ParameterName)
	clu, err := h.clusterOperator.GetClusterEx(request.Request.Context(), cluName, "0")
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	switch clu.Status.Phase {
	case v1.ClusterInstalling, v1.ClusterInstallFailed,
		v1.ClusterUpdating, v1.ClusterUpdateFailed,
		v1.ClusterUpgrading, v1.ClusterUpgradeFailed,
		v1.ClusterBackingUp, v1.ClusterRestoring, v1.ClusterRestoreFailed,
		v1.ClusterTerminating, v1.ClusterTerminateFailed:
		if dryRun {
			response.WriteHeader(http.StatusOK)
			return
		}
		clu.Status.Phase = v1.ClusterRunning
		_, err := h.clusterOperator.UpdateCluster(request.Request.Context(), clu)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		response.WriteHeader(http.StatusOK)
	default:
		restplus.HandleBadRequest(response, request,
			fmt.Errorf("not supported reset %s status", clu.Status.Phase))
		return
	}
}

func (h *handler) ListLeases(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	if q.Watch {
		h.watchLeases(request, response, q)
		return
	}
	if clientrest.IsInformerRawQuery(request.Request) {
		result, err := h.leaseOperator.ListLeases(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
		return
	}
	response.WriteHeader(http.StatusOK)
}

func (h *handler) watchLeases(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.leaseOperator.WatchLease(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("Lease"), req, resp, timeout)
}

func (h *handler) DescribeLease(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	resourceVersion := request.QueryParameter(query.ParameterResourceVersion)
	c, err := h.leaseOperator.GetLease(request.Request.Context(), name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

// ---- domain

func (h *handler) ListDomains(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)

	if q.Watch {
		h.watchDomain(request, response, q)
		return
	}

	if clientrest.IsInformerRawQuery(request.Request) {
		result, err := h.clusterOperator.ListDomains(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
		return
	}
	result, err := h.clusterOperator.ListDomainsEx(request.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) watchDomain(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.clusterOperator.WatchDomain(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("Domain"), req, resp, timeout)
}

func (h *handler) CheckDomainExists(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	result, err := h.clusterOperator.ListDomainsEx(request.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	if len(result.Items) > 0 {
		response.Header().Set(resourceExistCheckerHeader, "true")
		response.WriteHeader(http.StatusOK)
		return
	}

	// parse domain from selector ,e.g. metadata.name=github.com
	spilt := strings.Split(q.FieldSelector, "=")
	if len(spilt) != 2 {
		restplus.HandleBadRequest(response, request, err)
		response.WriteHeader(http.StatusOK)
		return
	}
	checkDomain := spilt[1]

	exists, err := h.checkDomain(request.Request.Context(), checkDomain)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	if exists {
		response.Header().Set(resourceExistCheckerHeader, "true")
		return
	}
	response.Header().Set(resourceExistCheckerHeader, "false")
	response.WriteHeader(http.StatusOK)
}

func (h *handler) checkDomain(ctx context.Context, checkDomain string) (bool, error) {
	checkDomain = strings.ToLower(checkDomain)
	domains, err := h.clusterOperator.ListDomains(ctx, &query.Query{})
	if err != nil {
		return false, err
	}
	for _, domain := range domains.Items {
		domain.Name = strings.ToLower(domain.Name)
		// check a.example.com, exists example.com return true
		// check example.com, exists a.example.com return true
		if strings.Contains(checkDomain, "."+domain.Name) || strings.Contains(domain.Name, "."+checkDomain) {
			return true, nil
		}
	}
	return false, nil
}

func (h *handler) GetDomain(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	result, err := h.clusterOperator.GetDomain(request.Request.Context(), name)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) CreateDomains(request *restful.Request, response *restful.Response) {
	d := new(v1.Domain)
	if err := request.ReadEntity(d); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	if errs := validation.ValidateDomain(d); len(errs) > 0 {
		restplus.HandleBadRequest(response, request, errs.ToAggregate())
		return
	}

	exists, err := h.checkDomain(request.Request.Context(), d.Name)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	err = h.checkSyncCluster(d.Spec.SyncCluster)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	if exists {
		restplus.HandleBadRequest(response, request, fmt.Errorf("domain %s is already exist", d.Name))
		return
	}

	createDomain, err := h.clusterOperator.CreateDomain(request.Request.Context(), d)
	if err != nil {
		if apimachineryErrors.IsAlreadyExists(err) {
			restplus.HandleBadRequest(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, createDomain)
}

func (h *handler) checkSyncCluster(syncCluster []string) error {
	clusters, err := h.clusterOperator.ListClusters(context.Background(), &query.Query{
		ResourceVersion:      "0",
		ResourceVersionMatch: query.ResourceVersionMatchNotOlderThan,
	})
	if err != nil {
		return err
	}
	m := make(map[string]struct{}, len(clusters.Items))
	for _, item := range clusters.Items {
		m[item.Name] = struct{}{}
	}

	for _, clusterName := range syncCluster {
		_, ok := m[clusterName]
		if !ok {
			return fmt.Errorf("cluster %s is not exist", clusterName)
		}
	}
	return nil
}

func (h *handler) UpdateDomain(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	d := new(v1.Domain)
	if err := request.ReadEntity(d); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	if name != d.Name {
		restplus.HandleBadRequest(response, request, fmt.Errorf("the name of the object (%s) does not match "+
			"the name on the URL (%s)", d.Name, name))
		return
	}

	err := h.checkSyncCluster(d.Spec.SyncCluster)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	updated, err := h.updateDomain(request.Request.Context(), d)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, updated)
}

func (h *handler) updateDomain(ctx context.Context, domain *v1.Domain) (*v1.Domain, error) {
	d, err := h.clusterOperator.GetDomain(ctx, domain.Name)
	if err != nil {
		return nil, err
	}
	d.Spec.Description = domain.Spec.Description
	d.Spec.SyncCluster = domain.Spec.SyncCluster
	return h.clusterOperator.UpdateDomain(ctx, d)
}

func (h *handler) DeleteDomain(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)

	_, err := h.clusterOperator.GetDomain(request.Request.Context(), name)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			logger.Debug("domain has already not exist when delete", zap.String("domain", name))
			response.WriteHeader(http.StatusOK)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	err = h.clusterOperator.DeleteDomain(request.Request.Context(), name)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			logger.Debug("domain has already not exist when delete", zap.String("domain", name))
			response.WriteHeader(http.StatusOK)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	response.WriteHeader(http.StatusOK)
}

// ------- record

func (h *handler) ListRecords(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	subdomain := request.QueryParameter(query.ParameterSubDomain)
	q := query.ParseQueryParameter(request)

	result, err := h.clusterOperator.ListRecordsEx(request.Request.Context(), name, subdomain, q)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) GetRecord(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	subdomain := request.PathParameter(query.ParameterSubDomain)
	var (
		result *v1.Domain
		err    error
	)
	result, err = h.clusterOperator.GetDomain(request.Request.Context(), name)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	record, ok := result.Spec.Records[subdomain]
	if !ok {
		restplus.HandleNotFound(response, request, apimachineryErrors.NewNotFound(v1.Resource("record"), subdomain))
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, record)
}

func (h *handler) CreateRecords(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)

	record := new(v1.Record)
	if err := request.ReadEntity(record); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	if name != record.Domain {
		restplus.HandleBadRequest(response, request, fmt.Errorf("the name of the object (%s) does not match "+
			"the name on the URL (%s)", name, record.Domain))
		return
	}
	err := checkRecord(record)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	getDomain, err := h.clusterOperator.GetDomain(request.Request.Context(), name)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	key := record.RR + "." + getDomain.Name
	_, ok := getDomain.Spec.Records[key]
	if ok {
		restplus.HandleBadRequest(response, request, fmt.Errorf("record %s already exists", key))
		return
	}
	// ignore lower upper case
	key = strings.ToLower(key)
	for k := range getDomain.Spec.Records {
		if key == strings.ToLower(k) {
			restplus.HandleBadRequest(response, request, fmt.Errorf("record %s already exists", k))
			return
		}
	}

	createRecord, err := h.updateRecord(request.Request.Context(), getDomain, *record)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, createRecord)
}

func checkRecord(r *v1.Record) error {
	if len(r.ParseRecord) == 0 {
		return fmt.Errorf("resolve record cann not be empty")
	}
	return nil
}

func (h *handler) UpdateRecord(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	record := new(v1.Record)
	if err := request.ReadEntity(record); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	if name != record.Domain {
		restplus.HandleBadRequest(response, request, fmt.Errorf("the name of the object (%s) does not match "+
			"the domain on the URL (%s)", record.Domain, name))
		return
	}

	err := checkRecord(record)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	getDomain, err := h.clusterOperator.GetDomain(request.Request.Context(), name)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	key := record.RR + "." + getDomain.Name
	if _, ok := getDomain.Spec.Records[key]; !ok {
		restplus.HandleNotFound(response, request, fmt.Errorf("record %s not found", key))
		return
	}

	updated, err := h.updateRecord(request.Request.Context(), getDomain, *record)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, updated)
}

func (h *handler) updateRecord(ctx context.Context, domain *v1.Domain, record v1.Record) (*v1.Domain, error) {
	if domain.Spec.Records == nil {
		domain.Spec.Records = make(map[string]v1.Record)
	}
	key := record.RR + "." + record.Domain
	//  新建时填充 createTime
	if _, ok := domain.Spec.Records[key]; !ok {
		record.CreateTime = metav1.NewTime(time.Now())
	}

	domain.Spec.Records[key] = record
	return h.clusterOperator.UpdateDomain(ctx, domain)
}

func (h *handler) DeleteRecord(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	subdomain := request.PathParameter(query.ParameterSubDomain)

	domain, err := h.clusterOperator.GetDomain(request.Request.Context(), name)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			logger.Debug("domain has already not exist when delete", zap.String("domain", name))
			response.WriteHeader(http.StatusOK)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_, ok := domain.Spec.Records[subdomain]
	if !ok {
		restplus.HandleNotFound(response, request, fmt.Errorf("record %s not found", subdomain))
		return
	}
	deleted, err := h.deleteRecord(request.Request.Context(), domain, subdomain)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, deleted)
}

func (h *handler) deleteRecord(ctx context.Context, domain *v1.Domain, subdomain string) (*v1.Domain, error) {
	d, err := h.clusterOperator.GetDomain(ctx, domain.Name)
	if err != nil {
		return nil, err
	}
	delete(d.Spec.Records, subdomain)
	return h.clusterOperator.UpdateDomain(ctx, d)
}

var (
	upGrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024 * 1024 * 10,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type SSHCredential struct {
	Username []byte `json:"username"`
	Password []byte `json:"password"`
	Port     int    `json:"port"`
}

func (h *handler) SSHToPod(request *restful.Request, response *restful.Response) {
	clusterName := request.PathParameter(query.ParameterName)
	clientcfg, clientset, err := h.getCluster(clusterName)
	if err != nil {
		restplus.HandleError(response, request, err)
		return
	}

	// find kubectl pod by label
	list, err := clientset.CoreV1().Pods("kube-system").List(
		request.Request.Context(),
		metav1.ListOptions{
			LabelSelector: "k8s-app=kc-kubectl",
		},
	)
	if err != nil {
		logger.Errorf("kubectl console, find kubectl pod failed: %v", err)
		restplus.HandleInternalError(response, request, err)
		return
	}
	if len(list.Items) == 0 {
		restplus.HandleInternalError(response, request, errors.New("kubectl pod not exists"))
		logger.Errorf("kubectl console, kubectl pod not exists: %v", err)
		return
	}
	podName := list.Items[rand.Intn(len(list.Items))].GetName()

	h.execPod(request, response, clientcfg,
		"kube-system", podName, "kc-kubectl", []string{"sh", "-c", sshutils.MagicExecShell})
}

func parseCommand(c string) []string {
	if c == "" {
		return []string{"sh", "-c", sshutils.MagicExecShell}
	}
	parts := strings.Split(c, " ")
	cmd := parts[0:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cmd = append(cmd, p)
		}
	}
	return cmd
}

func (h *handler) ExecPod(request *restful.Request, response *restful.Response) {
	clusterName := request.PathParameter(query.ParameterName)
	namespace := request.PathParameter("namespace")
	pod := request.PathParameter("pod")
	container := request.QueryParameter("container")
	command := parseCommand(request.QueryParameter("command"))

	clientcfg, clientset, err := h.getCluster(clusterName)
	if err != nil {
		restplus.HandleError(response, request, err)
		return
	}

	// find kubectl pod by label
	po, err := clientset.CoreV1().Pods(namespace).Get(
		request.Request.Context(),
		pod,
		metav1.GetOptions{},
	)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		logger.Errorf("kubectl console, find kubectl pod failed: %v", err)
		restplus.HandleInternalError(response, request, err)
		return
	}
	if po.Status.Phase != corev1.PodRunning {
		restplus.HandleBadRequest(response, request, fmt.Errorf("pod %s is not running", pod))
		return
	}

	if container != "" {
		found := false
		for _, c := range po.Spec.Containers {
			if c.Name == container {
				found = true
				break
			}
		}
		if !found {
			restplus.HandleBadRequest(response, request, fmt.Errorf("container %s not found", container))
			return
		}
	} else {
		if len(po.Spec.Containers) != 1 {
			restplus.HandleBadRequest(response, request, fmt.Errorf("pod %s has more than one container", pod))
			return
		}
	}
	h.execPod(request, response, clientcfg, namespace, pod, container, command)
}

func (h *handler) SSHToNode(request *restful.Request, response *restful.Response) {
	var (
		priKey           []byte
		setting          = &v1.PlatformSetting{}
		quitChan         = make(chan struct{}, 3)
		gracefulExitChan = make(chan struct{})
	)
	nodeName := request.PathParameter(query.ParameterName)
	msg := request.QueryParameter("msg")
	cols := query.GetIntValueWithDefault(request, ParameterCols, 150)
	rows := query.GetIntValueWithDefault(request, ParameterRows, 35)

	wsConn, err := upGrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	defer wsConn.Close()

	err = wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))
	if err != nil {
		logger.Errorf("%s", err)
	}
	wsConn.SetPongHandler(func(appData string) error {
		logger.Debug("in pong handler...")
		err = wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))
		if err != nil {
			logger.Errorf("%s", err)
		}
		return nil
	})
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func() {
		defer logger.Debug("websocket send ping return...")
		for {
			select {
			case <-ticker.C:
				// wsConn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-quitChan:
				return
			}
		}
	}()
	wsConn.SetCloseHandler(func(code int, text string) error {
		logger.Debug("in close handler...")
		message := websocket.FormatCloseMessage(code, text)
		err = wsConn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		if err != nil {
			logger.Errorf("%s", err)
		}
		quitChan <- struct{}{}
		return nil
	})

	node, err := h.clusterOperator.GetNodeEx(request.Request.Context(), nodeName, "0")
	if err != nil {
		_ = wsConn.CloseHandler()(4000, "BadRequest: parameter error")
		return
	}
	credential, err := decodeMsgToSSH(msg)
	if err != nil {
		_ = wsConn.CloseHandler()(4000, "BadRequest: parameter error")
		return
	}
	setting, err = h.platformOperator.GetPlatformSetting(request.Request.Context())
	if err != nil {
		_ = wsConn.CloseHandler()(4000, "BadRequest: parameter error")
		return
	}
	if setting == nil {
		_ = wsConn.CloseHandler()(4000, "BadRequest: parameter error")
		return
	}
	if setting.Terminal.PrivateKey != "" {
		priKey, err = base64.StdEncoding.DecodeString(setting.Terminal.PrivateKey)
		if err != nil {
			_ = wsConn.CloseHandler()(4000, "BadRequest: parameter error")
			return
		}
	} else {
		_ = wsConn.CloseHandler()(4000, "BadRequest: parameter error")
		return
	}

	credential.Username, err = certs.RsaDecrypt(credential.Username, priKey)
	if err != nil {
		_ = wsConn.CloseHandler()(4000, "BadRequest: parameter error")
		return
	}
	credential.Password, err = certs.RsaDecrypt(credential.Password, priKey)
	if err != nil {
		_ = wsConn.CloseHandler()(4000, "BadRequest: parameter error")
		return
	}
	sshClient, err := sshutils.NewSSHClient(string(credential.Username), string(credential.Password), node.Status.Ipv4DefaultIP, credential.Port)
	if err != nil {
		_ = wsConn.CloseHandler()(4001, "username or password incorrect")
		return
	}
	defer sshClient.Close()

	sshConn, err := sshutils.NewLoginSSHWSSession(cols, rows, true, sshClient, wsConn)
	if err != nil {
		_ = wsConn.CloseHandler()(4002, "ssh connection failed")
		return
	}
	defer sshConn.Close()

	sshConn.Start(quitChan)
	go sshConn.Wait(quitChan, gracefulExitChan)
	select {
	case <-quitChan:
		return
	case <-gracefulExitChan:
		time.Sleep(1 * time.Second)
		message := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		if err := wsConn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second)); err != nil {
			logger.Errorf("close websocket error due to: %s", err.Error())
		}
		return
	}
}

func decodeMsgToSSH(msg string) (*SSHCredential, error) {
	c := &SSHCredential{}
	decoded, err := base64.StdEncoding.DecodeString(msg)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(decoded, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (h *handler) ListTemplates(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	if q.Watch {
		h.watchTemplates(request, response, q)
		return
	}
	templates, err := h.clusterOperator.ListTemplatesEx(request.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, templates)
}

func (h *handler) watchTemplates(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.clusterOperator.WatchTemplates(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("Templates"), req, resp, timeout)
}

func (h *handler) DescribeTemplate(request *restful.Request, response *restful.Response) {
	templateName := request.PathParameter(query.ParameterName)
	template, err := h.clusterOperator.GetTemplateEx(request.Request.Context(), templateName, "0")
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, template)
}

func (h *handler) CreateTemplate(request *restful.Request, response *restful.Response) {
	template := &v1.Template{}
	err := request.ReadEntity(template)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	ok, err := h.checkTemplateExist(request.Request.Context(), template.Annotations[common.AnnotationDisplayName])
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	if ok {
		restplus.HandleBadRequest(response, request, fmt.Errorf("template '%s' already exists", template.Annotations[common.AnnotationDisplayName]))
		return
	}

	template.ObjectMeta.GenerateName = "tmpl-"
	template, err = h.clusterOperator.CreateTemplate(request.Request.Context(), template)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusCreated, template)
}

func (h *handler) UpdateTemplate(request *restful.Request, response *restful.Response) {
	template := &v1.Template{}
	err := request.ReadEntity(template)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	templateName := request.PathParameter(query.ParameterName)
	if templateName != template.Name {
		restplus.HandleBadRequest(response, request, fmt.Errorf("template name not match"))
		return
	}
	template, err = h.clusterOperator.UpdateTemplate(request.Request.Context(), template)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, template)
}

func (h *handler) DeleteTemplate(request *restful.Request, response *restful.Response) {
	templateName := request.PathParameter(query.ParameterName)
	err := h.clusterOperator.DeleteTemplate(request.Request.Context(), templateName)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	response.WriteHeader(http.StatusOK)
}

func (h *handler) CheckTemplateExists(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)

	displayName := q.FieldSelector
	exists, err := h.checkTemplateExist(request.Request.Context(), displayName)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	if exists {
		response.Header().Set(resourceExistCheckerHeader, "true")
		return
	}
	response.Header().Set(resourceExistCheckerHeader, "false")
	response.WriteHeader(http.StatusOK)
}

func (h *handler) checkTemplateExist(ctx context.Context, name string) (bool, error) {
	templates, err := h.clusterOperator.ListTemplates(ctx, query.New())
	if err != nil {
		return false, err
	}
	for _, template := range templates.Items {
		if template.Annotations[common.AnnotationDisplayName] == name {
			return true, nil
		}
	}
	return false, nil
}

func (h *handler) DescribeBackupPoint(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	result, err := h.clusterOperator.GetBackupPointEx(request.Request.Context(), name, resourceVersion)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) ListBackupPoints(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	ctx := request.Request.Context()

	if q.Watch {
		h.watchBackupPoints(request, response, q)
		return
	}
	if clientrest.IsInformerRawQuery(request.Request) {
		result, err := h.clusterOperator.ListBackupPoints(ctx, q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	} else {
		result, err := h.clusterOperator.ListBackupPointEx(ctx, q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	}
}

func (h *handler) watchBackupPoints(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.clusterOperator.WatchBackupPoints(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("BackupPoint"), req, resp, timeout)
}

func (h *handler) CreateBackupPoint(request *restful.Request, response *restful.Response) {
	bp := new(v1.BackupPoint)
	if err := request.ReadEntity(bp); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	bp.StorageType = strings.ToLower(bp.StorageType)
	if bp.StorageType == bs.S3Storage && len([]rune(bp.S3Config.Bucket)) <= 3 {
		restplus.HandleBadRequest(response, request, fmt.Errorf("bucket name cannot be shorter than 3 characters"))
		return
	}

	createdBp, err := h.clusterOperator.CreateBackupPoint(request.Request.Context(), bp)
	if err != nil {
		if apimachineryErrors.IsAlreadyExists(err) {
			restplus.HandleBadRequest(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, createdBp)
}

func (h *handler) DeleteBackupPoint(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	q := query.ParseQueryParameter(request)
	clusters, err := h.clusterOperator.ListClusters(request.Request.Context(), &query.Query{
		Pagination:      query.NoPagination(),
		ResourceVersion: "0",
		LabelSelector:   common.LabelBackupPoint,
		FieldSelector:   "",
	})
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	if clus := h.checkBackupPointInUseByCluster(clusters, name); len(clus) > 0 {
		restplus.HandleInternalError(response, request, fmt.Errorf("backup point is in use by clusters %v, please update cluster first", clus))
		return
	}
	backups, err := h.clusterOperator.ListBackups(request.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	if ok := h.checkBackupPointInUseByBackup(backups, name); ok {
		restplus.HandleInternalError(response, request, errors.New("backup point is in use, please delete backup first"))
		return
	}
	err = h.clusterOperator.DeleteBackupPoint(request.Request.Context(), name)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			logger.Debug("backup point has already not exist when delete", zap.String("backupPoint", name))
			response.WriteHeader(http.StatusOK)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	response.WriteHeader(http.StatusOK)
}

func (h *handler) UpdateBackupPoint(req *restful.Request, resp *restful.Response) {
	bp := &v1.BackupPoint{}
	if err := req.ReadEntity(bp); err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}

	name := req.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", req.QueryParameter(query.ParameterResourceVersion))
	obp, err := h.clusterOperator.GetBackupPointEx(req.Request.Context(), name, resourceVersion)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}

	if bp.StorageType == bs.FSStorage && obp.StorageType == bs.FSStorage && bp.S3Config == nil {
		// only fs backup point description can be modified
		obp.Description = bp.Description
	}

	if bp.StorageType == bs.S3Storage && obp.StorageType == bs.S3Storage && bp.FsConfig == nil {
		obp.S3Config.AccessKeyID = bp.S3Config.AccessKeyID
		obp.S3Config.AccessKeySecret = bp.S3Config.AccessKeySecret
		obp.Description = bp.Description
	}

	_, err = h.clusterOperator.UpdateBackupPoint(req.Request.Context(), obp)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}

	_ = resp.WriteHeaderAndEntity(http.StatusOK, obp)
}

func (h *handler) DescribeCronBackup(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	result, err := h.clusterOperator.GetCronBackupEx(request.Request.Context(), name, resourceVersion)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) ListCronBackups(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	if q.Watch {
		h.watchCronBackups(request, response, q)
		return
	}
	if clientrest.IsInformerRawQuery(request.Request) {
		result, err := h.clusterOperator.ListCronBackups(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	} else {
		result, err := h.clusterOperator.ListCronBackupEx(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	}
}

func (h *handler) watchCronBackups(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.clusterOperator.WatchCronBackups(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("CronBackup"), req, resp, timeout)
}

func (h *handler) CreateCronBackup(request *restful.Request, response *restful.Response) {
	cb := &v1.CronBackup{}
	if err := request.ReadEntity(cb); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	if cb.Spec.RunAt != nil {
		if cb.Spec.RunAt.Time.Add(3 * time.Second).Before(time.Now()) {
			restplus.HandleBadRequest(response, request, fmt.Errorf("the specified run time should be later than the current time"))
			return
		}
	}

	ok, err := h.checkCronBackupExist(request.Request.Context(), cb.Name, cb.Spec.ClusterName)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	if ok {
		restplus.HandleBadRequest(response, request, fmt.Errorf("cron backup '%s' of '%s' already exists", cb.Name, cb.Spec.ClusterName))
		return
	}

	cb.Labels[common.LabelCronBackupEnable] = ""
	createdCB, err := h.clusterOperator.CreateCronBackup(request.Request.Context(), cb)
	if err != nil && !apimachineryErrors.IsAlreadyExists(err) {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, createdCB)
}

func (h *handler) checkCronBackupExist(ctx context.Context, name, cluster string) (bool, error) {
	q := &query.Query{
		Pagination: query.NoPagination(),
		FuzzySearch: map[string]string{
			"name":    name,
			"cluster": cluster,
		},
	}
	cronBackups, err := h.clusterOperator.ListCronBackupEx(ctx, q)
	if err != nil {
		return false, err
	}

	if cronBackups.TotalCount > 0 {
		return true, nil
	}
	return false, nil
}

func (h *handler) DeleteCronBackup(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	err := h.clusterOperator.DeleteCronBackup(request.Request.Context(), name)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			logger.Debug("cronBackup not exist when delete", zap.String("cronBackup", name))
			response.WriteHeader(http.StatusOK)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	response.WriteHeader(http.StatusOK)
}

func (h *handler) UpdateCronBackup(req *restful.Request, resp *restful.Response) {
	cb := &v1.CronBackup{}
	if err := req.ReadEntity(cb); err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}
	name := req.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", req.QueryParameter(query.ParameterResourceVersion))
	ocb, err := h.clusterOperator.GetCronBackupEx(req.Request.Context(), name, resourceVersion)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}

	if cb.Spec.RunAt != nil {
		// disposable performed cron backup cannot be edited
		if cb.Status.LastSuccessfulTime != nil {
			if after := metav1.NewTime(time.Now()).After(cb.Status.LastSuccessfulTime.Time); after {
				reason := fmt.Errorf("disposable performed cron backup cannot be edited")
				restplus.HandlerErrorWithCustomCode(resp, req, http.StatusExpectationFailed, 417, "edit cron backup failed", reason)
				return
			}
		}
		ocb.Spec.RunAt = cb.Spec.RunAt
		ocb.Status.NextScheduleTime = cb.Spec.RunAt
	}

	if cb.Spec.Schedule != "" {
		now := func() time.Time {
			return time.Now()
		}
		Schedule := cronbackupcontroller.ParseSchedule(cb.Spec.Schedule, now())
		// update the next schedule time
		s, _ := cron.NewParser(4 | 8 | 16 | 32 | 64).Parse(Schedule)
		nextRunAt := metav1.NewTime(s.Next(time.Now()))
		ocb.Status.NextScheduleTime = &nextRunAt
	}
	ocb.Spec.Schedule = cb.Spec.Schedule
	ocb.Spec.MaxBackupNum = cb.Spec.MaxBackupNum
	_, err = h.clusterOperator.UpdateCronBackup(req.Request.Context(), ocb)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, ocb)
}

func (h *handler) EnableCronBackup(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter(query.ParameterName)
	ctx := req.Request.Context()
	resourceVersion := strutil.StringDefaultIfEmpty("0", req.QueryParameter(query.ParameterResourceVersion))
	cronBackup, err := h.clusterOperator.GetCronBackupEx(ctx, name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}

	if _, ok := cronBackup.Labels[common.LabelCronBackupEnable]; ok {
		restplus.HandleBadRequest(resp, req, fmt.Errorf("the cronbackup %s is already enable:%v", cronBackup.Name, cronBackup.Labels[common.LabelCronBackupEnable]))
	}
	delete(cronBackup.Labels, common.LabelCronBackupDisable)
	cronBackup.Labels[common.LabelCronBackupEnable] = ""
	updateCronBackup, err := h.clusterOperator.UpdateCronBackup(ctx, cronBackup)
	if err != nil {
		if apimachineryErrors.IsConflict(err) {
			restplus.HandleBadRequest(resp, req, fmt.Errorf("the cronBackup %s has been modified; please apply "+
				"your changes to the latest version and try again", cronBackup.Name))
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, updateCronBackup)
}

func (h *handler) DisableCronBackup(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter(query.ParameterName)
	ctx := req.Request.Context()
	resourceVersion := strutil.StringDefaultIfEmpty("0", req.QueryParameter(query.ParameterResourceVersion))
	cronBackup, err := h.clusterOperator.GetCronBackupEx(ctx, name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}

	if _, ok := cronBackup.Labels[common.LabelCronBackupDisable]; ok {
		restplus.HandleBadRequest(resp, req, fmt.Errorf("the cronbackup %s is already disable:%v", cronBackup.Name, cronBackup.Labels[common.LabelCronBackupDisable]))
	}
	delete(cronBackup.Labels, common.LabelCronBackupEnable)
	cronBackup.Labels[common.LabelCronBackupDisable] = ""
	updateCronBackup, err := h.clusterOperator.UpdateCronBackup(ctx, cronBackup)
	if err != nil {
		if apimachineryErrors.IsConflict(err) {
			restplus.HandleBadRequest(resp, req, fmt.Errorf("the cronbackup %s has been modified; please apply "+
				"your changes to the latest version and try again", updateCronBackup.Name))
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, updateCronBackup)
}

func (h *handler) ListBackupsWithCronBackup(req *restful.Request, resp *restful.Response) {
	var result []v1.Backup
	// cronBackup name in path
	cronBackupName := req.PathParameter(query.ParameterName)
	ctx := req.Request.Context()
	cronBackup, err := h.clusterOperator.GetCronBackupEx(ctx, cronBackupName, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}
	q := query.ParseQueryParameter(req)

	backupList, err := h.clusterOperator.ListBackups(ctx, q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	for _, b := range backupList.Items {
		if len(b.GetOwnerReferences()) != 0 {
			if b.GetOwnerReferences()[0].UID == cronBackup.UID {
				result = append(result, b)
			}
		}
	}

	_ = resp.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) ListConfigMaps(req *restful.Request, resp *restful.Response) {
	q := query.ParseQueryParameter(req)
	if q.Watch {
		h.watchConfigMap(req, resp, q)
		return
	}
	if clientrest.IsInformerRawQuery(req.Request) {
		result, err := h.coreOperator.ListConfigMaps(req.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
		_ = resp.WriteHeaderAndEntity(http.StatusOK, result)
	} else {
		result, err := h.coreOperator.ListConfigMapsEx(req.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
		_ = resp.WriteHeaderAndEntity(http.StatusOK, result)
	}
}

func (h *handler) DescribeConfigMap(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", req.QueryParameter(query.ParameterResourceVersion))
	c, err := h.coreOperator.GetConfigMapEx(req.Request.Context(), name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) CreateConfigMap(req *restful.Request, resp *restful.Response) {
	cm := &v1.ConfigMap{}
	err := req.ReadEntity(cm)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	cm, err = h.coreOperator.CreateConfigMap(req.Request.Context(), cm)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusCreated, cm)
}

func (h *handler) UpdateConfigMap(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	cm := &v1.ConfigMap{}
	if err := req.ReadEntity(cm); err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}
	dryRun := query.GetBoolValueWithDefault(req, query.ParamDryRun, false)

	if name != cm.Name {
		restplus.HandleBadRequest(resp, req, errors.New("name in url path not same with body"))
		return
	}

	_, err := h.coreOperator.GetConfigMapEx(req.Request.Context(), name, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleBadRequest(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}

	// TODO: check configmap Immutable

	if !dryRun {
		cm, err = h.coreOperator.UpdateConfigMap(req.Request.Context(), cm)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, cm)
}

func (h *handler) DeleteConfigMap(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	dryRun := query.GetBoolValueWithDefault(req, query.ParamDryRun, false)
	_, err := h.coreOperator.GetConfigMapEx(req.Request.Context(), name, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleBadRequest(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}
	if !dryRun {
		err = h.coreOperator.DeleteConfigMap(req.Request.Context(), name)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
	}
	resp.WriteHeader(http.StatusOK)
}

func (h *handler) watchConfigMap(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.coreOperator.WatchConfigMaps(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("ConfigMap"), req, resp, timeout)
}

func (h *handler) ListCloudProviders(req *restful.Request, resp *restful.Response) {
	q := query.ParseQueryParameter(req)
	if q.Watch {
		h.watchCloudProvider(req, resp, q)
		return
	}
	if clientrest.IsInformerRawQuery(req.Request) {
		result, err := h.clusterOperator.ListCloudProviders(req.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
		_ = resp.WriteHeaderAndEntity(http.StatusOK, result)
	} else {
		result, err := h.clusterOperator.ListCloudProvidersEx(req.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
		_ = resp.WriteHeaderAndEntity(http.StatusOK, result)
	}
}

func (h *handler) DescribeCloudProvider(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", req.QueryParameter(query.ParameterResourceVersion))
	c, err := h.clusterOperator.GetCloudProviderEx(req.Request.Context(), name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) PreCheckCloudProvider(req *restful.Request, resp *restful.Response) {
	cp := &v1.CloudProvider{}
	err := req.ReadEntity(cp)
	if err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}

	provider, err := clustermanage.GetProvider(clustermanage.Operator{ClusterReader: h.clusterOperator, CloudProviderReader: h.clusterOperator}, *cp)
	if err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}
	check, err := provider.PreCheck(req.Request.Context())
	if err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}
	if !check {
		restplus.HandleBadRequest(resp, req, errors.New("preCheck failed"))
		return
	}

	resp.WriteHeader(http.StatusOK)
}

func (h *handler) CreateCloudProvider(req *restful.Request, resp *restful.Response) {
	cp := &v1.CloudProvider{}
	ctx := req.Request.Context()
	if err := req.ReadEntity(cp); err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	if err := h.providerValidate(req.Request.Context(), cp); err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}

	provider, err := clustermanage.GetProvider(clustermanage.Operator{ClusterReader: h.clusterOperator, CloudProviderReader: h.clusterOperator}, *cp)
	if err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}
	check, err := provider.PreCheck(ctx)
	if err != nil {
		restplus.HandleBadRequest(resp, req, fmt.Errorf("preCheck connect failed: %v", err))
		return
	}
	if !check {
		restplus.HandleBadRequest(resp, req, errors.New("preCheck failed"))
		return
	}

	cp, err = h.clusterOperator.CreateCloudProvider(req.Request.Context(), cp)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}

	_ = resp.WriteHeaderAndEntity(http.StatusCreated, cp)
}

func (h *handler) providerValidate(ctx context.Context, cp *v1.CloudProvider) error {
	if cp.SSH.PrivateKey != "" && cp.SSH.Password != "" {
		return errors.New("can't specify both password and privateKey")
	}
	return nil
}

func (h *handler) UpdateCloudProvider(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	cp := &v1.CloudProvider{}
	if err := req.ReadEntity(cp); err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}

	if err := h.providerValidate(req.Request.Context(), cp); err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}

	dryRun := query.GetBoolValueWithDefault(req, query.ParamDryRun, false)

	if name != cp.Name {
		restplus.HandleBadRequest(resp, req, errors.New("name in url path not same with body"))
		return
	}

	oldCP, err := h.clusterOperator.GetCloudProviderEx(req.Request.Context(), name, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleBadRequest(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}

	if !dryRun {
		cp.Status = oldCP.Status // ignore status update from client
		cp, err = h.clusterOperator.UpdateCloudProvider(req.Request.Context(), cp)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, cp)
}

func (h *handler) SyncCloudProvider(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	dryRun := query.GetBoolValueWithDefault(req, query.ParamDryRun, false)
	cp, err := h.clusterOperator.GetCloudProviderEx(req.Request.Context(), name, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleBadRequest(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}

	if !dryRun {
		conditionReady := cloudprovidercontroller.NewCondition(v1.CloudProviderReady, v1.ConditionFalse, v1.CloudProviderSyncing, "user triggered sync")
		cloudprovidercontroller.SetCondition(&cp.Status, *conditionReady)
		// update annotation to trigger sync
		if cp.Annotations == nil {
			cp.Annotations = make(map[string]string)
		}
		cp.Annotations[common.AnnotationProviderSyncTime] = time.Now().Format(time.RFC3339)
		_, err = h.clusterOperator.UpdateCloudProvider(req.Request.Context(), cp)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
	}
	resp.WriteHeader(http.StatusOK)
}

func (h *handler) DeleteCloudProvider(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	dryRun := query.GetBoolValueWithDefault(req, query.ParamDryRun, false)
	_, err := h.clusterOperator.GetCloudProviderEx(req.Request.Context(), name, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleBadRequest(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}
	if !dryRun {
		err = h.clusterOperator.DeleteCloudProvider(req.Request.Context(), name)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
	}
	resp.WriteHeader(http.StatusOK)
}

func (h *handler) watchCloudProvider(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.clusterOperator.WatchCloudProviders(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("CloudProvider"), req, resp, timeout)
}

func (h *handler) CreateRegistry(req *restful.Request, resp *restful.Response) {
	reg := &v1.Registry{}
	err := req.ReadEntity(reg)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	if err = h.registryValidate(req.Request.Context(), reg); err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}

	reg, err = h.clusterOperator.CreateRegistry(req.Request.Context(), reg)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}

	_ = resp.WriteHeaderAndEntity(http.StatusCreated, reg)
}

func (h *handler) UpdateRegistry(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	reg := &v1.Registry{}
	if err := req.ReadEntity(reg); err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}

	if err := h.registryValidate(req.Request.Context(), reg); err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}

	dryRun := query.GetBoolValueWithDefault(req, query.ParamDryRun, false)

	if name != reg.Name {
		restplus.HandleBadRequest(resp, req, errors.New("name in url path not same with body"))
		return
	}

	_, err := h.clusterOperator.GetRegistryEx(req.Request.Context(), name, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleBadRequest(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}

	if !dryRun {
		reg, err = h.clusterOperator.UpdateRegistry(req.Request.Context(), reg)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, reg)
}

func (h *handler) DescribeRegistry(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", req.QueryParameter(query.ParameterResourceVersion))
	reg, err := h.clusterOperator.GetRegistryEx(req.Request.Context(), name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, reg)
}

func (h *handler) ListRegistry(req *restful.Request, resp *restful.Response) {
	q := query.ParseQueryParameter(req)
	ctx := req.Request.Context()

	if q.Watch {
		h.watchRegistries(req, resp, q)
		return
	}
	if clientrest.IsInformerRawQuery(req.Request) {
		result, err := h.clusterOperator.ListRegistries(ctx, q)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
		_ = resp.WriteHeaderAndEntity(http.StatusOK, result)
	} else {
		result, err := h.clusterOperator.ListRegistriesEx(ctx, q)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
		_ = resp.WriteHeaderAndEntity(http.StatusOK, result)
	}
}

func (h *handler) watchRegistries(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.clusterOperator.WatchRegistries(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, v1.SchemeGroupVersion.WithKind("Registry"), req, resp, timeout)
}

func (h *handler) DeleteRegistry(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	dryRun := query.GetBoolValueWithDefault(req, query.ParamDryRun, false)
	_, err := h.clusterOperator.GetRegistryEx(req.Request.Context(), name, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleBadRequest(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}
	if !dryRun {
		err = h.clusterOperator.DeleteRegistry(req.Request.Context(), name)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
	}
	resp.WriteHeader(http.StatusOK)
}

func (h *handler) registryValidate(_ context.Context, cp *v1.Registry) error {
	if cp.Scheme == "" {
		return fmt.Errorf("scheme cannot be empty")
	}
	if cp.Scheme != "http" && cp.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}
	if cp.Host == "" {
		return fmt.Errorf("scheme must be http or https")
	}
	if cp.CA != "" {
		p, _ := pem.Decode([]byte(cp.CA))
		if p == nil {
			return fmt.Errorf("invalidate certificate")
		}
	}
	return nil
}

func (h *handler) getCluster(clusterName string) (*rest.Config, *kubernetes.Clientset, error) {
	clu, err := h.clusterOperator.GetCluster(context.TODO(), clusterName)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			return nil, nil, restful.ServiceError{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("cluster %s not exists", clusterName),
				Header:  nil,
			}
		}
		logger.Errorf("get cluster %s failed: %v", clusterName, err)
		return nil, nil, err
	}
	if clu.KubeConfig == nil {
		return nil, nil, restful.ServiceError{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("cluster %s clientset not init", clusterName),
			Header:  nil,
		}
	}

	return client.FromKubeConfig(clu.KubeConfig)
}

func (h *handler) execPod(
	request *restful.Request,
	response *restful.Response,
	rc *rest.Config,
	namespace,
	pod,
	container string,
	cmd []string) {
	wsConn, err := upGrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		logger.Errorf("upgrade err: %v", err)
		return
	}
	session := &sshutils.TerminalSession{
		Conn:     wsConn,
		SizeChan: make(chan remotecommand.TerminalSize),
	}
	t := sshutils.NewTerminaler(rc)
	err = t.StartProcess(namespace, pod, container, cmd, session)
	if err != nil {
		logger.Errorf("start process err: %v", err)
		session.Close(2, err.Error())
		return
	}
	session.Close(1, "Process exited")
}
