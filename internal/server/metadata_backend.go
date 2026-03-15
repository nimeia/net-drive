package server

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"developer-mount/internal/protocol"
)

type nodeRecord struct {
	id       uint64
	parentID uint64
	relPath  string
}

type dirCursor struct {
	id      uint64
	nodeID  uint64
	entries []protocol.DirEntry
}

type readHandle struct {
	id     uint64
	nodeID uint64
	file   *os.File
	size   int64
}

type metadataBackend struct {
	rootPath string

	nextNodeID   atomic.Uint64
	nextCursorID atomic.Uint64
	nextHandleID atomic.Uint64

	mu          sync.RWMutex
	nodesByID   map[uint64]nodeRecord
	nodesByPath map[string]uint64
	cursors     map[uint64]dirCursor
	handles     map[uint64]*readHandle
}

func newMetadataBackend(rootPath string) (*metadataBackend, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("abs root path: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("stat root path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root path must be a directory")
	}
	b := &metadataBackend{
		rootPath:    absRoot,
		nodesByID:   make(map[uint64]nodeRecord),
		nodesByPath: make(map[string]uint64),
		cursors:     make(map[uint64]dirCursor),
		handles:     make(map[uint64]*readHandle),
	}
	b.nextNodeID.Store(1)
	b.nextCursorID.Store(1000)
	b.nextHandleID.Store(2000)
	b.nodesByID[1] = nodeRecord{id: 1, parentID: 1, relPath: ""}
	b.nodesByPath[""] = 1
	return b, nil
}

func (b *metadataBackend) RootNodeID() uint64 { return 1 }

func (b *metadataBackend) Lookup(parentNodeID uint64, name string) (protocol.NodeInfo, error) {
	if name == "" || name == "." || strings.Contains(name, string(filepath.Separator)) || strings.Contains(name, "/") || name == ".." {
		return protocol.NodeInfo{}, os.ErrNotExist
	}
	parent, err := b.nodeByID(parentNodeID)
	if err != nil {
		return protocol.NodeInfo{}, err
	}
	childRel := strings.TrimPrefix(filepath.Join(parent.relPath, name), string(filepath.Separator))
	return b.nodeInfoForPath(childRel, parentNodeID)
}

func (b *metadataBackend) GetAttr(nodeID uint64) (protocol.NodeInfo, error) {
	rec, err := b.nodeByID(nodeID)
	if err != nil {
		return protocol.NodeInfo{}, err
	}
	return b.nodeInfoForPath(rec.relPath, rec.parentID)
}

