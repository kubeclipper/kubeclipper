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

package constatns

const (
	DefaultAdminUser     = "admin"
	DefaultAdminUserPass = "Thinkbig1"
)

const (
	DeployConfigConfigMapName = "deploy-config"
	DeployConfigConfigMapKey  = "DeployConfig"

	KcCertsConfigMapName     = "kc-ca"
	KcEtcdCertsConfigMapName = "kc-etcd"
	KcNatsCertsConfigMapName = "kc-nats"
)

const (
	ClusterServiceSubnet = "10.96.0.0/12"
	ClusterPodSubnet     = "172.25.0.0/16"
)
