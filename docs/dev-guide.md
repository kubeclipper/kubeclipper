# KubeClipper Development Guide

## 1. Development Environment

### 1.1 Architecture

```txt
Arch
             +------------------+               +-------------------+
             | Ubuntu 24.04 2C4G|               |    Ubuntu 22.04   |
             |   Development    |               | KC-Agent(Optional)|
             +--------+---------+               +----------+--------+
                      |                                    |
                      |                                    |
                      |                                    |
+---------------------+------------------------------------+--------------------+
```

Development Environment：

1. Docker & containerd are both in `/usr/bin/containerd`
2. If deploy cluster, new containerd will be installed in `/usr/local/bin/containerd` by kubeclipper, which could co-exist with old containerd

Config SSH (permit root login with id_rsa) & `apt update -y && apt upgrade -y`

```console
# cat /etc/os-release 
PRETTY_NAME="Ubuntu 24.04.2 LTS"
NAME="Ubuntu"
VERSION_ID="24.04"
VERSION="24.04.2 LTS (Noble Numbat)"
VERSION_CODENAME=noble
ID=ubuntu
ID_LIKE=debian
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"
BUG_REPORT_URL="https://bugs.launchpad.net/ubuntu/"
PRIVACY_POLICY_URL="https://www.ubuntu.com/legal/terms-and-policies/privacy-policy"
UBUNTU_CODENAME=noble
LOGO=ubuntu-logo

# uname -a
Linux VM-4-12-ubuntu 6.8.0-51-generic #52-Ubuntu SMP PREEMPT_DYNAMIC Thu Dec  5 13:09:44 UTC 2024 x86_64 x86_64 x86_64 GNU/Linux
```

### 1.2 Install Golang in Development Environment

```bash
wget https://golang.google.cn/dl/go1.24.0.linux-amd64.tar.gz
# wget https://golang.google.cn/dl/go1.24.3.linux-arm64.tar.gz

tar zxvf go1.24.0.linux-amd64.tar.gz
g

mkdir -p /opt/go

vi /etc/profile
```

```bash
export GOROOT=/usr/local/go
export GOPATH=/opt/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
export GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
```

### 1.3 Install Docker for Development Environment

```bash
apt install docker.io docker-compose -y
systemctl enable docker --now
```

Confirm Version

```console
root@VM-4-12-ubuntu:~# which docker
/usr/bin/docker

root@VM-4-12-ubuntu:~# which containerd
/usr/bin/containerd

root@VM-4-12-ubuntu:~# containerd -version
containerd github.com/containerd/containerd 1.7.24

root@VM-4-12-ubuntu:~# docker version
Client:
 Version:           26.1.3
 API version:       1.45
 Go version:        go1.22.2
 Git commit:        26.1.3-0ubuntu1~24.04.1
 Built:             Mon Oct 14 14:29:26 2024
 OS/Arch:           linux/amd64
 Context:           default

Server:
 Engine:
  Version:          26.1.3
  API version:      1.45 (minimum version 1.24)
  Go version:       go1.22.2
  Git commit:       26.1.3-0ubuntu1~24.04.1
  Built:            Mon Oct 14 14:29:26 2024
  OS/Arch:          linux/amd64
  Experimental:     false
 containerd:
  Version:          1.7.24
  GitCommit:        
 runc:
  Version:          1.1.12-0ubuntu3.1
  GitCommit:        
 docker-init:
  Version:          0.19.0
  GitCommit:        
```

Config Docker Mirror in China Mainland

```json
vi /etc/docker/daemon.json

{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://mirror.baidubce.com",
    "http://hub-mirror.c.163.com"
  ]
}
```

Restart Docker service

```bash
systemctl daemon-reload
systemctl restart docker
```

Test Docker

```bash
docker run hello-world
```

### 1.4 Install Prerequisite Packages

```bash
apt install build-essential git curl wget net-tools -y
apt install ntp -y
```

### 1.5 Clone & compile

