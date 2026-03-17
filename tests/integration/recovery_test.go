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
)

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func newRecoveryEnv(t *testing.T, root string, retention int, initial time.Time, clientInstanceID string) (*client.Client, *testClock) {
	t.Helper()

	clock := &testClock{now: initial}
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
	if _, err := cli.CreateSession(clientInstanceID, 30); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	return cli, clock
}

func reconnectAndResume(t *testing.T, addr string, sessionID uint64, clientInstanceID string) *client.Client {
	t.Helper()

	cli := client.New(addr)
	if err := cli.Connect(); err != nil {
		t.Fatalf("Connect(resume client) error = %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	if _, err := cli.Hello(); err != nil {
		t.Fatalf("Hello(resume client) error = %v", err)
	}
	if _, err := cli.Auth("devmount-dev-token"); err != nil {
		t.Fatalf("Auth(resume client) error = %v", err)
	}
	if _, err := cli.ResumeSession(sessionID, clientInstanceID); err != nil {
		t.Fatalf("ResumeSession() error = %v", err)
	}
	return cli
}

func TestRecoveryResumeSessionMatrix(t *testing.T) {
	now := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)

	t.Run("success", func(t *testing.T) {
		root := t.TempDir()
		cli, _ := newRecoveryEnv(t, root, 64, now, "resume-client")
		sessionID := cli.SessionID
		addr := cli.Addr
		if err := cli.Close(); err != nil {
			t.Fatalf("Close(cli) error = %v", err)
		}

		resumed := reconnectAndResume(t, addr, sessionID, "resume-client")
		if resumed.SessionID != sessionID {
			t.Fatalf("resumed session id = %d, want %d", resumed.SessionID, sessionID)
		}
		if _, err := resumed.Heartbeat(); err != nil {
			t.Fatalf("Heartbeat(resumed) error = %v", err)
		}
	})

	t.Run("client-instance-mismatch", func(t *testing.T) {
		root := t.TempDir()
		cli, _ := newRecoveryEnv(t, root, 64, now, "resume-client")
		sessionID := cli.SessionID
		addr := cli.Addr
		if err := cli.Close(); err != nil {
			t.Fatalf("Close(cli) error = %v", err)
		}

		resumeClient := client.New(addr)
		if err := resumeClient.Connect(); err != nil {
			t.Fatalf("Connect() error = %v", err)
		}
		defer resumeClient.Close()
		if _, err := resumeClient.Hello(); err != nil {
			t.Fatalf("Hello() error = %v", err)
		}
		if _, err := resumeClient.Auth("devmount-dev-token"); err != nil {
			t.Fatalf("Auth() error = %v", err)
		}
		_, err := resumeClient.ResumeSession(sessionID, "other-client")
		if err == nil || !strings.Contains(err.Error(), string(protocol.ErrAccessDenied)) {
			t.Fatalf("ResumeSession() error = %v, want %s", err, protocol.ErrAccessDenied)
		}
	})

	t.Run("expired-session", func(t *testing.T) {
		root := t.TempDir()
		cli, clock := newRecoveryEnv(t, root, 64, now, "resume-client")
		sessionID := cli.SessionID
		addr := cli.Addr
		clock.Advance(31 * time.Second)
		if err := cli.Close(); err != nil {
			t.Fatalf("Close(cli) error = %v", err)
		}

		resumeClient := client.New(addr)
		if err := resumeClient.Connect(); err != nil {
			t.Fatalf("Connect() error = %v", err)
		}
		defer resumeClient.Close()
		if _, err := resumeClient.Hello(); err != nil {
			t.Fatalf("Hello() error = %v", err)
		}
		if _, err := resumeClient.Auth("devmount-dev-token"); err != nil {
			t.Fatalf("Auth() error = %v", err)
		}
		_, err := resumeClient.ResumeSession(sessionID, "resume-client")
		if err == nil || !strings.Contains(err.Error(), string(protocol.ErrSessionExpired)) {
			t.Fatalf("ResumeSession() error = %v, want %s", err, protocol.ErrSessionExpired)
		}
	})
}

func TestRecoveryHandleMatrix(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "resume.txt"), []byte("hello-resume"), 0o644); err != nil {
		t.Fatalf("WriteFile(resume.txt) error = %v", err)
	}
	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	cli1, _ := newRecoveryEnv(t, root, 64, now, "watch-client")

	resumeNode, err := cli1.Lookup(cli1.RootNodeID, "resume.txt")
	if err != nil {
		t.Fatalf("Lookup(resume.txt) error = %v", err)
	}
	openResp, err := cli1.OpenRead(resumeNode.Entry.NodeID)
	if err != nil {
		t.Fatalf("OpenRead(resume.txt) error = %v", err)
	}

	recovered, err := cli1.RecoverHandles([]protocol.RecoverHandleSpec{
		{
			PreviousHandleID: openResp.HandleID,
			NodeID:           resumeNode.Entry.NodeID,
			Writable:         false,
		},
		{
			PreviousHandleID: 9999,
			NodeID:           999999,
			Writable:         false,
		},
	})
	if err != nil {
		t.Fatalf("RecoverHandles() error = %v", err)
	}
	if len(recovered.Handles) != 2 {
		t.Fatalf("expected 2 recover results, got %d", len(recovered.Handles))
	}
	if recovered.Handles[0].Error != "" || recovered.Handles[0].HandleID == 0 {
		t.Fatalf("expected successful recovered handle, got %+v", recovered.Handles[0])
	}
	readResp, err := cli1.Read(recovered.Handles[0].HandleID, 0, 64)
	if err != nil {
		t.Fatalf("Read(recovered handle) error = %v", err)
	}
	if string(readResp.Data) != "hello-resume" {
		t.Fatalf("recovered read data = %q, want %q", string(readResp.Data), "hello-resume")
	}
	if recovered.Handles[1].Error == "" {
		t.Fatalf("expected per-handle recovery error for missing node")
	}
}

