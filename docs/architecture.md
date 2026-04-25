# ATOMFederationOS v10 — Architecture

## Overview

ATOMFederationOS v10 is a deterministic distributed execution OS. Its core guarantee:

> **Replay(T) == Execution(T)** — any execution can be bitwise-reproduced from any point in time.

---

## System Layers

| Layer | Module | Responsibility |
|-------|--------|---------------|
| **User Space** | `atom-agent` | Sandbox execution, DRL monitoring, SBS enforcement |
| **Orchestration** | `atom-operator` | Kubernetes Operator, CRDs, reconciliation loops |
| **Federation** | `atom-federation` | Cross-cluster sync, LogicalClock, message ordering |
| **Kernel** | `atom-kernel` | DeterministicScheduler, DESC, ReplayEngine, GEB |

---

## Data Flow

```
User YAML/SDK
    ↓
atom-operator (Kubernetes API)
    ↓ [Workload allocation]
atom-agent (per-node, sandboxed)
    ↓ [Execute with SBS enforcement]
atom-kernel (deterministic tick execution)
    ↓ [Replay log committed]
atom-federation (cross-cluster sync)
```

---

## Core Components

### Deterministic Kernel (`atom-kernel`)

| Component | Description |
|-----------|-------------|
| `DeterministicClock` | Logical tick counter — no `time.Now()` |
| `DeterministicRNG` | Seeded Xorshift32 — reproducible random |
| `DeterministicUUIDFactory` | FNV-1a + MAC-based — no `uuid.uuid4()` |
| `GlobalExecutionSequencer` | FNV-1a hash ordering — no global locks |
| `GlobalExecutionBarrier` | GEB — quorum sync before tick execution |
| `DeterministicScheduler` | LockstepMode, hash-based tie-breaking |

### Replay Engine

| Guarantee | Mechanism |
|-----------|-----------|
| Bit-identical replay | FNV-1a seed from input hash |
| Cross-node consistency | GEB.commit(N) before N+1 |
| Crash recovery | DeterministicSnapshot + WAL |
| Ordering guarantee | LogicalClock + ReplayableMessageQueue |

### Kubernetes Operator (`atom-operator`)

| Controller | Reconciles |
|------------|------------|
| `ATOMClusterReconciler` | `ATOMCluster` CRD — cluster lifecycle |
| `WorkflowReconciler` | `Workflow` CRD — DAG execution |
| `TaskReconciler` | `Task` CRD — task lifecycle + sandbox |
| `PolicyReconciler` | `Policy` CRD — SBS policy enforcement |

**Reconciliation loop pattern:**

```go
func (r *WorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    // 1. Fetch
    wf := &atomv1.Workflow{}
    if err := r.Get(ctx, req.NamespacedName, wf); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. Compute desired
    desired := r.computeDesired(wf)

    // 3. Update status (BEFORE mutate to avoid reset loops)
    if !reflect.DeepEqual(wf.Status, desired.Status) {
        wf.Status = desired.Status
        if err := r.Status().Update(ctx, wf); err != nil {
            return ctrl.Result{}, err
        }
    }

    // 4. Apply mutations
    op, err := r.mutate(ctx, wf, desired)
    if err != nil {
        return ctrl.Result{}, err
    }
    return ctrl.Result{Requeue: op.Requeue}, nil
}
```

### SBS Enforcement (`atom-agent`)

The System Boundary Spec runs inside every agent sandbox:

```go
type SBSEnforcer struct {
    invariants []Invariant
    mode       SBSMode  // AUDIT | LOG | ENFORCE
}

func (e *SBSEnforcer) Enforce(state LayerState) error {
    for _, inv := range e.invariants {
        if !inv.Check(state) {
            if e.mode == ENFORCE {
                return &InvariantViolation{Name: inv.Name, State: state}
            }
            log.Warn("invariant violated", "name", inv.Name)
        }
    }
    return nil
}
```

---

## Determinism Constraints (C1-C10)

| ID | Constraint | Mechanism |
|----|-----------|-----------|
| C1 | No `time.time()` in control flow | `DeterministicClock` tick |
| C2 | No `uuid.uuid4()` for identity | `DeterministicUUIDFactory` |
| C3 | No `random.*` in scheduling | `DeterministicRNG` (seeded) |
| C4 | No non-deterministic `asyncio.sleep()` | LockstepMode |
| C5 | All FS ops via `AtomicFileWrite` | 2-phase commit |
| C6 | All network via `ReplayableMessageQueue` | Total ordering |
| C7 | All tick boundaries via GEB | Quorum sync |
| C8 | No probabilistic scheduling | Hash-based deterministic |
| C9 | Replay = bitwise-identical output | Certified replay |
| C10 | No RL-019/020/021 kernel changes | Hard invariant |

---

## Deployment Topology

```
┌──────────────────────────────────────────────────────┐
│                  Kubernetes Cluster                   │
│                                                      │
│  ┌─────────────┐     ┌─────────────┐                │
│  │ atom-op     │     │ atom-op     │                │
│  │ (leader)    │     │ (replica)   │                │
│  └──────┬──────┘     └──────┬──────┘                │
│         │                   │                        │
│  ┌──────┴──────┐     ┌──────┴──────┐                │
│  │ atom-agent  │     │ atom-agent  │                │
│  │ (node-1)    │     │ (node-2)    │                │
│  │ └─kernel    │     │ └─kernel    │                │
│  └─────────────┘     └─────────────┘                │
└─────────────────────────────────────────────────────┘
         │                         │
         └─────── GEB.sync() ──────┘
         (no node starts tick N+1
          until all commit N)
```

---

## Version History

| Version | Key Change |
|---------|-----------|
| v0.5.1 | SBS v1 — GlobalInvariantEngine |
| v0.6.0 | Single-source version, Typer CLI |
| v9.x | DRL v7, coherence layer |
| v10.0 | Go modules (kernel/operator/agent/federation), RL-022, GEB, deterministic Kubernetes |
