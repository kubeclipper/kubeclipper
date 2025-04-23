# ubuntu24.04.dev.md

## KubeClipper dev

### install golang

```bash
# ubuntu 24.04
# ubuntu-24.04.2-live-server-amd64.iso
# go1.24.0.linux-amd64.tar.gz

# https://golang.google.cn/dl/
wget https://golang.google.cn/dl/go1.24.0.linux-amd64.tar.gz

tar zxf go1.24.0.linux-amd64.tar.gz
sudo mv go /usr/local/

sudo chmod 777 /opt
mkdir -p /opt/go

vim /etc/profile
export GOROOT=/usr/local/go
export GOPATH=/opt/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
export GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
```

### install docker

```bash
# docker-ce/noble,now 5:28.0.1-1\~ubuntu.24.04~noble amd64 [installed]
# https://www.cnblogs.com/ylz8401/p/18251415

sudo apt update
sudo apt install apt-transport-https curl -y

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor \
-o /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) \
signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu \
$(. /etc/os-release && echo "$VERSION_CODENAME") \
stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt update
sudo apt install docker-ce docker-ce-cli containerd.io docker-buildx-plugin \
docker-compose-plugin -y

# 安装的组件包括：

# docker-ce：Docker Engine。
# docker-ce-cli：用于与 Docker 守护进程通信的命令行工具。
# containerd.io：管理容器生命周期的容器运行时环境。
# docker-buildx-plugin：增强镜像构建功能的 Docker 扩展工具，特别是在多平台构建方面。
# docker-compose-plugin：通过单个 YAML 文件管理多容器 Docker 应用的配置管理插件。

# 使用以下命令检查 Docker 的运行状态：

sudo systemctl is-active dockerd
sudo systemctl is-active docker

sudo docker run hello-world

sudo vim /etc/docker/daemon.json

{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://mirror.baidubce.com",
    "http://hub-mirror.c.163.com"
  ]
}

sudo systemctl daemon-reload
sudo systemctl restart docker

sudo docker info

sudo usermod -aG docker ${USER}

# 卸载 Docker

# sudo apt purge docker-ce docker-ce-cli containerd.io docker-buildx-plugin  \
# docker-compose-plugin docker-ce-rootless-extras
# sudo rm -rf /var/lib/docker
# sudo rm -rf /var/lib/containerd
```

### install make & ntp

```bash
# build-essential/noble,now 12.10ubuntu1 amd64 [installed]
# GNU Make 4.3
# ntp/noble,now 1:4.2.8p15+dfsg-2~1.2.2+dfsg1-4build2 all [installed]

sudo apt install -y build-essential git curl wget net-tools -y
apt install ntp -y
```

### clone & comple

```bash
git clone https://github.com/kubeclipper/kubeclipper.git
or
git clone git@github.com:drcwr/kubeclipper.git
git checkout release-1.4
make build
```

### deploy

```bash
sudo su
vim /etc/ssh/sshd_config
PermitRootLogin yes
systemctl restart ssh
passwd xxx

curl -sfL https://oss.kubeclipper.io/get-kubeclipper.sh | KC_REGION=cn bash -

kcctl deploy --help
kcctl deploy --user root --passwd {local-host-user-pwd} --pkg kc-minimal.tar.gz
```

### clean

```bash
kcctl clean --help

# Uninstall the entire kubeclipper platform.
kcctl clean --all
kcctl clean -A
```

### debug kcctl using gdb

```bash
apt install gdb -y

export GOLDFLAGS=""
make build
or
make build-cli

gdb dist/kcctl
b resource.go:278
r login --host http://127.0.0.1 --username admin --password Thinkbig1
r resource list
```

### ssh-keygen

```bash
ssh-keygen -t rsa -b 4096
ssh-keygen -f ~/.ssh/id_rsa.pub -e -m pem > id_rsa.pem
```

## tarball k8s v1.32.2

### add root passwd

```bash
vim /etc/ssh/sshd_config
PermitRootLogin yes
systemctl restart ssh

passwd
```

### deploy MUST with passwd or pk-file FOR 'kcctl resource' cmd

```bash
kcctl deploy --server $IPADDR_SERVER --agent $IPADDR_AGENT --passwd xxx 或者 --pk-file
```

### tarball

```bash
./tarball-kubernetes.sh -a amd64 -v 1.32.2 -o /tmp

k8s_ver='v1.32.2'
mkdir -p k8s/${k8s_ver}/amd64
pushd k8s/${k8s_ver}/amd64
cp cp /tmp/k8s/${k8s_ver}/amd64/* . -a
popd
tar -zcvf k8s-${k8s_ver}-amd64.tar.gz  k8s
kcctl login --host http://127.0.0.1 --username admin --password Thinkbig1
kcctl resource push --pkg k8s-${k8s_ver}-amd64.tar.gz --type k8s
```

### add k8s v1.32.2 info

```bash
vim /opt/kubeclipper-server/resource/metadata.json
```

### deploy k8s using kcctl,add pause tag

```bash
vim /etc/containerd/config.toml
# from
# sandbox_image = "registry.k8s.io/pause:"
# to
# sandbox_image = "registry.k8s.io/pause:3.10"
```

### bakup /tmp/.k8s and reboot host server

```bash
mkdir ~/bak -p
cp /tmp/.k8s ~/bak -a
# after reboot
cp ~/bak/.k8s /tmp -a
# continue deploy k8s with web GUI
```

## deploy kubeedge

### kind deploy k8s

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

### keadm

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

vim /etc/containerd/config.toml

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

## kube api config

### cloud

```bash
kubectl edit cm cloudcore -n kubeedge

      dynamicController:
        enable: true

docker exec -it kind-control-plane bash
crictl pods
# restart cloudcore pod
crictl stopp cloudcore-5d9ccb9dc8-lv2qb

```

### edge

```bash
vim /etc/kubeedge/config/edgecore.yaml
modules:
  ...
  edgeMesh:
    enable: false
  ...
  metaManager:
    metaServer:
      enable: true

vim /etc/kubeedge/config/edgecore.yaml
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

  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100  161k    0  161k    0     0   588k      0 --:--:-- --:--:-- --:--:--  586k
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
