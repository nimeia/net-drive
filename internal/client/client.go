package client

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"developer-mount/internal/protocol"
	"developer-mount/internal/transport"
)

type Client struct {
	Addr       string
	Conn       net.Conn
	nextReqID  atomic.Uint64
	SessionID  uint64
	RootNodeID uint64
}

func New(addr string) *Client {
	c := &Client{Addr: addr, RootNodeID: 1}
	c.nextReqID.Store(1)
	return c
}

func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", c.Addr)
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

func (c *Client) Hello() (protocol.HelloResp, error) {
	req := protocol.HelloReq{
		ClientName:                "devmount-client",
		ClientVersion:             "0.2.0",
		SupportedProtocolVersions: []uint8{protocol.Version},
		Capabilities:              protocol.DefaultCapabilities(),
	}
	var resp protocol.HelloResp
	if err := c.request(protocol.ChannelControl, protocol.OpcodeHelloReq, req, &resp); err != nil {
		return protocol.HelloResp{}, err
	}
	return resp, nil
}

func (c *Client) Auth(token string) (protocol.AuthResp, error) {
	req := protocol.AuthReq{Scheme: "dev-token", Token: token}
	var resp protocol.AuthResp
	if err := c.request(protocol.ChannelControl, protocol.OpcodeAuthReq, req, &resp); err != nil {
		return protocol.AuthResp{}, err
	}
	return resp, nil
}

func (c *Client) CreateSession(clientInstanceID string, leaseSeconds uint32) (protocol.CreateSessionResp, error) {
	req := protocol.CreateSessionReq{ClientInstanceID: clientInstanceID, RequestedLeaseSeconds: leaseSeconds, MountName: "workspace-dev"}
	var resp protocol.CreateSessionResp
	if err := c.request(protocol.ChannelControl, protocol.OpcodeCreateSessionReq, req, &resp); err != nil {
		return protocol.CreateSessionResp{}, err
	}
	c.SessionID = resp.SessionID
	return resp, nil
}

func (c *Client) Heartbeat() (protocol.HeartbeatResp, error) {
	req := protocol.HeartbeatReq{SessionID: c.SessionID, ClientTime: protocol.NowRFC3339(time.Now())}
	var resp protocol.HeartbeatResp
	if err := c.request(protocol.ChannelControl, protocol.OpcodeHeartbeatReq, req, &resp); err != nil {
		return protocol.HeartbeatResp{}, err
	}
	return resp, nil
}

func (c *Client) Lookup(parentNodeID uint64, name string) (protocol.LookupResp, error) {
	var resp protocol.LookupResp
	if err := c.request(protocol.ChannelMetadata, protocol.OpcodeLookupReq, protocol.LookupReq{ParentNodeID: parentNodeID, Name: name}, &resp); err != nil {
		return protocol.LookupResp{}, err
	}
	return resp, nil
}

func (c *Client) GetAttr(nodeID uint64) (protocol.GetAttrResp, error) {
	var resp protocol.GetAttrResp
	if err := c.request(protocol.ChannelMetadata, protocol.OpcodeGetAttrReq, protocol.GetAttrReq{NodeID: nodeID}, &resp); err != nil {
		return protocol.GetAttrResp{}, err
	}
	return resp, nil
}

func (c *Client) OpenDir(nodeID uint64) (protocol.OpenDirResp, error) {
	var resp protocol.OpenDirResp
	if err := c.request(protocol.ChannelMetadata, protocol.OpcodeOpenDirReq, protocol.OpenDirReq{NodeID: nodeID}, &resp); err != nil {
		return protocol.OpenDirResp{}, err
	}
	return resp, nil
}

func (c *Client) ReadDir(dirCursorID uint64, cookie uint64, maxEntries uint32) (protocol.ReadDirResp, error) {
	var resp protocol.ReadDirResp
	if err := c.request(protocol.ChannelMetadata, protocol.OpcodeReadDirReq, protocol.ReadDirReq{DirCursorID: dirCursorID, Cookie: cookie, MaxEntries: maxEntries}, &resp); err != nil {
		return protocol.ReadDirResp{}, err
	}
	return resp, nil
}

func (c *Client) Open(nodeID uint64) (protocol.OpenResp, error) {
	var resp protocol.OpenResp
	if err := c.request(protocol.ChannelData, protocol.OpcodeOpenReq, protocol.OpenReq{NodeID: nodeID}, &resp); err != nil {
		return protocol.OpenResp{}, err
	}
	return resp, nil
}

func (c *Client) Read(handleID uint64, offset int64, length uint32) (protocol.ReadResp, error) {
	var resp protocol.ReadResp
	if err := c.request(protocol.ChannelData, protocol.OpcodeReadReq, protocol.ReadReq{HandleID: handleID, Offset: offset, Length: length}, &resp); err != nil {
		return protocol.ReadResp{}, err
	}
	return resp, nil
}

func (c *Client) CloseHandle(handleID uint64) (protocol.CloseResp, error) {
	var resp protocol.CloseResp
	if err := c.request(protocol.ChannelData, protocol.OpcodeCloseReq, protocol.CloseReq{HandleID: handleID}, &resp); err != nil {
		return protocol.CloseResp{}, err
	}
	return resp, nil
}

func (c *Client) request(channel protocol.Channel, opcode protocol.Opcode, reqPayload any, respPayload any) error {
	if c.Conn == nil {
		return fmt.Errorf("client is not connected")
	}
	requestID := c.nextReqID.Add(1)
	header := protocol.Header{Channel: channel, Opcode: opcode, Flags: protocol.FlagRequest, RequestID: requestID, SessionID: c.SessionID}
	frame, err := transport.EncodeFrame(header, reqPayload)
	if err != nil {
		return err
	}
	if _, err := c.Conn.Write(frame); err != nil {
		return err
	}
	respHeader, payload, err := transport.DecodeFrame(c.Conn)
	if err != nil {
		return err
	}
	if respHeader.RequestID != requestID {
		return fmt.Errorf("request id mismatch: want=%d got=%d", requestID, respHeader.RequestID)
	}
	if respHeader.Flags&protocol.FlagError != 0 {
		errPayload, decodeErr := transport.DecodePayload[protocol.ErrorResp](payload)
		if decodeErr != nil {
			return fmt.Errorf("decode error payload: %w", decodeErr)
		}
		return fmt.Errorf("%s: %s", errPayload.Code, errPayload.Message)
	}
	return decodeInto(payload, respPayload)
}

func decodeInto(payload []byte, out any) error {
	switch target := out.(type) {
	case *protocol.HelloResp:
		resp, err := transport.DecodePayload[protocol.HelloResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.AuthResp:
		resp, err := transport.DecodePayload[protocol.AuthResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.CreateSessionResp:
		resp, err := transport.DecodePayload[protocol.CreateSessionResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.HeartbeatResp:
		resp, err := transport.DecodePayload[protocol.HeartbeatResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.LookupResp:
		resp, err := transport.DecodePayload[protocol.LookupResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.GetAttrResp:
		resp, err := transport.DecodePayload[protocol.GetAttrResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.OpenDirResp:
		resp, err := transport.DecodePayload[protocol.OpenDirResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.ReadDirResp:
		resp, err := transport.DecodePayload[protocol.ReadDirResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.OpenResp:
		resp, err := transport.DecodePayload[protocol.OpenResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.ReadResp:
		resp, err := transport.DecodePayload[protocol.ReadResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.CloseResp:
		resp, err := transport.DecodePayload[protocol.CloseResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	default:
		return fmt.Errorf("unsupported response target %T", out)
	}
}
