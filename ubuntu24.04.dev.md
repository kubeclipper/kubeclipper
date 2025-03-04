# dev 
## ubuntu 24.04

## install golang 
### go1.24.0.linux-amd64.tar.gz
```

https://golang.google.cn/dl/
wget https://golang.google.cn/dl/go1.24.0.linux-amd64.tar.gz

tar zxf go1.24.0.linux-amd64.tar.gz
sudo mv go /usr/local/

export GOROOT=/usr/local/go
sudo chmod 777 /opt
mkdir /opt/go
export GOPATH=/opt/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
export GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

```

## install docker 
### docker-ce/noble,now 5:28.0.1-1\~ubuntu.24.04~noble amd64 [installed]
```
https://www.cnblogs.com/ylz8401/p/18251415

sudo apt update
sudo apt install apt-transport-https curl -y

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt update
sudo apt install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin -y

安装的组件包括：

docker-ce：Docker Engine。
docker-ce-cli：用于与 Docker 守护进程通信的命令行工具。
containerd.io：管理容器生命周期的容器运行时环境。
docker-buildx-plugin：增强镜像构建功能的 Docker 扩展工具，特别是在多平台构建方面。
docker-compose-plugin：通过单个 YAML 文件管理多容器 Docker 应用的配置管理插件。
第 6 步：检查 Docker 服务状态
使用以下命令检查 Docker 的运行状态：

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

卸载 Docker

# sudo apt purge docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin docker-ce-rootless-extras
# sudo rm -rf /var/lib/docker
# sudo rm -rf /var/lib/containerd



```

## install make & ntp 
### build-essential/noble,now 12.10ubuntu1 amd64 [installed]
#### GNU Make 4.3
### ntp/noble,now 1:4.2.8p15+dfsg-2~1.2.2+dfsg1-4build2 all [installed]

```
sudo apt install -y build-essential git curl wget net-tools
apt install ntp

```

## clone & comple
```
git clone
git checkout 
make build
```

## deploy
```
sudo su

curl -sfL https://oss.kubeclipper.io/get-kubeclipper.sh | KC_REGION=cn bash -

kcctl deploy --user root

```
