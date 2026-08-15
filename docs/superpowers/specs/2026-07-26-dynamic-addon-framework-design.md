---
comet_change: dynamic-addon-framework
role: technical-design
canonical_spec: openspec
status: draft
---

# KubeClipper 动态 Addon 框架设计

## 1. 背景

KubeClipper 当前将集群扩展组件建模为 `Cluster.Addons`，插件元数据、参数校验、安装步骤、卸载步骤、模板渲染和健康检查均由编译进 `kubeclipper-server` 与 `kubeclipper-agent` 的 Go 实现提供。

以 MetalLB 为例，插件通过 `init()` 同时注册 `component.Interface`、`TemplateRender` 和 `AgentStep`。新增插件需要：

1. 在 `pkg/component` 增加 Go 包和模板；
2. 在 server 和 agent 的 `main.go` 中增加 blank import；
3. 重新编译并同时发布 server、agent；
4. 在 Console 中补充插件分类、详情页或 UI schema 特例；
5. 将插件资源与 KubeClipper 安装包一起发布。

该模型适合少量内置组件，但无法支持插件独立发布和运行时发现。大部分 Kubernetes Addon 最终通过 Helm Chart、YAML Manifest 或 Operator 安装，不需要为每个 Addon 编译专属 Go 执行代码。

## 2. 设计目标

### 2.1 目标

- 插件包以 OCI Artifact 独立发布到 Registry；
- KubeClipper 通过 Catalog 动态发现插件及版本；
- 新增 Helm/Manifest 类型插件时无需修改或重新编译 server、agent、console；
- 复用 `Operation -> Step -> Agent` 领域模型，并以 Operation Engine v2 的持久状态机、恢复、幂等投递和取消语义作为执行基础设施；
- 由 Addon Controller 维护插件期望状态、执行状态和失败恢复；
- 插件默认继承目标集群的 `Cluster.ImageRegistry`；
- 支持安装、升级、卸载、健康检查和幂等重试；
- server 与 agent 只内置稳定、通用、类型化的执行器；
- 插件包、安装实例和执行 Operation 有明确的职责边界。

### 2.2 非目标

- 不兼容现有 `component.Interface` 插件 API；
- 不提供 Go `.so` 动态加载；
- 首版不允许插件分发任意 Go 二进制或任意 RPC 服务；
- 首版不提供复杂依赖求解，仅支持直接依赖、冲突和确定性版本解析；
- 首版不允许外部插件向集群创建、升级、备份等核心生命周期注入 hook；
- 不把 Registry 全量仓库扫描作为插件发现机制；
- 不在插件 manifest 中保存完整的 `helm install` 或 `kubectl` shell 命令。

## 3. 核心决策

### 3.1 插件是 Addon Package，不是进程内代码插件

插件包只包含声明式元数据、Chart/Manifest、配置 schema、默认值和健康检查定义。插件包不能注册 Go interface，也不能向 server 或 agent 注入代码。

首版支持两种引擎：

- `Helm`：使用内置 Helm SDK 执行器；
- `Manifest`：使用内置 Kubernetes apply/delete 执行器。

复杂组件应把持续控制逻辑封装为目标集群内的 Operator，KubeClipper 仍只负责安装和观察 Operator。

### 3.2 Addon Controller 负责协调，Operation Engine v2 负责一次执行

Addon Controller 不直接在 reconcile 中执行 Helm，也不在 HTTP handler 中启动 goroutine。它负责：

1. 观察 `AddonInstallation` 的期望状态；
2. 解析指定的 `AddonPackage` 和 OCI digest；
3. 合并默认值、用户配置和集群 Registry 配置；
4. 生成不可变 Operation；
5. 观察 Operation 最终状态；
6. 更新 `AddonInstallation.status`；
7. 在删除时通过 finalizer 确保卸载完成。

Operation 表示某一 generation 的一次安装、升级或卸载尝试。Operation 成功不等同于通信成功或命令退出码为 0，Addon Controller 只认可 v2 `status.phase=Succeeded`。

Addon 执行阶段依赖 `docs/superpowers/specs/2026-07-26-operation-engine-v2-design.md`。Addon Controller 不得调用 legacy `DeliverTaskOperation`，所有 Addon Operation 必须绑定 Cluster name + UID；canonical ExecutionLock 由服务端根据 Cluster UID 生成。Helm/Manifest executor 必须对同一 Task UID 重复 Reconcile 可收敛，并包含显式 Verify。

