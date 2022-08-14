<p align="center">
<a href="https://kubeclipper.io/"><img src="docs/img/kubeclipper.gif" alt="banner" width="200px"></a>
</p>

<p align="center">
<b>Manage kubernetes in the most light and convenient way</b>
</p>

<!-- TODO: 添加 cicd 执行情况，代码质量等标签 -->

<p align="center">
  <img alt="repo status" src="https://img.shields.io/badge/-Repo_Status_>-000000?style=flat-square&logo=github&logoColor=white" />
  <a href="https://codecov.io/gh/kubeclipper-labs/kubeclipper" target="_blank"><img alt="coverage" src="https://codecov.io/gh/kubeclipper-labs/kubeclipper/branch/master/graph/badge.svg"/></a>
  <img alt="last commit" src="https://img.shields.io/github/last-commit/kubeclipper-labs/kubeclipper?style=flat-square">
  <img alt="Issues" src="https://img.shields.io/github/issues/kubeclipper-labs/kubeclipper?style=flat-square&labelColor=343b41"/>
  <img alt="Pull Requests" src="https://img.shields.io/github/issues-pr/kubeclipper-labs/kubeclipper?style=flat-square&labelColor=343b41"/>
  <img alt="contributors" src="https://img.shields.io/github/contributors/kubeclipper-labs/kubeclipper?style=flat-square"/>
  <img alt="apache2.0" src="https://img.shields.io/badge/License-Apache_2.0-blue?style=flat-square" />
  <img alt="Stars" src="https://img.shields.io/github/stars/kubeclipper-labs/kubeclipper?style=flat-square&labelColor=343b41"/>
  <img alt="Forks" src="https://img.shields.io/github/forks/kubeclipper-labs/kubeclipper?style=flat-square&labelColor=343b41"/>
</p>

<p align="center">
  <img alt="github actions" src="https://img.shields.io/badge/-Github_Actions_>-000000?style=flat-square&logo=github-actions&logoColor=white" />
  <img alt="code-check-test" src="https://github.com/kubeclipper-labs/kubeclipper/actions/workflows/code-check-test.yml/badge.svg" />
  <img alt="build-kc" src="https://github.com/kubeclipper-labs/kubeclipper/actions/workflows/build-kc.yml/badge.svg" />
</p>

---

## KubeClipper

> English | [中文](README_zh.md)

<!-- TODO: 介绍 -->

## Features

<!-- TODO: 功能列表 -->

## Quick Start

For users who are new to KubeClipper and want to get started quickly, it is recommended to use the All-in-One installation mode, which can help you quickly deploy KubeClipper with zero configuration.

### Preparations

KubeClipper itself does not take up too many resources, but in order to run Kubernetes better in the future,  it is recommended that the hardware configuration should not be lower than the minimum requirements.

You only need to prepare a host with reference to the following requirements for machine hardware and operating system.

#### Hardware recommended configuration

- Make sure your machine meets the minimum hardware requirements: CPU >= 2 cores, RAM >= 2GB.
- Operating System: CentOS 7.x / Ubuntu 18.04 / Ubuntu 20.04.

#### Node requirements

- Nodes must be able to connect via `SSH`.

- You can use the `sudo` / `curl` / `wget` / `tar` command on this node.

> It is recommended that your operating system is in a clean state (no additional software is installed), otherwise, conflicts may occur.



### Deploy KubeClipper

#### Download kcctl

KubeClipper provides command line tools 🔧 kcctl to simplify operations.

You can download the latest version of kcctl directly with the following command:

```bash
curl -sfL https://oss.kubeclipper.io/kcctl.sh | sh -
# In China, you can add cn env, we use registry.aliyuncs.com/google_containers instead of k8s.gcr.io
curl -sfL https://oss.kubeclipper.io/kcctl.sh | KC_REGION=cn sh -
```

> You can also download the specified version on the [GitHub Release Page ](https://github.com/kubeclipper-labs/kubeclipper/releases).

Check if the installation is successful with the following command:

```bash
kcctl version
```

#### Get Started with Installation

In this quick start tutorial, you only need to run  just one command for installation, and the template looks like this:

```bash
kcctl deploy  [--user root] (--passwd SSH_PASSWD | --pk-file SSH_PRIVATE_KEY)
```

If you use the ssh password method, the command is as follows:

```bash
kcctl deploy --user root --passwd $SSH_PASSWD
```

If you use the private key method is as follows:

```bash
kcctl deploy --user root --pk-file $SSH_PRIVATE_KEY
```

> You only need to provide a ssh user and ssh password or ssh private key to deploy KubeClipper.

After you runn this command, kcctl will check your installation environment and enter the installation process, if the conditions are met.

After printing the KubeClipper banner, the installation is complete.

```bash
 _   __      _          _____ _ _
| | / /     | |        /  __ \ (_)
| |/ / _   _| |__   ___| /  \/ |_ _ __  _ __   ___ _ __
|    \| | | | '_ \ / _ \ |   | | | '_ \| '_ \ / _ \ '__|
| |\  \ |_| | |_) |  __/ \__/\ | | |_) | |_) |  __/ |
\_| \_/\__,_|_.__/ \___|\____/_|_| .__/| .__/ \___|_|
                                 | |   | |
                                 |_|   |_|
```

### Login Console

When deployed successfully, you can open a browser and visit `http://$IP ` to enter the KubeClipper console.

![](docs/img/console-login.png)

 You can log in with the default account and password `admin / Thinkbig1 `.

> You may need to configure port forwarding rules and open ports in security groups for external users to access the console.

### Create a k8s cluster

When `kubeclipper` is deployed successfully, you can use the **kcctl** **tool** or **console** to create a  k8s cluster. In the quick start tutorial, we use the kcctl tool to create.

First, log in with the default account and password to obtain the token, which is convenient for subsequent interaction between kcctl and kc-server.

```bash
kcctl login -H http://localhost  -u admin -p Thinkbig1
```

Then create a k8s cluster with the following command:

```bash
NODE=$(kcctl get node -o yaml|grep ipv4DefaultIP:|sed 's/ipv4DefaultIP: //')

kcctl create cluster --master $NODE --name demo --untaint-master
```

The cluster creation will be completed in about 3 minutes, or you can use the following command to view the cluster status:

```bash
kcctl get cluster -o yaml|grep status -A5
```

> You can also enter the console to view real-time logs.

Once the cluster enter  the `Running` state , it means that the creation is complete. You can use `kubectl get cs` command to view the cluster status.

## Development and Debugging

1. fork repo and clone
2. run etcd locally, usually use docker / podman to run etcd container
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
3. change `kubeclipper-server.yaml` etcd.serverList to your locally etcd cluster
4. `make build`
5. `./dist/kubeclipper-server serve`

## Architecture

![kc-arch1](docs/img/kc-arch.png)

![kc-arch2](docs/img/kc-arch2.png)

## Contributing

Please follow [Community](https://github.com/kubeclipper-labs/community) to join us.
