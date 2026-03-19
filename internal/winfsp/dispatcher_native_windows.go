//go:build windows

package winfsp

import (
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	platformwindows "developer-mount/internal/platform/windows"
)

const (
	fileAttributeReadOnly  = 0x00000001
	fileAttributeDirectory = 0x00000010

	fileDirectoryFile = 0x00000001

	statusBufferTooSmall NTStatus = 0xC0000023
)

type nativeVolumeParams [504]byte

type nativeVolumeInfo struct {
	TotalSize         uint64
	FreeSize          uint64
	VolumeLabelLength uint16
	VolumeLabel       [32]uint16
}

type nativeFileInfo struct {
	FileAttributes uint32
	ReparseTag     uint32
	AllocationSize uint64
	FileSize       uint64
	CreationTime   uint64
	LastAccessTime uint64
	LastWriteTime  uint64
	ChangeTime     uint64
	IndexNumber    uint64
	HardLinks      uint32
	EaSize         uint32
}

type nativeDirInfo struct {
	Size        uint16
	FileInfo    nativeFileInfo
	NextOffset  uint64
	_padding    [16]byte
	FileNameBuf [1]uint16
}

type nativeFileSystem struct {
	Version               uint16
	_                     uint16
	UserContext           uintptr
	VolumeName            [256]uint16
	VolumeHandle          uintptr
	EnterOperation        uintptr
	LeaveOperation        uintptr
	Operations            [21]uintptr
	Interface             uintptr
	DispatcherThread      uintptr
	DispatcherThreadCount uint32
	DispatcherResult      uint32
	MountPoint            uintptr
	MountHandle           uintptr
	DebugLog              uint32
	OpGuardStrategy       uint32
	OpGuardLock           [8]byte
	ContextFlags          uint16
	ReservedFlags         uint16
}

type nativeFileSystemInterface struct {
	GetVolumeInfo      uintptr
	SetVolumeLabel     uintptr
	GetSecurityByName  uintptr
	Create             uintptr
	Open               uintptr
	Overwrite          uintptr
	Cleanup            uintptr
	Close              uintptr
	Read               uintptr
	Write              uintptr
	Flush              uintptr
	GetFileInfo        uintptr
	SetBasicInfo       uintptr
	SetFileSize        uintptr
	CanDelete          uintptr
	Rename             uintptr
	GetSecurity        uintptr
	SetSecurity        uintptr
	ReadDirectory      uintptr
	ResolveReparse     uintptr
	GetReparsePoint    uintptr
	SetReparsePoint    uintptr
	DeleteReparsePoint uintptr
	GetStreamInfo      uintptr
	GetDirInfoByName   uintptr
	Control            uintptr
	SetDelete          uintptr
	CreateEx           uintptr
	OverwriteEx        uintptr
	GetEa              uintptr
	SetEa              uintptr
	Obsolete0          uintptr
	DispatcherStopped  uintptr
	Reserved           [31]uintptr
}

type nativeFSContext struct {
	callbacks   *Callbacks
	addDirInfo  *syscall.Proc
	mountPoint  string
	volumeName  string
	readOnly    bool
	serviceRoot string
}

type nativeMountHandle struct {
	bindings  dispatcherBindings
	config    HostConfig
	callbacks *Callbacks
	iface     *nativeFileSystemInterface
	fs        uintptr
}

var (
	nativeFSRegistryMu sync.RWMutex
	nativeFSRegistry   = map[uintptr]*nativeFSContext{}

	procConvertStringSecurityDescriptorToSecurityDescriptor = syscall.NewLazyDLL("advapi32.dll").NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW")
	procLocalFree                                           = syscall.NewLazyDLL("kernel32.dll").NewProc("LocalFree")
	procFindFirstFileW                                      = syscall.NewLazyDLL("kernel32.dll").NewProc("FindFirstFileW")
	procFindClose                                           = syscall.NewLazyDLL("kernel32.dll").NewProc("FindClose")

	cbGetVolumeInfo     = syscall.NewCallback(goGetVolumeInfo)
	cbSetVolumeLabel    = syscall.NewCallback(goSetVolumeLabel)
	cbGetSecurityByName = syscall.NewCallback(goGetSecurityByName)
	cbCreate            = syscall.NewCallback(goCreate)
	cbOpen              = syscall.NewCallback(goOpen)
	cbOverwrite         = syscall.NewCallback(goOverwrite)
	cbCleanup           = syscall.NewCallback(goCleanup)
	cbClose             = syscall.NewCallback(goClose)
	cbRead              = syscall.NewCallback(goRead)
	cbWrite             = syscall.NewCallback(goWrite)
	cbFlush             = syscall.NewCallback(goFlush)
	cbGetFileInfo       = syscall.NewCallback(goGetFileInfo)
	cbSetBasicInfo      = syscall.NewCallback(goSetBasicInfo)
	cbSetFileSize       = syscall.NewCallback(goSetFileSize)
	cbCanDelete         = syscall.NewCallback(goCanDelete)
	cbRename            = syscall.NewCallback(goRename)
	cbGetSecurity       = syscall.NewCallback(goGetSecurity)
	cbSetSecurity       = syscall.NewCallback(goSetSecurity)
	cbReadDirectory     = syscall.NewCallback(goReadDirectory)
	cbGetDirInfoByName  = syscall.NewCallback(goGetDirInfoByName)
	cbSetDelete         = syscall.NewCallback(goSetDelete)
	cbDispatcherStopped = syscall.NewCallback(goDispatcherStopped)
)

