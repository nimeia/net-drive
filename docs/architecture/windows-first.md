# Windows-first Architecture

## Goal

Deliver a Windows-first, editor-optimized remote filesystem whose user-visible behavior is as close as practical to local storage for development workloads.

## Layering

```text
Windows Apps (VS Code / Git / Terminal)
        |
      WinFsp   <- future mount adapter
        |
   Local VFS Core
        |
   Protocol Client
        |
   TCP placeholder transport (current implementation)
        |
   Server Runtime
        |
   Local filesystem backend
```

## Current status

The repository currently includes:
- protocol client/server bring-up
- metadata and data read/write baseline
- pull-based watcher/journal baseline
- recovery and reconnect baseline
- editor-focused cache/prefetch optimizations
- productization helpers: config, status endpoints, audit log, build/package scripts

## Engineering choices

- Go is used for the current protocol/server/client bring-up to maximize iteration speed and testability.
- Wire contracts remain transport-agnostic so a later QUIC transport can replace the TCP placeholder.
- WinFsp integration remains deferred until control-plane, filesystem semantics, and recovery contracts are stable.
- Iter 13 introduces `internal/clientcore` as the reusable client runtime boundary that the future WinFsp adapter will depend on instead of the earlier demo-only client package.
- Iter 14 adds `internal/mountcore`, `internal/platform/windows`, `internal/winfsp/adapter`, and a `cmd/devmount-winfsp` smoke CLI to validate the read-only mount path before bringing in a concrete WinFsp callback binding.
- Iter 15 adds `internal/winfsp` callback mapping, NTSTATUS translation, and Windows-only host build tags so the callback layer can compile for Windows without breaking Linux/macOS development.
