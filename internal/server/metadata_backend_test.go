package server

import (
	"os"
	"path/filepath"
	"testing"
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
