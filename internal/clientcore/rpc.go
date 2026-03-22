package clientcore

import (
	"fmt"

	"developer-mount/internal/protocol"
	"developer-mount/internal/transport"
)

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
	return DecodeInto(payload, respPayload)
}

func requestDecode[T any](c *Client, channel protocol.Channel, opcode protocol.Opcode, reqPayload any) (T, error) {
	var resp T
	if err := c.request(channel, opcode, reqPayload, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

func DecodeInto(payload []byte, out any) error {
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
	case *protocol.CloseDirResp:
		resp, err := transport.DecodePayload[protocol.CloseDirResp](payload)
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