func newNativeFileSystem(dllPath string, config HostConfig, callbacks *Callbacks) (*nativeMountHandle, error) {
	bindings, err := probeDispatcherBindings(dllPath)
	if err != nil {
		nativeTraceError("native.create.bindings", "failed to probe dispatcher bindings", map[string]string{"dll_path": dllPath, "error": err.Error()})
		return nil, err
	}
	nativeTraceInfo("native.create.begin", "creating WinFsp file system", map[string]string{"mount_point": config.MountPoint, "dll_path": dllPath})
	handle := &nativeMountHandle{bindings: bindings, config: config, callbacks: callbacks}
	iface := &nativeFileSystemInterface{
		GetVolumeInfo:     cbGetVolumeInfo,
		SetVolumeLabel:    cbSetVolumeLabel,
		GetSecurityByName: cbGetSecurityByName,
		Create:            cbCreate,
		Open:              cbOpen,
		Overwrite:         cbOverwrite,
		Cleanup:           cbCleanup,
		Close:             cbClose,
		Read:              cbRead,
		Write:             cbWrite,
		Flush:             cbFlush,
		GetFileInfo:       cbGetFileInfo,
		SetBasicInfo:      cbSetBasicInfo,
		SetFileSize:       cbSetFileSize,
		CanDelete:         cbCanDelete,
		Rename:            cbRename,
		GetSecurity:       cbGetSecurity,
		SetSecurity:       cbSetSecurity,
		ReadDirectory:     cbReadDirectory,
		GetDirInfoByName:  cbGetDirInfoByName,
		SetDelete:         cbSetDelete,
		DispatcherStopped: cbDispatcherStopped,
	}
	handle.iface = iface
	params := defaultVolumeParams(config, callbacks)
	devicePath, err := syscall.UTF16PtrFromString(winfspDiskDeviceName)
	if err != nil {
		return nil, err
	}
	var fs uintptr
	status, _, _ := bindings.create.Call(
		uintptr(unsafe.Pointer(devicePath)),
		uintptr(unsafe.Pointer(&params)),
		uintptr(unsafe.Pointer(iface)),
		uintptr(unsafe.Pointer(&fs)),
	)
	if NTStatus(status) != StatusSuccess {
		handle.Close()
		msg := fmt.Sprintf("FspFileSystemCreate failed with ntstatus=0x%08x", uint32(status))
		if uint32(status) == 0xc0000033 {
			msg += fmt.Sprintf(" (WinFsp rejected the create parameters; volume params or interface layout may be invalid, mount=%q)", winfspMountPointForAPI(config.MountPoint))
		}
		nativeTraceError("native.create.failed", msg, map[string]string{"mount_point": config.MountPoint, "ntstatus": fmt.Sprintf("0x%08x", uint32(status)), "status_name": StatusName(NTStatus(status))})
		return nil, fmt.Errorf(msg)
	}
	handle.fs = fs
	nativeTraceInfo("native.create.ready", "created WinFsp file system", map[string]string{"mount_point": config.MountPoint})
	registerNativeFS(fs, &nativeFSContext{
		callbacks:   callbacks,
		addDirInfo:  bindings.addDirInfo,
		mountPoint:  config.MountPoint,
		volumeName:  callbacksVolumeName(callbacks),
		readOnly:    true,
		serviceRoot: "/",
	})
	return handle, nil
}

func winfspMountPointForAPI(mountPoint string) string {
	return strings.TrimSpace(mountPoint)
}

