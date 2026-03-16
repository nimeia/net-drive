package clientcore

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
			}, protocol.CreateSessionResp{SessionID: 9001, LeaseSeconds: 30, ExpiresAt: "2026-03-16T00:00:30Z", State: "active"})
		})

		resp, err := cli.CreateSession("client-A", 30)
		if err != nil {
			t.Fatalf("CreateSession() error = %v", err)
		}
		if resp.SessionID != 9001 || cli.SessionID != 9001 {
			t.Fatalf("session ids = resp:%d client:%d, want 9001", resp.SessionID, cli.SessionID)
		}
		state := cli.SnapshotState()
		if state.ClientInstanceID != "client-A" || state.LeaseSeconds != 30 || state.SessionState != "active" {
			t.Fatalf("SnapshotState() = %+v", state)
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

func TestRecoverySnapshotTracksHandlesWatchesAndNodes(t *testing.T) {
	cli := New("pipe")
	cli.SessionID = 77
	cli.state.ClientInstanceID = "client-A"
	cli.trackNodeIDs(7, 3)
	cli.updateHandle(TrackedHandle{HandleID: 10, NodeID: 7, Writable: true, DeleteOnClose: true})
	cli.updateWatch(TrackedWatch{WatchID: 20, NodeID: 3, Recursive: true, LastAckedSeq: 9})
	cli.updateDirCursor(TrackedDirCursor{DirCursorID: 30, NodeID: 3})

	snap := cli.SnapshotRecoveryState()
	if snap.SessionID != 77 || snap.ClientInstanceID != "client-A" {
		t.Fatalf("SnapshotRecoveryState() header = %+v", snap)
	}
	if len(snap.Handles) != 1 || snap.Handles[0].PreviousHandleID != 10 || !snap.Handles[0].Writable || !snap.Handles[0].DeleteOnClose {
		t.Fatalf("SnapshotRecoveryState().Handles = %+v", snap.Handles)
	}
	if len(snap.Watches) != 1 || snap.Watches[0].PreviousWatchID != 20 || snap.Watches[0].AfterSeq != 9 {
		t.Fatalf("SnapshotRecoveryState().Watches = %+v", snap.Watches)
	}
	if got, want := snap.NodeIDs, []uint64{1, 3, 7}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("SnapshotRecoveryState().NodeIDs = %v, want %v", got, want)
	}
}

func TestRecoveryResultsReplaceTrackedHandlesAndWatches(t *testing.T) {
	cli := New("pipe")
	cli.updateHandle(TrackedHandle{HandleID: 10, NodeID: 7, Writable: true})
	cli.updateWatch(TrackedWatch{WatchID: 20, NodeID: 3, Recursive: true, LastAckedSeq: 8, LastSeenSeq: 10})

	cli.applyRecoveredHandles([]protocol.RecoverHandleResult{{PreviousHandleID: 10, HandleID: 100, NodeID: 7, Size: 42}})
	cli.applyResubscribedWatches([]protocol.ResubscribeResult{{PreviousWatchID: 20, WatchID: 200, StartSeq: 11, AckedSeq: 9}})

	state := cli.SnapshotState()
	if _, ok := state.TrackedHandles[10]; ok {
		t.Fatalf("old handle still present: %+v", state.TrackedHandles)
	}
	handle, ok := state.TrackedHandles[100]
	if !ok || !handle.Writable || handle.LastKnownSize != 42 {
		t.Fatalf("recovered handle = %+v", handle)
	}
	if _, ok := state.TrackedWatches[20]; ok {
		t.Fatalf("old watch still present: %+v", state.TrackedWatches)
	}
	watch, ok := state.TrackedWatches[200]
	if !ok || !watch.Recursive || watch.LastAckedSeq != 9 || watch.LastSeenSeq != 11 {
		t.Fatalf("resubscribed watch = %+v", watch)
	}
}

func TestDecodeIntoUnsupportedTarget(t *testing.T) {
	var out struct{}
	err := DecodeInto([]byte(`{"ok":true}`), &out)
	if err == nil || !strings.Contains(err.Error(), "unsupported response target") {
		t.Fatalf("DecodeInto() error = %v, want unsupported response target", err)
	}
}
