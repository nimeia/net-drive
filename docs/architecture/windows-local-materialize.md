# Iter 17 — Local materialize bridge for Windows testing

## Goal

Add a practical bridge that downloads remote workspace content into a local Windows directory, so the current branch can already be exercised through normal local file workflows even before the real WinFsp host binding is finished.

## What this iteration adds

- `internal/materialize` recursive downloader over `mountcore`
- `cmd/devmount-winfsp -op materialize -path <remote> -local-path <dir>`
- Win32 config test UI support for `materialize`
- local name validation to block path traversal or invalid separator injection
- chunked file download based on existing read protocol

## Why this shape

The current repository already has a stable read-only namespace/data path:

- `clientcore` manages the protocol session
- `mountcore` resolves paths, opens directories, enumerates entries, opens files, and reads bytes
- `winclient` / `devmount-winfsp` expose smoke and test entry points

The missing part for a true Windows mount is the native WinFsp host callback binding. That is still outside this iteration. Instead of waiting on that native layer, this iteration provides a usable "load remote files as local files" path now.

## Behavior

- if `-path` points to a directory, the whole subtree is mirrored into `-local-path`
- if `-path` points to a file, that file is downloaded to `-local-path`
- file reads are chunked; directories are enumerated page by page
- invalid entry names such as `..`, `a/b`, or `a\b` are rejected

## Current boundary

This is **not** yet a kernel-visible mounted volume. It is a local materialization flow intended for:

- Windows smoke testing
- inspecting remote content with Explorer / editors
- comparing remote workspace state against local tool behavior
- validating path/read/read-dir semantics before real WinFsp binding lands

## Suggested next steps

1. add optional cleanup/sync policy for extra local files
2. persist materialize presets in the Win32 test UI
3. add diff/refresh support for repeated pulls into the same directory
4. continue with native WinFsp binding so `-op mount` becomes a real mounted filesystem
