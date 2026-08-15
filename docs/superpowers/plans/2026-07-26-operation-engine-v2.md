---
change: operation-engine-v2
design-doc: docs/superpowers/specs/2026-07-26-operation-engine-v2-design.md
base-ref: 6e17ee7
branch: feat/operation-engine-v2
status: proposed
decision_date: 2026-08-05
target_transport: kubeclipper-api-https
---

# Operation Engine v2 无 NATS 实施计划

> 实施状态（2026-08-07）：v2 Operation/Task/Lock、HTTPS Agent、mTLS 注册、单 worker、业务入口迁移、备份状态观察和 server composition 已落地；旧 NATS runtime、配置、目录、依赖和旧 Operation storage route 已删除。`core/v1.Operation` 相关类型仍作为部分业务 plan 的内部转换输入保留，后续应在业务 builder 完全改为 v2 类型后删除生成 client/informer 和该 plan 类型。

## 1. 实施原则

- 最终发布物不包含 NATS 进程、配置、端口、证书、代码和 Go 依赖；
- 不兼容旧 Agent、旧 Operation，不做 bridge、双栈、fallback 或数据迁移；
- 在 `operations.kubeclipper.io/v1alpha1` 实现 Operation、OperationTask 和 ExecutionLock，最终发布删除旧 `core/v1.Operation` API/runtime；
- 所有 Task 在执行前必须先持久化到 etcd；Watch 只负责通知；
- Controller 独占 Operation status 和 Step 顺序；Agent 正常负责 Running Task status，Controller 可在 deadline + termination grace period 后 CAS 终结失联的 Running Task；Controller 与 API handler 共用注入的 typed `OperationStore`，不通过 HTTP 自调用；
- Agent 无本地任务数据库，重启后从 etcd 中的 Running Task恢复；
- Controller 按 Step retryLimit 为失败 Node创建新 Task attempt；人工 `/retry` 在原 Operation 上继续第一个未完成 Step；
- 同一 Cluster 的 Operation 按 `creationTimestamp + resourceVersion + UID` 严格串行；后续 Operation 保持 Pending，前序 Operation terminal 后才允许继续；
- 外部副作用和 Task status 不可原子提交，不承诺 exactly-once；
- 所有 executor 必须支持同一 Task UID 重复 Reconcile；
- Operation/Task 默认不支持 DELETE；取消只持久化 `desiredState=Cancelled`，Cluster 删除时由受控流程按 target UID 清理安全终态历史；首版不实现通用 finalizer 或 history GC；
- Operation Controller 不读取 Node Ready 或 Lease，Node 离线只会让已创建 Task保持 Pending；
- ExecutionLock 只负责同一 Cluster 的互斥，不负责先后顺序；顺序由 Controller 查询同 target Operation 后选择最早未终态对象保证；
- 首版只有 Operation deadline，不增加 Step timeout；
- 首版证书只由 `kcctl join` 生成并通过现有 SSH 安装链路预置，不实现 CSR/bootstrap/自动 rotation；
- 每个 Agent 使用唯一且永不复用的 AgentID；Node name 等于 AgentID；`kcctl join` 为每个 AgentID 独立签发同时支持 clientAuth/serverAuth 的 mTLS certificate，安装及重新纳管禁止跨节点复用 AgentID 或 credential；Node object UID 只用于 Task reference 校验；
- 先跑通 no-op 垂直闭环，再抽象公共代码或批量迁移业务；
- 日志、Node Ready、Node Lease 和 Watch connection 不进入 Operation 正确性路径。
- OCI 是分发层：使用 Artifact 的业务在创建 Operation 前固定 digest，blob/credential 不进入 etcd；Agent/static server bootstrap 迁移不和 v2 最小闭环捆绑。

## 2. 交付顺序

```text
Phase 0  冻结最小契约和 NATS 清单
   |
Phase 1  Operation/Task/Lock + fake Agent 垂直闭环
   |
Phase 2  真实 Agent API、预置证书、单 worker、首个真实 executor
   |
Phase 3  业务动作和日志逐项迁移
   |
Phase 4  切换 Agent composition、删除 NATS、发布硬化
```

第一个架构 Gate 在 Phase 1，而不是 PKI、日志或真实 kubeadm 之后：

```text
create Operation
  -> acquire Lock
  -> create Task
  -> fake Agent List/Watch
  -> Pending -> Running -> Succeeded
  -> Controller advance Step
  -> Operation Succeeded
  -> release Lock
```

开发期间旧 NATS 路径可以暂时保留可构建，但一个业务动作只能完整走旧路径或完整走 v2。最终切换前，所有业务入口和 Agent composition 必须在同一 release branch 收敛到 v2。

## 3. Phase 0：冻结契约和迁移清单

### T0.1 建立 NATS inventory

建立机器可检查的清单：

- `pkg/simple/client/natsio`、`pkg/service/delivery`、`pkg/service/task` 的 import 和 call site；
- 所有 subject、request/reply、callback、连接状态和日志请求；
- cluster create/delete/upgrade、node add/remove、backup/restore、certificate、addon/component、CRI/registry、online command；
- `pkg/service/task/handler.go` 的 Step.RetryTimes 内存循环、`pkg/clusteroperation/operation.go` 的 AutomaticRetry/Retry，以及 `pkg/apis/core/v1/handler.go` 的原 Operation `/retry`；
- Agent registration、Get Node、Node status、Lease 和 logs；
- 每一项的当前 caller、builder、handler、Agent executor、结果消费者和测试位置。

为每项标记唯一迁移类别：

| 当前行为 | v2 owner |
|---|---|
| 修改节点或集群状态 | typed Operation builder/executor |
| 周期性健康/证书观察 | Agent -> Node status，server controller 汇总到 Cluster |
| Kubernetes API 已有数据 | kc-server cluster client |
| Node status / Lease | scoped HTTPS API |
| Operation logs | Agent local HTTPS endpoint |
| 任意在线 shell | 删除 |

重点提前处理当前同步调用：

- `pkg/controller/cluster_status.go` 的证书状态命令；
- `pkg/controller/clustercontroller/controller.go` 的 ServiceAccount token 获取；
- 同文件读取 apiserver certificate 的命令；
- `DeliverStep` 更新 CRI registry 的路径。

完成条件：每个 NATS call site 都有明确 owner 和删除 Phase；CI 禁止新增清单外 NATS 引用。

当前仓库至少需要覆盖以下已确认入口，inventory 以实际 `rg` 结果为准而不是把此表当静态白名单：

