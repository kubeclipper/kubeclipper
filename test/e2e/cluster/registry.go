package cluster

import (
	"context"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func initRegistries() []*corev1.Registry {
	return []*corev1.Registry{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Registry",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-http",
				Annotations: map[string]string{
					common.AnnotationDescription: "http://kubeclipper.io",
				},
			},
			RegistrySpec: corev1.RegistrySpec{
				Scheme: "http",
				Host:   "kubeclipper.io",
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Registry",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-https-skip",
				Annotations: map[string]string{
					common.AnnotationDescription: "https://skip.kubeclipper.io",
				},
			},
			RegistrySpec: corev1.RegistrySpec{
				Scheme:     "https",
				Host:       "skip.kubeclipper.io",
				SkipVerify: true,
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Registry",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-https-not-skip",
				Annotations: map[string]string{
					common.AnnotationDescription: "https://not-skip.kubeclipper.io",
				},
			},
			RegistrySpec: corev1.RegistrySpec{
				Scheme:     "https",
				Host:       "not-skip.kubeclipper.io",
				SkipVerify: false,
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Registry",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-https-not-skip-ca",
				Annotations: map[string]string{
					common.AnnotationDescription: "https://not-skip-ca.kubeclipper.io",
				},
			},
			RegistrySpec: corev1.RegistrySpec{
				Scheme:     "https",
				Host:       "not-skip-ca.kubeclipper.io",
				SkipVerify: false,
				CA:         "-----BEGIN RSA PRIVATE KEY-----\nProc-Type: 4,ENCRYPTED\nDEK-Info: DES-EDE3-CBC,1129B2A58F917A50\n\njrdxQ7hg2prOXcaeUn2hvmfELKAhDLpD+LAEN9QoM9H1WXv/XparNqwOy0uMw8k8\ngMeBNhx6ViGWtw4Lu5rJOYSzu9aLwfRgP/aD8fqa2NEdbLFIFmOopOm/Rws2Xr5P\nyEm9OQK542ca0GU1A+SyJZsI/3xfoTLPkW2QxxZyNvew0C1jaDq/NO3xI9V8oC8T\ntFjTfB/DFT4FUzH5UYl5o3DyK+T+nNuRtVQr5g8GJG2saY4BGcfLci+XFw3xq62T\nd6yIQ43UeNrOYyjmgoGaked2+SfzajT5OspOH4QW4SXaDhWbqqIvUyxOIGg3uFU1\n4Cx7xGv/UQCFtArCQQ0f0YrFzIQwtUBikMi25DoL5/RO1NS9Lw9E8OYucmQ99q9M\nO2i80f5k3y2VHj2l5lcsmAQc5c0ONPoNd3MxhT4MRaWuuFitN7IRTyPeTjfnwOmU\n2YXNBVREIDNNOZ6YyA2elo5HoqinE65zHjcNQK+ntv68Kq3wQx4grDjpJDuGMroo\nU3OjY0R9t6OSKy+sg6ct1AwfsWVsWq7Www5Aw7RiaCOWyzViX9joAUCrgIP4nVEX\nie5n2YNsLPm8eRm0XdLUFT0Xz4bmq+RRvkKmKBjkEjZ5L9nwPMmb+vaqgq0DlDRJ\nDJpJaWJ0N7xSD5oISI5uoeZ+l0uUhT+McR9E93ubH4Z7LQEDvUsK0PVSUJdev3ai\nDOMxC7dkLpUPOl/5Wkr1avAk6QF8GsYgR9endcEI5wGKrbXz9CQ1Prubeg5VfVMv\nWLDbfFV8zS0J//tkhii7GKaknp2WMLSs1/YgNFYZvLoTnZT+p812dGFULNUXxbNa\nYtqau2LO3yO2WOEfY599sIn/G0LxCbHcxQpJWExAR6n8i00KyuWmSiFEpHP4kRUh\nFzpQEW3Fdi/5cFIYRtzFUzS+skQoSpMAH5w8Nm5s4gDfTFhf/46T5WiLr30Js4pd\n3Le/u4qhe7F16eo5OonEfZ1UY4NFw0w5LLNt3zaOa5/+YLELaE037nn4m/EmuFoK\n1XfIlKrMEtm0ZI535p2sIbfAIHkzre8Xtb2DN3c4ylVKoYukBdMbcnYjvyyP9FON\nw4iq7xuoXnNWYVMnpJU/Ed15QDrMgtcOfYlWkZ/LY6hYOP8ss4T35RTGIY0wGYkt\nNGxcNIotT0h5l17iyfpxS1EDrheYNGLNrDKx4Vx0+yJqEOV7Yw/YBGBkQjDiuXQ0\n6SdDriIfy/dBTTANWL9E3cGIi6b586OZ49/fF0BxqYkga1ufqg6sg/qRPPeJ0M+W\nvMdhgY4zxT1tbSB61EBiNc1mYz7FKiHhLNJ8E8eOP4KeVE79IZIiDTRwvzuoeJAg\nvn6wItPbx9w7INtJXKk3BqRUvcPFnGF7X8RM9a51dMmVBvRjaLl60ZZTx0AwS6pX\n4QV7imIxBD5FGTT6+Nt8cfbvHIZR14rtuRW1Wsgj+/8lfx2k66QVTGrbsgIkyE9v\n65aJ8ikMehv6drYAcCWlBckNhOZLA2DOoIWH2kKWZa6dcsibNFlUPUcFH229ynNw\n2JGAU+nAiBVWJbTChtjfpJjwYhX2dX7BybnVKwCGm2DqjGa80oOVQA==\n-----END RSA PRIVATE KEY-----",
			},
		},
	}
}

func afterEachDeleteRegistries(ctx context.Context, f *framework.Framework) {
	f.AddAfterEach("delete registries", func(f *framework.Framework, failed bool) {
		registries, err := f.Client.ListRegistries(ctx, kc.Queries{})
		framework.ExpectNoError(err)

		ginkgo.By(" delete registries ")

		if len(registries.Items) > 0 {
			for _, reg := range registries.Items {
				_ = f.Client.DeleteRegistry(ctx, reg.Name)
			}
		}
	})
}
