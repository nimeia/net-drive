# Iter 24 — WinFsp dispatcher host v1 scaffold

## Goal

Extend the Windows mount runtime from simple WinFsp preflight validation toward a selectable host backend model that can later absorb a full dispatcher-backed WinFsp user-mode filesystem host.

## Delivered scope

- `winclient.Config` adds `HostBackend`
  - `auto`
  - `preflight`
  - `dispatcher-v1`
- profile storage / CLI / Win32 UI all surface the same backend field
- `internal/winfsp` binding probe now reports:
  - requested backend
  - effective backend
  - dispatcher API readiness
  - dispatcher status text
- Windows binding probe checks WinFsp dispatcher entry points:
  - `FspFileSystemCreate`
  - `FspFileSystemSetMountPoint`
  - `FspFileSystemStartDispatcher`
  - `FspFileSystemStopDispatcher`
  - `FspFileSystemDelete`
- `internal/winfsp/host_windows.go` routes host execution by effective backend
- `internal/winfsp/dispatcher_windows.go` provides the first dispatcher host scaffold
- runtime/UI/diagnostics all display requested/effective backend and dispatcher state

## What is real in this iteration

This iteration already provides **real** value in three areas:

1. backend selection is explicit and persisted
2. WinFsp dispatcher API availability is genuinely probed from the native DLL
3. runtime and diagnostics no longer treat all host paths as the same opaque “mount backend”

## Important boundary

This is **not** the final ABI bridge for a full Explorer-served WinFsp filesystem.

`dispatcher-v1` in this iteration is a scaffold boundary, not a claim that the repository already contains a production-complete `FSP_FILE_SYSTEM_INTERFACE` implementation plus full dispatcher/service loop wiring.

Today the dispatcher path does:

- validate that dispatcher APIs are present
- select a dispatcher-oriented effective backend
- route the host through a dedicated dispatcher scaffold entry

It does **not** yet fully implement:

- final WinFsp interface table binding
- complete callback marshaling to adapter operations through the dispatcher runtime
- production-grade dispatcher thread/service lifecycle
- full Windows host smoke evidence from Explorer or third-party tools

## Why this intermediate step matters

Without an explicit dispatcher scaffold, every later WinFsp host improvement would stay mixed into the same “preflight-only” path, making it hard to reason about backend behavior in UI, logs, tests, and diagnostics.

This iteration gives the project a clean seam for future work:

1. finish native interface binding
2. attach dispatcher lifecycle metrics/logging
3. add real Windows host smoke verification
4. decide when `auto` should upgrade from `preflight` to dispatcher-backed host execution