| 当前区域 | 已确认职责 |
|---|---|
| `pkg/simple/client/natsio` | NATS client/server、request/reply、subscription |
| `pkg/service/delivery` | server 投递 Operation/Step/Cmd、日志 request |
| `pkg/service/task` | Agent 注册、Node/Lease proxy、Task 执行和内存 retry |
| `pkg/controller/operationcontroller` | goroutine 投递、AutomaticRetry、旧 Operation finalizer |
| `pkg/apis/core/v1/handler.go` | Operation 创建/retry/cancel/log 等入口 |
| cluster/cronbackup/status controllers | `DeliverTaskOperation`、`DeliverStep`、`DeliverCmd` 同步调用 |
| deploy/join/config/server composition | NATS 证书、端口、进程和配置生成 |
| `go.mod/go.sum` | `nats.go`、`nats-server` 及仅由它们引入的依赖 |

### T0.2 冻结最小 API 契约

只冻结设计文档中的最小字段：

- Operation：target、desiredState、retryGeneration、timeout、ordered Steps/retryLimit、status/observedRetryGeneration/reason/message/deadline；
- Task：operation、retryGeneration、step、node、attempt、executor、inline payload、deadline、status/result；
- Lock：target、holder；
- Operation/Task terminal phase；
- outputs 和 payload 大小上限；
- Operation/Task DELETE 拒绝规则、cancel 和 `/retry` 行为。
- 删除旧 DryRun/ForceSkipError/ErrIgnore 执行语义；如产品保留 preview，只做纯 plan validation且不创建 Operation/Task/Lock；

明确禁止在 Phase 0 增加：

- fence、独立 Attempt 资源、Agent 内存中的整 Step retry loop；
- 为 API 对象额外计算的 plan/payload/result digest；外部 OCI Artifact digest 不在此列；
- Dispatch、Result、ExecutionLease、AgentSession；
- AgentCertificateRequest、bootstrap token、approver；
- DAG、表达式语言、通用 shell executor。

### T0.3 业务计划 fixtures

为每个现有 Operation 保存业务意图 fixture：

- 有序 Step ID；
- 每 Step 的目标 Node name + UID；
- typed executor 和版本化 payload；
- Operation timeout；
- 跨 Step 的 `stepID + nodeUID + outputKey -> payload field`；
- Cluster UID targetRef；
- 稳定 Operation name：owner UID + generation + action，或客户端一次生成并跨 HTTP retry 复用的 request UUID；
- 每个 executor 的 postcondition 和 partial-effect 边界。
- 每 Step 的 retryLimit，以及每个 executor 的稳定输入和 Observe/Act/Verify 行为；

fixture 不复制旧 NATS wire format。无法说明重复 Reconcile 行为的动作标记为 blocked，不进入批量迁移。

### Phase 0 Gate

- NATS inventory 完整且可由 CI 检查；
- 三资源最小 schema 和状态表经过 review；
- 所有当前业务动作已分类；
- 没有尚未说明用途的新资源或状态字段。

## 4. Phase 1：最小 API 与 no-op 垂直闭环

### T1.1 新增 operations API types

新增独立但最小的 operations API group：

```text
pkg/scheme/operations/v1alpha1/
  register.go
  operation_types.go
  operationtask_types.go
  executionlock_types.go
```

完成 scheme、deepcopy、OpenAPI、clientset、lister、informer 和 fake client。新 group 不提供旧 Operation conversion、bridge 或双写。

迁移期未迁移业务继续完整使用旧 NATS/core Operation，已迁移业务完整使用新 group。正式发布前删除旧 Operation 的 Commands、AutomaticRetry、ErrIgnore、旧 status、API/storage/client 和 runtime；旧 `RetryTimes` 的产品语义映射为 v2 Step `retryLimit`，不保留 Agent 内存循环实现。

### T1.2 storage、selector 和 status

复用 `pkg/server/registry` 模式新增 Operation/Task/Lock storage，并在 `pkg/models/operationv2` 提供统一 typed `OperationStore`：

- Operation status 只通过进程内 `OperationStore` 更新；Agent 只获得 Task `/status` 写入口，不提供 OperationTask 普通 Create/Update/Delete API；
- Task `spec.nodeRef.name` 和 `spec.operationRef.uid` 是 selectable fields；
- Operation `spec.targetRef.uid` 是 selectable field，`ListByTargetUID` 使用真实 storage 的强一致读取；
- `ListTasksByOperationUID` 使用真实 storage 强一致读取，供 attempt/terminal/Lock/cleanup 安全边界复核；
- Agent List/Watch 只按 `spec.nodeRef.name` 过滤；terminal Task保留在 informer cache 中并由 worker 忽略，不增加 active phase selector；
- Task event 直接从 immutable `spec.operationRef` 映射 Operation；不把客户端 label 当正确性依据；
- List/Watch 传递 resourceVersion 和 timeout；storage 的 APIStatus 必须原样映射，过旧 resourceVersion 返回 410 而不是 500；bookmark 不是首版要求；
- Agent List 返回原生 `OperationTaskList + ListMeta.resourceVersion`，Watch 返回标准 `watch.Event` 流；不能复用 Console pageable response；首版不实现 continue/bookmark；
- Task `/status` 请求包含 Task UID、resourceVersion 和 status，`OperationStore` 使用三者 CAS 处理 Agent Pending -> Running 与 Controller Pending -> Cancelled 的竞争；
- Agent 可以写 Pending -> Running 和 Running -> Succeeded/Failed/TimedOut；
- Controller 可以写 Pending -> Cancelled，并在 deadline + termination grace period 后将 Running -> TimedOut；
- Task attempt terminal phase 不可逆；Succeeded Operation 不可逆，Failed/TimedOut/Cancelled 只能通过 `/retry` 开启下一轮；
- Task startedAt/finishedAt 使用 server time；
- Lock 获取依赖 storage Create 原子语义；释放必须通过 `OperationStore.ReleaseLock(operationUID)` 校验 holder，并使用 Lock UID precondition，禁止按 name 无条件 Delete。

`OperationStore` 在 server 启动时由 `storageFactory` 构造一次并注入 Controller。Controller reconcile 不访问 `storageFactory`、裸 `rest.StandardStorage` 或 etcd client；API handler 和 Controller 共用 Store 的 CAS、transition validation、Task terminal immutable、Succeeded Operation immutable 和 Lock holder 校验。Controller 内部使用受信任 actor，不通过 kc-server loopback HTTP，也不依赖普通 IAM RBAC。

使用真实 API storage 做集成测试，不只测 fake client。

注册最小 route，不开放额外 CRUD：