func (h *nativeMountHandle) Start() error {
	if h.fs == 0 {
		return fmt.Errorf("native file system is not created")
	}
	nativeTraceInfo("native.start.begin", "starting WinFsp mount", map[string]string{"mount_point": h.config.MountPoint})
	var mountPtr uintptr
	if strings.TrimSpace(h.config.MountPoint) != "" {
		mount := winfspMountPointForAPI(h.config.MountPoint)
		ptr, err := syscall.UTF16PtrFromString(mount)
		if err != nil {
			return err
		}
		mountPtr = uintptr(unsafe.Pointer(ptr))
	}
	status, _, _ := h.bindings.setMountPoint.Call(h.fs, mountPtr)
	if NTStatus(status) != StatusSuccess {
		nativeTraceError("native.start.set_mount_point_failed", "FspFileSystemSetMountPoint failed", map[string]string{"mount_point": h.config.MountPoint, "ntstatus": fmt.Sprintf("0x%08x", uint32(status)), "status_name": StatusName(NTStatus(status))})
		return fmt.Errorf("FspFileSystemSetMountPoint(%s) failed with ntstatus=0x%08x", h.config.MountPoint, uint32(status))
	}
	status, _, _ = h.bindings.startDispatcher.Call(h.fs, 0)
	if NTStatus(status) != StatusSuccess {
		nativeTraceError("native.start.dispatcher_failed", "FspFileSystemStartDispatcher failed", map[string]string{"mount_point": h.config.MountPoint, "ntstatus": fmt.Sprintf("0x%08x", uint32(status)), "status_name": StatusName(NTStatus(status))})
		return fmt.Errorf("FspFileSystemStartDispatcher failed with ntstatus=0x%08x", uint32(status))
	}
	if err := probeMountedRoot(h.config.MountPoint); err != nil {
		nativeTraceError("native.start.root_probe_failed", "mounted root probe failed", map[string]string{"mount_point": h.config.MountPoint, "error": err.Error()})
		return err
	}
	nativeTraceInfo("native.start.ready", "WinFsp mount ready", map[string]string{"mount_point": h.config.MountPoint})
	return nil
}

func (h *nativeMountHandle) Stop() {
	if h.fs == 0 {
		return
	}
	nativeTraceInfo("native.stop", "stopping WinFsp mount", map[string]string{"mount_point": h.config.MountPoint})
	h.bindings.stopDispatcher.Call(h.fs)
	h.bindings.removeMountPoint.Call(h.fs)
}

func (h *nativeMountHandle) Close() {
	if h.fs != 0 {
		nativeTraceInfo("native.close", "closing WinFsp file system", map[string]string{"mount_point": h.config.MountPoint})
		unregisterNativeFS(h.fs)
		h.bindings.deleteFS.Call(h.fs)
		h.fs = 0
	}
	if h.bindings.dll != nil {
		_ = h.bindings.dll.Release()
		h.bindings.dll = nil
	}
}

func defaultVolumeParams(config HostConfig, callbacks *Callbacks) nativeVolumeParams {
	info, _ := callbacks.GetVolumeInfo()
	var params nativeVolumeParams
	putUint16(params[:], 0, uint16(len(params)))
	putUint16(params[:], 2, 4096)
	putUint16(params[:], 4, 1)
	putUint16(params[:], 6, uint16(info.MaxComponentLength))
	putUint64(params[:], 8, windowsFileTime(time.Now()))
	putUint32(params[:], 16, uint32(time.Now().Unix()))
	putUint32(params[:], 24, 300000)
	putUint32(params[:], 28, 1000)
	putUint32(params[:], 32, 1000)
	fileSystemAttributes := uint32(0)
	fileSystemAttributes |= 1 << 1 // CasePreservedNames
	fileSystemAttributes |= 1 << 2 // UnicodeOnDisk
	fileSystemAttributes |= 1 << 3 // PersistentAcls
	fileSystemAttributes |= 1 << 9 // ReadOnlyVolume
	if info.CaseSensitive {
		fileSystemAttributes |= 1 << 0
	}
	putUint32(params[:], 36, fileSystemAttributes)
	additionalFlags := uint32(0)
	additionalFlags |= 1 << 10 // PostCleanupWhenModifiedOnly
	additionalFlags |= 1 << 16 // UmFileContextIsUserContext2
	putUint32(params[:], 36, fileSystemAttributes|additionalFlags)
	copyUTF16At(params[:], 424, 16, "devmnt")
	if strings.HasPrefix(config.MountPoint, `\\`) {
		copyUTF16At(params[:], 40, 192, config.VolumePrefix)
	}
	return params
}

func registerNativeFS(fs uintptr, ctx *nativeFSContext) {
	nativeFSRegistryMu.Lock()
	defer nativeFSRegistryMu.Unlock()
	nativeFSRegistry[fs] = ctx
}

func unregisterNativeFS(fs uintptr) {
	nativeFSRegistryMu.Lock()
	defer nativeFSRegistryMu.Unlock()
	delete(nativeFSRegistry, fs)
}

func lookupNativeFS(fs uintptr) *nativeFSContext {
	nativeFSRegistryMu.RLock()
	defer nativeFSRegistryMu.RUnlock()
	return nativeFSRegistry[fs]
}

func goGetVolumeInfo(fileSystem, volumeInfo uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	info, status := ctx.callbacks.GetVolumeInfo()
	if status != StatusSuccess {
		return uintptr(status)
	}
	fillNativeVolumeInfo((*nativeVolumeInfo)(unsafe.Pointer(volumeInfo)), info)
	return uintptr(StatusSuccess)
}

