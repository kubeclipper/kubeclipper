---
change: operation-engine-v2
status: phase-0-frozen
source: `rg -l 'github.com/nats-io|pkg/simple/client/natsio|pkg/service/delivery|pkg/service/task'`
---

# Operation v2 NATS Inventory

This is the migration checklist for the breaking v2 release. It is intentionally
based on repository call sites, not on the old wire format. A new NATS reference
must either be added here with an owner or fail the zero-reference CI gate.

| Area | Current responsibility | v2 owner | Removal gate |
|---|---|---|---|
| `pkg/simple/client/natsio` | NATS client/server, request/reply and subscriptions | none | Phase 4 |
| `pkg/service/delivery` | Server-to-agent task, command and log delivery | Operation Controller / Agent HTTPS | Phase 3 |
| `pkg/service/task` | Agent registration, Node status, Lease proxy and command execution | Agent control client, Node/Lease client and Task runner | Phase 3 |
| `pkg/controller/operationcontroller` | Legacy Operation goroutines, retries and callbacks | `pkg/controller/operationv2` | Phase 4 |
| `pkg/apis/core/v1/handler.go` | Legacy Operation create/retry/cancel/log endpoints | v2 API handlers and business builders | Phase 3/4 |
| `pkg/controller/cluster_status.go` | Synchronous certificate/health commands | typed observation or direct cluster client | Phase 3 |
| `pkg/controller/clustercontroller/controller.go` | ServiceAccount token and apiserver certificate commands | direct cluster client or typed Task | Phase 3 |
| `pkg/controller/cronbackupcontroller/controller.go` | Backup command delivery | typed backup Task | Phase 3 |
| `pkg/scheme/core/v1/k8s/kubeadm_step.go` | Legacy command/step builders | versioned v2 executor payloads | Phase 3 |
| `pkg/agent/agent.go` | Legacy task service composition | v2 Agent control client and runner | Phase 4 |
| `pkg/agent/config/config.go` | NATS configuration embedded in Agent config | API endpoint and mTLS credentials | Phase 4 |
| `pkg/controller-runtime/manager/manager.go` | `CmdDelivery` dependency exposed to controllers | remove after all business callers migrate | Phase 4 |
| `pkg/server/server.go` | NATS/delivery service and legacy Operation wiring | v2 Store/Controller/Agent API wiring | Phase 4 |
| `pkg/server/config/config.go` | NATS server/client flags and persisted config | remove NATS fields and flags | Phase 4 |
| `go.mod`, `go.sum` | NATS client/server and NATS-only transitive dependencies | none | Phase 4 |

## Current call-site classes

The following classes are tracked separately because replacing them with an
Operation would change semantics:

- Cluster/node/backup/certificate/addon/CRI/registry mutations become complete
  ordered Operation plans. The Controller creates Tasks; the Agent executes one
  Task at a time and writes status.
- Node status and Lease are liveness/observation paths. They use scoped HTTPS
  API calls and never gate scheduling, retry or lock release.
- Kubernetes API reads (for example ServiceAccount tokens) stay in the server
  cluster client when no node-side side effect is required.
- Operation logs are observability only: Agent-local files are read through a
  bounded mTLS endpoint and cannot change a Task or Operation phase.
- The existing arbitrary online shell endpoint has no v2 replacement and is
  removed with the legacy delivery path.

## Zero-reference check

After Phase 4 the following command must return no Go source or deployment
configuration paths (documentation and this inventory are excluded):

```sh
rg -n --glob '*.go' --glob '*.yaml' --glob '*.yml' \
  'github.com/nats-io|pkg/simple/client/natsio|pkg/service/delivery|pkg/service/task|nats' .
```

The check is a release gate, not a migration mechanism. No bridge, dual stack,
fallback or data conversion is permitted.