- Operation Create/List/Watch/Get、`/cancel`、`/retry`；
- OperationTask List/Watch/Get 和 `/status`；
- OperationTask `/logs` 只作为 Phase 3 的 server proxy；
- 不注册普通 Operation/Task DELETE、Task Create/Update、Lock 外部 CRUD；
- Cluster lifecycle 历史清理只调用受信任进程内 Store 方法。

### T1.3 validation 和生命周期

Operation validation：

- metadata.name 必填并作为 create idempotency key，禁止 generateName；AlreadyExists 后调用方 GET 并比较 immutable plan；
- Create strategy 强制 desiredState=Active、retryGeneration=0、phase=Pending并丢弃客户端 status；不注册普通 Operation Update；
- target kind/name/UID 必填；
- Create 时 GET target 并校验 kind/name/UID，拒绝用伪造 UID 绕开 Cluster Lock；
- Steps/targets 非空，Step ID 唯一，同 Step Node UID 不重复，targets name/UID 必填，inputs 只能引用更早 Step；
- executor/payload schema 已注册；
- Operation timeout、对象大小、Step 数、targets 数有硬上限；Step 不设置 engine-level timeout；
- 冻结并测试设计文档中的首版常量：timeout 1m-24h/default 90m、Agent kill grace 30s、server termination grace 2m、Operation 512KiB、256 Steps、每 Step 1000 targets/128KiB payload、Task 256KiB、retryLimit 0-3、message 4KiB；
- 等待前序 Operation/Lock 的排队时间不消耗 timeout；获得 Lock并开始本 generation 时才由 server time 设置 deadline；
- plan 创建后不可变；cancel 只允许非终态 Operation 的 Active -> Cancelled，并使用 resourceVersion CAS；`/retry` 只允许 Failed/TimedOut/Cancelled、无 active Task且仍为该 target 最新 Operation，并使用 resourceVersion CAS 递增 retryGeneration；
- Create API 不做“当前是否已有 Running Operation”的非原子预检查；同 Cluster 的新 Operation 统一创建为 Pending，由 Controller 顺序调度；
- retryLimit 有小的硬上限；每个 Step/Node 第一次出现 Task 的 base generation 最多运行 `1 + retryLimit` 次；
- 后续每个人工 retry generation 对已有失败 Step/Node只允许一个 Task，不重置 retryLimit；在该 generation 才首次到达的新 Step仍有自己的 base generation 自动额度；
- 首版不限制人工 `/retry` 总次数；retryGeneration 只作为单调请求序号，不作为配额；
- Operation status 不保存 currentStep 或通用 Conditions；reason 使用有限枚举，message 是有界且脱敏的当前结果摘要；
- 修改集群的 targetRef 必须是 Cluster UID。

Task validation：

- name 等于包含 retryGeneration 和 attempt 的确定性 Task name；
- Create strategy 强制 phase=Pending并丢弃 status；
- Operation UID、Step ID 和 Node reference 与 Operation plan 一致；
- spec immutable，Task 没有 desiredState；
- Agent 只能写合法 status transition；
- stale terminal update 返回 Conflict；Agent GET 最新 Task，已是预期 terminal 则确认完成，否则按最新 phase、deadline 和实际状态重新归约；
- Operation cancel 不修改 Running Task；deadline 到达后 Agent 终止命令并提交 TimedOut，Agent 失联超过 termination grace period 时 Controller 可 CAS 终结 Running Task；
- Succeeded reason 必须为空；Failed、TimedOut 和 Cancelled reason 必须与 phase 匹配；
- Task TimedOut 只允许由持久 deadline 触发；executor 内部更短的技术超时写 Failed；
- 同一 `(retryGeneration,stepID,nodeUID,attempt)` 唯一；任一历史 attempt Succeeded 后不得创建新 attempt；
- retryLimit 只统计进入过 Running 的 Task；被 Controller 从 Pending 取消的 sibling 不消耗执行次数；
- outputs 只允许 Succeeded、声明 key 和 16 KiB 总量。

Lock validation：

- name 等于 target kind + UID 的确定性名称；
- holder Operation 存在且 targetRef 完全一致；
- 只有 Operation Controller identity 可以 Create/Delete；
- Lock 无 status、timeout、renew 或 update 路径。

生命周期：

- Operation 和 Task API 默认不向用户、Agent 或普通 Controller 提供 DELETE；Cluster 删除流程在删除 Operation 完成、无 active Task 且 Lock 已释放后，按 target Cluster UID 清理该 Cluster 的 Operation/Task 历史；
- cancel 只允许 `Operation.spec.desiredState: Active -> Cancelled`；
- Operation terminal 且已无 Pending/Running Task后，先持久化 terminal status，再删除 Lock；普通场景历史保留；
- 首版不实现通用 TTL/history GC，只增加 Operation/Task 数量和 etcd 占用告警；Cluster 删除清理按 target UID 执行，必须幂等且只清理安全终态历史；
- Cluster lifecycle 在删除业务 Operation terminal、无 active Task且 Lock 已释放后调用 `CleanupByTargetUID`，成功后才删除 Cluster 对象；失败时保持 Terminating 重试，不给 Operation/Task 增加 finalizer；
- Agent 本地日志使用独立时间/容量策略清理，不依赖 Operation 删除。

### T1.4 最小 reducer 和 Controller

在最终保留的 `pkg/controller/operationv2` 内实现小型纯函数：

- derive current Step；
- reduce Task phases；
- cancel/failure/timeout/retry reducer；
- attempt aggregation 和 deterministic Task/Lock name。

reducer 必须用固定优先级：全部计划成功 > 等待 Running Task > desiredState Cancelled > deadline TimedOut > retry 耗尽 Failed > 继续当前 Step。所有 Task/Operation terminal phase 不可逆，竞态由 resourceVersion CAS 决胜。

不要提前创建通用 state/identity framework。

Controller：

