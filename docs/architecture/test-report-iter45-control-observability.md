# Iter 45 — control path latency sampling + sampled soak report expansion

## Delivered

- Added server-side control path latency sampling for `hello`, `auth`, `create_session`, `resume_session`, and `heartbeat`.
- Exposed control latency counters through `/runtimez`.
- Expanded sampled soak CSV/report to include session/journal lock-wait counters and control-op latency counters.
- Updated Task.md with the next-step performance worklist and marked the completed P0 items in this iteration.

## Validation to run locally

```bash
go test ./internal/server -run 'TestServerSnapshotRuntimeIncludesCountsAndLockSamples|TestStatusHandlerHealthzStatusAndRuntimez' -count=1
go build ./cmd/devmount-soak
go run ./cmd/devmount-soak -dry-run
```
