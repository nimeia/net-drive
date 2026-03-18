package adapter

import "developer-mount/internal/mountcore"

type Adapter struct{ mount *mountcore.Mount }
type OpenResult struct {
	HandleID uint64
	Info     mountcore.FileInfo
}
type VolumeInfo = mountcore.VolumeInfo
type DirectoryPage = mountcore.DirectoryPage
type ReadResult = mountcore.ReadResult

func New(mount *mountcore.Mount) *Adapter                              { return &Adapter{mount: mount} }
func (a *Adapter) GetVolumeInfo() VolumeInfo                           { return a.mount.VolumeInfo() }
func (a *Adapter) GetFileInfo(path string) (mountcore.FileInfo, error) { return a.mount.GetAttr(path) }
func (a *Adapter) Open(path string) (OpenResult, error) {
	h, err := a.mount.Open(path)
	if err != nil {
		return OpenResult{}, err
	}
	return OpenResult{HandleID: h.HandleID, Info: h.Info}, nil
}
func (a *Adapter) OpenDirectory(path string) (OpenResult, error) {
	h, err := a.mount.OpenDirectory(path)
	if err != nil {
		return OpenResult{}, err
	}
	return OpenResult{HandleID: h.HandleID, Info: h.Info}, nil
}
func (a *Adapter) ReadDirectory(handleID uint64, cookie uint64, maxEntries uint32) (DirectoryPage, error) {
	return a.mount.ReadDirectory(handleID, cookie, maxEntries)
}
func (a *Adapter) Read(handleID uint64, offset int64, length uint32) (ReadResult, error) {
	return a.mount.Read(handleID, offset, length)
}
func (a *Adapter) Close(handleID uint64) error  { return a.mount.Close(handleID) }
func (a *Adapter) Snapshot() mountcore.Snapshot { return a.mount.Snapshot() }
