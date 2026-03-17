# WinFsp Read-only MVP (Iter 14)

Iter 14 turns the Iter 13 client/runtime refactor into a real mount-facing read-only path.

## Delivered boundaries

```text
cmd/devmount-winfsp/
internal/platform/windows/
internal/mountcore/
internal/winfsp/adapter/
```

### `internal/platform/windows`
- normalize incoming WinFsp-style paths to rooted slash paths
- reject parent traversal outside the mount root
- split and join cached mount paths deterministically

### `internal/mountcore`
- path -> node resolution over `Lookup`
- cache of resolved paths
- `GetAttr` refresh by node id
- local file-handle table wrapping remote read handles
- local directory-handle table wrapping remote dir cursors
- read-only open / read / close flow
- directory enumeration flow with child-path cache population
- diagnostics snapshot of cached paths and open handles

### `internal/winfsp/adapter`
- `GetVolumeInfo`
- `GetFileInfo`
- `Open`
- `OpenDirectory`
- `ReadDirectory`
- `Read`
- `Close`

## Request mapping now implemented
- path lookup -> `Lookup`
- attr refresh -> `GetAttr`
- open directory -> `OpenDir`
- read directory -> `ReadDir`
- open file for read -> `OpenRead`
- read file -> `Read`
- close file -> `CloseHandle`
- close directory -> local handle release only