### 3.3 Agent 只执行通用类型化命令

采用 Operation -> OperationTask -> Agent 的持久化执行拓扑。Operation Controller 按 Step 顺序创建绑定到目标 Node name + UID 的不可变 `OperationTask`；Agent 通过出站 mTLS HTTPS 对自身 Task 执行 List/Watch，并以单 worker 执行。Agent 重启后从 etcd 重新 List，优先对遗留 Running Task调用相同 Reconcile。Task 的 terminal result 与 phase 原子写入 Task status。Watch 只负责降低延迟，断线后的 List/relist、resourceVersion、确定性 Task identity 和重复 reconcile 才提供正确性。Agent 不读取整个 Operation，也不决定下一个 Addon Step。

新增统一的 `CommandExecutor` 命令类型。首版 executor：

- `HelmRelease/v1`；
- `ManifestApply/v1`；
- `ResourceCondition/v1`。

它们被一次性编译进 agent，与具体插件名称无关。插件包只能引用 KubeClipper 支持的 executor 和版本。

### 3.4 OCI Registry 是分发层，不是运行模型

插件包使用 OCI Artifact 发布。Artifact Registry 存储插件定义、Chart、schema 和默认 values；目标集群的 Image Registry 存储插件运行时容器镜像。二者职责分开，但可以是同一个 Registry 服务。

插件运行镜像默认继承目标集群的 `Cluster.ImageRegistry`。用户安装插件时不再次选择 Registry。

## 4. 总体架构

```text
                         +-----------------------------+
                         | OCI Registry                |
                         |                             |
                         | addon-catalog:stable        |
                         | addons/metallb@sha256:...   |
                         | addons/nfs-csi@sha256:...   |
                         +--------------+--------------+
                                        |
                                 sync metadata
                                        |
+-------------+       CRUD       +------v----------------+
| Console/API |----------------->| AddonPackage Cache    |
+------+------+                  +-----------------------+
       |
       | create/update/delete AddonInstallation
       v
+----------------------+     resolve/plan     +---------------------+
| AddonInstallation    |<-------------------->| Addon Controller    |
| desired/status       |                      +----------+----------+
+----------------------+                                 |
                                                        | create/watch
                                                        v
                                             +---------------------+
                                             | Operation           |
                                             | typed executor step |
                                             +----------+----------+
                                                        |
                                              create/reconcile
                                                        v
                                             +---------------------+
                                             | OperationTask       |
                                             | node-bound work     |
                                             +----------+----------+
                                                        |
                                            HTTPS List/Watch (mTLS)
                                                        |
                                             +----------v----------+
                                             | Target master Agent |
                                             | Helm/Manifest SDK   |
                                             +----------+----------+
                                                        |
                                             +----------v----------+
                                             | Kubernetes cluster  |
                                             +---------------------+
```

## 5. OCI Artifact 与 Catalog

### 5.1 Artifact 命名

一个插件版本对应一个不可变 Artifact：

```text
registry.example.com/kubeclipper/addons/metallb:0.14.9
registry.example.com/kubeclipper/addons/metallb@sha256:<digest>
```

tag 用于发布和人工识别；安装解析后始终锁定 digest。

### 5.2 Artifact media type

```text
artifactType:
  application/vnd.kubeclipper.addon.v1

config:
  application/vnd.kubeclipper.addon.config.v1+yaml

layers:
  application/vnd.kubeclipper.addon.chart.v1.tar+gzip
  application/vnd.kubeclipper.addon.values.v1+yaml
  application/vnd.kubeclipper.addon.schema.v1+json
  application/vnd.kubeclipper.addon.ui-schema.v1+json
  application/vnd.kubeclipper.addon.manifests.v1.tar+gzip
```

首版 Artifact 必须包含一个 config descriptor。Helm 插件必须包含 Chart layer；Manifest 插件必须包含 manifests layer。values、schema 和 ui-schema 按 manifest 声明决定是否必需。

### 5.3 Catalog Artifact

Registry 的 catalog/tag API 在不同实现之间行为不一致，server 不扫描整个 Registry。平台配置一个固定 Catalog Artifact：

