//go:build windows

package winclientusergui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"developer-mount/internal/winclient"
	"developer-mount/internal/winclientdiag"
	"developer-mount/internal/winclientlog"
	"developer-mount/internal/winclientproduct"
	"developer-mount/internal/winclientrecovery"
	"developer-mount/internal/winclientruntime"
	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winclientstore"
)

type app struct {
	hInstance, hwnd uintptr
	controls        map[int]uintptr
	store           winclientstore.Store
	state           winclientstore.State
	logger          winclientlog.Logger
	recovery        winclientrecovery.Store
	runtime         *winclientruntime.Runtime
	supportExePath  string
	lastReport      winclientdiag.Report
}

var activeApp *app

func Run() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	h, _, err := procGetModuleHandle.Call(0)
	if h == 0 {
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
	recovery, err := winclientrecovery.OpenDefault()
	if err != nil {
		return err
	}
	exePath, _ := os.Executable()
	a := &app{hInstance: h, controls: map[int]uintptr{}, store: store, logger: logger, recovery: recovery, runtime: winclientruntime.New(nil), supportExePath: winclientproduct.SupportConsolePath(exePath)}
	activeApp = a
	className := syscall.StringToUTF16Ptr("DeveloperMountUserClientWindow")
	cursor, _, _ := procLoadCursor.Call(0, uintptr(idcArrow))
	icon, _, _ := procLoadIcon.Call(0, uintptr(idiApplication))
	wc := wndClassEx{CbSize: uint32(unsafe.Sizeof(wndClassEx{})), LpfnWndProc: syscall.NewCallback(windowProc), HInstance: h, HIcon: icon, HCursor: cursor, HbrBackground: uintptr(colorWindow + 1), LpszClassName: className, HIconSm: icon}
	atom, _, regErr := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW failed: %w", regErr)
	}
	title := syscall.StringToUTF16Ptr("Developer Mount")
	hwnd, _, createErr := procCreateWindowEx.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), uintptr(wsOverlappedWindow|wsVisible), cwUseDefault, cwUseDefault, 1120, 940, 0, 0, h, 0)
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
			activeApp.loadState()
			activeApp.refreshViews()
			procSetTimer.Call(hwnd, timerRefreshID, 1000, 0)
		}
		return 0
	case wmCommand:
		if activeApp != nil {
			activeApp.handleCommand(wParam)
			return 0
		}
	case wmTimer:
		if activeApp != nil {
			activeApp.refreshViews()
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

func (a *app) initControls() {
	a.controls[idHeaderStatus] = a.create("STATIC", "当前未连接任何工作区", wsChild|wsVisible, 0, 16, 16, 1060, 24, idHeaderStatus)
	a.controls[idConnect] = a.create("BUTTON", "连接", wsChild|wsVisible|wsTabStop, 0, 16, 52, 100, 30, idConnect)
	a.controls[idDisconnect] = a.create("BUTTON", "断开", wsChild|wsVisible|wsTabStop, 0, 128, 52, 100, 30, idDisconnect)
	a.controls[idOpenExplorer] = a.create("BUTTON", "打开挂载位置", wsChild|wsVisible|wsTabStop, 0, 240, 52, 140, 30, idOpenExplorer)
	a.controls[idOpenSupportConsole] = a.create("BUTTON", "打开 Support Console", wsChild|wsVisible|wsTabStop, 0, 392, 52, 180, 30, idOpenSupportConsole)
	a.controls[idWorkspaceName] = a.create("EDIT", "default", wsChild|wsVisible|wsBorder, wsExClientEdge, 16, 98, 180, 24, idWorkspaceName)
	a.controls[idWorkspaceDisplayName] = a.create("EDIT", "默认工作区", wsChild|wsVisible|wsBorder, wsExClientEdge, 208, 98, 180, 24, idWorkspaceDisplayName)
	a.controls[idServerAddr] = a.create("EDIT", "127.0.0.1:17890", wsChild|wsVisible|wsBorder, wsExClientEdge, 400, 98, 200, 24, idServerAddr)
	a.controls[idToken] = a.create("EDIT", "devmount-dev-token", wsChild|wsVisible|wsBorder, wsExClientEdge, 612, 98, 220, 24, idToken)
	a.controls[idMountPoint] = a.create("EDIT", "M:", wsChild|wsVisible|wsBorder, wsExClientEdge, 844, 98, 90, 24, idMountPoint)
	a.controls[idRemotePath] = a.create("EDIT", "/", wsChild|wsVisible|wsBorder, wsExClientEdge, 946, 98, 70, 24, idRemotePath)
	a.controls[idHomeSummary] = a.create("EDIT", "", wsChild|wsVisible|wsBorder|wsVScroll|wsHScroll|esLeft|esMultiline|esAutoVScroll|esAutoHScroll|esReadOnly|esWantReturn, wsExClientEdge, 16, 140, 1060, 740, idHomeSummary)
}
func (a *app) create(class, text string, style, ex uint32, x, y, w, h, id int) uintptr {
	hwnd, _, _ := procCreateWindowEx.Call(uintptr(ex), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(class))), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))), uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), a.hwnd, uintptr(id), a.hInstance, 0)
	return hwnd
}
func (a *app) setText(id int, value string) {
	procSetWindowText.Call(a.controls[id], uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(value))))
}
func (a *app) text(id int) string {
	hwnd := a.controls[id]
	length, _, _ := procGetWindowTextLen.Call(hwnd)
	buf := make([]uint16, length+1)
	procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}
