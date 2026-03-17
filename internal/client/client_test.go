package client

import (
	"net"
	"strings"
	"testing"

	"developer-mount/internal/protocol"
	"developer-mount/internal/transport"
)

func runPipeServer(t *testing.T, serverConn net.Conn, fn func(net.Conn)) {
	t.Helper()
	go func() {
		defer serverConn.Close()
		fn(serverConn)
	}()
}

func writePipeResponse(t *testing.T, conn net.Conn, header protocol.Header, payload any) {
	t.Helper()
	frame, err := transport.EncodeFrame(header, payload)
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	if _, err := conn.Write(frame); err != nil {
		t.Fatalf("Write(response) error = %v", err)
	}
}

func TestClientRequestEncodesHeaderAndDecodesResponse(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	cli := New("pipe")
	cli.Conn = clientConn
	cli.SessionID = 77

	runPipeServer(t, serverConn, func(conn net.Conn) {
		header, payload, err := transport.DecodeFrame(conn)
		if err != nil {
			t.Errorf("DecodeFrame(request) error = %v", err)
			return
		}
		if header.Channel != protocol.ChannelMetadata {
			t.Errorf("header.Channel = %d, want %d", header.Channel, protocol.ChannelMetadata)
		}
		if header.Opcode != protocol.OpcodeGetAttrReq {
			t.Errorf("header.Opcode = %d, want %d", header.Opcode, protocol.OpcodeGetAttrReq)
		}
		if header.Flags != protocol.FlagRequest {
			t.Errorf("header.Flags = %d, want %d", header.Flags, protocol.FlagRequest)
		}
		if header.SessionID != 77 {
			t.Errorf("header.SessionID = %d, want %d", header.SessionID, 77)
		}
		req, err := transport.DecodePayload[protocol.GetAttrReq](payload)
		if err != nil {
			t.Errorf("DecodePayload(GetAttrReq) error = %v", err)
			return
		}
		if req.NodeID != 123 {
			t.Errorf("req.NodeID = %d, want %d", req.NodeID, 123)
		}

		writePipeResponse(t, conn, protocol.Header{
			Channel:   protocol.ChannelMetadata,
			Opcode:    protocol.OpcodeGetAttrResp,
			RequestID: header.RequestID,
			SessionID: header.SessionID,
		}, protocol.GetAttrResp{Entry: protocol.NodeInfo{NodeID: 123, Name: "hello.txt"}})
	})

	resp, err := cli.GetAttr(123)
	if err != nil {
		t.Fatalf("GetAttr() error = %v", err)
	}
	if resp.Entry.NodeID != 123 || resp.Entry.Name != "hello.txt" {
		t.Fatalf("GetAttr() resp = %+v", resp)
	}
}

func TestClientRequestErrorAndMismatchPaths(t *testing.T) {
	t.Run("error-response", func(t *testing.T) {
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()

		cli := New("pipe")
		cli.Conn = clientConn

		runPipeServer(t, serverConn, func(conn net.Conn) {
			header, _, err := transport.DecodeFrame(conn)
			if err != nil {
				t.Errorf("DecodeFrame(request) error = %v", err)
				return
			}
			writePipeResponse(t, conn, protocol.Header{
				Channel:   protocol.ChannelControl,
				Opcode:    protocol.OpcodeAuthResp,
				Flags:     protocol.FlagError,
				RequestID: header.RequestID,
			}, protocol.ErrorResp{Code: protocol.ErrAuthRequired, Message: "authentication required"})
		})

		_, err := cli.Auth("wrong")
		if err == nil || !strings.Contains(err.Error(), string(protocol.ErrAuthRequired)) {
			t.Fatalf("Auth() error = %v, want %s", err, protocol.ErrAuthRequired)
		}
	})

	t.Run("request-id-mismatch", func(t *testing.T) {
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()

		cli := New("pipe")
		cli.Conn = clientConn

		runPipeServer(t, serverConn, func(conn net.Conn) {
			header, _, err := transport.DecodeFrame(conn)
			if err != nil {
				t.Errorf("DecodeFrame(request) error = %v", err)
				return
			}
			writePipeResponse(t, conn, protocol.Header{
				Channel:   protocol.ChannelControl,
				Opcode:    protocol.OpcodeHelloResp,
				RequestID: header.RequestID + 1,
			}, protocol.HelloResp{ServerName: "devmount-server"})
		})

		_, err := cli.Hello()
		if err == nil || !strings.Contains(err.Error(), "request id mismatch") {
			t.Fatalf("Hello() error = %v, want request id mismatch", err)
		}
	})

	t.Run("malformed-error-payload", func(t *testing.T) {
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()

		cli := New("pipe")
		cli.Conn = clientConn

		runPipeServer(t, serverConn, func(conn net.Conn) {
			header, _, err := transport.DecodeFrame(conn)
			if err != nil {
				t.Errorf("DecodeFrame(request) error = %v", err)
				return
			}
			writePipeResponse(t, conn, protocol.Header{
				Channel:   protocol.ChannelControl,
				Opcode:    protocol.OpcodeAuthResp,
				Flags:     protocol.FlagError,
				RequestID: header.RequestID,
			}, "not-an-error-object")
		})

		_, err := cli.Auth("whatever")
		if err == nil || !strings.Contains(err.Error(), "decode error payload") {
			t.Fatalf("Auth() error = %v, want decode error payload", err)
		}
	})
}

