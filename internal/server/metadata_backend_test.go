package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetadataBackendReadOnlyFlow(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello world"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}

	entry, err := backend.Lookup(backend.RootNodeID(), "hello.txt")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if entry.Size != int64(len("hello world")) {
		t.Fatalf("unexpected size: got %d", entry.Size)
	}

	handleID, _, err := backend.OpenFile(entry.NodeID)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer func() { _ = backend.CloseHandle(handleID) }()

	data, eof, err := backend.ReadFile(handleID, 0, 5)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "hello" || eof {
		t.Fatalf("unexpected read result: data=%q eof=%v", string(data), eof)
	}

	cursorID, err := backend.OpenDir(backend.RootNodeID())
	if err != nil {
		t.Fatalf("OpenDir() error = %v", err)
	}
	resp, err := backend.ReadDir(cursorID, 0, 10)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("unexpected directory entry count: got %d want 2", len(resp.Entries))
	}
}

func TestMetadataBackendAttrCacheRefreshesAfterTTL(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "hello.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	fakeNow := time.Date(2026, 3, 15, 3, 0, 0, 0, time.UTC)
	backend.now = func() time.Time { return fakeNow }
	backend.cacheTTL = time.Second

	entry, err := backend.Lookup(backend.RootNodeID(), "hello.txt")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	firstStats := backend.Stats()
	if firstStats.AttrMisses == 0 {
		t.Fatalf("expected at least one attr miss on first lookup")
	}

	cached, err := backend.GetAttr(entry.NodeID)
	if err != nil {
		t.Fatalf("GetAttr() error = %v", err)
	}
	if cached.Size != int64(len("hello")) {
		t.Fatalf("cached size = %d, want %d", cached.Size, len("hello"))
	}
	statsAfterHit := backend.Stats()
	if statsAfterHit.AttrHits == 0 {
		t.Fatalf("expected attr cache hit after repeated getattr")
	}

	if err := os.WriteFile(filePath, []byte("hello, iter3"), 0o644); err != nil {
		t.Fatalf("WriteFile(update) error = %v", err)
	}
	beforeTTL, err := backend.GetAttr(entry.NodeID)
	if err != nil {
		t.Fatalf("GetAttr(before ttl) error = %v", err)
	}
	if beforeTTL.Size != int64(len("hello")) {
		t.Fatalf("before ttl size = %d, want cached size %d", beforeTTL.Size, len("hello"))
	}

	fakeNow = fakeNow.Add(2 * time.Second)
	refreshed, err := backend.GetAttr(entry.NodeID)
	if err != nil {
		t.Fatalf("GetAttr(after ttl) error = %v", err)
	}
	if refreshed.Size != int64(len("hello, iter3")) {
		t.Fatalf("refreshed size = %d, want %d", refreshed.Size, len("hello, iter3"))
	}
}

func TestMetadataBackendNegativeCacheExpires(t *testing.T) {
	root := t.TempDir()
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	fakeNow := time.Date(2026, 3, 15, 4, 0, 0, 0, time.UTC)
	backend.now = func() time.Time { return fakeNow }
	backend.cacheTTL = time.Second

	if _, err := backend.Lookup(backend.RootNodeID(), "missing.txt"); err == nil {
		t.Fatalf("expected first missing lookup to fail")
	}
	if _, err := backend.Lookup(backend.RootNodeID(), "missing.txt"); err == nil {
		t.Fatalf("expected negative-cached lookup to still fail")
	}
	stats := backend.Stats()
	if stats.NegativeHits == 0 {
		t.Fatalf("expected negative cache hit")
	}

	missingPath := filepath.Join(root, "missing.txt")
	if err := os.WriteFile(missingPath, []byte("now-present"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := backend.Lookup(backend.RootNodeID(), "missing.txt"); err == nil {
		t.Fatalf("expected negative cache to hide new file until ttl expiry")
	}

	fakeNow = fakeNow.Add(2 * time.Second)
	entry, err := backend.Lookup(backend.RootNodeID(), "missing.txt")
	if err != nil {
		t.Fatalf("Lookup(after ttl) error = %v", err)
	}
	if entry.Name != "missing.txt" {
		t.Fatalf("entry name = %q, want missing.txt", entry.Name)
	}
}

func TestMetadataBackendDirSnapshotCacheAndRootPrefetch(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	fakeNow := time.Date(2026, 3, 15, 5, 0, 0, 0, time.UTC)
	backend.now = func() time.Time { return fakeNow }
	backend.cacheTTL = time.Second

	stats := backend.Stats()
	if stats.RootPrefetches == 0 {
		t.Fatalf("expected root prefetch to run during initialization")
	}
	if err := backend.prefetchRoot(); err != nil {
		t.Fatalf("prefetchRoot() error = %v", err)
	}

	cursorID, err := backend.OpenDir(backend.RootNodeID())
	if err != nil {
		t.Fatalf("OpenDir() error = %v", err)
	}
	listing, err := backend.ReadDir(cursorID, 0, 10)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(listing.Entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(listing.Entries))
	}
	stats = backend.Stats()
	if stats.DirSnapshotHits == 0 {
		t.Fatalf("expected prefetched dir snapshot hit")
	}

	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatalf("WriteFile(new) error = %v", err)
	}
	cursorID, err = backend.OpenDir(backend.RootNodeID())
	if err != nil {
		t.Fatalf("OpenDir(cached) error = %v", err)
	}
	listing, err = backend.ReadDir(cursorID, 0, 10)
	if err != nil {
		t.Fatalf("ReadDir(cached) error = %v", err)
	}
	if len(listing.Entries) != 1 {
		t.Fatalf("cached entry count = %d, want 1", len(listing.Entries))
	}

	fakeNow = fakeNow.Add(2 * time.Second)
	cursorID, err = backend.OpenDir(backend.RootNodeID())
	if err != nil {
		t.Fatalf("OpenDir(refreshed) error = %v", err)
	}
	listing, err = backend.ReadDir(cursorID, 0, 10)
	if err != nil {
		t.Fatalf("ReadDir(refreshed) error = %v", err)
	}
	if len(listing.Entries) != 2 {
		t.Fatalf("refreshed entry count = %d, want 2", len(listing.Entries))
	}
}
