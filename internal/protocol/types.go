package protocol

import "time"

const (
	Magic              = "DMNT"
	Version      uint8 = 1
	HeaderLength       = 32
)

type Channel uint8

const (
	ChannelControl  Channel = 1
	ChannelMetadata Channel = 2
	ChannelData     Channel = 3
	ChannelEvents   Channel = 4
	ChannelRecovery Channel = 5
)

type Opcode uint8

const (
	OpcodeHelloReq          Opcode = 1
	OpcodeHelloResp         Opcode = 2
	OpcodeAuthReq           Opcode = 3
	OpcodeAuthResp          Opcode = 4
	OpcodeCreateSessionReq  Opcode = 5
	OpcodeCreateSessionResp Opcode = 6
	OpcodeResumeSessionReq  Opcode = 7
	OpcodeResumeSessionResp Opcode = 8
	OpcodeHeartbeatReq      Opcode = 9
	OpcodeHeartbeatResp     Opcode = 10
	OpcodeErrorResp         Opcode = 11

	OpcodeLookupReq   Opcode = 20
	OpcodeLookupResp  Opcode = 21
	OpcodeGetAttrReq  Opcode = 22
	OpcodeGetAttrResp Opcode = 23
	OpcodeOpenDirReq  Opcode = 24
	OpcodeOpenDirResp Opcode = 25
	OpcodeReadDirReq  Opcode = 26
	OpcodeReadDirResp Opcode = 27
	OpcodeOpenReq     Opcode = 28
	OpcodeOpenResp    Opcode = 29
	OpcodeReadReq     Opcode = 30
	OpcodeReadResp    Opcode = 31
	OpcodeCloseReq    Opcode = 32
	OpcodeCloseResp   Opcode = 33
)

const (
	FlagRequest uint32 = 1 << iota
	FlagResponse
	FlagError
	FlagAckRequired
	FlagReplay
	FlagCompressed
)

type Header struct {
	Magic         [4]byte
	Version       uint8
	HeaderLength  uint8
	Channel       Channel
	Opcode        Opcode
	Flags         uint32
	RequestID     uint64
	SessionID     uint64
	PayloadLength uint32
}

type CapabilitySet struct {
	Transport []string `json:"transport"`
	Channels  []string `json:"channels"`
	Features  []string `json:"features"`
}

type HelloReq struct {
	ClientName                string        `json:"client_name"`
	ClientVersion             string        `json:"client_version"`
	SupportedProtocolVersions []uint8       `json:"supported_protocol_versions"`
	Capabilities              CapabilitySet `json:"capabilities"`
}

type HelloResp struct {
	ServerName              string        `json:"server_name"`
	ServerVersion           string        `json:"server_version"`
	SelectedProtocolVersion uint8         `json:"selected_protocol_version"`
	ServerTime              string        `json:"server_time"`
	Capabilities            CapabilitySet `json:"capabilities"`
}

type AuthReq struct {
	Scheme string `json:"scheme"`
	Token  string `json:"token"`
}

type AuthResp struct {
	Authenticated  bool     `json:"authenticated"`
	PrincipalID    string   `json:"principal_id,omitempty"`
	DisplayName    string   `json:"display_name,omitempty"`
	GrantedFeature []string `json:"granted_features"`
}

type CreateSessionReq struct {
	ClientInstanceID      string `json:"client_instance_id"`
	RequestedLeaseSeconds uint32 `json:"requested_lease_seconds"`
	MountName             string `json:"mount_name,omitempty"`
}

type CreateSessionResp struct {
	SessionID    uint64 `json:"session_id"`
	LeaseSeconds uint32 `json:"lease_seconds"`
	ExpiresAt    string `json:"expires_at"`
	State        string `json:"state"`
}

type ResumeSessionReq struct {
	SessionID           uint64 `json:"session_id"`
	ClientInstanceID    string `json:"client_instance_id"`
	LastKnownServerTime string `json:"last_known_server_time,omitempty"`
}

type ResumeSessionResp struct {
	SessionID    uint64 `json:"session_id"`
	LeaseSeconds uint32 `json:"lease_seconds"`
	ExpiresAt    string `json:"expires_at"`
	State        string `json:"state"`
}

type HeartbeatReq struct {
	SessionID  uint64 `json:"session_id"`
	ClientTime string `json:"client_time"`
}

