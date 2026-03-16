# Iter 9 — core hardening / test coverage baseline

This iteration focuses on making the core safer and faster before any WinFsp or mount-layer work.

## Added coverage

### 1. Control-plane negative contract coverage
New integration coverage validates:
- auth-before-hello rejection
- unsupported protocol version rejection
- wrong token rejection
- unsupported channel rejection

File:
- `tests/integration/control_plane_negative_test.go`

### 2. Session gating + backend error mapping
New integration coverage validates:
- metadata requests without an active session return `ERR_SESSION_NOT_FOUND`
- metadata requests after lease expiry return `ERR_SESSION_EXPIRED`
- backend-to-protocol mapping for:
  - `ERR_NOT_FOUND`
  - `ERR_IS_DIR`
  - `ERR_NOT_DIR`
  - `ERR_ACCESS_DENIED`
  - `ERR_INVALID_HANDLE`
  - `ERR_ALREADY_EXISTS`

File:
- `tests/integration/control_plane_negative_test.go`

### 3. Metadata backend edge matrix
New unit coverage validates:
- create with overwrite=false vs overwrite=true
- opendir(file) / openfile(dir) type errors
- invalid handle paths for read / flush / truncate / close
- write on read-only handle
- readdir pagination and cursor bounds
- sparse write semantics
- cross-directory rename with replace/no-replace behavior

File:
- `internal/server/metadata_backend_edge_test.go`

### 4. Journal edge coverage
New unit coverage validates:
- poll `maxEvents` truncation
- ack monotonic behavior
- watch-not-found for poll / ack / resync
- watch path matching rules
- resubscribe behavior when previous watch is absent but explicit node is provided

File:
- `internal/server/journal_test.go`

## Added benchmark baseline

First-round benchmark files:
- `internal/server/metadata_backend_bench_test.go`
- `internal/server/journal_bench_test.go`
- `internal/transport/codec_bench_test.go`

Benchmarks cover:
- hot lookup
- hot getattr
- readdir snapshot hit
- small-file cached read
- create/write/flush/close
- journal poll
- frame encode/decode

## Suggested run order

```bash
go test ./internal/transport
go test ./internal/server
go test ./tests/integration
go test ./internal/server ./internal/transport -bench . -run '^$'
```

## Remaining high-value gaps

Still worth adding in the next wave:
- client package unit tests
- config / status / audit tests
- deeper recovery edge matrix
- multi-client concurrency and churn tests
- benchmark thresholds and trend tracking
