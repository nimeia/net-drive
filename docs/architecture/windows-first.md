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
   TCP placeholder transport (Iter 1)
        |
   Server Runtime
        |
   Local filesystem backend (Iter 1)
```

## Engineering choices

- Use Go for current protocol/server/client bring-up to maximize iteration speed and testability in the sandbox.
- Keep wire contracts explicit and transport-agnostic so a later QUIC transport can replace the Iter 1 TCP placeholder.
- Defer WinFsp integration until the control-plane and protocol contracts are stable.
