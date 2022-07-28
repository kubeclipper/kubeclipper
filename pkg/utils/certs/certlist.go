package certs

const (
	defaultPKI = "/etc/kubernetes/pki"
	etcdPKI    = "/etc/kubernetes/pki/etcd"
)

type Certification []*Config

func GetDefaultCerts() Certification {
	return Certification{
		RootCACert(),
		APIServerCert(),
		KubeletClientCert(),
		FrontProxyCACert(),
		FrontProxyClientCert(),
		EtcdCACert(),
		EtcdServerCert(),
		EtcdPeerCert(),
		EtcdHealthcheckCert(),
		EtcdAPIClientCert(),
	}
}

func RootCACert() *Config {
	return &Config{
		Path:       defaultPKI,
		BaseName:   "ca",
		CAName:     "",
		CommonName: "ca",
	}
}

func APIServerCert() *Config {
	return &Config{
		Path:       defaultPKI,
		BaseName:   "apiserver",
		CAName:     "ca",
		CommonName: "apiserver",
	}
}

func KubeletClientCert() *Config {
	return &Config{
		Path:       defaultPKI,
		BaseName:   "apiserver-kubelet-client",
		CAName:     "ca",
		CommonName: "apiserver-kubelet-client",
	}
}

func FrontProxyCACert() *Config {
	return &Config{
		Path:       defaultPKI,
		BaseName:   "front-proxy-ca",
		CAName:     "",
		CommonName: "front-proxy-ca",
	}
}

func FrontProxyClientCert() *Config {
	return &Config{
		Path:       defaultPKI,
		BaseName:   "front-proxy-client",
		CAName:     "front-proxy-ca",
		CommonName: "front-proxy-client",
	}
}

func EtcdCACert() *Config {
	return &Config{
		Path:       etcdPKI,
		BaseName:   "ca",
		CAName:     "",
		CommonName: "etcd-ca",
	}
}

func EtcdServerCert() *Config {
	// name: etcd-server
	// baseName: etcd/server
	// fileName: server
	return &Config{
		Path:       etcdPKI,
		BaseName:   "server",
		CAName:     "ca",
		CommonName: "etcd-server",
	}
}

func EtcdPeerCert() *Config {
	return &Config{
		Path:       etcdPKI,
		BaseName:   "peer",
		CAName:     "ca",
		CommonName: "etcd-peer",
	}
}

func EtcdHealthcheckCert() *Config {
	return &Config{
		Path:       etcdPKI,
		BaseName:   "healthcheck-client",
		CAName:     "ca",
		CommonName: "etcd-healthcheck-client",
	}
}

func EtcdAPIClientCert() *Config {
	return &Config{
		Path:       defaultPKI,
		BaseName:   "apiserver-etcd-client",
		CAName:     "ca",
		CommonName: "apiserver-etcd-client",
	}
}
