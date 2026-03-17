# Iter 19 — Windows client shell pages

## Goal

Move the Win32 client from a single testing form to a product-shaped shell with three primary surfaces:

- Dashboard
- Profiles
- Diagnostics

## What changed

### Dashboard
- current runtime phase and status
- current profile summary
- current mount target summary
- quick actions for `Start Mount`, `Stop Mount`, and `Show Mount CLI`

### Profiles
- named profile save/load/delete
- connection and mount defaults
- path/local-path/offset/read/max-entries tuning fields
- restore-defaults action

### Diagnostics
- advanced smoke operations over the current in-memory profile
- runtime summary snapshot
- mount CLI preview
- `volume|getattr|readdir|read|materialize` execution output

## Why this split

The previous single-page window mixed three concerns:

1. profile editing
2. runtime control
3. diagnostics / smoke testing

The page split keeps the existing Win32 form approach, but makes the product direction explicit and gives later iterations a place to attach tray, diagnostics export, and richer mount management.

## Current limitation

This is still a native Win32 shell, not a finished product UI system. Layout is fixed-position and optimized for early iteration, but the navigation and responsibilities now match the productization plan.
