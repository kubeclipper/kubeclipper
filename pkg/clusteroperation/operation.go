/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package clusteroperation

import (
	"fmt"

	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type Options struct {
	cluster           *v1.Cluster
	pendingOperation  v1.PendingOperation
	extra             *component.ExtraMetadata
	additionalOptions *AdditionalOptions // additional information
	operator          cluster.Operator
}

// AdditionalOptions some types of operations require additional information, which can be put here
type AdditionalOptions struct{}

// Interface all operation types must implement this interface
type Interface interface {
	Builder() (*v1.Operation, error)
}

type OptionInterface interface {
}

// BuildOperationAdapter create an operation based on the operation type
func BuildOperationAdapter(
	cluster *v1.Cluster, pendingOp v1.PendingOperation,
	extra *component.ExtraMetadata, addition *AdditionalOptions,
	operator cluster.Operator,
) (*v1.Operation, error) {
	options := Options{
		cluster:           cluster,
		pendingOperation:  pendingOp,
		extra:             extra,
		additionalOptions: addition,
		operator:          operator,
	}

	var instance Interface
	switch pendingOp.OperationType {
	case v1.OperationAddNodes, v1.OperationRemoveNodes:
		instance = NewNodeOperation(options)
	case v1.OperationInstallComponents, v1.OperationUninstallComponents:
	case v1.OperationCreateCluster:
	case v1.OperationDeleteCluster:
	case v1.OperationUpgradeCluster:
	case v1.OperationBackupCluster:
	case v1.OperationDeleteBackup:
	case v1.OperationRecoverCluster:
	case v1.OperationUpdateCertification:
	case v1.OperationUpdateAPIServerCertification:
		// TODO support all operations
	default:
		return &v1.Operation{}, fmt.Errorf("unsupported %s operation type", pendingOp.OperationType)
	}

	return instance.Builder()
}

// SupportConcurrent whether concurrency is supported
func SupportConcurrent(opType string) bool {
	switch opType {
	case v1.OperationAddNodes, v1.OperationRemoveNodes:
		return true
	default:
		return false
	}
}

// GetClusterPhase get the cluster phase based on the type of operation
func GetClusterPhase(opType string) v1.ClusterPhase {
	switch opType {
	case v1.OperationCreateCluster:
		return v1.ClusterInstalling
	case v1.OperationDeleteCluster:
		return v1.ClusterTerminating
	case v1.OperationUpgradeCluster:
		return v1.ClusterUpgrading
	case v1.OperationBackupCluster, v1.OperationDeleteBackup:
		return v1.ClusterBackingUp
	case v1.OperationRecoverCluster:
		return v1.ClusterRestoring
	default:
		// OperationAddNodes,OperationRemoveNodes
		// OperationInstallComponents,OperationUninstallComponents
		// OperationUpdateCertification
		return v1.ClusterUpdating
	}
}
