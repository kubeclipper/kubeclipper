---
comet_change: operation-engine-v2
role: technical-design
canonical_spec: local
status: implementing
base_ref: 6e17ee7
decision_date: 2026-08-05
supersedes: legacy in-process operation execution and all NATS-based agent protocols
---

# KubeClipper Operation Engine v2：基于 API 的最小可靠执行引擎

> 实现注记（2026-08-07）：当前 server/agent 已按本文使用 mTLS HTTPS、etcd-backed Operation/OperationTask/ExecutionLock 和本地 Agent 日志代理；NATS runtime、配置和 Go 依赖已移除。`core/v1.Operation` 暂时只作为业务 plan 的内部输入供转换器使用，不再注册旧 Operation HTTP storage 或 controller。

## 1. 决策摘要

Operation Engine v2 彻底移除 NATS，也不引入其他消息队列。`kc-server` 和 `kc-agent` 只通过 mTLS HTTPS API 交互，持久控制状态通过现有 API storage 保存到 etcd。

v2 只使用三个执行资源：

- `Operation`：用户提交的完整线性执行计划；
- `OperationTask`：Controller 为当前 Step 创建的单节点执行单元；
- `ExecutionLock`：保证同一个目标同时只有一个修改型 Operation。

现有 `Node` 和 `Lease` 继续表示节点身份、状态和存活性。Task 日志保存在 Agent 本地，由 kc-server 按需通过 Agent 的只读 HTTPS 端点读取。

明确不引入：

- `OperationDispatch`、`OperationResult`、`ExecutionLease`、`AgentSession`；
- fencing token、独立 Attempt 资源和 Agent 内存中的整 Step 重试循环；
- Agent 本地任务数据库或 bbolt inbox；
- 普通 Operation/Task DELETE、finalizer 和通用自动历史 GC；Cluster 删除后的受控 target-UID 清理除外；
- 自定义 CSR、bootstrap token、approver 和自动证书轮换系统；
- 为 API 对象额外计算的 plan/payload/result digest 等与 immutable、UID、resourceVersion 重复的字段；外部 OCI Artifact 仍必须使用内容 digest 固定版本；
- 任意 shell executor、DAG、补偿事务和 exactly-once 承诺。
- Operation execution dry-run、`ErrIgnore` 和 `ForceSkipError`；需要预览时由业务层提供不创建 Task 的纯 plan validation/preview。

v2 是 breaking release，只支持全新安装或节点重新纳管，不读取旧 Operation，不兼容旧 Agent，也不提供 bridge、双写或数据迁移。三个资源放在 `operations.kubeclipper.io/v1alpha1`。独立 API group 不是兼容层：它用于隔离执行引擎所有权，并让迁移期间旧业务和 v2 垂直链路可以分别构建、测试；正式发布时删除旧 `core/v1.Operation` API 和 runtime。

```text
User / Business Controller
          |
          | create complete ordered plan
          v
      Operation <---------------------------+
          |                                  |
          | level-driven reconcile           | reduce Task facts
          v                                  |
   OperationTask -- List/Watch over HTTPS --> kc-agent
          |
          +-- immutable assignment
          +-- phase-owned execution status

Target serialization:       ExecutionLock -> etcd
Node status/liveness:        Agent -> Node / Lease API
Task logs on demand:         kc-server -> Agent HTTPS -> local file
All durable control facts:   API storage -> etcd
```

Watch 只降低延迟，不承担可靠性。恢复依赖持久对象、List/relist、resourceVersion 和 Controller/Agent 的重复 reconcile。

## 2. 第一性原理

### 2.1 系统必须保存哪些事实

一次跨节点长任务只需要持久化五类事实：

1. 用户希望完成什么，以及 Step 的顺序；
2. 当前哪个 Operation 有权修改目标；
3. 每个 Step 由哪个 Node 执行什么 payload；
4. 每个 Node 在每个 Step 的各次 Task attempt 是未执行、执行中、成功、失败、超时还是取消；
5. 当前 Step 是否仍有重试额度，以及用户是否显式请求再次重试。

因此：

- 完整计划保存在 `Operation.spec`；
- 排他所有权保存在 `ExecutionLock`；
- 节点分配和执行事实保存在 `OperationTask`；
- retry attempt 直接保存为同一 Operation 下的新 `OperationTask`；
- 日志、Watch 连接、HTTP response 和 goroutine 都不是业务事实。

### 2.2 通信成功不等于目标已经达成

以下状态不能互相替代：

- HTTP 200：API 请求成功；
- Watch `ADDED`：Agent 看到了对象；
- Task `Running`：Agent 已取得执行许可；
- 进程 exit code 0：命令返回成功；
- executor 确认 postcondition：目标状态已经达成。

只有最后一项允许 Task 进入 `Succeeded`。executor 必须在 `Reconcile` 内部完成必要验证，不能把“命令启动成功”当成业务成功。

### 2.3 外部副作用无法 exactly-once

外部命令和 etcd status 不能组成原子事务。Agent 可能在副作用完成后、写回 Task terminal status 前退出。

v2 明确采用 at-least-once reconcile：

- Agent 必须先确认 Task 已在 etcd 中变为 `Running`，再执行副作用；
- Agent 重启后重新执行遗留 `Running` Task；
- 同一个 Task UID 可能调用 executor 多次；
- executor 必须检查实际状态并收敛，而不是无条件重复命令；
- 无法安全重入的动作不允许接入 v2。

可靠性来自可重复 reconcile，不来自 Agent 本地执行历史、消息 ACK、fence 或 Lease 超时。

bbolt inbox 也不能消除这个窗口：写“完成”之前崩溃仍会重复执行，先写“完成”再执行又可能永久漏执行；若要解决仍需要外部系统可观察的 postcondition。它只会新增本地数据库损坏、升级和清理边界，因此 v2 不引入 Agent 任务数据库。

### 2.4 Step 必须可重入、可验证、可收敛

v2 不要求底层命令本身天然幂等。要求每个 Step 的 executor 能够根据实际状态重复调用并收敛：

1. `Observe`：读取目标的实际状态；
2. 已达到目标：直接返回成功；
3. 明确尚未开始：使用固定输入执行动作；
4. 部分完成：从已存在的实际状态继续；
5. `Verify`：成功前验证明确的 postcondition；
6. executor 只执行前台同步命令；命令正常退出后再判断成功或失败。Agent 重启时先 Observe，目标未达到则重新 Reconcile 同一个 Running Task；不能证明可重入和收敛的动作不接入 v2。

每个 Step 在接入前必须写清楚目标状态、稳定输入、postcondition 和 partial-effect 边界。不能证明重复调用可收敛的动作不得接入 v2，也不能通过重新生成随机 token、文件名或成员 ID 来掩盖不确定状态。

例如：

- `kubeadm join` 先检查 Node 注册、kubelet 配置和证书，再决定是否继续；
- 备份使用固定 artifact reference，并分别验证生成和有效性；
- restore、证书替换等无法确认现场的动作必须由 executor 在再次执行前 Observe；无法证明可收敛的实现不得接入 v2。

### 2.5 重试是新的执行事实，不是重开旧事实

Watch 断开和 Lease 过期不改变执行结论。v2 将 Operation deadline 定义为强制执行期限：Agent 在 deadline 到达时终止前台命令；Agent 失联超过固定 termination grace period 后，Controller 可将仍为 Running 的 Task 归约为 TimedOut。这是以“所有接入 executor 均前台执行且 deadline 可终止”为前提的明确可用性取舍。一个 Task attempt 的 terminal status 永不重开；Step retry 必须为未成功 Node 创建新 Task。旧 Task 保留原 UID 和结果，新 Task 使用递增 attempt 和新 UID。

Controller 只在当前 Step 已无 Running Task 时创建 retry attempt，避免同一 Node 上两个可能有副作用的 Task 重叠。已经确认 Succeeded 的 `(stepID, nodeUID)` 永不重跑；Failed、TimedOut、Cancelled 才是 retry 候选。

## 3. Kubernetes 参照与边界

| Kubernetes | KubeClipper v2 | 决策 |
|---|---|---|
| API Server + etcd | kc-server API + existing etcd storage | 采用 |
| Job controller + backoffLimit | Operation Controller + Step retryLimit | 采用 level-driven reconcile 和有界重试 |
| replacement Pod | OperationTask attempt | 每次重试创建新 Task，不重开旧 Task |
| kubelet List/Watch Pod | Agent List/Watch Task | 采用 |
| kubelet sync loop | Agent single worker + typed `Reconcile` | 采用重复收敛原则 |
| Node Lease | Agent Lease | 只用于 liveness |
| pre-provisioned kubeconfig | kcctl 通过 SSH 预置每节点证书 | 首版采用 |
| apiserver -> kubelet logs | kc-server -> Agent Task logs | 采用按需读取 |

不机械照搬：

- Kubernetes Pod 主要由容器运行时提供可观察实际状态；KC 执行 kubeadm、备份恢复和证书更新，因此 executor 必须自己实现 postcondition；
- KC Operation 有明确的跨节点 Step 顺序，Agent 不读取 Operation，也不决定下一 Step；
- KC 修改整个集群时需要目标排他，保留一个最小 `ExecutionLock`；
- KC 已通过 SSH 安装 Agent，首版没有必要重建完整 kubelet TLS bootstrap/CSR 系统；
- Node Lease 不能证明 destructive command 已停止；
- Node name 是 Agent 的授权边界，Node UID 是 Task 的防误执行校验，不实现完整 NodeAuthorizer/NodeRestriction 子系统。

## 4. 正确性不变量

