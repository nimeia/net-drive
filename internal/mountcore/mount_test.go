package mountcore

import (
	"fmt"
	"strings"
	"testing"

	"developer-mount/internal/protocol"
)

type fakeClient struct {
	nodes             map[uint64]protocol.NodeInfo
	children          map[uint64][]protocol.DirEntry
	lookupCount       int
	getAttrCount      int
	openDirCount      int
	readDirCount      int
	openReadCount     int
	readCount         int
	closeHandleCount  int
	closedHandles     []uint64
	nextDirCursorID   uint64
	nextHandleID      uint64
	fileData          map[uint64][]byte
	dirCursorToNodeID map[uint64]uint64
	handleToNodeID    map[uint64]uint64
}

func newFakeClient() *fakeClient {
	nodes := map[uint64]protocol.NodeInfo{1: {NodeID: 1, ParentNodeID: 1, Name: "", FileType: protocol.FileTypeDirectory}, 2: {NodeID: 2, ParentNodeID: 1, Name: "src", FileType: protocol.FileTypeDirectory}, 3: {NodeID: 3, ParentNodeID: 2, Name: "main.go", FileType: protocol.FileTypeFile, Size: 12}, 4: {NodeID: 4, ParentNodeID: 1, Name: "README.md", FileType: protocol.FileTypeFile, Size: 9}}
	children := map[uint64][]protocol.DirEntry{1: {{NodeID: 2, Name: "src", FileType: protocol.FileTypeDirectory}, {NodeID: 4, Name: "README.md", FileType: protocol.FileTypeFile, Size: 9}}, 2: {{NodeID: 3, Name: "main.go", FileType: protocol.FileTypeFile, Size: 12}}}
	return &fakeClient{nodes: nodes, children: children, nextDirCursorID: 100, nextHandleID: 200, fileData: map[uint64][]byte{3: []byte("hello world!"), 4: []byte("readme.md")}, dirCursorToNodeID: map[uint64]uint64{}, handleToNodeID: map[uint64]uint64{}}
}
func (f *fakeClient) Lookup(parentNodeID uint64, name string) (protocol.LookupResp, error) {
	f.lookupCount++
	for _, entry := range f.children[parentNodeID] {
		if strings.EqualFold(entry.Name, name) {
			return protocol.LookupResp{Entry: f.nodes[entry.NodeID]}, nil
		}
	}
	return protocol.LookupResp{}, fmt.Errorf("%s: not found", protocol.ErrNotFound)
}
func (f *fakeClient) GetAttr(nodeID uint64) (protocol.GetAttrResp, error) {
	f.getAttrCount++
	node, ok := f.nodes[nodeID]
	if !ok {
		return protocol.GetAttrResp{}, fmt.Errorf("%s: not found", protocol.ErrNotFound)
	}
	return protocol.GetAttrResp{Entry: node}, nil
}
func (f *fakeClient) OpenDir(nodeID uint64) (protocol.OpenDirResp, error) {
	f.openDirCount++
	f.nextDirCursorID++
	f.dirCursorToNodeID[f.nextDirCursorID] = nodeID
	return protocol.OpenDirResp{DirCursorID: f.nextDirCursorID}, nil
}
func (f *fakeClient) ReadDir(dirCursorID uint64, cookie uint64, maxEntries uint32) (protocol.ReadDirResp, error) {
	f.readDirCount++
	nodeID := f.dirCursorToNodeID[dirCursorID]
	entries := f.children[nodeID]
	start := int(cookie)
	if start >= len(entries) {
		return protocol.ReadDirResp{EOF: true, NextCookie: cookie}, nil
	}
	end := start + int(maxEntries)
	if end > len(entries) {
		end = len(entries)
	}
	page := entries[start:end]
	return protocol.ReadDirResp{Entries: page, NextCookie: uint64(end), EOF: end >= len(entries)}, nil
}
func (f *fakeClient) OpenRead(nodeID uint64) (protocol.OpenResp, error) {
	f.openReadCount++
	f.nextHandleID++
	f.handleToNodeID[f.nextHandleID] = nodeID
	return protocol.OpenResp{HandleID: f.nextHandleID, Size: f.nodes[nodeID].Size}, nil
}
func (f *fakeClient) Read(handleID uint64, offset int64, length uint32) (protocol.ReadResp, error) {
	f.readCount++
	nodeID := f.handleToNodeID[handleID]
	data := f.fileData[nodeID]
	if offset >= int64(len(data)) {
		return protocol.ReadResp{Offset: offset, EOF: true}, nil
	}
	start := int(offset)
	end := start + int(length)
	if end > len(data) {
		end = len(data)
	}
	return protocol.ReadResp{Data: data[start:end], Offset: offset, EOF: end >= len(data)}, nil
}
func (f *fakeClient) CloseHandle(handleID uint64) (protocol.CloseResp, error) {
	f.closeHandleCount++
	f.closedHandles = append(f.closedHandles, handleID)
	return protocol.CloseResp{Closed: true}, nil
}