func goSetVolumeLabel(fileSystem, volumeLabel, volumeInfo uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	info, status := ctx.callbacks.GetVolumeInfo()
	if status == StatusSuccess && volumeInfo != 0 {
		fillNativeVolumeInfo((*nativeVolumeInfo)(unsafe.Pointer(volumeInfo)), info)
	}
	return uintptr(StatusAccessDenied)
}

func goGetSecurityByName(fileSystem, fileName, fileAttributes, securityDescriptor, securityDescriptorSize uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		nativeTraceError("native.cb.get_security_by_name", "missing native fs context", nil)
		return uintptr(StatusInternalError)
	}
	path, err := utf16PtrToMountPath(fileName)
	if err != nil {
		nativeTraceError("native.cb.get_security_by_name", "invalid path", map[string]string{"error": err.Error()})
		return uintptr(StatusObjectPathNotFound)
	}
	info, status := ctx.callbacks.GetSecurityByName(path)
	if status != StatusSuccess {
		nativeTraceError("native.cb.get_security_by_name", "GetSecurityByName failed", map[string]string{"path": path, "ntstatus": fmt.Sprintf("0x%08x", uint32(status)), "status_name": StatusName(status)})
		return uintptr(status)
	}
	if fileAttributes != 0 {
		attr, err := fileAttributesForPath(ctx.callbacks, path)
		if err == nil {
			*(*uint32)(unsafe.Pointer(fileAttributes)) = attr
		}
	}
	result := copySecurityDescriptor(info.Descriptor, securityDescriptor, securityDescriptorSize)
	nativeTraceInfo("native.cb.get_security_by_name", "GetSecurityByName completed", map[string]string{"path": path, "result": fmt.Sprintf("0x%08x", uint32(result))})
	return result
}

func goCreate(fileSystem, fileName, createOptions, grantedAccess, fileAttributes, securityDescriptor, allocationSize, fileContext, fileInfo uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	path, err := utf16PtrToMountPath(fileName)
	if err != nil {
		return uintptr(StatusObjectPathNotFound)
	}
	isDirectory := (uint32(createOptions) & fileDirectoryFile) != 0
	status := ctx.callbacks.Create(path, isDirectory)
	if status != StatusSuccess {
		return uintptr(status)
	}
	return uintptr(StatusAccessDenied)
}

func goOpen(fileSystem, fileName, createOptions, grantedAccess, fileContext, fileInfo uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		nativeTraceError("native.cb.open", "missing native fs context", nil)
		return uintptr(StatusInternalError)
	}
	path, err := utf16PtrToMountPath(fileName)
	if err != nil {
		nativeTraceError("native.cb.open", "invalid path", map[string]string{"error": err.Error()})
		return uintptr(StatusObjectPathNotFound)
	}
	meta, metaStatus := ctx.callbacks.GetFileInfo(path)
	if metaStatus != StatusSuccess {
		nativeTraceError("native.cb.open", "GetFileInfo failed", map[string]string{"path": path, "ntstatus": fmt.Sprintf("0x%08x", uint32(metaStatus)), "status_name": StatusName(metaStatus)})
		return uintptr(metaStatus)
	}
	var result OpenResult
	var status NTStatus
	if meta.IsDirectory || (uint32(createOptions)&fileDirectoryFile) != 0 {
		result, status = ctx.callbacks.OpenDirectory(path)
	} else {
		result, status = ctx.callbacks.Open(path)
	}
	if status != StatusSuccess {
		nativeTraceError("native.cb.open", "open failed", map[string]string{"path": path, "ntstatus": fmt.Sprintf("0x%08x", uint32(status)), "status_name": StatusName(status)})
		return uintptr(status)
	}
	*(*uintptr)(unsafe.Pointer(fileContext)) = uintptr(result.HandleID)
	fillNativeFileInfo((*nativeFileInfo)(unsafe.Pointer(fileInfo)), result.Info, true)
	nativeTraceInfo("native.cb.open", "open completed", map[string]string{"path": path, "handle_id": fmt.Sprintf("%d", result.HandleID), "directory": fmt.Sprintf("%t", result.Info.IsDirectory)})
	return uintptr(StatusSuccess)
}

func goOverwrite(fileSystem, fileContext, fileAttributes, replaceFileAttributes, allocationSize, fileInfo uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	status := ctx.callbacks.Overwrite(uint64(fileContext), uint64(allocationSize), uint32(fileAttributes), replaceFileAttributes != 0)
	if status == StatusSuccess && fileInfo != 0 {
		if info, st := ctx.callbacks.GetFileInfoByHandle(uint64(fileContext)); st == StatusSuccess {
			fillNativeFileInfo((*nativeFileInfo)(unsafe.Pointer(fileInfo)), info, true)
		}
	}
	return uintptr(status)
}

