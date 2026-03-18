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
	Path          string   `json:"path"`
	Descriptor    string   `json:"descriptor"`
	Owner         string   `json:"owner"`
	Group         string   `json:"group"`
	Access        []string `json:"access,omitempty"`
	ReadOnly      bool     `json:"read_only"`
	HandleBound   bool     `json:"handle_bound"`
	Directory     bool     `json:"directory"`
	DeleteOnClose bool     `json:"delete_on_close,omitempty"`
	CleanupState  string   `json:"cleanup_state,omitempty"`
	FlushState    string   `json:"flush_state,omitempty"`
	Source        string   `json:"source,omitempty"`
	Summary       string   `json:"summary,omitempty"`
}

type handleState struct {
	Info          FileInfo
	DeleteOnClose bool
	Cleaned       bool
	Flushed       bool
}

type Callbacks struct {
	adapter *adapterpkg.Adapter
	mu      sync.Mutex
	handles map[uint64]handleState
}

func NewCallbacks(a *adapterpkg.Adapter) *Callbacks {
	return &Callbacks{adapter: a, handles: map[uint64]handleState{}}
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
	state, ok := c.lookupHandle(handleID)
	if !ok {
		return StatusInvalidHandle
	}
	state.Cleaned = true
	c.updateHandle(handleID, state)
	return StatusSuccess
}
func (c *Callbacks) Flush(handleID uint64) NTStatus {
	state, ok := c.lookupHandle(handleID)
	if !ok {
		return StatusInvalidHandle
	}
	state.Flushed = true
	c.updateHandle(handleID, state)
	return StatusSuccess
}
func (c *Callbacks) GetSecurityByName(path string) (SecurityInfo, NTStatus) {
	info, status := c.GetFileInfo(path)
	if status != StatusSuccess {
		return SecurityInfo{}, status
	}
	return securityInfoFromDescriptor(DefaultNativeSecurityDescriptor(info, SecurityDescriptorOptions{Source: SecuritySourceByName})), StatusSuccess
}
func (c *Callbacks) GetSecurity(handleID uint64) (SecurityInfo, NTStatus) {
	state, ok := c.lookupHandle(handleID)
	if !ok {
		return SecurityInfo{}, StatusInvalidHandle
	}
	descriptor := DefaultNativeSecurityDescriptor(state.Info, SecurityDescriptorOptions{HandleBound: true, DeleteOnClose: state.DeleteOnClose, Cleaned: state.Cleaned, Flushed: state.Flushed, Source: SecuritySourceByHandle})
	return securityInfoFromDescriptor(descriptor), StatusSuccess
}
func (c *Callbacks) CanDelete(path string) NTStatus {
	if _, status := c.GetFileInfo(path); status != StatusSuccess {
		return status
	}
	return StatusAccessDenied
}
func (c *Callbacks) SetDeleteOnClose(handleID uint64, enabled bool) NTStatus {
	state, ok := c.lookupHandle(handleID)
	if !ok {
		return StatusInvalidHandle
	}
	state.DeleteOnClose = enabled
	c.updateHandle(handleID, state)
	if enabled {
		return StatusAccessDenied
	}
	return StatusSuccess
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
		c.handles = map[uint64]handleState{}
	}
	c.handles[handleID] = handleState{Info: info}
}
func (c *Callbacks) lookupHandle(handleID uint64) (handleState, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	info, ok := c.handles[handleID]
	return info, ok
}
func (c *Callbacks) updateHandle(handleID uint64, state handleState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.handles == nil {
		c.handles = map[uint64]handleState{}
	}
	c.handles[handleID] = state
}
func (c *Callbacks) untrackHandle(handleID uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.handles, handleID)
}
func securityInfoFromDescriptor(d NativeSecurityDescriptor) SecurityInfo {
	return SecurityInfo{Path: d.Path, Descriptor: d.SDDL, Owner: d.Owner, Group: d.Group, Access: d.Access, ReadOnly: d.ReadOnly, HandleBound: d.HandleBound, Directory: d.Directory, DeleteOnClose: d.DeleteOnClose, CleanupState: d.CleanupState, FlushState: d.FlushState, Source: string(d.Source), Summary: d.Summary()}
}