```bash
git clone git@github.com:duicikeyihangaolou/kubeclipper.git
cd kubeclipper
git checkout release-1.4

# go clean -modcache
# go mod tidy
# go mod download

make build
```

Check assembly results:

```console
# ls dist/
kcctl  kubeclipper-agent  kubeclipper-proxy  kubeclipper-server
```

## 2. Installing & Debugging

### 2.1 Deploy KC in Deployment Environment

Deploy kcctl

```console
# curl -sfL https://oss.kubeclipper.io/get-kubeclipper.sh | KC_REGION=cn KC_VERSION=release-1.4 bash -
[INFO]  The release-1.4 version will be installed
[INFO]  KC_REGION is assigned cn, kc.env file will be created
[INFO]  env: Creating environment file /etc/kc/kc.env}
mkdir: created directory '/etc/kc'
[INFO]  Downloading package https://oss.kubeclipper.io/kc/release-1.4/kc-linux-amd64.tar.gz
kubeclipper-server
kubeclipper-agent
kubeclipper-proxy
kcctl
[INFO]  Installing kcctl to /usr/local/bin/kcctl

  Kcctl has been installed successfully!
    Run 'kcctl version' to view the version.
    Run 'kcctl -h' for more help.

        __ ______________________
       / //_/ ____/ ____/_  __/ /
      / ,< / /   / /     / / / /
     / /| / /___/ /___  / / / /___
    /_/ |_\____/\____/ /_/ /_____/
    repository: github.com/kubeclipper
```

Deploy KC with AIO mode

```bash
# Make sure you can ssh localhost without password
# deploy MUST with passwd or pk-file FOR 'kcctl resource' cmd
#
# ssh-keygen -t rsa
# cat /root/.ssh/id_rsa.pub >> authorized_keys

KC_VERSION=release-1.4 kcctl deploy
# kcctl deploy --help
# kcctl deploy --user root --passwd {local-host-user-pwd} --pkg kc-minimal.tar.gz
# kcctl deploy --server $IPADDR_SERVER --agent $IPADDR_AGENT --passwd xxx # or --pk-file
```

```console
# KC_VERSION=release-1.4 kcctl deploy
2025-05-30T10:08:59+08:00	INFO	Using auto detected IPv4 address on interface eth0: 10.0.4.12/22
[2025-05-30T10:08:59+08:00][INFO] node-ip-detect inherits from ip-detect: first-found
[2025-05-30T10:08:59+08:00][INFO] run in aio mode.
[2025-05-30T10:08:59+08:00][INFO] ============>kc-etcd PRECHECK ...
[2025-05-30T10:08:59+08:00][INFO] ============>kc-etcd PRECHECK OK!
[2025-05-30T10:08:59+08:00][INFO] ============>kc-server PRECHECK ...
[2025-05-30T10:08:59+08:00][INFO] ============>kc-server PRECHECK OK!
[2025-05-30T10:08:59+08:00][INFO] ============>kc-agent PRECHECK ...
[2025-05-30T10:08:59+08:00][INFO] ============>kc-agent PRECHECK OK!
[2025-05-30T10:08:59+08:00][INFO] ============>TIME-LAG PRECHECK ...
[2025-05-30T10:08:59+08:00][INFO] BaseLine Time: 2025-05-30T10:08:59+08:00
[2025-05-30T10:08:59+08:00][INFO] [10.0.4.12] -0.949546294 seconds
[2025-05-30T10:08:59+08:00][INFO] all nodes time lag less then 5 seconds
[2025-05-30T10:08:59+08:00][INFO] ============>TIME-LAG PRECHECK OK!
[2025-05-30T10:08:59+08:00][INFO] ============>NTP PRECHECK ...
[2025-05-30T10:08:59+08:00][INFO] ============>NTP PRECHECK OK!
[2025-05-30T10:08:59+08:00][INFO] ============>sudo PRECHECK ...
[2025-05-30T10:08:59+08:00][INFO] ============>sudo PRECHECK OK!
[2025-05-30T10:08:59+08:00][INFO] ============>ipDetect PRECHECK ...
[2025-05-30T10:08:59+08:00][INFO] ============>ipDetect PRECHECK OK!
[2025-05-30T10:09:02+08:00][INFO] ------ Send packages ------
--2025-05-30 10:09:02--  https://oss.kubeclipper.io/release/release-1.4/kc-amd64.tar.gz
2025-05-30 10:11:29 (6.44 MB/s) - ‘kc-amd64.tar.gz’ saved [982526790/982526790]
10.0.4.12: done!   
[2025-05-30T10:11:44+08:00][INFO] ------ Install kc-etcd ------
[2025-05-30T10:11:50+08:00][INFO] ------ Install kc-server ------
[2025-05-30T10:12:03+08:00][INFO] ------ Install kc-agent ------
[2025-05-30T10:12:03+08:00][INFO] ------ Install kc-console ------
[2025-05-30T10:12:04+08:00][INFO] ------ Delete intermediate files ------
[2025-05-30T10:12:04+08:00][INFO] ------ Dump configs ------
[2025-05-30T10:12:04+08:00][INFO] ------ Upload configs ------
10.0.4.12: done!   

 _   __      _          _____ _ _
| | / /     | |        /  __ \ (_)
| |/ / _   _| |__   ___| /  \/ |_ _ __  _ __   ___ _ __
|    \| | | | '_ \ / _ \ |   | | | '_ \| '_ \ / _ \ '__|
| |\  \ |_| | |_) |  __/ \__/\ | | |_) | |_) |  __/ |
\_| \_/\__,_|_.__/ \___|\____/_|_| .__/| .__/ \___|_|
                                 | |   | |
                                 |_|   |_|
        repository: github.com/kubeclipper
```

