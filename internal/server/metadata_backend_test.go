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

	handleID, _, err := backend.OpenFile(entry.NodeID, false, false)
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

func TestMetadataBackendWriteAndFlushFlow(t *testing.T) {
	root := t.TempDir()
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}

	createResp, handleID, err := backend.CreateFile(backend.RootNodeID(), "notes.txt", false)
	if err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}
	if createResp.Name != "notes.txt" {
		t.Fatalf("create name = %q, want notes.txt", createResp.Name)
	}

	writeResp, newSize, err := backend.WriteFile(handleID, 0, []byte("iter4-data"))
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if writeResp != len("iter4-data") || newSize != int64(len("iter4-data")) {
		t.Fatalf("unexpected write result: written=%d size=%d", writeResp, newSize)
	}
	if err := backend.FlushHandle(handleID); err != nil {
		t.Fatalf("FlushHandle() error = %v", err)
	}
	if err := backend.CloseHandle(handleID); err != nil {
		t.Fatalf("CloseHandle() error = %v", err)
	}

	entry, err := backend.Lookup(backend.RootNodeID(), "notes.txt")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if entry.Size != int64(len("iter4-data")) {
		t.Fatalf("entry size = %d, want %d", entry.Size, len("iter4-data"))
	}

	readHandle, _, err := backend.OpenFile(entry.NodeID, false, false)
	if err != nil {
		t.Fatalf("OpenFile(read) error = %v", err)
	}
	defer func() { _ = backend.CloseHandle(readHandle) }()
	data, eof, err := backend.ReadFile(readHandle, 0, 64)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "iter4-data" || !eof {
		t.Fatalf("unexpected persisted data: %q eof=%v", string(data), eof)
	}
}

func TestMetadataBackendTruncateRenameAndDeleteOnClose(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello world"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}

	entry, err := backend.Lookup(backend.RootNodeID(), "hello.txt")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	writeHandle, _, err := backend.OpenFile(entry.NodeID, true, true)
	if err != nil {
		t.Fatalf("OpenFile(write) error = %v", err)
	}
	if _, size, err := backend.WriteFile(writeHandle, 0, []byte("abc")); err != nil || size != 3 {
		t.Fatalf("WriteFile() error=%v size=%d", err, size)
	}
	if size, err := backend.TruncateHandle(writeHandle, 2); err != nil || size != 2 {
		t.Fatalf("TruncateHandle() error=%v size=%d", err, size)
	}
	if err := backend.FlushHandle(writeHandle); err != nil {
		t.Fatalf("FlushHandle() error = %v", err)
	}
	if err := backend.CloseHandle(writeHandle); err != nil {
		t.Fatalf("CloseHandle() error = %v", err)
	}

	entry, err = backend.Lookup(backend.RootNodeID(), "hello.txt")
	if err != nil {
		t.Fatalf("Lookup(after truncate) error = %v", err)
	}
	if entry.Size != 2 {
		t.Fatalf("size after truncate = %d, want 2", entry.Size)
	}

	_, tmpHandle, err := backend.CreateFile(backend.RootNodeID(), ".hello.tmp", false)
	if err != nil {
		t.Fatalf("CreateFile(tmp) error = %v", err)
	}
	if _, _, err := backend.WriteFile(tmpHandle, 0, []byte("replaced")); err != nil {
		t.Fatalf("WriteFile(tmp) error = %v", err)
	}
	if err := backend.FlushHandle(tmpHandle); err != nil {
		t.Fatalf("FlushHandle(tmp) error = %v", err)
	}
	if err := backend.CloseHandle(tmpHandle); err != nil {
		t.Fatalf("CloseHandle(tmp) error = %v", err)
	}
	if _, err := backend.RenamePath(backend.RootNodeID(), ".hello.tmp", backend.RootNodeID(), "hello.txt", true); err != nil {
		t.Fatalf("RenamePath() error = %v", err)
	}

	entry, err = backend.Lookup(backend.RootNodeID(), "hello.txt")
	if err != nil {
		t.Fatalf("Lookup(after rename) error = %v", err)
	}
	readHandle, _, err := backend.OpenFile(entry.NodeID, false, false)
	if err != nil {
		t.Fatalf("OpenFile(renamed) error = %v", err)
	}
	data, _, err := backend.ReadFile(readHandle, 0, 64)
	if err != nil {
		t.Fatalf("ReadFile(renamed) error = %v", err)
	}
	_ = backend.CloseHandle(readHandle)
	if string(data) != "replaced" {
		t.Fatalf("renamed file data = %q, want replaced", string(data))
	}

	_, deleteHandle, err := backend.CreateFile(backend.RootNodeID(), "delete-me.tmp", false)
	if err != nil {
		t.Fatalf("CreateFile(delete) error = %v", err)
	}
	if err := backend.SetDeleteOnClose(deleteHandle, true); err != nil {
		t.Fatalf("SetDeleteOnClose() error = %v", err)
	}
	if err := backend.CloseHandle(deleteHandle); err != nil {
		t.Fatalf("CloseHandle(delete) error = %v", err)
	}
	if _, err := backend.Lookup(backend.RootNodeID(), "delete-me.tmp"); !isNotExist(err) {
		t.Fatalf("expected delete-on-close file to be absent, got err=%v", err)
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
	backend.mu.Lock()
	backend.attrCache = map[string]attrCacheEntry{}
	backend.dirSnapshots = map[uint64]dirSnapshotEntry{}
	backend.negativeCache = map[string]negativeCacheEntry{}
	backend.smallFileCache = map[string]smallFileCacheEntry{}
	backend.mu.Unlock()
	backend.mu.Lock()
	backend.attrCache = map[string]attrCacheEntry{}
	backend.dirSnapshots = map[uint64]dirSnapshotEntry{}
	backend.negativeCache = map[string]negativeCacheEntry{}
	backend.smallFileCache = map[string]smallFileCacheEntry{}
	backend.mu.Unlock()
	backend.mu.Lock()
	backend.attrCache = map[string]attrCacheEntry{}
	backend.dirSnapshots = map[uint64]dirSnapshotEntry{}
	backend.negativeCache = map[string]negativeCacheEntry{}
	backend.smallFileCache = map[string]smallFileCacheEntry{}
	backend.mu.Unlock()

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

func TestMetadataBackendSmallFileCacheAndInvalidation(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"demo"}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	entry, err := backend.Lookup(backend.RootNodeID(), "package.json")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	h, _, err := backend.OpenFile(entry.NodeID, false, false)
	if err != nil {
		t.Fatalf("OpenFile(read) error = %v", err)
	}
	data, eof, err := backend.ReadFile(h, 0, 64)
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}
	if string(data) != `{"name":"demo"}` || !eof {
		t.Fatalf("unexpected first read result %q eof=%v", string(data), eof)
	}
	if _, _, err := backend.ReadFile(h, 0, 64); err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	_ = backend.CloseHandle(h)
	stats := backend.Stats()
	if stats.SmallFileHits == 0 {
		t.Fatalf("expected small-file cache hit after repeated read")
	}

	wh, _, err := backend.OpenFile(entry.NodeID, true, true)
	if err != nil {
		t.Fatalf("OpenFile(write) error = %v", err)
	}
	if _, _, err := backend.WriteFile(wh, 0, []byte(`{"name":"demo2"}`)); err != nil {
		t.Fatalf("WriteFile(update) error = %v", err)
	}
	if err := backend.FlushHandle(wh); err != nil {
		t.Fatalf("FlushHandle(update) error = %v", err)
	}
	if err := backend.CloseHandle(wh); err != nil {
		t.Fatalf("CloseHandle(update) error = %v", err)
	}
	rh, _, err := backend.OpenFile(entry.NodeID, false, false)
	if err != nil {
		t.Fatalf("OpenFile(re-read) error = %v", err)
	}
	data, eof, err = backend.ReadFile(rh, 0, 64)
	if err != nil {
		t.Fatalf("ReadFile(after update) error = %v", err)
	}
	_ = backend.CloseHandle(rh)
	if string(data) != `{"name":"demo2"}` || !eof {
		t.Fatalf("unexpected updated read result %q eof=%v", string(data), eof)
	}
}

