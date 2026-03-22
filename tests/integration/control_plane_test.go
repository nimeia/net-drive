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
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello from iter4"), 0o644); err != nil {
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

	if _, err := cli.Hello(); err != nil {
		t.Fatalf("Hello() error = %v", err)
	}
	if authResp, err := cli.Auth("devmount-dev-token"); err != nil || !authResp.Authenticated {
		t.Fatalf("Auth() error=%v authenticated=%v", err, authResp.Authenticated)
	}
	if sessionResp, err := cli.CreateSession("client-1", 30); err != nil || sessionResp.SessionID == 0 {
		t.Fatalf("CreateSession() error=%v session=%+v", err, sessionResp)
	}
	if _, err := cli.Heartbeat(); err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
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
	if _, err := cli.CloseDir(dirResp.DirCursorID); err != nil {
		t.Fatalf("CloseDir() error = %v", err)
	}
	if _, err := cli.ReadDir(dirResp.DirCursorID, 0, 10); err == nil {
		t.Fatalf("ReadDir(closed cursor) error = nil, want error")
	}

	openResp, err := cli.OpenRead(lookupResp.Entry.NodeID)
	if err != nil {
		t.Fatalf("OpenRead() error = %v", err)
	}
	readResp, err := cli.Read(openResp.HandleID, 0, 5)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if string(readResp.Data) != "hello" {
		t.Fatalf("read data = %q, want hello", string(readResp.Data))
	}
	if _, err := cli.CloseHandle(openResp.HandleID); err != nil {
		t.Fatalf("CloseHandle() error = %v", err)
	}

	createResp, err := cli.Create(cli.RootNodeID, "notes.txt", false)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := cli.Write(createResp.HandleID, 0, []byte("iter4-write")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if _, err := cli.Flush(createResp.HandleID); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	if _, err := cli.CloseHandle(createResp.HandleID); err != nil {
		t.Fatalf("CloseHandle(create) error = %v", err)
	}

	notesLookup, err := cli.Lookup(cli.RootNodeID, "notes.txt")
	if err != nil {
		t.Fatalf("Lookup(notes) error = %v", err)
	}
	if notesLookup.Entry.Size != int64(len("iter4-write")) {
		t.Fatalf("notes size = %d, want %d", notesLookup.Entry.Size, len("iter4-write"))
	}

	notesOpen, err := cli.OpenWrite(notesLookup.Entry.NodeID, true)
	if err != nil {
		t.Fatalf("OpenWrite() error = %v", err)
	}
	if _, err := cli.Write(notesOpen.HandleID, 0, []byte("abcde")); err != nil {
		t.Fatalf("Write(overwrite) error = %v", err)
	}
	if _, err := cli.Truncate(notesOpen.HandleID, 3); err != nil {
		t.Fatalf("Truncate() error = %v", err)
	}
	if _, err := cli.Flush(notesOpen.HandleID); err != nil {
		t.Fatalf("Flush(overwrite) error = %v", err)
	}
	if _, err := cli.CloseHandle(notesOpen.HandleID); err != nil {
		t.Fatalf("CloseHandle(overwrite) error = %v", err)
	}

	notesOpenRead, err := cli.OpenRead(notesLookup.Entry.NodeID)
	if err != nil {
		t.Fatalf("OpenRead(notes) error = %v", err)
	}
	readResp, err = cli.Read(notesOpenRead.HandleID, 0, 32)
	if err != nil {
		t.Fatalf("Read(notes) error = %v", err)
	}
	if string(readResp.Data) != "abc" {
		t.Fatalf("notes data = %q, want abc", string(readResp.Data))
	}
	if _, err := cli.CloseHandle(notesOpenRead.HandleID); err != nil {
		t.Fatalf("CloseHandle(read notes) error = %v", err)
	}

	tmpResp, err := cli.Create(cli.RootNodeID, ".hello.tmp", false)
	if err != nil {
		t.Fatalf("Create(tmp) error = %v", err)
	}
	if _, err := cli.Write(tmpResp.HandleID, 0, []byte("replaced-from-temp")); err != nil {
		t.Fatalf("Write(tmp) error = %v", err)
	}
	if _, err := cli.Flush(tmpResp.HandleID); err != nil {
		t.Fatalf("Flush(tmp) error = %v", err)
	}
	if _, err := cli.CloseHandle(tmpResp.HandleID); err != nil {
		t.Fatalf("CloseHandle(tmp) error = %v", err)
	}
	if _, err := cli.Rename(cli.RootNodeID, ".hello.tmp", cli.RootNodeID, "hello.txt", true); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	helloLookup, err := cli.Lookup(cli.RootNodeID, "hello.txt")
	if err != nil {
		t.Fatalf("Lookup(renamed hello) error = %v", err)
	}
	helloReadOpen, err := cli.OpenRead(helloLookup.Entry.NodeID)
	if err != nil {
		t.Fatalf("OpenRead(renamed hello) error = %v", err)
	}
	readResp, err = cli.Read(helloReadOpen.HandleID, 0, 64)
	if err != nil {
		t.Fatalf("Read(renamed hello) error = %v", err)
	}
	if string(readResp.Data) != "replaced-from-temp" {
		t.Fatalf("hello data after replace = %q, want replaced-from-temp", string(readResp.Data))
	}
	if _, err := cli.CloseHandle(helloReadOpen.HandleID); err != nil {
		t.Fatalf("CloseHandle(renamed hello) error = %v", err)
	}

	deleteResp, err := cli.Create(cli.RootNodeID, "delete-me.tmp", false)
	if err != nil {
		t.Fatalf("Create(delete temp) error = %v", err)
	}
	if _, err := cli.SetDeleteOnClose(deleteResp.HandleID, true); err != nil {
		t.Fatalf("SetDeleteOnClose() error = %v", err)
	}
	if _, err := cli.CloseHandle(deleteResp.HandleID); err != nil {
		t.Fatalf("CloseHandle(delete temp) error = %v", err)
	}
	if _, err := cli.Lookup(cli.RootNodeID, "delete-me.tmp"); err == nil {
		t.Fatalf("expected delete-on-close file lookup to fail")
	}
}