### 2.2 Check Services

```console
# systemctl list-unit-files | grep kc
kc-agent.service                             enabled         enabled
kc-console.service                           enabled         enabled
kc-etcd.service                              enabled         enabled
kc-server.service                            enabled         enabled
```

### 2.3 Clean

```bash
kcctl clean --help

# Uninstall the entire kubeclipper platform.
kcctl clean
kcctl clean -A
```

### 2.4 Debug kcctl using gdb

```bash
apt install gdb -y

export GOLDFLAGS=""
cd ~/kubeclipper
make build
# or
# make build-cli

# Start GDB debugger and load the kcctl binary for debugging
gdb dist/kcctl
# Set a breakpoint at line 278 of resource.go to pause execution
b resource.go:278
# Run kcctl with login command to authenticate against local server
r login --host http://127.0.0.1 --username admin --password Thinkbig1
# Run kcctl to list cluster resources (triggers breakpoint if hit)
r resource list
# Quit debug session
q
```

OR, debug with VSCode, see `.vscode/launch.json`.

## 3. Tarball k8s

### 3.1 Tarball

Take v1.32.2 as example:

```bash
cd ~/kubeclipper/scripts
bash tarball-kubernetes.sh -a amd64 -v 1.32.2 -o /tmp

k8s_ver='v1.32.2'
mkdir -p k8s/${k8s_ver}/amd64
pushd k8s/${k8s_ver}/amd64
cp /tmp/k8s/${k8s_ver}/amd64/* . -a
popd
tar -zcvf k8s-${k8s_ver}-amd64.tar.gz k8s
kcctl login --host http://127.0.0.1 --username admin --password Thinkbig1
kcctl resource push --pkg k8s-${k8s_ver}-amd64.tar.gz --type k8s
```

### 3.2 Add k8s v1.32.2 info

```bash
vi /opt/kubeclipper-server/resource/metadata.json
```

### 3.3 Deploy k8s using kcctl, add pause tag

```bash
vi /etc/containerd/config.toml
# from
# sandbox_image = "registry.k8s.io/pause:"
# to
# sandbox_image = "registry.k8s.io/pause:3.10"
```

### 3.4 Backup /tmp/.k8s and reboot host server

