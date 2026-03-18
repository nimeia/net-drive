# Iter 33 — WinFsp native security descriptor / cleanup / delete-on-close semantics

## Goal

Close the remaining semantic gap around Explorer security queries and delete gestures in the read-only WinFsp path.

## Delivered scope

- native-style security descriptor model with:
  - SDDL
  - owner / group
  - access summary
  - by-name / by-handle source
  - cleanup / flush / delete-on-close state
- callback coverage extended with:
  - `CanDelete`
  - `SetDeleteOnClose`
- delete-on-close intent is now tracked on the in-memory handle state even though the read-only client still returns `STATUS_ACCESS_DENIED`
- cleanup/flush state is visible through `GetSecurity(handle)` for diagnostics and validation
- Explorer smoke and request matrix now include an explicit `explorer-delete-denied` scenario

## Boundary

This iteration still keeps the client read-only. The new callbacks do not mutate remote state; they provide deterministic Windows-facing semantics so Explorer and diagnostics can distinguish:

- supported read-only denials
- true missing callback gaps

## Why this matters

Before this iteration, delete-related gestures were only indirectly represented through callback tables and diagnostics. After this iteration:

- delete intent is explicitly modelled
- read-only denial is explicit
- security descriptors expose more realistic Windows metadata
- cleanup semantics are visible across callbacks, bridge, ABI, and service warmup
