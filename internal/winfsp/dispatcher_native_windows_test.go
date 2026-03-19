//go:build windows

package winfsp

import (
	"syscall"
	"testing"
	"unsafe"
)

func TestNativeWinFspStructSizes(t *testing.T) {
	if got := unsafe.Sizeof(nativeVolumeParams{}); got != 504 {
		t.Fatalf("sizeof(nativeVolumeParams) = %d, want 504", got)
	}
	if got := unsafe.Sizeof(nativeFileSystemInterface{}); got != 64*unsafe.Sizeof(uintptr(0)) {
		t.Fatalf("sizeof(nativeFileSystemInterface) = %d, want %d", got, 64*unsafe.Sizeof(uintptr(0)))
	}
}

func TestNewNativeDirInfoCopiesNameWithoutNul(t *testing.T) {
	info := FileInfo{Name: "alpha", IsDirectory: true}
	dir, backing := newNativeDirInfo(info)
	if dir == nil {
		t.Fatalf("newNativeDirInfo() returned nil dir")
	}
	wantWords := len(syscall.StringToUTF16(info.Name)) - 1
	wantSize := 104 + wantWords*2
	if len(backing) != wantSize {
		t.Fatalf("len(backing) = %d, want %d", len(backing), wantSize)
	}
	if int(dir.Size) != wantSize {
		t.Fatalf("dir.Size = %d, want %d", dir.Size, wantSize)
	}
	nameWords := unsafe.Slice((*uint16)(unsafe.Pointer(&dir.FileNameBuf[0])), wantWords)
	if got := syscall.UTF16ToString(append([]uint16(nil), nameWords...)); got != info.Name {
		t.Fatalf("FileNameBuf = %q, want %q", got, info.Name)
	}
}

func TestWriteNativeDirInfoCopiesNameWithoutNul(t *testing.T) {
	info := FileInfo{Name: "beta.txt"}
	wantWords := len(syscall.StringToUTF16(info.Name)) - 1
	backing := make([]byte, 104+wantWords*2)
	writeNativeDirInfo((*nativeDirInfo)(unsafe.Pointer(&backing[0])), info)
	dir := (*nativeDirInfo)(unsafe.Pointer(&backing[0]))
	if int(dir.Size) != len(backing) {
		t.Fatalf("dir.Size = %d, want %d", dir.Size, len(backing))
	}
	nameWords := unsafe.Slice((*uint16)(unsafe.Pointer(&dir.FileNameBuf[0])), wantWords)
	if got := syscall.UTF16ToString(append([]uint16(nil), nameWords...)); got != info.Name {
		t.Fatalf("FileNameBuf = %q, want %q", got, info.Name)
	}
}
