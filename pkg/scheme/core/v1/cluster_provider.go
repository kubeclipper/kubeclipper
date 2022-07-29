package v1

const (
	ClusterKubeadm string = "kubeadm"
	// TODO: other cluster type
)

type ProviderSpec struct {
	Name string `json:"name"`
}