func goCleanup(fileSystem, fileContext, fileName, flags uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return 0
	}
	_ = ctx.callbacks.Cleanup(uint64(fileContext))
	return 0
}

func goClose(fileSystem, fileContext uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return 0
	}
	_ = ctx.callbacks.Close(uint64(fileContext))
	return 0
}

func goRead(fileSystem, fileContext, buffer, offset, length, bytesTransferred uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	data, _, status := ctx.callbacks.Read(uint64(fileContext), int64(offset), uint32(length))
	if status != StatusSuccess {
		return uintptr(status)
	}
	if len(data) > int(length) {
		data = data[:int(length)]
	}
	if len(data) > 0 {
		copy(unsafe.Slice((*byte)(unsafe.Pointer(buffer)), len(data)), data)
	}
	*(*uint32)(unsafe.Pointer(bytesTransferred)) = uint32(len(data))
	return uintptr(StatusSuccess)
}

func goWrite(fileSystem, fileContext, buffer, offset, length, writeToEndOfFile, constrainedIo, bytesTransferred, fileInfo uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	var data []byte
	if length > 0 {
		data = append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(buffer)), int(length))...)
	}
	written, status := ctx.callbacks.Write(uint64(fileContext), int64(offset), data, constrainedIo != 0)
	*(*uint32)(unsafe.Pointer(bytesTransferred)) = written
	if status == StatusSuccess && fileInfo != 0 {
		if info, st := ctx.callbacks.GetFileInfoByHandle(uint64(fileContext)); st == StatusSuccess {
			fillNativeFileInfo((*nativeFileInfo)(unsafe.Pointer(fileInfo)), info, true)
		}
	}
	return uintptr(status)
}

func goFlush(fileSystem, fileContext, fileInfo uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	if fileContext == 0 {
		return uintptr(StatusSuccess)
	}
	status := ctx.callbacks.Flush(uint64(fileContext))
	if status == StatusSuccess && fileInfo != 0 {
		if info, st := ctx.callbacks.GetFileInfoByHandle(uint64(fileContext)); st == StatusSuccess {
			fillNativeFileInfo((*nativeFileInfo)(unsafe.Pointer(fileInfo)), info, true)
		}
	}
	return uintptr(status)
}

func goGetFileInfo(fileSystem, fileContext, fileInfo uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	info, status := ctx.callbacks.GetFileInfoByHandle(uint64(fileContext))
	if status != StatusSuccess {
		return uintptr(status)
	}
	fillNativeFileInfo((*nativeFileInfo)(unsafe.Pointer(fileInfo)), info, true)
	return uintptr(StatusSuccess)
}

func goSetBasicInfo(fileSystem, fileContext, fileAttributes, creationTime, lastAccessTime, lastWriteTime, changeTime, fileInfo uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	status := ctx.callbacks.SetBasicInfo(uint64(fileContext), uint32(fileAttributes))
	if status == StatusSuccess && fileInfo != 0 {
		if info, st := ctx.callbacks.GetFileInfoByHandle(uint64(fileContext)); st == StatusSuccess {
			fillNativeFileInfo((*nativeFileInfo)(unsafe.Pointer(fileInfo)), info, true)
		}
	}
	return uintptr(status)
}

func goSetFileSize(fileSystem, fileContext, newSize, setAllocationSize, fileInfo uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	status := ctx.callbacks.SetFileSize(uint64(fileContext), int64(newSize), setAllocationSize != 0)
	if status == StatusSuccess && fileInfo != 0 {
		if info, st := ctx.callbacks.GetFileInfoByHandle(uint64(fileContext)); st == StatusSuccess {
			fillNativeFileInfo((*nativeFileInfo)(unsafe.Pointer(fileInfo)), info, true)
		}
	}
	return uintptr(status)
}

func goCanDelete(fileSystem, fileContext, fileName uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	path, err := resolvePathForContext(ctx.callbacks, uint64(fileContext), fileName)
	if err != nil {
		return uintptr(StatusObjectPathNotFound)
	}
	return uintptr(ctx.callbacks.CanDelete(path))
}

func goRename(fileSystem, fileContext, fileName, newFileName, replaceIfExists uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	newPath, err := utf16PtrToMountPath(newFileName)
	if err != nil {
		return uintptr(StatusObjectPathNotFound)
	}
	return uintptr(ctx.callbacks.Rename(uint64(fileContext), newPath, replaceIfExists != 0))
}

