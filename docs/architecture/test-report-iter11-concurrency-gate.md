# Iter 11 Core Hardening: concurrency, transport negatives, smoke, benchmark gate

This round extends the test surface from single-client happy paths into runtime stability and regression control.

## Added test areas

### Concurrency and jitter
- `tests/integration/concurrency_jitter_test.go`
- multi-client concurrent create/write/flush/rename against a shared watcher
- heartbeat interleaving with file writes on one client connection
- repeated close/reconnect/resume jitter loops with read verification

### Transport negative coverage
- `internal/transport/codec_negative_test.go`
- short/truncated length prefix
- truncated frame body
- invalid magic
- unsupported header version
- invalid header length
- payload length mismatch

### Command smoke coverage
- `tests/integration/cmd_smoke_test.go`
- builds `cmd/devmount-server` and `cmd/devmount-client`
- boots the server with JSON config
- checks `/status`
- runs the client binary end to end and verifies stdout

### Benchmark regression gate
- `internal/benchgate/benchgate.go`
- `internal/benchgate/benchgate_test.go`
- `cmd/devmount-benchgate/main.go`
- `benchmarks/thresholds.json`
- `scripts/benchmark_gate.sh`

## Intent

The goal of this round is not only more raw test count. It is to increase confidence in:
- concurrent client load
- session stability under reconnect jitter
- transport corruption handling
- CLI entrypoint viability
- benchmark regression visibility in local and CI-style flows

## Remaining gaps after Iter 11
- higher-volume soak tests (minutes rather than seconds)
- benchmark trend storage/history
- optional benchmark gate wiring into CI
- fault injection around half-close / slow-reader / delayed writer transport cases
