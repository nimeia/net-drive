package winfsp

import (
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
type Callbacks struct{ adapter *adapterpkg.Adapter }

func NewCallbacks(a *adapterpkg.Adapter) *Callbacks { return &Callbacks{adapter: a} }
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
	return OpenResult{HandleID: result.HandleID, Info: result.Info}, StatusSuccess
}
func (c *Callbacks) OpenDirectory(path string) (OpenResult, NTStatus) {
	result, err := c.adapter.OpenDirectory(path)
	if err != nil {
		return OpenResult{}, mapError(err)
	}
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
func (c *Callbacks) Close(handleID uint64) NTStatus {
	if err := c.adapter.Close(handleID); err != nil {
		return mapError(err)
	}
	return StatusSuccess
}
func (c *Callbacks) Snapshot() mountcore.Snapshot { return c.adapter.Snapshot() }
