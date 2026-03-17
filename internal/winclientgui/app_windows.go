//go:build windows

package winclientgui

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"developer-mount/internal/winclient"
	"developer-mount/internal/winclientruntime"
	"developer-mount/internal/winclientstore"
)

type app struct {
	hInstance    uintptr
	hwnd         uintptr
	controls     map[int]uintptr
	pageControls map[uiPage][]uintptr
	runner       winclient.Runner
	operations   []winclient.Operation
	store        winclientstore.Store
	state        winclientstore.State
	runtime      *winclientruntime.Runtime
	currentPage  uiPage
}

var activeApp *app

func Run() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hInstance, _, err := procGetModuleHandle.Call(0)
	if hInstance == 0 {
		return fmt.Errorf("GetModuleHandleW failed: %w", err)
	}
	store, err := winclientstore.OpenDefault()
	if err != nil {
		return err
	}

	a := &app{
		hInstance:    hInstance,
		controls:     map[int]uintptr{},
		pageControls: map[uiPage][]uintptr{},
		runner:       winclient.NewRunner(),
		operations:   winclient.Operations(),
		store:        store,
		runtime:      winclientruntime.New(nil),
	}
	activeApp = a

	className := syscall.StringToUTF16Ptr("DeveloperMountWin32ProductWindow")
	cursor, _, _ := procLoadCursor.Call(0, uintptr(idcArrow))
	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   syscall.NewCallback(windowProc),
		HInstance:     hInstance,
		HCursor:       cursor,
		HbrBackground: uintptr(colorWindow + 1),
		LpszClassName: className,
	}
	atom, _, regErr := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW failed: %w", regErr)
	}

	title := syscall.StringToUTF16Ptr("Developer Mount Windows Client")
	hwnd, _, createErr := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		uintptr(wsOverlappedWindow|wsVisible),
		cwUseDefault,
		cwUseDefault,
		1100,
		920,
		0,
		0,
		hInstance,
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW failed: %w", createErr)
	}
	a.hwnd = hwnd
	procShowWindow.Call(hwnd, swShowDefault)
	procUpdateWindow.Call(hwnd)

	var m msg
	for {
		ret, _, msgErr := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		switch int32(ret) {
		case -1:
			return fmt.Errorf("GetMessageW failed: %w", msgErr)
		case 0:
			return nil
		default:
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
		}
	}
}

func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmCreate:
		if activeApp != nil {
			activeApp.hwnd = hwnd
			activeApp.initControls()
			if err := activeApp.loadProfiles(); err != nil {
				activeApp.resetDefaults()
				activeApp.setText(idProfileName, "default")
				activeApp.setOutput("profile store load failed: " + err.Error())
			}
			activeApp.showPage(pageDashboard)
			activeApp.refreshRuntimeViews()
			procSetTimer.Call(hwnd, timerRefreshID, 1000, 0)
		}
		return 0
	case wmCommand:
		if activeApp != nil {
			activeApp.handleCommand(wParam)
			return 0
		}
	case wmTimer:
		if activeApp != nil && wParam == timerRefreshID {
			activeApp.refreshRuntimeViews()
			return 0
		}
	case wmDestroy:
		procKillTimer.Call(hwnd, timerRefreshID)
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}
