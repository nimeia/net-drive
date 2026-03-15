# Iter 2 Test Report

## Commands

```bash
go build ./...
go test ./...
```

## Result

- `go build ./...` ✅
- `go test ./...` ✅

## Scope validated

- control-plane handshake (`Hello / Auth / CreateSession / Heartbeat`)
- read-only metadata path (`GetAttr / Lookup / OpenDir / ReadDir`)
- read-only data path (`Open / Read / Close`)
- server local directory backend
- end-to-end integration over TCP placeholder transport
