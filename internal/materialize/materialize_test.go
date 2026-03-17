package materialize

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"developer-mount/internal/mountcore"
	"developer-mount/internal/protocol"
)

type fakeClient struct{}
func (fakeClient) Lookup(parentNodeID uint64, name string) (protocol.LookupResp, error) {
	switch parentNodeID { case 1: switch name { case "docs": return protocol.LookupResp{Entry: protocol.NodeInfo{NodeID: 2, ParentNodeID: 1, Name: "docs", FileType: protocol.FileTypeDirectory}}, nil; case "README.md": return protocol.LookupResp{Entry: protocol.NodeInfo{NodeID: 3, ParentNodeID: 1, Name: "README.md", FileType: protocol.FileTypeFile, Size: 5}}, nil }; case 2: if name == "guide.txt" { return protocol.LookupResp{Entry: protocol.NodeInfo{NodeID: 4, ParentNodeID: 2, Name: "guide.txt", FileType: protocol.FileTypeFile, Size: 7}}, nil } }
	return protocol.LookupResp{}, errors.New("ERR_NOT_FOUND: missing")
}
func (fakeClient) GetAttr(nodeID uint64) (protocol.GetAttrResp, error) {
	switch nodeID { case 1: return protocol.GetAttrResp{Entry: protocol.NodeInfo{NodeID: 1, ParentNodeID: 1, FileType: protocol.FileTypeDirectory}}, nil; case 2: return protocol.GetAttrResp{Entry: protocol.NodeInfo{NodeID: 2, ParentNodeID: 1, Name: "docs", FileType: protocol.FileTypeDirectory}}, nil; case 3: return protocol.GetAttrResp{Entry: protocol.NodeInfo{NodeID: 3, ParentNodeID: 1, Name: "README.md", FileType: protocol.FileTypeFile, Size: 5, ModTime: "2026-03-17T08:00:00Z"}}, nil; case 4: return protocol.GetAttrResp{Entry: protocol.NodeInfo{NodeID: 4, ParentNodeID: 2, Name: "guide.txt", FileType: protocol.FileTypeFile, Size: 7}}, nil }
	return protocol.GetAttrResp{}, errors.New("ERR_NOT_FOUND: missing")
}
func (fakeClient) OpenDir(nodeID uint64) (protocol.OpenDirResp, error) { return protocol.OpenDirResp{DirCursorID: nodeID}, nil }
func (fakeClient) ReadDir(dirCursorID uint64, cookie uint64, maxEntries uint32) (protocol.ReadDirResp, error) {
	switch dirCursorID { case 1: return protocol.ReadDirResp{Entries: []protocol.DirEntry{{NodeID: 2, Name: "docs", FileType: protocol.FileTypeDirectory}, {NodeID: 3, Name: "README.md", FileType: protocol.FileTypeFile, Size: 5}}, EOF: true}, nil; case 2: return protocol.ReadDirResp{Entries: []protocol.DirEntry{{NodeID: 4, Name: "guide.txt", FileType: protocol.FileTypeFile, Size: 7}}, EOF: true}, nil }
	return protocol.ReadDirResp{}, errors.New("ERR_NOT_FOUND: dir")
}
func (fakeClient) OpenRead(nodeID uint64) (protocol.OpenResp, error) { return protocol.OpenResp{HandleID: nodeID, Size: 16}, nil }
func (fakeClient) Read(handleID uint64, offset int64, length uint32) (protocol.ReadResp, error) {
	content := map[uint64]string{3: "hello", 4: "welcome"}[handleID]
	if content == "" { return protocol.ReadResp{}, errors.New("ERR_INVALID_HANDLE") }
	if offset >= int64(len(content)) { return protocol.ReadResp{Offset: offset, EOF: true}, nil }
	end := int(offset) + int(length); if end > len(content) { end = len(content) }
	return protocol.ReadResp{Data: []byte(content[offset:end]), Offset: offset, EOF: end == len(content)}, nil
}
func (fakeClient) CloseHandle(handleID uint64) (protocol.CloseResp, error) { return protocol.CloseResp{Closed: true}, nil }
func TestMaterializeDirectoryTree(t *testing.T) {
	mount := mountcore.New(fakeClient{}, mountcore.Options{RootNodeID: 1, ReadOnly: true}); m := New(mount, 3, 16); dir := t.TempDir(); stats, err := m.Sync(context.Background(), "/", dir)
	if err != nil { t.Fatalf("Sync(/) error = %v", err) }
	if stats.Directories != 2 || stats.Files != 2 || stats.Bytes != 12 { t.Fatalf("stats = %+v", stats) }
	readme, _ := os.ReadFile(filepath.Join(dir, "README.md")); if string(readme) != "hello" { t.Fatalf("README.md = %q, want hello", string(readme)) }
	guide, _ := os.ReadFile(filepath.Join(dir, "docs", "guide.txt")); if string(guide) != "welcome" { t.Fatalf("guide.txt = %q, want welcome", string(guide)) }
}
func TestMaterializeSingleFile(t *testing.T) {
	mount := mountcore.New(fakeClient{}, mountcore.Options{RootNodeID: 1, ReadOnly: true}); m := New(mount, 64, 16); dir := t.TempDir(); stats, err := m.Sync(context.Background(), "/README.md", filepath.Join(dir, "copy.txt"))
	if err != nil { t.Fatalf("Sync(file) error = %v", err) }
	if stats.Directories != 0 || stats.Files != 1 || stats.Bytes != 5 { t.Fatalf("stats = %+v", stats) }
	body, _ := os.ReadFile(filepath.Join(dir, "copy.txt")); if string(body) != "hello" { t.Fatalf("copy.txt = %q, want hello", string(body)) }
}
func TestValidateLocalNameRejectsTraversal(t *testing.T) {
	for _, name := range []string{"..", "a/b", `a\\b`, ""} { if _, err := validateLocalName(name); err == nil { t.Fatalf("validateLocalName(%q) error = nil, want error", name) } }
}