func TestRecoveryRevalidateAndResubscribeMatrix(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "resume.txt"), []byte("hello-resume"), 0o644); err != nil {
		t.Fatalf("WriteFile(resume.txt) error = %v", err)
	}
	now := time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC)
	cli1, _ := newRecoveryEnv(t, root, 64, now, "watch-client")

	resumeNode, err := cli1.Lookup(cli1.RootNodeID, "resume.txt")
	if err != nil {
		t.Fatalf("Lookup(resume.txt) error = %v", err)
	}
	sub, err := cli1.Subscribe(cli1.RootNodeID, true)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	ghostResp, err := cli1.Create(cli1.RootNodeID, "ghost.tmp", false)
	if err != nil {
		t.Fatalf("Create(ghost.tmp) error = %v", err)
	}
	ghostNodeID := ghostResp.Entry.NodeID
	if _, err := cli1.SetDeleteOnClose(ghostResp.HandleID, true); err != nil {
		t.Fatalf("SetDeleteOnClose(ghost.tmp) error = %v", err)
	}
	if _, err := cli1.CloseHandle(ghostResp.HandleID); err != nil {
		t.Fatalf("CloseHandle(ghost.tmp) error = %v", err)
	}

	poll, err := cli1.PollEvents(sub.WatchID, sub.StartSeq, 16)
	if err != nil {
		t.Fatalf("PollEvents() error = %v", err)
	}
	lastAck := sub.StartSeq
	if len(poll.Events) > 0 {
		lastAck = poll.Events[len(poll.Events)-1].EventSeq
		if _, err := cli1.AckEvents(sub.WatchID, lastAck); err != nil {
			t.Fatalf("AckEvents() error = %v", err)
		}
	}

	sessionID := cli1.SessionID
	addr := cli1.Addr
	if err := cli1.Close(); err != nil {
		t.Fatalf("Close(cli1) error = %v", err)
	}

	cli2 := reconnectAndResume(t, addr, sessionID, "watch-client")

	revalidated, err := cli2.RevalidateNodes([]uint64{resumeNode.Entry.NodeID, ghostNodeID, 999999})
	if err != nil {
		t.Fatalf("RevalidateNodes() error = %v", err)
	}
	if len(revalidated.Entries) != 3 {
		t.Fatalf("expected 3 revalidate entries, got %d", len(revalidated.Entries))
	}
	if !revalidated.Entries[0].Exists {
		t.Fatalf("expected resume node to exist")
	}
	if revalidated.Entries[1].Exists {
		t.Fatalf("expected ghost node to be absent after delete-on-close")
	}
	if revalidated.Entries[2].Exists {
		t.Fatalf("expected unknown node to be absent")
	}

	resub, err := cli2.ResubscribeWatches([]protocol.ResubscribeSpec{{
		PreviousWatchID: sub.WatchID,
		NodeID:          cli2.RootNodeID,
		Recursive:       true,
		AfterSeq:        0, // implementation should floor to acked seq
	}})
	if err != nil {
		t.Fatalf("ResubscribeWatches() error = %v", err)
	}
	if len(resub.Watches) != 1 {
		t.Fatalf("expected one resubscribe result, got %d", len(resub.Watches))
	}
	if resub.Watches[0].Error != "" {
		t.Fatalf("unexpected resubscribe error: %s", resub.Watches[0].Error)
	}
	if resub.Watches[0].StartSeq != lastAck || resub.Watches[0].AckedSeq != lastAck {
		t.Fatalf("resubscribe seq = start:%d ack:%d, want both %d", resub.Watches[0].StartSeq, resub.Watches[0].AckedSeq, lastAck)
	}

	createResp, err := cli2.Create(cli2.RootNodeID, "after-resume.txt", false)
	if err != nil {
		t.Fatalf("Create(after-resume.txt) error = %v", err)
	}
	if _, err := cli2.Write(createResp.HandleID, 0, []byte("after")); err != nil {
		t.Fatalf("Write(after-resume.txt) error = %v", err)
	}
	if _, err := cli2.Flush(createResp.HandleID); err != nil {
		t.Fatalf("Flush(after-resume.txt) error = %v", err)
	}
	if _, err := cli2.CloseHandle(createResp.HandleID); err != nil {
		t.Fatalf("CloseHandle(after-resume.txt) error = %v", err)
	}

	poll2, err := cli2.PollEvents(resub.Watches[0].WatchID, resub.Watches[0].StartSeq, 16)
	if err != nil {
		t.Fatalf("PollEvents(resubscribed) error = %v", err)
	}
	foundCreate := false
	for _, evt := range poll2.Events {
		if evt.Path == "after-resume.txt" && evt.EventType == protocol.EventCreate {
			foundCreate = true
			break
		}
	}
	if !foundCreate {
		t.Fatalf("expected create event for after-resume.txt in resubscribed watch")
	}
}
