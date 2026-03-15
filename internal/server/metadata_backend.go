package server

import (
	"errors"
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

const defaultMetadataCacheTTL = 2 * time.Second

var (
	errInvalidHandle = errors.New("invalid handle")
	errAlreadyExists = errors.New("already exists")
	errNotDir        = errors.New("not a directory")
	errIsDir         = errors.New("is a directory")
	errAccessDenied  = errors.New("access denied")
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

type fileHandle struct {
	id            uint64
	nodeID        uint64
	parentID      uint64
	relPath       string
	file          *os.File
	size          int64
	writable      bool
	deleteOnClose bool
}

type metadataBackend struct {
	rootPath string
	now      func() time.Time
	cacheTTL time.Duration

	nextNodeID   atomic.Uint64
	nextCursorID atomic.Uint64
	nextHandleID atomic.Uint64

	stats metadataCacheStats

	mu            sync.RWMutex
	nodesByID     map[uint64]nodeRecord
	nodesByPath   map[string]uint64
	cursors       map[uint64]dirCursor
	handles       map[uint64]*fileHandle
	attrCache     map[string]attrCacheEntry
	negativeCache map[string]negativeCacheEntry
	dirSnapshots  map[uint64]dirSnapshotEntry
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
		rootPath:      absRoot,
		now:           time.Now,
		cacheTTL:      defaultMetadataCacheTTL,
		nodesByID:     make(map[uint64]nodeRecord),
		nodesByPath:   make(map[string]uint64),
		cursors:       make(map[uint64]dirCursor),
		handles:       make(map[uint64]*fileHandle),
		attrCache:     make(map[string]attrCacheEntry),
		negativeCache: make(map[string]negativeCacheEntry),
		dirSnapshots:  make(map[uint64]dirSnapshotEntry),
	}
	b.nextNodeID.Store(1)
	b.nextCursorID.Store(1000)
	b.nextHandleID.Store(2000)
	b.nodesByID[1] = nodeRecord{id: 1, parentID: 1, relPath: ""}
	b.nodesByPath[""] = 1
	if _, err := b.refreshNodeInfo("", 1); err != nil {
		return nil, err
	}
	_ = b.prefetchRoot()
	return b, nil
}

func (b *metadataBackend) RootNodeID() uint64 { return 1 }

func (b *metadataBackend) Stats() MetadataCacheStats {
	return b.stats.snapshot()
}

func (b *metadataBackend) Lookup(parentNodeID uint64, name string) (protocol.NodeInfo, error) {
	if !isValidSingleName(name) {
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
	entries, err := b.snapshotDir(nodeID)
	if err != nil {
		return 0, err
	}
	cursorID := b.nextCursorID.Add(1)
	b.mu.Lock()
	b.cursors[cursorID] = dirCursor{id: cursorID, nodeID: nodeID, entries: entries}
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
	return protocol.ReadDirResp{
		Entries:    append([]protocol.DirEntry(nil), cursor.entries[start:end]...),
		NextCookie: uint64(end),
		EOF:        end >= len(cursor.entries),
	}, nil
}

func (b *metadataBackend) OpenFile(nodeID uint64, writable, truncate bool) (uint64, int64, error) {
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
		return 0, 0, errIsDir
	}
	flags := os.O_RDONLY
	if writable {
		flags = os.O_RDWR
		if truncate {
			flags |= os.O_TRUNC
		}
	}
	f, err := os.OpenFile(abs, flags, 0)
	if err != nil {
		if os.IsPermission(err) {
			return 0, 0, errAccessDenied
		}
		return 0, 0, err
	}
	currentSize := info.Size()
	if truncate {
		currentSize = 0
	}
	handleID := b.nextHandleID.Add(1)
	b.mu.Lock()
	b.handles[handleID] = &fileHandle{id: handleID, nodeID: nodeID, parentID: rec.parentID, relPath: rec.relPath, file: f, size: currentSize, writable: writable}
	b.mu.Unlock()
	b.invalidatePath(rec.relPath, rec.parentID)
	return handleID, currentSize, nil
}

func (b *metadataBackend) CreateFile(parentNodeID uint64, name string, overwrite bool) (protocol.NodeInfo, uint64, error) {
	if !isValidSingleName(name) {
		return protocol.NodeInfo{}, 0, os.ErrNotExist
	}
	parent, err := b.nodeByID(parentNodeID)
	if err != nil {
		return protocol.NodeInfo{}, 0, err
	}
	parentInfo, err := b.nodeInfoForPath(parent.relPath, parent.parentID)
	if err != nil {
		return protocol.NodeInfo{}, 0, err
	}
	if parentInfo.FileType != protocol.FileTypeDirectory {
		return protocol.NodeInfo{}, 0, errNotDir
	}
	relPath := strings.TrimPrefix(filepath.Join(parent.relPath, name), string(filepath.Separator))
	abs := b.absPath(relPath)
	flags := os.O_CREATE | os.O_RDWR
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	f, err := os.OpenFile(abs, flags, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return protocol.NodeInfo{}, 0, errAlreadyExists
		}
		if os.IsPermission(err) {
			return protocol.NodeInfo{}, 0, errAccessDenied
		}
		return protocol.NodeInfo{}, 0, err
	}
	nodeID := b.ensureNode(relPath, parentNodeID)
	handleID := b.nextHandleID.Add(1)
	b.mu.Lock()
	b.handles[handleID] = &fileHandle{id: handleID, nodeID: nodeID, parentID: parentNodeID, relPath: relPath, file: f, size: 0, writable: true}
	b.mu.Unlock()
	b.invalidatePath(relPath, parentNodeID)
	entry, err := b.refreshNodeInfo(relPath, parentNodeID)
	if err != nil {
		_ = f.Close()
		return protocol.NodeInfo{}, 0, err
	}
	return entry, handleID, nil
}