- informer 监听 Operation、Task 和 Lock；
- rate-limited workqueue 只保存 Operation key；
- Task event 通过 immutable `spec.operationRef` 映射回 Operation；
- Operation terminal 事件通过 target UID index 重新入队该 Cluster 的 Pending Operation，保证前序完成后后续 Operation及时获得 reconcile；
- reconcile 只通过 informer/lister 读取 Operation、Task 和 Lock，不读取 Node Ready/Lease；
- 所有写入通过注入的 typed `OperationStore`，不通过 HTTP 自调用或 `storageFactory` 直写；
- reconcile 先处理 cancel/retry generation，再获取 Lock；
- Lock 冲突明确 `RequeueAfter`；
- 同一 target 存在更早的 Pending/Running Operation 时，当前 Operation 不获取 Lock、不创建 Task，等待前序 Operation terminal 后重新入队；
- 顺序判断使用 `OperationStore.ListByTargetUID` 的强一致读取，不只依赖可能滞后的 informer lister；
- 等待 Lock 或 deadline 时，`RequeueAfter` 不晚于 Operation deadline；deadline 到达且仍有 Running Task时，再以 termination grace period 为界 requeue，执行最终 TimedOut 收敛；
- 进入当前 Step 后立即为未成功 Node物化确定性 Task attempt；部分 Create 后退出时补齐缺失 Task；
- Node/Agent 离线时 Task保持 Pending，由 Operation deadline 收敛；
- 只物化当前 Step Task；
- 失败后取消同 Step Pending sibling，Running sibling自然结束；当前 Step 无 Running 后才创建 retry attempt；
- 自动 retry 只在每个未成功 Node 都有下一次额度时启动；任一必需 Node 已耗尽额度则整个 Step Failed，不能只重跑其余 Node；人工 `/retry` 为当前 Step 每个未成功 Node各授权一次 attempt；
- deadline 到达时把 Pending Task写为 Cancelled；Agent 负责终止 Running Task并提交 TimedOut，失联超过 termination grace period 时 Controller 可 CAS 写 TimedOut；
- 自动 retry 只到 retryLimit；人工 `/retry` 在原 Operation 上只恢复第一个未完成 Step，成功 Step/Node不重跑；
- retry handler 只用 resourceVersion CAS 更新 desiredState/retryGeneration，不写 status、不获取 Lock、不创建 Task；Controller 先检查 target 最新 Operation，过期 retry 只推进 observed generation，发出 `RetrySkippedNotLatest` 审计事件并保留原 terminal phase/reason/message，不创建 Task；仍为最新时获取 Lock 后再次校验，仍最新才设置 Pending/Running/deadline；
- observedRetryGeneration 只表示已处理，不表示 Task 已物化；仅对仍为最新 Operation 的 generation，reconcile 才补齐缺失 Task；已跳过 generation 不创建 Task；真正开始 retry 时清空旧 reason/message/finishedAt并设置新的 startedAt/deadline；
- 创建新 attempt、写 Operation terminal、释放 Lock或清理历史前，通过 `OperationStore.ListTasksByOperationUID` 强一致读取并重验条件；cache 只允许导致晚推进，不能支撑不可逆结论；
- Store/List/Watch/CAS timeout、Conflict、leader loss 和 response loss 只返回并重排队，不得写业务 Failed/TimedOut或释放 Lock；
- 不启动业务 goroutine，不等待 Agent response；
- leadership 丢失时 worker 跟随 manager context 停止。

### T1.5 fake Agent no-op loop

实现测试用 fake Agent 和 `Noop/v1` executor：

- initial List `spec.nodeRef.name=<self>` 的全部 Task，fake worker 忽略 terminal Task；
- 从 List resourceVersion 建 Watch；
- Pending -> Running CAS；
- 模拟 Controller 对未启动 Pending Task的 cancel/deadline Pending -> Cancelled CAS；
- 模拟 deadline + termination grace period 后 Controller 对遗留 Running Task 的 TimedOut CAS；
- 写 Succeeded/Failed/TimedOut status；
- 模拟 response loss、duplicate event、Watch disconnect 和 410；
- 模拟 Agent 在 Running 前后退出；
- 不引入 PKI、日志或真实 shell。

### T1.6 Phase 1 correctness tests

必须覆盖：

- 单 Operation、单 Step、单 Node 完成；
- 两 Step 不提前创建第二 Step；
- Step 内多 Node 全成功屏障；
- Node/Agent 离线不阻止 Task 创建，Pending Task在 Agent恢复后执行或在 Operation deadline 后取消；
- 同 Cluster 两个 Operation 只有一个 Lock holder；
- 两个 Pending Operation 按创建时间和 UID 顺序执行，后创建者不能先进入 Running 或创建 Task；
- 后序 Operation 排队期间 deadline 为空且不消耗 timeout，但 cancel 可立即收敛；
- Lock Delete 响应丢失后另一个 Operation 获取同名 Lock，旧 Operation 重试释放不得删除新 Lock；
- Lock 已创建、部分 Task 已创建、Task terminal 后 Controller 退出；
- Operation Create response 丢失后用同一 name 重试，不生成第二个计划；同名但 plan 不同必须报冲突；
- Running CAS response 丢失；
- Agent Pending -> Running 与 Controller Pending -> Cancelled CAS 竞争；
- Running Task 超过 deadline 后 Agent 终止命令并写 TimedOut；Agent 失联超过 termination grace period 后 Controller CAS 写 TimedOut并释放 Lock；
- Agent terminal PUT 与 Controller grace-period TimedOut CAS 竞争；CAS 输方 GET 最新 Task并接受不可逆终态；
- cancel 与最后 Task success 并发；
- cancel response 丢失后 GET desiredState确认，不能用旧对象无条件重放；cancel/retry/status 请求均使用 name + UID + resourceVersion，旧对象或同名重建对象不能被写入；
- 一个 Task Failed 后 Pending sibling cancel、Running sibling自然结束；
- retryLimit 内只为失败/超时/取消 Node创建 attempt+1，已成功 Node不重跑；
- base generation 使用 `1 + retryLimit`，同一 Step后续每个人工 generation 只创建一个 Task；在人工 generation 才首次到达的新 Step仍获得自己的 base retryLimit；
- 一个 Node 自动额度耗尽而其他 Node仍有额度时，Step直接 Failed，不创建无法使 Step 成功的部分 retry；
- retry Task Create response 丢失、Controller restart 和 deterministic attempt 补建；
- `/retry` response 丢失后通过 spec/observed generation判断、并发请求 CAS、原 Operation继续以及新 deadline；
- retry generation observed 后、Task Create 前或部分 Create 后退出，Controller restart 必须补齐；
- 已存在更新的同 target Operation 时跳过旧 Operation retry，推进 observed generation、保留原 terminal status并记录原因；
- Pending sibling Cancelled 不消耗 retryLimit；
- 旧 attempt 仍 Running 时自动和人工 retry 都被阻止；
- 普通 Operation/Task DELETE 被 API 拒绝；Cluster 删除清理可删除该 Cluster 的安全终态历史；
- 伪造 target UID 和非法 Lock holder 被拒绝；
- 等待 Lock/deadline 的 RequeueAfter 在 controller restart 后重建；
- Node 永久离线且 Task Running 超过 termination grace period 时，Controller 将 Task/Operation 收敛为 TimedOut并释放 Lock；
- Task terminal status、Succeeded Operation 和 plan spec immutability。
- informer cache 落后于 Task Create/status 时，强一致安全边界复核阻止错误 terminal、新 attempt或释锁；
- etcd/API timeout、Conflict 和 leader loss 不改变业务 phase，恢复后从持久事实继续；
- Cluster 删除后的 Operation/Task 历史清理按 target UID 生效，重复清理安全，active Task/Lock 存在时不清理；

