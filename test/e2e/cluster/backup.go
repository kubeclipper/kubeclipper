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

package cluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func initBackup(nodes *corev1.Node, name string, bp string) *corev1.Backup {
	return &corev1.Backup{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: corev1.BackupStatus{},
		ClusterNodes: map[string]string{
			nodes.Status.Ipv4DefaultIP: nodes.Status.NodeInfo.Hostname,
		},
		PreferredNode:   "",
		BackupPointName: bp,
	}
}

// TODO: use os.CreateTemp instead /tmp
func initBackUpPoint(bp string) *corev1.BackupPoint {
	return &corev1.BackupPoint{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BackupPoint",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: bp,
		},
		StorageType: "FS",
		Description: "",
		FsConfig: &corev1.FsConfig{
			BackupRootDir: "/tmp",
		},
		S3Config: nil,
	}
}

func initRecovery(backupName string) *corev1.Recovery {
	return &corev1.Recovery{
		TypeMeta:      metav1.TypeMeta{},
		ObjectMeta:    metav1.ObjectMeta{},
		UseBackupName: backupName,
		Description:   "",
	}
}
