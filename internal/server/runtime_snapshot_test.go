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
	handleID, _, err := backend.OpenFile(readme.NodeID, false, false)
	if err != nil {
		t.Fatalf("OpenFile(README.md) error = %v", err)
	}
	defer func() { _ = backend.CloseHandle(handleID) }()
	cursorID, err := backend.OpenDir(backend.RootNodeID())
	if err != nil {
		t.Fatalf("OpenDir(root) error = %v", err)
	}
	defer func() { _ = backend.CloseDir(cursorID) }()

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
	if snap.Metadata.Diagnostics.DirRefresh.Count == 0 || snap.Metadata.Locks.WriteHold.Count == 0 {
		t.Fatalf("metadata diagnostics missing refresh/hold samples: %+v %+v", snap.Metadata.Diagnostics, snap.Metadata.Locks)
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

func TestMetadataRuntimeSnapshotPrunesExpiredCaches(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("demo"), 0o644); err != nil {
		t.Fatalf("WriteFile(README.md) error = %v", err)
	}
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	fakeNow := time.Now().Add(10 * time.Second)
	backend.now = func() time.Time { return fakeNow }
	backend.cacheTTL = time.Second

	backend.mu.Lock()
	backend.attrCache["stale.txt"] = attrCacheEntry{info: protocol.NodeInfo{Name: "stale.txt"}, expiresAt: fakeNow.Add(-time.Second)}
	backend.negativeCache["missing.txt"] = negativeCacheEntry{expiresAt: fakeNow.Add(-time.Second)}
	backend.dirSnapshots[9999] = dirSnapshotEntry{entries: []protocol.DirEntry{{Name: "stale.txt"}}, expiresAt: fakeNow.Add(-time.Second)}
	backend.smallFileCache["stale.txt"] = smallFileCacheEntry{data: []byte("stale"), expiresAt: fakeNow.Add(-time.Second)}
	backend.mu.Unlock()

	snap := backend.RuntimeSnapshot()
	if snap.AttrCache != 0 || snap.NegativeCache != 0 || snap.DirSnapshots != 0 || snap.SmallFileCache != 0 {
		t.Fatalf("RuntimeSnapshot() = %+v, want expired caches pruned", snap)
	}
	if snap.Diagnostics.CachePrune.Count == 0 {
		t.Fatalf("RuntimeSnapshot() diagnostics = %+v, want prune count", snap.Diagnostics)
	}
}

func TestMetadataRuntimeSnapshotIncludesWriteFlushRenameDiagnostics(t *testing.T) {
	root := t.TempDir()
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	entry, handleID, err := backend.CreateFile(backend.RootNodeID(), "tmp.txt", false)
	if err != nil {
		t.Fatalf("CreateFile(tmp.txt) error = %v", err)
	}
	if _, _, err := backend.WriteFile(handleID, 0, []byte("payload")); err != nil {
		t.Fatalf("WriteFile(tmp.txt) error = %v", err)
	}
	if err := backend.FlushHandle(handleID); err != nil {
		t.Fatalf("FlushHandle(tmp.txt) error = %v", err)
	}
	if err := backend.CloseHandle(handleID); err != nil {
		t.Fatalf("CloseHandle(tmp.txt) error = %v", err)
	}
	if _, err := backend.RenamePath(backend.RootNodeID(), entry.Name, backend.RootNodeID(), "final.txt", false); err != nil {
		t.Fatalf("RenamePath(tmp.txt->final.txt) error = %v", err)
	}

	snap := backend.RuntimeSnapshot()
	if snap.Diagnostics.WriteSyscall.Count == 0 || snap.Diagnostics.FlushSync.Count == 0 || snap.Diagnostics.RenameSyscall.Count == 0 {
		t.Fatalf("RuntimeSnapshot() diagnostics = %+v, want write/flush/rename counts", snap.Diagnostics)
	}
}
