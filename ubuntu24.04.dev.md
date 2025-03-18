# ubuntu24.04.dev.md

- KubeClipper dev

1. install golang

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

1. install docker

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

1. install make & ntp

```bash
# build-essential/noble,now 12.10ubuntu1 amd64 [installed]
# GNU Make 4.3
# ntp/noble,now 1:4.2.8p15+dfsg-2~1.2.2+dfsg1-4build2 all [installed]

sudo apt install -y build-essential git curl wget net-tools -y
apt install ntp -y
```

1. clone & comple

```bash
git clone https://github.com/kubeclipper/kubeclipper.git
or
git clone git@github.com:drcwr/kubeclipper.git
git checkout release-1.4
make build
```

1. deploy

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

1. clean

```bash
kcctl clean --help

# Uninstall the entire kubeclipper platform.
kcctl clean --all
kcctl clean -A
```

1. debug kcctl using gdb

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

1. ssh-keygen

```bash
ssh-keygen -t rsa -b 4096
ssh-keygen -f ~/.ssh/id_rsa.pub -e -m pem > id_rsa.pem
```

- tarball k8s v1.32.2

1. add root passwd

```bash
vim /etc/ssh/sshd_config
PermitRootLogin yes
systemctl restart ssh

passwd
```

1. deploy MUST with passwd or pk-file FOR 'kcctl resource' cmd

```bash
kcctl deploy --server $IPADDR_SERVER --agent $IPADDR_AGENT --passwd xxx 或者 --pk-file
```

1. tarball

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

1. add k8s v1.32.2 info

```bash
vim /opt/kubeclipper-server/resource/metadata.json
```

1. deploy k8s using kcctl,add pause tag

```bash
vim /etc/containerd/config.toml
# from
# sandbox_image = "registry.k8s.io/pause:"
# to
# sandbox_image = "registry.k8s.io/pause:3.10"
```

1. bakup /tmp/.k8s and reboot host server

```bash
mkdir ~/bak -p
cp /tmp/.k8s ~/bak -a
# after reboot
cp ~/bak/.k8s /tmp -a
# continue deploy k8s with web GUI
```
