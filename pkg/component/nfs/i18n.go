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

package nfsprovisioner

import "github.com/kubeclipper/kubeclipper/pkg/component"

func initI18nForComponentMeta() error {
	return component.AddI18nMessages(component.I18nMessages{
		{
			ID:      "nfs.metaTitle",
			English: "NFS Setting",
			Chinese: "NFS 设置",
		},
		{
			ID:      "nfs.serverAddr",
			English: "ServerAddr",
			Chinese: "服务地址",
		},
		{
			ID:      "nfs.sharedPath",
			English: "SharedPath",
			Chinese: "共享路径",
		},
		{
			ID:      "nfs.scName",
			English: "StorageClassName",
			Chinese: "存储类名",
		},
		{
			ID:      "nfs.isDefaultSC",
			English: "IsDefault",
			Chinese: "是否默认存储类",
		},
		{
			ID:      "nfs.reclaimPolicy",
			English: "ReclaimPolicy",
			Chinese: "回收策略",
		},
		{
			ID:      "nfs.archiveOnDelete",
			English: "ArchiveOnDelete",
			Chinese: "删除时是否归档",
		},
		{
			ID:      "nfs.mountOptions",
			English: "MountOptions",
			Chinese: "挂载选项",
		},
		{
			ID:      "nfs.replicas",
			English: "Replicas",
			Chinese: "副本数",
		},
		{
			ID:      "nfs.imageRepoMirror",
			English: "NFS Image Repository Mirror",
			Chinese: "NFS 镜像仓库代理",
		},
	})
}
