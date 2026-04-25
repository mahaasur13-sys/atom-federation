# Contributing to ATOMFederationOS v10

## Setup

```bash
git clone https://github.com/mahaasur13-sys/ATOMFederationOS.git
cd ATOMFederationOS

# Install Go 1.23+
make deps

# Install kubectl + kind for local dev
make dev-deps
```

## Development

```bash
# Start local K8s (kind)
make dev-up

# Run all tests
make test-all

# Run deterministic tests only
make determinism

# Lint all modules
make lint

# Build all Go modules
make build
```

## Module Responsibilities

| Module | Language | Main Files |
|--------|----------|------------|
| `atom-kernel` | Go | `pkg/deterministic/*.go` |
| `atom-operator` | Go | `controllers/*.go`, `api/v1alpha1/*.go` |
| `atom-agent` | Go | `pkg/sandbox/*.go` |
| `atom-federation` | Go | `pkg/clock/*.go`, `pkg/queue/*.go` |
| `sbs` | Python | `sbs/*.py` |

## Determinism Requirements

Before submitting a PR, verify:

1. **No forbidden imports** — no `time.Now()`, `uuid.uuid4()`, `crypto/rand`, `random.*` in control flow paths
2. **Go deterministic tests pass:** `go test ./... -run TestDeterministic`
3. **Python SBS tests pass:** `pytest sbs/tests/ -q`
4. **Build succeeds:** `make build`

## Commit Convention

```
<type>(<scope>): <description>

Types: feat, fix, docs, refactor, test, chore
Scopes: kernel, operator, agent, federation, sbs, ci
```

## PR Checklist

- [ ] `make test-all` passes locally
- [ ] Determinism tests pass
- [ ] No forbidden imports detected
- [ ] Documentation updated (if changing API)
- [ ] Examples updated (if adding new features)
- [ ] ADRs updated (if changing architecture)

## ADRs (Architecture Decision Records)

Located in `docs/adr/`. For significant architectural decisions, create an ADR:

```markdown
# ADR-XXX: <Title>

**Status:** Accepted

**Context:** <Problem statement>

**Decision:** <What we decided>

**Consequences:** <What changes as a result>
```
