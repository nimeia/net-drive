package adapter

import (
	"developer-mount/internal/mountcore"
	"developer-mount/internal/protocol"
	"testing"
)

type adapterFakeClient struct{}

func (adapterFakeClient) Lookup(parentNodeID uint64, name string) (protocol.LookupResp, error) {
	if name == "file.txt" {
		return protocol.LookupResp{Entry: protocol.NodeInfo{NodeID: 2, ParentNodeID: 1, Name: "file.txt", FileType: protocol.FileTypeFile, Size: 4}}, nil
	}
	return protocol.LookupResp{Entry: protocol.NodeInfo{NodeID: 3, ParentNodeID: 1, Name: name, FileType: protocol.FileTypeDirectory}}, nil
}
func (adapterFakeClient) GetAttr(nodeID uint64) (protocol.GetAttrResp, error) {
	switch nodeID {
	case 1:
		return protocol.GetAttrResp{Entry: protocol.NodeInfo{NodeID: 1, ParentNodeID: 1, FileType: protocol.FileTypeDirectory}}, nil
	case 2:
		return protocol.GetAttrResp{Entry: protocol.NodeInfo{NodeID: 2, ParentNodeID: 1, Name: "file.txt", FileType: protocol.FileTypeFile, Size: 4}}, nil
	default:
		return protocol.GetAttrResp{Entry: protocol.NodeInfo{NodeID: 3, ParentNodeID: 1, Name: "dir", FileType: protocol.FileTypeDirectory}}, nil
	}
}
func (adapterFakeClient) OpenDir(nodeID uint64) (protocol.OpenDirResp, error) {
	return protocol.OpenDirResp{DirCursorID: 10}, nil
}
func (adapterFakeClient) ReadDir(dirCursorID uint64, cookie uint64, maxEntries uint32) (protocol.ReadDirResp, error) {
	return protocol.ReadDirResp{Entries: []protocol.DirEntry{{NodeID: 2, Name: "file.txt", FileType: protocol.FileTypeFile, Size: 4}}, NextCookie: 1, EOF: true}, nil
}
func (adapterFakeClient) OpenRead(nodeID uint64) (protocol.OpenResp, error) {
	return protocol.OpenResp{HandleID: 20, Size: 4}, nil
}
func (adapterFakeClient) Read(handleID uint64, offset int64, length uint32) (protocol.ReadResp, error) {
	return protocol.ReadResp{Data: []byte("data"), Offset: offset, EOF: true}, nil
}
func (adapterFakeClient) CloseHandle(handleID uint64) (protocol.CloseResp, error) {
	return protocol.CloseResp{Closed: true}, nil
}

func TestAdapterReadOnlyOperations(t *testing.T) {
	mount := mountcore.New(adapterFakeClient{}, mountcore.Options{RootNodeID: 1, VolumeName: "workspace", ReadOnly: true})
	adapter := New(mount)
	volume := adapter.GetVolumeInfo()
	if !volume.ReadOnly || volume.Name != "workspace" {
		t.Fatalf("GetVolumeInfo() = %+v", volume)
	}
	openResult, err := adapter.Open(`/file.txt`)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	readResult, err := adapter.Read(openResult.HandleID, 0, 4)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if string(readResult.Data) != "data" {
		t.Fatalf("Read().Data = %q, want data", string(readResult.Data))
	}
	if err := adapter.Close(openResult.HandleID); err != nil {
		t.Fatalf("Close(file) error = %v", err)
	}
	dirResult, err := adapter.OpenDirectory(`/dir`)
	if err != nil {
		t.Fatalf("OpenDirectory() error = %v", err)
	}
	page, err := adapter.ReadDirectory(dirResult.HandleID, 0, 16)
	if err != nil {
		t.Fatalf("ReadDirectory() error = %v", err)
	}
	if len(page.Entries) != 1 || page.Entries[0].Path != "/dir/file.txt" {
		t.Fatalf("ReadDirectory() page = %+v", page)
	}
	if err := adapter.Close(dirResult.HandleID); err != nil {
		t.Fatalf("Close(dir) error = %v", err)
	}
}
