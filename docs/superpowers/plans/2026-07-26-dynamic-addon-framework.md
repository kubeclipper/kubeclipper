---
change: dynamic-addon-framework
design-doc: docs/superpowers/specs/2026-07-26-dynamic-addon-framework-design.md
base-ref: 6e17ee7
branch: feat/dynamic-addon-framework
depends-on: docs/superpowers/plans/2026-07-26-operation-engine-v2.md
---

# 动态 Addon 框架实现计划

## 实施原则

- 目标版本不兼容旧插件 API；
- 实施期间允许旧代码暂时存在，但不为旧 API 增加 adapter 或双写逻辑；
- 每个阶段必须有独立测试和可观察结果；
- Addon Controller 只协调，Operation/Agent 负责执行；
- 插件内容只允许声明式 OCI Artifact，不加载第三方 Go 代码；
- Addon API、OCI Catalog 和 resolver 可与 Operation Engine v2 并行开发；Addon Controller 执行和通用 Executor 集成必须等待 Operation Engine v2 Phase 4 Gate 完成。

## Phase 0：Operation Engine v2 前置门禁

执行 `docs/superpowers/plans/2026-07-26-operation-engine-v2.md` 到 Phase 4 Gate，并满足：

- v2 Operation 由单一 reconciler 执行，server 重启后可以接管；
- terminal status 不可被旧 attempt 覆盖；
- Controller 创建绑定 Node name + UID 的稳定 OperationTask，Agent 单 worker 在重启后重新 Reconcile 遗留 Running Task；
- cancel、timeout、retry 和 target fencing 已通过故障注入测试；
- Addon Operation builder 声明 Cluster UID targetRef 和 Verify Step；Helm/Manifest executor 对同一 Task UID 重复 Reconcile 可收敛；canonical ExecutionLock 由服务端生成；
- Addon Controller 只观察 terminal status，不直接调用 delivery。

Addon Phase 1-3 的 API、Catalog、cache 和 resolver 可以在此前并行开发；Phase 4 通用 Executor 的端到端接入和 Addon Controller 启用受此门禁约束。

## Phase 1：Addon API 与存储

### T1.1 新增 addons API group

**新增目录**：

```text
pkg/scheme/addons/v1/
pkg/apis/addons/v1/
pkg/models/addon/
pkg/generated/clientset/.../addons/
pkg/generated/informers/.../addons/
pkg/generated/listers/.../addons/
```

资源：

- `AddonPackage`、`AddonPackageList`；
- `AddonInstallation`、`AddonInstallationList`。

`AddonPackage.metadata.name` 使用 `<package>-<normalized-version>`，原始 package name、SemVer 和 channel 分别保存到 spec/labels，避免多版本资源同名冲突。

执行现有 codegen，补充 deepcopy、client、informer、lister、REST registry 和 OpenAPI 定义。

### T1.2 AddonPackage 只读 API

实现：

```text
GET /api/addons.kubeclipper.io/v1/packages
GET /api/addons.kubeclipper.io/v1/packages/{resourceName}
POST /api/addons.kubeclipper.io/v1/packages:sync
```

普通用户无 create/update/delete package 权限；sync 仅管理员可调用。

### T1.3 AddonInstallation CRUD

实现创建、列表、查询、更新、删除和 label selector。创建时只做同步校验与持久化，不执行安装。

### T1.4 RBAC 与审计

在 `pkg/server/config.go` 增加 packages/installations 资源权限，记录安装、更新、删除和 catalog sync 审计事件。

## Phase 2：OCI Catalog 与缓存

### T2.1 Addon server 配置

**文件**：

- `pkg/server/config/config.go`
- `cmd/kubeclipper-server/app/options/*`
- `kubeclipper-server.yaml`
- deploy config/template

增加：

```yaml
addon:
  catalogRef: ""
  registryRef: ""
  syncInterval: 10m
  cacheDir: /var/lib/kubeclipper/addons
  cacheMaxSize: 10Gi
```

未配置 catalogRef 时 Addon Catalog 功能禁用，API 返回明确状态。

### T2.2 OCI artifact client

**新增目录**：`pkg/addon/artifact`

实现：