### Phase 1 Gate

在不启动 NATS 的测试环境中，no-op 两 Step Operation 能经过 server restart、leader switch、Watch disconnect/410、duplicate event、status response loss、cancel、自动 Step retry 和人工 `/retry`，且不重跑已成功 Node、不产生同 Node并发 attempt。

Phase 1 Gate 失败时不实现完整 Agent 或业务 executor。

## 5. Phase 2：真实 Agent 和首个真实 executor

### T2.1 kcctl join 预置每节点证书

扩展现有 deploy/join SSH 流程：

- `kcctl join` 生成唯一且永不复用的 AgentID，并预创建 `Node(name=AgentID)`；
- 使用现有 CA 为每个 AgentID 独立生成 cert/key，CN 为 `system:kc-agent:<AgentID>`、Organization 为 `system:kc-agents`、DNS SAN 为 AgentID，包含 clientAuth 和 serverAuth；同一张节点证书用于 Agent API client 和 Phase 3 日志 server，不复用 kcctl 或旧 NATS certificate；
- 只由能够读取本地 API CA 私钥的 `kcctl join` 签发；kc-server 不在线代签，Agent 私钥、CA 私钥和完整 client credential 不写入 etcd 或 ConfigMap；
- Node object UID 不作为证书身份，只写入 Task nodeRef，并由 Agent 启动后缓存校验；
- 将 cert/key/CA certificate、kc-server endpoint 和 AgentID 写入对应 Agent；CA private key 不下发；
- kc-server 配置独立 `system:kc-server` client certificate，用于访问 Agent 日志端点；
- 安装前校验证书文件权限和到期时间。

首版证书更新通过 kcctl 停止 Agent、重新签发、SSH 替换和重启完成。增加 expiry 指标/告警，但不实现 CSR、token、approver 或自动 rotation。

### T2.2 Agent API client 和现有 RBAC

实现 mTLS API client：

- Node GET/status、Lease Create-or-Update、Task List/Watch/Get 和 `PUT /operationtasks/{name}/status`；
- 有界 timeout 和 exponential backoff；
- 不在 client 内隐藏业务 retry 或状态机；
- 使用 API path/version 自然拒绝旧 Agent，不增加协议协商资源。

服务端复用现有 x509 authenticator 和 RBAC，不创建每 Agent 的 User、Role 或 RoleBinding，也不实现 NodeAuthorizer/NodeRestriction：

- 保留现有 listener 的可选客户端证书模式以兼容 kcctl 的其他认证方式，但 Agent 专用 route 必须要求 TLS peer certificate 已由配置的 CA 验证；无证书或证书校验失败直接返回 Unauthorized；
- certificate CN 映射为 user name，Organization `system:kc-agents` 映射为 group；
- 增加一个内置 `kc-agent` GlobalRole 和一个 `Kind: Group, Name: system:kc-agents` 的 GlobalRoleBinding，只授予 Node、Node/status、Lease、OperationTask 和 OperationTask/status 所需最小 verbs；
- 从 certificate CN 提取 AgentID 做资源归属校验；
- 强制 Task List/Watch 使用证书 AgentID 的 node name selector；
- Task Get 和 Task `/status` 校验 `nodeRef.name == AgentID`；
- 只允许自身 Node status，并按 Node name Create-or-Update 自身 Lease；
- 禁止 Agent 创建 Task、修改 Operation/Lock、修改 Task payload 或访问其他 Agent 的资源；请求参数和 body 不能覆盖证书 AgentID。

首版不实现 CRL、在线撤销或 Agent CSR。私钥泄露或 Agent 永久下线时，停止旧 Agent、删除/禁用旧 Node，并使用新的、永不复用的 AgentID 重新纳管。

### T2.3 OperationTask informer 和 single worker

- 使用生成的 OperationTask clientset 和 client-go SharedInformer/Reflector，不手写 List/Watch 状态机；
- initial List 处理本 Node 已有 Task，terminal Task只进 cache、不进执行队列；
- Watch 从 collection resourceVersion 开始；
- EOF/网络错误由 Reflector 退避；410 触发 relist 和 cache Replace；
- bounded in-memory queue；
- Task UID 去重；
- 本机 OS singleton lock；
- 遗留 Running Task优先；多个 Running 时 fail closed；
- 无 Running 时按 creationTimestamp + resourceVersion + UID 选 Pending；
- 启动 Pending 或进程重启后恢复 Running 前实时 GET Task；cache 只负责唤醒和排序；
- 确认 Running status 后才执行；
- 确认 terminal status 后才选下一 Task；
- API 断连期间不启动下一 Task；
- 遗留 Running Task deadline 已过时，Agent 终止命令并确认退出后写 TimedOut；重启后的 Agent 不创建新 Task，先 Observe 同一 Task。

不增加 bbolt、task inbox、Agent session 或本地完成记录。

### T2.4 Executor registry

实现单接口：

```go
type Executor interface {
    Reconcile(ctx context.Context, task TaskSpec, log io.Writer) (TaskResult, error)
}
```

- executor 使用版本化名字和 typed payload；
- 不要求底层命令字面幂等，但必须通过实际状态检查实现可重入、可验证、可收敛；
- 输入来自不可变 Task spec，不在每次 Reconcile 中重新生成随机 token、文件名或成员 ID；
- 成功返回前验证 postcondition；
- 同 Task UID 重复调用必须收敛；
- Observe/进程状态暂时不确定时保持 Task Running并做有界安全观察，不写 Failed、不创建新 attempt；只能由正常失败/postcondition失败或统一 deadline 终止路径收敛；
- runner 只处理 Success、Failed、TimedOut 三种执行结论；Operation cancel 不取消 Running Task context，deadline 到达时终止命令并等待退出；
- runner 以前台进程组执行命令；deadline 时 TERM、固定 kill grace 后 KILL，并 Wait 到受控进程组退出；禁止 `nohup`、shell `&`、daemonize 和不可跟踪后台修改动作；systemd 配置必须在 Agent 退出时清理受控子进程；
- API/Watch/status 短暂错误只做有界通信重试；不得用通用 Retryable 结果循环执行外部命令；
- 新的业务 attempt 只由 Controller 创建新 Task表达；
- 正常命令退出且 postcondition 成功为 Success，正常退出但失败为 Failed，被 deadline 强制终止且已退出为 TimedOut；不能证明可重入的 executor 不得接入；
- stdout/stderr 写 bounded local spool；
- 不注册通用 shell executor。