func TestClientSessionTrackingOnCreateAndResume(t *testing.T) {
	t.Run("create-session-updates-session-id", func(t *testing.T) {
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()

		cli := New("pipe")
		cli.Conn = clientConn

		runPipeServer(t, serverConn, func(conn net.Conn) {
			header, payload, err := transport.DecodeFrame(conn)
			if err != nil {
				t.Errorf("DecodeFrame(request) error = %v", err)
				return
			}
			req, err := transport.DecodePayload[protocol.CreateSessionReq](payload)
			if err != nil {
				t.Errorf("DecodePayload(CreateSessionReq) error = %v", err)
				return
			}
			if req.ClientInstanceID != "client-A" {
				t.Errorf("ClientInstanceID = %q, want %q", req.ClientInstanceID, "client-A")
			}
			writePipeResponse(t, conn, protocol.Header{
				Channel:   protocol.ChannelControl,
				Opcode:    protocol.OpcodeCreateSessionResp,
				RequestID: header.RequestID,
			}, protocol.CreateSessionResp{SessionID: 9001})
		})

		resp, err := cli.CreateSession("client-A", 30)
		if err != nil {
			t.Fatalf("CreateSession() error = %v", err)
		}
		if resp.SessionID != 9001 || cli.SessionID != 9001 {
			t.Fatalf("session ids = resp:%d client:%d, want 9001", resp.SessionID, cli.SessionID)
		}
	})

	t.Run("resume-session-restores-previous-id-on-error", func(t *testing.T) {
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()

		cli := New("pipe")
		cli.Conn = clientConn
		cli.SessionID = 444

		runPipeServer(t, serverConn, func(conn net.Conn) {
			header, _, err := transport.DecodeFrame(conn)
			if err != nil {
				t.Errorf("DecodeFrame(request) error = %v", err)
				return
			}
			writePipeResponse(t, conn, protocol.Header{
				Channel:   protocol.ChannelControl,
				Opcode:    protocol.OpcodeResumeSessionResp,
				Flags:     protocol.FlagError,
				RequestID: header.RequestID,
			}, protocol.ErrorResp{Code: protocol.ErrSessionExpired, Message: "session expired"})
		})

		_, err := cli.ResumeSession(555, "client-A")
		if err == nil || !strings.Contains(err.Error(), string(protocol.ErrSessionExpired)) {
			t.Fatalf("ResumeSession() error = %v, want %s", err, protocol.ErrSessionExpired)
		}
		if cli.SessionID != 444 {
			t.Fatalf("cli.SessionID = %d, want previous %d", cli.SessionID, 444)
		}
	})
}

func TestDecodeIntoUnsupportedTarget(t *testing.T) {
	var out struct{}
	err := decodeInto([]byte(`{"ok":true}`), &out)
	if err == nil || !strings.Contains(err.Error(), "unsupported response target") {
		t.Fatalf("decodeInto() error = %v, want unsupported response target", err)
	}
}