```text
registry.example.com/kubeclipper/addon-catalog:stable
```

Catalog 示例：

```yaml
apiVersion: addons.kubeclipper.io/v1alpha1
kind: AddonCatalog

entries:
  - name: metallb
    versions:
      - version: 0.14.9
        channels: [stable]
        ref: registry.example.com/kubeclipper/addons/metallb@sha256:abc123

  - name: nfs-csi
    versions:
      - version: 4.11.0
        channels: [stable]
        ref: registry.example.com/kubeclipper/addons/nfs-csi@sha256:def456
```

Catalog tag 可以变化，但同步后每个 package ref 必须包含 digest。重复的 `name + version`、缺失 digest、重复 digest 指向冲突 package 等情况会使本次同步失败，继续提供上一次成功缓存。

### 5.4 Registry 配置

首版平台只配置一个 Addon Catalog，不引入多仓库资源：

```yaml
addon:
  catalogRef: registry.example.com/kubeclipper/addon-catalog:stable
  registryRef: platform-registry
  syncInterval: 10m
  cacheDir: /var/lib/kubeclipper/addons
  cacheMaxSize: 10Gi
```

`registryRef` 引用现有 `Registry` 资源，以复用 endpoint、TLS、CA 和认证配置。多 Catalog、多租户可在后续版本扩展。

## 6. Addon Package Manifest

### 6.1 Helm 示例

```yaml
apiVersion: addons.kubeclipper.io/v1alpha1
kind: AddonPackage

metadata:
  name: metallb

spec:
  version: 0.14.9
  category: load-balancer
  display:
    title: MetalLB
    description: Bare-metal load balancer

  engine:
    type: Helm
    helm:
      chartLayer: chart

  release:
    name: metallb
    namespace: metallb-system
    createNamespace: true

  install:
    atomic: true
    wait: true
    timeout: 10m
    historyMax: 5

  config:
    defaultsLayer: values
    schemaLayer: values-schema
    uiSchemaLayer: ui-schema

  images:
    inheritClusterRegistry: true
    entries:
      - name: controller
        originalRepository: quay.io/metallb/controller
        mirrorRepository: metallb/controller
        tag: v0.14.9
        repositoryValue: controller.image.repository
        tagValue: controller.image.tag
      - name: speaker
        originalRepository: quay.io/metallb/speaker
        mirrorRepository: metallb/speaker
        tag: v0.14.9
        repositoryValue: speaker.image.repository
        tagValue: speaker.image.tag

  health:
    - apiVersion: apps/v1
      kind: Deployment
      namespace: metallb-system
      name: metallb-controller
      condition: Available
    - apiVersion: apps/v1
      kind: DaemonSet
      namespace: metallb-system
      name: metallb-speaker
      condition: Available
```

插件包不包含 Helm shell 命令。`release` 和 `install` 字段由 Addon Controller 编译为 `HelmRelease/v1` executor payload。

### 6.2 Manifest 示例

```yaml
apiVersion: addons.kubeclipper.io/v1alpha1
kind: AddonPackage

metadata:
  name: metrics-server

spec:
  version: 0.7.2
  category: monitoring

  engine:
    type: Manifest
    manifest:
      manifestsLayer: manifests
      template: true

  config:
    defaultsLayer: values
    schemaLayer: values-schema

  install:
    fieldManager: kubeclipper-addon-controller
    forceConflicts: false
    wait: true
    timeout: 10m

  health:
    - apiVersion: apps/v1
      kind: Deployment
      namespace: kube-system
      name: metrics-server
      condition: Available
```

## 7. API 资源

### 7.1 AddonPackage

`AddonPackage` 是从 Catalog 同步得到的只读 API 资源。用户不能通过普通 CRUD 创建或修改。Package 以 `name/version` 为逻辑键，status 记录解析和缓存结果。

为了在现有 Kubernetes 风格存储中同时保存多个版本，资源名使用 `<package>-<normalized-version>`，并保留稳定 labels：

```yaml
metadata:
  name: metallb-0-14-9
  labels:
    addons.kubeclipper.io/name: metallb
    addons.kubeclipper.io/version: 0.14.9
    addons.kubeclipper.io/channel: stable
```

Catalog sync 根据逻辑键做唯一性校验，API 和 Console 不依赖 normalized resource name 解析业务含义。