- 使用 `oras-go/v2` 拉取 generic OCI Artifact；
- resolve tag -> digest；
- media type 与 descriptor 校验；
- config 和指定 layer 拉取；
- Registry TLS、CA 和认证复用 `Registry`；
- 凭证不进入日志；
- digest mismatch hard error。

### T2.3 Catalog parser

**新增目录**：`pkg/addon/catalog`

实现 Catalog v1alpha1 schema、重复检测、digest 强制、版本排序和 active revision。

### T2.4 Package parser

**新增目录**：`pkg/addon/package`

实现 AddonPackage manifest、Helm/Manifest engine 校验、schema layer、image entries 和 health checks 校验。

### T2.5 Content-addressed cache

实现 digest 路径、singleflight、临时文件 + 原子 rename、引用保护和容量回收。增加 cache hit/miss 指标。

### T2.6 Catalog sync controller

定时同步 Catalog，仅在全部新 package 元数据校验成功后切换 active revision。Registry 故障时继续提供上次成功缓存。

## Phase 3：Resolver 与 Addon Controller

### T3.1 Values resolver

**新增目录**：`pkg/addon/resolver`

实现：

- Chart/package/user/system values 深度合并；
- JSON Schema 校验；
- 稳定 JSON 序列化和 SHA-256 values hash；
- 保留路径检测，禁止用户覆盖系统镜像注入值。

### T3.2 Cluster Registry resolver

复用 `componentutils.ResolveImageRegistry` 的能力并迁移到不依赖 `pkg/component` 的公共包。

规则：

- 无 Cluster.ImageRegistry：保留 original repository；
- 有 Registry：写入 `<host>/<mirrorRepository>`；
- package 未声明 mapping：不猜测；
- 认证信息不写入 resolved values/status。

### T3.3 Operation plan compiler

**新增目录**：`pkg/addon/operation`

根据 package engine 生成 install、upgrade、uninstall、verify Operation。选择一个可用 master Agent，写入 installation、cluster、generation 和 action labels。

### T3.4 AddonInstallation controller

**新增目录**：`pkg/addon/controller`

实现 informer/workqueue reconciler、phase 状态机、observedGeneration、operationRef、terminal status 观察、错误 conditions 和 finalizer。

### T3.5 幂等与并发

使用 `installationUID/generation/action` 唯一键避免重复 Operation；使用 resourceVersion 防止多 server 重复提交。

## Phase 4：通用 Executor

### T4.1 Executor command 协议

**文件**：

- `pkg/scheme/core/v1/operation_types.go`
- `pkg/service/task/handler.go`
- 新增 `pkg/executor/registry.go`

新增 `CommandExecutor` 和 versioned `ExecutorCommand`。未知 apiVersion/kind hard error。

### T4.2 HelmRelease executor

**新增目录**：`pkg/executor/helm`

能力：

- 从 OCI digest 获取 Chart layer；
- Helm SDK UpgradeInstall、Upgrade、Uninstall；
- atomic、wait、timeout、historyMax；
- 结构化 result：release name、namespace、revision、status；
- dry-run 和日志脱敏；
- 幂等重试。

### T4.3 ManifestApply executor

**新增目录**：`pkg/executor/manifest`

能力：

- OCI manifests layer；
- 受限模板渲染；
- Server-side Apply；
- inventory 记录 GVK/namespace/name/UID；
- 按 inventory 删除；
- dry-run。

### T4.4 ResourceCondition executor

**新增目录**：`pkg/executor/condition`

支持 Deployment、DaemonSet、StatefulSet、Job 和通用 condition；支持 `Ready` 与 `NotFound`。

### T4.5 Agent 注册

server 只编译 plan，agent 注册 executor。删除对插件包的 blank import 依赖。补充 server/agent executor protocol version 测试。

## Phase 5：Console

在 Console 仓库另建实现分支，完成：

### T5.1 Addon Catalog 页面

动态显示 package、分类、版本、描述和可用状态。

### T5.2 Schema 表单

复用现有 FormRender，输入 AddonPackage config schema 和 UI schema，删除插件名称特例。

### T5.3 Cluster Addon 页面

读取 AddonInstallation，显示 phase、conditions、version、operation 和日志入口。

### T5.4 安装、升级、卸载交互

安装创建 Installation；升级 PATCH packageRef/values；卸载 DELETE Installation。所有长任务展示 operationRef。

### T5.5 删除硬编码

