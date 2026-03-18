package winfsp

import (
	"fmt"
	"sync"
)

type ABIVolumeInfo struct {
	Name                    string
	ReadOnly, CaseSensitive bool
	MaxComponentLength      uint32
}
type ABIFileInfo struct {
	Path        string
	NodeID      uint64
	Size        uint64
	Mode        uint32
	IsDirectory bool
}
type ABIDirectoryEntry struct {
	Path        string
	NodeID      uint64
	Size        uint64
	IsDirectory bool
}
type DispatcherABIState struct {
	BridgeReady             bool
	RootPath                string
	Requests, ActiveHandles uint64
	LastOperation           string
	LastNTStatus            NTStatus
	LastError               string
}

func (s DispatcherABIState) Summary() string {
	summary := fmt.Sprintf("bridge_ready=%v root=%s requests=%d active_handles=%d", s.BridgeReady, defaultDispatcherValue(s.RootPath, "-"), s.Requests, s.ActiveHandles)
	if s.LastOperation != "" {
		summary += " last_op=" + s.LastOperation
	}
	if s.LastError != "" {
		summary += " last_error=" + s.LastError
	}
	if s.LastNTStatus != 0 {
		summary += fmt.Sprintf(" ntstatus=0x%08x", uint32(s.LastNTStatus))
	}
	return summary
}

type DispatcherABI struct {
	mu     sync.Mutex
	bridge *DispatcherBridge
	state  DispatcherABIState
}

func NewDispatcherABI(bridge *DispatcherBridge) *DispatcherABI { return &DispatcherABI{bridge: bridge} }
func (a *DispatcherABI) Snapshot() DispatcherABIState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state
}
func (a *DispatcherABI) Initialize(rootPath string) error {
	if a.bridge == nil {
		return fmt.Errorf("dispatcher ABI bridge is not initialized")
	}
	if err := a.bridge.Initialize(rootPath); err != nil {
		a.record("Initialize", StatusInternalError, err.Error())
		return err
	}
	a.mu.Lock()
	a.state.BridgeReady = true
	a.state.RootPath = rootPath
	a.state.LastOperation = "Initialize"
	a.state.LastNTStatus = StatusSuccess
	a.state.LastError = ""
	a.mu.Unlock()
	return nil
}
func (a *DispatcherABI) GetVolumeInfo() (ABIVolumeInfo, NTStatus) {
	info, status := a.bridge.GetVolumeInfo()
	a.record("GetVolumeInfo", status, "")
	return ABIVolumeInfo{Name: info.Name, ReadOnly: info.ReadOnly, CaseSensitive: info.CaseSensitive, MaxComponentLength: info.MaxComponentLength}, status
}
func (a *DispatcherABI) GetFileInfo(path string) (ABIFileInfo, NTStatus) {
	info, status := a.bridge.GetFileInfo(path)
	a.record("GetFileInfo", status, path)
	return ABIFileInfo{Path: info.Path, NodeID: info.NodeID, Size: info.Size, Mode: info.Mode, IsDirectory: info.IsDirectory}, status
}
func (a *DispatcherABI) Open(path string) (uint64, ABIFileInfo, NTStatus) {
	result, status := a.bridge.Open(path)
	a.recordWithHandle("Open", status, path, status == StatusSuccess, false)
	return result.HandleID, ABIFileInfo{Path: result.Info.Path, NodeID: result.Info.NodeID, Size: result.Info.Size, Mode: result.Info.Mode, IsDirectory: result.Info.IsDirectory}, status
}
func (a *DispatcherABI) OpenDirectory(path string) (uint64, ABIFileInfo, NTStatus) {
	result, status := a.bridge.OpenDirectory(path)
	a.recordWithHandle("OpenDirectory", status, path, status == StatusSuccess, false)
	return result.HandleID, ABIFileInfo{Path: result.Info.Path, NodeID: result.Info.NodeID, Size: result.Info.Size, Mode: result.Info.Mode, IsDirectory: result.Info.IsDirectory}, status
}
func (a *DispatcherABI) ReadDirectory(handleID, cookie uint64, maxEntries uint32) ([]ABIDirectoryEntry, uint64, bool, NTStatus) {
	page, status := a.bridge.ReadDirectory(handleID, cookie, maxEntries)
	a.record("ReadDirectory", status, fmt.Sprintf("handle=%d", handleID))
	entries := make([]ABIDirectoryEntry, 0, len(page.Entries))
	for _, entry := range page.Entries {
		entries = append(entries, ABIDirectoryEntry{Path: entry.Path, NodeID: entry.NodeID, Size: entry.Size, IsDirectory: entry.IsDirectory})
	}
	return entries, page.NextCookie, page.EOF, status
}
func (a *DispatcherABI) Read(handleID uint64, offset int64, length uint32) ([]byte, bool, NTStatus) {
	data, eof, status := a.bridge.Read(handleID, offset, length)
	a.record("Read", status, fmt.Sprintf("handle=%d offset=%d length=%d eof=%v", handleID, offset, length, eof))
	return data, eof, status
}
func (a *DispatcherABI) Close(handleID uint64) NTStatus {
	status := a.bridge.Close(handleID)
	a.recordWithHandle("Close", status, fmt.Sprintf("handle=%d", handleID), false, status == StatusSuccess)
	return status
}
func (a *DispatcherABI) record(op string, status NTStatus, detail string) {
	a.recordWithHandle(op, status, detail, false, false)
}
func (a *DispatcherABI) recordWithHandle(op string, status NTStatus, detail string, openHandle, closeHandle bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Requests++
	a.state.LastOperation = op
	a.state.LastNTStatus = status
	if openHandle && status == StatusSuccess {
		a.state.ActiveHandles++
	}
	if closeHandle && status == StatusSuccess && a.state.ActiveHandles > 0 {
		a.state.ActiveHandles--
	}
	if status == StatusSuccess {
		a.state.LastError = ""
		return
	}
	if detail == "" {
		detail = op
	}
	a.state.LastError = fmt.Sprintf("%s failed with ntstatus=0x%08x", detail, uint32(status))
}
