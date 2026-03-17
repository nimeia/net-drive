# Iter 20 — Windows client mount runtime state machine

## Goal

Introduce a product-facing mount runtime layer between the UI and the raw `clientcore -> mountcore -> winfsp.Host` chain.

## New package

```text
internal/winclientruntime/
  runtime.go
  runtime_test.go
```

## Responsibilities

`internal/winclientruntime` owns:

- building a mount session from `winclient.Config`
- mapping that into a state machine
- exposing a stable snapshot for the UI
- starting and stopping the host lifecycle

## State machine

```text
idle -> connecting -> mounted -> stopping -> idle
                         \-> error
connecting -> error
```

### Snapshot fields

The runtime snapshot now carries:

- phase
- status text
- last error
- active profile
- server / mount point / volume prefix / remote path
- client instance id
- session id
- principal id
- server name/version
- lease expiry

## UI integration

### Dashboard
- shows live runtime phase and summary
- exposes Start Mount / Stop Mount
- exposes mount CLI preview

### Diagnostics
- shows runtime summary
- shows mount CLI preview alongside advanced smoke commands

## Why this layer matters

Before this iteration, the UI only knew how to execute direct smoke operations. There was no stable abstraction for:

- long-lived mount lifecycle
- UI-readable states
- future reconnect / retry / tray integration

This runtime layer makes later iterations cleaner because tray, notifications, reconnect policy, and diagnostics export can all bind to the same snapshot model instead of directly reading low-level protocol code.

## Current boundary

The Windows WinFsp host implementation on this branch is still a host shell, not a full SDK-backed mount runtime. Even so, the state machine is now product-shaped and testable on non-Windows hosts by swapping the runtime builder/session in tests.
