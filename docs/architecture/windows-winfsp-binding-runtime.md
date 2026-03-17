# Iter 22 — WinFsp binding preflight integrated into mount runtime

## Goal

Move the Windows mount runtime closer to a real WinFsp-backed product flow by making WinFsp environment validation part of runtime startup instead of leaving it as an implicit external dependency.

## Delivered

- new `internal/winfsp` binding probe abstraction
- Windows binding probe discovers:
  - WinFsp user-mode DLL path
  - WinFsp launcher path when present
- Windows binding probe calls the native WinFsp `FspFileSystemPreflight` API against the requested mount point
- `winclientruntime` records host binding metadata into its runtime snapshot
- Dashboard and Diagnostics show:
  - host binding backend
  - host DLL path
  - launcher path
  - binding/preflight summary
- `host_windows.go` now reuses the same binding probe during run

## Why this iteration matters

Before this step, the runtime state machine could shape the mount lifecycle, but it could not tell the user whether WinFsp itself was actually available or whether the requested mount point passed WinFsp's own preflight checks.

After this step, runtime startup fails early when:

- the WinFsp DLL cannot be found
- the requested mount point is invalid for WinFsp preflight

This makes the Windows client substantially more product-like because mount failures are now tied to concrete host-binding information instead of generic placeholder errors.

## Important boundary

This is **not yet** a full WinFsp SDK callback dispatcher implementation. The host path now performs real WinFsp DLL discovery and native preflight validation, but the request-serving loop is still the lightweight host shell introduced earlier.

That means the current branch now has:

- product-shaped runtime state machine
- real WinFsp dependency discovery
- real native preflight integration
- UI-visible host-binding diagnostics

It still does **not** have a complete user-mode file system dispatcher that services Explorer traffic through the WinFsp SDK callback table.

## Recommended next step

Implement the full SDK-backed dispatcher/service loop and then promote the runtime phase from “mounted after preflight + host start” to “mounted after dispatcher start confirmation”.
