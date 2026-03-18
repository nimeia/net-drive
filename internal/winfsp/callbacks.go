package winfsp

import (
	"sync"

	"developer-mount/internal/mountcore"
	adapterpkg "developer-mount/internal/winfsp/adapter"
)

type FileInfo = mountcore.FileInfo
type DirectoryPage = mountcore.DirectoryPage
type VolumeInfo = mountcore.VolumeInfo

type OpenResult struct {
	HandleID uint64
	Info     FileInfo
}

type SecurityInfo struct {
	Path        string `json:"path"`
	Descriptor  string `json:"descriptor"`
	ReadOnly    bool   `json:"read_only"`
	HandleBound bool   `json:"handle_bound"`
}

type Callbacks struct {
	adapter *adapterpkg.Adapter
	mu      sync.Mutex
	handles map[uint64]FileInfo
}

func NewCallbacks(a *adapterpkg.Adapter) *Callbacks {
	return &Callbacks{adapter: a, handles: map[uint64]FileInfo{}}
}
func (c *Callbacks) GetVolumeInfo() (VolumeInfo, NTStatus) {
	return c.adapter.GetVolumeInfo(), StatusSuccess
}
func (c *Callbacks) GetFileInfo(path string) (FileInfo, NTStatus) {
	info, err := c.adapter.GetFileInfo(path)
	if err != nil {
		return FileInfo{}, mapError(err)
	}
	return info, StatusSuccess
}
func (c *Callbacks) Open(path string) (OpenResult, NTStatus) {
	result, err := c.adapter.Open(path)
	if err != nil {
		return OpenResult{}, mapError(err)
	}
	c.trackHandle(result.HandleID, result.Info)
	return OpenResult{HandleID: result.HandleID, Info: result.Info}, StatusSuccess
}
func (c *Callbacks) OpenDirectory(path string) (OpenResult, NTStatus) {
	result, err := c.adapter.OpenDirectory(path)
	if err != nil {
		return OpenResult{}, mapError(err)
	}
	c.trackHandle(result.HandleID, result.Info)
	return OpenResult{HandleID: result.HandleID, Info: result.Info}, StatusSuccess
}
func (c *Callbacks) ReadDirectory(handleID uint64, cookie uint64, maxEntries uint32) (DirectoryPage, NTStatus) {
	page, err := c.adapter.ReadDirectory(handleID, cookie, maxEntries)
	if err != nil {
		return DirectoryPage{}, mapError(err)
	}
	return page, StatusSuccess
}
func (c *Callbacks) Read(handleID uint64, offset int64, length uint32) ([]byte, bool, NTStatus) {
	result, err := c.adapter.Read(handleID, offset, length)
	if err != nil {
		return nil, false, mapError(err)
	}
	return result.Data, result.EOF, StatusSuccess
}
func (c *Callbacks) Cleanup(handleID uint64) NTStatus {
	if _, ok := c.lookupHandle(handleID); !ok {
		return StatusInvalidHandle
	}
	return StatusSuccess
}
func (c *Callbacks) Flush(handleID uint64) NTStatus {
	if _, ok := c.lookupHandle(handleID); !ok {
		return StatusInvalidHandle
	}
	return StatusSuccess
}
func (c *Callbacks) GetSecurityByName(path string) (SecurityInfo, NTStatus) {
	info, status := c.GetFileInfo(path)
	if status != StatusSuccess {
		return SecurityInfo{}, status
	}
	return SecurityInfo{Path: info.Path, Descriptor: defaultSecurityDescriptor(info), ReadOnly: true, HandleBound: false}, StatusSuccess
}
func (c *Callbacks) GetSecurity(handleID uint64) (SecurityInfo, NTStatus) {
	info, ok := c.lookupHandle(handleID)
	if !ok {
		return SecurityInfo{}, StatusInvalidHandle
	}
	return SecurityInfo{Path: info.Path, Descriptor: defaultSecurityDescriptor(info), ReadOnly: true, HandleBound: true}, StatusSuccess
}
func (c *Callbacks) Close(handleID uint64) NTStatus {
	if err := c.adapter.Close(handleID); err != nil {
		return mapError(err)
	}
	c.untrackHandle(handleID)
	return StatusSuccess
}
func (c *Callbacks) Snapshot() mountcore.Snapshot { return c.adapter.Snapshot() }

func (c *Callbacks) trackHandle(handleID uint64, info FileInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.handles == nil {
		c.handles = map[uint64]FileInfo{}
	}
	c.handles[handleID] = info
}
func (c *Callbacks) lookupHandle(handleID uint64) (FileInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	info, ok := c.handles[handleID]
	return info, ok
}
func (c *Callbacks) untrackHandle(handleID uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.handles, handleID)
}
func defaultSecurityDescriptor(info FileInfo) string {
	if info.IsDirectory {
		return "O:BAG:BAD:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FR;;;WD)"
	}
	return "O:BAG:BAD:PAI(A;;FA;;;SY)(A;;FA;;;BA)(A;;FR;;;WD)"
}
