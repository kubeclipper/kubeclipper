package cluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func initBackup(node *corev1.Node, name string, bp string) *corev1.Backup {
	return &corev1.Backup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Backup",
			APIVersion: corev1.SchemeGroupVersion.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				common.AnnotationDescription: "",
			},
		},
		Status: corev1.BackupStatus{},
		ClusterNodes: map[string]string{
			node.Status.Ipv4DefaultIP: node.Status.NodeInfo.Hostname,
		},
		PreferredNode:   node.Name,
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