func (b *metadataBackend) ReadFile(handleID uint64, offset int64, length uint32) ([]byte, bool, error) {
	h, err := b.handleByID(handleID)
	if err != nil {
		return nil, false, err
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

func (b *metadataBackend) WriteFile(handleID uint64, offset int64, data []byte) (int, int64, error) {
	h, err := b.handleByID(handleID)
	if err != nil {
		return 0, 0, err
	}
	if !h.writable {
		return 0, 0, errAccessDenied
	}
	if offset < 0 {
		return 0, 0, fmt.Errorf("invalid offset")
	}
	n, err := h.file.WriteAt(data, offset)
	if err != nil {
		return n, h.size, err
	}
	b.mu.Lock()
	if end := offset + int64(n); end > h.size {
		h.size = end
	}
	newSize := h.size
	b.mu.Unlock()
	b.invalidatePath(h.relPath, h.parentID)
	return n, newSize, nil
}

func (b *metadataBackend) FlushHandle(handleID uint64) error {
	h, err := b.handleByID(handleID)
	if err != nil {
		return err
	}
	if err := h.file.Sync(); err != nil {
		return err
	}
	b.invalidatePath(h.relPath, h.parentID)
	_, _ = b.refreshNodeInfo(h.relPath, h.parentID)
	return nil
}

func (b *metadataBackend) TruncateHandle(handleID uint64, size int64) (int64, error) {
	h, err := b.handleByID(handleID)
	if err != nil {
		return 0, err
	}
	if !h.writable {
		return 0, errAccessDenied
	}
	if err := h.file.Truncate(size); err != nil {
		return 0, err
	}
	b.mu.Lock()
	h.size = size
	b.mu.Unlock()
	b.invalidatePath(h.relPath, h.parentID)
	_, _ = b.refreshNodeInfo(h.relPath, h.parentID)
	return size, nil
}

func (b *metadataBackend) SetDeleteOnClose(handleID uint64, enabled bool) error {
	h, err := b.handleByID(handleID)
	if err != nil {
		return err
	}
	b.mu.Lock()
	h.deleteOnClose = enabled
	b.mu.Unlock()
	return nil
}

func (b *metadataBackend) CloseHandle(handleID uint64) error {
	b.mu.Lock()
	h, ok := b.handles[handleID]
	if !ok {
		b.mu.Unlock()
		return errInvalidHandle
	}
	delete(b.handles, handleID)
	b.mu.Unlock()

	closeErr := h.file.Close()
	if closeErr != nil {
		return closeErr
	}
	if h.deleteOnClose {
		if err := os.Remove(b.absPath(h.relPath)); err != nil && !os.IsNotExist(err) {
			return err
		}
		b.removePathMapping(h.relPath)
		b.invalidatePath(h.relPath, h.parentID)
		b.recordNegative(h.relPath)
		return nil
	}
	b.invalidatePath(h.relPath, h.parentID)
	return nil
}

func (b *metadataBackend) RenamePath(srcParentNodeID uint64, srcName string, dstParentNodeID uint64, dstName string, replace bool) (protocol.NodeInfo, error) {
	if !isValidSingleName(srcName) || !isValidSingleName(dstName) {
		return protocol.NodeInfo{}, os.ErrNotExist
	}
	srcParent, err := b.nodeByID(srcParentNodeID)
	if err != nil {
		return protocol.NodeInfo{}, err
	}
	dstParent, err := b.nodeByID(dstParentNodeID)
	if err != nil {
		return protocol.NodeInfo{}, err
	}
	srcRel := strings.TrimPrefix(filepath.Join(srcParent.relPath, srcName), string(filepath.Separator))
	dstRel := strings.TrimPrefix(filepath.Join(dstParent.relPath, dstName), string(filepath.Separator))
	srcAbs := b.absPath(srcRel)
	dstAbs := b.absPath(dstRel)
	if _, err := os.Stat(srcAbs); err != nil {
		return protocol.NodeInfo{}, err
	}
	if _, err := os.Stat(dstAbs); err == nil && !replace {
		return protocol.NodeInfo{}, errAlreadyExists
	}
	if err := os.Rename(srcAbs, dstAbs); err != nil {
		if os.IsPermission(err) {
			return protocol.NodeInfo{}, errAccessDenied
		}
		return protocol.NodeInfo{}, err
	}
	b.movePathMapping(srcRel, dstRel, dstParentNodeID)
	b.invalidatePath(srcRel, srcParentNodeID)
	b.invalidatePath(dstRel, dstParentNodeID)
	entry, err := b.refreshNodeInfo(dstRel, dstParentNodeID)
	if err != nil {
		return protocol.NodeInfo{}, err
	}
	return entry, nil
}

func (b *metadataBackend) handleByID(handleID uint64) (*fileHandle, error) {
	b.mu.RLock()
	h, ok := b.handles[handleID]
	b.mu.RUnlock()
	if !ok {
		return nil, errInvalidHandle
	}
	return h, nil
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
	relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
	if b.isNegativeCached(relPath) {
		b.stats.negativeHits.Add(1)
		return protocol.NodeInfo{}, os.ErrNotExist
	}
	b.stats.negativeMisses.Add(1)
	if info, ok := b.cachedNodeInfo(relPath); ok {
		b.stats.attrHits.Add(1)
		return b.withResolvedParent(info, relPath, parentHint), nil
	}
	b.stats.attrMisses.Add(1)
	return b.refreshNodeInfo(relPath, parentHint)
}

func (b *metadataBackend) snapshotDir(nodeID uint64) ([]protocol.DirEntry, error) {
	if entries, ok := b.cachedDirSnapshot(nodeID); ok {
		b.stats.dirSnapshotHits.Add(1)
		return entries, nil
	}
	b.stats.dirSnapshotMisses.Add(1)
	return b.refreshDirSnapshot(nodeID)
}

func (b *metadataBackend) refreshDirSnapshot(nodeID uint64) ([]protocol.DirEntry, error) {
	rec, err := b.nodeByID(nodeID)
	if err != nil {
		return nil, err
	}
	abs := b.absPath(rec.relPath)
	entries, err := os.ReadDir(abs)
	if err != nil {
		if isNotDir(err) {
			return nil, errNotDir
		}
		return nil, err
	}
	items := make([]protocol.DirEntry, 0, len(entries))
	for _, entry := range entries {
		childRel := strings.TrimPrefix(filepath.Join(rec.relPath, entry.Name()), string(filepath.Separator))
		childInfo, err := b.nodeInfoForPath(childRel, nodeID)
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
	b.mu.Lock()
	b.dirSnapshots[nodeID] = dirSnapshotEntry{entries: append([]protocol.DirEntry(nil), items...), expiresAt: b.now().Add(b.cacheTTL)}
	b.mu.Unlock()
	return append([]protocol.DirEntry(nil), items...), nil
}

func (b *metadataBackend) refreshNodeInfo(relPath string, parentHint uint64) (protocol.NodeInfo, error) {
	abs := b.absPath(relPath)
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			b.recordNegative(relPath)
			return protocol.NodeInfo{}, err
		}
		return protocol.NodeInfo{}, err
	}
	b.clearNegative(relPath)
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
	entry := protocol.NodeInfo{
		NodeID:       nodeID,
		ParentNodeID: parentID,
		Name:         info.Name(),
		FileType:     fileType,
		Size:         info.Size(),
		Mode:         uint32(info.Mode()),
		ModTime:      protocol.NowRFC3339(info.ModTime()),
	}
	b.mu.Lock()
	b.attrCache[relPath] = attrCacheEntry{info: entry, expiresAt: b.now().Add(b.cacheTTL)}
	b.mu.Unlock()
	return entry, nil
}

func (b *metadataBackend) cachedNodeInfo(relPath string) (protocol.NodeInfo, bool) {
	b.mu.RLock()
	entry, ok := b.attrCache[relPath]
	b.mu.RUnlock()
	if !ok || !b.isFresh(entry.expiresAt) {
		if ok {
			b.mu.Lock()
			delete(b.attrCache, relPath)
			b.mu.Unlock()
		}
		return protocol.NodeInfo{}, false
	}
	return entry.info, true
}

func (b *metadataBackend) cachedDirSnapshot(nodeID uint64) ([]protocol.DirEntry, bool) {
	b.mu.RLock()
	snapshot, ok := b.dirSnapshots[nodeID]
	b.mu.RUnlock()
	if !ok || !b.isFresh(snapshot.expiresAt) {
		if ok {
			b.mu.Lock()
			delete(b.dirSnapshots, nodeID)
			b.mu.Unlock()
		}
		return nil, false
	}
	return append([]protocol.DirEntry(nil), snapshot.entries...), true
}

func (b *metadataBackend) isNegativeCached(relPath string) bool {
	b.mu.RLock()
	entry, ok := b.negativeCache[relPath]
	b.mu.RUnlock()
	if !ok {
		return false
	}
	if !b.isFresh(entry.expiresAt) {
		b.mu.Lock()
		delete(b.negativeCache, relPath)
		b.mu.Unlock()
		return false
	}
	return true
}

func (b *metadataBackend) recordNegative(relPath string) {
	b.mu.Lock()
	b.negativeCache[relPath] = negativeCacheEntry{expiresAt: b.now().Add(b.cacheTTL)}
	b.mu.Unlock()
}

func (b *metadataBackend) clearNegative(relPath string) {
	b.mu.Lock()
	delete(b.negativeCache, relPath)
	b.mu.Unlock()
}

func (b *metadataBackend) withResolvedParent(info protocol.NodeInfo, relPath string, parentHint uint64) protocol.NodeInfo {
	if relPath == "" {
		info.ParentNodeID = info.NodeID
		return info
	}
	if parentHint != 0 {
		info.ParentNodeID = parentHint
		return info
	}
	parentPath := filepath.Dir(relPath)
	if parentPath == "." {
		parentPath = ""
	}
	info.ParentNodeID = b.ensureNode(parentPath, 1)
	return info
}

func (b *metadataBackend) prefetchRoot() error {
	b.stats.rootPrefetches.Add(1)
	_, err := b.refreshDirSnapshot(b.RootNodeID())
	return err
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

func (b *metadataBackend) movePathMapping(srcRel, dstRel string, dstParentID uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id, ok := b.nodesByPath[srcRel]
	if ok {
		delete(b.nodesByPath, srcRel)
		b.nodesByPath[dstRel] = id
		rec := b.nodesByID[id]
		rec.relPath = dstRel
		rec.parentID = dstParentID
		b.nodesByID[id] = rec
	}
	delete(b.attrCache, srcRel)
	delete(b.attrCache, dstRel)
	delete(b.negativeCache, srcRel)
	delete(b.negativeCache, dstRel)
}

func (b *metadataBackend) removePathMapping(relPath string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if id, ok := b.nodesByPath[relPath]; ok {
		delete(b.nodesByPath, relPath)
		delete(b.nodesByID, id)
	}
	delete(b.attrCache, relPath)
	delete(b.negativeCache, relPath)
}

func (b *metadataBackend) invalidatePath(relPath string, parentID uint64) {
	relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
	b.mu.Lock()
	delete(b.attrCache, relPath)
	delete(b.negativeCache, relPath)
	if parentID != 0 {
		delete(b.dirSnapshots, parentID)
	}
	if relPath == "" {
		delete(b.dirSnapshots, 1)
	}
	b.mu.Unlock()
}

func (b *metadataBackend) absPath(relPath string) string {
	if relPath == "" {
		return b.rootPath
	}
	return filepath.Join(b.rootPath, relPath)
}

func (b *metadataBackend) isFresh(expiresAt time.Time) bool {
	return !b.now().After(expiresAt)
}

func isValidSingleName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.Contains(name, "/") && !strings.Contains(name, string(filepath.Separator))
}

func isNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}

