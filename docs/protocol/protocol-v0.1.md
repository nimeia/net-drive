# Developer Mount Protocol v0.1

## 1. Scope

Protocol v0.1 defines the early control-plane and read-only data-plane baseline for a Windows-first, editor-optimized remote filesystem.

This revision freezes:
- wire framing
- common identifiers
- header fields
- control-channel message schema
- read-only metadata and data message schema
- error model
- session lifecycle expectations

This revision does **not** yet implement:
- watcher streaming
- recovery replay
- file locking semantics
- write / flush / rename semantics

## 2. Transport

### 2.1 Current implementation

The current Iter 2 code uses a **TCP placeholder transport**.

Wire format:
1. 4-byte big-endian frame length
2. fixed-size binary header
3. payload bytes

Payload encoding in Iter 2:
- UTF-8 JSON

## 3. Frame Header

The header is always 32 bytes.

| Offset | Size | Field | Type | Description |
|---|---:|---|---|---|
| 0 | 4 | magic | bytes | ASCII `DMNT` |
| 4 | 1 | version | uint8 | protocol version, current `1` |
| 5 | 1 | header_length | uint8 | bytes in header, current `32` |
| 6 | 1 | channel | uint8 | logical channel id |
| 7 | 1 | opcode | uint8 | message opcode |
| 8 | 4 | flags | uint32 | request/response/error bits |
| 12 | 8 | request_id | uint64 | correlates request/response |
| 20 | 8 | session_id | uint64 | `0` until a session exists |
| 28 | 4 | payload_length | uint32 | number of payload bytes |

## 4. Channels

| Value | Name | Purpose |
|---:|---|---|
| 1 | control | handshake, auth, session, heartbeat |
| 2 | metadata | lookup, getattr, readdir |
| 3 | data | open, read, close |
| 4 | events | reserved |
| 5 | recovery | reserved |

## 5. Read-only Iter 2 opcodes

### Metadata channel
- `LookupReq / LookupResp`
- `GetAttrReq / GetAttrResp`
- `OpenDirReq / OpenDirResp`
- `ReadDirReq / ReadDirResp`

### Data channel
- `OpenReq / OpenResp`
- `ReadReq / ReadResp`
- `CloseReq / CloseResp`

## 6. Node model

### NodeInfo

```json
{
  "node_id": 2,
  "parent_node_id": 1,
  "name": "hello.txt",
  "file_type": "file",
  "size": 11,
  "mode": 420,
  "mod_time": "2026-03-15T00:00:00Z"
}
```

### LookupReq

```json
{
  "parent_node_id": 1,
  "name": "hello.txt"
}
```

### LookupResp

```json
{
  "entry": {
    "node_id": 2,
    "parent_node_id": 1,
    "name": "hello.txt",
    "file_type": "file",
    "size": 11,
    "mode": 420,
    "mod_time": "2026-03-15T00:00:00Z"
  }
}
```

### GetAttrReq

```json
{
  "node_id": 1
}
```

### OpenDirReq

```json
{
  "node_id": 1
}
```

### OpenDirResp

```json
{
  "dir_cursor_id": 1001
}
```

### ReadDirReq

```json
{
  "dir_cursor_id": 1001,
  "cookie": 0,
  "max_entries": 64
}
```

### ReadDirResp

```json
{
  "entries": [
    {
      "node_id": 2,
      "name": "hello.txt",
      "file_type": "file",
      "size": 11,
      "mode": 420,
      "mod_time": "2026-03-15T00:00:00Z"
    }
  ],
  "next_cookie": 1,
  "eof": false
}
```

### OpenReq

```json
{
  "node_id": 2
}
```

### OpenResp

```json
{
  "handle_id": 2001,
  "size": 11
}
```

### ReadReq

```json
{
  "handle_id": 2001,
  "offset": 0,
  "length": 5
}
```

### ReadResp

```json
{
  "data": "aGVsbG8=",
  "eof": false,
  "offset": 0
}
```

### CloseReq

```json
{
  "handle_id": 2001
}
```

### CloseResp

```json
{
  "closed": true
}
```

## 7. Error Codes

| Code | Meaning |
|---|---|
| ERR_INVALID_REQUEST | malformed payload or invalid field set |
| ERR_UNSUPPORTED_VERSION | no compatible protocol version |
| ERR_UNSUPPORTED_OPERATION | opcode/schema known but not implemented |
| ERR_AUTH_REQUIRED | authentication missing or invalid |
| ERR_SESSION_EXPIRED | session timed out |
| ERR_SESSION_NOT_FOUND | requested session unknown |
| ERR_NOT_FOUND | node / cursor / handle not found |
| ERR_NOT_DIR | opendir on non-directory |
| ERR_IS_DIR | open-file on directory |
| ERR_INVALID_HANDLE | invalid read/close handle |
| ERR_INTERNAL | internal server error |