1. etcd 中的 Operation、Task 和 Lock 是唯一控制面事实源；HTTP API 和进程内 Controller 都通过同一套 typed storage 访问它们。
2. 高层业务 API 只持久化业务期望；业务 Controller 创建完整 Operation并观察结果，不直接创建 Task、写 Operation Running status或启动执行 goroutine。没有独立业务资源的直接 Operation API 由客户端创建 Operation。
3. Operation Controller 是 `Operation.status` 的唯一写入者。
4. Operation Controller 是 Task spec 的唯一创建者；Task spec 创建后不可变，不存在 Task desiredState 更新。
5. Task 从 `Running` 开始由绑定节点 Agent 独占 status；Controller 可以把从未 Running 的 Pending Task终结为 Cancelled，也可以在 deadline + termination grace period 后把仍为 Running 的 Task CAS 终结为 TimedOut。服务端按当前 phase 校验写入者和状态边。
6. 一个 Task 只绑定一个 Node name + UID、retry generation 和 attempt；同一 `(Operation UID, retryGeneration, Step ID, Node UID, attempt)` 最多一个 Task。
7. 当前 Step 全部成功之前，Controller 不创建下一 Step Task。
8. Task attempt 的 terminal phase 不可逆；Succeeded Operation 永不可重开。Failed、TimedOut、Cancelled Operation 只能通过 `/retry` 进入下一执行轮次。
9. 一个 Cluster UID 同时最多存在一个 ExecutionLock holder。
10. 同一 Cluster UID 的 Operation 按 `creationTimestamp + resourceVersion + UID` 排序；Controller 使用强一致的 typed Store 查询该 target 的 Operation，不能只依赖可能滞后的 informer cache。只有最早的未终态 Operation 可以获取 Lock、进入 Running 或创建 Task，后续 Operation 保持 Pending。前一个 Operation 进入任一 terminal phase 后，才允许下一个继续。
11. Operation Controller 不读取 Node Ready 或 Lease；Lease 过期、Watch 断开或日志失败不改变 Task/Operation 结论。
12. Agent 重启重复 Reconcile 原 Running Task，不创建新 Task。
13. Controller 只按 Step `retryLimit` 自动重试；人工 `/retry` 仍使用原 Operation，并只为当前未完成 Step 的未成功 Node 创建一个新 Task attempt。
14. 每个 Agent 使用唯一且永不复用的 `AgentID`；`Node.metadata.name = AgentID`，Agent mTLS certificate 的 CN 绑定该 AgentID，身份不得跨节点复用。Node object UID 只用于 Task reference 和对象重建校验，不是证书身份；每个进程同时只执行一个 Task，systemd 和本机 singleton lock 保证本机只有一个活跃 Agent。
15. 旧 attempt 仍有 Running Task时禁止创建新 attempt、释放 Lock或接受人工 retry。
16. 日志和进度只用于可观测性，不参与 Step 推进。
17. Task output 之外的 Message、Condition、日志和审计摘要不得携带 token、kubeconfig 或私钥明文。
18. Operation name 是创建请求的幂等键：调用方在首次请求前确定并在通信重试中复用；API 不接受 `generateName`，AlreadyExists 时调用方 GET 并确认 immutable plan 是否相同。
19. informer cache 只用于调度和普通归约；创建新 attempt、写 Operation terminal、释放 Lock和清理历史前，Controller 必须通过 Store 强一致列出该 Operation 的 Task并重新验证安全条件。

## 5. API 模型

以下结构表达契约，字段名可根据仓库代码风格微调。

### 5.1 Operation

```go
type OperationSpec struct {
    TargetRef       ObjectReference       `json:"targetRef"` // kind/name/uid
    Action          string                `json:"action"`
    DesiredState    OperationDesiredState `json:"desiredState"` // Active, Cancelled
    RetryGeneration int64                 `json:"retryGeneration,omitempty"`
    Timeout         metav1.Duration       `json:"timeout"`
    Steps           []OperationStep       `json:"steps"`
}

type OperationStep struct {
    ID         string               `json:"id"`
    Targets    []NodeReference      `json:"targets"` // name + uid
    Executor   string               `json:"executor"`
    Payload    runtime.RawExtension `json:"payload"`
    Inputs     []StepInput          `json:"inputs,omitempty"`
    RetryLimit int32                `json:"retryLimit,omitempty"` // additional automatic attempts per node
}

type StepInput struct {
    Field       string    `json:"field"`
    FromStepID  string    `json:"fromStepID"`
    FromNodeUID types.UID `json:"fromNodeUID"`
    OutputKey   string    `json:"outputKey"`
}

type OperationStatus struct {
    Phase                   OperationPhase  `json:"phase"`
    ObservedRetryGeneration int64          `json:"observedRetryGeneration,omitempty"`
    Reason                  OperationReason `json:"reason,omitempty"`
    Message                 string          `json:"message,omitempty"`
    Deadline                *metav1.Time    `json:"deadline,omitempty"`
    StartedAt               *metav1.Time    `json:"startedAt,omitempty"`
    FinishedAt              *metav1.Time    `json:"finishedAt,omitempty"`
}
```

`OperationReason` 首版只保留 `StepFailed`、`DeadlineExceeded`、`CancelledByRequest` 和 `InvalidExecutionFacts`；Succeeded/Pending/Running 的 reason 为空。`message` 最大 4 KiB，只概括当前 generation 的结果，节点级事实和详细错误从 Task 查询。

规则：

- `targetRef.uid` 必填；所有修改集群的 Operation 都以 Cluster UID 为 target，不设计多锁或 Node 子锁；
- `metadata.name` 必填且由调用方稳定生成，禁止 `generateName`。业务 Controller 使用 owner UID + generation + action；交互式客户端生成一次 request UUID 并在 HTTP retry 中复用；
- Create 时服务端强制 `desiredState=Active`、`retryGeneration=0`、`status.phase=Pending` 并清空客户端提供的 status；之后不提供普通 spec Update，只有 cancel/retry 子资源可修改控制字段；
- Operation Create 时服务端 GET target 并校验 kind/name/UID，客户端不能用伪造 UID 创建另一把 Lock；
- `spec.targetRef.uid` 是服务端可查询字段，供强一致 `ListByTargetUID` 做顺序判断；不依赖客户端可篡改 label；
- Steps 顺序就是执行顺序，首版不支持 DAG；
- Steps 和每个 Step 的 targets 都不能为空；Step ID 在 Operation 内唯一，同一 Step 内 Node UID 不重复；目标 Node name + UID 创建时固定；
- Operation Create 时校验目标 Node name + UID 存在；后续 Node 是否在线不影响 Task 创建；
- executor 和 payload 使用版本化 typed schema；不提供表达式或模板语言；
- `inputs` 只能引用 plan 中位置更早的 Step，并只允许把其成功 Task 的一个字符串 output 写入 typed schema 声明的顶层字段；注入后重新执行 schema validation；
- 创建后 plan 字段不可变；只有取消接口可以把当前执行轮次的 `desiredState: Active -> Cancelled`，`/retry` 可以在满足前置条件时将其恢复为 Active 并递增 `retryGeneration`；
- `retryLimit` 表示每个 `(stepID, nodeUID)` 首次执行之外允许的自动 attempt 数；必须设置小的硬上限；
- API validation 对 Operation 总大小、Step 数和单 Step target 数设置硬上限，避免单对象或 Task burst 压垮 etcd；
- status 不保存 `currentStepID` 或通用 `Conditions`；当前 Step、完成数量和节点明细始终从 Task 推导。`reason` 使用有限枚举，`message` 只保存有界摘要，不参与状态判断，也不得包含 outputs 或凭证。

首版使用编译期常量，不为每个限制增加配置项。Phase 0 必须用所有现有业务 fixture 验证这些值足够：

| 限制 | 首版值 |
|---|---:|
| 默认 Operation timeout | 90 min |
| Operation timeout 范围 | 1 min - 24 h |
| Agent TERM -> KILL grace | 30 s |
| Server termination grace | 2 min |
| Operation 最大序列化大小 | 512 KiB |
| Step 数量 | 256 |
| 单 Step target 数量 | 1000 |
| 单 Step payload | 128 KiB |
| Task 最大序列化大小 | 256 KiB |
| retryLimit | 0-3 |
| Operation/Task message | 4 KiB |

若真实 fixture 超限，应先缩小 payload 或把 blob 改为固定 artifact reference；只有仍无法表达合法业务时才 review 常量，不增加 Payload 资源。

Operation phase：

```text
Pending -> Running -> Succeeded
                   -> Failed
                   -> TimedOut
                   -> Cancelled
Pending ----------> Cancelled

Failed/TimedOut/Cancelled -- explicit /retry --> Pending
```

当多个事实同时出现时，Controller 使用固定归约优先级，避免 cancel、deadline 和最后一个 Task 完成的竞态产生不同结果：

1. 所有计划中的 `(stepID, nodeUID)` 都已有 Succeeded Task时，Operation 为 Succeeded；
2. 仍有 Running Task时保持 Running；cancel 或 deadline 只阻止新 Task并取消 Pending Task；
3. 已无 active Task且 `desiredState=Cancelled` 时为 Cancelled；
4. 已无 active Task且 deadline 已到时为 TimedOut；
5. 已无 active Task、当前 Step 未完成且没有可用自动 retry 时为 Failed；
6. 其余情况保持 Pending/Running并继续物化当前 Step。

因此，cancel 是“停止后续工作”，不是覆盖已经确认的成功；deadline 也不能把已经 CAS 成功写入的 terminal Task 改写为另一结果。并发 terminal 写入由 resourceVersion CAS 决胜，输的一方 GET 最新对象并接受已持久化终态。

Operation 创建 API 不因已有 Active Operation 而拒绝请求；创建后即可接受并保持 Pending，但同一 Cluster UID 必须按 `creationTimestamp + resourceVersion + UID` 严格排队。只有最早的未终态 Operation 可以获取 Lock；后续 Operation 不创建 Task、不执行，只等待前序 Operation 进入 Succeeded、Failed、TimedOut 或 Cancelled。获取 Lock 后进入 Running。不引入 `Cancelling`：取消过程由 `spec.desiredState=Cancelled` 和当前非终态 Task 表达。Succeeded 永久不可逆；Failed、TimedOut、Cancelled 是当前执行轮次的终态，只能由带 resourceVersion precondition 的 `/retry` 开启下一轮。

`POST /operations/{name}/cancel` 只接受当前为 Pending/Running 且 `desiredState=Active` 的 Operation。请求携带 Operation UID 和 resourceVersion，handler 只用 CAS 更新 `desiredState=Cancelled`，不写 status、不修改 Task、不释放 Lock。response 丢失时客户端 GET Operation；已为 Cancelled 表示请求成功，不应重复构造旧对象更新。terminal Operation 拒绝 cancel。

