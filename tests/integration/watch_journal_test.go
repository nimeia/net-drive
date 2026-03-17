package integration

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"developer-mount/internal/client"
	"developer-mount/internal/protocol"
	"developer-mount/internal/server"
)

func newConnectedClient(t *testing.T, root string, retention int) *client.Client {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	srv := server.New(ln.Addr().String())
	srv.RootPath = root
	srv.JournalRetention = retention
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = ln.Close() })

	cli := client.New(ln.Addr().String())
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
	if _, err := cli.CreateSession("watch-client", 30); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	return cli
}

func TestWatchJournalSubscribePollAck(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "initial.txt"), []byte("seed"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	cli := newConnectedClient(t, root, 64)

	sub, err := cli.Subscribe(cli.RootNodeID, true)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	createResp, err := cli.Create(cli.RootNodeID, "watch.txt", false)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := cli.Write(createResp.HandleID, 0, []byte("hello-watch")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if _, err := cli.Flush(createResp.HandleID); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	if _, err := cli.CloseHandle(createResp.HandleID); err != nil {
		t.Fatalf("CloseHandle() error = %v", err)
	}
	if _, err := cli.Rename(cli.RootNodeID, "watch.txt", cli.RootNodeID, "watch-renamed.txt", true); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	deleteResp, err := cli.Create(cli.RootNodeID, "delete-me.tmp", false)
	if err != nil {
		t.Fatalf("Create(delete) error = %v", err)
	}
	if _, err := cli.SetDeleteOnClose(deleteResp.HandleID, true); err != nil {
		t.Fatalf("SetDeleteOnClose() error = %v", err)
	}
	if _, err := cli.CloseHandle(deleteResp.HandleID); err != nil {
		t.Fatalf("CloseHandle(delete) error = %v", err)
	}

	poll, err := cli.PollEvents(sub.WatchID, sub.StartSeq, 32)
	if err != nil {
		t.Fatalf("PollEvents() error = %v", err)
	}
	if poll.Overflow || poll.NeedsResync {
		t.Fatalf("unexpected overflow in normal poll: %+v", poll)
	}
	if len(poll.Events) < 6 {
		t.Fatalf("expected at least 6 events, got %d", len(poll.Events))
	}
	wantOrder := []protocol.EventType{
		protocol.EventCreate,
		protocol.EventContentChanged,
		protocol.EventRenameFrom,
		protocol.EventRenameTo,
		protocol.EventCreate,
		protocol.EventDelete,
	}
	for i, want := range wantOrder {
		if poll.Events[i].EventType != want {
			t.Fatalf("event[%d] type = %s, want %s", i, poll.Events[i].EventType, want)
		}
	}
	lastSeq := poll.Events[len(poll.Events)-1].EventSeq
	ack, err := cli.AckEvents(sub.WatchID, lastSeq)
	if err != nil {
		t.Fatalf("AckEvents() error = %v", err)
	}
	if ack.AckedSeq != lastSeq {
		t.Fatalf("acked seq = %d, want %d", ack.AckedSeq, lastSeq)
	}
}

func TestWatchJournalOverflowAndResync(t *testing.T) {
	root := t.TempDir()
	cli := newConnectedClient(t, root, 2)
	sub, err := cli.Subscribe(cli.RootNodeID, true)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		resp, err := cli.Create(cli.RootNodeID, name, false)
		if err != nil {
			t.Fatalf("Create(%s) error = %v", name, err)
		}
		if _, err := cli.CloseHandle(resp.HandleID); err != nil {
			t.Fatalf("CloseHandle(%s) error = %v", name, err)
		}
	}
	poll, err := cli.PollEvents(sub.WatchID, sub.StartSeq, 16)
	if err != nil {
		t.Fatalf("PollEvents() error = %v", err)
	}
	if !poll.Overflow || !poll.NeedsResync {
		t.Fatalf("expected overflow + needs_resync, got %+v", poll)
	}
	resync, err := cli.Resync(sub.WatchID)
	if err != nil {
		t.Fatalf("Resync() error = %v", err)
	}
	seen := map[string]bool{}
	for _, entry := range resync.Entries {
		seen[entry.RelativePath] = true
	}
	for _, want := range []string{"", "a.txt", "b.txt", "c.txt"} {
		if !seen[want] {
			t.Fatalf("resync snapshot missing %q", want)
		}
	}
}

func TestWatchJournalNonRecursiveRootOnly(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	cli := newConnectedClient(t, root, 64)
	sub, err := cli.Subscribe(cli.RootNodeID, false)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	resp, err := cli.Create(cli.RootNodeID, "root.txt", false)
	if err != nil {
		t.Fatalf("Create(root.txt) error = %v", err)
	}
	if _, err := cli.CloseHandle(resp.HandleID); err != nil {
		t.Fatalf("CloseHandle(root.txt) error = %v", err)
	}
	nestedNode, err := cli.Lookup(cli.RootNodeID, "nested")
	if err != nil {
		t.Fatalf("Lookup(nested) error = %v", err)
	}
	nestedResp, err := cli.Create(nestedNode.Entry.NodeID, "child.txt", false)
	if err != nil {
		t.Fatalf("Create(child.txt) error = %v", err)
	}
	if _, err := cli.CloseHandle(nestedResp.HandleID); err != nil {
		t.Fatalf("CloseHandle(child.txt) error = %v", err)
	}
	// allow timestamps to differ if needed; protocol is request/response so events are already recorded.
	poll, err := cli.PollEvents(sub.WatchID, sub.StartSeq, 16)
	if err != nil {
		t.Fatalf("PollEvents() error = %v", err)
	}
	for _, evt := range poll.Events {
		if evt.Path == "nested/child.txt" {
			t.Fatalf("non-recursive watch should not include nested child event: %+v", evt)
		}
	}
	foundRoot := false
	for _, evt := range poll.Events {
		if evt.Path == "root.txt" {
			foundRoot = true
		}
	}
	if !foundRoot {
		t.Fatalf("expected root.txt event in non-recursive watch poll")
	}
	_ = time.Second
}
