//go:build windows

package winclientgui

import (
	"context"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"developer-mount/internal/winclient"
	"developer-mount/internal/winclientdiag"
	"developer-mount/internal/winclientlog"
	"developer-mount/internal/winclientruntime"
	"developer-mount/internal/winclientstore"
)

type app struct {
	hInstance       uintptr
	hwnd            uintptr
	controls        map[int]uintptr
	pageControls    map[uiPage][]uintptr
	runner          winclient.Runner
	operations      []winclient.Operation
	store           winclientstore.Store
	state           winclientstore.State
	logger          winclientlog.Logger
	runtime         *winclientruntime.Runtime
	currentPage     uiPage
	trayInitialized bool
	exitRequested   bool
	sentHideTip     bool
	lastTrayPhase   winclientruntime.Phase
	lastTrayError   string
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
	logger, err := winclientlog.OpenDefault()
	if err != nil {
		return err
	}
	a := &app{hInstance: hInstance, controls: map[int]uintptr{}, pageControls: map[uiPage][]uintptr{}, runner: winclient.NewRunner(), operations: winclient.Operations(), store: store, logger: logger, runtime: winclientruntime.New(nil)}
	activeApp = a
	_ = a.logInfo("win32 client startup")
	className := syscall.StringToUTF16Ptr("DeveloperMountWin32ProductWindow")
	cursor, _, _ := procLoadCursor.Call(0, uintptr(idcArrow))
	icon, _, _ := procLoadIcon.Call(0, uintptr(idiApplication))
	wc := wndClassEx{CbSize: uint32(unsafe.Sizeof(wndClassEx{})), LpfnWndProc: syscall.NewCallback(windowProc), HInstance: hInstance, HIcon: icon, HCursor: cursor, HbrBackground: uintptr(colorWindow + 1), LpszClassName: className, HIconSm: icon}
	atom, _, regErr := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW failed: %w", regErr)
	}
	title := syscall.StringToUTF16Ptr("Developer Mount Windows Client")
	hwnd, _, createErr := procCreateWindowEx.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), uintptr(wsOverlappedWindow|wsVisible), cwUseDefault, cwUseDefault, 1120, 940, 0, 0, hInstance, 0)
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
			activeApp.initTray()
			if err := activeApp.loadProfiles(); err != nil {
				activeApp.resetDefaults()
				activeApp.setText(idProfileName, "default")
				activeApp.setOutput("profile store load failed: " + err.Error())
				_ = activeApp.logError("profile store load failed: " + err.Error())
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
	case wmSize:
		if activeApp != nil && wParam == sizeMinimized {
			activeApp.hideToTray()
			return 0
		}
	case wmClose:
		if activeApp != nil && !activeApp.exitRequested {
			activeApp.hideToTray()
			return 0
		}
	case trayCallbackMsg:
		if activeApp != nil {
			activeApp.handleTrayMessage(lParam)
			return 0
		}
	case wmDestroy:
		if activeApp != nil {
			activeApp.removeTray()
			_ = activeApp.logInfo("win32 client shutdown")
		}
		procKillTimer.Call(hwnd, timerRefreshID)
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func (a *app) logInfo(message string) error  { return a.logger.Info(message) }
func (a *app) logError(message string) error { return a.logger.Error(message) }
func (a *app) currentDiagnosticsReport() (winclientdiag.Report, error) {
	cfg, err := a.readConfigFields()
	if err != nil {
		return winclientdiag.Report{}, err
	}
	tail, _ := a.logger.Tail(16 * 1024)
	checker := winclientdiag.NewChecker()
	return checker.Run(context.Background(), cfg, a.runtime.Snapshot(), a.store.Path(), a.logger.Path(), tail), nil
}