func goGetSecurity(fileSystem, fileContext, securityDescriptor, securityDescriptorSize uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		nativeTraceError("native.cb.get_security", "missing native fs context", nil)
		return uintptr(StatusInternalError)
	}
	info, status := ctx.callbacks.GetSecurity(uint64(fileContext))
	if status != StatusSuccess {
		nativeTraceError("native.cb.get_security", "GetSecurity failed", map[string]string{"handle_id": fmt.Sprintf("%d", fileContext), "ntstatus": fmt.Sprintf("0x%08x", uint32(status)), "status_name": StatusName(status)})
		return uintptr(status)
	}
	result := copySecurityDescriptor(info.Descriptor, securityDescriptor, securityDescriptorSize)
	nativeTraceInfo("native.cb.get_security", "GetSecurity completed", map[string]string{"handle_id": fmt.Sprintf("%d", fileContext), "result": fmt.Sprintf("0x%08x", uint32(result))})
	return result
}

func goSetSecurity(fileSystem, fileContext, securityInformation, modificationDescriptor uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	return uintptr(ctx.callbacks.SetSecurity(uint64(fileContext), ""))
}

func goReadDirectory(fileSystem, fileContext, pattern, marker, buffer, length, bytesTransferred uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		nativeTraceError("native.cb.read_directory", "missing native fs context", nil)
		return uintptr(StatusInternalError)
	}
	entries, status := collectDirectoryEntries(ctx.callbacks, uint64(fileContext))
	if status != StatusSuccess {
		nativeTraceError("native.cb.read_directory", "collectDirectoryEntries failed", map[string]string{"handle_id": fmt.Sprintf("%d", fileContext), "ntstatus": fmt.Sprintf("0x%08x", uint32(status)), "status_name": StatusName(status)})
		return uintptr(status)
	}
	markerText := utf16PtrToString(marker)
	patternText := utf16PtrToString(pattern)
	nativeTraceInfo("native.cb.read_directory", "ReadDirectory started", map[string]string{"handle_id": fmt.Sprintf("%d", fileContext), "entries": fmt.Sprintf("%d", len(entries)), "marker": markerText, "pattern": patternText})
	*(*uint32)(unsafe.Pointer(bytesTransferred)) = 0
	complete := true
	for _, entry := range entries {
		name := entry.Name
		if name == "" {
			continue
		}
		if markerText != "" && strings.Compare(strings.ToLower(name), strings.ToLower(markerText)) <= 0 {
			continue
		}
		dirInfo, backing := newNativeDirInfo(entry)
		ok, _, _ := ctx.addDirInfo.Call(
			uintptr(unsafe.Pointer(dirInfo)),
			buffer,
			length,
			bytesTransferred,
		)
		_ = backing
		if ok == 0 {
			nativeTraceInfo("native.cb.read_directory", "directory buffer filled before EOF", map[string]string{"handle_id": fmt.Sprintf("%d", fileContext), "entry": name})
			complete = false
			break
		}
	}
	if complete {
		ctx.addDirInfo.Call(0, buffer, length, bytesTransferred)
	}
	nativeTraceInfo("native.cb.read_directory", "ReadDirectory completed", map[string]string{"handle_id": fmt.Sprintf("%d", fileContext), "bytes_transferred": fmt.Sprintf("%d", *(*uint32)(unsafe.Pointer(bytesTransferred))), "complete": fmt.Sprintf("%t", complete)})
	return uintptr(StatusSuccess)
}

func goGetDirInfoByName(fileSystem, fileContext, fileName, dirInfo uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		nativeTraceError("native.cb.get_dir_info_by_name", "missing native fs context", nil)
		return uintptr(StatusInternalError)
	}
	name := utf16PtrToString(fileName)
	parent, status := ctx.callbacks.GetFileInfoByHandle(uint64(fileContext))
	if status != StatusSuccess {
		nativeTraceError("native.cb.get_dir_info_by_name", "GetFileInfoByHandle failed", map[string]string{"handle_id": fmt.Sprintf("%d", fileContext), "ntstatus": fmt.Sprintf("0x%08x", uint32(status)), "status_name": StatusName(status)})
		return uintptr(status)
	}
	childPath := platformwindows.JoinMountPath(parent.Path, name)
	info, status := ctx.callbacks.GetFileInfo(childPath)
	if status != StatusSuccess {
		nativeTraceError("native.cb.get_dir_info_by_name", "GetFileInfo failed", map[string]string{"path": childPath, "ntstatus": fmt.Sprintf("0x%08x", uint32(status)), "status_name": StatusName(status)})
		return uintptr(status)
	}
	writeNativeDirInfo((*nativeDirInfo)(unsafe.Pointer(dirInfo)), info)
	nativeTraceInfo("native.cb.get_dir_info_by_name", "GetDirInfoByName completed", map[string]string{"path": childPath, "name": name})
	return uintptr(StatusSuccess)
}

func goSetDelete(fileSystem, fileContext, fileName, deleteFile uintptr) uintptr {
	ctx := lookupNativeFS(fileSystem)
	if ctx == nil {
		return uintptr(StatusInternalError)
	}
	return uintptr(ctx.callbacks.SetDeleteOnClose(uint64(fileContext), deleteFile != 0))
}

