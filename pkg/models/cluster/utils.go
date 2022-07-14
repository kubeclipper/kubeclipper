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

package cluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type nodeList []v1.Node

func less(i, j metav1.ObjectMeta) bool {
	t1 := i.CreationTimestamp.Time
	t2 := j.CreationTimestamp.Time
	if t1.After(t2) {
		return true
	} else if t1.Before(t2) {
		return false
	} else {
		return i.Name > j.Name
	}
}

func (l nodeList) Len() int      { return len(l) }
func (l nodeList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l nodeList) Less(i, j int) bool {
	return less(l[i].ObjectMeta, l[j].ObjectMeta)
}

type clusterList []v1.Cluster

func (l clusterList) Len() int      { return len(l) }
func (l clusterList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l clusterList) Less(i, j int) bool {
	return less(l[i].ObjectMeta, l[j].ObjectMeta)
}

type regionList []v1.Region

func (l regionList) Len() int      { return len(l) }
func (l regionList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l regionList) Less(i, j int) bool {
	return less(l[i].ObjectMeta, l[j].ObjectMeta)
}

type backupList []v1.Backup

func (l backupList) Len() int      { return len(l) }
func (l backupList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l backupList) Less(i, j int) bool {
	return less(l[i].ObjectMeta, l[j].ObjectMeta)
}

type recoveryList []v1.Recovery

func (l recoveryList) Len() int      { return len(l) }
func (l recoveryList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l recoveryList) Less(i, j int) bool {
	return less(l[i].ObjectMeta, l[j].ObjectMeta)
}

type domainList []v1.Domain

func (l domainList) Len() int      { return len(l) }
func (l domainList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l domainList) Less(i, j int) bool {
	return less(l[i].ObjectMeta, l[j].ObjectMeta)
}

type recordList []v1.Record

func (l recordList) Len() int      { return len(l) }
func (l recordList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l recordList) Less(i, j int) bool {
	ti := l[i].CreateTime.Time
	tj := l[j].CreateTime.Time
	if ti.After(tj) {
		return true
	} else if ti.Before(tj) {
		return false
	} else {
		return l[i].RR > l[j].RR
	}
}
