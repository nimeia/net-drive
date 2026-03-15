package integration

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"developer-mount/internal/client"
	"developer-mount/internal/server"
)

func TestControlPlaneHandshakeAndSession(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello from iter2"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer ln.Close()

	srv := server.New(ln.Addr().String())
	srv.RootPath = root
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

	rootAttr, err := cli.GetAttr(cli.RootNodeID)
	if err != nil {
		t.Fatalf("GetAttr() error = %v", err)
	}
	if rootAttr.Entry.FileType != "directory" {
		t.Fatalf("root type = %s, want directory", rootAttr.Entry.FileType)
	}

	lookupResp, err := cli.Lookup(cli.RootNodeID, "hello.txt")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if lookupResp.Entry.Name != "hello.txt" {
		t.Fatalf("lookup name = %s, want hello.txt", lookupResp.Entry.Name)
	}

	dirResp, err := cli.OpenDir(cli.RootNodeID)
	if err != nil {
		t.Fatalf("OpenDir() error = %v", err)
	}
	listResp, err := cli.ReadDir(dirResp.DirCursorID, 0, 10)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(listResp.Entries) != 2 {
		t.Fatalf("entry count = %d, want 2", len(listResp.Entries))
	}

	openResp, err := cli.Open(lookupResp.Entry.NodeID)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	readResp, err := cli.Read(openResp.HandleID, 0, 5)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if string(readResp.Data) != "hello" {
		t.Fatalf("read data = %q, want hello", string(readResp.Data))
	}
	closeResp, err := cli.CloseHandle(openResp.HandleID)
	if err != nil {
		t.Fatalf("CloseHandle() error = %v", err)
	}
	if !closeResp.Closed {
		t.Fatalf("expected closed response")
	}
}