### T2.5 Node status 和 Lease

- Agent GET Node 并记录 UID；
- 低频更新 Node status、版本、能力、管理 IP 和 log port；
- certificate、API client、reflector、single worker 和 executor registry 可用后才报告 Ready；日志 endpoint 不参与 Ready；
- 按 Node name Create-or-Update 自身 Lease，使用 server time；
- Lease 过期只改变 liveness，不阻止当前 Step Task创建；
- 测试 Lease 过期不阻止 Task创建、不触发 retry；Task deadline 和 termination grace period 独立验证，不用 Lease 释放 Lock。

### T2.6 首个真实 executor

选择副作用小、postcondition 清楚的动作，例如只读 preflight 或声明式 manifest apply。不要选择 kubeadm reset、etcd restore 或 certificate rotation。

完成：

- typed payload schema；
- Operation builder；
- Reconcile 的 actual-state 检查；
- Observe/Act/Verify 的完整行为说明；
- partial-effect 和重复调用测试；
- 一个小型 output 注入下一 Step 的示例；
- server/Agent kill -9 和断网测试。

### Phase 2 Gate

真实 kc-agent 使用 SSH 预置的每节点证书，在 NATS 不可用时完成两 Step Operation。对 Reconcile 前、执行中、postcondition 后、terminal status 前逐点 kill -9，Agent 重启后重新 Reconcile 同一 Running Task并收敛。

额外验证 runner 的前台进程组约束：Agent kill -9、systemd restart 和 deadline TERM/KILL 后不存在仍修改目标的受控子进程；确认退出前不得写 TimedOut。

进入 destructive executor 批量迁移前，必须验证 Agent deadline 终止、termination grace period、Controller 强制 TimedOut 和 Lock 释放流程；不得用 Lease 过期直接释放 Lock。

## 6. Phase 3：业务和日志迁移

每类业务使用同一模板：

```text
fixture
  -> typed payload
  -> actual-state/postcondition
  -> stable-input/reentry classification
  -> Reconcile partial-effect tests
  -> high-level API persists desired business object
  -> business controller builds stable-name complete Operation
  -> controller observes terminal Operation
  -> business controller reduces owner status
  -> delete legacy NATS call site
```

业务对象 status 与 Operation Create 不做跨对象事务。Operation name 由 owner UID + generation + action 稳定生成；Controller 每次先 GET 稳定 name，只有 NotFound 才解析 mutable input并构建 plan。Create 成功、status reference 回写前退出时直接接管已有 Operation，不重新解析 tag；Create 的并发 AlreadyExists 使用 GET/compare 收敛。handler/controller 不启动投递 goroutine。

### T3.1 Cluster create/delete/upgrade

- kubeadm init/join/reset 拆成可观察 Step；
- control-plane/worker 顺序由 Operation Steps 明示；
- kubeadm config、static Pod、etcd member、certificate 和 Node 状态分别检查；
- upgrade 明确逐节点顺序；
- reset/delete 重复调用不能破坏已清理状态；
- 不能证明重入安全的动作不接入 v2，不用 shell 逃逸。

### T3.2 Node add/remove

- 所有集群成员变更使用 Cluster UID Lock，不增加 Node 子锁；
- Node name + UID 在计划创建时固定；
- drain、member removal、reset 是独立有序 Step；
- 最后一个 Agent Task 不能停止 Agent、删除证书或删除 Node；
- Operation terminal 后由 Controller 完成 Agent/Node 清理。

### T3.3 Backup/restore

- snapshot 文件使用对象存储或现有外部存储，Task 只保存 reference；
- snapshot 完成和有效性分别检查；
- restore 拆成可识别阶段，固定 artifact reference；
- 现场不兼容或无法确认时失败，并阻止后续 Step；
- CronBackup controller 只创建/观察 Operation，不调用 delivery。

### T3.4 Certificate

- 私钥和证书正文不进入 Operation/Task status 或日志；
- replacement 顺序由 Steps 明示；
- 每步定义实际证书 serial/SAN/expiry postcondition；
- partial rollout 不因 timeout 自动 retry。

### T3.5 Addon/component/CRI/registry

- 动态 Addon 统一创建 Operation；
- Addon/业务 Controller 在创建 Operation 前把 OCI tag 解析并锁定为 digest；Task 只携带 digest reference 和有界参数；
- reconcile 先 GET 稳定 Operation name，只有 NotFound 才解析 tag；已有 Operation 的 digest 不因 Catalog/tag 后续变化而重算；
- Chart、镜像、备份和其他 blob 不进入 etcd；Registry credential 不进入 Operation、Task、outputs 或日志；
- OCI executor 接入前明确复用 Node 本地凭证或实现按 Task/Agent 身份授权的运行时凭证读取，不为 Operation Engine 新增 Artifact/Secret 资源；
- digest cache 命中可离线执行；Registry/cache 均缺失时 Task 正常失败，不重新解析 tag 或回退其他内容；
- Catalog sync、tag resolve 和 cache GC 不创建 Operation、不获取 Cluster Lock；
- Helm/manifest executor 按 release/resource 实际状态 Reconcile；
- CRI/registry rollout 使用有序节点 Step；
- 删除直接 `DeliverStep` 的路径。

### T3.6 替换同步 DeliverCmd

- 证书到期观察改为 Agent typed status，不周期创建 Operation；
- ServiceAccount token 通过目标 Kubernetes API 管理，不远程执行 kubectl shell；
- apiserver certificate 只上报必要的 serial/SAN/expiry，不传原始私钥材料；
- 删除 arbitrary online command API；
- 确有修改动作时先定义 typed executor，再创建 Operation。

### T3.7 跨 Step outputs

- outputs 只接受 schema 声明的 string key；
- key/value/总量按设计文档限制；
- 只从唯一 Succeeded Task读取；
- Controller 注入下一 Task inline payload；
- input 只写 schema 声明的顶层字段，注入后再次 validation；
- output 有效期由消费 executor 负责；首版不增加通用 expiry 元数据或 Controller 预校验，过期 output 按普通 Task 失败处理，需要新 Operation 重新生成；
- join token 不复制到日志、Message、Condition、审计摘要和普通列表；
- Task read RBAC 仅给管理员。

### T3.8 日志和 CLI/Console

