# Iter 23 — Windows client logs / self-check / diagnostics export

## Goal

Add a first product-grade diagnostics loop around the Windows client so local testing and later support workflows are not limited to raw mount errors.

## Delivered scope

- new `internal/winclientlog`
  - default log path resolution under the user config directory
  - append-only structured text logging
  - bounded tail reading for diagnostics export
- new `internal/winclientdiag`
  - WinFsp binding/self-check summary
  - TCP reachability check for the configured server address
  - config summary + runtime snapshot summary
  - diagnostics ZIP export with text and JSON forms
- Win32 UI additions
  - `Run Self-Check`
  - `Export Diagnostics`
  - diagnostics summary now includes requested/effective backend, binding state, log path, and export path guidance
- tray additions
  - tray menu can trigger diagnostics export without reopening the whole workflow
- CLI additions in `cmd/devmount-winfsp`
  - `-op selfcheck`
  - `-op export-diagnostics`

## Diagnostics ZIP contents

Current export includes:

- `report.txt`
- `report.json`
- `log-tail.txt` when local log tail data exists

This keeps the first version easy to inspect manually and easy to attach to bug reports.

## Current behavior

Self-check currently focuses on the most useful client-side failures in this branch:

- invalid configuration
- WinFsp DLL / launcher / preflight state
- requested backend vs effective backend
- TCP reachability to `addr`
- local log path presence
- recent log tail summary

This is intentionally a client-side diagnostics baseline. It does **not** yet collect:

- full crash dumps
- Windows event logs
- ETW traces
- remote server diagnostics bundle
- automatic redaction policies beyond what the current report fields already avoid emitting

## Why this shape

The current Windows client is still transitioning from a test harness into a product shell. A compact diagnostics layer gives immediate value without forcing the UI to depend directly on raw WinFsp probing or ad-hoc file packaging code.

That separation also keeps later work open for:

1. richer health checks
2. better error taxonomy
3. support bundle expansion
4. installer/runtime diagnostics reuse
