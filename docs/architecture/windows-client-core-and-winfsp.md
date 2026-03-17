# Windows Client Core and WinFsp Integration Plan

## Iter 13 outcome

Iter 13 introduces a dedicated `internal/clientcore` package as the runtime-facing protocol client core for the future Windows mount.

The repository previously exposed only `internal/client`, which mixed together:
- RPC framing and request/response decoding
- session lifecycle updates
- metadata/data/watch/recovery protocol calls
- ad-hoc client-side state such as session id

That layout was sufficient for the demo CLI, but it was not a good boundary for a WinFsp adapter, because mount callbacks would have needed to depend directly on a single monolithic client object.

## New layering

```text
Windows Apps (Explorer / VS Code / Git / Terminal)
        |
      WinFsp adapter                <- Iter 14+
        |
   Mount runtime / namespace cache  <- Iter 14+
        |
   internal/clientcore              <- Iter 13
        |
   protocol / transport
        |
   server runtime
```

## `internal/clientcore` responsibilities

`internal/clientcore` is now the durable client-side protocol layer intended to be reused by:
- the demo CLI (`cmd/devmount-client` via the `internal/client` compatibility wrapper)
- a future WinFsp mount process
- future diagnostics and recovery tooling

### Package split

```text
internal/clientcore/
  client.go      -> connection lifecycle, root runtime object, state snapshots
  rpc.go         -> request serialization, response decode, protocol error handling
  session.go     -> hello/auth/create-session/resume/heartbeat
  metadata.go    -> lookup/getattr/opendir/readdir/rename
  data.go        -> open/create/read/write/flush/truncate/delete-on-close/close
  watch.go       -> subscribe/poll/ack/resync tracking
  recovery.go    -> recover handles / revalidate / resubscribe state application
  state.go       -> tracked handles, tracked watches, tracked nodes, recovery snapshot
```

## Runtime state model

The core now maintains a structured `RuntimeState` snapshot instead of only a session id.

Tracked state includes:
- client identity (`client_name`, `client_version`, `client_instance_id`)
- selected session information (`session_id`, lease, expiry, state)
- authenticated principal and granted features
- server handshake result
- tracked open handles
- tracked directory cursors
- tracked watches
- tracked node ids that should be candidates for revalidation
- last heartbeat timestamp / error

This gives the future WinFsp runtime enough information to:
- rebuild recovery requests after reconnect
- export diagnostics without scraping multiple subsystems
- reason about which nodes are likely stale
- keep mount orchestration separate from low-level RPC code

## Recovery-oriented client state

Iter 13 does **not** implement full mount recovery yet, but it adds the state model needed for it.

### Tracked handles

Each successful open/create call records a `TrackedHandle` with:
- remote handle id
- node id
- writable / delete-on-close bits
- last known size
- last write / flush / truncate timestamps

Close removes the tracked handle. Recover-handle responses replace old handle ids with the new server-issued ids.

### Tracked watches

Each subscribe call records:
- watch id
- node id
- recursive bit
- last seen seq
- last acked seq

Poll updates the last seen sequence. Ack updates the durable sequence used for recovery. Resubscribe responses replace old watch ids while preserving node and recursion metadata.

### Tracked nodes

The client core keeps a deduplicated node-id set sourced from:
- lookups/getattr
- opened directories
- readdir results
- open/create/rename results
- watch events and resync snapshots

This becomes the initial input for Iter 14 cache invalidation and Iter 16 revalidate flows.

## Compatibility strategy

The old `internal/client` package remains as a compatibility wrapper:
- `type Client = clientcore.Client`
- `New()` delegates to `clientcore.New()`
- `decodeInto()` delegates to `clientcore.DecodeInto()`

This keeps existing tests and the demo CLI working while allowing new WinFsp-specific work to depend directly on `internal/clientcore`.

## Planned WinFsp boundary

Iter 14 should introduce a dedicated Windows adapter with the following boundaries.

### WinFsp adapter layer

Suggested directory layout:

```text
cmd/devmount-winfsp/
internal/platform/windows/
internal/winfsp/adapter/
internal/mountcore/
```

### Suggested responsibility split

#### `internal/mountcore`
Pure Go, platform-neutral mount runtime orchestration:
- path-to-node cache
- inode/node lookup policy
- directory enumeration cursor lifecycle
- open/create/overwrite decision mapping
- reconnect / heartbeat / recovery controller
- watcher-driven invalidation hooks
- structured diagnostics snapshot for the mounted workspace

#### `internal/winfsp/adapter`
Thin Windows binding layer:
- maps WinFsp callbacks to `mountcore`
- marshals Windows file info structs
- translates Win32/NT status and create dispositions
- owns mount/unmount lifecycle and service registration hooks

#### `internal/platform/windows`
Windows-only utilities:
- path normalization rules specific to Windows
- handle/share-mode translation helpers
- attribute/timestamp translation
- install/runtime prerequisite checks (later productization)

## WinFsp request mapping targets

The first WinFsp MVP should support only the read-only subset:
- `GetVolumeInfo`
- `Open` / `GetFileInfo`
- `OpenDirectory` / `ReadDirectory`
- `Read`
- `Close`

Protocol mapping should be:
- path resolution -> `Lookup` + `GetAttr`
- directory enumeration -> `OpenDir` + repeated `ReadDir`
- file reads -> `OpenRead` + `Read`
- close -> `CloseHandle`

Writes, replacement saves, and delete-pending semantics stay deferred to Iter 15.

## Testing implications

Iter 13 adds unit coverage for:
- request header / error / mismatch behavior in `internal/clientcore`
- session tracking state updates
- recovery snapshot generation
- replacement of recovered handle/watch ids
- compatibility preservation in `internal/client`

The next Windows-facing test layers should be:
1. pure-Go `mountcore` contract tests
2. Windows-only adapter tests behind build tags
3. local manual smoke tests with Explorer / VS Code against a mounted workspace

## Non-goals for Iter 13

This iteration intentionally does **not** implement:
- WinFsp callback wiring
- Windows build tags
- mount lifecycle commands
- local namespace cache eviction policy
- full Windows create/share/delete semantics
- installer / service integration

Those remain deferred so the client-side protocol/runtime boundary can stabilize first.

## Exit criteria for Iter 13

Iter 13 is considered complete when:
- protocol-facing client code is split into `internal/clientcore`
- state snapshots and recovery snapshots exist
- `internal/client` remains compatible for the demo CLI and existing tests
- the WinFsp boundary is documented clearly enough to begin Iter 14 implementation
