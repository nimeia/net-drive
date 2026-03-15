# Iter 7 Test Report

Generated: 2026-03-15T00:00:00Z

## Covered optimizations

- workspace profile inference
- root hot-dir / hot-file prefetch
- small-file cache hits and invalidation
- priority-aware prefetch scheduling

## Commands

### Focused backend tests

```bash
go test ./internal/server -run 'TestMetadataBackend(AttrCacheRefreshesAfterTTL|NegativeCacheExpires|DirSnapshotCacheAndRootPrefetch|SmallFileCacheAndInvalidation|WorkspaceProfileAndPrefetchPriority)$' -count=1
```

### Full build

```bash
go build ./...
```

### Full test

```bash
go test ./...
```

## Results

- focused backend tests: passed
- `go build ./...`: passed
- `go test ./...`: passed