- 复用 `pkg/oplog` 文件和 offset 读取，identity 改为 Task UID；
- Agent 提供 mTLS `/v1/tasks/{taskUID}/logs?offset&limit`，默认端口 `10260`；
- kc-server 根据 Task nodeRef + Node management IP连接，不接受用户 URL；
- TLS dial IP，ServerName 使用 Node name/AgentID 校验每节点 serving certificate；Agent 只接受 `system:kc-server` client identity；
- 单次最大 1 MiB，固定 timeout；
- 单 Task/目录总配额和截断行为；
- 日志连接失败时按请求返回不可用并记录指标，不增加 AgentEndpointReady Condition；
- kcctl/Console 展示 Operation -> Step -> Node -> Task attempt，并继续 offset follow；
- 日志不可达和磁盘满不改变 Task phase；
- 删除 NATS OperationStepLog/DeliverLogRequest 和 Agent handler。

### Phase 3 Gate

- inventory 中所有业务动作都有 v2 builder/executor 或明确删除决定；
- 没有 handler/controller 调用 `DeliverTaskOperation`、`DeliverStep`、`DeliverCmd`；
- 所有真实 executor 有相同 Task UID 重入证据；
- 不能证明可重入、可验证、可收敛的动作没有进入 v2；
- 所有 OCI-backed executor 固定 digest，blob/credential 不进入 Operation/Task/日志；
- 日志不经过 NATS；
- Agent registration、Node status 和 Lease 不经过 NATS。

## 7. Phase 4：切换、删除 NATS 和发布硬化

### T4.1 切换 Agent/server composition

只有 Phase 3 Gate 通过后切换：

Agent composition 只包含：

```text
certificate/config loader
API client
Node/Lease reporter
Task reflector
single worker
executor registry
read-only log server
```

删除 `pkg/service/task` 和 NATS client 初始化。startup preflight 只检查证书、Node、singleton lock 和 API 连通性；日志端点失败不阻止 Task执行。

server 删除 legacy Operation goroutine/delivery wiring，只保留 API、Controller、三资源 storage、typed OperationStore 和 Agent log client。

### T4.2 全新安装和重新纳管

全新安装：

- 不部署 NATS；
- 为每个 Agent 预置独立证书；
- kc-server 到 Agent `10260` 的网络规则已配置；日志不可达不阻止执行 smoke；
- no-op/preflight Operation 通过后运行全业务 smoke。

重新纳管：

1. 停止旧 Operation 并备份必要业务数据；
2. 停止旧 Agent，确认没有遗留外部命令；
3. 部署 v2 server；
4. 预创建 Node并签发新 Agent certificate；
5. SSH 替换 Agent 配置和证书；
6. 启动 v2 Agent，确认 Node UID、Lease 和 Task selector；
7. 运行 no-op/preflight Operation；
8. 再开放 destructive Operation。

不导入旧 NATS task history，不继续旧 Running Operation。

### T4.3 删除 NATS runtime 和代码

在 inventory 清零后删除：

- `pkg/simple/client/natsio`；
- `pkg/service/delivery`；
- `pkg/service/task` 中 NATS 注册、订阅、request/reply 和 callback；
- server/Agent NATS composition；
- NATS subject/message/adapter；
- legacy Operation goroutine；
- 旧 `core/v1.Operation` API、storage、model、client、informer 和旧 schema；
- NATS deployment/systemd/container；
- 9889/9890 service、firewall、health check；
- NATS credential、TLS cert 和配置；
- installer、manifest、sample、CI、dashboard 和 docs 引用；
- `go.mod/go.sum` 中 nats.go 和仅为 NATS 引入的依赖。

若旧目录含可复用的非 NATS 逻辑，先用已有测试保护，再搬到明确的新 package。

static server 不因为“Operation v2 完成”而自动删除。若 OCI 只替换 Addon/Chart 内容分发，static server 仍可暂时承担 Agent binary、extension 和 join bootstrap；若目标是完全删除 static server，则另建安装链路任务，先验证全新安装、重新纳管、离线安装和升级全部改用 OCI，再删除 `pkg/service/staticresource`、`pkg/simple/staticserver` 及配置。该任务可以和 Phase 3 后段并行，但不能阻塞 Phase 1/2 正确性 Gate，也不能在未验证 bootstrap 前提前删除现有可用路径。

### T4.4 对抗性故障测试

自动化设计文档故障矩阵：

- server/leader 在 Lock、Task create、Task terminal、Operation status、Lock delete 前后退出；
- Agent 在 Running 前、Reconcile partial effect、postcondition 后、status PUT 前后退出；
- duplicate/out-of-order Watch、disconnect、410 和 API response loss；
- cancel/completion 和 failure/sibling cancel 竞态；Operation/Task DELETE 拒绝；
- 自动 retry、人工 retry generation、部分 attempt Create、已成功 Node跳过和旧 Running Task阻止 retry；
- 更新的同 target Operation 与旧 Operation retry 竞争；
- 同 Cluster Operation Lock 竞争；
- Agent 永久丢失且遗留 Running Task超过 termination grace period 时，Controller 将 Task/Operation 收敛为 TimedOut并释放 Lock；
- Lease 过期、日志 endpoint 不可达、日志磁盘配额耗尽；
- 大量 Agent relist、Task burst、etcd compaction 和 backpressure；
- `go test -race`、24h/72h soak；
- destructive executor 只在隔离环境执行。

### T4.5 Release Gate

发布必须同时满足：

- NATS inventory 和运行引用为零；
- 全新安装无 NATS 通过全业务 E2E；
- 重新纳管 runbook 已实际演练；
- Operation/Task/Lock API 和状态机冻结；
- 所有 executor 有 postcondition、partial-effect 和重复 Reconcile 测试；
- 不可删除约束、Step attempt、人工 `/retry`、Watch 410、auth、redaction、log quota 测试通过；
- 长期 Running/持锁、retry 次数、Lease、证书到期和日志磁盘有指标/告警；
- Operation/Task 对象数量和 etcd 占用有容量告警；
- OCI-backed Task 固定 digest，Registry/cache 故障与 credential redaction 测试通过；
- 对抗性审查中的残余风险在 release notes 和运维文档中明确。

## 8. 工作量、依赖和并行边界

以下是熟悉 KubeClipper 和 Go/Kubernetes API machinery 的工程师人日估算，包含实现、单元/集成测试和本阶段故障测试，不包含测试环境排队、产品 UI 大改和发布观察期：