删除：

- `PLUGINS`、`STORAGES` 名单；
- 固定 plugin version `v1`；
- KubeSphere/MetalLB 详情页分支；
- 按插件名维护的 uiSchemas。

## Phase 6：内置插件迁移与旧代码删除

### T6.1 发布 MetalLB Artifact

将 MetalLB Chart、values schema、image mappings 和 health checks 发布为 OCI Artifact，作为 Helm executor 验证样例。

### T6.2 发布 NFS CSI Artifact

优先使用上游 Helm Chart；如无满足需求的 Chart，使用 Manifest engine package。

### T6.3 处理 NFS Provisioner

该插件已 deprecated。目标版本不迁移或只发布 deprecated package，由产品决策确认；不保留 Go 实现。

### T6.4 删除旧插件 API

删除：

- `pkg/component/metallb`、`nfs`、`nfscsi`；
- component/template/agent step registry；
- server/agent blank imports；
- component config API；
- cluster plugins PATCH API；
- `Cluster.Addons` 字段和相关 schema/handler/controller；
- 旧 Console 插件路径。

### T6.5 升级门禁

`kcctl upgrade` 在检测到旧 `Cluster.Addons` 非空时拒绝直接升级并给出清理说明，或运行一次性迁移工具。长期服务代码不维护双模型。

## Phase 7：验证与发布

### T7.1 单元测试与静态检查

```bash
go test ./pkg/addon/... ./pkg/executor/... ./pkg/apis/addons/...
go test ./pkg/service/delivery/... ./pkg/service/task/...
go vet ./pkg/addon/... ./pkg/executor/...
```

### T7.2 集成测试

测试矩阵：

- registry anonymous/auth/TLS/custom CA；
- catalog unchanged/updated/invalid/offline cache；
- Helm install/upgrade/uninstall/failure/timeout；
- Manifest apply/delete/inventory；
- Cluster.ImageRegistry empty/configured；
- server restart、agent disconnect、duplicate reconcile；
- deletion finalizer failure/retry。

### T7.3 E2E

```text
1. 启动本地 Registry
2. 推送 Catalog、MetalLB、NFS CSI、metrics-server Artifact
3. 部署 KubeClipper 与目标 Kubernetes 集群
4. 从 Console/API 动态发现插件
5. 安装、升级、卸载并校验 Operation 日志
6. 发布第四个插件，不重启 KubeClipper 即发现并安装
```

### T7.4 文档

补充：

- Addon Package author guide；
- OCI publish CLI；
- Catalog maintenance guide；
- values schema/image mapping guide；
- 平台升级和旧插件清理说明；
- 故障排查与缓存管理。

## 推荐执行顺序

```text
Phase 0
  -> Phase 1
  -> Phase 2
  -> Phase 3
  -> Phase 4
  -> Phase 5
  -> Phase 6
  -> Phase 7
```

Phase 1 与 Phase 2 可在 API schema 稳定后部分并行；Phase 3 依赖 Phase 0、1、2；Phase 4 与 Phase 3 通过 executor payload schema 协作；Console 在 API v1alpha1 固定后开始。

## 建议提交拆分

1. `fix(operation): propagate terminal delivery errors`
2. `feat(addon): add package and installation API types`
3. `feat(addon): add OCI catalog and content cache`
4. `feat(addon): add package and values resolver`
5. `feat(addon): add installation reconciler`
6. `feat(executor): add versioned executor command`
7. `feat(executor): add Helm release executor`
8. `feat(executor): add manifest and condition executors`
9. `feat(console): add dynamic addon catalog and installation views`
10. `chore(addon): migrate builtin plugins to OCI packages`
11. `refactor(component): remove legacy plugin framework`
12. `test(addon): add catalog and lifecycle e2e coverage`

## Definition of Done

- 设计文档中的 8 条验收标准全部通过；
- 相关单元、集成和 E2E 测试通过；
- server、agent、console 中没有具体插件名称的执行分支；
- 新插件发布不需要 KubeClipper 代码变更和二进制发布；
- 插件默认继承目标集群 Registry；
- Operation 失败不会提交 Ready 或完成删除；
- Catalog/Registry 短暂不可用不影响已缓存插件的查看和重试；
- 旧插件框架和公开 API 已删除，并提供清晰的升级说明。
