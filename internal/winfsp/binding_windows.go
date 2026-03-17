//go:build windows

package winfsp

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

const winfspDiskDeviceName = `\\Device\\WinFsp.Disk`

func probeBinding(config HostConfig) (BindingInfo, error) {
	info := BindingInfo{
		Backend:    "winfsp-native-preflight",
		Available:  false,
		MountPoint: config.MountPoint,
	}

	dllPath, err := findWinFspDLL()
	if err != nil {
		info.PreflightError = err.Error()
		info.Note = "Install WinFsp Developer components so the user-mode DLL is available."
		return info, err
	}
	info.DLLPath = dllPath
	info.Available = true
	info.LauncherPath = findWinFspLauncher()

	if strings.TrimSpace(config.MountPoint) == "" {
		info.PreflightError = "mount point is required"
		return info, fmt.Errorf(info.PreflightError)
	}
	if err := preflightMount(dllPath, config.MountPoint); err != nil {
		info.PreflightError = err.Error()
		return info, err
	}
	info.PreflightOK = true
	info.Note = "WinFsp DLL loaded and FspFileSystemPreflight succeeded for the requested mount point."
	return info, nil
}

func findWinFspDLL() (string, error) {
	for _, candidate := range candidateDLLPaths() {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("winfsp DLL not found in PATH or standard install locations")
}

func findWinFspLauncher() string {
	for _, candidate := range candidateLauncherPaths() {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func candidateDLLPaths() []string {
	base := winfspInstallBases()
	names := []string{"winfsp-x64.dll", "winfsp-x86.dll", "winfsp-a64.dll"}
	candidates := make([]string, 0, len(base)*6+len(names))
	for _, name := range names {
		candidates = append(candidates, name)
	}
	for _, root := range base {
		for _, name := range names {
			candidates = append(candidates,
				filepath.Join(root, name),
				filepath.Join(root, "bin", name),
			)
		}
		entries, _ := os.ReadDir(filepath.Join(root, "SxS"))
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			for _, name := range names {
				candidates = append(candidates, filepath.Join(root, "SxS", entry.Name(), "bin", name))
			}
		}
	}
	return dedupeNonEmpty(candidates)
}

func candidateLauncherPaths() []string {
	base := winfspInstallBases()
	names := []string{"launchctl-x64.exe", "launchctl-x86.exe", "launchctl-a64.exe", "launchctl.exe"}
	candidates := make([]string, 0, len(base)*4)
	for _, root := range base {
		for _, name := range names {
			candidates = append(candidates, filepath.Join(root, "bin", name))
		}
		entries, _ := os.ReadDir(filepath.Join(root, "SxS"))
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			for _, name := range names {
				candidates = append(candidates, filepath.Join(root, "SxS", entry.Name(), "bin", name))
			}
		}
	}
	return dedupeNonEmpty(candidates)
}

func winfspInstallBases() []string {
	roots := []string{}
	for _, envKey := range []string{"ProgramFiles(x86)", "ProgramW6432", "ProgramFiles"} {
		if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
			roots = append(roots, filepath.Join(value, "WinFsp"))
		}
	}
	return dedupeNonEmpty(roots)
}

func dedupeNonEmpty(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := value
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func preflightMount(dllPath, mountPoint string) error {
	dll, err := syscall.LoadDLL(dllPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", dllPath, err)
	}
	defer dll.Release()

	proc, err := dll.FindProc("FspFileSystemPreflight")
	if err != nil {
		return fmt.Errorf("find FspFileSystemPreflight: %w", err)
	}
	devicePath, err := syscall.UTF16PtrFromString(winfspDiskDeviceName)
	if err != nil {
		return err
	}
	mountPtr, err := syscall.UTF16PtrFromString(mountPoint)
	if err != nil {
		return err
	}
	status, _, _ := proc.Call(uintptr(unsafe.Pointer(devicePath)), uintptr(unsafe.Pointer(mountPtr)))
	if NTStatus(status) != StatusSuccess {
		return fmt.Errorf("FspFileSystemPreflight(%s) failed with ntstatus=0x%08x", mountPoint, uint32(status))
	}
	return nil
}
