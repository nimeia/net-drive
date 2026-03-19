//go:build windows

package winclientgui

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	bifReturnOnlyFSDirs = 0x0001
	bifNewDialogStyle   = 0x0040
	maxPath             = 260
)

type browseInfo struct {
	HwndOwner      uintptr
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}

var (
	ole32                 = syscall.NewLazyDLL("ole32.dll")
	procSHBrowseForFolder = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromID   = shell32.NewProc("SHGetPathFromIDListW")
	procCoTaskMemFree     = ole32.NewProc("CoTaskMemFree")
)

func chooseFolder(owner uintptr, title string) (string, bool, error) {
	display := make([]uint16, maxPath)
	info := browseInfo{
		HwndOwner:      owner,
		PszDisplayName: &display[0],
		LpszTitle:      syscall.StringToUTF16Ptr(title),
		UlFlags:        bifReturnOnlyFSDirs | bifNewDialogStyle,
	}
	pidl, _, _ := procSHBrowseForFolder.Call(uintptr(unsafe.Pointer(&info)))
	if pidl == 0 {
		return "", false, nil
	}
	defer procCoTaskMemFree.Call(pidl)
	path := make([]uint16, maxPath)
	ok, _, err := procSHGetPathFromID.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		return "", false, fmt.Errorf("SHGetPathFromIDListW failed: %w", err)
	}
	return syscall.UTF16ToString(path), true, nil
}
