# WinFsp Callback Host and Windows-only Build Tags (Iter 15)

Iter 15 closes the gap between the read-only adapter and a mount-host-facing callback layer.

## New boundaries

```text
internal/winfsp/
  status.go
  callbacks.go
  host.go
  host_windows.go
  host_other.go
```

## What Iter 15 adds
- NTSTATUS mapping in `status.go`
- callback bridge over `internal/winfsp/adapter`
- Windows-only host shell via `host_windows.go` / `host_other.go`
- `-op mount` in `cmd/devmount-winfsp`

## Verification strategy
- cross-platform tests for `internal/winfsp` callback mapping
- Windows-only compile checks via `GOOS=windows GOARCH=amd64 go build`
