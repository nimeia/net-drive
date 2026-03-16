package client

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"developer-mount/internal/protocol"
	"developer-mount/internal/transport"
)

type Client struct {
	Addr       string
	Conn       net.Conn
	reqMu      sync.Mutex
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
	var resp protocol.HelloResp
	if err := c.request(protocol.ChannelControl, protocol.OpcodeHelloReq, protocol.HelloReq{
		ClientName:                "devmount-client",
		ClientVersion:             "0.5.0",
		SupportedProtocolVersions: []uint8{protocol.Version},
		Capabilities:              protocol.DefaultCapabilities(),
	}, &resp); err != nil {
		return protocol.HelloResp{}, err
	}
	return resp, nil
}

func (c *Client) Auth(token string) (protocol.AuthResp, error) {
	var resp protocol.AuthResp
	if err := c.request(protocol.ChannelControl, protocol.OpcodeAuthReq, protocol.AuthReq{Scheme: "dev-token", Token: token}, &resp); err != nil {
		return protocol.AuthResp{}, err
	}
	return resp, nil
}

func (c *Client) CreateSession(clientInstanceID string, leaseSeconds uint32) (protocol.CreateSessionResp, error) {
	var resp protocol.CreateSessionResp
	if err := c.request(protocol.ChannelControl, protocol.OpcodeCreateSessionReq, protocol.CreateSessionReq{ClientInstanceID: clientInstanceID, RequestedLeaseSeconds: leaseSeconds, MountName: "workspace-dev"}, &resp); err != nil {
		return protocol.CreateSessionResp{}, err
	}
	c.SessionID = resp.SessionID
	return resp, nil
}

func (c *Client) ResumeSession(sessionID uint64, clientInstanceID string) (protocol.ResumeSessionResp, error) {
	prev := c.SessionID
	c.SessionID = sessionID
	var resp protocol.ResumeSessionResp
	if err := c.request(protocol.ChannelControl, protocol.OpcodeResumeSessionReq, protocol.ResumeSessionReq{SessionID: sessionID, ClientInstanceID: clientInstanceID, LastKnownServerTime: protocol.NowRFC3339(time.Now())}, &resp); err != nil {
		c.SessionID = prev
		return protocol.ResumeSessionResp{}, err
	}
	c.SessionID = resp.SessionID
	return resp, nil
}

