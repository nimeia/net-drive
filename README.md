# Developer Mount

Developer Mount is a Windows-first, editor-optimized remote filesystem project.

This repository currently contains:
- protocol and architecture documents for Iter 0
- a control-plane baseline for Iter 1
- a TCP placeholder transport that can later be replaced by QUIC
- a minimal client/server demo and tests

## Current scope

The current code implements only the control channel baseline:
- hello / capability negotiation
- auth
- create session
- heartbeat

It does not yet implement:
- WinFsp integration
- metadata operations
- file read/write operations
- watcher streaming
- recovery replay

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
go run ./cmd/devmount-server
```

In another terminal, run the demo client:

```bash
go run ./cmd/devmount-client
```