```go
type AddonPackage struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              AddonPackageSpec   `json:"spec"`
    Status            AddonPackageStatus `json:"status,omitempty"`
}

type AddonPackageSpec struct {
    Version       string                  `json:"version"`
    Category      string                  `json:"category"`
    Artifact      AddonArtifactReference  `json:"artifact"`
    Engine        AddonEngine             `json:"engine"`
    Release       *AddonReleaseSpec       `json:"release,omitempty"`
    Install       AddonInstallPolicy      `json:"install,omitempty"`
    Config        AddonConfigSpec         `json:"config,omitempty"`
    Images        AddonImageSpec          `json:"images,omitempty"`
    Health        []AddonHealthCheck      `json:"health,omitempty"`
}

type AddonPackageStatus struct {
    Phase      AddonPackagePhase `json:"phase"`
    Digest     string            `json:"digest"`
    SyncedAt   metav1.Time       `json:"syncedAt"`
    Conditions []Condition       `json:"conditions,omitempty"`
}
```

读取 API：

```text
GET /api/addons.kubeclipper.io/v1/packages
GET /api/addons.kubeclipper.io/v1/packages/{resourceName}
GET /api/addons.kubeclipper.io/v1/packages?labelSelector=addons.kubeclipper.io/name=metallb
POST /api/addons.kubeclipper.io/v1/packages:sync
```

### 7.2 AddonInstallation

`AddonInstallation` 是插件在某一集群上的唯一事实来源：

```go
type AddonInstallation struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              AddonInstallationSpec   `json:"spec"`
    Status            AddonInstallationStatus `json:"status,omitempty"`
}

type AddonInstallationSpec struct {
    ClusterRef ClusterReference       `json:"clusterRef"`
    PackageRef AddonPackageReference  `json:"packageRef"`
    Values     runtime.RawExtension   `json:"values,omitempty"`
}

type AddonInstallationStatus struct {
    Phase                 AddonInstallationPhase `json:"phase"`
    ObservedGeneration    int64                  `json:"observedGeneration,omitempty"`
    ResolvedArtifact      string                 `json:"resolvedArtifact,omitempty"`
    ResolvedValuesHash    string                 `json:"resolvedValuesHash,omitempty"`
    InstalledVersion      string                 `json:"installedVersion,omitempty"`
    OperationRef          string                 `json:"operationRef,omitempty"`
    HelmRelease           *HelmReleaseStatus     `json:"helmRelease,omitempty"`
    ManagedResourceDigest string                 `json:"managedResourceDigest,omitempty"`
    Conditions            []Condition            `json:"conditions,omitempty"`
}
```

CRUD API：

```text
POST   /api/addons.kubeclipper.io/v1/installations
GET    /api/addons.kubeclipper.io/v1/installations
GET    /api/addons.kubeclipper.io/v1/installations/{name}
PATCH  /api/addons.kubeclipper.io/v1/installations/{name}
DELETE /api/addons.kubeclipper.io/v1/installations/{name}
```

按集群查询通过 label selector 完成：

```text
GET /api/addons.kubeclipper.io/v1/installations?labelSelector=kubeclipper.io/cluster=cluster-a
```

创建、升级和删除返回资源当前状态；执行进度通过 `status.operationRef` 查询已有 Operation API。

## 8. 配置与 Registry 合并

### 8.1 Values 来源

最终 Helm values 按以下顺序合并：

```text
Chart 默认 values
  < AddonPackage 默认 values
  < AddonInstallation.spec.values
  < KubeClipper 系统注入值
```

系统注入值优先级最高，仅用于镜像地址等保留字段。用户通过插件 schema 配置业务参数，不需要选择 Registry。

### 8.2 默认继承 Cluster.ImageRegistry

Addon Controller 读取目标集群：

```go
registry, err := ResolveImageRegistry(ctx, cluster.ImageRegistry, clusterOperator)
```

处理规则：

1. `Cluster.ImageRegistry` 为空时，使用 package 中的 `originalRepository`；
2. 集群指定 Registry 时，使用 `<registry.Host>/<mirrorRepository>`；
3. package 未声明 image entry 时不猜测 Helm values 字段，保持 Chart 原值；
4. system injection 在用户 values 之后执行，防止普通配置绕过集群 Registry 策略；
5. Registry 认证沿用集群 Registry 配置，不在 `AddonInstallation` 中重复保存账号密码。