func (c *Client) Heartbeat() (protocol.HeartbeatResp, error) {
	var resp protocol.HeartbeatResp
	if err := c.request(protocol.ChannelControl, protocol.OpcodeHeartbeatReq, protocol.HeartbeatReq{SessionID: c.SessionID, ClientTime: protocol.NowRFC3339(time.Now())}, &resp); err != nil {
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

func (c *Client) Rename(srcParentNodeID uint64, srcName string, dstParentNodeID uint64, dstName string, replace bool) (protocol.RenameResp, error) {
	var resp protocol.RenameResp
	if err := c.request(protocol.ChannelMetadata, protocol.OpcodeRenameReq, protocol.RenameReq{SrcParentNodeID: srcParentNodeID, SrcName: srcName, DstParentNodeID: dstParentNodeID, DstName: dstName, ReplaceExisting: replace}, &resp); err != nil {
		return protocol.RenameResp{}, err
	}
	return resp, nil
}

func (c *Client) OpenRead(nodeID uint64) (protocol.OpenResp, error) {
	return c.open(nodeID, false, false)
}
func (c *Client) OpenWrite(nodeID uint64, truncate bool) (protocol.OpenResp, error) {
	return c.open(nodeID, true, truncate)
}

func (c *Client) open(nodeID uint64, writable, truncate bool) (protocol.OpenResp, error) {
	var resp protocol.OpenResp
	if err := c.request(protocol.ChannelData, protocol.OpcodeOpenReq, protocol.OpenReq{NodeID: nodeID, Writable: writable, Truncate: truncate}, &resp); err != nil {
		return protocol.OpenResp{}, err
	}
	return resp, nil
}

func (c *Client) Create(parentNodeID uint64, name string, overwrite bool) (protocol.CreateResp, error) {
	var resp protocol.CreateResp
	if err := c.request(protocol.ChannelData, protocol.OpcodeCreateReq, protocol.CreateReq{ParentNodeID: parentNodeID, Name: name, Overwrite: overwrite}, &resp); err != nil {
		return protocol.CreateResp{}, err
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

func (c *Client) Write(handleID uint64, offset int64, data []byte) (protocol.WriteResp, error) {
	var resp protocol.WriteResp
	if err := c.request(protocol.ChannelData, protocol.OpcodeWriteReq, protocol.WriteReq{HandleID: handleID, Offset: offset, Data: data}, &resp); err != nil {
		return protocol.WriteResp{}, err
	}
	return resp, nil
}

func (c *Client) Flush(handleID uint64) (protocol.FlushResp, error) {
	var resp protocol.FlushResp
	if err := c.request(protocol.ChannelData, protocol.OpcodeFlushReq, protocol.FlushReq{HandleID: handleID}, &resp); err != nil {
		return protocol.FlushResp{}, err
	}
	return resp, nil
}

func (c *Client) Truncate(handleID uint64, size int64) (protocol.TruncateResp, error) {
	var resp protocol.TruncateResp
	if err := c.request(protocol.ChannelData, protocol.OpcodeTruncateReq, protocol.TruncateReq{HandleID: handleID, Size: size}, &resp); err != nil {
		return protocol.TruncateResp{}, err
	}
	return resp, nil
}

func (c *Client) SetDeleteOnClose(handleID uint64, enabled bool) (protocol.SetDeleteOnCloseResp, error) {
	var resp protocol.SetDeleteOnCloseResp
	if err := c.request(protocol.ChannelData, protocol.OpcodeSetDeleteOnCloseReq, protocol.SetDeleteOnCloseReq{HandleID: handleID, DeleteOnClose: enabled}, &resp); err != nil {
		return protocol.SetDeleteOnCloseResp{}, err
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

func (c *Client) Subscribe(nodeID uint64, recursive bool) (protocol.SubscribeResp, error) {
	var resp protocol.SubscribeResp
	if err := c.request(protocol.ChannelEvents, protocol.OpcodeSubscribeReq, protocol.SubscribeReq{NodeID: nodeID, Recursive: recursive}, &resp); err != nil {
		return protocol.SubscribeResp{}, err
	}
	return resp, nil
}

func (c *Client) PollEvents(watchID, afterSeq uint64, maxEvents uint32) (protocol.PollEventsResp, error) {
	var resp protocol.PollEventsResp
	if err := c.request(protocol.ChannelEvents, protocol.OpcodePollEventsReq, protocol.PollEventsReq{WatchID: watchID, AfterSeq: afterSeq, MaxEvents: maxEvents}, &resp); err != nil {
		return protocol.PollEventsResp{}, err
	}
	return resp, nil
}

func (c *Client) AckEvents(watchID, eventSeq uint64) (protocol.AckEventsResp, error) {
	var resp protocol.AckEventsResp
	if err := c.request(protocol.ChannelEvents, protocol.OpcodeAckEventsReq, protocol.AckEventsReq{WatchID: watchID, EventSeq: eventSeq}, &resp); err != nil {
		return protocol.AckEventsResp{}, err
	}
	return resp, nil
}

func (c *Client) Resync(watchID uint64) (protocol.ResyncResp, error) {
	var resp protocol.ResyncResp
	if err := c.request(protocol.ChannelEvents, protocol.OpcodeResyncReq, protocol.ResyncReq{WatchID: watchID}, &resp); err != nil {
		return protocol.ResyncResp{}, err
	}
	return resp, nil
}

func (c *Client) RecoverHandles(handles []protocol.RecoverHandleSpec) (protocol.RecoverHandlesResp, error) {
	var resp protocol.RecoverHandlesResp
	if err := c.request(protocol.ChannelRecovery, protocol.OpcodeRecoverHandlesReq, protocol.RecoverHandlesReq{Handles: handles}, &resp); err != nil {
		return protocol.RecoverHandlesResp{}, err
	}
	return resp, nil
}

func (c *Client) RevalidateNodes(nodeIDs []uint64) (protocol.RevalidateResp, error) {
	var resp protocol.RevalidateResp
	if err := c.request(protocol.ChannelRecovery, protocol.OpcodeRevalidateReq, protocol.RevalidateReq{NodeIDs: nodeIDs}, &resp); err != nil {
		return protocol.RevalidateResp{}, err
	}
	return resp, nil
}

func (c *Client) ResubscribeWatches(watches []protocol.ResubscribeSpec) (protocol.ResubscribeResp, error) {
	var resp protocol.ResubscribeResp
	if err := c.request(protocol.ChannelRecovery, protocol.OpcodeResubscribeReq, protocol.ResubscribeReq{Watches: watches}, &resp); err != nil {
		return protocol.ResubscribeResp{}, err
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

	c.reqMu.Lock()
	defer c.reqMu.Unlock()

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
	case *protocol.ResumeSessionResp:
		resp, err := transport.DecodePayload[protocol.ResumeSessionResp](payload)
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
	case *protocol.RenameResp:
		resp, err := transport.DecodePayload[protocol.RenameResp](payload)
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
	case *protocol.CreateResp:
		resp, err := transport.DecodePayload[protocol.CreateResp](payload)
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
	case *protocol.WriteResp:
		resp, err := transport.DecodePayload[protocol.WriteResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.FlushResp:
		resp, err := transport.DecodePayload[protocol.FlushResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.TruncateResp:
		resp, err := transport.DecodePayload[protocol.TruncateResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.SetDeleteOnCloseResp:
		resp, err := transport.DecodePayload[protocol.SetDeleteOnCloseResp](payload)
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
	case *protocol.SubscribeResp:
		resp, err := transport.DecodePayload[protocol.SubscribeResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.PollEventsResp:
		resp, err := transport.DecodePayload[protocol.PollEventsResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.AckEventsResp:
		resp, err := transport.DecodePayload[protocol.AckEventsResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.ResyncResp:
		resp, err := transport.DecodePayload[protocol.ResyncResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.RecoverHandlesResp:
		resp, err := transport.DecodePayload[protocol.RecoverHandlesResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.RevalidateResp:
		resp, err := transport.DecodePayload[protocol.RevalidateResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	case *protocol.ResubscribeResp:
		resp, err := transport.DecodePayload[protocol.ResubscribeResp](payload)
		if err != nil {
			return err
		}
		*target = resp
		return nil
	default:
		return fmt.Errorf("unsupported response target %T", out)
	}
}
