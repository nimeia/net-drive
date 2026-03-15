# Developer Mount

Developer Mount is a Windows-first, editor-optimized remote filesystem project.

This repository currently contains:
- protocol and architecture documents for Iter 0
- a control-plane baseline for Iter 1
- an Iter 2 read-only metadata and file access baseline
- an Iter 3 metadata cache and directory snapshot baseline
- a TCP placeholder transport that can later be replaced by QUIC
- minimal client/server demos and tests

## Current scope

The current code implements:
- hello / capability negotiation
- auth
- create session
- heartbeat
- lookup
- getattr
- opendir / readdir
- read-only open / read / close
- metadata attr cache / negative cache / dir snapshot cache
- root prefetch for initial directory snapshot warm-up

It does not yet implement:
- WinFsp integration
- write / flush / rename / replace
- watcher streaming
- recovery replay
- writeback cache

## Repository layout

```text
cmd/
  devmount-server/
  devmount-client/
docs/
  architecture/
  protocol/
internal/
  client/
  protocol/
  server/
  transport/
tests/
  integration/
Task.md
```

## Build

```bash
go build ./...
```

## Test

```bash
go test ./...
```

## Demo

Start the server and expose a local directory:

```bash
go run ./cmd/devmount-server -root /path/to/workspace
```

In another terminal, run the demo client:

```bash
go run ./cmd/devmount-client
```