func (b *metadataBackend) OpenDir(nodeID uint64) (uint64, error) {
	rec, err := b.nodeByID(nodeID)
	if err != nil {
		return 0, err
	}
	abs := b.absPath(rec.relPath)
	entries, err := os.ReadDir(abs)
	if err != nil {
		return 0, err
	}
	items := make([]protocol.DirEntry, 0, len(entries))
	for _, entry := range entries {
		childInfo, err := b.Lookup(nodeID, entry.Name())
		if err != nil {
			continue
		}
		items = append(items, protocol.DirEntry{
			NodeID:   childInfo.NodeID,
			Name:     childInfo.Name,
			FileType: childInfo.FileType,
			Size:     childInfo.Size,
			Mode:     childInfo.Mode,
			ModTime:  childInfo.ModTime,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	cursorID := b.nextCursorID.Add(1)
	b.mu.Lock()
	b.cursors[cursorID] = dirCursor{id: cursorID, nodeID: nodeID, entries: items}
	b.mu.Unlock()
	return cursorID, nil
}

func (b *metadataBackend) ReadDir(cursorID uint64, cookie uint64, maxEntries uint32) (protocol.ReadDirResp, error) {
	b.mu.RLock()
	cursor, ok := b.cursors[cursorID]
	b.mu.RUnlock()
	if !ok {
		return protocol.ReadDirResp{}, os.ErrNotExist
	}
	start := int(cookie)
	if start < 0 || start > len(cursor.entries) {
		start = len(cursor.entries)
	}
	if maxEntries == 0 {
		maxEntries = 128
	}
	end := start + int(maxEntries)
	if end > len(cursor.entries) {
		end = len(cursor.entries)
	}
	resp := protocol.ReadDirResp{Entries: append([]protocol.DirEntry(nil), cursor.entries[start:end]...), NextCookie: uint64(end), EOF: end >= len(cursor.entries)}
	return resp, nil
}

func (b *metadataBackend) OpenFile(nodeID uint64) (uint64, int64, error) {
	rec, err := b.nodeByID(nodeID)
	if err != nil {
		return 0, 0, err
	}
	abs := b.absPath(rec.relPath)
	info, err := os.Stat(abs)
	if err != nil {
		return 0, 0, err
	}
	if info.IsDir() {
		return 0, 0, fmt.Errorf("is directory")
	}
	f, err := os.Open(abs)
	if err != nil {
		return 0, 0, err
	}
	handleID := b.nextHandleID.Add(1)
	b.mu.Lock()
	b.handles[handleID] = &readHandle{id: handleID, nodeID: nodeID, file: f, size: info.Size()}
	b.mu.Unlock()
	return handleID, info.Size(), nil
}

func (b *metadataBackend) ReadFile(handleID uint64, offset int64, length uint32) ([]byte, bool, error) {
	b.mu.RLock()
	h, ok := b.handles[handleID]
	b.mu.RUnlock()
	if !ok {
		return nil, false, os.ErrNotExist
	}
	if offset < 0 {
		return nil, false, fmt.Errorf("invalid offset")
	}
	if length == 0 {
		length = 4096
	}
	buf := make([]byte, length)
	n, err := h.file.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return nil, false, err
	}
	buf = buf[:n]
	eof := offset+int64(n) >= h.size || err == io.EOF
	return buf, eof, nil
}

func (b *metadataBackend) CloseHandle(handleID uint64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	h, ok := b.handles[handleID]
	if !ok {
		return os.ErrNotExist
	}
	delete(b.handles, handleID)
	return h.file.Close()
}

func (b *metadataBackend) nodeByID(nodeID uint64) (nodeRecord, error) {
	b.mu.RLock()
	rec, ok := b.nodesByID[nodeID]
	b.mu.RUnlock()
	if !ok {
		return nodeRecord{}, os.ErrNotExist
	}
	return rec, nil
}

func (b *metadataBackend) nodeInfoForPath(relPath string, parentHint uint64) (protocol.NodeInfo, error) {
	abs := b.absPath(relPath)
	info, err := os.Stat(abs)
	if err != nil {
		return protocol.NodeInfo{}, err
	}
	nodeID := b.ensureNode(relPath, parentHint)
	parentID := parentHint
	if relPath == "" {
		parentID = nodeID
	} else if parentID == 0 {
		parentPath := filepath.Dir(relPath)
		if parentPath == "." {
			parentPath = ""
		}
		parentID = b.ensureNode(parentPath, 1)
	}
	fileType := protocol.FileTypeFile
	if info.IsDir() {
		fileType = protocol.FileTypeDirectory
	}
	return protocol.NodeInfo{
		NodeID:       nodeID,
		ParentNodeID: parentID,
		Name:         info.Name(),
		FileType:     fileType,
		Size:         info.Size(),
		Mode:         uint32(info.Mode()),
		ModTime:      protocol.NowRFC3339(info.ModTime()),
	}, nil
}

func (b *metadataBackend) ensureNode(relPath string, parentHint uint64) uint64 {
	relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
	b.mu.RLock()
	if id, ok := b.nodesByPath[relPath]; ok {
		b.mu.RUnlock()
		return id
	}
	b.mu.RUnlock()

	if relPath == "" {
		return 1
	}
	if parentHint == 0 {
		parentPath := filepath.Dir(relPath)
		if parentPath == "." {
			parentPath = ""
		}
		parentHint = b.ensureNode(parentPath, 1)
	}
	id := b.nextNodeID.Add(1)
	b.mu.Lock()
	defer b.mu.Unlock()
	if existing, ok := b.nodesByPath[relPath]; ok {
		return existing
	}
	b.nodesByPath[relPath] = id
	b.nodesByID[id] = nodeRecord{id: id, parentID: parentHint, relPath: relPath}
	return id
}

func (b *metadataBackend) absPath(relPath string) string {
	if relPath == "" {
		return b.rootPath
	}
	return filepath.Join(b.rootPath, relPath)
}

func isNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}

func isNotDir(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not a directory")
}

func isDir(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "is directory")
}

func fileTimeToRFC3339(t time.Time) string {
	return protocol.NowRFC3339(t)
}