示例：

```text
originalRepository = quay.io/metallb/controller
mirrorRepository   = metallb/controller
cluster registry   = harbor.example.com

resolved repository = harbor.example.com/metallb/controller
```

Controller 写入 package 声明的 value path：

```yaml
controller:
  image:
    repository: harbor.example.com/metallb/controller
    tag: v0.14.9
```

### 8.3 Schema 与 Secret

- `values.schema.json` 负责后端校验和 Console 动态表单；
- `ui.schema.json` 只描述 widget、排序、帮助信息等展示行为；
- Secret 值不得直接写入 package、Installation status 或 Operation 日志；
- 首版插件配置如需 Secret，使用现有 Kubernetes Secret 名称引用，由 Helm values 传递引用名；
- 后续可增加平台 SecretRef 注入，不阻塞首版。

## 9. Operation 与通用 Executor

### 9.1 扩展现有 Command

不重写集群生命周期已有的 shell/custom command。新增一个通用 executor command：

```go
const CommandExecutor CommandType = "executor"

type ExecutorCommand struct {
    APIVersion string               `json:"apiVersion"`
    Kind       string               `json:"kind"`
    Spec       runtime.RawExtension `json:"spec"`
}

type Command struct {
    Type          CommandType      `json:"type"`
    ShellCommand  []string         `json:"shellCommand,omitempty"`
    Identity      string           `json:"identity,omitempty"`
    CustomCommand []byte           `json:"customCommand,omitempty"`
    Template      *TemplateCommand `json:"template,omitempty"`
    Executor      *ExecutorCommand `json:"executor,omitempty"`
}
```

Agent 侧 registry 只按 `apiVersion/kind` 注册 KubeClipper 内置 executor，不注册插件名称：

```text
execution.kubeclipper.io/v1/HelmRelease
execution.kubeclipper.io/v1/ManifestApply
execution.kubeclipper.io/v1/ResourceCondition
```

### 9.2 HelmRelease/v1

```yaml
apiVersion: execution.kubeclipper.io/v1
kind: HelmRelease
spec:
  action: UpgradeInstall
  artifact:
    ref: registry.example.com/kubeclipper/addons/metallb@sha256:abc123
    layer: chart
  releaseName: metallb
  namespace: metallb-system
  createNamespace: true
  atomic: true
  wait: true
  timeout: 10m
  historyMax: 5
  values: {}
```

支持 action：

- `UpgradeInstall`：首次安装和幂等重试；
- `Upgrade`：升级已存在 release；
- `Uninstall`：卸载 release。

Executor 使用 Helm SDK，不执行 shell 拼接。Artifact 按 digest 拉取并缓存。Registry credentials 在任务投递时按 `registryRef` 解析，不持久化到 Operation spec 或日志。

### 9.3 ManifestApply/v1

```yaml
apiVersion: execution.kubeclipper.io/v1
kind: ManifestApply
spec:
  action: Apply
  artifact:
    ref: registry.example.com/kubeclipper/addons/metrics-server@sha256:def456
    layer: manifests
  fieldManager: kubeclipper-addon-controller
  forceConflicts: false
  values: {}
```

支持 `Apply` 和 `Delete`。Apply 后记录资源 inventory；Delete 使用安装 revision 的 inventory，不重新根据当前 Catalog 推测。

### 9.4 ResourceCondition/v1

```yaml
apiVersion: execution.kubeclipper.io/v1
kind: ResourceCondition
spec:
  expected: Ready
  timeout: 5m
  resources:
    - apiVersion: apps/v1
      kind: Deployment
      namespace: metallb-system
      name: metallb-controller
      condition: Available
```

健康检查失败会使 Operation 失败，不能设置统一的 `ErrIgnore: true`。

### 9.5 Operation 示例

Addon Controller 为一次 Helm 安装生成：

