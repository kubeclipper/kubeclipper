<p align="center">
<a href="https://kubeclipper.io/"><img src="docs/img/kubeclipper.gif" alt="banner" width="200px"></a>
</p>

<p align="center">
<b>Manage kubernetes in the most light and convenient way</b>
</p>

<!-- TODO: 添加 cicd 执行情况，代码质量等标签 -->

<p align="center">
  <img alt="repo status" src="https://img.shields.io/badge/-Repo_Status_>-000000?style=flat-square&logo=github&logoColor=white" />
  <a href="https://codecov.io/gh/kubeclipper/kubeclipper" target="_blank"><img alt="coverage" src="https://codecov.io/gh/kubeclipper/kubeclipper/branch/master/graph/badge.svg"/></a>
  <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/kubeclipper/kubeclipper"/>
  <a href="https://www.codacy.com/gh/kubeclipper/kubeclipper/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=kubeclipper/kubeclipper&amp;utm_campaign=Badge_Grade"><img src="https://app.codacy.com/project/badge/Grade/6d077c30cb3e4e269b891380c22d5fc0"/></a>
  <img alt="last commit" src="https://img.shields.io/github/last-commit/kubeclipper/kubeclipper?style=flat-square">
  <img alt="Issues" src="https://img.shields.io/github/issues/kubeclipper/kubeclipper?style=flat-square&labelColor=343b41"/>
  <img alt="Pull Requests" src="https://img.shields.io/github/issues-pr/kubeclipper/kubeclipper?style=flat-square&labelColor=343b41"/>
  <img alt="contributors" src="https://img.shields.io/github/contributors/kubeclipper/kubeclipper?style=flat-square"/>
  <img alt="apache2.0" src="https://img.shields.io/badge/License-Apache_2.0-blue?style=flat-square" />
  <img alt="Stars" src="https://img.shields.io/github/stars/kubeclipper/kubeclipper?style=flat-square&labelColor=343b41"/>
  <img alt="Forks" src="https://img.shields.io/github/forks/kubeclipper/kubeclipper?style=flat-square&labelColor=343b41"/>
</p>

<p align="center">
  <img alt="github actions" src="https://img.shields.io/badge/-Github_Actions_>-000000?style=flat-square&logo=github-actions&logoColor=white" />
  <img alt="code-check-test" src="https://github.com/kubeclipper/kubeclipper/actions/workflows/code-check-test.yml/badge.svg" />
  <img alt="build-kc" src="https://github.com/kubeclipper/kubeclipper/actions/workflows/build-kc.yml/badge.svg" />
</p>

---

## KubeClipper

> 中文 | [English](README.md)

<!-- TODO: 介绍 -->

## Features

<details>
  <summary><b>✨ 创建集群</b></summary>
  <ul>
  <li>支持在线部署、代理部署、离线部署</li>
  <li>管理常用镜像仓库</li>
  <li>从模版创建集群/安装插件</li>
  <li>支持多版本 K8S、CRI 部署</li>
  <li>NFS 存储支持</li>
  </ul>
</details>

<details>
  <summary><b>☸️ 集群管理</b></summary>
  <ul>
  <li>多区域、多集群管理</li>
  <li>访问集群 kubectl web console</li>
  <li>查看集群安装过程中的实时日志</li>
  <li>编辑集群（元数据等）</li>
  <li>删除集群</li>
  <li>添加/移除节点</li>
  <li>创建失败后从断点重试</li>
  <li>集群备份/还原、定时备份</li>
  <li>集群版本升级</li>
  <li>整个集群 / 单个插件保存为模版</li>
  <li>集群备份存储位置管理</li>
  </ul>
</details>

<details>
  <summary><b>🌐 区域 / 节点管理</b></summary>
  <ul>
  <li>添加 agent 节点并指定区域（kcctl）</li>
  <li>节点状态管理</li>
  <li>连接节点终端</li>
  <li>节点启用/禁用</li>
  <li>查看区域下节点和集群列表</li>
  </ul>
</details>

<details>
  <summary><b>🚪 访问控制</b></summary>
  <ul>
  <li>用户和角色管理</li>
  <li>自定义角色管理</li>
  <li>OIDC 集成</li>
  </ul>
</details>

## Quick Start

对于初次接触 KubeClipper 并想快速上手的用户，建议使用 All-in-One 安装模式，它能够帮助您零配置快速部署 KubeClipper。

### 准备工作

KubeClipper 本身并不会占用太多资源，但是为了后续更好的运行 Kubernetes 建议硬件配置不低于最低要求。

您仅需参考以下对机器硬件和操作系统的要求准备一台主机。

#### 硬件推荐配置

- 确保您的机器满足最低硬件要求：CPU >= 2 核，内存 >= 2GB。
- 操作系统：CentOS 7.x / Ubuntu 18.04 / Ubuntu 20.04。

#### 节点要求

- 节点必须能够通过 `SSH` 连接。
- 节点上可以使用 `sudo` / `curl` / `wget` / `tar` 命令。