func goDispatcherStopped(fileSystem, normally uintptr) uintptr {
	return 0
}

func fillNativeVolumeInfo(dst *nativeVolumeInfo, info VolumeInfo) {
	if dst == nil {
		return
	}
	dst.TotalSize = 1 << 40
	dst.FreeSize = 1 << 40
	copyUTF16Slice(dst.VolumeLabel[:], info.Name)
	dst.VolumeLabelLength = uint16(len(syscall.StringToUTF16(info.Name[:min(len(info.Name), 31)]))-1) * 2
}

func fillNativeFileInfo(dst *nativeFileInfo, info FileInfo, readOnly bool) {
	if dst == nil {
		return
	}
	dst.FileAttributes = fileAttributesFromInfo(info, readOnly)
	if info.IsDirectory {
		dst.AllocationSize = 0
		dst.FileSize = 0
	} else {
		dst.FileSize = uint64(max64(info.Size, 0))
		dst.AllocationSize = roundUp(dst.FileSize, 4096)
	}
	mod := parseModTime(info.ModTime)
	dst.CreationTime = mod
	dst.LastAccessTime = mod
	dst.LastWriteTime = mod
	dst.ChangeTime = mod
	dst.IndexNumber = info.NodeID
	dst.HardLinks = 0
	dst.EaSize = 0
}

func writeNativeDirInfo(dst *nativeDirInfo, info FileInfo) {
	if dst == nil {
		return
	}
	name := syscall.StringToUTF16(info.Name)
	nameWords := utf16ContentWords(name)
	backing := make([]byte, 104+nameWords*2)
	tmp := (*nativeDirInfo)(unsafe.Pointer(&backing[0]))
	tmp.Size = uint16(len(backing))
	fillNativeFileInfo(&tmp.FileInfo, info, true)
	if nameWords > 0 {
		copy(unsafe.Slice((*uint16)(unsafe.Pointer(&tmp.FileNameBuf[0])), nameWords), name[:nameWords])
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(dst)), len(backing)), backing)
}

func newNativeDirInfo(info FileInfo) (*nativeDirInfo, []byte) {
	name := syscall.StringToUTF16(info.Name)
	nameWords := utf16ContentWords(name)
	size := 104 + nameWords*2
	backing := make([]byte, size)
	dir := (*nativeDirInfo)(unsafe.Pointer(&backing[0]))
	dir.Size = uint16(size)
	fillNativeFileInfo(&dir.FileInfo, info, true)
	if nameWords > 0 {
		copy(unsafe.Slice((*uint16)(unsafe.Pointer(&dir.FileNameBuf[0])), nameWords), name[:nameWords])
	}
	return dir, backing
}

func utf16ContentWords(value []uint16) int {
	if len(value) == 0 {
		return 0
	}
	return len(value) - 1
}

type win32FindData struct {
	FileAttributes    uint32
	CreationTime      [2]uint32
	LastAccessTime    [2]uint32
	LastWriteTime     [2]uint32
	FileSizeHigh      uint32
	FileSizeLow       uint32
	Reserved0         uint32
	Reserved1         uint32
	FileName          [260]uint16
	AlternateFileName [14]uint16
}

func probeMountedRoot(mountPoint string) error {
	probePath := strings.TrimSpace(mountPoint)
	if probePath == "" {
		return nil
	}
	if len(probePath) == 2 && probePath[1] == ':' {
		probePath += `\*`
	} else {
		probePath = strings.TrimRight(probePath, `\/`) + `\*`
	}
	ptr, err := syscall.UTF16PtrFromString(probePath)
	if err != nil {
		return err
	}
	var lastErr syscall.Errno
	for attempt := 0; attempt < 20; attempt++ {
		var data win32FindData
		handle, _, callErr := procFindFirstFileW.Call(uintptr(unsafe.Pointer(ptr)), uintptr(unsafe.Pointer(&data)))
		if handle != ^uintptr(0) {
			procFindClose.Call(handle)
			return nil
		}
		if errno, ok := callErr.(syscall.Errno); ok {
			if errno == 2 || errno == 18 {
				return nil
			}
			lastErr = errno
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr == 0 {
		lastErr = 1359
	}
	return fmt.Errorf("mount point %s is visible but root enumeration failed with win32=%d", mountPoint, lastErr)
}

func collectDirectoryEntries(callbacks *Callbacks, handleID uint64) ([]FileInfo, NTStatus) {
	var entries []FileInfo
	var cookie uint64
	for {
		page, status := callbacks.ReadDirectory(handleID, cookie, 256)
		if status != StatusSuccess {
			return nil, status
		}
		entries = append(entries, page.Entries...)
		if page.EOF {
			return entries, StatusSuccess
		}
		cookie = page.NextCookie
	}
}

func resolvePathForContext(callbacks *Callbacks, handleID uint64, fileName uintptr) (string, error) {
	if fileName != 0 {
		return utf16PtrToMountPath(fileName)
	}
	info, status := callbacks.GetFileInfoByHandle(handleID)
	if status != StatusSuccess {
		return "", fmt.Errorf("handle %d: %s", handleID, StatusName(status))
	}
	return info.Path, nil
}

func fileAttributesForPath(callbacks *Callbacks, path string) (uint32, error) {
	info, status := callbacks.GetFileInfo(path)
	if status != StatusSuccess {
		return 0, fmt.Errorf("get file info: %s", StatusName(status))
	}
	return fileAttributesFromInfo(info, true), nil
}

func fileAttributesFromInfo(info FileInfo, readOnly bool) uint32 {
	attr := uint32(0)
	if readOnly {
		attr |= fileAttributeReadOnly
	}
	if info.IsDirectory {
		attr |= fileAttributeDirectory
	}
	return attr
}

func utf16PtrToString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	return syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(ptr))[:])
}