func isNotDir(err error) bool {
	return errors.Is(err, errNotDir) || (err != nil && strings.Contains(strings.ToLower(err.Error()), "not a directory"))
}

func isDir(err error) bool {
	return errors.Is(err, errIsDir) || (err != nil && strings.Contains(strings.ToLower(err.Error()), "is a directory"))
}

type HandleSnapshot struct {
	NodeID        uint64
	ParentNodeID  uint64
	RelPath       string
	Name          string
	DeleteOnClose bool
	Size          int64
}

func (b *metadataBackend) HandleSnapshot(handleID uint64) (HandleSnapshot, error) {
	h, err := b.handleByID(handleID)
	if err != nil {
		return HandleSnapshot{}, err
	}
	return HandleSnapshot{
		NodeID:        h.nodeID,
		ParentNodeID:  h.parentID,
		RelPath:       h.relPath,
		Name:          filepath.Base(h.relPath),
		DeleteOnClose: h.deleteOnClose,
		Size:          h.size,
	}, nil
}

func (b *metadataBackend) RelPathByNodeID(nodeID uint64) (string, error) {
	rec, err := b.nodeByID(nodeID)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rec.relPath), nil
}

func (b *metadataBackend) SnapshotSubtree(nodeID uint64, recursive bool) ([]protocol.SnapshotEntry, error) {
	rec, err := b.nodeByID(nodeID)
	if err != nil {
		return nil, err
	}
	rootInfo, err := b.GetAttr(nodeID)
	if err != nil {
		return nil, err
	}
	entries := []protocol.SnapshotEntry{{RelativePath: "", Entry: rootInfo}}
	if rootInfo.FileType != protocol.FileTypeDirectory {
		return entries, nil
	}
	baseAbs := b.absPath(rec.relPath)
	var walk func(currentRel string) error
	walk = func(currentRel string) error {
		currentAbs := b.absPath(currentRel)
		dirEntries, err := os.ReadDir(currentAbs)
		if err != nil {
			return err
		}
		for _, dirEntry := range dirEntries {
			childRel := strings.TrimPrefix(filepath.Join(currentRel, dirEntry.Name()), string(filepath.Separator))
			childInfo, err := b.nodeInfoForPath(childRel, nodeID)
			if err != nil {
				return err
			}
			relToRoot, err := filepath.Rel(baseAbs, b.absPath(childRel))
			if err != nil {
				return err
			}
			relToRoot = filepath.ToSlash(relToRoot)
			entries = append(entries, protocol.SnapshotEntry{RelativePath: relToRoot, Entry: childInfo})
			if recursive && dirEntry.IsDir() {
				if err := walk(childRel); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(rec.relPath); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].RelativePath < entries[j].RelativePath })
	return entries, nil
}