> 建议您的操作系统处于干净状态（不安装任何其他软件），否则可能会发生冲突。

### 部署 KubeClipper

#### 下载 kcctl

KubeClipper 提供了命令行工具🔧 kcctl 以简化运维工作，您可以直接使用以下命令下载最新版 kcctl：

```bash
# 安装最新的 release 版本
curl -sfL https://oss.kubeclipper.io/get-kubeclipper.sh | bash -
# 如果你在中国，你可以在安装时使用 cn  环境变量, 此时我们会使用 registry.aliyuncs.com/google_containers 代替 k8s.gcr.io
curl -sfL https://oss.kubeclipper.io/get-kubeclipper.sh | KC_REGION=cn bash -
# 默认会下载最新版本，你可以通过指定VERSION下载所需版本. 比如指定安装 master 开发版本 (现在可选择的版本 master / v1.2.1 / v1.2.0)
curl -sfL https://oss.kubeclipper.io/get-kubeclipper.sh | VERSION=master bash -
```

> 您也可以在 **[GitHub Release Page](https://github.com/kubeclipper/kubeclipper/releases)** 下载指定版本。

通过以下命令检测是否安装成功:

```bash
kcctl version
```

#### 开始安装

在本快速入门教程中，您只需执行一个命令即可安装 KubeClipper：

如果想运行 AIO 模式

```bash
# 安装默认版本
kcctl deploy
# 通过指定 KC_VERSION 的值，指定安装的版本，比如安装 master 分支
KC_VERSION=master kcctl deploy
```

如果想安装多个节点，可以使用 `kcctl deploy -h` 获取更多帮助信息

执行该命令后，Kcctl 将检查您的安装环境，若满足条件将会进入安装流程。在打印出如下的 KubeClipper banner 后即表示安装完成。

```console
 _   __      _          _____ _ _
| | / /     | |        /  __ \ (_)
| |/ / _   _| |__   ___| /  \/ |_ _ __  _ __   ___ _ __
|    \| | | | '_ \ / _ \ |   | | | '_ \| '_ \ / _ \ '__|
| |\  \ |_| | |_) |  __/ \__/\ | | |_) | |_) |  __/ |
\_| \_/\__,_|_.__/ \___|\____/_|_| .__/| .__/ \___|_|
                                 | |   | |
                                 |_|   |_|
```

### 登录控制台

安装完成后，打开浏览器，访问 `http://$IP` 即可进入 KubeClipper 控制台。

![console](docs/img/console-login.png)

您可以使用默认帐号密码 `admin / Thinkbig1` 进行登录。

> 您可能需要配置端口转发规则并在安全组中开放端口，以便外部用户访问控制台。

### 创建 k8s 集群

部署成功后您可以使用 **kcctl 工具**或者通过**控制台**创建 k8s 集群。在本快速入门教程中使用 kcctl 工具进行创建。

首先使用默认帐号密码进行登录获取 token，便于后续 kcctl 和 kc-server 进行交互。

```bash
# 如果您运行 kc-server 的节点 ip 是 192.168.234.3
# 在实际执行时你应该替换成您自己的 kc-server 节点 ip
kcctl login -H http://192.168.234.3:8080 -u admin -p Thinkbig1
```

然后使用以下命令创建 k8s 集群:

```bash
NODE=$(kcctl get node -o yaml|grep ipv4DefaultIP:|sed 's/ipv4DefaultIP: //')

kcctl create cluster --master $NODE --name demo --untaint-master
```

大概 3 分钟左右即可完成集群创建,也可以使用以下命令查看集群状态

```bash
kcctl get cluster -o yaml|grep status -A5
```

> 您也可以进入控制台查看实时日志。

进入 Running 状态即表示集群安装完成,您可以使用 `kubectl get cs` 命令来查看集群健康状况。

## 开发和调试

1. fork repo and clone
2. 本地运行 etcd, 通常使用 docker / podman 启动 etcd 容器，启动命令参考如下

   ```bash
   export HostIP="Your-IP"
   docker run -d \
   --net host \
   k8s.gcr.io/etcd:3.5.0-0 etcd \
   --advertise-client-urls http://${HostIP}:2379 \
   --initial-advertise-peer-urls http://${HostIP}:2380 \
   --initial-cluster=infra0=http://${HostIP}:2380 \
   --listen-client-urls http://${HostIP}:2379,http://127.0.0.1:2379 \
   --listen-metrics-urls http://127.0.0.1:2381 \
   --listen-peer-urls http://${HostIP}:2380 \
   --name infra0 \
   --snapshot-count=10000 \
   --data-dir=/var/lib/etcd
   ```

3. 更新 `kubeclipper-server.yaml` 中 etcd 的配置
4. `make build`
5. `./dist/kubeclipper-server serve`

## Architecture

![kc-arch1](docs/img/kc-arch.png)

![kc-arch2](docs/img/kc-arch2.png)

## Contributing

请参考 [Community](https://github.com/kubeclipper/community) 的相关文档，加入我们