`POST /operations/{name}/retry` 只接受 Failed、TimedOut 或 Cancelled Operation，并要求该 Operation 已无 Pending/Running Task、仍是该 target 最新创建的 Operation。请求必须携带 Operation UID 和 resourceVersion；handler 使用一致性 Store 读取 Operation 和关联 Task，校验身份、phase、active Task 和 latest-target 约束后，只通过 CAS 把 desiredState 恢复为 Active、把 retryGeneration 加一。API 不修改 status、不获取 Lock、也不创建 Task。重复或并发请求只有一个 CAS 成功。

Controller 通过 informer 观察到 `spec.retryGeneration > status.observedRetryGeneration` 后才处理 retry：先查询该 target 的最新 Operation；如果当前 Operation 已不是最新，则不获取 Lock、不创建 Task，只推进 `observedRetryGeneration`，发出 `RetrySkippedNotLatest` 审计事件和结构化日志，原 terminal phase/reason/message 保持不变。仍为最新时获取 Lock 后再次校验；若已被更新 Operation 取代则释放刚获得的 Lock并按 skipped 处理，否则使用 server time 持久化 Pending/Running、startedAt 和新 deadline，最后只为第一个未完成 Step 的未成功 Node创建新 Task。该请求不清空历史，也不重置旧 Task。

### 5.2 OperationTask

```go
type OperationTaskSpec struct {
    OperationRef   ObjectReference      `json:"operationRef"` // name + uid
    StepID         string               `json:"stepID"`
    NodeRef        NodeReference        `json:"nodeRef"` // name + uid
    RetryGeneration int64               `json:"retryGeneration"`
    Attempt        int32                `json:"attempt"`
    Executor       string               `json:"executor"`
    Payload        runtime.RawExtension `json:"payload"`
    Deadline       metav1.Time          `json:"deadline"`
}

type OperationTaskStatus struct {
    Phase      TaskPhase    `json:"phase"`
    Result     *TaskResult  `json:"result,omitempty"`
    StartedAt  *metav1.Time `json:"startedAt,omitempty"`
    FinishedAt *metav1.Time `json:"finishedAt,omitempty"`
}

type TaskResult struct {
    Reason   TaskResultReason  `json:"reason,omitempty"`
    Message  string            `json:"message,omitempty"`
    Outputs  map[string]string `json:"outputs,omitempty"`
}
```

Task name 由 `operation UID + retry generation + step ID + node UID + attempt` 的确定性 hash 生成。Controller 重启后重复 Create 收到 AlreadyExists 时 GET 并比较 immutable spec；不一致表示实现不变量被破坏。

Task phase：

```text
Pending -> Running -> Succeeded
                   -> Failed
                   -> TimedOut
Pending ----------> Cancelled
```

TaskResult reason 只用于诊断，不参与 Controller 的安全判断。使用有限枚举，至少包括：

- `ExecutionFailed`；
- `DeadlineExceeded`（只用于 Task deadline 到达后终止的 Running Task）；
- `OperationCancelled`；
- `SiblingFailed`；
- `OperationDeadlineExceededBeforeStart`。

规则：

- `OperationStore` 使用 resourceVersion CAS 防止并发覆盖；HTTP handler 另行校验 Agent 身份和对象关系；
- Task Create 由服务端强制初始化为 Pending并清空外来 status；spec 创建后不可变；
- `spec.operationRef.uid` 和 `spec.nodeRef.name` 是 selectable fields，分别供 Controller 强一致确认和 Agent List/Watch；
- Agent 可以写 `Pending -> Running` 和 `Running -> terminal`；
- Controller 只能在确认 Task 从未 Running 时写 `Pending -> Cancelled`，或在 deadline + termination grace period 后将仍为 Running 的 Task CAS 为 `TimedOut`；
- server transition time 是 `StartedAt/FinishedAt` 的权威来源，不信任 Agent 自报时间；
- stale status update 返回 Conflict；Agent 必须 GET 最新 Task，已是预期 terminal 则确认完成，否则根据最新 phase、deadline 和实际状态重新归约；
- Operation cancel 不修改 Running Task，也不向 Agent 发送终止命令；deadline 到达时 Agent 终止当前命令并确认进程退出后提交 TimedOut；Agent 失联超过 termination grace period 时 Controller 可 CAS 将 Running Task 收敛为 TimedOut；
- Succeeded 的 reason 为空；Failed、TimedOut 和 Cancelled 必须使用与 phase 匹配的有限 reason；
- Task TimedOut 只表示 Operation/Task deadline 强制结束；executor 自己更短的技术超时按命令失败写 Failed，不生成第二种持久 timeout 语义；
- 一个 `(stepID, nodeUID)` 的任一历史 attempt 已 Succeeded 后，Controller 永不为它创建新 attempt；
- 最新 attempt 为 Failed、TimedOut 或 Cancelled，且当前 Step 已无 Running Task时，Controller 才能按 retryLimit 或已观察到的人工 retry generation创建下一 attempt；
- `retryLimit` 只计算已经进入 Running 的 execution attempt；Controller 在 sibling 失败后取消的 Pending Task不消耗 retry budget，但下一 Task仍使用递增 attempt 保持名称唯一；
- outputs 只允许出现在 `Succeeded` Task；失败诊断放在受限长度的 Reason/Message；
- payload 是 Controller 解析 inputs 后写入 Task spec 的最终不可变快照，不再引入 Payload 资源或可变引用；
- Task serialized payload 必须有硬上限，stdout、大文件、备份和证书 artifact 不进入 Task。

### 5.3 ExecutionLock

```go
type ExecutionLockSpec struct {
    TargetRef ObjectReference `json:"targetRef"`
    HolderRef ObjectReference `json:"holderRef"` // Operation name + uid
}
```

Lock name 由 target kind + UID 计算。Controller 通过 etcd Create 原子获取 Lock：

- Create 成功：当前 Operation 获得目标所有权；
- AlreadyExists 且 holder 是自己：继续 reconcile；
- AlreadyExists 且 holder 是其他 Operation：保持 Pending 并 `RequeueAfter`；
- Operation terminal 且该 Operation 已无 Pending/Running Task 后，通过 `OperationStore.ReleaseLock(targetUID, lockUID, holderOperationUID)` 删除 Lock；读取 Lock 后必须确认 holder 是当前 Operation，并使用该 Lock 的 UID precondition 删除；Delete response 丢失时重复执行同一带 UID 的释放逻辑。

API validation 要求 Lock name 与 target kind/UID 一致，holder Operation 存在且 targetRef 完全相同；只有 Operation Controller identity 可以 Create/Delete Lock，普通用户和 Agent 无权直接操作。Store 不提供按 name 无条件删除 Lock 的方法。

Lock 不是 Lease，没有 renew、timeout、status、effectsMayExist 或 fence。Controller 只有在该 Operation 不再存在非终态 Task 时才释放 Lock，避免 retry 或下一个 Operation 与旧 Running Task 重叠。

不能只把 holder 写在某个 Operation status：两个 Operation 是两个 etcd key，无法用一次对象 CAS 原子声明“整个 Cluster 当前没人执行”。也不把 holder 写进 Cluster status，因为 Cluster status 已有业务 Controller writer，会把执行互斥和领域状态耦合。canonical target-UID Lock 把这件事缩成一次 Create/Delete。Kubernetes Job 通常不需要这种目标锁，是因为一个 Job只管理自己的 Pod；KC 的多个独立 Operation 会修改同一个外部 Cluster。若删除 ExecutionLock，就必须把“全局只有一个正确 leader、同 target 永不并发 reconcile”提升为安全前提，故障面反而更隐蔽。

正常结束必须按固定顺序持久化：先写 Operation terminal status，再通过 holder 校验和 UID precondition 删除 Lock。任何一步 response 丢失都通过 GET 后重复 reconcile；如果 Lock 已被其他 Operation 重新获取，旧 Operation 不得删除它。Operation 本身不会被删除，因此不会出现 terminal status 尚未持久化、对象或 Task 已被回收的窗口。

### 5.4 字段所有权

| 对象/字段 | 写入者 | 允许行为 |
|---|---|---|
| Operation plan spec | 用户/业务 controller | Create 后不可变 |
| Operation control spec | cancel/retry API | Active -> Cancelled；合法 retry 时恢复 Active 并递增 retryGeneration |
| Operation status | Operation Controller | 按 Task/Lock 事实归约 |
| Task spec | Operation Controller | 按 step/node/attempt Create，之后不可变 |
| Pending Task status | Agent 或 Operation Controller | Agent 可进入 Running；Controller 可安全终结未启动 Task |
| Running Task status | 绑定 Agent；Controller 在 deadline + grace period 后 | Agent: Running -> Succeeded/Failed/TimedOut；Controller: Running -> TimedOut |
| ExecutionLock | Operation Controller | Create 获取；Operation terminal 且无 active Task 后 Delete |
| Node status / Lease | 对应 Agent | 受 node name 限制更新；Lease 可按固定名称 Create-or-Update |

### 5.4.1 Controller 写入路径

Controller 与 kc-server API 在同一个进程内运行，不通过 HTTP 自调用。server 启动时从 `storageFactory` 构造一个 v2 `OperationStore`，只把这个 typed interface 注入 Controller：

```text
Agent / 用户         -> kc-server HTTP handler -> OperationStore -> etcd
Operation Controller -> informer/lister 读取   -> OperationStore -> etcd
```

`OperationStore` 是三类资源的唯一进程内读写门面，至少提供 Operation/Task 的 spec/status 更新、按 target UID 的强一致 Operation 查询、按 Operation UID 的强一致 Task 查询、Task 创建、Lock 获取和带 UID precondition 的 Lock 释放。Controller 不在 reconcile 中访问 `storageFactory`，不直接操作 `rest.StandardStorage`，也不直接使用 etcd client。

HTTP handler 和 Controller 调用同一个 Store 方法及同一套 transition validator。HTTP 请求需要认证和授权；Controller 使用受信任的内部 actor 参数，不经过 IAM 或 loopback HTTP。resourceVersion CAS、Task terminal immutable、Succeeded Operation immutable、spec/status 所有权和 Lock holder 校验在 Store 层始终生效。

### 5.4.2 最小更新接口