```bash
mkdir ~/bak -p
cp /tmp/.k8s ~/bak -a
# after reboot
cp ~/bak/.k8s /tmp -a
# continue deploy k8s with web GUI
```

## 4. Demo with kubeedge

### 4.1 kind deploy k8s

```bash
# go 1.16+ and docker, podman or nerdctl installed 

# cat portmapping.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 10000
    hostPort: 10000
  - containerPort: 10001
    hostPort: 10001
  - containerPort: 10002
    hostPort: 10002
  - containerPort: 10003
    hostPort: 10003
  - containerPort: 10004
    hostPort: 10004
    # optional: set the bind address on the host
    # 0.0.0.0 is the current default
    # listenAddress: "127.0.0.1"
    # optional: set the protocol to one of TCP, UDP, SCTP.
    # TCP is the default
    protocol: TCP

kind create cluster --config portmapping.yaml

# kind delete cluster

# cp to host computer
docker exec  kind-control-plane tar -c -f -  /usr/bin/kubectl > kubectl.tar
tar xf kubectl.tar
mv usr/bin/kubectl /usr/bin/
```

### 4.2  deploy kubeedge with keadm

```bash
# cloudnode
wget https://github.com/kubeedge/kubeedge/releases/download/v1.20.0/keadm-v1.20.0-linux-amd64.tar.gz
# scp keadm-v1.20.0-linux-amd64.tar.gz edgenode
tar -zxvf keadm-v1.20.0-linux-amd64.tar.gz
cp keadm-v1.20.0-linux-amd64/keadm/keadm /usr/local/bin/keadm
keadm version

wget https://github.com/kubeedge/kubeedge/releases/download/v1.20.0/kubeedge-v1.20.0-linux-amd64.tar.gz
wget https://github.com/kubeedge/kubeedge/releases/download/v1.20.0/edgesite-v1.20.0-linux-amd64.tar.gz

mkdir /etc/kubeedge
cp kubeedge-v1.20.0-linux-amd64.tar.gz  edgesite-v1.20.0-linux-amd64.tar.gz /etc/kubeedge

docker pull kubeedge/installation-package:v1.20.0
docker pull kubeedge/iptables-manager:v1.20.0
docker pull kubeedge/cloudcore:v1.20.0
kind load docker-image kubeedge/installation-package:v1.20.0 --name kind
kind load docker-image kubeedge/iptables-manager:v1.20.0 --name kind
kind load docker-image kubeedge/cloudcore:v1.20.0  --name kind

# keadm init --advertise-address="THE-EXPOSED-IP" --kubeedge-version=v1.20.0
keadm init  --advertise-address=192.168.1.208 -v 7

docker exec  kind-control-plane ss -tuln

kubectl get all -n kubeedge 
kubectl get nodes
# kubectl  delete nodes k8s-node1

keadm gettoken


# edgenode

vi /etc/containerd/config.toml

runtime_type = "io.containerd.runc.v2"
# disabled_plugins = ["cri"]
disabled_plugins = []
SystemdCgroup = true


# edge node install cni for cri
wget https://github.com/containernetworking/plugins/releases/download/v1.6.2/cni-plugins-linux-amd64-v1.6.2.tgz
mkdir -p /opt/cni/bin
tar Cxzvf /opt/cni/bin cni-plugins-linux-amd64-v1.6.2.tgz

mkdir -p /etc/cni/net.d/
$ cat >/etc/cni/net.d/10-containerd-net.conflist <<EOF
{
  "cniVersion": "1.0.0",
  "name": "containerd-net",
  "plugins": [
    {
      "type": "bridge",
      "bridge": "cni0",
      "isGateway": true,
      "ipMasq": true,
      "promiscMode": true,
      "ipam": {
        "type": "host-local",
        "ranges": [
          [{
            "subnet": "10.88.0.0/16"
          }],
          [{
            "subnet": "2001:db8:4860::/64"
          }]
        ],
        "routes": [
          { "dst": "0.0.0.0/0" },
          { "dst": "::/0" }
        ]
      }
    },
    {
      "type": "portmap",
      "capabilities": {"portMappings": true}
    }
  ]
}
EOF

systemctl restart containerd.service

ctr -n k8s.io image ls

docker pull kubeedge/installation-package:v1.20.0
docker save kubeedge/installation-package:v1.20.0 -o keip.tar
ctr -n k8s.io image import keip.tar

docker pull pause:3.8
docker save pause:3.8 -o pause.3.8.tar
ctr -n k8s.io image import pause.3.8.tar


ctr -n k8s.io image pull docker.io/library/eclipse-mosquitto:1.6.15
# or
docker pull eclipse-mosquitto:1.6.15
docker save eclipse-mosquitto:1.6.15 -o tt.tar
ctr -n k8s.io image import tt.tar

docker pull docker.io/kindest/kindnetd:v20250214-acbabc1a
docker save docker.io/kindest/kindnetd:v20250214-acbabc1a -o kek.tar
ctr -n k8s.io image import kek.tar

docker pull registry.k8s.io/kube-proxy:v1.32.2
docker save registry.k8s.io/kube-proxy:v1.32.2 -o kep.tar
ctr -n k8s.io image import kep.tar

ctr -n k8s.io image  ls

ctr -n  kube-system c ls
ctr -n  kube-system t ls
ctr -n  kube-system i ls


keadm join  --cloudcore-ipport=172.18.0.2:10000 \
--token=keadm-gettoken-cloudnode --kubeedge-version=v1.20.0 \
-p=unix:///var/run/docker.sock

systemctl status  edgecore.service
# systemctl restart  edgecore.service
journalctl -u edgecore.service -xe

```

