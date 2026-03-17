//go:build windows

package winclientgui

import "syscall"

type uiPage int

const (
	pageDashboard uiPage = iota
	pageProfiles
	pageDiagnostics
)

const (
	cwUseDefault = ^uintptr(0x7fffffff)

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
	wmCreate           = 0x0001
	wmDestroy          = 0x0002
	wmCommand          = 0x0111
	wmTimer            = 0x0113
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
	timerRefreshID     = 1
)

const (
	idPageSelect = 1001 + iota
	idHeaderStatus
	idDashboardSummary
	idStartMount
	idStopMount
	idMountPreview
	idProfileName
	idSavedProfiles
	idSaveProfile
	idLoadProfile
	idDeleteProfile
	idAddr
	idToken
	idClientInstance
	idLeaseSeconds
	idMountPoint
	idVolumePrefix
	idPath
	idLocalPath
	idOffset
	idLength
	idMaxEntries
	idOperation
	idRun
	idPreview
	idDefaults
	idRefreshDiagnostics
	idClear
	idDiagnosticsSummary
	idOutput
)

type point struct {
	X int32
	Y int32
}

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

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procCreateWindowEx   = user32.NewProc("CreateWindowExW")
	procDefWindowProc    = user32.NewProc("DefWindowProcW")
	procDispatchMessage  = user32.NewProc("DispatchMessageW")
	procEnableWindow     = user32.NewProc("EnableWindow")
	procGetMessage       = user32.NewProc("GetMessageW")
	procGetModuleHandle  = kernel32.NewProc("GetModuleHandleW")
	procGetWindowText    = user32.NewProc("GetWindowTextW")
	procGetWindowTextLen = user32.NewProc("GetWindowTextLengthW")
	procKillTimer        = user32.NewProc("KillTimer")
	procLoadCursor       = user32.NewProc("LoadCursorW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procRegisterClassEx  = user32.NewProc("RegisterClassExW")
	procSendMessage      = user32.NewProc("SendMessageW")
	procSetTimer         = user32.NewProc("SetTimer")
	procSetWindowText    = user32.NewProc("SetWindowTextW")
	procShowWindow       = user32.NewProc("ShowWindow")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
)
