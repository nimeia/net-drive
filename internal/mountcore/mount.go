package mountcore

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	platformwindows "developer-mount/internal/platform/windows"
	"developer-mount/internal/protocol"
)

var (
	ErrInvalidHandle = errors.New("mountcore: invalid handle")
	ErrIsDirectory   = errors.New("mountcore: path is a directory")
	ErrNotDirectory  = errors.New("mountcore: path is not a directory")
)

type ProtocolClient interface {
	Lookup(parentNodeID uint64, name string) (protocol.LookupResp, error)
	GetAttr(nodeID uint64) (protocol.GetAttrResp, error)
	OpenDir(nodeID uint64) (protocol.OpenDirResp, error)
	ReadDir(dirCursorID uint64, cookie uint64, maxEntries uint32) (protocol.ReadDirResp, error)
	OpenRead(nodeID uint64) (protocol.OpenResp, error)
	Read(handleID uint64, offset int64, length uint32) (protocol.ReadResp, error)
	CloseHandle(handleID uint64) (protocol.CloseResp, error)
}

type Options struct {
	RootNodeID       uint64
	VolumeName       string
	ReadOnly         bool
	CaseSensitive    bool
	ReadDirBatchSize uint32
}

type VolumeInfo struct {
	Name               string
	ReadOnly           bool
	CaseSensitive      bool
	MaxComponentLength uint32
}

type FileInfo struct {
	Path         string
	NodeID       uint64
	ParentNodeID uint64
	Name         string
	IsDirectory  bool
	Size         int64
	Mode         uint32
	ModTime      string
}

type FileHandle struct {
	HandleID       uint64
	RemoteHandleID uint64
	Path           string
	Info           FileInfo
}

type DirectoryHandle struct {
	HandleID       uint64
	RemoteCursorID uint64
	Path           string
	Info           FileInfo
}

type DirectoryPage struct {
	Entries    []FileInfo
	NextCookie uint64
	EOF        bool
}

type ReadResult struct {
	Data   []byte
	Offset int64
	EOF    bool
}

type Snapshot struct {
	CachedPaths      []string
	OpenFileHandles  []FileHandle
	OpenDirHandles   []DirectoryHandle
	RootNodeID       uint64
	ReadDirBatchSize uint32
	VolumeName       string
}

type Mount struct {
	client ProtocolClient
	opts   Options

	mu               sync.RWMutex
	pathCache        map[string]protocol.NodeInfo
	fileHandles      map[uint64]FileHandle
	directoryHandles map[uint64]DirectoryHandle
	nextLocalHandle  uint64
}

func New(client ProtocolClient, opts Options) *Mount {
	if opts.RootNodeID == 0 {
		opts.RootNodeID = 1
	}
	if opts.VolumeName == "" {
		opts.VolumeName = "devmount"
	}
	if opts.ReadDirBatchSize == 0 {
		opts.ReadDirBatchSize = 128
	}
	return &Mount{
		client: client,
		opts:   opts,
		pathCache: map[string]protocol.NodeInfo{
			"/": {NodeID: opts.RootNodeID, ParentNodeID: opts.RootNodeID, Name: "", FileType: protocol.FileTypeDirectory},
		},
		fileHandles:      map[uint64]FileHandle{},
		directoryHandles: map[uint64]DirectoryHandle{},
		nextLocalHandle:  1,
	}
}

func (m *Mount) VolumeInfo() VolumeInfo {
	return VolumeInfo{
		Name:               m.opts.VolumeName,
		ReadOnly:           m.opts.ReadOnly,
		CaseSensitive:      m.opts.CaseSensitive,
		MaxComponentLength: 255,
	}
}

func (m *Mount) Lookup(path string) (FileInfo, error) {
	normalized, entry, err := m.resolvePath(path)
	if err != nil {
		return FileInfo{}, err
	}
	return fileInfoFromNode(normalized, entry), nil
}

func (m *Mount) GetAttr(path string) (FileInfo, error) {
	normalized, entry, err := m.resolvePath(path)
	if err != nil {
		return FileInfo{}, err
	}
	resp, err := m.client.GetAttr(entry.NodeID)
	if err != nil {
		return FileInfo{}, err
	}
	m.cachePath(normalized, resp.Entry)
	return fileInfoFromNode(normalized, resp.Entry), nil
}

