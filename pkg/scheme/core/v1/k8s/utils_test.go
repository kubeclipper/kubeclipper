package k8s

import (
	"fmt"
	"testing"
)

func Test_extractJoinCommands(t *testing.T) {
	output := `[2025-05-12T11:19:03+08:00] + /usr/bin/bash -c kubeadm reset -f && rm -rf /var/lib/etcd

[reset] Reading configuration from the "kubeadm-config" ConfigMap in namespace "kube-system"...
[reset] Use 'kubeadm init phase upload-config --config your-config-file' to re-upload it.
[preflight] Running pre-flight checks
W0512 11:19:03.475785   19977 reset.go:137] [reset] Unable to fetch the kubeadm-config ConfigMap from cluster: failed to get config map: Get "https://apiserver.cluster.local:6443/api/v1/namespaces/kube-system/configmaps/kubeadm-config?timeout=10s": dial tcp 192.168.10.50:6443: connect: connection refused
W0512 11:19:03.476278   19977 removeetcdmember.go:106] [reset] No kubeadm config, using etcd pod spec to get data directory
[reset] Stopping the kubelet service
[reset] Unmounting mounted directories in "/var/lib/kubelet"
[reset] Deleting contents of directories: [/etc/kubernetes/manifests /var/lib/kubelet /etc/kubernetes/pki]
[reset] Deleting files: [/etc/kubernetes/admin.conf /etc/kubernetes/super-admin.conf /etc/kubernetes/kubelet.conf /etc/kubernetes/bootstrap-kubelet.conf /etc/kubernetes/controller-manager.conf /etc/kubernetes/scheduler.conf]

The reset process does not perform cleanup of CNI plugin configuration,
network filtering rules and kubeconfig files.

For information on how to perform this cleanup manually, please see:
    https://k8s.io/docs/reference/setup-tools/kubeadm/kubeadm-reset/

[2025-05-12T11:19:05+08:00] + /bin/bash -c ps -ef | grep kube-proxy | grep -v grep | awk '{print $2}'

[2025-05-12T11:19:05+08:00] + /bin/bash -c ps -ef | grep kube-apiserver | grep -v grep | awk '{print $2}'

[2025-05-12T11:19:05+08:00] + /bin/bash -c ps -ef | grep kube-controller | grep -v grep | awk '{print $2}'

[2025-05-12T11:19:05+08:00] + /bin/bash -c ps -ef | grep kube-scheduler | grep -v grep | awk '{print $2}'

[2025-05-12T11:19:05+08:00] + /bin/bash -c ps -ef | grep containerd-shim | grep -v grep | awk '{print $2}'

[2025-05-12T11:19:05+08:00] + /bin/bash -c ps -ef | grep etcd | grep -v grep | grep -v "/usr" | awk '{print $2}'

[2025-05-12T11:19:05+08:00] + /usr/bin/kubeadm init --config /tmp/.k8s/kubeadm.yaml --upload-certs

[init] Using Kubernetes version: v1.33.0
[preflight] Running pre-flight checks
W0512 11:19:05.602581   20232 initconfiguration.go:126] Usage of CRI endpoints without URL scheme is deprecated and can cause kubelet errors in the future. Automatically prepending scheme "unix" to the "criSocket" with value "/run/containerd/containerd.sock". Please update your configuration!
	[WARNING SystemVerification]: cgroups v1 support is in maintenance mode, please migrate to cgroups v2
	[WARNING Hostname]: hostname "kc-1" could not be reached
	[WARNING Hostname]: hostname "kc-1": lookup kc-1 on 192.168.10.4:53: no such host
[preflight] Pulling images required for setting up a Kubernetes cluster
[preflight] This might take a minute or two, depending on the speed of your internet connection
[preflight] You can also perform this action beforehand using 'kubeadm config images pull'
[certs] Using certificateDir folder "/etc/kubernetes/pki"
[certs] Generating "ca" certificate and key
[certs] Generating "apiserver" certificate and key
[certs] apiserver serving cert is signed for DNS names [apiserver.cluster.local kc-1 kubernetes kubernetes.default kubernetes.default.svc kubernetes.default.svc.cluster.local] and IPs [10.96.0.1 192.168.10.50]
[certs] Generating "apiserver-kubelet-client" certificate and key
[certs] Generating "front-proxy-ca" certificate and key
[certs] Generating "front-proxy-client" certificate and key
[certs] Generating "etcd/ca" certificate and key
[certs] Generating "etcd/server" certificate and key
[certs] etcd/server serving cert is signed for DNS names [kc-1 localhost] and IPs [192.168.10.50 127.0.0.1 ::1]
[certs] Generating "etcd/peer" certificate and key
[certs] etcd/peer serving cert is signed for DNS names [kc-1 localhost] and IPs [192.168.10.50 127.0.0.1 ::1]
[certs] Generating "etcd/healthcheck-client" certificate and key
[certs] Generating "apiserver-etcd-client" certificate and key
[certs] Generating "sa" key and public key
[kubeconfig] Using kubeconfig folder "/etc/kubernetes"
[kubeconfig] Writing "admin.conf" kubeconfig file
[kubeconfig] Writing "super-admin.conf" kubeconfig file
[kubeconfig] Writing "kubelet.conf" kubeconfig file
[kubeconfig] Writing "controller-manager.conf" kubeconfig file
[kubeconfig] Writing "scheduler.conf" kubeconfig file
[etcd] Creating static Pod manifest for local etcd in "/etc/kubernetes/manifests"
[control-plane] Using manifest folder "/etc/kubernetes/manifests"
[control-plane] Creating static Pod manifest for "kube-apiserver"
[control-plane] Creating static Pod manifest for "kube-controller-manager"
[control-plane] Creating static Pod manifest for "kube-scheduler"
[kubelet-start] Writing kubelet environment file with flags to file "/var/lib/kubelet/kubeadm-flags.env"
[kubelet-start] Writing kubelet configuration to file "/var/lib/kubelet/config.yaml"
[kubelet-start] Starting the kubelet
[wait-control-plane] Waiting for the kubelet to boot up the control plane as static Pods from directory "/etc/kubernetes/manifests"
[kubelet-check] Waiting for a healthy kubelet at http://127.0.0.1:10248/healthz. This can take up to 4m0s
[kubelet-check] The kubelet is healthy after 1.503094476s
[control-plane-check] Waiting for healthy control plane components. This can take up to 4m0s
[control-plane-check] Checking kube-apiserver at https://192.168.10.50:6443/livez
[control-plane-check] Checking kube-controller-manager at https://127.0.0.1:10257/healthz
[control-plane-check] Checking kube-scheduler at https://127.0.0.1:10259/livez
[control-plane-check] kube-controller-manager is healthy after 2.549832845s
[control-plane-check] kube-scheduler is healthy after 4.108102555s
[control-plane-check] kube-apiserver is healthy after 7.003896696s
[upload-config] Storing the configuration used in ConfigMap "kubeadm-config" in the "kube-system" Namespace
[kubelet] Creating a ConfigMap "kubelet-config" in namespace kube-system with the configuration for the kubelets in the cluster
[upload-certs] Storing the certificates in Secret "kubeadm-certs" in the "kube-system" Namespace
[upload-certs] Using certificate key:
bcd603ea2ef3d514af4c9126cb8a65344c322cf4b59149e5abdb21a6ae344346
[mark-control-plane] Marking the node kc-1 as control-plane by adding the labels: [node-role.kubernetes.io/control-plane node.kubernetes.io/exclude-from-external-load-balancers]
[mark-control-plane] Marking the node kc-1 as control-plane by adding the taints [node-role.kubernetes.io/control-plane:NoSchedule]
[bootstrap-token] Using token: w5ko5a.bo1jqizb0fouk39k
[bootstrap-token] Configuring bootstrap tokens, cluster-info ConfigMap, RBAC Roles
[bootstrap-token] Configured RBAC rules to allow Node Bootstrap tokens to get nodes
[bootstrap-token] Configured RBAC rules to allow Node Bootstrap tokens to post CSRs in order for nodes to get long term certificate credentials
[bootstrap-token] Configured RBAC rules to allow the csrapprover controller automatically approve CSRs from a Node Bootstrap Token
[bootstrap-token] Configured RBAC rules to allow certificate rotation for all node client certificates in the cluster
[bootstrap-token] Creating the "cluster-info" ConfigMap in the "kube-public" namespace
[kubelet-finalize] Updating "/etc/kubernetes/kubelet.conf" to point to a rotatable kubelet client certificate and key
[addons] Applied essential addon: CoreDNS
[addons] Applied essential addon: kube-proxy

Your Kubernetes control-plane has initialized successfully!

To start using your cluster, you need to run the following as a regular user:

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config

Alternatively, if you are the root user, you can run:

  export KUBECONFIG=/etc/kubernetes/admin.conf

You should now deploy a pod network to the cluster.
Run "kubectl apply -f [podnetwork].yaml" with one of the options listed at:
  https://kubernetes.io/docs/concepts/cluster-administration/addons/

You can now join any number of control-plane nodes running the following command on each as root:

  kubeadm join apiserver.cluster.local:6443 --token w5ko5a.bo1jqizb0fouk39k \
	--discovery-token-ca-cert-hash sha256:9315d54759f9f65ff4317866141b2301daf44d1c177f5c5fdbc69484becfcefb \
	--control-plane --certificate-key bcd603ea2ef3d514af4c9126cb8a65344c322cf4b59149e5abdb21a6ae344346

Please note that the certificate-key gives access to cluster sensitive data, keep it secret!
As a safeguard, uploaded-certs will be deleted in two hours; If necessary, you can use
"kubeadm init phase upload-certs --upload-certs" to reload certs afterward.

Then you can join any number of worker nodes by running the following on each as root:

kubeadm join apiserver.cluster.local:6443 --token w5ko5a.bo1jqizb0fouk39k \
	--discovery-token-ca-cert-hash sha256:9315d54759f9f65ff4317866141b2301daf44d1c177f5c5fdbc69484becfcefb`
	targetMasterCmd := `kubeadm join apiserver.cluster.local:6443 --token w5ko5a.bo1jqizb0fouk39k --discovery-token-ca-cert-hash sha256:9315d54759f9f65ff4317866141b2301daf44d1c177f5c5fdbc69484becfcefb --control-plane --certificate-key bcd603ea2ef3d514af4c9126cb8a65344c322cf4b59149e5abdb21a6ae344346`
	targetWorkerCmd := `kubeadm join apiserver.cluster.local:6443 --token w5ko5a.bo1jqizb0fouk39k --discovery-token-ca-cert-hash sha256:9315d54759f9f65ff4317866141b2301daf44d1c177f5c5fdbc69484becfcefb`

	masterCmd, workerCmd := extractJoinCommands(output)

	fmt.Println("Master Join Command:")
	fmt.Println(masterCmd)
	fmt.Println("\nWorker Join Command:")
	fmt.Println(workerCmd)

	if masterCmd != targetMasterCmd {
		t.Fatalf("master cmd is not equal, expect %s, but got %s", targetMasterCmd, masterCmd)
	}
	if workerCmd != targetWorkerCmd {
		t.Fatalf("worker cmd is not equal, expect %s, but got %s", targetWorkerCmd, workerCmd)
	}

}
