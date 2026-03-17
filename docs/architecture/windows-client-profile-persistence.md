# Iter 18 — Win32 client profile persistence baseline

## Goal

Move the Win32 client from a disposable test form toward a reusable client console by adding persisted configuration profiles.

## Delivered scope

- new `internal/winclientstore` package
- JSON-backed profile store under the user config directory
- named profile save / load / delete flows
- restore last active profile on next startup
- cross-platform unit tests for the profile store
- Win32 client console updates for profile management

## Why this iteration first

The current Win32 UI is still a test harness. Before investing in a richer Dashboard / Mounts / Diagnostics shell, the client needs a durable configuration model that survives restarts and supports more than one target environment.

This iteration intentionally keeps the data model simple:

- schema version
- active profile name
- profile map of normalized `winclient.Config` values

That is enough to support:

- switching between multiple servers or mount presets
- restoring the most recent working setup
- growing into future settings pages and secure credential storage

## Current boundary

This iteration does **not** yet provide:

- DPAPI / Credential Manager token protection
- tray integration
- async execution and cancellation
- real WinFsp mount lifecycle management
- Dashboard / Mounts / Diagnostics page separation

## Suggested next steps

1. split the current console into Dashboard / Profiles / Diagnostics sections
2. move token storage to DPAPI or Windows Credential Manager
3. add recent run status and profile health snapshots
4. connect profiles to future real mount runtime instances
