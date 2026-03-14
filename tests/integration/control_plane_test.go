package integration

import (
	"net"
	"testing"
	"time"

	"developer-mount/internal/client"
	"developer-mount/internal/server"
)

func TestControlPlaneHandshakeAndSession(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer ln.Close()

	srv := server.New(ln.Addr().String())
	fixedNow := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	srv.Now = func() time.Time { return fixedNow }
	go func() {
		_ = srv.Serve(ln)
	}()

	cli := client.New(ln.Addr().String())
	if err := cli.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer cli.Close()

	helloResp, err := cli.Hello()
	if err != nil {
		t.Fatalf("Hello() error = %v", err)
	}
	if helloResp.SelectedProtocolVersion != 1 {
		t.Fatalf("selected protocol version = %d, want 1", helloResp.SelectedProtocolVersion)
	}

	authResp, err := cli.Auth("devmount-dev-token")
	if err != nil {
		t.Fatalf("Auth() error = %v", err)
	}
	if !authResp.Authenticated {
		t.Fatalf("expected authenticated response")
	}

	sessionResp, err := cli.CreateSession("client-1", 30)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if sessionResp.SessionID == 0 {
		t.Fatalf("expected non-zero session id")
	}

	hbResp, err := cli.Heartbeat()
	if err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}
	if hbResp.SessionID != sessionResp.SessionID {
		t.Fatalf("heartbeat session mismatch: got %d want %d", hbResp.SessionID, sessionResp.SessionID)
	}
}
