package server

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestMetadataBackendCreateOverwriteAndAlreadyExists(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "dup.txt")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}

	if _, _, err := backend.CreateFile(backend.RootNodeID(), "dup.txt", false); !errors.Is(err, errAlreadyExists) {
		t.Fatalf("CreateFile(no-overwrite) error = %v, want errAlreadyExists", err)
	}

	entry, handleID, err := backend.CreateFile(backend.RootNodeID(), "dup.txt", true)
	if err != nil {
		t.Fatalf("CreateFile(overwrite) error = %v", err)
	}
	if entry.Name != "dup.txt" {
		t.Fatalf("entry name = %q, want dup.txt", entry.Name)
	}
	if err := backend.CloseHandle(handleID); err != nil {
		t.Fatalf("CloseHandle() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(path) error = %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("overwritten file size = %d, want 0", len(data))
	}
}

func TestMetadataBackendOpenDirFileAndOpenFileDirErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}

	fileNode, err := backend.Lookup(backend.RootNodeID(), "hello.txt")
	if err != nil {
		t.Fatalf("Lookup(file) error = %v", err)
	}
	if _, err := backend.OpenDir(fileNode.NodeID); !isNotDir(err) {
		t.Fatalf("OpenDir(file) error = %v, want not-dir", err)
	}

	dirNode, err := backend.Lookup(backend.RootNodeID(), "nested")
	if err != nil {
		t.Fatalf("Lookup(dir) error = %v", err)
	}
	if _, _, err := backend.OpenFile(dirNode.NodeID, false, false); !isDir(err) {
		t.Fatalf("OpenFile(dir) error = %v, want is-dir", err)
	}
}

func TestMetadataBackendInvalidHandleAndReadOnlyWriteErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}

	fileNode, err := backend.Lookup(backend.RootNodeID(), "hello.txt")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	readHandle, _, err := backend.OpenFile(fileNode.NodeID, false, false)
	if err != nil {
		t.Fatalf("OpenFile(read-only) error = %v", err)
	}
	defer func() { _ = backend.CloseHandle(readHandle) }()

	if _, _, err := backend.WriteFile(readHandle, 0, []byte("x")); !errors.Is(err, errAccessDenied) {
		t.Fatalf("WriteFile(read-only) error = %v, want errAccessDenied", err)
	}
	if _, _, err := backend.ReadFile(999999, 0, 16); !errors.Is(err, errInvalidHandle) {
		t.Fatalf("ReadFile(invalid) error = %v, want errInvalidHandle", err)
	}
	if err := backend.FlushHandle(999999); !errors.Is(err, errInvalidHandle) {
		t.Fatalf("FlushHandle(invalid) error = %v, want errInvalidHandle", err)
	}
	if _, err := backend.TruncateHandle(999999, 1); !errors.Is(err, errInvalidHandle) {
		t.Fatalf("TruncateHandle(invalid) error = %v, want errInvalidHandle", err)
	}
	if err := backend.CloseHandle(999999); !errors.Is(err, errInvalidHandle) {
		t.Fatalf("CloseHandle(invalid) error = %v, want errInvalidHandle", err)
	}
}

func TestMetadataBackendReadDirPaginationAndCursorBounds(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"c.txt", "a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}

	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}

	cursorID, err := backend.OpenDir(backend.RootNodeID())
	if err != nil {
		t.Fatalf("OpenDir() error = %v", err)
	}

	page1, err := backend.ReadDir(cursorID, 0, 2)
	if err != nil {
		t.Fatalf("ReadDir(page1) error = %v", err)
	}
	if len(page1.Entries) != 2 || page1.EOF {
		t.Fatalf("page1 entries=%d eof=%v, want 2,false", len(page1.Entries), page1.EOF)
	}
	if got := []string{page1.Entries[0].Name, page1.Entries[1].Name}; got[0] != "a.txt" || got[1] != "b.txt" {
		t.Fatalf("page1 names = %v, want [a.txt b.txt]", got)
	}

	page2, err := backend.ReadDir(cursorID, page1.NextCookie, 2)
	if err != nil {
		t.Fatalf("ReadDir(page2) error = %v", err)
	}
	if len(page2.Entries) != 1 || !page2.EOF || page2.Entries[0].Name != "c.txt" {
		t.Fatalf("page2 = %+v, want single c.txt and EOF", page2)
	}

	page3, err := backend.ReadDir(cursorID, 99, 2)
	if err != nil {
		t.Fatalf("ReadDir(page3) error = %v", err)
	}
	if len(page3.Entries) != 0 || !page3.EOF {
		t.Fatalf("page3 entries=%d eof=%v, want 0,true", len(page3.Entries), page3.EOF)
	}

	if _, err := backend.ReadDir(999999, 0, 1); !os.IsNotExist(err) {
		t.Fatalf("ReadDir(invalid cursor) error = %v, want os.ErrNotExist", err)
	}
}

