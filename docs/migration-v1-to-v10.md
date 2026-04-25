# Migration Guide: v1 SBS → ATOMFederationOS v10

## Overview

v1 (SBS module) was a single Python library. v10 is a multi-language distributed OS with 4 Go modules + Python SBS. This guide covers the key changes.

---

## What Changed

| Aspect | v1 (SBS) | v10 |
|--------|----------|-----|
| Language | Python only | Go (kernel/operator/agent/federation) + Python (SBS) |
| Deployment | Library import | Kubernetes Operator |
| Determinism | Python-level | Kernel-level (Go, zero external deps) |
| Replay | `ReplayEngine` (Python) | `ReplayEngine` (Go, bit-identical) |
| Scheduling | Single-node | Multi-node with GEB |
| Federation | Not built-in | `atom-federation` module |
| CRDs | Not applicable | `ATOMCluster`, `Workflow`, `Task`, `Policy` |

---

## Quick Comparison

### v1 — Using SBS

```python
from sbs import SystemBoundarySpec, GlobalInvariantEngine

spec = SystemBoundarySpec(allow_split_brain=False)
engine = GlobalInvariantEngine(spec)

ok = engine.evaluate(
    drl_state={"leader": "node-1", "term": 3},
    ccl_state={"leader": "node-1", "stale_reads": 0},
    f2_state={"leader": "node-1", "commit_index": 10},
    desc_state={"leader": "node-1", "commit_index": 10},
)
```

### v10 — Using CRDs

```bash
# Deploy operator
kubectl apply -f https://github.com/mahaasur13-sys/atom-operator/releases/latest/install.yaml

# Create cluster
kubectl apply -f - << 'EOF'
apiVersion: atom.atomesh.io/v1alpha1
kind: ATOMCluster
metadata:
  name: home-lab
spec:
  version: "10.0"
  clusterID: "home-lab-001"
  deterministic:
    tickIntervalMs: 100
    quorumRatio: 0.67
EOF

# Run workflow
kubectl apply -f workflow.yaml
```

### v10 — Using Go API

```go
import "github.com/mahaasur13-sys/atom-kernel/pkg/deterministic"

func main() {
    clock := deterministic.NewDeterministicClock(42)
    
    // Deterministic RNG — same seed → same sequence
    rng := deterministic.NewDeterministicRNG(clock.Seed())
    fmt.Println(rng.Int63()) // deterministic
    
    // FNV-1a UUID — no crypto/rand
    id := deterministic.NewDeterministicUUID("input-hash", "node-1")
    fmt.Println(id) // reproducible
}
```

---

## Key v10 Guarantees

### C1: No `time.Now()` in control flow

```go
// v10: Logical tick only
type DeterministicClock struct {
    tick uint64
    seed uint64
}
// time.Now() NEVER used in tick computation
```

### C2: No `uuid.uuid4()` for identity

```go
// v10: FNV-1a based
func NewDeterministicUUID(inputHash string, nodeID string) string {
    h := fnv.New64a()
    h.Write([]byte(inputHash))
    h.Write([]byte(nodeID))
    return fmt.Sprintf("%016x-%s", h.Sum64(), nodeID)
}
```

### C6: All network via ReplayableMessageQueue

```go
// v10: Total ordering guaranteed
queue := NewReplayableMessageQueue()
queue.Enqueue(msgA, LogicalClock{Tick: 5, NodeID: "n1"})
queue.Enqueue(msgB, LogicalClock{Tick: 5, NodeID: "n2"})
// Deterministic order: msgA before msgB (hash tiebreak)
```

---

## ADRs (Architecture Decision Records)

| ADR | Decision | Rationale |
|-----|----------|-----------|
| ADR-001 | Go for kernel/operator | Performance, static binaries, K8s native |
| ADR-002 | FNV-1a over SHA-256 | Pure Go, deterministic, no crypto deps |
| ADR-003 | GEB for tick sync | Guarantees no node starts N+1 before all commit N |
| ADR-004 | CRD-based control plane | K8s-native, declarative, battle-tested |
| ADR-005 | SBS remains Python | Existing v1 codebase, rapid iteration |

---

## Status Mapping

| v1 Status | v10 Status |
|-----------|------------|
| ✅ PASS | `WorkflowStatusSucceeded` |
| ❌ FAIL | `WorkflowStatusFailed` |
| ⏳ RUNNING | `WorkflowStatusRunning` |
| 🔄 PENDING | `WorkflowStatusPending` |

---

## Unsupported in v10

- Python 3.9 and below (requires 3.10+)
- Non-Kubernetes deployments (Operator required)
- Single-node without K8s (use `make dev-up` for local dev)
- Python-only deterministic replay (Go engine is authoritative)
