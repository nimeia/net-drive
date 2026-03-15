# Iter 3 Test Report

## Scope

This iteration validates the metadata cache and directory snapshot baseline.

Covered areas:
- attr cache hit and TTL refresh
- negative cache expiry
- dir snapshot cache hit and refresh
- root prefetch warm-up
- existing control-plane and read-only integration flow

## Commands

```bash
go build ./...
go test ./...
```

## Result

- `go build ./...` ✅
- `go test ./...` ✅

## Key tests

- `TestMetadataBackendAttrCacheRefreshesAfterTTL`
- `TestMetadataBackendNegativeCacheExpires`
- `TestMetadataBackendDirSnapshotCacheAndRootPrefetch`
- `TestControlPlaneHandshakeAndSession`