etcd 的 resourceVersion CAS 已经负责并发写冲突，不再为 status update 增加特殊幂等层。Store 只暴露按字段所有权划分的 typed 方法：

```go
UpdateOperationStatus(ctx, operationName, operationUID, resourceVersion, status)
UpdateOperationControl(ctx, operationName, operationUID, resourceVersion, desiredState, retryGeneration)
UpdateTaskStatus(ctx, agentID, taskName, taskUID, resourceVersion, status)
```

Operation status 和 Task 创建只由进程内 Controller 调用 Store，不提供对应的外部写 API。Agent 只使用 `PUT /operationtasks/{name}/status`，请求包含它最近一次 GET 到的 Task UID、resourceVersion 和 status；服务端原子读取当前 Task、同时校验 name + UID + resourceVersion，保留 spec/metadata、校验 Agent 与 nodeRef 的关系及合法状态边，再写入 status。外部不提供 OperationTask 的普通 Create/Update/Delete。

Controller 的 Pending -> Cancelled 与 Agent 的 Pending -> Running 并发时，旧 resourceVersion 的一方收到 Conflict；调用方重新 GET，并按最新 phase 重新归约。不会静默覆盖，也不允许用旧完整对象重放。

### 5.4.3 最小 HTTP API

沿用仓库现有 `/api/<group>/<version>` 风格。首版只暴露完成闭环所需接口：

| 调用者 | API | 说明 |
|---|---|---|
| 用户/业务 Controller | `POST /api/operations.kubeclipper.io/v1alpha1/operations` | 创建完整 immutable plan |
| 用户/业务 Controller | `GET /api/operations.kubeclipper.io/v1alpha1/operations[/{name}]` | List/Watch/Get Operation |
| 用户 | `POST /api/operations.kubeclipper.io/v1alpha1/operations/{name}/cancel` | 携带 Operation UID + resourceVersion，CAS 设置 `desiredState=Cancelled` |
| 用户 | `POST /api/operations.kubeclipper.io/v1alpha1/operations/{name}/retry` | 携带 Operation UID + resourceVersion，CAS 递增 `retryGeneration` |
| Agent/管理员 | `GET /api/operations.kubeclipper.io/v1alpha1/operationtasks[/{name}]` | Agent 只可 List/Watch/Get 自己的 Task |
| Agent | `PUT /api/operations.kubeclipper.io/v1alpha1/operationtasks/{name}/status` | 提交 Task UID + resourceVersion + status |
| 管理员 | `GET /api/operations.kubeclipper.io/v1alpha1/operationtasks/{name}/logs` | kc-server 授权后代理 Agent 本地日志 |

`ExecutionLock` 不提供面向普通用户或 Agent 的 CRUD route；只由注入 Controller 的 `OperationStore` 操作。Operation/Task 普通 DELETE route 不注册。Cluster 删除清理使用仅供受信任 lifecycle controller 调用的进程内 Store 方法，不开放“带特殊参数的 DELETE”后门。Lock release 必须同时校验 lock UID 和 holder Operation UID。

### 5.5 为什么是三个资源

| 资源 | 是否保留 | 第一性原理 |
|---|---|---|
| Operation | 保留 | 保存用户意图和完整顺序计划 |
| OperationTask | 保留 | 隔离每个 Agent 的写入，提供单节点 List/Watch 对象 |
| ExecutionLock | 保留 | 通过一个原子对象串行化同一 Cluster 的修改 |
| Dispatch | 删除 | Task spec 已经是持久分配 |
| Result | 删除 | Task status 原子保存 terminal 和小结果 |
| ExecutionLease | 删除 | 时间不能证明外部进程停止 |
| AgentSession | 删除 | 每个 Agent 身份唯一且不复用，Node identity + Task 对象已足够 |

## 6. Operation Controller

### 6.1 业务计划创建边界

Cluster、Backup、AddonInstallation 等已有业务资源继续保存领域期望。其 HTTP handler 只校验并持久化该资源，不同步投递命令，也不在“更新业务 status + 创建 Operation”之间假设原子性。对应业务 Controller：

1. 从业务对象的 UID、generation/action 计算稳定 Operation name；
2. 先 GET 该 name；存在则校验 owner generation/action/target 后直接接管，不重新解析 tag、token等 mutable input；
3. 只有 NotFound 时才读取当前实际状态和外部引用，冻结 digest/稳定输入并构造完整有序 plan；
4. Create Operation；AlreadyExists 表示并发/response-loss，GET 并比较本次已构造的 target/plan；不同则报告不变量错误；
5. 将 Operation reference 写回业务 status；
6. 观察 terminal Operation并归约业务 status。

如果 Controller 在 Create 成功后、reference 回写前退出，下一次 reconcile 会根据相同 owner UID/generation/action 算出同一 Operation name并先 GET，不会创建第二个计划，也不会因 OCI tag 等外部值已变化而重写输入。业务 status 只是便捷索引；Operation 与业务 status 不需要跨对象事务，也不为此增加 Owner/Request 资源。

Operation 创建时 Steps、targets、executor 和 payload 已全部固定。Operation Controller 不理解 Cluster/Addon 业务，也不在执行过程中重新生成 plan；它只按 Step 屏障创建 Task。这保留当前“创建 Operation 时计划已确定”的模式，只把投递和结果回调改为持久化 reconcile。

### 6.2 执行 Controller

Controller 使用 informer + rate-limited workqueue，队列只保存 Operation key。每次 reconcile 都从 lister 重新读取普通对象，并通过注入的 `OperationStore` 写入；严格顺序判断额外使用 Store 的强一致 target 查询。不等待 Agent response，不启动业务 goroutine，也不通过 kc-server HTTP API 自调用。

cache 可以让 Controller 保守地晚推进，但不能用于不可逆结论。准备创建新 attempt、写 Succeeded/Failed/TimedOut/Cancelled、释放 Lock 或执行 Cluster 历史清理前，Controller 使用 `OperationStore.ListTasksByOperationUID` 强一致读取并重新运行相关 reducer 条件。若事实变化则放弃本次动作并重新排队。这样避免 Task Create/status 已写入 etcd但尚未进入 informer cache 时，Controller错误终结 Operation或让两个 attempt 重叠。

Store/List/Watch/CAS timeout、Conflict、leader loss 和 response loss 都是控制面通信错误：Controller 返回 error/requeue并保持 Operation/Task/Lock 事实不变，不能据此写 Failed、TimedOut或释放 Lock。Operation terminal 只能来自强一致确认后的 Task facts、用户 desiredState、持久 deadline，或确定且不可恢复的 plan/Task 不变量错误。

同一 Cluster 的顺序由持久化 Operation 对象决定，不依赖 workqueue 的处理顺序。Controller 通过强一致的 typed Store 查询该 `targetRef.uid` 的所有 Operation，按 `creationTimestamp + resourceVersion + UID` 排序；不能只依赖可能滞后的 informer cache。只有最早的未终态 Operation 可以获取 ExecutionLock、进入 Running 和物化 Task。后续 Operation 保持 Pending，并在前序 Operation 进入 terminal phase 后重新入队。Operation informer handler 必须按 target UID 将该 Cluster 的 Pending Operation 入队，不能只重新入队发生变化的对象。ExecutionLock 负责并发互斥，排序规则负责先来先执行；两者都不能被 Node Ready 或 Lease 替代。

等待 Lock 或 deadline 的分支必须返回有界 `RequeueAfter`。内存 timer 丢失不会改变事实：Controller 重启后 initial List 会重新入队，并从持久化的 `status.deadline` 重新计算。首次执行和每次人工 retry 都由 Controller 使用 server time 设置新的 deadline；Step 自动 retry 共享当前执行轮次的 deadline。deadline 到达且仍有 Running Task时，Controller 必须再以 `terminationGracePeriod` 为界 requeue，以执行最终 TimedOut 收敛。

```text
reconcile(Operation)
  -> process cancel / retry generation first
  -> if terminal and no new retry: ensure no active Task, release Lock, return
  -> list same-target Operations and select earliest non-terminal Operation
  -> if another earlier Operation exists, keep Pending and requeue after predecessor terminal
  -> acquire ExecutionLock; on contention requeue
  -> check Operation deadline
  -> at deadline: cancel Pending Tasks, stop creating Tasks, requeue grace expiry
  -> at deadline + grace: CAS remaining Running Tasks to TimedOut
  -> derive current Step from all persisted Task attempts
  -> materialize missing current-Step Tasks for unfinished nodes
  -> if any Task fails: cancel Pending siblings, let Running siblings finish
  -> when current Step has no Running Task:
       if every unfinished node is eligible for the next attempt: create Tasks
       else reduce Operation to Failed/TimedOut/Cancelled
  -> if every target in current Step has a Succeeded attempt: advance
  -> if final Step succeeded: Operation Succeeded, release Lock
```

Step 内所有目标 Task 可以并行。首版只有 `AllSucceeded`：

- Controller 进入当前 Step 后立即为尚无 Succeeded attempt 的目标创建确定性 Task；部分 Create 后退出时，新 leader补齐缺失 Task；
- 同一 Cluster 的后续 Operation 即使已经创建，也只能保持 Pending；前序 Operation 进入任一 terminal phase 后，Controller 才重新归约并允许它获取 Lock；
- Node/Agent 离线时 Task 保持 Pending，Controller 不以 Ready 或 Lease 作为创建门槛；
- 任何 Task 明确失败后，不再创建后续 Step；
- Controller 通过 CAS 把同 Step 尚为 Pending 的 sibling Task写为 Cancelled；已经 Running 的 sibling 不强制停止，Controller 等待其自然结束；
- 当前 Step 已无 Running Task后，Controller 按 `(stepID,nodeUID)` 聚合 attempt：任一历史 attempt Succeeded 即完成；否则只为未成功 Node 创建下一 attempt；
- 自动 retry 只有在当前 Step 的每个未成功 Node 都允许下一次 attempt 时才启动；任一必需 Node 已耗尽额度，整个 Step 直接 Failed，不能只重跑其余 Node；人工 `/retry` 为所有未成功 Node 各授权一次新 attempt；
- 对每个 `(stepID,nodeUID)`，第一次出现 Task 的 retryGeneration 是它的 base generation；该 generation 最多执行 `1 + retryLimit` 个进入 Running 的 Task；被 Controller 从 Pending 取消的 Task不消耗次数；
- 之后每个更高的人工 retry generation 对该未成功 Node最多允许一个 Task，不再附带新的自动 retryLimit。若某个后续 Step 在较高 generation 才第一次到达，该 generation仍是该 Step/Node 的 base generation，正常享有 `1 + retryLimit`；
- `desiredState=Cancelled` 或当前 generation deadline 已到达时，Controller 不创建任何自动 retry attempt；
- Task attempt name 包含 attempt，旧 terminal Task 不修改；
- 自动重试额度耗尽后，Operation 进入 Failed 或 TimedOut。用户可在确认无 active Task 后对原 Operation 调用 `/retry`；
- 不支持 ErrIgnore、ForceSkipError、quorum 或 best-effort completion。
- v2 Task 一旦进入 Running 就代表真实执行许可，不提供 dry-run Task；preview 不能创建 Operation/Task或占用 Lock。