func TestLookupAndGetAttrUsePathCache(t *testing.T) {
	client := newFakeClient()
	mount := New(client, Options{RootNodeID: 1, VolumeName: "workspace", ReadOnly: true})
	info, err := mount.Lookup(`\\src\\main.go`)
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if info.Path != "/src/main.go" || info.NodeID != 3 {
		t.Fatalf("Lookup() = %+v", info)
	}
	if client.lookupCount != 2 {
		t.Fatalf("lookupCount = %d, want 2", client.lookupCount)
	}
	info, err = mount.GetAttr(`/src/main.go`)
	if err != nil {
		t.Fatalf("GetAttr() error = %v", err)
	}
	if info.Size != 12 || info.IsDirectory {
		t.Fatalf("GetAttr() = %+v", info)
	}
	if client.lookupCount != 2 {
		t.Fatalf("lookupCount changed = %d, want cached 2", client.lookupCount)
	}
	if client.getAttrCount != 1 {
		t.Fatalf("getAttrCount = %d, want 1", client.getAttrCount)
	}
}
func TestOpenReadAndClose(t *testing.T) {
	client := newFakeClient()
	mount := New(client, Options{RootNodeID: 1, ReadOnly: true})
	handle, err := mount.Open(`/README.md`)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if handle.HandleID == 0 || handle.RemoteHandleID == 0 || handle.Info.Path != "/README.md" {
		t.Fatalf("Open() = %+v", handle)
	}
	readResp, err := mount.Read(handle.HandleID, 0, 5)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if string(readResp.Data) != "readm" {
		t.Fatalf("Read().Data = %q, want %q", string(readResp.Data), "readm")
	}
	if err := mount.Close(handle.HandleID); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if client.closeHandleCount != 1 || len(client.closedHandles) != 1 {
		t.Fatalf("closed handles = %v", client.closedHandles)
	}
}
func TestOpenDirectoryReadDirectoryAndClose(t *testing.T) {
	client := newFakeClient()
	mount := New(client, Options{RootNodeID: 1, ReadOnly: true, ReadDirBatchSize: 1})
	handle, err := mount.OpenDirectory(`/`)
	if err != nil {
		t.Fatalf("OpenDirectory() error = %v", err)
	}
	page, err := mount.ReadDirectory(handle.HandleID, 0, 1)
	if err != nil {
		t.Fatalf("ReadDirectory() error = %v", err)
	}
	if len(page.Entries) != 1 || page.Entries[0].Path != "/src" || page.EOF {
		t.Fatalf("ReadDirectory() page = %+v", page)
	}
	if err := mount.Close(handle.HandleID); err != nil {
		t.Fatalf("Close(dir) error = %v", err)
	}
	if client.closeHandleCount != 0 {
		t.Fatalf("directory close should not call remote close, got %d", client.closeHandleCount)
	}
}
func TestOpenDirectoryRejectsFileAndOpenRejectsDirectory(t *testing.T) {
	client := newFakeClient()
	mount := New(client, Options{RootNodeID: 1, ReadOnly: true})
	if _, err := mount.OpenDirectory(`/README.md`); err != ErrNotDirectory {
		t.Fatalf("OpenDirectory(file) error = %v, want %v", err, ErrNotDirectory)
	}
	if _, err := mount.Open(`/src`); err != ErrIsDirectory {
		t.Fatalf("Open(directory) error = %v, want %v", err, ErrIsDirectory)
	}
}
