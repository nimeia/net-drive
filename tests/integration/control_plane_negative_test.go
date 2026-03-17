package integration

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"developer-mount/internal/client"
	"developer-mount/internal/protocol"
	"developer-mount/internal/server"
	"developer-mount/internal/transport"
)

type negativeClock struct {
	now time.Time
}

func (c *negativeClock) Now() time.Time          { return c.now }
func (c *negativeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func startNegativeServer(t *testing.T, root string, retention int, now time.Time) (string, *negativeClock) {
	t.Helper()

	clock := &negativeClock{now: now}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	srv := server.New(ln.Addr().String())
	srv.RootPath = root
	srv.JournalRetention = retention
	srv.Now = clock.Now
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = ln.Close() })
	return ln.Addr().String(), clock
}

func newRawConn(t *testing.T, addr string) net.Conn {
	t.Helper()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func writeRawRequest(t *testing.T, conn net.Conn, header protocol.Header, payload any) {
	t.Helper()

	header.Flags = protocol.FlagRequest
	frame, err := transport.EncodeFrame(header, payload)
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	if _, err := conn.Write(frame); err != nil {
		t.Fatalf("Write(frame) error = %v", err)
	}
}

func readErrorResponse(t *testing.T, conn net.Conn, wantCode protocol.ErrorCode) protocol.ErrorResp {
	t.Helper()

	header, payload, err := transport.DecodeFrame(conn)
	if err != nil {
		t.Fatalf("DecodeFrame() error = %v", err)
	}
	if header.Flags&protocol.FlagError == 0 {
		t.Fatalf("expected error response, got flags=%d opcode=%d", header.Flags, header.Opcode)
	}
	resp, err := transport.DecodePayload[protocol.ErrorResp](payload)
	if err != nil {
		t.Fatalf("DecodePayload(ErrorResp) error = %v", err)
	}
	if resp.Code != wantCode {
		t.Fatalf("error code = %s, want %s", resp.Code, wantCode)
	}
	return resp
}

func connectAuthedSessionClient(t *testing.T, addr, instanceID string) *client.Client {
	t.Helper()

	cli := client.New(addr)
	if err := cli.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	if _, err := cli.Hello(); err != nil {
		t.Fatalf("Hello() error = %v", err)
	}
	if _, err := cli.Auth("devmount-dev-token"); err != nil {
		t.Fatalf("Auth() error = %v", err)
	}
	if _, err := cli.CreateSession(instanceID, 30); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	return cli
}

func TestControlPlaneNegativeHandshakeMatrix(t *testing.T) {
	root := t.TempDir()
	addr, _ := startNegativeServer(t, root, 64, time.Date(2026, 3, 16, 9, 0, 0, 0, time.UTC))

	t.Run("auth-before-hello", func(t *testing.T) {
		conn := newRawConn(t, addr)
		writeRawRequest(t, conn, protocol.Header{Channel: protocol.ChannelControl, Opcode: protocol.OpcodeAuthReq, RequestID: 1}, protocol.AuthReq{
			Scheme: "dev-token",
			Token:  "devmount-dev-token",
		})
		resp := readErrorResponse(t, conn, protocol.ErrInvalidRequest)
		if !strings.Contains(resp.Message, "hello required") {
			t.Fatalf("error message = %q, want hello-required hint", resp.Message)
		}
	})

	t.Run("unsupported-version", func(t *testing.T) {
		conn := newRawConn(t, addr)
		writeRawRequest(t, conn, protocol.Header{Channel: protocol.ChannelControl, Opcode: protocol.OpcodeHelloReq, RequestID: 2}, protocol.HelloReq{
			ClientName:                "neg-client",
			ClientVersion:             "0.1.0",
			SupportedProtocolVersions: []uint8{99},
			Capabilities:              protocol.DefaultCapabilities(),
		})
		readErrorResponse(t, conn, protocol.ErrUnsupportedVersion)
	})

	t.Run("wrong-token", func(t *testing.T) {
		conn := newRawConn(t, addr)
		writeRawRequest(t, conn, protocol.Header{Channel: protocol.ChannelControl, Opcode: protocol.OpcodeHelloReq, RequestID: 3}, protocol.HelloReq{
			ClientName:                "neg-client",
			ClientVersion:             "0.1.0",
			SupportedProtocolVersions: []uint8{protocol.Version},
			Capabilities:              protocol.DefaultCapabilities(),
		})
		if _, payload, err := transport.DecodeFrame(conn); err != nil {
			t.Fatalf("DecodeFrame(hello) error = %v", err)
		} else if _, err := transport.DecodePayload[protocol.HelloResp](payload); err != nil {
			t.Fatalf("DecodePayload(HelloResp) error = %v", err)
		}
		writeRawRequest(t, conn, protocol.Header{Channel: protocol.ChannelControl, Opcode: protocol.OpcodeAuthReq, RequestID: 4}, protocol.AuthReq{
			Scheme: "dev-token",
			Token:  "wrong-token",
		})
		readErrorResponse(t, conn, protocol.ErrAuthRequired)
	})

	t.Run("unsupported-channel", func(t *testing.T) {
		conn := newRawConn(t, addr)
		writeRawRequest(t, conn, protocol.Header{Channel: protocol.Channel(99), Opcode: 250, RequestID: 5}, map[string]any{"noop": true})
		readErrorResponse(t, conn, protocol.ErrUnsupportedOp)
	})
}

func TestSessionGatingAndProtocolErrorMapping(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "hello.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	addr, clock := startNegativeServer(t, root, 64, time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC))

	t.Run("metadata-requires-active-session", func(t *testing.T) {
		cli := client.New(addr)
		if err := cli.Connect(); err != nil {
			t.Fatalf("Connect() error = %v", err)
		}
		defer cli.Close()
		if _, err := cli.Hello(); err != nil {
			t.Fatalf("Hello() error = %v", err)
		}
		if _, err := cli.Auth("devmount-dev-token"); err != nil {
			t.Fatalf("Auth() error = %v", err)
		}
		if _, err := cli.GetAttr(cli.RootNodeID); err == nil || !strings.Contains(err.Error(), string(protocol.ErrSessionNotFound)) {
			t.Fatalf("GetAttr() error = %v, want %s", err, protocol.ErrSessionNotFound)
		}
		if _, err := cli.CreateSession("negative-client", 30); err != nil {
			t.Fatalf("CreateSession() error = %v", err)
		}
		clock.Advance(31 * time.Second)
		if _, err := cli.GetAttr(cli.RootNodeID); err == nil || !strings.Contains(err.Error(), string(protocol.ErrSessionExpired)) {
			t.Fatalf("GetAttr() expired error = %v, want %s", err, protocol.ErrSessionExpired)
		}
	})

	t.Run("backend-errors-map-to-protocol-errors", func(t *testing.T) {
		cli := connectAuthedSessionClient(t, addr, "mapping-client")

		if _, err := cli.GetAttr(999999); err == nil || !strings.Contains(err.Error(), string(protocol.ErrNotFound)) {
			t.Fatalf("GetAttr(missing) error = %v, want %s", err, protocol.ErrNotFound)
		}

		rootAttr, err := cli.GetAttr(cli.RootNodeID)
		if err != nil {
			t.Fatalf("GetAttr(root) error = %v", err)
		}
		if _, err := cli.OpenRead(rootAttr.Entry.NodeID); err == nil || !strings.Contains(err.Error(), string(protocol.ErrIsDir)) {
			t.Fatalf("OpenRead(root) error = %v, want %s", err, protocol.ErrIsDir)
		}

		fileNode, err := cli.Lookup(cli.RootNodeID, "hello.txt")
		if err != nil {
			t.Fatalf("Lookup(hello.txt) error = %v", err)
		}
		if _, err := cli.OpenDir(fileNode.Entry.NodeID); err == nil || !strings.Contains(err.Error(), string(protocol.ErrNotDir)) {
			t.Fatalf("OpenDir(file) error = %v, want %s", err, protocol.ErrNotDir)
		}

		readHandle, err := cli.OpenRead(fileNode.Entry.NodeID)
		if err != nil {
			t.Fatalf("OpenRead(hello.txt) error = %v", err)
		}
		if _, err := cli.Write(readHandle.HandleID, 0, []byte("x")); err == nil || !strings.Contains(err.Error(), string(protocol.ErrAccessDenied)) {
			t.Fatalf("Write(read-only) error = %v, want %s", err, protocol.ErrAccessDenied)
		}
		if _, err := cli.CloseHandle(readHandle.HandleID); err != nil {
			t.Fatalf("CloseHandle(read-only) error = %v", err)
		}

		if _, err := cli.Read(987654, 0, 8); err == nil || !strings.Contains(err.Error(), string(protocol.ErrInvalidHandle)) {
			t.Fatalf("Read(invalid handle) error = %v, want %s", err, protocol.ErrInvalidHandle)
		}

		if _, err := cli.Create(cli.RootNodeID, "hello.txt", false); err == nil || !strings.Contains(err.Error(), string(protocol.ErrAlreadyExists)) {
			t.Fatalf("Create(existing) error = %v, want %s", err, protocol.ErrAlreadyExists)
		}
	})
}