当前实现中的 `Step.RetryTimes` 语义保留为 `retryLimit`，但实现从 Agent 内存 `for` 循环改为 Controller 创建持久 Task attempt。它类似 Kubernetes Job controller 创建 replacement Pod：Operation/Step 身份保持不变，执行尝试是不同对象。

## 7. Agent 执行模型

### 7.1 List/Watch

Agent 使用生成的 OperationTask clientset 和 client-go SharedInformer/Reflector，按证书中的 Node name 观察 Task：

1. List `spec.nodeRef.name=<self>` 的全部 Task；server 根据 Agent identity 强制该 selector，不能由请求方放宽；
2. 记录 collection resourceVersion；
3. 校验每个 Task 的 Node UID 等于本地注册 Node UID；
4. 先处理遗留 `Running` Task，没有 Running 时按 `creationTimestamp + resourceVersion + UID` 选择 Pending Task；
5. 从 List resourceVersion 开始 Watch；
6. EOF/网络错误由 Reflector 退避重连；收到 `410 Gone` 时由 Reflector relist 并 Replace cache；
7. 重复/乱序事件只更新内存对象，不并发执行同一 Task。

kc-server 只需提供 `spec.nodeRef.name` 可索引 selector，不增加 active phase selector。terminal Task仍保留在本 Node 的 informer cache 中但不入执行队列；未删除 Cluster 的历史容量由 etcd 容量监控和 Cluster 删除清理解决，不把清理职责编码进 Watch selector。List 和 Watch 创建错误必须保留 storage 的 APIStatus；过旧 resourceVersion 返回 `410 Gone`，不能被包装为 500。Node UID 是 Agent 防止同名 Node 重建后误执行旧 Task 的本地校验，不作为首版 List 权限边界。

Agent Task API 必须是 client-go 可直接消费的 Kubernetes 风格协议，而不是 Console 分页 API：普通 List 返回 `OperationTaskList` 和 `ListMeta.resourceVersion`，Watch 返回标准 `watch.Event` 流；支持 `fieldSelector`、`resourceVersion`、`watch=true` 和 `timeoutSeconds`。initial List 使用空 resourceVersion 获取当前集合；Watch 从返回的 collection RV 开始。storage 返回的 `metav1.Status` 和 HTTP code 原样保留，尤其是 etcd compaction 对应的 410；首版不要求 pagination、continue token或 watch bookmark。

### 7.2 单 worker

每个 Agent 只有一个 Task worker，并使用本机 OS singleton lock 防止同一主机启动两个 Agent 进程。安装和重新纳管保证每个 Agent 使用唯一 Node name + UID 和独立证书，身份不得复制到其他节点；因此不设计跨主机 session 或 active-installation 仲裁。

选择规则：

- terminal Task 忽略；
- 一个 Running Task：优先重新 Reconcile；
- 多个 Running Task：不执行任何一个，报告 Node Condition 并等待人工处理；
- 没有 Running Task：选择最早的 Pending Task；
- 当前 Task terminal status 得到 server 确认后才选择下一个；
- API 断连期间可以继续当前 Running Task，但不得从 cache 启动下一 Task。

informer cache 只用于唤醒和排序。worker 每次准备启动 Pending Task，或在 Agent 进程重启后恢复 Running Task时，必须先对该 Task 做一次实时 GET；如果已 terminal、Node UID 改变或对象不存在，则不执行。已经在本进程内运行的 Task遇到 API 断连可以继续到本地 deadline，不需要为每次执行循环 GET API。

### 7.3 执行协议

```text
observe Task
  -> live GET before a new/recovered execution
  -> validate Node UID / executor schema / attempt / deadline
  -> CAS Pending -> Running and wait for server confirmation
  -> executor.Reconcile(actual state -> desired state)
  -> PUT /operationtasks/{name}/status with resourceVersion
  -> GET/Watch confirm terminal status
  -> select next Task
```

Agent 启动时看到 Running Task，直接再次调用同一个 executor Reconcile。未确认 Running 写入 etcd 前绝不开始副作用。

如果遗留 Running Task 的 deadline 已过，Agent 不创建新 Task；若该 Task 仍为 Running，先终止本地前台命令并确认退出，再尝试写 TimedOut。若 Controller 已在 termination grace period 后写入终态，Agent 接受该终态且不再执行。

如果 terminal status response 丢失，Agent GET Task：

- 已是自己提交的 terminal 内容：完成；
- 仍是 Running：再次检查实际状态并提交；
- 出现不同 terminal 内容：停止并报告不变量错误。

如果重试 status update 收到 Conflict，同样先 GET 最新 Task；不得把旧完整对象换成新 resourceVersion 后直接重放。

### 7.4 Executor 契约

```go
type Executor interface {
    Reconcile(ctx context.Context, task TaskSpec, log io.Writer) (TaskResult, error)
}
```

每个 executor 使用版本化名字和 typed payload schema。对于相同 Task UID 的重复 Reconcile：

- 不要求底层命令字面上幂等，但要求 executor 通过实际状态检查实现可重入和收敛；
- 输入必须来自不可变 Task spec，不在每次 Reconcile 中重新生成随机 token、文件名或成员 ID；
- 已达到目标状态时直接成功；
- 部分完成时从实际状态继续；
- 重复调用不能创建重复成员、重复备份或破坏已正确配置；
- 返回成功前必须验证 postcondition；
- executor 只产生三类执行结论：命令正常退出且 postcondition 成功为 Succeeded，命令正常退出但 exit code 非 0 或 postcondition 未达到为 Failed，命令被 deadline 强制终止且已确认退出为 TimedOut；
- 判断依据来自真实系统状态，不依赖 Agent 本地任务历史。

如果 Agent 暂时不能判断受控前台进程或目标现场（API/observe 暂时失败、进程状态查询尚未返回），不得把“不确定”写成 Failed，也不得创建新 attempt；Task 保持 Running，runner 对安全的 Observe 做有界重试，直到得到上述三类结论或统一 deadline 终止路径收敛为 TimedOut。只有正常退出后的非零结果或已验证 postcondition 未达到才是 Failed。

Validate、执行和 Verify 由 executor 内部实现，不拆成多套公共状态机。Operation cancel 不取消 Running Task 的 context；deadline 到达时 executor 必须终止前台命令并等待退出，超时可能留下部分副作用，后续 retry 必须重新 Observe。

所有外部命令必须由 runner 以前台子进程组启动。deadline 到达时先发送 TERM，经过固定且有界的本地 kill grace 后发送 KILL，并 `Wait` 到受控进程组退出，再提交 TimedOut。executor 不得使用 `nohup`、shell `&`、脱离进程组的 daemonize 或把实际修改动作委托给不可跟踪的后台进程。systemd unit 应配置 Agent 退出时清理其受控子进程。这个约束覆盖正常崩溃和超时路径，不为人为复制身份、手工启动后台命令等非法部署增加新状态。

runner 只识别三类执行结论：

- 成功：postcondition 已确认，Task Succeeded；
- 失败：postcondition 未达到，Task Failed；
- 超时：命令被 deadline 强制终止且已确认退出，Task TimedOut。

API timeout、status Conflict 和 Watch 断开只做通信重试，不形成业务 attempt。executor 可以在一个 Running Task 内对明确安全的短暂观察错误做有界退避，但不得用通用 `Retryable` 结果无限重复外部命令。新的业务执行 attempt 只由 Controller 创建新 Task 表达。

v2 不提供通用 shell executor。现有 CommandShell/CommandCustom 必须迁移为类型化 executor，或者删除。

## 8. 取消、超时和重试

### 8.1 Cancel

Operation `desiredState` 变为 `Cancelled` 后：

1. Controller 不再创建新 Task；
2. Controller 通过 CAS 把当前仍为 Pending 的 Task直接写为 Cancelled，不执行副作用；
3. Running Task 不收到取消信号，继续执行到 Succeeded、Failed 或 TimedOut；
4. 当前 Running Task全部结束后，所有计划 Step 都成功则 Operation 为 Succeeded，否则为 Cancelled；
5. ExecutionLock 在所有 Task terminal 前保持不变。

取消是 graceful stop，不是回滚，也不承诺撤销已经完成的 Step。未来 Step 尚未创建 Task，Console 根据 Operation desiredState 将其展示为未执行/已取消，不为展示状态额外创建 Task。Operation 已 terminal 后拒绝 cancel。Cancelled Operation 只有通过 `/retry` 才能恢复 Active。

### 8.2 Timeout

所有时间判断使用 server time：

- 等待前序 Operation 或 Lock 的 Pending 排队时间不计入 timeout，此时 `status.deadline=nil`；用户仍可 cancel；
- 首次执行和人工 retry 的 Operation deadline = 本轮 server start time + Operation timeout；
- Task deadline 是创建该 attempt 时的当前 Operation deadline；
- deadline 到达后 Controller 不再创建新 Task，并通过 CAS 把 Pending Task写为 Cancelled；
- deadline 到达时 Agent 终止命令并确认退出；Agent 失联时 Controller 等待固定 termination grace period 后可通过 CAS 将 Running Task 写为 TimedOut；
- 命令正常退出且 postcondition 已达到：Task Succeeded；命令正常退出但 exit code 非 0 或 postcondition 未达到：Task Failed；命令被 deadline 强制终止且已确认退出：Task TimedOut。

