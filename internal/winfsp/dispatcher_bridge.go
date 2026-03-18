package winfsp

import (
	"fmt"
	"sync"
)

type DispatcherBridgeState struct {
	Initialized  bool
	RootPath     string
	VolumeName   string
	LastNTStatus NTStatus
	LastError    string
	CallCount    map[string]uint64
}

func (s DispatcherBridgeState) Summary() string {
	status := fmt.Sprintf("initialized=%v root=%s volume=%s", s.Initialized, defaultDispatcherValue(s.RootPath, "-"), defaultDispatcherValue(s.VolumeName, "-"))
	if s.LastError != "" {
		status += fmt.Sprintf(" last_error=%s", s.LastError)
	}
	if s.LastNTStatus != 0 {
		status += fmt.Sprintf(" ntstatus=0x%08x", uint32(s.LastNTStatus))
	}
	return status
}

type DispatcherBridge struct {
	mu        sync.Mutex
	callbacks *Callbacks
	state     DispatcherBridgeState
}

func NewDispatcherBridge(callbacks *Callbacks) *DispatcherBridge {
	return &DispatcherBridge{callbacks: callbacks, state: DispatcherBridgeState{CallCount: map[string]uint64{}}}
}
func (b *DispatcherBridge) Snapshot() DispatcherBridgeState {
	b.mu.Lock()
	defer b.mu.Unlock()
	c := b.state
	c.CallCount = map[string]uint64{}
	for k, v := range b.state.CallCount {
		c.CallCount[k] = v
	}
	return c
}
func (b *DispatcherBridge) Initialize(rootPath string) error {
	if rootPath == "" {
		rootPath = "/"
	}
	info, status := b.GetVolumeInfo()
	if status != StatusSuccess {
		return StatusError(status, fmt.Errorf("GetVolumeInfo failed during dispatcher bridge initialization"))
	}
	if _, status := b.GetFileInfo(rootPath); status != StatusSuccess {
		return StatusError(status, fmt.Errorf("GetFileInfo(%s) failed during dispatcher bridge initialization", rootPath))
	}
	b.mu.Lock()
	b.state.Initialized = true
	b.state.RootPath = rootPath
	b.state.VolumeName = info.Name
	b.mu.Unlock()
	return nil
}
func (b *DispatcherBridge) GetVolumeInfo() (VolumeInfo, NTStatus) {
	info, status := b.callbacks.GetVolumeInfo()
	b.record("GetVolumeInfo", status, "")
	return info, status
}
func (b *DispatcherBridge) GetFileInfo(path string) (FileInfo, NTStatus) {
	info, status := b.callbacks.GetFileInfo(path)
	b.record("GetFileInfo", status, path)
	return info, status
}
func (b *DispatcherBridge) Open(path string) (OpenResult, NTStatus) {
	result, status := b.callbacks.Open(path)
	b.record("Open", status, path)
	return result, status
}
func (b *DispatcherBridge) OpenDirectory(path string) (OpenResult, NTStatus) {
	result, status := b.callbacks.OpenDirectory(path)
	b.record("OpenDirectory", status, path)
	return result, status
}
func (b *DispatcherBridge) ReadDirectory(handleID uint64, cookie uint64, maxEntries uint32) (DirectoryPage, NTStatus) {
	page, status := b.callbacks.ReadDirectory(handleID, cookie, maxEntries)
	b.record("ReadDirectory", status, fmt.Sprintf("handle=%d cookie=%d max=%d", handleID, cookie, maxEntries))
	return page, status
}
func (b *DispatcherBridge) Read(handleID uint64, offset int64, length uint32) ([]byte, bool, NTStatus) {
	data, eof, status := b.callbacks.Read(handleID, offset, length)
	b.record("Read", status, fmt.Sprintf("handle=%d offset=%d length=%d eof=%v", handleID, offset, length, eof))
	return data, eof, status
}
func (b *DispatcherBridge) Close(handleID uint64) NTStatus {
	status := b.callbacks.Close(handleID)
	b.record("Close", status, fmt.Sprintf("handle=%d", handleID))
	return status
}
func (b *DispatcherBridge) record(op string, status NTStatus, detail string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state.CallCount == nil {
		b.state.CallCount = map[string]uint64{}
	}
	b.state.CallCount[op]++
	b.state.LastNTStatus = status
	if status == StatusSuccess {
		b.state.LastError = ""
		return
	}
	if detail == "" {
		detail = op
	}
	b.state.LastError = fmt.Sprintf("%s failed with ntstatus=0x%08x", detail, uint32(status))
}
func defaultDispatcherValue(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
