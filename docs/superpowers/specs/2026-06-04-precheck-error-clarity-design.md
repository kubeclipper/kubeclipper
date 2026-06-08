---
comet_change: precheck-node-error-clarity
role: technical-design
canonical_spec: openspec
---

## Overview

Optimize precheck error messages in `kcctl deploy` to include node IP and role (server/agent), using a merged display format for multi-node failures.

## Architecture Decision

Adopt a minimal-invasion approach: `precheckFunc` signature remains unchanged, returning error messages without host info. Role annotation is handled uniformly by `precheckService` at the caller level.

## Design Details

### 1. nodeRole method

Add `nodeRole(ip string) string` to `DeployOptions`:

```go
func (d *DeployOptions) nodeRole(ip string) string {
    isServer := false
    isAgent := false
    for _, s := range d.deployConfig.ServerIPs {
        if s == ip { isServer = true; break }
    }
    for _, a := range d.deployConfig.Agents.ListIP() {
        if a == ip { isAgent = true; break }
    }
    switch {
    case isServer && isAgent: return "server+agent"
    case isServer: return "server"
    case isAgent: return "agent"
    default: return ""
    }
}
```

### 2. precheckService refactor

Replace `chan error` with structured collection and grouped output:

```go
type nodeError struct {
    host string
    err  error
}
```

- Collect `nodeError{host, err}` per node via goroutines
- Group by `err.Error()`: `map[string][]string` (key=message, value=node refs like `[server@10.0.0.1]`)
- Output grouped format:
  ```
  ============>NTP PRECHECK FAILED!
  chronyd or ntpd service not running, may cause service internal error:
    - [server@10.0.0.2]
    - [agent@10.0.0.5]
  ```
- `PRECHECK OK/FAILED` markers unchanged for backward compatibility

### 3. precheckTimeLag refactor

Replace `chan struct{}` with structured error collection:

```go
type timeLagError struct {
    host string
    lag  float64
}
```

- Collect per-node lag data for nodes exceeding 5s threshold
- On failure, output grouped format:
  ```
  ============>TIME-LAG PRECHECK FAILED!
  time lag exceeds 5s threshold:
    - [server@10.0.0.2] 7.8s
    - [agent@10.0.0.5] -6.3s
  ```

### 4. precheckPortFunc adjustment

Remove host from error message to enable grouping:

- Before: `"port %d is already in use on %s, required by %s"`
- After: `"port %d is already in use, required by %s"`

Role and IP annotation handled by precheckService.

Also adjust the tool-missing error:
- Before: `"port check tool (ss or netstat) not found on %s, skip port availability check..."`
- After: `"port check tool (ss or netstat) not found, skip port availability check, deployment may fail if port is occupied"`

### 5. sudo.MultiNIC refactor

Replace `sshutils.CmdBatch` with custom concurrent collection (same pattern as precheckService):

- Use goroutines + WaitGroup to check all nodes
- Collect multi-NIC nodes into a slice
- On failure, output merged format:
  ```
  ============>ipDetect PRECHECK FAILED!
  node has multiple network interfaces, --ip-detect not specified (default: first-found may choose wrong interface):
    - [agent@10.0.0.5]
    - [agent@10.0.0.8]
  ```
- Role is fixed as `"agent"` since MultiNIC only checks agent nodes

Fixes existing bug: CmdBatch stops on first error, missing other multi-NIC nodes.

### 6. Unchanged parts

- `sudo.PreCheck`: errors already contain `user@host`, no change needed
- `precheckFunc` signature: unchanged
- `generateCommonPreCheckFunc` / `precheckNtpFunc`: error messages unchanged (no host in message)

## Data Flow

```
precheckFunc(host) → error("<message without host>")
        ↓
precheckService collects: nodeError{host, err}
        ↓
Group by err.Error(): map[message][]nodeRef
        ↓
Output per group:
  <message>:
    - [<role>@<ip>]
    - [<role>@<ip>]
```

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Merged display changes CLI output format | Low — output is human-readable, not machine-parsed | Keep `PRECHECK OK/FAILED` markers unchanged |
| MultiNIC refactor introduces sync dependency in sudo package | Low — standard library only | No external risk |
| nodeRole iterates lists, O(n) per call | Negligible — typically < 100 nodes | Acceptable |

## Testing Strategy

- `nodeRole` unit tests: table-driven, covering server-only, agent-only, server+agent, unknown IP
- Build verification: `go build ./cmd/kcctl`
- Manual verification: observe error output format in multi-node deploy scenarios
