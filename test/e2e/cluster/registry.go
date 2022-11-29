package cluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func initRegistry(name string) *corev1.Registry {
	return &corev1.Registry{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Registry",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RegistrySpec: corev1.RegistrySpec{
			Scheme:     "https",
			Host:       "registry.kubeclipper.io",
			SkipVerify: true,
		},
	}
}
