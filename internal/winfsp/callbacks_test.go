package winfsp

import (
	"developer-mount/internal/mountcore"
	"developer-mount/internal/protocol"
	adapterpkg "developer-mount/internal/winfsp/adapter"
	"errors"
	"strings"
	"testing"
)

type callbackFakeClient struct{}

func (callbackFakeClient) Lookup(parentNodeID uint64, name string) (protocol.LookupResp, error) {
	switch name {
	case "file.txt", "README.md":
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
		t.Fatalf("GetFileInfo missing status = 0x%08x", uint32(status))
	}
	if status := callbacks.Create(`/new.txt`, false); status != StatusAccessDenied {
		t.Fatalf("Create(new) status = 0x%08x", uint32(status))
	}
	fileOpen, status := callbacks.Open(`/file.txt`)
	if status != StatusSuccess {
		t.Fatalf("Open(file) status = 0x%08x", uint32(status))
	}
	secByName, status := callbacks.GetSecurityByName(`/file.txt`)
	if status != StatusSuccess || secByName.Descriptor == "" || len(secByName.Access) == 0 {
		t.Fatalf("GetSecurityByName() = %+v status=0x%08x", secByName, uint32(status))
	}
	if _, eof, status := callbacks.Read(fileOpen.HandleID, 0, 4); status != StatusSuccess || !eof {
		t.Fatalf("Read status=0x%08x eof=%v", uint32(status), eof)
	}
	if _, status := callbacks.Write(fileOpen.HandleID, 0, []byte("x"), false); status != StatusAccessDenied {
		t.Fatalf("Write(file) status = 0x%08x", uint32(status))
	}
	for name, got := range map[string]NTStatus{"SetBasicInfo": callbacks.SetBasicInfo(fileOpen.HandleID, 0), "SetFileSize": callbacks.SetFileSize(fileOpen.HandleID, 0, false), "SetSecurity": callbacks.SetSecurity(fileOpen.HandleID, "x"), "Rename": callbacks.Rename(fileOpen.HandleID, "/file-renamed.txt", false), "Overwrite": callbacks.Overwrite(fileOpen.HandleID, 0, 0, false)} {
		if got != StatusAccessDenied {
			t.Fatalf("%s status = 0x%08x", name, uint32(got))
		}
	}
	if status := callbacks.CanDelete(`/file.txt`); status != StatusAccessDenied {
		t.Fatalf("CanDelete(file) status = 0x%08x", uint32(status))
	}
	if status := callbacks.SetDeleteOnClose(fileOpen.HandleID, true); status != StatusAccessDenied {
		t.Fatalf("SetDeleteOnClose status = 0x%08x", uint32(status))
	}
	_ = callbacks.Flush(fileOpen.HandleID)
	sec, status := callbacks.GetSecurity(fileOpen.HandleID)
	if status != StatusSuccess || !sec.DeleteOnClose || sec.FlushState != "flushed" || !strings.Contains(sec.Summary, "delete-on-close") {
		t.Fatalf("GetSecurity(handle) = %+v status=0x%08x", sec, uint32(status))
	}
	_ = callbacks.Cleanup(fileOpen.HandleID)
	sec, status = callbacks.GetSecurity(fileOpen.HandleID)
	if status != StatusSuccess || sec.CleanupState != "delete-on-close-denied" {
		t.Fatalf("GetSecurity(after-cleanup) = %+v status=0x%08x", sec, uint32(status))
	}
}
