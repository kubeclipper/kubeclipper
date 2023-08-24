/*
 *
 *  * Copyright 2021 KubeClipper Authors.
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

package k8s

const kubeadmTemplate = `
kind: ClusterConfiguration
apiVersion: kubeadm.k8s.io/{{.ClusterConfigAPIVersion}}
etcd:
  local:
{{with .Etcd.DataDir}}    dataDir: "{{.}}"{{end}}
    extraArgs:
      auto-compaction-retention: '1'
      election-timeout: '1500'
      heartbeat-interval: '300'
      quota-backend-bytes: '8589934592'
      snapshot-count: '5000'
networking:
  serviceSubnet: {{ range .Networking.Services.CIDRBlocks }}{{ . }}{{- end }}
  podSubnet: {{ range .Networking.Pods.CIDRBlocks }}{{ . }}{{- end }}
  dnsDomain: {{.Networking.DNSDomain}}
kubernetesVersion: {{.KubernetesVersion}}
controlPlaneEndpoint: {{.ControlPlaneEndpoint}}
apiServer:
  extraVolumes:
  - name: localtime
    hostPath: "/etc/localtime"
    mountPath: "/etc/localtime"
    readOnly: true
    pathType: File
  certSANs:{{range .CertSANs}}
  - {{.}}{{end}}
controllerManager:
  extraVolumes:
  - name: localtime
    hostPath: "/etc/localtime"
    mountPath: "/etc/localtime"
    readOnly: true
    pathType: File
scheduler:
  extraVolumes:
  - name: localtime
    hostPath: "/etc/localtime"
    mountPath: "/etc/localtime"
    readOnly: true
    pathType: File
{{with .LocalRegistry}}imageRepository: {{.}}{{end}}
{{with .ClusterName}}clusterName: {{.}}{{end}}
{{if gt (len .FeatureGates) 0}}
featureGates:{{range $key,$value := .FeatureGates}}
  {{$key}}: {{$value}}{{end}}{{end}}
---
kind: KubeProxyConfiguration
apiVersion: kubeproxy.config.k8s.io/v1alpha1
mode: {{if eq .Networking.ProxyMode "ipvs"}}ipvs{{else}}iptables{{end}}
{{if eq .Networking.ProxyMode "ipvs"}}{{if .Networking.WorkerNodeVip}}ipvs:
  excludeCIDRs:
  - "{{.Networking.WorkerNodeVip}}/32"{{end}}{{end}}
---
apiVersion: kubelet.config.k8s.io/v1beta1
authentication:
  anonymous:
    enabled: false
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt
kind: KubeletConfiguration
cgroupDriver: systemd
healthzBindAddress: 127.0.0.1
healthzPort: 10248
imageGCHighThresholdPercent: 85
imageGCLowThresholdPercent: 80
imageMinimumGCAge: 2m0s
memorySwap: {}
staticPodPath: /etc/kubernetes/manifests
streamingConnectionIdleTimeout: 0s
syncFrequency: 3s
volumeStatsAggPeriod: 1m
---
apiVersion: kubeadm.k8s.io/{{.ClusterConfigAPIVersion}}
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: {{.AdvertiseAddress}}
  bindPort: 6443
nodeRegistration:
{{- if .Kubelet.IPAsName }}
  name: "{{.Kubelet.NodeIP}}"
{{- end}}
{{- if eq .ContainerRuntime  "containerd"}}
  criSocket: /run/containerd/containerd.sock
{{end}}
  kubeletExtraArgs:
    root-dir: {{.Kubelet.RootDir}}
    node-ip: {{.Kubelet.NodeIP}}
`

const KubeadmJoinTemplate = `apiVersion: kubeadm.k8s.io/{{.ClusterConfigAPIVersion}}
kind: JoinConfiguration
discovery:
  bootstrapToken:
    apiServerEndpoint: {{.ControlPlaneEndpoint}}
    token: {{.BootstrapToken}}
    caCertHashes:
      - {{.CACertHashes}}
  timeout: 5m0s
{{- if .IsControlPlane}}
controlPlane: 
  localAPIEndpoint:
    advertiseAddress: {{.AdvertiseAddress}}
    bindPort: 6443
  certificateKey: {{.CertificateKey}}{{end}}
nodeRegistration:
{{- if .Kubelet.IPAsName }}
  name: "{{.Kubelet.NodeIP}}"
{{- end}}
{{- if eq .ContainerRuntime  "containerd"}}
  criSocket: /run/containerd/containerd.sock
{{end}}
  kubeletExtraArgs:
    root-dir: {{.Kubelet.RootDir}}
    node-ip: {{.Kubelet.NodeIP}}
    resolv-conf: {{.Kubelet.ResolvConf}}
`

const lvscareV111 = `
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  labels:
    component: kube-lvscare
    tier: control-plane
  name: kube-lvscare
  namespace: kube-system
spec:
  containers:
  - args:
    - care
    - --vs
    - {{.WorkerNodeVIP}}:6443
    - --health-path
    - /healthz
    - --health-schem
    - https{{range .Masters}}
    - --rs
    - {{.}}:6443{{end}}
    command:
    - /usr/bin/lvscare
    #image: fanux/lvscare:v1.1.1
    image: {{with .LocalRegistry}}{{.}}/{{end}}fanux/lvscare:v1.1.1
    imagePullPolicy: IfNotPresent
    name: kube-lvscare
    resources: {}
    securityContext:
      privileged: true
    volumeMounts:
    - mountPath: /lib/modules
      name: lib-modules
      readOnly: true
  hostNetwork: true
  priorityClassName: system-cluster-critical
  volumes:
  - hostPath:
      path: /lib/modules
      type: ""
    name: lib-modules
status: {}
`

// KubectlPodTemplate kubectl terminal service yaml template
const KubectlPodTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: kc-kubectl
  name: kc-kubectl
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: kc-kubectl
  template:
    metadata:
      labels:
        k8s-app: kc-kubectl
    spec:
      tolerations:
        - key: CriticalAddonsOnly
          operator: Exists
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
        - key: node-role.kubernetes.io/control-plane
          effect: NoSchedule
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/master
                operator: In
                values:
                - ""
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/control-plane
                operator: In
                values:
                - ""
      serviceAccountName: kc-kubectl
      containers:
        - image: {{with .ImageRegistryAddr}}{{.}}/{{end}}kubeclipper/kubectl
          imagePullPolicy: IfNotPresent
          name: kc-kubectl
          resources:
            requests:
              memory: "100Mi"
              cpu: "100m"
            limits:
              memory: "500Mi"
              cpu: "500m"
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    k8s-app: kc-kubectl
  name: kc-kubectl
  namespace: kube-system
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: kc-kubectl-rolebind
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
subjects:
  - kind: ServiceAccount
    name: kc-kubectl
    namespace: kube-system
`