func TestMetadataBackendSparseWriteAndCrossDirectoryRename(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("Mkdir(src) error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "dst"), 0o755); err != nil {
		t.Fatalf("Mkdir(dst) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "dst", "existing.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing) error = %v", err)
	}

	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}

	srcDir, err := backend.Lookup(backend.RootNodeID(), "src")
	if err != nil {
		t.Fatalf("Lookup(src) error = %v", err)
	}
	dstDir, err := backend.Lookup(backend.RootNodeID(), "dst")
	if err != nil {
		t.Fatalf("Lookup(dst) error = %v", err)
	}

	_, handleID, err := backend.CreateFile(srcDir.NodeID, "sparse.bin", false)
	if err != nil {
		t.Fatalf("CreateFile(sparse.bin) error = %v", err)
	}
	if _, size, err := backend.WriteFile(handleID, 5, []byte("xy")); err != nil || size != 7 {
		t.Fatalf("WriteFile(sparse) error=%v size=%d, want size 7", err, size)
	}
	if err := backend.FlushHandle(handleID); err != nil {
		t.Fatalf("FlushHandle(sparse) error = %v", err)
	}
	if err := backend.CloseHandle(handleID); err != nil {
		t.Fatalf("CloseHandle(sparse) error = %v", err)
	}

	updatedEntry, err := backend.Lookup(srcDir.NodeID, "sparse.bin")
	if err != nil {
		t.Fatalf("Lookup(updated sparse.bin) error = %v", err)
	}
	readHandle, _, err := backend.OpenFile(updatedEntry.NodeID, false, false)
	if err != nil {
		t.Fatalf("OpenFile(sparse read) error = %v", err)
	}
	defer func() { _ = backend.CloseHandle(readHandle) }()
	data, eof, err := backend.ReadFile(readHandle, 0, 16)
	if err != nil {
		t.Fatalf("ReadFile(sparse) error = %v", err)
	}
	if !eof || len(data) != 7 || data[5] != 'x' || data[6] != 'y' {
		t.Fatalf("sparse read = %v eof=%v, want len=7 ..xy", data, eof)
	}
	for i := 0; i < 5; i++ {
		if data[i] != 0 {
			t.Fatalf("sparse hole byte[%d] = %d, want 0", i, data[i])
		}
	}

	if _, err := backend.RenamePath(srcDir.NodeID, "sparse.bin", dstDir.NodeID, "existing.txt", false); !errors.Is(err, errAlreadyExists) {
		t.Fatalf("RenamePath(no replace) error = %v, want errAlreadyExists", err)
	}
	renamed, err := backend.RenamePath(srcDir.NodeID, "sparse.bin", dstDir.NodeID, "existing.txt", true)
	if err != nil {
		t.Fatalf("RenamePath(replace) error = %v", err)
	}
	if renamed.ParentNodeID != dstDir.NodeID || renamed.Name != "existing.txt" {
		t.Fatalf("renamed entry = %+v, want dst/existing.txt", renamed)
	}
	if _, err := backend.Lookup(srcDir.NodeID, "sparse.bin"); !os.IsNotExist(err) {
		t.Fatalf("Lookup(old path) error = %v, want os.ErrNotExist", err)
	}
}
