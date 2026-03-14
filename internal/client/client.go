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
	Addr      string
	Conn      net.Conn
	nextReqID atomic.Uint64
	SessionID uint64
}

func New(addr string) *Client {
	c := &Client{Addr: addr}
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
		ClientVersion:             "0.1.0",
		SupportedProtocolVersions: []uint8{protocol.Version},
		Capabilities:              protocol.DefaultCapabilities(),
	}
	var resp protocol.HelloResp
	if err := c.request(protocol.OpcodeHelloReq, req, &resp); err != nil {
		return protocol.HelloResp{}, err
	}
	return resp, nil
}

func (c *Client) Auth(token string) (protocol.AuthResp, error) {
	req := protocol.AuthReq{Scheme: "dev-token", Token: token}
	var resp protocol.AuthResp
	if err := c.request(protocol.OpcodeAuthReq, req, &resp); err != nil {
		return protocol.AuthResp{}, err
	}
	return resp, nil
}

func (c *Client) CreateSession(clientInstanceID string, leaseSeconds uint32) (protocol.CreateSessionResp, error) {
	req := protocol.CreateSessionReq{ClientInstanceID: clientInstanceID, RequestedLeaseSeconds: leaseSeconds, MountName: "workspace-dev"}
	var resp protocol.CreateSessionResp
	if err := c.request(protocol.OpcodeCreateSessionReq, req, &resp); err != nil {
		return protocol.CreateSessionResp{}, err
	}
	c.SessionID = resp.SessionID
	return resp, nil
}

func (c *Client) Heartbeat() (protocol.HeartbeatResp, error) {
	req := protocol.HeartbeatReq{SessionID: c.SessionID, ClientTime: protocol.NowRFC3339(time.Now())}
	var resp protocol.HeartbeatResp
	if err := c.request(protocol.OpcodeHeartbeatReq, req, &resp); err != nil {
		return protocol.HeartbeatResp{}, err
	}
	return resp, nil
}

func (c *Client) request(opcode protocol.Opcode, reqPayload any, respPayload any) error {
	if c.Conn == nil {
		return fmt.Errorf("client is not connected")
	}
	requestID := c.nextReqID.Add(1)
	header := protocol.Header{Channel: protocol.ChannelControl, Opcode: opcode, Flags: protocol.FlagRequest, RequestID: requestID, SessionID: c.SessionID}
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
	default:
		return fmt.Errorf("unsupported response target %T", out)
	}
}
