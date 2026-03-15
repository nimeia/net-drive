# Developer Mount Protocol v0.1

## Scope

Protocol v0.1 is the bring-up contract for a Windows-first, editor-optimized remote filesystem.

Currently implemented areas:
- control channel: hello / auth / create-session / resume-session / heartbeat
- metadata channel: lookup / getattr / opendir / readdir / rename
- data channel: open / create / read / write / flush / truncate / set-delete-on-close / close
- events channel: subscribe / poll-events / ack-events / resync-snapshot
- recovery channel: recover-handles / revalidate / resubscribe

Still deferred:
- push-style watcher event streaming
- lease / oplock-style cache negotiation
- full Windows file semantic coverage
- full handle replay semantics

## Transport

Current implementation:
- TCP placeholder transport
- 4-byte big-endian frame length
- 32-byte fixed header
- JSON payload

Planned evolution:
- QUIC transport replacing TCP placeholder without changing higher-level message schema

## Frame Header

The header is always 32 bytes.

| Offset | Size | Field | Type | Description |
|---|---:|---|---|---|
| 0 | 4 | magic | bytes | ASCII `DMNT` |
| 4 | 1 | version | uint8 | protocol version, current `1` |
| 5 | 1 | header_length | uint8 | bytes in header, current `32` |
| 6 | 1 | channel | uint8 | logical channel id |
| 7 | 1 | opcode | uint8 | message opcode |
| 8 | 4 | flags | uint32 | request / response / error bits |
| 12 | 8 | request_id | uint64 | correlates request/response |
| 20 | 8 | session_id | uint64 | `0` until a session exists |
| 28 | 4 | payload_length | uint32 | payload size |

### Channels

| Value | Name |
|---:|---|
| 1 | control |
| 2 | metadata |
| 3 | data |
| 4 | events |
| 5 | recovery |

### Flags

| Bit | Constant |
|---:|---|
| 0 | `FlagRequest` |
| 1 | `FlagResponse` |
| 2 | `FlagError` |
| 3 | `FlagAckRequired` |
| 4 | `FlagReplay` |
| 5 | `FlagCompressed` |

## Implemented Opcodes

### Control

| Opcode | Name |
|---:|---|
| 1 | HelloReq |
| 2 | HelloResp |
| 3 | AuthReq |
| 4 | AuthResp |
| 5 | CreateSessionReq |
| 6 | CreateSessionResp |
| 7 | ResumeSessionReq |
| 8 | ResumeSessionResp |
| 9 | HeartbeatReq |
| 10 | HeartbeatResp |
| 11 | ErrorResp |

### Metadata

| Opcode | Name |
|---:|---|
| 20 | LookupReq |
| 21 | LookupResp |
| 22 | GetAttrReq |
| 23 | GetAttrResp |
| 24 | OpenDirReq |
| 25 | OpenDirResp |
| 26 | ReadDirReq |
| 27 | ReadDirResp |
| 28 | RenameReq |
| 29 | RenameResp |

### Data

| Opcode | Name |
|---:|---|
| 40 | OpenReq |
| 41 | OpenResp |
| 42 | CreateReq |
| 43 | CreateResp |
| 44 | ReadReq |
| 45 | ReadResp |
| 46 | WriteReq |
| 47 | WriteResp |
| 48 | FlushReq |
| 49 | FlushResp |
| 50 | TruncateReq |
| 51 | TruncateResp |
| 52 | SetDeleteOnCloseReq |
| 53 | SetDeleteOnCloseResp |
| 54 | CloseReq |
| 55 | CloseResp |

### Events

| Opcode | Name |
|---:|---|
| 60 | SubscribeReq |
| 61 | SubscribeResp |
| 62 | PollEventsReq |
| 63 | PollEventsResp |
| 64 | AckEventsReq |
| 65 | AckEventsResp |
| 66 | ResyncReq |
| 67 | ResyncResp |

### Recovery

| Opcode | Name |
|---:|---|
| 70 | RecoverHandlesReq |
| 71 | RecoverHandlesResp |
| 72 | RevalidateReq |
| 73 | RevalidateResp |
| 74 | ResubscribeReq |
| 75 | ResubscribeResp |

## Message Schema Highlights

### ResumeSessionReq

```json
{
  "session_id": 1001,
  "client_instance_id": "client-1",
  "last_known_server_time": "2026-03-14T12:00:20Z"
}
```

### OpenReq

```json
{
  "node_id": 42,
  "writable": true,
  "truncate": false
}
```

### CreateReq

```json
{
  "parent_node_id": 1,
  "name": ".tmp-save",
  "overwrite": false
}
```

### SubscribeReq

```json
{
  "node_id": 1,
  "recursive": true
}
```

### RecoverHandlesReq

```json
{
  "handles": [
    {
      "previous_handle_id": 2001,
      "node_id": 7,
      "writable": false,
      "delete_on_close": false
    }
  ]
}
```

### RevalidateReq

```json
{
  "node_ids": [7, 8, 9]
}
```

### ResubscribeReq

```json
{
  "watches": [
    {
      "previous_watch_id": 5001,
      "node_id": 1,
      "recursive": true,
      "after_seq": 12
    }
  ]
}
```

## Session Rules

1. Client must send `HelloReq` first.
2. After `HelloResp`, client may send `AuthReq`.
3. After `AuthResp`, client may send `CreateSessionReq`.
4. `ResumeSessionReq` is implemented and requires matching `session_id` and `client_instance_id`.
5. Metadata, data, events, and recovery channels require an active session.
6. After session creation or resume, the client must keep the session alive with `HeartbeatReq`.

## Error Codes

| Code | Meaning |
|---|---|
| ERR_INVALID_REQUEST | malformed payload or invalid field set |
| ERR_UNSUPPORTED_VERSION | no compatible protocol version |
| ERR_UNSUPPORTED_OPERATION | opcode/schema known but not implemented |
| ERR_AUTH_REQUIRED | authentication missing or invalid |
| ERR_SESSION_EXPIRED | session timed out |
| ERR_SESSION_NOT_FOUND | requested session unknown |
| ERR_NOT_FOUND | missing path / node / handle target |
| ERR_ALREADY_EXISTS | create or rename target already exists |
| ERR_NOT_DIR | target is not a directory |
| ERR_IS_DIR | target is a directory |
| ERR_INVALID_HANDLE | handle is unknown or closed |
| ERR_ACCESS_DENIED | write/open mode denied |
| ERR_WATCH_NOT_FOUND | watch id is unknown |
| ERR_RECOVERY_FAILED | recovery operation could not be completed |
| ERR_INTERNAL | internal server error |
