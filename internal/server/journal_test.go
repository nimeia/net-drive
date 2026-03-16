package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"developer-mount/internal/protocol"
)

func TestJournalPollMaxEventsAndAckMonotonic(t *testing.T) {
	root := t.TempDir()
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	j := newJournalBroker(backend, func() time.Time { return time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC) }, 16)

	watchID, startSeq, err := j.Subscribe(backend.RootNodeID(), true)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if startSeq != 0 {
		t.Fatalf("startSeq = %d, want 0", startSeq)
	}

	j.Append(protocol.WatchEvent{EventType: protocol.EventCreate, Path: "a.txt"})
	j.Append(protocol.WatchEvent{EventType: protocol.EventCreate, Path: "b.txt"})
	j.Append(protocol.WatchEvent{EventType: protocol.EventCreate, Path: "c.txt"})

	resp, err := j.Poll(watchID, 0, 2)
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if len(resp.Events) != 2 || resp.LatestSeq != 3 {
		t.Fatalf("poll len/latest = %d/%d, want 2/3", len(resp.Events), resp.LatestSeq)
	}

	acked, err := j.Ack(watchID, resp.Events[1].EventSeq)
	if err != nil {
		t.Fatalf("Ack(high) error = %v", err)
	}
	if acked != resp.Events[1].EventSeq {
		t.Fatalf("AckedSeq = %d, want %d", acked, resp.Events[1].EventSeq)
	}
	acked, err = j.Ack(watchID, 1)
	if err != nil {
		t.Fatalf("Ack(low) error = %v", err)
	}
	if acked != resp.Events[1].EventSeq {
		t.Fatalf("AckedSeq after lower ack = %d, want monotonic %d", acked, resp.Events[1].EventSeq)
	}
}

func TestJournalWatchNotFoundErrors(t *testing.T) {
	root := t.TempDir()
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	j := newJournalBroker(backend, time.Now, 8)

	if _, err := j.Poll(999999, 0, 16); err == nil || err.Error() != "watch not found" {
		t.Fatalf("Poll() error = %v, want watch not found", err)
	}
	if _, err := j.Ack(999999, 1); err == nil || err.Error() != "watch not found" {
		t.Fatalf("Ack() error = %v, want watch not found", err)
	}
	if _, err := j.Resync(999999); err == nil || err.Error() != "watch not found" {
		t.Fatalf("Resync() error = %v, want watch not found", err)
	}
}

func TestPathMatchesWatchVariants(t *testing.T) {
	tests := []struct {
		name      string
		root      string
		recursive bool
		path      string
		want      bool
	}{
		{name: "recursive-root-matches-nested", root: "", recursive: true, path: "a/b.txt", want: true},
		{name: "non-recursive-root-rejects-nested", root: "", recursive: false, path: "a/b.txt", want: false},
		{name: "non-recursive-root-allows-direct-child", root: "", recursive: false, path: "a.txt", want: true},
		{name: "recursive-subtree-matches-child", root: "src", recursive: true, path: "src/main.go", want: true},
		{name: "recursive-subtree-matches-grandchild", root: "src", recursive: true, path: "src/internal/x.go", want: true},
		{name: "non-recursive-subtree-rejects-grandchild", root: "src", recursive: false, path: "src/internal/x.go", want: false},
		{name: "old-path-cleaning", root: "./src/", recursive: true, path: "./src/main.go", want: true},
		{name: "different-subtree", root: "src", recursive: true, path: "docs/readme.md", want: false},
	}
	for _, tt := range tests {
		if got := pathMatchesWatch(tt.root, tt.recursive, tt.path); got != tt.want {
			t.Fatalf("%s: pathMatchesWatch(%q,%v,%q) = %v, want %v", tt.name, tt.root, tt.recursive, tt.path, got, tt.want)
		}
	}
}

func TestJournalResubscribeUnknownPreviousWatchUsesExplicitNode(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	j := newJournalBroker(backend, time.Now, 16)

	results, err := j.Resubscribe([]protocol.ResubscribeSpec{{
		PreviousWatchID: 7777,
		NodeID:          backend.RootNodeID(),
		Recursive:       false,
		AfterSeq:        5,
	}})
	if err != nil {
		t.Fatalf("Resubscribe() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Error != "" || results[0].WatchID == 0 {
		t.Fatalf("unexpected result = %+v", results[0])
	}
	if results[0].StartSeq != 5 || results[0].AckedSeq != 5 {
		t.Fatalf("start/acked = %d/%d, want 5/5", results[0].StartSeq, results[0].AckedSeq)
	}
}
