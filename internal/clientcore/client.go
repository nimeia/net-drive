package clientcore

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultClientName    = "devmount-client"
	defaultClientVersion = "0.6.0"
	defaultMountName     = "workspace-dev"
)

type Client struct {
	Addr       string
	Conn       net.Conn
	Dial       func(network, addr string) (net.Conn, error)
	SessionID  uint64
	RootNodeID uint64

	reqMu     sync.Mutex
	nextReqID atomic.Uint64

	stateMu sync.RWMutex
	state   RuntimeState
}

type RuntimeState struct {
	ClientName              string
	ClientVersion           string
	ClientInstanceID        string
	MountName               string
	LeaseSeconds            uint32
	SessionExpiresAt        string
	SessionState            string
	PrincipalID             string
	DisplayName             string
	GrantedFeatures         []string
	ServerName              string
	ServerVersion           string
	SelectedProtocolVersion uint8
	ServerCapabilities      []string
	TrackedHandles          map[uint64]TrackedHandle
	TrackedDirCursors       map[uint64]TrackedDirCursor
	TrackedWatches          map[uint64]TrackedWatch
	TrackedNodes            map[uint64]struct{}
	LastHeartbeatAt         time.Time
	LastHeartbeatError      string
}

func New(addr string) *Client {
	c := &Client{Addr: addr, RootNodeID: 1}
	c.nextReqID.Store(1)
	c.state = RuntimeState{
		ClientName:        defaultClientName,
		ClientVersion:     defaultClientVersion,
		MountName:         defaultMountName,
		TrackedHandles:    map[uint64]TrackedHandle{},
		TrackedDirCursors: map[uint64]TrackedDirCursor{},
		TrackedWatches:    map[uint64]TrackedWatch{},
		TrackedNodes:      map[uint64]struct{}{1: {}},
	}
	return c
}

func (c *Client) Connect() error {
	dial := c.Dial
	if dial == nil {
		dial = net.Dial
	}
	conn, err := dial("tcp", c.Addr)
	if err != nil {
		return err
	}
	c.Conn = conn
	return nil
}

func (c *Client) Close() error {
	if c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}

func (c *Client) SnapshotState() RuntimeState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state.clone()
}

func (c *Client) SnapshotRecoveryState() RecoverySnapshot {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state.recoverySnapshot(c.SessionID)
}