func TestMetadataBackendWorkspaceProfileAndPrefetchPriority(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"demo"}`), 0o644); err != nil {
		t.Fatalf("WriteFile(package.json) error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("Mkdir(src) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.ts"), []byte("export const x = 1;"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.ts) error = %v", err)
	}
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	profile := backend.Profile()
	if !profile.IsHotDir("src") {
		t.Fatalf("expected src to be a hot dir")
	}
	if !profile.IsHotFile("package.json") {
		t.Fatalf("expected package.json to be a hot file")
	}
	stats := backend.Stats()
	if stats.RootPrefetches == 0 {
		t.Fatalf("expected root prefetch to run")
	}
	if stats.HotDirPrefetches == 0 {
		t.Fatalf("expected hot dir prefetches")
	}
	if stats.HotFilePrefetches == 0 {
		t.Fatalf("expected hot file prefetches")
	}
	if stats.HighPriorityPrefetches == 0 {
		t.Fatalf("expected high priority prefetch work")
	}
	if stats.NormalPriorityPrefetches == 0 {
		t.Fatalf("expected normal priority prefetch work")
	}
	entry, err := backend.Lookup(backend.RootNodeID(), "package.json")
	if err != nil {
		t.Fatalf("Lookup(package.json) error = %v", err)
	}
	h, _, err := backend.OpenFile(entry.NodeID, false, false)
	if err != nil {
		t.Fatalf("OpenFile(package.json) error = %v", err)
	}
	if _, _, err := backend.ReadFile(h, 0, 64); err != nil {
		t.Fatalf("ReadFile(package.json) error = %v", err)
	}
	_ = backend.CloseHandle(h)
	stats = backend.Stats()
	if stats.SmallFileHits == 0 {
		t.Fatalf("expected prefetched small file cache to be used")
	}
	srcEntry, err := backend.Lookup(backend.RootNodeID(), "src")
	if err != nil {
		t.Fatalf("Lookup(src) error = %v", err)
	}
	cursorID, err := backend.OpenDir(srcEntry.NodeID)
	if err != nil {
		t.Fatalf("OpenDir(src) error = %v", err)
	}
	listing, err := backend.ReadDir(cursorID, 0, 10)
	if err != nil {
		t.Fatalf("ReadDir(src) error = %v", err)
	}
	if len(listing.Entries) != 1 || listing.Entries[0].Name != "main.ts" {
		t.Fatalf("unexpected src listing: %+v", listing.Entries)
	}
	stats = backend.Stats()
	if stats.DirSnapshotHits == 0 {
		t.Fatalf("expected prefetched dir snapshot hit for src")
	}
}
