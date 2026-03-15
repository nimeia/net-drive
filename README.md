# Developer Mount

Developer Mount is a Windows-first, editor-optimized remote filesystem project.

This repository currently contains:
- protocol and architecture documents for Iter 0
- a control-plane baseline for Iter 1
- read-only metadata and file access from Iter 2
- metadata cache and directory snapshot logic from Iter 3
- write/save-path baseline from Iter 4
- watcher and journal polling baseline from Iter 5
- a TCP placeholder transport that can later be replaced by QUIC
- a minimal client/server demo and tests

## Current scope

The current code implements:
- hello / capability negotiation
- auth
- create session
- heartbeat
- lookup / getattr / opendir / readdir
- open(read) / read / close
- create / open(write) / write / flush / truncate / close
- rename / replace-existing
- delete-on-close
- metadata cache / negative cache / dir snapshot cache / root prefetch
- subscribe / poll events / ack / resync snapshot
- journal-backed event ordering with bounded retention

It does not yet implement:
- WinFsp integration
- push-style watcher streaming
- handle/session recovery replay
- lease / oplock-style invalidation
- full Windows file semantic coverage

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

Start the server:

```bash
go run ./cmd/devmount-server -root /path/to/workspace
```

In another terminal, run the demo client:

```bash
go run ./cmd/devmount-client
```
