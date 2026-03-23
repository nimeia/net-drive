//go:build windows

package winclientusergui

import "syscall"

type uiPage int

const (
	pageHome uiPage = iota
	pageWorkspaces
	pageSettings
	pageHelp
)
const (
	cwUseDefault       = ^uintptr(0x7fffffff)
	wsOverlapped       = 0x00000000
	wsCaption          = 0x00C00000
	wsSysMenu          = 0x00080000
	wsThickFrame       = 0x00040000
	wsMinimizeBox      = 0x00020000
	wsMaximizeBox      = 0x00010000
	wsVisible          = 0x10000000
	wsChild            = 0x40000000
	wsTabStop          = 0x00010000
	wsVScroll          = 0x00200000
	wsHScroll          = 0x00100000
	wsBorder           = 0x00800000
	wsOverlappedWindow = wsOverlapped | wsCaption | wsSysMenu | wsThickFrame | wsMinimizeBox | wsMaximizeBox
	esLeft             = 0x0000
	esAutoVScroll      = 0x0040
	esMultiline        = 0x0004
	esReadOnly         = 0x0800
	esWantReturn       = 0x1000
	esAutoHScroll      = 0x0080
	bsPushButton       = 0x00000000
	cbsDropDownList    = 0x0003
	wsExClientEdge     = 0x00000200
	swHide             = 0
	swShow             = 5
	swShowDefault      = 10
	swRestore          = 9
	wmCreate           = 0x0001
	wmDestroy          = 0x0002
	wmClose            = 0x0010
	wmCommand          = 0x0111
	wmTimer            = 0x0113
	wmApp              = 0x8000
	wmSize             = 0x0005
	wmLButtonDblClk    = 0x0203
	wmRButtonUp        = 0x0205
	sizeMinimized      = 1
	bnClicked          = 0
	cbnSelChange       = 1
	cbAddString        = 0x0143
	cbGetLBTEXT        = 0x0148
	cbGetLBTEXTLen     = 0x0149
	cbResetContent     = 0x014B
	cbSetCurSel        = 0x014E
	cbGetCurSel        = 0x0147
	colorWindow        = 5
	idcArrow           = 32512
	idiApplication     = 32512
	timerRefreshID     = 1
	trayCallbackMsg    = wmApp + 1
)
const (
	nimAdd         = 0
	nimModify      = 1
	nimDelete      = 2
	nifMessage     = 0x0001
	nifIcon        = 0x0002
	nifTip         = 0x0004
	nifInfo        = 0x0010
	niifInfo       = 0x0001
	mfString       = 0
	mfSeparator    = 0x0800
	tpmLeftAlign   = 0
	tpmBottomAlign = 0x20
	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100
)
const (
	idPageSelect = 2001 + iota
	idHeaderStatus
	idHomeSummary
	idConnect
	idDisconnect
	idOpenExplorer
	idOpenSupportConsole
	idWorkspaceName
	idWorkspaceDisplayName
	idSavedWorkspaces
	idSaveWorkspace
	idLoadWorkspace
	idDeleteWorkspace
	idServerAddr
	idToken
	idMountPoint
	idRemotePath
	idVolumePrefix
	idWorkspaceAutoMount
	idWorkspaceSummary
	idDefaultWorkspace
	idAutoReconnect
	idLaunchOnLogin
	idSaveSettings
	idSettingsSummary
	idRunSelfCheck
	idExportSupport
	idOpenLogFolder
	idHelpSummary
)
const (
	idTrayOpen = 7001 + iota
	idTrayHome
	idTrayWorkspaces
	idTrayHelp
	idTrayConnect
	idTrayDisconnect
	idTrayExportSupport
	idTrayOpenSupportConsole
	idTrayExit
)

type point struct{ X, Y int32 }
type msg struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}
type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}
type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
}

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	shell32              = syscall.NewLazyDLL("shell32.dll")
	procAppendMenu       = user32.NewProc("AppendMenuW")
	procCreatePopupMenu  = user32.NewProc("CreatePopupMenu")
	procDestroyMenu      = user32.NewProc("DestroyMenu")
	procCreateWindowEx   = user32.NewProc("CreateWindowExW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procDefWindowProc    = user32.NewProc("DefWindowProcW")
	procDispatchMessage  = user32.NewProc("DispatchMessageW")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procGetMessage       = user32.NewProc("GetMessageW")
	procGetModuleHandle  = kernel32.NewProc("GetModuleHandleW")
	procGetWindowText    = user32.NewProc("GetWindowTextW")
	procGetWindowTextLen = user32.NewProc("GetWindowTextLengthW")
	procKillTimer        = user32.NewProc("KillTimer")
	procLoadCursor       = user32.NewProc("LoadCursorW")
	procLoadIcon         = user32.NewProc("LoadIconW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procRegisterClassEx  = user32.NewProc("RegisterClassExW")
	procSendMessage      = user32.NewProc("SendMessageW")
	procSetForegroundWnd = user32.NewProc("SetForegroundWindow")
	procSetTimer         = user32.NewProc("SetTimer")
	procSetWindowText    = user32.NewProc("SetWindowTextW")
	procShellNotifyIcon  = shell32.NewProc("Shell_NotifyIconW")
	procShowWindow       = user32.NewProc("ShowWindow")
	procTrackPopupMenu   = user32.NewProc("TrackPopupMenu")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
)