func (a *app) loadState() {
	state, _ := a.store.Load()
	a.state = state
	if cfg, ok := a.state.Profiles[a.state.ActiveProfile]; ok {
		a.applyWorkspace(a.state.ActiveProfile, cfg)
	}
}
func (a *app) applyWorkspace(name string, cfg winclient.Config) {
	cfg = cfg.Normalized()
	meta := a.state.WorkspaceMeta[name]
	a.setText(idWorkspaceName, name)
	a.setText(idWorkspaceDisplayName, winclientproduct.WorkspaceDisplayName(name, meta))
	a.setText(idServerAddr, cfg.Addr)
	a.setText(idToken, cfg.Token)
	a.setText(idMountPoint, cfg.MountPoint)
	a.setText(idRemotePath, cfg.Path)
}
func (a *app) currentConfig() (string, winclient.Config, error) {
	defaults := winclient.DefaultConfig()
	name := strings.TrimSpace(a.text(idWorkspaceName))
	if name == "" {
		name = "default"
	}
	cfg := winclient.Config{Addr: a.text(idServerAddr), Token: a.text(idToken), ClientInstanceID: defaults.ClientInstanceID, LeaseSeconds: defaults.LeaseSeconds, MountPoint: a.text(idMountPoint), VolumePrefix: defaults.VolumePrefix, Path: a.text(idRemotePath), LocalPath: defaults.LocalPath, HostBackend: winclient.HostBackendAuto, Length: defaults.Length, MaxEntries: defaults.MaxEntries}.Normalized()
	return name, cfg, cfg.Validate(winclient.OperationMount)
}
func (a *app) handleCommand(wParam uintptr) {
	id := int(uint16(wParam & 0xffff))
	if int(uint16((wParam>>16)&0xffff)) != bnClicked {
		return
	}
	switch id {
	case idConnect:
		a.start()
	case idDisconnect:
		a.stop()
	case idOpenExplorer:
		a.openExplorer()
	case idOpenSupportConsole:
		a.openSupport()
	}
}
func (a *app) start() {
	name, cfg, err := a.currentConfig()
	if err != nil {
		a.setOutput("配置错误：" + err.Error())
		return
	}
	if prepared, _, err := winclient.PrepareDirectoryMountPoint(cfg.MountPoint, cfg.VolumePrefix); err == nil {
		cfg.MountPoint = prepared
		a.setText(idMountPoint, prepared)
	}
	if err := a.runtime.Start(cfg, name); err != nil {
		a.setOutput("连接失败：" + winclientproduct.FriendlyRuntimeError(err.Error()))
		return
	}
	meta := winclientstore.WorkspaceMeta{DisplayName: strings.TrimSpace(a.text(idWorkspaceDisplayName)), LastUsedAt: time.Now().UTC().Format(time.RFC3339)}
	state, _ := a.store.SaveWorkspace(name, cfg, meta)
	a.state = state
	a.refreshViews()
}
func (a *app) stop() { _ = a.runtime.Stop(); a.refreshViews() }
func (a *app) openExplorer() {
	mp := a.runtime.Snapshot().MountPoint
	if mp == "" {
		_, cfg, _ := a.currentConfig()
		mp = cfg.MountPoint
	}
	if len(mp) == 2 && mp[1] == ':' {
		mp += `\`
	}
	_ = exec.Command("explorer.exe", mp).Start()
}
func (a *app) openSupport() {
	if a.supportExePath != "" {
		_ = exec.Command(a.supportExePath).Start()
	}
}
func (a *app) setOutput(text string) {
	a.setText(idHomeSummary, strings.ReplaceAll(text, "\n", "\r\n"))
}
func (a *app) refreshViews() {
	snap := a.runtime.Snapshot()
	a.setText(idHeaderStatus, winclientproduct.FriendlyStatus(snap))
	a.setOutput(winclientproduct.HomeSummary(snap, a.state) + "\n\n" + winclientproduct.WorkspacesSummary(a.state) + "\n\n" + winclientproduct.SettingsSummary(a.state) + "\n\n" + winclientproduct.HelpSummary(a.lastReport, a.logger.Path(), a.supportExePath))
}
func (a *app) diagnostics() (winclientdiag.Report, error) {
	_, cfg, err := a.currentConfig()
	if err != nil {
		return winclientdiag.Report{}, err
	}
	tail, _ := a.logger.Tail(16 * 1024)
	checker := winclientdiag.NewChecker()
	state, _ := a.recovery.Load()
	return checker.Run(context.Background(), cfg, a.runtime.Snapshot(), a.store.Path(), a.logger.Path(), a.recovery.Path(), state, tail, winclientsmoke.DefaultExplorerSmoke()), nil
}
