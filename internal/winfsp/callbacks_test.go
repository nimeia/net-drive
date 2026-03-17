package winfsp

import (
	"developer-mount/internal/mountcore"
	"developer-mount/internal/protocol"
	adapterpkg "developer-mount/internal/winfsp/adapter"
	"errors"
	"testing"
)

type callbackFakeClient struct{}

func (callbackFakeClient) Lookup(parentNodeID uint64, name string) (protocol.LookupResp, error) {
	switch name {
	case "file.txt":
		return protocol.LookupResp{Entry: protocol.NodeInfo{NodeID: 2, ParentNodeID: 1, Name: name, FileType: protocol.FileTypeFile, Size: 4}}, nil
	case "dir":
		return protocol.LookupResp{Entry: protocol.NodeInfo{NodeID: 3, ParentNodeID: 1, Name: name, FileType: protocol.FileTypeDirectory}}, nil
	default:
		return protocol.LookupResp{}, errors.New("ERR_NOT_FOUND: not found")
	}
}
func (callbackFakeClient) GetAttr(nodeID uint64) (protocol.GetAttrResp, error) {
	switch nodeID {
	case 1:
		return protocol.GetAttrResp{Entry: protocol.NodeInfo{NodeID: 1, ParentNodeID: 1, FileType: protocol.FileTypeDirectory}}, nil
	case 2:
		return protocol.GetAttrResp{Entry: protocol.NodeInfo{NodeID: 2, ParentNodeID: 1, Name: "file.txt", FileType: protocol.FileTypeFile, Size: 4}}, nil
	case 3:
		return protocol.GetAttrResp{Entry: protocol.NodeInfo{NodeID: 3, ParentNodeID: 1, Name: "dir", FileType: protocol.FileTypeDirectory}}, nil
	default:
		return protocol.GetAttrResp{}, errors.New("ERR_NOT_FOUND: not found")
	}
}
func (callbackFakeClient) OpenDir(nodeID uint64) (protocol.OpenDirResp, error) {
	return protocol.OpenDirResp{DirCursorID: 10}, nil
}
func (callbackFakeClient) ReadDir(dirCursorID uint64, cookie uint64, maxEntries uint32) (protocol.ReadDirResp, error) {
	return protocol.ReadDirResp{Entries: []protocol.DirEntry{{NodeID: 2, Name: "file.txt", FileType: protocol.FileTypeFile, Size: 4}}, NextCookie: 1, EOF: true}, nil
}
func (callbackFakeClient) OpenRead(nodeID uint64) (protocol.OpenResp, error) {
	return protocol.OpenResp{HandleID: 20, Size: 4}, nil
}
func (callbackFakeClient) Read(handleID uint64, offset int64, length uint32) (protocol.ReadResp, error) {
	return protocol.ReadResp{Data: []byte("data"), Offset: offset, EOF: true}, nil
}
func (callbackFakeClient) CloseHandle(handleID uint64) (protocol.CloseResp, error) {
	return protocol.CloseResp{Closed: true}, nil
}

func TestCallbacksMapReadOnlyFlow(t *testing.T) {
	mount := mountcore.New(callbackFakeClient{}, mountcore.Options{RootNodeID: 1, ReadOnly: true})
	callbacks := NewCallbacks(adapterpkg.New(mount))
	if _, status := callbacks.GetFileInfo(`/missing.txt`); status != StatusObjectNameNotFound {
		t.Fatalf("GetFileInfo missing status = 0x%08x, want 0x%08x", uint32(status), uint32(StatusObjectNameNotFound))
	}
	fileOpen, status := callbacks.Open(`/file.txt`)
	if status != StatusSuccess {
		t.Fatalf("Open(file) status = 0x%08x", uint32(status))
	}
	data, eof, status := callbacks.Read(fileOpen.HandleID, 0, 4)
	if status != StatusSuccess || string(data) != "data" || !eof {
		t.Fatalf("Read() = data=%q eof=%v status=0x%08x", string(data), eof, uint32(status))
	}
	if status := callbacks.Close(fileOpen.HandleID); status != StatusSuccess {
		t.Fatalf("Close(file) status = 0x%08x", uint32(status))
	}
	if _, status := callbacks.Open(`/dir`); status != StatusFileIsADirectory {
		t.Fatalf("Open(dir) status = 0x%08x, want dir status", uint32(status))
	}
	dirOpen, status := callbacks.OpenDirectory(`/dir`)
	if status != StatusSuccess {
		t.Fatalf("OpenDirectory(dir) status = 0x%08x", uint32(status))
	}
	page, status := callbacks.ReadDirectory(dirOpen.HandleID, 0, 16)
	if status != StatusSuccess || len(page.Entries) != 1 || page.Entries[0].Path != "/dir/file.txt" {
		t.Fatalf("ReadDirectory() = %+v status=0x%08x", page, uint32(status))
	}
	if status := callbacks.Close(dirOpen.HandleID); status != StatusSuccess {
		t.Fatalf("Close(dir) status = 0x%08x", uint32(status))
	}
}
