package server

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func benchmarkBackendWithFixture(b *testing.B) (*metadataBackend, uint64) {
	b.Helper()

	root := b.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		b.Fatalf("Mkdir(src) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "small.txt"), []byte("small-file-payload"), 0o644); err != nil {
		b.Fatalf("WriteFile(small.txt) error = %v", err)
	}
	backend, err := newMetadataBackend(root)
	if err != nil {
		b.Fatalf("newMetadataBackend() error = %v", err)
	}
	entry, err := backend.Lookup(backend.RootNodeID(), "small.txt")
	if err != nil {
		b.Fatalf("Lookup(small.txt) error = %v", err)
	}
	return backend, entry.NodeID
}

func BenchmarkMetadataLookupHot(b *testing.B) {
	backend, _ := benchmarkBackendWithFixture(b)
	if _, err := backend.Lookup(backend.RootNodeID(), "small.txt"); err != nil {
		b.Fatalf("warm Lookup() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := backend.Lookup(backend.RootNodeID(), "small.txt"); err != nil {
			b.Fatalf("Lookup() error = %v", err)
		}
	}
}

func BenchmarkMetadataGetAttrHot(b *testing.B) {
	backend, nodeID := benchmarkBackendWithFixture(b)
	if _, err := backend.GetAttr(nodeID); err != nil {
		b.Fatalf("warm GetAttr() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := backend.GetAttr(nodeID); err != nil {
			b.Fatalf("GetAttr() error = %v", err)
		}
	}
}

func BenchmarkMetadataReadDirSnapshotHit(b *testing.B) {
	backend, _ := benchmarkBackendWithFixture(b)
	if _, err := backend.snapshotDir(backend.RootNodeID()); err != nil {
		b.Fatalf("warm snapshotDir() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := backend.snapshotDir(backend.RootNodeID()); err != nil {
			b.Fatalf("snapshotDir() error = %v", err)
		}
	}
}

func BenchmarkMetadataReadSmallFileCached(b *testing.B) {
	backend, nodeID := benchmarkBackendWithFixture(b)
	handleID, _, err := backend.OpenFile(nodeID, false, false)
	if err != nil {
		b.Fatalf("OpenFile() error = %v", err)
	}
	defer func() { _ = backend.CloseHandle(handleID) }()
	if _, _, err := backend.ReadFile(handleID, 0, 64); err != nil {
		b.Fatalf("warm ReadFile() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := backend.ReadFile(handleID, 0, 64); err != nil {
			b.Fatalf("ReadFile() error = %v", err)
		}
	}
}

func BenchmarkMetadataCreateWriteFlushClose(b *testing.B) {
	root := b.TempDir()
	backend, err := newMetadataBackend(root)
	if err != nil {
		b.Fatalf("newMetadataBackend() error = %v", err)
	}
	payload := []byte("benchmark-payload")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("bench-%d.txt", i)
		_, handleID, err := backend.CreateFile(backend.RootNodeID(), name, false)
		if err != nil {
			b.Fatalf("CreateFile() error = %v", err)
		}
		if _, _, err := backend.WriteFile(handleID, 0, payload); err != nil {
			b.Fatalf("WriteFile() error = %v", err)
		}
		if err := backend.FlushHandle(handleID); err != nil {
			b.Fatalf("FlushHandle() error = %v", err)
		}
		if err := backend.CloseHandle(handleID); err != nil {
			b.Fatalf("CloseHandle() error = %v", err)
		}
	}
}