type HeartbeatResp struct {
	SessionID  uint64 `json:"session_id"`
	ServerTime string `json:"server_time"`
	ExpiresAt  string `json:"expires_at"`
	State      string `json:"state"`
}

type FileType string

const (
	FileTypeFile      FileType = "file"
	FileTypeDirectory FileType = "directory"
)

type NodeInfo struct {
	NodeID       uint64   `json:"node_id"`
	ParentNodeID uint64   `json:"parent_node_id"`
	Name         string   `json:"name"`
	FileType     FileType `json:"file_type"`
	Size         int64    `json:"size"`
	Mode         uint32   `json:"mode"`
	ModTime      string   `json:"mod_time"`
}

type LookupReq struct {
	ParentNodeID uint64 `json:"parent_node_id"`
	Name         string `json:"name"`
}

type LookupResp struct {
	Entry NodeInfo `json:"entry"`
}

type GetAttrReq struct {
	NodeID uint64 `json:"node_id"`
}

type GetAttrResp struct {
	Entry NodeInfo `json:"entry"`
}

type OpenDirReq struct {
	NodeID uint64 `json:"node_id"`
}

type OpenDirResp struct {
	DirCursorID uint64 `json:"dir_cursor_id"`
}

type DirEntry struct {
	NodeID   uint64   `json:"node_id"`
	Name     string   `json:"name"`
	FileType FileType `json:"file_type"`
	Size     int64    `json:"size"`
	Mode     uint32   `json:"mode"`
	ModTime  string   `json:"mod_time"`
}

type ReadDirReq struct {
	DirCursorID uint64 `json:"dir_cursor_id"`
	Cookie      uint64 `json:"cookie"`
	MaxEntries  uint32 `json:"max_entries"`
}

type ReadDirResp struct {
	Entries    []DirEntry `json:"entries"`
	NextCookie uint64     `json:"next_cookie"`
	EOF        bool       `json:"eof"`
}

type OpenReq struct {
	NodeID uint64 `json:"node_id"`
}

type OpenResp struct {
	HandleID uint64 `json:"handle_id"`
	Size     int64  `json:"size"`
}

type ReadReq struct {
	HandleID uint64 `json:"handle_id"`
	Offset   int64  `json:"offset"`
	Length   uint32 `json:"length"`
}

type ReadResp struct {
	Data   []byte `json:"data"`
	EOF    bool   `json:"eof"`
	Offset int64  `json:"offset"`
}

type CloseReq struct {
	HandleID uint64 `json:"handle_id"`
}

type CloseResp struct {
	Closed bool `json:"closed"`
}

type ErrorCode string

const (
	ErrInvalidRequest     ErrorCode = "ERR_INVALID_REQUEST"
	ErrUnsupportedVersion ErrorCode = "ERR_UNSUPPORTED_VERSION"
	ErrUnsupportedOp      ErrorCode = "ERR_UNSUPPORTED_OPERATION"
	ErrAuthRequired       ErrorCode = "ERR_AUTH_REQUIRED"
	ErrSessionExpired     ErrorCode = "ERR_SESSION_EXPIRED"
	ErrSessionNotFound    ErrorCode = "ERR_SESSION_NOT_FOUND"
	ErrCapabilityMismatch ErrorCode = "ERR_CAPABILITY_MISMATCH"
	ErrNotFound           ErrorCode = "ERR_NOT_FOUND"
	ErrNotDir             ErrorCode = "ERR_NOT_DIR"
	ErrIsDir              ErrorCode = "ERR_IS_DIR"
	ErrInvalidHandle      ErrorCode = "ERR_INVALID_HANDLE"
	ErrAccessDenied       ErrorCode = "ERR_ACCESS_DENIED"
	ErrInternal           ErrorCode = "ERR_INTERNAL"
)

type ErrorResp struct {
	Code      ErrorCode      `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

func DefaultCapabilities() CapabilitySet {
	return CapabilitySet{
		Transport: []string{"tcp-json"},
		Channels:  []string{"control", "metadata", "data"},
		Features:  []string{"auth-basic", "session-create", "heartbeat", "lookup", "getattr", "readdir", "read-only-open"},
	}
}

func NowRFC3339(now time.Time) string {
	return now.UTC().Format(time.RFC3339)
}
