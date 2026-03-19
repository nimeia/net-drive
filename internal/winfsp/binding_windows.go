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

const winfspDiskDeviceName = `WinFsp.Disk`

func probeBinding(config HostConfig) (BindingInfo, error) {
	requested := normalizeRequestedBackend(config.Backend)
	info := BindingInfo{RequestedBackend: requested, Backend: "winfsp-native-preflight", EffectiveBackend: "winfsp-native-preflight", Available: false, MountPoint: config.MountPoint}

	dllPath, err := findWinFspDLL()
	if err != nil {
		info.PreflightError = err.Error()
		info.Note = "Install WinFsp Developer components so the user-mode DLL is available."
		return info, err
	}
	info.DLLPath = dllPath
	info.Available = true
	info.LauncherPath = findWinFspLauncher()

	if dispatcher, dispatcherErr := probeDispatcherBindings(dllPath); dispatcherErr != nil {
		info.DispatcherStatus = dispatcherErr.Error()
		info.CallbackBridgeStatus = "callback bridge unavailable until dispatcher APIs are ready"
		info.ServiceLoopStatus = "service loop unavailable until dispatcher APIs are ready"
	} else {
		_ = dispatcher
		info.DispatcherReady = true
		info.DispatcherStatus = "dispatcher APIs ready"
		info.CallbackBridgeReady = true
		info.CallbackBridgeStatus = "callback bridge scaffold ready"
		info.ServiceLoopReady = true
		info.ServiceLoopStatus = "dispatcher service loop scaffold ready"
	}
	if strings.TrimSpace(config.MountPoint) == "" {
		info.PreflightError = "mount point is required"
		return info, fmt.Errorf(info.PreflightError)
	}
	if shouldSkipNativePreflight(config.MountPoint) {
		info.PreflightOK = true
		info.Note = "WinFsp DLL loaded. Native preflight is skipped for drive-letter mount points; syntax and occupancy are validated locally before start."
	} else {
		if err := preflightMount(dllPath, config.MountPoint); err != nil {
			info.PreflightError = err.Error()
			return info, err
		}
		info.PreflightOK = true
		info.Note = "WinFsp DLL loaded and FspFileSystemPreflight succeeded for the requested mount point."
	}

	// When user asks for "auto", prefer dispatcher-v1 when the dispatcher APIs are available.
	if requested == "auto" && info.DispatcherReady {
		requested = "dispatcher-v1"
	}
	
	switch requested {
	case "dispatcher-v1":
		if !info.DispatcherReady {
			return info, fmt.Errorf("dispatcher-v1 requested but dispatcher APIs are unavailable: %s", info.DispatcherStatus)
		}
		info.Backend = "winfsp-dispatcher-v1"
		info.EffectiveBackend = "winfsp-dispatcher-v1"
	case "preflight", "auto":
		info.Backend = "winfsp-native-preflight"
		info.EffectiveBackend = "winfsp-native-preflight"
	default:
		return info, fmt.Errorf("unsupported host backend %q", requested)
	}
	info = populateBindingDerived(info, nil, nil, nil)
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
	pathDirs := pathEntries()
	candidates := make([]string, 0, len(base)*6+len(pathDirs)*len(names)+len(names))
	for _, name := range names {
		candidates = append(candidates, name)
	}
	for _, dir := range pathDirs {
		for _, name := range names {
			candidates = append(candidates, filepath.Join(dir, name))
		}
	}
	for _, root := range base {
		for _, name := range names {
			candidates = append(candidates, filepath.Join(root, name), filepath.Join(root, "bin", name))
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
	pathDirs := pathEntries()
	candidates := make([]string, 0, len(base)*4+len(pathDirs)*len(names))
	for _, dir := range pathDirs {
		for _, name := range names {
			candidates = append(candidates, filepath.Join(dir, name))
		}
	}
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

func pathEntries() []string {
	return dedupeNonEmpty(filepath.SplitList(os.Getenv("PATH")))
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
		return fmt.Errorf("FspFileSystemPreflight(%s) failed with ntstatus=0x%08x%s", mountPoint, uint32(status), ntStatusHint(NTStatus(status), mountPoint))
	}
	return nil
}

func ntStatusHint(status NTStatus, mountPoint string) string {
	switch uint32(status) {
	case 0xc0000033:
		hint := fmt.Sprintf(" (WinFsp rejected mount point %q with STATUS_OBJECT_NAME_INVALID)", mountPoint)
		// Common cause: attempting to mount to a drive letter without a trailing "\\".
		// WinFsp expects a well-formed volume root (e.g. "Z:\\").
		if len(mountPoint) == 2 && ((mountPoint[0] >= 'a' && mountPoint[0] <= 'z') || (mountPoint[0] >= 'A' && mountPoint[0] <= 'Z')) && mountPoint[1] == ':' {
			hint += " (try using a path like \"Z:\\\" or a directory mount point)"
		}
		return hint
	default:
		return ""
	}
}

func shouldSkipNativePreflight(mountPoint string) bool {
	mountPoint = strings.TrimSpace(mountPoint)
	return len(mountPoint) == 2 && ((mountPoint[0] >= 'a' && mountPoint[0] <= 'z') || (mountPoint[0] >= 'A' && mountPoint[0] <= 'Z')) && mountPoint[1] == ':'
}