func utf16PtrToMountPath(ptr uintptr) (string, error) {
	return platformwindows.NormalizeMountPath(utf16PtrToString(ptr))
}

func copyUTF16Slice(dst []uint16, value string) {
	src := syscall.StringToUTF16(value)
	if len(src) > len(dst) {
		src = src[:len(dst)]
		if len(src) > 0 {
			src[len(src)-1] = 0
		}
	}
	copy(dst, src)
}

func copyUTF16At(buf []byte, offset int, maxWords int, value string) {
	src := syscall.StringToUTF16(value)
	if len(src) > maxWords {
		src = src[:maxWords]
		if len(src) > 0 {
			src[len(src)-1] = 0
		}
	}
	for i, ch := range src {
		putUint16(buf, offset+i*2, ch)
	}
}

func putUint16(buf []byte, offset int, value uint16) {
	buf[offset+0] = byte(value)
	buf[offset+1] = byte(value >> 8)
}

func putUint32(buf []byte, offset int, value uint32) {
	buf[offset+0] = byte(value)
	buf[offset+1] = byte(value >> 8)
	buf[offset+2] = byte(value >> 16)
	buf[offset+3] = byte(value >> 24)
}

func putUint64(buf []byte, offset int, value uint64) {
	putUint32(buf, offset, uint32(value))
	putUint32(buf, offset+4, uint32(value>>32))
}

func callbacksVolumeName(callbacks *Callbacks) string {
	info, status := callbacks.GetVolumeInfo()
	if status != StatusSuccess {
		return "devmount"
	}
	return info.Name
}

func copySecurityDescriptor(sddl string, dstPtr, sizePtr uintptr) uintptr {
	if sizePtr == 0 {
		return uintptr(StatusInternalError)
	}
	sizeInOut := (*uintptr)(unsafe.Pointer(sizePtr))
	if strings.TrimSpace(sddl) == "" {
		*sizeInOut = 0
		return uintptr(StatusSuccess)
	}
	src, length, err := securityDescriptorFromSDDL(sddl)
	if err != nil {
		return uintptr(StatusInternalError)
	}
	if dstPtr == 0 || *sizeInOut < uintptr(length) {
		*sizeInOut = uintptr(length)
		return uintptr(statusBufferTooSmall)
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(dstPtr)), length), src)
	*sizeInOut = uintptr(length)
	return uintptr(StatusSuccess)
}

func securityDescriptorFromSDDL(sddl string) ([]byte, uint32, error) {
	ptr, err := syscall.UTF16PtrFromString(sddl)
	if err != nil {
		return nil, 0, err
	}
	var sd uintptr
	var size uint32
	r1, _, callErr := procConvertStringSecurityDescriptorToSecurityDescriptor.Call(
		uintptr(unsafe.Pointer(ptr)),
		1,
		uintptr(unsafe.Pointer(&sd)),
		uintptr(unsafe.Pointer(&size)),
	)
	if r1 == 0 {
		return nil, 0, callErr
	}
	defer procLocalFree.Call(sd)
	buf := append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(sd)), int(size))...)
	return buf, size, nil
}

func parseModTime(value string) uint64 {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return 0
		}
	}
	return windowsFileTime(t)
}

func windowsFileTime(t time.Time) uint64 {
	if t.IsZero() {
		return 0
	}
	return uint64(t.UTC().UnixNano()/100) + 116444736000000000
}

func roundUp(v uint64, unit uint64) uint64 {
	if v == 0 || unit == 0 {
		return v
	}
	rem := v % unit
	if rem == 0 {
		return v
	}
	return v + unit - rem
}

func max64(v int64, floor int64) int64 {
	if v < floor {
		return floor
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