Agent 使用 Task 中的 server-time deadline 安排本地终止，因此被纳管节点必须启用常规时间同步。Agent 在 grace period 后提交迟到 terminal status 时，服务端 CAS 失败后必须 GET 最新 Task并接受 Controller 已写入的 TimedOut，不能覆盖终态。

Operation deadline 到达后，Controller 不再创建新 Task。Agent 在线时负责终止命令并提交 TimedOut；Agent 失联超过 termination grace period 后，Controller 终结 Running Task、归约 Operation 并释放 Lock。`terminationGracePeriod` 是首版固定的 2 分钟 engine 常量，不进入 Operation spec，也不形成第二套 Task timeout。超时可能留下部分副作用，后续 retry 必须通过 Observe/Act/Verify 收敛。

首版不提供 Step timeout。executor 可以在 typed payload 或实现内部设置单条命令的技术超时，但它不形成 Operation Engine 的第二套持久状态机。

### 8.3 Retry

Step 自动 retry 和用户手工 retry 共用同一套 Task attempt 机制：

- Step `retryLimit` 控制每个未成功 Node 的自动附加 attempt 数；
- 自动 retry 发生时 Operation 保持 Running，不增加 `Retrying` phase；
- `/retry` 只接受 Failed、TimedOut、Cancelled、已无 Pending/Running Task且仍为该 target 最新 Operation 的对象；
- v2 首版不限制人工 `/retry` 总次数；retryGeneration 是单调递增的请求序号。对已有 Task 的当前 Step，每次成功调用只为未完成 Node额外授权该 generation 一个 Task；它不重置 base generation 的 retryLimit；
- server 使用 resourceVersion CAS 递增 `spec.retryGeneration` 并恢复 desiredState=Active；
- Controller 观察新 generation，设置新 deadline，从第一个未完成 Step继续；
- 已 Succeeded 的 Step 和当前 Step 中已 Succeeded 的 Node不重跑；Failed、TimedOut、Cancelled Node创建 attempt+1 的新 Task；
- 新 Task复制上一 attempt 的稳定 executor/payload，只更新 name、UID、retryGeneration、attempt 和 deadline；跨 Step input 继续读取历史唯一 Succeeded Task 的 output；
- 旧 Task terminal status 不修改，Operation/Task 不删除。

`status.observedRetryGeneration` 只表示 Controller 已处理该 spec generation，不表示 Task 已全部创建。对于仍为最新 Operation 的 generation，Controller 即使已经 observed，也必须在每次 reconcile 中根据当前 generation 和 Task facts补齐缺失 Task。若在 Lock 后或部分 Task Create 后退出，新 leader仍按确定性名称补齐；若 generation 已因出现更新 Operation 而跳过，则不创建 Task，原 terminal status 保持不变。`status.startedAt/deadline/finishedAt/reason/message` 表示当前 generation；获取 Lock 后开始 retry 时清空旧 reason/message/finishedAt，并持久化新的 server-time startedAt/deadline。

如果 `/retry` 请求的 response 丢失，客户端先 GET Operation：`spec.retryGeneration` 已高于请求前 generation、`observedRetryGeneration` 已推进或 Operation 已重新 Pending/Running，均表示请求已生效；不得无条件再次递增 generation。

## 9. Agent 注册、认证和 Lease

### 9.1 首版注册流程

首版只通过 `kcctl join` 和现有 SSH 安装链路纳管 Agent，不实现自定义 CSR API。`AgentID` 延续当前 Agent 使用的 UUID，由 `kcctl join` 生成并写入 Node name 和 Agent 配置：

```text
kcctl join 生成唯一 AgentID
  -> pre-create Node(name=AgentID)
  -> 使用现有 CA 签发每 Agent 唯一 mTLS certificate
  -> SSH 写入 Agent certificate/key/CA 和 server endpoint
  -> start kc-agent
  -> Agent 使用证书 mTLS GET Node(AgentID)，缓存当前 Node object UID
  -> Agent updates own Node status and creates/updates its Lease
  -> Agent starts Task List/Watch and local log endpoint
```

约束：

- 每个 Agent 使用一张独立 mTLS certificate，不共享 Agent credential；证书 CN 为 `system:kc-agent:<AgentID>`、Organization 为 `system:kc-agents`，DNS SAN 为 AgentID，EKU 包含 `clientAuth` 和 `serverAuth`。同一张证书用于 Agent 访问 kc-server 和提供本地日志 HTTPS；不复用 kcctl certificate 或旧 NATS certificate；
- 证书只由能够读取本地 API CA 私钥的 `kcctl join` 签发；kc-server 不在线代签，Agent 私钥、签发用 CA 私钥和完整 client credential 不写入 etcd 或 ConfigMap；`kcctl join` 必须先完成 Node 预创建和 SSH 预置，再启动 Agent 和创建依赖该 Agent 的 Task；
- 现有 HTTPS listener 可以继续使用 `VerifyClientCertIfGiven` 以兼容 kcctl 的其他认证方式，但所有 Agent 专用 API route 必须要求 TLS peer certificate 已存在且已由配置的 CA 验证；无证书、证书链错误或 CN 不符合 `system:kc-agent:<AgentID>` 的请求在进入 handler 前返回 Unauthorized；
- kc-server 从证书 CN 提取 AgentID，并要求对应 Node 存在且 `Node.metadata.name == AgentID`；请求 body 中的 node name 不可信；
- Node object UID 不进入证书，只用于 Task `nodeRef.uid` 校验和 Agent 启动后的当前对象缓存；AgentID 永不复用，因此旧证书不能获得新 AgentID 的权限；
- kc-server 访问 Agent 日志端点时，以 AgentID 作为 TLS ServerName 校验该节点证书，并使用独立的 `system:kc-server` client certificate；Agent 日志端点只信任该 server 身份。日志连通性不进入 Operation 正确性路径；
- 证书过期前告警；首版通过 kcctl 重新签发并 SSH 替换，自动 rotation 后续独立设计；
- 首版不实现 CRL、在线撤销或 Agent CSR；若私钥泄露、Agent 永久下线或需要重新纳管，必须先停止旧 Agent、删除/禁用旧 Node，再生成新的 AgentID、Node object UID 和证书；AgentID/credential 永不复用，CA 私钥不下发 Agent；
- API version 已经是协议边界，不额外实现 major-version negotiation 状态机。

### 9.2 复用现有 RBAC 的最小授权

Agent API 复用 kcctl 已使用的 x509 authenticator 和现有 RBAC，不实现 Kubernetes NodeAuthorizer/NodeRestriction，也不创建每 Agent 的 User、Role 或 RoleBinding：

- x509 authenticator 将 certificate CN 映射为 user name，将 Organization `system:kc-agents` 映射为 group；
- kc-server 初始化一个内置 `kc-agent` GlobalRole，以及一个 `Kind: Group, Name: system:kc-agents` 的 GlobalRoleBinding；
- `kc-agent` Role 只允许 GET Node、UPDATE Node/status、GET/CREATE/UPDATE Lease、LIST/WATCH/GET OperationTask 和 UPDATE OperationTask/status；不授予 Operation、ExecutionLock、Task Create/Delete 或 Task 普通 Update；
- Node、Lease 和 Task handler/store 继续做归属校验：证书 CN 中的 AgentID 必须等于 Node/Lease name 或 `task.spec.nodeRef.name`；List/Watch 由服务端固定该 AgentID 的 selector；
- 请求参数和 body 中的 AgentID 不参与身份判断，不能覆盖证书中的 AgentID；其他用户继续使用现有 RBAC。

RBAC 决定 Agent 可以调用哪些 API，归属校验决定它可以操作哪个对象。归属校验只是几个固定相等关系，不形成新的可配置授权系统。

### 9.3 Node status 和 Lease

- Node status 低频报告版本、能力、管理 IP、日志 secure port 和 Conditions；
- Agent 只有在 certificate、API client、Task reflector、single worker 和 executor registry 可用后才报告 Ready；日志 endpoint 不影响 Ready；
- Lease 以 Node name 为固定名称，由 Agent Create-or-Update，使用 server time和最小字段；
- Operation Controller 不读取 Node Ready 或 Lease；Node/Agent 离线时已创建 Task保持 Pending；
- Lease 不参与 Task 创建、retry、取消、结果判断或 Lock 释放；
- Lease 过期只表示 Agent liveness unknown。

## 10. 跨 Step 小型输出

跨 Step 数据直接保存在成功 Task 的 `result.outputs map[string]string`：

- 单 key 最长 128 bytes；
- 单 value 最长 4 KiB；
- 单 Task outputs 总计不超过 16 KiB；
- 只有 executor schema 声明的 key 可以写；
- Controller 按 `fromStepID + fromNodeUID + outputKey` 读取；
- 来源 Task 必须唯一且 Succeeded；缺失、重复或超限使 Operation Failed；
- 值只注入下一 Task 的声明字段，形成最终 immutable payload；
- 不提供表达式、模板语言、Output 资源或 artifact 传输。

output 有效期由具体业务 executor 自己负责，Operation Controller 不维护通用 expiry 元数据，也不在人工 retry 前做有效期校验。首版假定 kubeadm join token 等短字符串的正常 TTL 足以覆盖常规执行和 retry；若消费 executor 发现 output 已过期，按普通执行失败写入 Task，用户需要创建新的 Operation 重新生成该 output。旧成功 Task 的 output 不修改。

join token 等短期小字符串允许进入 outputs。这是明确接受的简化：具备 Task read 权限的用户可以看到这些值。因此 Task read 权限必须限制给管理员，日志、Message、Condition、审计摘要和普通列表展示必须隐藏 outputs。部署可启用 etcd at-rest encryption，但它不是执行正确性的前提。长期私钥、大型证书包和备份文件不得使用 outputs。

## 11. 日志

Task 日志只写 Agent 本地文件：

```text
/var/log/kubeclipper-agent/tasks/<taskUID>.log
```

同一 Running Task 重入时 append restart 分隔行，不 truncate。Agent 提供只读 mTLS 端点：

```text
GET /v1/tasks/{taskUID}/logs?offset=N&limit=M
```

读取链路：

