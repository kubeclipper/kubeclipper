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

package framework

import "time"

var (
	// Default timeouts to be used in TimeoutContext
	clusterInstall      = 20 * time.Minute
	clusterInstallShort = 5 * time.Minute
	clusterDelete       = 10 * time.Minute
	commonTimeout       = 5 * time.Minute
)

// TimeoutContext contains timeout settings for several actions.
type TimeoutContext struct {
	// ClusterInstall is how long to wait for the pod to be started.
	// Use it in create ha cluster case
	ClusterInstall time.Duration

	// ClusterInstallShort is same as `ClusterInstall`, but shorter.
	// Use it in create aio cluster case
	ClusterInstallShort time.Duration

	// ClusterDelete is how long to wait for the cluster to be deleted.
	ClusterDelete time.Duration

	CommonTimeout time.Duration
}

// NewTimeoutContextWithDefaults returns a TimeoutContext with default values.
func NewTimeoutContextWithDefaults() *TimeoutContext {
	return &TimeoutContext{
		ClusterInstall:      clusterInstall,
		ClusterInstallShort: clusterInstallShort,
		ClusterDelete:       clusterDelete,
		CommonTimeout:       commonTimeout,
	}
}