func (m *Mount) Open(path string) (FileHandle, error) {
	info, err := m.GetAttr(path)
	if err != nil {
		return FileHandle{}, err
	}
	if info.IsDirectory {
		return FileHandle{}, ErrIsDirectory
	}
	resp, err := m.client.OpenRead(info.NodeID)
	if err != nil {
		return FileHandle{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.allocLocalHandleLocked()
	h := FileHandle{HandleID: id, RemoteHandleID: resp.HandleID, Path: info.Path, Info: info}
	m.fileHandles[id] = h
	return h, nil
}

func (m *Mount) OpenDirectory(path string) (DirectoryHandle, error) {
	info, err := m.GetAttr(path)
	if err != nil {
		return DirectoryHandle{}, err
	}
	if !info.IsDirectory {
		return DirectoryHandle{}, ErrNotDirectory
	}
	resp, err := m.client.OpenDir(info.NodeID)
	if err != nil {
		return DirectoryHandle{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.allocLocalHandleLocked()
	h := DirectoryHandle{HandleID: id, RemoteCursorID: resp.DirCursorID, Path: info.Path, Info: info}
	m.directoryHandles[id] = h
	return h, nil
}

func (m *Mount) ReadDirectory(handleID uint64, cookie uint64, maxEntries uint32) (DirectoryPage, error) {
	m.mu.RLock()
	h, ok := m.directoryHandles[handleID]
	m.mu.RUnlock()
	if !ok {
		return DirectoryPage{}, ErrInvalidHandle
	}
	if maxEntries == 0 {
		maxEntries = m.opts.ReadDirBatchSize
	}
	resp, err := m.client.ReadDir(h.RemoteCursorID, cookie, maxEntries)
	if err != nil {
		return DirectoryPage{}, err
	}
	entries := make([]FileInfo, 0, len(resp.Entries))
	for _, entry := range resp.Entries {
		childPath := platformwindows.JoinMountPath(h.Path, entry.Name)
		node := protocol.NodeInfo{NodeID: entry.NodeID, ParentNodeID: h.Info.NodeID, Name: entry.Name, FileType: entry.FileType, Size: entry.Size, Mode: entry.Mode, ModTime: entry.ModTime}
		m.cachePath(childPath, node)
		entries = append(entries, fileInfoFromNode(childPath, node))
	}
	return DirectoryPage{Entries: entries, NextCookie: resp.NextCookie, EOF: resp.EOF}, nil
}

func (m *Mount) Read(handleID uint64, offset int64, length uint32) (ReadResult, error) {
	m.mu.RLock()
	h, ok := m.fileHandles[handleID]
	m.mu.RUnlock()
	if !ok {
		return ReadResult{}, ErrInvalidHandle
	}
	resp, err := m.client.Read(h.RemoteHandleID, offset, length)
	if err != nil {
		return ReadResult{}, err
	}
	return ReadResult{Data: resp.Data, Offset: resp.Offset, EOF: resp.EOF}, nil
}

func (m *Mount) Close(handleID uint64) error {
	m.mu.Lock()
	if h, ok := m.fileHandles[handleID]; ok {
		delete(m.fileHandles, handleID)
		m.mu.Unlock()
		_, err := m.client.CloseHandle(h.RemoteHandleID)
		return err
	}
	if _, ok := m.directoryHandles[handleID]; ok {
		delete(m.directoryHandles, handleID)
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()
	return ErrInvalidHandle
}

func (m *Mount) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cachedPaths := make([]string, 0, len(m.pathCache))
	for path := range m.pathCache {
		cachedPaths = append(cachedPaths, path)
	}
	sort.Strings(cachedPaths)
	fileHandles := make([]FileHandle, 0, len(m.fileHandles))
	for _, h := range m.fileHandles {
		fileHandles = append(fileHandles, h)
	}
	dirHandles := make([]DirectoryHandle, 0, len(m.directoryHandles))
	for _, h := range m.directoryHandles {
		dirHandles = append(dirHandles, h)
	}
	return Snapshot{CachedPaths: cachedPaths, OpenFileHandles: fileHandles, OpenDirHandles: dirHandles, RootNodeID: m.opts.RootNodeID, ReadDirBatchSize: m.opts.ReadDirBatchSize, VolumeName: m.opts.VolumeName}
}

func (m *Mount) resolvePath(input string) (string, protocol.NodeInfo, error) {
	normalized, err := platformwindows.NormalizeMountPath(input)
	if err != nil {
		return "", protocol.NodeInfo{}, err
	}
	m.mu.RLock()
	if entry, ok := m.pathCache[normalized]; ok {
		m.mu.RUnlock()
		return normalized, entry, nil
	}
	m.mu.RUnlock()
	parts, err := platformwindows.SplitMountPath(normalized)
	if err != nil {
		return "", protocol.NodeInfo{}, err
	}
	currentPath := "/"
	current := protocol.NodeInfo{NodeID: m.opts.RootNodeID, ParentNodeID: m.opts.RootNodeID, FileType: protocol.FileTypeDirectory}
	for _, part := range parts {
		nextPath := platformwindows.JoinMountPath(currentPath, part)
		m.mu.RLock()
		cached, ok := m.pathCache[nextPath]
		m.mu.RUnlock()
		if ok {
			currentPath = nextPath
			current = cached
			continue
		}
		resp, err := m.client.Lookup(current.NodeID, part)
		if err != nil {
			return "", protocol.NodeInfo{}, fmt.Errorf("lookup %s: %w", nextPath, err)
		}
		currentPath = nextPath
		current = resp.Entry
		m.cachePath(currentPath, current)
	}
	return normalized, current, nil
}

func (m *Mount) cachePath(path string, entry protocol.NodeInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pathCache[path] = entry
}

func (m *Mount) allocLocalHandleLocked() uint64 {
	id := m.nextLocalHandle
	m.nextLocalHandle++
	return id
}

func fileInfoFromNode(path string, node protocol.NodeInfo) FileInfo {
	name := node.Name
	if name == "" && path != "/" {
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		name = parts[len(parts)-1]
	}
	return FileInfo{Path: path, NodeID: node.NodeID, ParentNodeID: node.ParentNodeID, Name: name, IsDirectory: node.FileType == protocol.FileTypeDirectory, Size: node.Size, Mode: node.Mode, ModTime: node.ModTime}
}
