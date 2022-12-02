package cluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func initCronBackup(cronBackupName, cluster string) *corev1.CronBackup {
	return &corev1.CronBackup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronBackup",
			APIVersion: corev1.SchemeGroupVersion.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: cronBackupName,
			Labels: map[string]string{
				common.LabelCronBackupEnable: "",
			},
		},
		Spec: corev1.CronBackupSpec{
			ClusterName:  cluster,
			Schedule:     "*/1 * * * *",
			MaxBackupNum: 2,
		},
		Status: corev1.CronBackupStatus{},
	}
}