```yaml
metadata:
  name: addon-metallb-install-abc123
  labels:
    kubeclipper.io/cluster: cluster-a
    kubeclipper.io/addon-installation: metallb-default
    kubeclipper.io/action: AddonInstall
    kubeclipper.io/generation: "1"

steps:
  - id: install-release
    name: installHelmRelease
    nodes:
      - id: selected-master-id
    action: install
    timeout: 10m
    commands:
      - type: executor
        executor:
          apiVersion: execution.kubeclipper.io/v1
          kind: HelmRelease
          spec: {}

  - id: verify-release
    name: verifyAddonResources
    nodes:
      - id: selected-master-id
    action: install
    timeout: 5m
    commands:
      - type: executor
        executor:
          apiVersion: execution.kubeclipper.io/v1
          kind: ResourceCondition
          spec: {}
```

## 10. Addon 生命周期

### 10.1 状态机

```text
Pending
  -> Resolving
  -> Installing
  -> Verifying
  -> Ready

Ready -> Upgrading -> Verifying -> Ready

Pending/Installing/Upgrading/Verifying -> Failed

Ready/Failed -> Uninstalling -> Deleted
                         \----> DeleteFailed
```

建议 phase：

```go
const (
    AddonPending       AddonInstallationPhase = "Pending"
    AddonResolving     AddonInstallationPhase = "Resolving"
    AddonInstalling    AddonInstallationPhase = "Installing"
    AddonUpgrading     AddonInstallationPhase = "Upgrading"
    AddonVerifying     AddonInstallationPhase = "Verifying"
    AddonReady         AddonInstallationPhase = "Ready"
    AddonFailed        AddonInstallationPhase = "Failed"
    AddonUninstalling  AddonInstallationPhase = "Uninstalling"
    AddonDeleteFailed  AddonInstallationPhase = "DeleteFailed"
)
```

### 10.2 安装

1. API 持久化 `AddonInstallation`，generation 为 1；
2. Controller 校验 package、schema、cluster、master 和 Registry；
3. Controller 锁定 Artifact digest，生成 final values 和 hash；
4. Controller 以 `installationUID/generation/action` 为幂等键创建 Operation；
5. status 进入 `Installing` 并记录 operationRef；
6. Operation executor 安装并检查资源；
7. Operation 成功后 status 进入 `Ready`，更新 observedGeneration；
8. Operation 失败后 status 进入 `Failed`，保留 resolved digest 和错误条件。

### 10.3 升级

用户修改 `packageRef.version` 或 values 后 generation 增加。Controller 对比 `observedGeneration`，生成 Upgrade Operation。升级前保留上一次成功的 package digest、values hash 和 Helm revision。

首版使用 Helm `atomic` 回滚 Helm release。Manifest engine 首版不承诺自动回滚；失败后保留 inventory 和条件，允许用户修正后重试。

### 10.4 卸载

`AddonInstallation` 使用 finalizer：

```text
addons.kubeclipper.io/uninstall
```

收到 deletionTimestamp 后：

1. phase 进入 `Uninstalling`；
2. Controller 使用 status 中已安装 revision 生成卸载 Operation；
3. Helm 使用 release name/namespace 卸载；Manifest 使用 inventory 删除；
4. 执行 NotFound/资源清理验证；
5. 成功后移除 finalizer；
6. 失败时保留资源和 finalizer，phase 为 `DeleteFailed`。

首版不提供伪装成功的 force delete。管理员如需强制遗弃，必须通过单独的 orphan 管理操作并保留审计记录。

### 10.5 并发与幂等

- 同一 Installation 同时只能存在一个非终态 Addon Operation；
- Operation 幂等键为 `installation UID + generation + action`；
- Controller 重启后通过 Installation status 和 Operation 查询恢复；
- Agent executor 必须将 Helm release name 和 namespace 作为幂等边界；
- Catalog tag 变化不影响正在执行的 digest；
- status 更新使用 resourceVersion 乐观并发控制。

## 11. 缓存设计

### 11.1 Server 元数据缓存

Server 定时对 Catalog manifest 执行 resolve/HEAD：

1. digest 未变化：更新 lastCheckedAt，不解析 package；
2. digest 变化：拉取 Catalog config；
3. 并发拉取 package config 和 schema，小型元数据写入数据库；
4. 同步全部成功后原子切换 active catalog revision；
5. 失败则继续提供上一次成功 revision。

Console 列表只读取本地 `AddonPackage`，不在请求路径访问 Registry。

### 11.2 Blob 缓存

Server 和 Agent 使用 content-addressed cache：

```text
/var/lib/kubeclipper/addons/blobs/sha256/<digest>
```