```text
kcctl / Console
  -> kc-server Task log API
  -> authorize caller and GET OperationTask
  -> resolve Task nodeRef and Node management IP
  -> mTLS GET kc-agent:10260 Task log
  -> stream bounded response
```

规则：

- 用户不能提供 Agent URL 或文件路径；
- Agent 只接受 server-to-agent mTLS identity；
- kc-server 从 Task 获取 `nodeRef.name`，以该 AgentID 作为 TLS ServerName 校验 Agent serving certificate；Agent 校验 kc-server client certificate 身份；
- `offset >= 0`，单次 limit 最大 1 MiB，固定超时和响应大小上限；
- 单 Task 文件和整个日志目录都有硬配额；超限后停止追加并记录日志截断，但不能阻止 Task terminal status；
- token、kubeconfig、证书私钥必须 redaction；
- Agent 不可达、文件丢失和日志超时不改变 Task/Operation phase；
- 本地日志按 terminal age、单 Task 配额和目录总配额清理，不依赖 Operation 删除；首版不保证节点删除后的日志可恢复。

SSH 可达不等于所有 kc-server 副本都能访问 Agent `10260`。日志请求连接失败时直接返回不可用并记录指标；不增加 `AgentEndpointReady` Condition，也不把日志连通性作为安装、Agent Ready 或 Operation 执行门槛。

现有日志路径为 `GetOperationLog -> NATS DeliverLogRequest -> Agent oplog`。v2 复用 `pkg/oplog`、offset 和 `kcctl --follow`，只替换中间 transport 和日志 identity。

## 12. 资源生命周期

- Operation 和 OperationTask 默认是不可删除的执行账本，API 不向用户、Agent 或普通 Controller 提供 DELETE；删除整个 Cluster 时允许 Cluster lifecycle controller 在删除 Operation 完成、没有 Pending/Running Task 且 Lock 已释放后，按 `targetRef.uid` 清理该 Cluster 的 Operation/Task 历史；
- 取消只能持久化 `Operation.spec.desiredState=Cancelled`，不能用删除表达取消；
- ExecutionLock 是短生命周期协调对象，只能由 Operation Controller 删除；
- Operation terminal 且已无 Pending/Running Task时，先持久化 terminal status，再删除 Lock；
- Failed、TimedOut、Cancelled Operation retry 时重新获取同名 Lock；历史 Task仍然保留；
- 首版不实现通用 TTL/history GC controller。Cluster 删除清理是唯一的受控历史清理路径，必须使用 Cluster UID、只清理安全终态且无 Lock 的对象，并支持重复执行；平时只监控 Operation/Task 数量和 etcd 占用；
- Cluster 删除固定顺序为：删除业务 Operation terminal -> 无 active Task -> Lock 已释放 -> `CleanupByTargetUID` 删除 Task/Operation 历史 -> 删除 Cluster 对象。cleanup 失败时 Cluster 保持 Terminating 并重试，因此不需要给 Operation/Task 增加 finalizer；
- 本地日志按独立的时间和磁盘配额清理，不依赖 Operation/Task 删除；
- Node remove 的 Agent Task 必须先成功上传 terminal status。停止 Agent、撤销证书和删除 Node 由 Controller 在 Operation terminal 后执行，不能由最后一个 Agent Task 自己完成。

## 13. 故障恢复矩阵

| 故障点 | etcd 中的事实 | 恢复行为 |
|---|---|---|
| Lock 创建前 Controller 退出 | Operation Pending | 新 leader再次 Create Lock |
| Operation Create 成功但 response 丢失 | 同名 Operation 已存在或不存在 | 调用方用同一 name 重试；AlreadyExists 后 GET 并比较 plan，不创建第二个 Operation |
| 业务 Controller Create Operation 后、回写 owner status 前退出 | Operation 已存在，owner status 可能为空 | owner reconcile 使用稳定 name GET/compare，再补写 reference |
| Lock 创建成功、response 丢失 | Lock holder 已存在或不存在 | GET Lock；自己持有则继续，否则重试 Create |
| Lock 后、Task 前退出 | Operation + Lock | 新 leader创建确定性 Task |
| 部分 Task 创建后退出 | Task 子集 | 补齐缺失 Task，验证已存在 spec |
| Agent Running CAS 前退出 | Task Pending | relist 后重新选择，无副作用 |
| Running CAS response 丢失 | Task Pending 或 Running | GET；只在确认 Running 后执行 |
| Running 后、副作用前退出 | Task Running | 重启后 Reconcile 同一 Task |
| 副作用中退出 | Task Running | 从实际状态继续 Reconcile |
| 副作用后、terminal 前退出 | Task Running | 已达到 postcondition 时直接成功 |
| terminal response 丢失 | Task Running 或 terminal | GET；terminal 则确认，Running 则重新检查 |
| Task terminal 后 Controller 退出 | Task terminal | 新 leader纯函数归约 Operation |
| Task 已更新但 informer cache 尚未观察 | etcd 新事实、cache 旧事实 | 不可逆动作前强一致 ListTasks；事实变化则重新排队 |
| 一个 Task 失败、兄弟 Task仍运行 | Task facts + Lock | 取消 Pending sibling，Running sibling自然结束；全部 terminal 后才 retry |
| retry Task 创建前后退出 | 旧 terminal Task + retry generation | 按含 attempt 的确定性名称补建，不重开旧 Task |
| retry generation observed 后、Task 创建前或部分创建后退出 | generation-tagged Task 子集 | observed 不代表完成；新 leader按 level-driven reducer补齐 |
| retry 请求 response 丢失 | retryGeneration/resourceVersion | GET 判断 generation；不得无条件再次递增 |
| retry 请求后出现更新的同 target Operation | Operation creation facts + Lock | Lock 获取前重检；旧 Operation 不得在新 Operation 之后续跑 |
| cancel 与最后成功并发 | desiredState + Task facts | 全部计划成功则 Succeeded，否则等待 Running 结束后 Cancelled |
| Watch 断开或 410 | Task 对象仍在 etcd | relist，不丢 Task |
| Lease 过期 | Task/Lock 不变 | 标记 liveness unknown，不重试、不释锁 |
| Agent 永久丢失且 Task Running | Task Running + deadline | deadline + termination grace period 后 Controller CAS 为 TimedOut，归约 Operation并释放 Lock |
| Agent 日志不可达或丢失 | Task terminal 事实仍在 etcd | 日志报错，执行结论不变 |

## 14. NATS 职责替换

| 当前 NATS 能力 | v2 替代 | 持久事实 |
|---|---|---|
| Operation/Step 投递 | Controller 创建 Task | OperationTask spec |
| request/reply result | Agent 写 Task status | OperationTask status |
| Operation goroutine/callback | Controller reconcile | Operation/Task/Lock |
| Agent 注册/Get Node | 预置证书 + Node API | Node |
| Node status publish | scoped Node status API | Node status |
| Lease proxy | scoped Lease API | Lease |
| terminate message | Operation desiredState + Controller reconcile | Operation spec + Pending Task status |
| 日志请求 | kc-server GET Agent HTTPS | Agent local file |
| 连接状态判断在线 | Node Lease | Lease |

现有 `DeliverCmd` 不能统一替换成 Operation：

- 修改机器期望状态：迁移为 typed OperationTask；
- 周期性证书/健康观察：Agent 写自身 Node status，由 server controller 汇总到 Cluster status；
- Kubernetes API 已有数据：kc-server 直接使用目标集群 client；
- 任意在线 shell：删除，不在 v2 提供替代入口。

## 15. 实现边界

### 15.1 OCI Artifact 与 static server 边界

OCI 是内容分发层，Operation 是执行控制层，两者不能互相替代：

```text
OCI Registry/cache
  -> immutable artifact bytes identified by digest
  -> Operation plan stores executor + digest reference + bounded parameters
  -> Controller creates immutable node-bound Task
  -> Agent fetches/verifies digest, Reconcile, Verify, writes Task status
```

集成规则：

- Addon/业务 Controller 必须在创建 Operation 前把 mutable tag 解析为 immutable digest；Agent 执行期间不得再次解析 tag；
- 业务 reconcile 必须先 GET 稳定 Operation name；只有 NotFound 才解析 tag。已有同名 Operation 的 digest 是该 generation 的既成事实；
- Operation/Task 只保存固定 artifact reference、executor 类型、必要 hash 和有界参数；Chart、镜像、备份、二进制和其他 blob 不进入 etcd；
- Registry credential 不进入 Operation、Task、outputs、日志或公开 cache metadata。OCI executor 接入前必须明确使用现有 Node 本地凭证，或实现按 Task/Agent 身份授权的运行时读取；Operation Engine 不新增 Secret/Artifact 资源；
- digest 内容已存在于校验通过的本地 cache 时可以离线执行；Registry 与 cache 都没有该 digest 时，Task 按普通执行失败，不能静默回退到同 tag 的其他内容；
- Catalog sync、tag resolve 和 cache GC 不修改目标 Cluster，不创建 Operation，也不获取 Cluster ExecutionLock；install/upgrade/uninstall 才创建 Operation；
- Addon Controller 只创建/观察 Operation，不创建 Task。Operation Controller 仍是 Step 顺序和 Task spec 的唯一 owner；
- static server 若只被 OCI 替换 Addon/Chart 分发，不影响 Operation 状态机；若 OCI 还替换 Agent binary、extension、`kcctl join` bootstrap assets，则属于安装/bootstrap 迁移，必须在 Operation v2 垂直闭环稳定后单独实施；
- 删除 static server 的前提是全新安装、重新纳管、离线安装和版本升级均已不再引用它。不要把 static server 删除设为 Operation v2 Phase 1/2 的前置条件。

### 15.2 代码边界

建议复用现有 core API、storage、clientset、informer 和 controller-runtime 结构，只增加必要包：

```text
pkg/scheme/operations/v1alpha1     Operation/OperationTask/ExecutionLock types
pkg/apis/operations/v1alpha1       REST handlers and route registration
pkg/server/registry/operationv2    Operation storage and status strategy
pkg/server/registry/operationtask  Task storage and selectors
pkg/server/registry/executionlock  Lock storage
pkg/models/operationv2             typed OperationStore（封装三资源 storage、CAS 和 transition validation）
pkg/controller/operationv2           level-driven reducer and reconciler
pkg/agent/controlclient            mTLS List/Watch/status client
pkg/agent/taskrunner               single worker
pkg/agent/executor                 typed Reconcile registry
pkg/agent/logserver                read-only Task log endpoint
pkg/server/agentclient             log client
```