| Phase | 主要工作 | 估算 |
|---|---|---:|
| 0 | inventory、API 契约、业务 fixtures | 3-5 人日 |
| 1 | 三资源 API/storage/Store、reducer、Controller、fake Agent | 10-15 人日 |
| 2 | Agent mTLS client、RBAC、Reflector、single worker、首个真实 executor | 8-12 人日 |
| 3 | 全部业务 builder/executor、outputs、日志链路 | 25-40 人日 |
| 4 | composition 切换、NATS/旧 Operation 删除、安装和故障硬化 | 7-10 人日 |
| 合计 | Operation v2 到无 NATS 发布 | 53-82 人日 |

单人串行约 11-16 工程周；两名熟悉代码的工程师在 Gate 不放松的前提下约 7-10 个日历周。最大不确定性不是 API/storage，而是 kubeadm、restore、证书和 registry 等 executor 能否清楚定义 postcondition 并通过重复 Reconcile 测试。

关键路径：

```text
API contract
  -> real storage + reducer/controller
  -> real Agent + first executor
  -> cluster create/add-node vertical path
  -> remaining business migration
  -> composition cutover
  -> delete NATS and legacy Operation
```

可以并行：

- Phase 0 后，API codegen/storage 与 executor fixtures 可并行；
- Phase 1 Gate 后，Agent PKI/client 与第一个 typed executor 可并行；
- Phase 2 Gate 后，各业务域 executor 可以由不同工程师并行，但共用冻结的 Executor/Task 契约；
- OCI Catalog/API/cache 可与 Operation Phase 1/2 并行，只有创建 Task 的执行集成必须等待 Phase 2 Gate；
- 日志链路可在 Agent mTLS 基础完成后并行，不阻塞 Operation 正确性验证。

不计入上述估算：动态 Addon 产品功能本身、OCI Registry/Catalog 完整实现、Console 大改，以及用 OCI 完全替换 Agent/extension/bootstrap static server。它们与 Operation v2 有接口依赖，但不是同一个正确性问题，应分别估算和验收。

## 9. 推荐 PR 拆分

| PR | 内容 | 合并 Gate |
|---|---|---|
| 1 | NATS inventory、业务分类、fixtures | 不改变运行语义 |
| 2 | operations/v1alpha1 Operation/Task/Lock types + generated code | schema/validation tests |
| 3 | 三资源 storage、typed OperationStore、status、selector、DELETE validation | real storage integration tests |
| 4 | minimal reducer + attempt-aware Controller + fake Agent + cancel/retry | Phase 1 Gate |
| 5 | kcctl per-node cert + Agent RBAC/client | mTLS/auth tests |
| 6 | Task reflector + single worker + Noop executor | reentry/410 tests |
| 7 | first real executor + builder + outputs | Phase 2 Gate |
| 8-N | business executors/builders by domain | per-domain failure matrix |
| N+1 | Agent log endpoint + server client + CLI/Console | logs remain non-critical |
| N+2 | new Agent/server composition + install/adopt | Phase 3 Gate |
| N+3 | delete NATS runtime/config/dependencies | zero-reference CI Gate |
| N+4 | fault/load/soak/release hardening | Release Gate |

PR 4 是架构验证点，PR 7 是真实副作用验证点。任一失败都不继续批量迁移。

## 10. 首批可执行任务

1. 生成 NATS call-site/subject inventory，并给每项标迁移类别和 owner；
2. 新增 operations/v1alpha1 Operation/OperationTask/ExecutionLock Go types；
3. 实现 validation、deterministic names 和 terminal transition table tests；
4. 增加三资源 real storage、typed OperationStore、Task status CAS 和 node name selector；
5. 实现单 Operation、单 Step、单 Node no-op Controller，并验证 Controller Store 写入与 Agent API 写入使用同一套校验；
6. 扩展到两 Step、Step 内多 Node 和 Cluster Lock；
7. 加 server crash、partial Task create、status response loss 和 Watch 410；
8. 加 graceful cancel、failure sibling cancel、Step attempt、人工 `/retry` 和 DELETE 拒绝；
9. 实现 kcctl 每节点证书、内置 Agent RBAC 和资源归属校验；
10. 实现 Agent reflector、single worker 和 Running Task reentry；
11. 选择第一个低风险真实 executor，完成 postcondition 和 partial-effect 测试。

在第 11 项完成前，不批量实现日志 UI、完整业务 executor 或证书自动轮换。

## 11. Deferred

以下内容明确不属于 Operation Engine v2 首版：

- 人工 `/retry` 总次数限制；
- 独立 Attempt API 资源和跨 Operation retry history 资源；
- DAG、quorum、best-effort completion；
- Agent TLS bootstrap、CSR、approver、自动 rotation 和 revocation service；
- server-side active Agent session；
- centralized LogStore；
- artifact API；
- automatic compensation；
- generic online command/shell executor；
- terminal Operation history GC。

只有出现经过验证的生产需求，且不能由当前模型解决时，才单独设计这些能力。

## 12. Definition of Done

- 只有 Operation、OperationTask、ExecutionLock 三个执行资源；
- 没有 fence、独立 Attempt 资源、Agent 内存整 Step retry loop、API 对象 digest、Dispatch、Result、ExecutionLease、AgentSession；OCI Artifact 仍以 digest 固定内容；
- Agent 仅通过 kc-server HTTPS API List/Watch Task并写 status；
- Controller 独占 Operation status 和 Step 顺序；
- Controller 使用注入的 typed OperationStore 写入，不通过 kc-server HTTP 自调用，不直接操作裸 etcd；
- Agent 无本地任务数据库，Running Task重启后原地 Reconcile；
- 每个 executor 的成功表示 postcondition 已确认；
- 每个 executor 使用稳定输入，并能根据实际状态重复调用后收敛；
- Step retry 创建新 Task attempt，旧 Task terminal 不重开，已成功 Step/Node不重跑；
- Operation/Task 默认不支持 DELETE，取消只修改 desiredState；Cluster 删除清理按 target UID 执行，首版无通用 finalizer/history GC；
- Agent 证书由 kcctl 每节点签发并通过 SSH 预置；
- 每个 Agent ID/Node UID/credential 唯一且禁止跨节点复用；
- Node Lease 只用于 liveness；
- outputs 只传受限小字符串；
- 日志只在 Agent 本地，通过 mTLS 按 Task UID/offset 读取；
- cancel、timeout、Step retry attempt、人工 `/retry`、不可删除约束和 Lock 有完整故障测试；
- 所有业务入口迁移或删除，NATS runtime/config/code/dependency 引用为零；
- 旧 core/v1 Operation API/storage/client/runtime 已删除；
- 全新安装、重新纳管、fault/race/load/soak Gate 通过。
