package server

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"developer-mount/internal/protocol"
)

func TestServerSnapshotRuntimeIncludesCountsAndLockSamples(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll(docs) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("demo"), 0o644); err != nil {
		t.Fatalf("WriteFile(README.md) error = %v", err)
	}
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	sessions := NewSessionManager()
	sessions.Create("developer", "runtime-active", 30, now)
	sessions.Create("developer", "runtime-expired", 1, now.Add(-2*time.Second))

	readme, err := backend.Lookup(backend.RootNodeID(), "README.md")
	if err != nil {
		t.Fatalf("Lookup(README.md) error = %v", err)
	}
	if _, _, err := backend.OpenFile(readme.NodeID, false, false); err != nil {
		t.Fatalf("OpenFile(README.md) error = %v", err)
	}
	if _, err := backend.OpenDir(backend.RootNodeID()); err != nil {
		t.Fatalf("OpenDir(root) error = %v", err)
	}

	journal := newJournalBroker(backend, func() time.Time { return now }, 32)
	watchID, _, err := journal.Subscribe(backend.RootNodeID(), true)
	if err != nil {
		t.Fatalf("Subscribe(root) error = %v", err)
	}
	journal.Append(protocol.WatchEvent{EventType: protocol.EventCreate, NodeID: readme.NodeID, ParentNodeID: backend.RootNodeID(), Name: "README.md", Path: "README.md"})
	journal.Append(protocol.WatchEvent{EventType: protocol.EventCreate, NodeID: readme.NodeID, ParentNodeID: backend.RootNodeID(), Name: "guide.md", Path: "docs/guide.md"})
	if _, err := journal.Ack(watchID, 1); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}

	srv := New("127.0.0.1:9100")
	srv.Backend = backend
	srv.SessionManager = sessions
	srv.Journal = journal
	srv.Control.observe("hello", 2*time.Millisecond, false)
	srv.Control.observe("auth", 3*time.Millisecond, true)
	srv.Control.observe("create_session", time.Millisecond, false)
	srv.Control.observe("resume_session", 4*time.Millisecond, false)
	srv.Control.observe("heartbeat", 1500*time.Microsecond, false)
	srv.Faults.observeSuppressed(io.EOF)
	srv.Faults.observeSuppressed(io.ErrUnexpectedEOF)
	srv.Faults.observeLogged()
	snap := srv.SnapshotRuntime(now)
	if snap.At.IsZero() {
		t.Fatalf("SnapshotRuntime().At is zero")
	}
	if snap.Metadata.Handles < 1 || snap.Metadata.DirCursors < 1 || snap.Metadata.Nodes < 1 {
		t.Fatalf("metadata snapshot = %+v, want non-zero handles/cursors/nodes", snap.Metadata)
	}
	if snap.Sessions.Total != 2 || snap.Sessions.Active != 1 || snap.Sessions.Expired != 1 {
		t.Fatalf("session snapshot = %+v, want total=2 active=1 expired=1", snap.Sessions)
	}
	if snap.Journal.Watches != 1 || snap.Journal.Events != 2 || snap.Journal.MaxWatchBacklog != 1 {
		t.Fatalf("journal snapshot = %+v, want watches=1 events=2 backlog=1", snap.Journal)
	}
	if snap.Metadata.Locks.Read.Acquires == 0 || snap.Sessions.Locks.Read.Acquires == 0 || snap.Journal.Locks.Read.Acquires == 0 {
		t.Fatalf("lock snapshots missing acquisitions: metadata=%+v sessions=%+v journal=%+v", snap.Metadata.Locks, snap.Sessions.Locks, snap.Journal.Locks)
	}
	if snap.Control.Hello.Count != 1 || snap.Control.Auth.Errors != 1 || snap.Control.ResumeSession.Count != 1 || snap.Control.Heartbeat.MaxWait == 0 {
		t.Fatalf("control snapshot = %+v", snap.Control)
	}
	if snap.Faults.SuppressedEOF != 1 || snap.Faults.SuppressedUnexpectedEOF != 1 || snap.Faults.Logged != 1 {
		t.Fatalf("fault snapshot = %+v", snap.Faults)
	}
}
