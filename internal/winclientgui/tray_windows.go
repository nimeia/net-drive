//go:build windows

package winclientgui

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"developer-mount/internal/winclientruntime"
)

func (a *app) initTray() {
	if a.trayInitialized {
		return
	}
	icon, _, _ := procLoadIcon.Call(0, uintptr(idiApplication))
	data := notifyIconData{
		CbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:             a.hwnd,
		UID:              1,
		UFlags:           nifMessage | nifIcon | nifTip,
		UCallbackMessage: trayCallbackMsg,
		HIcon:            icon,
	}
	copyUTF16(data.SzTip[:], "Developer Mount Windows Client")
	procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&data)))
	a.trayInitialized = true
}

func (a *app) removeTray() {
	if !a.trayInitialized {
		return
	}
	data := notifyIconData{CbSize: uint32(unsafe.Sizeof(notifyIconData{})), HWnd: a.hwnd, UID: 1}
	procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
	a.trayInitialized = false
}

func (a *app) syncTray(snapshot winclientruntime.Snapshot) {
	if !a.trayInitialized {
		return
	}
	data := notifyIconData{
		CbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:             a.hwnd,
		UID:              1,
		UFlags:           nifTip,
		UCallbackMessage: trayCallbackMsg,
	}
	tooltip := fmt.Sprintf("Support Console — %s", snapshot.Phase)
	if snapshot.MountPoint != "" {
		tooltip = fmt.Sprintf("Support Console — %s @ %s", snapshot.Phase, snapshot.MountPoint)
	}
	copyUTF16(data.SzTip[:], tooltip)
	procShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&data)))

	if snapshot.Phase != a.lastTrayPhase || snapshot.LastError != a.lastTrayError {
		title, message := trayMessageForTransition(snapshot)
		if title != "" && message != "" {
			a.showTrayNotification(title, message)
		}
		a.lastTrayPhase = snapshot.Phase
		a.lastTrayError = snapshot.LastError
	}
}

func trayMessageForTransition(snapshot winclientruntime.Snapshot) (string, string) {
	switch snapshot.Phase {
	case winclientruntime.PhaseMounted:
		return "Mount running", fmt.Sprintf("%s is active at %s", defaultText(snapshot.ServerAddr, "remote server"), defaultText(snapshot.MountPoint, "mount point"))
	case winclientruntime.PhaseStopping:
		return "Stopping mount", fmt.Sprintf("Stopping %s", defaultText(snapshot.MountPoint, "current mount"))
	case winclientruntime.PhaseError:
		return "Mount error", defaultText(snapshot.LastError, snapshot.StatusText)
	case winclientruntime.PhaseIdle:
		if snapshot.MountPoint != "" {
			return "Runtime idle", fmt.Sprintf("Last mount %s is stopped", snapshot.MountPoint)
		}
	}
	return "", ""
}

func (a *app) showTrayNotification(title, message string) {
	if !a.trayInitialized {
		return
	}
	data := notifyIconData{
		CbSize:      uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:        a.hwnd,
		UID:         1,
		UFlags:      nifInfo,
		DwInfoFlags: niifInfo,
	}
	copyUTF16(data.SzInfoTitle[:], title)
	copyUTF16(data.SzInfo[:], message)
	procShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&data)))
}

func (a *app) handleTrayMessage(lParam uintptr) {
	switch uint32(lParam) {
	case wmLButtonDblClk:
		a.showMainWindow(pageDashboard)
	case wmRButtonUp:
		a.showTrayMenu()
	}
}

func (a *app) showTrayMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	appendMenu(menu, mfString, idTrayOpen, "Open Support Console")
	appendMenu(menu, mfString, idTrayDashboard, "Show Dashboard")
	appendMenu(menu, mfString, idTrayProfiles, "Show Profiles")
	appendMenu(menu, mfString, idTrayDiagnostics, "Show Diagnostics")
	appendMenu(menu, mfSeparator, 0, "")
	appendMenu(menu, mfString, idTrayStartMount, "Start Mount")
	appendMenu(menu, mfString, idTrayStopMount, "Stop Mount")
	appendMenu(menu, mfString, idTrayExportDiagnostics, "Export Diagnostics")
	appendMenu(menu, mfSeparator, 0, "")
	appendMenu(menu, mfString, idTrayExit, "Exit")

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWnd.Call(a.hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmLeftAlign|tpmBottomAlign|tpmRightButton|tpmReturnCmd, uintptr(pt.X), uintptr(pt.Y), 0, a.hwnd, 0)
	if cmd != 0 {
		a.handleTrayCommand(int(cmd))
	}
}

func appendMenu(menu uintptr, flags uint32, id int, text string) {
	var textPtr uintptr
	if text != "" {
		textPtr = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text)))
	}
	procAppendMenu.Call(menu, uintptr(flags), uintptr(id), textPtr)
}

func (a *app) handleTrayCommand(id int) {
	switch id {
	case idTrayOpen, idTrayDashboard:
		a.showMainWindow(pageDashboard)
	case idTrayProfiles:
		a.showMainWindow(pageProfiles)
	case idTrayDiagnostics:
		a.showMainWindow(pageDiagnostics)
	case idTrayStartMount:
		a.startMount()
	case idTrayStopMount:
		a.stopMount()
	case idTrayExportDiagnostics:
		a.exportDiagnostics()
	case idTrayExit:
		a.exitRequested = true
		procShowWindow.Call(a.hwnd, swShow)
		procDestroyWindow.Call(a.hwnd)
	}
}

func (a *app) hideToTray() {
	procShowWindow.Call(a.hwnd, swHide)
	if !a.sentHideTip {
		a.showTrayNotification("Support Console still running", "Support Console is minimized to the notification area.")
		a.sentHideTip = true
	}
}

func (a *app) showMainWindow(page uiPage) {
	a.showPage(page)
	procShowWindow.Call(a.hwnd, swRestore)
	procSetForegroundWnd.Call(a.hwnd)
}

func copyUTF16(dst []uint16, value string) {
	value = strings.TrimSpace(value)
	if len(dst) == 0 {
		return
	}
	encoded, _ := syscall.UTF16FromString(value)
	if len(encoded) > len(dst) {
		encoded = encoded[:len(dst)]
		encoded[len(dst)-1] = 0
	}
	copy(dst, encoded)
}

func defaultText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
