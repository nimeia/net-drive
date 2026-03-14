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

type ErrorCode string

const (
	ErrInvalidRequest     ErrorCode = "ERR_INVALID_REQUEST"
	ErrUnsupportedVersion ErrorCode = "ERR_UNSUPPORTED_VERSION"
	ErrUnsupportedOp      ErrorCode = "ERR_UNSUPPORTED_OPERATION"
	ErrAuthRequired       ErrorCode = "ERR_AUTH_REQUIRED"
	ErrSessionExpired     ErrorCode = "ERR_SESSION_EXPIRED"
	ErrSessionNotFound    ErrorCode = "ERR_SESSION_NOT_FOUND"
	ErrCapabilityMismatch ErrorCode = "ERR_CAPABILITY_MISMATCH"
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
		Channels:  []string{"control"},
		Features:  []string{"auth-basic", "session-create", "heartbeat"},
	}
}

func NowRFC3339(now time.Time) string {
	return now.UTC().Format(time.RFC3339)
}
