# Iter 10 — second-round coverage expansion

This iteration continues the “make the core stable first” line after Iter 9.

## Newly added coverage

### 1. Client package unit tests
Added:
- `internal/client/client_test.go`

Coverage added:
- request header fields are encoded correctly
- response decode happy path
- error response decode path
- request-id mismatch handling
- malformed error payload handling
- `CreateSession()` updates `Client.SessionID`
- `ResumeSession()` restores the previous session id on failure
- `decodeInto()` unsupported target guard

### 2. Config / status / audit tests
Added:
- `internal/server/config_test.go`
- `internal/server/status_test.go`
- `internal/server/audit_test.go`

Coverage added:
- config file load happy path
- invalid JSON config handling
- `ApplyToServer()` field override behavior
- audit logger creation via config
- status snapshot fields and uptime
- `/healthz` and `/status` HTTP responses
- audit logger nil-safe behavior
- audit JSONL newline + timestamp formatting
- audit marshal-error path

### 3. Recovery deep-edge tests
Added:
- `tests/integration/recovery_deep_test.go`

Coverage added:
- writable handle recovery
- delete-on-close preserved across recovered handles
- rename followed by node revalidate
- resubscribe with unknown previous watch id and explicit node/after-seq

## Why this round matters

Compared with Iter 9, this round closes three important gaps:
- client-side request/response safety was previously untested
- runtime support surfaces (`config/status/audit`) are now directly covered
- recovery semantics now cover more than the base happy path

## Suggested local test commands

```bash
go test ./internal/client
go test ./internal/server
go test ./tests/integration -run 'TestRecovery|TestControlPlaneNegative|TestSessionGatingAndProtocolErrorMapping'
go test ./internal/server ./internal/transport -bench . -run '^$'
```

## Remaining high-value gaps

Still worth doing next:
- multi-client concurrency / churn tests
- heartbeat + file ops interleaving
- malformed frame / truncated frame transport negative tests
- `cmd/devmount-server` / `cmd/devmount-client` smoke tests
- benchmark thresholds and regression gating
