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

package metallb

import "github.com/kubeclipper/kubeclipper/pkg/component"

func initI18nForComponentMeta() error {
	return component.AddI18nMessages(component.I18nMessages{
		{
			ID:      "metallb.metaTitle",
			English: "metallb Setting",
			Chinese: "metallb 设置",
		},
		{
			ID:      "metallb.mode",
			English: "Mode",
			Chinese: "模式",
		},
		{
			ID:      "metallb.addresses",
			English: "IPAddressPool",
			Chinese: "IP 地址池",
		},
		{
			ID:      "metallb.imageRepoMirror",
			English: "metallb Image Repository Mirror",
			Chinese: "metallb 镜像仓库代理",
		},
	})
}
