package constatns

const (
	DefaultAdminUser     = "admin"
	DefaultAdminUserPass = "Thinkbig1"
)

const (
	DeployConfigConfigMapName = "deploy-config"
	DeployConfigConfigMapKey  = "DeployConfig"

	KcCertsConfigMapName     = "kc-ca"
	KcEtcdCertsConfigMapName = "kc-etcd"
	KcNatsCertsConfigMapName = "kc-nats"
)

const (
	ClusterServiceSubnet = "10.96.0.0/12"
	ClusterPodSubnet     = "172.25.0.0/16"
)