### 4.3 Config kube api in kubeedge node

#### 4.3.1 CloudCore side

```bash
kubectl edit cm cloudcore -n kubeedge

      dynamicController:
        enable: true

docker exec -it kind-control-plane bash
crictl pods
# restart cloudcore pod
crictl stopp cloudcore-5d9ccb9dc8-lv2qb

```

#### 4.3.2 EdgeCore Side

```bash
vi /etc/kubeedge/config/edgecore.yaml
modules:
  ...
  edgeMesh:
    enable: false
  ...
  metaManager:
    metaServer:
      enable: true

vi /etc/kubeedge/config/edgecore.yaml
modules:
  ...
  edged:
    ...
    tailoredKubeletConfig:
      ...
      clusterDNS:
      - 169.254.96.16
      clusterDomain: cluster.local

systemctl restart  edgecore.service

# metaServer port 10550
netstat -natp|grep 10550
curl 127.0.0.1:10550/api/v1/services
curl http://127.0.0.1:10550/api/v1/namespaces/kube-system/pods|jq '.items.[].metadata.name'

"coredns-668d6bf9bc-j8cq4"
"coredns-668d6bf9bc-kf4cl"
"etcd-kind-control-plane"
"kindnet-5bgh7"
"kindnet-qsllc"
"kube-apiserver-kind-control-plane"
"kube-controller-manager-kind-control-plane"
"kube-proxy-92w6x"
"kube-proxy-rcjfq"
"kube-scheduler-kind-control-plane"


# cloud
kubectl get pods -n kube-system -o wide

# edge memroy
ps -p <PID> -o rss
```

### 4.4 Deploy hostnetwork application

#### 4.4.1 edge node

```bash
docker pull nginx
docker save nginx:latest -o nginx.tar
ctr -n k8s.io image import nginx.tar
ctr image import nginx.tar
ctr -n k8s.io image list
ctr image list
netstat -natp|grep nginx

```

#### 4.4.2 cloud node

```bash
kubectl apply -f hostnginx.yaml
kubectl get pods -o wide
kubectl describe pod  nginx-deployment-799c65967c-q8nvt

```

#### 4.4.3 Application yaml file

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      hostNetwork: true
      containers:
      - name: nginx
        image: docker.io/library/nginx:latest
        imagePullPolicy: Never
        ports:
        - containerPort: 80
          hostPort: 80

```
