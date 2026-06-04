---
change: precheck-node-error-clarity
design-doc: docs/superpowers/specs/2026-06-04-precheck-error-clarity-design.md
base-ref: d3f44c92997da6cfdefff94fc0b3fe3a4666a7b1
---

## Implementation Plan

### T1: Add `nodeRole(ip string) string` method

**File**: `pkg/cli/deploy/deploy.go`

Add method to `DeployOptions`:

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

No imports needed — only uses existing fields.

### T2: Refactor `precheckService` — structured collection + grouped output

**File**: `pkg/cli/deploy/deploy.go`

1. Add `nodeError` struct near `precheckFunc`:

```go
type nodeError struct {
    host string
    err  error
}
```

2. Replace `precheckService` implementation:
   - Change `chan error` to `[]nodeError` (slice, not channel — we drain after `wg.Wait()`)
   - Collect `nodeError{host, err}` per node via goroutine (with mutex)
   - After `wg.Wait()`, group by `err.Error()`: `map[string][]string` (key=message, value=node refs)
   - Output each group: message line, then indented node refs
   - Node ref format: `[<role>@<ip>]` (using `d.nodeRole(host)`)
   - If `nodeRole` returns `""`, use `[<ip>]` as fallback

Output example:
```
============>NTP PRECHECK FAILED!
chronyd or ntpd service not running, may cause service internal error:
  - [server@10.0.0.2]
  - [agent@10.0.0.5]
```

3. Preserve existing behavior: `PRECHECK OK/FAILED` markers, `AssumeYes` handling, confirmation prompt.

### T3: Update `precheckTimeLag` — structured lag collection

**File**: `pkg/cli/deploy/deploy.go`

1. Add `timeLagError` struct:

```go
type timeLagError struct {
    host string
    lag  float64
}
```

2. Replace `chan struct{}` with `[]timeLagError` (mutex-protected slice)
3. Collect per-node lag data when `math.Abs(diff) > 5`
4. Also collect nodes that fail SSH or parsing (append with `lag: 0` and log the reason separately)
5. On failure, output merged format:

```
===========>TIME-LAG PRECHECK FAILED!
time lag exceeds 5s threshold:
  - [server@10.0.0.2] 7.8s
  - [agent@10.0.0.5] -6.3s
```

6. Preserve existing `PRECHECK OK` message for success case.

### T4: Update `sudo.MultiNIC` — replace CmdBatch with custom concurrent collection

**File**: `pkg/cli/sudo/sudo.go`

1. Remove `errorMultiNIC` variable
2. Replace `sshutils.CmdBatch` call with goroutines + WaitGroup:
   - Iterate `allNodes`, launch goroutine per node
   - Each goroutine: run `ip a|grep ": "|awk {'print $2'}|sed 's/://'`, check iface count
   - Collect multi-NIC nodes into `[]string` (mutex-protected)
3. On failure, output merged format:

```
===========>ipDetect PRECHECK FAILED!
node has multiple network interfaces, --ip-detect not specified (default: first-found may choose wrong interface):
  - [agent@10.0.0.5]
  - [agent@10.0.0.8]
```

4. Role is always `"agent"` since MultiNIC only checks agent nodes — pass role as parameter or hardcode.
5. Preserve existing `AssumeYes` handling and confirmation prompt.

### T5: Adjust `precheckPortFunc` error messages

**File**: `pkg/cli/deploy/deploy.go`

1. Port-in-use error — remove host from message:
   - Before: `"port %d is already in use on %s, required by %s"`
   - After: `"port %d is already in use, required by %s"`
2. Tool-missing error in `precheckPorts`:
   - Before: `"port check tool (ss or netstat) not found on %s, skip port availability check, deployment may fail if port is occupied"`
   - After: `"port check tool (ss or netstat) not found, skip port availability check, deployment may fail if port is occupied"`
3. Port check SSH failure — remove host from message:
   - Before: `"check port %d on %s failed: %w"`
   - After: `"check port %d failed: %w"`

Role and IP annotation handled by `precheckService` (T2).

### T6: Unit tests for `nodeRole`

**File**: `pkg/cli/deploy/deploy_test.go` (new)

Table-driven tests covering:
- Server-only IP → `"server"`
- Agent-only IP → `"agent"`
- AIO IP (in both ServerIPs and Agents) → `"server+agent"`
- Unknown IP → `""`
- Multiple servers and agents

### T7: Build verification

```bash
go build ./cmd/kcctl
go vet ./pkg/cli/deploy/ ./pkg/cli/sudo/
```

## Execution Order

T1 → T2 → T3 → T5 → T4 → T6 → T7

T1 is a prerequisite for T2/T3 (they call `nodeRole`). T5 removes host from port error messages so T2's grouping works correctly. T4 is independent but follows the same pattern. T6 depends on T1. T7 is final validation.

## Commit Strategy

One commit per task, message prefixed with change name:
- `feat(precheck): add nodeRole method to DeployOptions`
- `feat(precheck): refactor precheckService with structured error collection and grouped output`
- `feat(precheck): refactor precheckTimeLag with structured lag collection`
- `feat(precheck): adjust precheckPortFunc error messages for grouping`
- `feat(precheck): refactor MultiNIC with custom concurrent collection`
- `test(precheck): add unit tests for nodeRole method`
- `chore(precheck): verify build passes`