- 同 digest 下载使用 `singleflight`；
- 下载完成前写临时文件，校验 digest 后原子 rename；
- 只清理未被 active catalog 和 Installation revision 引用的 blob；
- Registry 短暂不可用时允许使用已校验缓存；
- cache 命中不跳过 schema 和 executor version 校验。

## 12. Console 设计

Console 完全由 API 驱动：

- 插件市场：读取 `AddonPackage`；
- 分类、标题、描述、版本和 deprecated 状态来自 package；
- 安装表单由 JSON Schema 和 UI Schema 渲染；
- 集群插件页读取 `AddonInstallation`；
- 安装、升级、卸载状态读取 phase、conditions 和 operationRef；
- Operation 日志复用现有日志页面；
- 不再维护 `PLUGINS`、`STORAGES`、固定 `v1` 或按插件名分支的详情组件；
- package 暂时从 Catalog 消失时，已安装实例仍按 resolved digest 展示，不崩溃、不丢失卸载入口。

## 13. 代码模块划分

建议新增：

```text
pkg/addon/
  catalog/       Catalog 同步、校验和 active revision
  artifact/      OCI pull、descriptor、缓存
  package/       AddonPackage manifest 解析和 schema
  resolver/      package、values、registry、executor 解析
  controller/    AddonInstallation reconcile
  operation/     Operation plan 编译
  inventory/     Manifest 资源 inventory

pkg/executor/
  registry.go
  helm/
  manifest/
  condition/

pkg/apis/addons/v1/
pkg/models/addon/
pkg/generated/... 现有 codegen 输出
```

API group：

```text
addons.kubeclipper.io/v1
```

现有 `pkg/component` 在新插件全部迁移后删除。非插件性质的 cluster/CNI/CRI stepper 不在本设计中迁移。

## 14. 现有代码切换

最终删除：

- `pkg/component/{metallb,nfs,nfscsi}`；
- `component.Register`、`RegisterTemplate`、`RegisterAgentStep`；
- server/agent main 中插件 blank import；
- `/config.kubeclipper.io/v1/components`；
- `PATCH /core.kubeclipper.io/v1/clusters/{cluster}/plugins`；
- `Cluster.Addons` 作为插件可写状态；
- Console 的插件名称和 UI 特例。

保留：

- Operation Engine v2 的存储、API、状态机和 target fencing；
- Operation/Task 状态、Task terminal result 和日志；
- `OperationTask` 持久化节点绑定、Agent List/Watch 与 Node name + UID 校验；
- v2 Operation 的取消、超时、幂等投递和分类重试能力；
- Cluster `ImageRegistry` 与 Registry 解析；
- 现有身份认证、RBAC、审计和 API machinery。

该版本不承诺读取旧 `Cluster.Addons` 并自动迁移。升级说明必须要求用户在升级前卸载旧插件，或由发行流程提供一次性离线迁移工具；一次性工具不进入长期运行架构。

## 15. Operation Engine v2 前置项

新 Addon Controller 依赖 `docs/superpowers/specs/2026-07-26-operation-engine-v2-design.md`，执行接入前必须满足：

1. Addon Operation 由 v2 reconciler 执行，handler 和 Addon Controller 不直接调用 delivery；
2. Operation terminal status 不可逆，旧 attempt 和迟到结果不能覆盖；
3. Helm/Manifest executor 具备稳定 Task UID/payload digest，并通过 partial-effect 故障测试证明重复 Reconcile 可收敛；
4. Addon 状态只能由 Controller 根据 terminal Operation 提交；
5. executor identity 缺失必须 hard error，不得静默跳过；
6. Operation status、Installation status 和日志更新必须覆盖 server/agent 重启、API/网络断连、Watch `410 Gone` 后 relist、重复事件和重复 Task 场景；
7. Addon Controller 不直接创建 Task；只有 Operation Controller 能按 Step barrier 创建下一批 Task。

## 16. 安全边界

- OCI ref 安装时必须解析并锁定 digest；
- Catalog 同步只解析静态元数据，不执行插件内容；
- Agent 不执行插件提供的任意 shell；
- values 和 executor payload 必须通过结构化 schema 校验；
- Helm/Manifest executor 限制文件路径在 content-addressed cache 内；
- Registry password 不写入 Operation、Installation、日志或 package cache；
- 插件安装权限沿用 KubeClipper 集群管理权限，新增 AddonPackage sync 管理权限；
- 首版至少记录 catalog digest、package digest、安装人、目标集群和 Operation；
- Artifact 签名策略作为后续增强，数据模型预留 verification condition，不阻塞首版。

