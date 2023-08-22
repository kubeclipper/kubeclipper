package cni

import (
	"bytes"
	"testing"

	"github.com/kubeclipper/kubeclipper/pkg/constatns"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func TestCNI_renderCalicoTo(t *testing.T) {
	tests := []struct {
		name    string
		stepper CalicoRunnable
		wantW   string
		wantErr bool
	}{
		{
			name: "base",
			stepper: CalicoRunnable{
				BaseCni: BaseCni{
					DualStack:   true,
					PodIPv4CIDR: constatns.ClusterPodSubnet,
					PodIPv6CIDR: "aaa:bbb",
					CNI: v1.CNI{
						LocalRegistry: "172.0.0.1:5000",
						Type:          "calico",
						Version:       "v3.26.1",
						Calico: &v1.Calico{
							IPv4AutoDetection: "first-found",
							IPv6AutoDetection: "first-found",
							Mode:              "Overlay-Vxlan-All",
							IPManger:          true,
							MTU:               1440,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt.stepper.NodeAddressDetectionV4 = ParseNodeAddressDetection(tt.stepper.Calico.IPv4AutoDetection)
		tt.stepper.NodeAddressDetectionV6 = ParseNodeAddressDetection(tt.stepper.Calico.IPv6AutoDetection)
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			err := tt.stepper.renderCalicoTo(w)
			if err != nil {
				t.Errorf("renderCalicoTo() error = %v", err)
				return
			}
			t.Log(w.String())
		})
	}
}