不提前创建通用 state/identity/fault framework。确定性 Task/Lock name 和 reducer 先放在 `pkg/controller/operationv2` 的小型纯函数文件中；只有出现真实复用需求时再抽包。旧 `pkg/controller/operationcontroller` 在迁移期只服务 legacy Operation，最终整体删除，不与 v2 共享状态机。

禁止：

- HTTP handler 或 controller 启动业务执行 goroutine；
- Controller 通过 HTTP API 自调用，或在 reconcile 中访问 `storageFactory`/裸 etcd；
- Agent 读取 Operation并推进 Step；
- 多个组件写 Operation status；
- Controller 根据 Node Ready、Lease、Watch connection 或日志推进状态；
- executor 绕过 Reconcile 契约执行任意命令；
- 为了自动恢复而把不确定副作用标记为成功或安全失败。

## 16. 测试要求

### 16.1 状态与 API

- Operation/Task 合法状态边、Task attempt terminal 不可逆、Operation `/retry` 唯一重开路径；
- spec immutability、Task status CAS、server transition time；
- 包含 retryGeneration/attempt 的 deterministic Task name 和 deterministic Lock name；
- Step barrier、失败后 sibling cancel、cancel/completion race；
- Operation/Task DELETE 被拒绝，retry 不删除或改写历史 Task；
- Task field selector、List resourceVersion、Watch reconnect 和 410 relist；
- Agent 只能写自身 Node name 的 Task status、Node status 和 Lease。
- Controller Store 写入与 Agent HTTP 写入复用同一套 transition validation 和 CAS 语义。

### 16.2 故障注入

对第 13 节每个持久化边界注入 server/Agent kill -9、API response loss 和断网，验证：

- Task 不丢失；
- 未确认 Running 前没有副作用；
- 重复 Watch 不并发执行；
- Running Task 重启后再次 Reconcile；
- 下一 Step 不提前创建；
- failed Step 只为未成功 Node创建新 attempt，已成功 Node不重跑；
- 旧 attempt Running 时不会创建新 attempt；
- Lease/日志故障不触发状态变化；
- 不确定现场保留 Lock；
- cancel 不删除或提前回收 Operation/Task。

### 16.3 Executor

每个真实 executor 必须列出 postcondition 和所有 partial-effect 边界，并测试同一 Task UID 重复 Reconcile：

- kubeadm init/join/reset；
- node add/remove；
- etcd backup/restore；
- certificate replace；
- addon/CRI/registry apply/remove。

测试至少覆盖：已达到目标、明确未开始、部分完成、API response loss、Agent 重启和实际状态未知；输入必须保持稳定。不能证明重复调用可收敛的 executor 不得发布。

### 16.4 指标和审计

最低限度暴露：

- Operation phase 数量、运行时长、排队时长和 retry generation；
- Task phase、attempt、执行时长和 status Conflict；
- 当前 Lock holder 和持锁时长；
- Agent List/Watch relist、410、断线和 workqueue depth；
- Agent Lease age、证书剩余有效期、日志磁盘使用和截断次数；
- Operation/Task 对象数量以及 etcd 数据库占用。

Operation create/cancel/retry、Controller 自动 retry、`RetrySkippedNotLatest`、强制 timeout、Cluster 历史清理都写审计事件。指标、事件和日志不得参与 reducer，也不得包含 outputs 或凭证。

## 17. 对抗性审查

本设计按“攻击每一个隐含假设”的方式审查，结论如下：

| 对抗场景 | 结果 | 设计依据 |
|---|---|---|
| Watch 被当消息队列，断开后丢任务 | 不成立 | Task 持久化，Agent initial List + 410 relist |
| Controller 重启导致重复 Task | 不成立 | 确定性 Task name + immutable spec |
| stale informer cache 导致错误 terminal/释锁 | 不成立 | attempt、terminal、release、cleanup 前强一致 Task list复核 |
| Operation Create response 丢失导致重复计划 | 不成立 | caller-stable Operation name，禁止 generateName，AlreadyExists 后 GET/compare |
| Agent 在 Running 写入前执行 | API/runner 明确禁止 | 必须等待 CAS response或 GET 确认 |
| 副作用完成后 Agent 崩溃 | 会重复 Reconcile，不承诺 exactly-once | executor postcondition 契约 |
| Lease 过期导致重复 destructive Task | 不成立 | retry 只由 Task terminal + retry policy 驱动，Lease 不进入正确性路径 |
| 客户端伪造 target UID 绕开 Cluster Lock | 被拒绝 | Operation Create 时 GET 并校验 target kind/name/UID |
| 两个 Operation 修改同一 Cluster | 被原子 Lock 串行化 | target UID Lock Create |
| 两个 Pending Operation 竞争执行 | 按创建顺序串行化 | Controller 按 `creationTimestamp + resourceVersion + UID` 只放行最早未终态 Operation |
| Lock 独立过期误放行 | 不成立 | Lock 无 TTL/renew；Controller 只在 Task 经 deadline 流程终结后删除 |
| Pending Task 所在 Agent 离线后永不结束 | 不成立 | deadline 后 Controller 可 CAS Pending -> Cancelled，Operation 最终 TimedOut |
| 等待 Lock/deadline 时没有新事件 | 不会永久悬挂 | 持久 deadline + bounded RequeueAfter + startup relist |
| Task 失败后其他节点继续启动 | 存在 Pending -> Running 竞态，但可收敛 | CAS 决定是否启动；已 Running Task自然结束 |
| cancel 与 Running Task success 并发 | 不丢真实成功 | cancel 不改 Running Task；已验证 Succeeded 保留 |
| retry 重跑已成功 Node | 不允许 | Controller 按 `(stepID,nodeUID)` 聚合，任一 attempt Succeeded 即跳过 |
| retry 与旧 Running Task 重叠 | 不允许 | retry API 和 reducer 都要求旧 attempt 无 Pending/Running Task |
| observedRetryGeneration 已推进但 Task 尚未创建 | 不丢请求 | observed 只表示已接收；reducer 始终补齐当前 generation Task |
| 较旧 Operation 在更新 Operation 后 retry | 跳过 | Controller 推进 observed generation、记录 skipped，保留原 terminal status且不创建 Task |
| OCI tag 在执行期间被覆盖 | 不影响当前执行 | Operation/Task 只携带创建前解析的 digest |
| Registry 临时不可用 | cache 命中可继续，否则 Task Failed | 不把 registry liveness 写进 Operation 正确性状态机 |
| 用户删除 Operation/Task 导致事实丢失 | 不成立 | 普通 API 不提供 DELETE；只有 Cluster 删除流程按 target UID 清理已安全终态历史 |
| 节点永久丢失且遗留 Running Task | 最终 TimedOut | deadline + termination grace period 后 Controller 终结 Task/Operation并释放 Lock |
| join token 泄漏到普通日志 | 通过 schema/redaction/展示约束防止 | outputs 仅 Task read 用户可见 |
| Agent 日志磁盘满影响执行结论 | 不允许 | bounded spool，日志失败不阻止 status |
| 两个 Agent 使用同一身份 | 不属于合法部署 | 安装/重新纳管为每个 Node UID 生成独立身份和证书，禁止复制复用 |

审查后仍接受的残余风险：

1. 外部副作用与 status 之间仍没有原子事务；
2. Agent 本地日志随节点或磁盘丢失；
3. Task read 用户可以读取 outputs 中的短期 token；
4. 首版证书轮换依赖 kcctl + SSH；
5. Agent 失联时 Controller 会在 deadline + termination grace period 后强制 TimedOut 并释放 Lock；该取舍依赖 executor 前台执行和 deadline 终止约束，极端情况下仍可能留下部分副作用；
6. 未删除 Cluster 的每次 retry 都会增加 Task 对象；Cluster 删除后由受控清理释放该 Cluster 历史，仍必须监控对象数量和存储容量。

这些风险均是显式运维边界，不通过新增 session、fence、inbox 或消息系统掩盖。

第 5 项不再增加人工 force-timeout API。进入 destructive executor 前必须通过故障测试验证 Agent deadline 终止、Controller grace-period 强制 TimedOut、迟到 status Conflict 和 Lock 释放顺序。

## 18. 验收标准

1. NATS 进程、配置、端口、证书、代码和 Go 依赖全部删除；
2. Operation、Task 和 Lock 的控制事实以及 Node/Lease 的状态事实全部进入 etcd；
3. Agent 只通过 HTTPS API List/Watch 自身 Task并写 status；
4. Controller 独占 Step 顺序和 Operation status；
5. 同一 Cluster 的修改型 Operation 由最小 Lock 串行化；
6. Agent 无本地任务数据库，重启后从 Running Task恢复；
7. 所有 executor 对相同 Task UID 重复 Reconcile 可收敛；
8. List/relist/resourceVersion/410、cancel、timeout、Step retry attempt、人工 `/retry` 和不可删除约束均有故障测试；
9. 日志只在 Agent 本地，通过 mTLS 按 Task UID/offset 读取，日志故障不改变结果；
10. OCI Artifact 若被 Task 使用，tag 在 Operation 创建前固定为 digest，blob 和 credential 不进入 etcd；
11. 全新安装和重新纳管流程不依赖 NATS，旧 Agent/Operation 明确不兼容。

## 19. 参考

- kubelet Pod ListWatch field selector：`kubernetes/pkg/kubelet/config/apiserver.go`
- Kubernetes Job controller：`kubernetes/pkg/controller/job/job_controller.go`
- Kubernetes Node Lease：`kubernetes/staging/src/k8s.io/component-helpers/apimachinery/lease/controller.go`
- API resourceVersion/Watch：https://kubernetes.io/docs/reference/using-api/api-concepts/
- kubelet 日志代理：https://kubernetes.io/docs/concepts/cluster-administration/logging/
- 当前 Operation/NATS 入口：`pkg/controller/operationcontroller/controller.go`、`pkg/agent/agent.go`、`pkg/service/delivery`、`pkg/service/task`