## 17. 可观测性

建议指标：

```text
kubeclipper_addon_catalog_sync_total{result}
kubeclipper_addon_catalog_sync_duration_seconds
kubeclipper_addon_artifact_pull_total{result,cache_hit}
kubeclipper_addon_reconcile_total{action,result}
kubeclipper_addon_operation_duration_seconds{action,engine,result}
kubeclipper_addon_installations{phase,engine}
```

日志统一携带：

```text
addonInstallation
cluster
package
packageVersion
artifactDigest
operation
generation
```

## 18. 测试策略

### 单元测试

- Catalog 重复版本、无 digest、错误 media type；
- OCI config/layer 解析；
- values 合并和 JSON Schema 校验；
- Cluster.ImageRegistry 为空/存在时的镜像映射；
- package resolver 与 executor version 校验；
- Installation 状态机和幂等键；
- Operation plan 的 install/upgrade/uninstall 输出；
- Helm/Manifest executor 输入校验；
- cache singleflight、digest 校验和回收引用。

### 集成测试

- 本地 OCI Registry + 测试 Catalog；
- 发布新 package 后动态发现，无 server/agent 重启；
- Helm 插件安装、升级、卸载；
- Manifest 插件安装和卸载；
- Agent 断线、Operation 失败后 Installation 进入 Failed；
- server 重启后继续观察已有 Operation；
- Catalog tag 更新不改变正在执行的 digest；
- Registry 不可用时使用已验证缓存；
- 卸载失败时 finalizer 和 Installation 保留；
- 重复 reconcile 不创建重复 Helm release。

### E2E 验收

使用 MetalLB、NFS CSI 和 metrics-server：

1. 三者全部以 OCI Addon Artifact 发布；
2. KubeClipper 源码中没有三个插件名称的安装分支；
3. Catalog 更新后 Console 自动出现新版本；
4. 安装默认继承目标集群 ImageRegistry；
5. 安装、升级、卸载均能查看 Operation 日志；
6. KubeClipper 和 Agent 不重新编译也能新增第四个 Helm 插件。

## 19. 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| Helm Chart 镜像 values 字段不统一 | 无法自动替换私有 Registry | package 显式声明 image entries 和 value paths，不做运行时猜测 |
| server 与目标 master 对 Registry 网络可达性不同 | server 可发现但 agent 无法拉 Chart | Catalog sync 与安装前分别做可达性检查，错误明确进入 Resolving/Failed |
| Operation 当前错误传播不可靠 | Installation 状态错误 | 将第 15 节正确性修复作为硬前置 |
| Catalog tag 被覆盖 | 同一版本内容变化 | 安装锁定 digest，package cache 以 digest 为键 |
| Manifest 卸载误删资源 | 用户资源损失 | 安装时保存 inventory 和 UID，卸载只删除受管 revision |
| 多 server 同时 reconcile | 重复创建 Operation | informer queue + resourceVersion + installation/action/generation 唯一键 |
| Registry 凭证泄漏 | 安全事件 | 仅运行时解析，禁止持久化到 Operation 和日志 |
| 不兼容切换导致旧 Addon 丢失管理 | 升级风险 | 明确 major 版本变更，提供升级前检查或一次性迁移工具 |

## 20. 验收标准

本设计完成实现的判断标准：

1. 发布一个新的 Helm Addon Artifact 和 Catalog entry 后，KubeClipper 无需修改源码、重新编译或重启即可发现；
2. 用户只提交 package、version 和业务 values，不需要重复配置 Registry；
3. Addon Controller 为安装、升级、卸载生成结构化 Operation；
4. Agent 仅使用通用 Helm/Manifest/Condition executor；
5. Installation 只有在 Operation terminal success 后进入 Ready 或完成删除；
6. server/agent 重启、重复 reconcile 和网络重试不会创建重复 release；
7. 旧插件名称不再出现在 server、agent、console 的静态注册或条件分支中；
8. 第四个第三方 Helm 插件可以独立发布并完成完整生命周期。
