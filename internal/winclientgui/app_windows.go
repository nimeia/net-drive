//go:build windows

package winclientgui

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"developer-mount/internal/winclient"
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
	swShowDefault      = 10
	wmCreate           = 0x0001
	wmDestroy          = 0x0002
	wmCommand          = 0x0111
	bnClicked          = 0
	cbnSelChange       = 1
	cbAddString        = 0x0143
	cbSetCurSel        = 0x014E
	cbGetCurSel        = 0x0147
	colorWindow        = 5
	idcArrow           = 32512
)

const (
	idAddr = 1001 + iota
	idToken
	idClientInstance
	idLeaseSeconds
	idMountPoint
	idVolumePrefix
	idPath
	idOffset
	idLength
	idMaxEntries
	idOperation
	idRun
	idPreview
	idDefaults
	idClear
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

type app struct {
	hInstance  uintptr
	hwnd       uintptr
	controls   map[int]uintptr
	runner     winclient.Runner
	operations []winclient.Operation
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
	procLoadCursor       = user32.NewProc("LoadCursorW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procRegisterClassEx  = user32.NewProc("RegisterClassExW")
	procSendMessage      = user32.NewProc("SendMessageW")
	procSetWindowText    = user32.NewProc("SetWindowTextW")
	procShowWindow       = user32.NewProc("ShowWindow")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
	activeApp            *app
)

func Run() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hInstance, _, err := procGetModuleHandle.Call(0)
	if hInstance == 0 {
		return fmt.Errorf("GetModuleHandleW failed: %w", err)
	}

	a := &app{
		hInstance:  hInstance,
		controls:   map[int]uintptr{},
		runner:     winclient.NewRunner(),
		operations: winclient.Operations(),
	}
	activeApp = a

	className := syscall.StringToUTF16Ptr("DeveloperMountWin32ConfigWindow")
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

	title := syscall.StringToUTF16Ptr("Developer Mount Win32 Config Test")
	hwnd, _, createErr := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		uintptr(wsOverlappedWindow|wsVisible),
		cwUseDefault,
		cwUseDefault,
		980,
		760,
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
			activeApp.resetDefaults()
			activeApp.setOutput("Use this window to test volume / getattr / readdir / read against devmount-server.\r\nYou can also copy the equivalent devmount-winfsp.exe command line.\r\n")
		}
		return 0
	case wmCommand:
		if activeApp != nil {
			activeApp.handleCommand(wParam)
			return 0
		}
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func (a *app) initControls() {
	xLabel := 16
	xInput := 140
	y := 16
	rowH := 24
	gap := 8
	labelW := 116
	inputW := 280
	rightLabelX := 460
	rightInputX := 590
	rightInputW := 160

	a.addLabel("Server Addr", xLabel, y, labelW, rowH)
	a.addEdit(idAddr, xInput, y, inputW, rowH)
	a.addLabel("Lease Seconds", rightLabelX, y, 116, rowH)
	a.addEdit(idLeaseSeconds, rightInputX, y, rightInputW, rowH)
	y += rowH + gap

	a.addLabel("Token", xLabel, y, labelW, rowH)
	a.addEdit(idToken, xInput, y, 610, rowH)
	y += rowH + gap

	a.addLabel("Client Instance", xLabel, y, labelW, rowH)
	a.addEdit(idClientInstance, xInput, y, inputW, rowH)
	a.addLabel("Operation", rightLabelX, y, 116, rowH)
	a.addCombo(idOperation, rightInputX, y, rightInputW, 160)
	y += rowH + gap

	a.addLabel("Mount Point", xLabel, y, labelW, rowH)
	a.addEdit(idMountPoint, xInput, y, inputW, rowH)
	a.addLabel("Volume Prefix", rightLabelX, y, 116, rowH)
	a.addEdit(idVolumePrefix, rightInputX, y, rightInputW, rowH)
	y += rowH + gap

	a.addLabel("Path", xLabel, y, labelW, rowH)
	a.addEdit(idPath, xInput, y, 610, rowH)
	y += rowH + gap

	a.addLabel("Offset", xLabel, y, labelW, rowH)
	a.addEdit(idOffset, xInput, y, inputW, rowH)
	a.addLabel("Read Length", rightLabelX, y, 116, rowH)
	a.addEdit(idLength, rightInputX, y, rightInputW, rowH)
	y += rowH + gap

	a.addLabel("Max Entries", xLabel, y, labelW, rowH)
	a.addEdit(idMaxEntries, xInput, y, inputW, rowH)
	y += rowH + 16

	a.addButton(idRun, "Run Test", xLabel, y, 120, 28)
	a.addButton(idPreview, "Show CLI", xLabel+132, y, 120, 28)
	a.addButton(idDefaults, "Defaults", xLabel+264, y, 120, 28)
	a.addButton(idClear, "Clear Output", xLabel+396, y, 120, 28)
	y += 44

	a.addOutput(idOutput, xLabel, y, 930, 560)

	combo := a.controls[idOperation]
	for _, op := range a.operations {
		procSendMessage.Call(combo, cbAddString, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(string(op)))))
	}
	procSendMessage.Call(combo, cbSetCurSel, 2, 0)
}

func (a *app) handleCommand(wParam uintptr) {
	id := int(uint16(wParam & 0xffff))
	code := int(uint16((wParam >> 16) & 0xffff))
	switch id {
	case idRun:
		if code == bnClicked {
			a.runSelectedOperation()
		}
	case idPreview:
		if code == bnClicked {
			a.showCLIPreview()
		}
	case idDefaults:
		if code == bnClicked {
			a.resetDefaults()
		}
	case idClear:
		if code == bnClicked {
			a.setOutput("")
		}
	case idOperation:
		if code == cbnSelChange {
			a.showCLIPreview()
		}
	}
}

func (a *app) runSelectedOperation() {
	cfg, op, err := a.readConfig()
	if err != nil {
		a.setOutput("configuration error: " + err.Error())
		return
	}
	button := a.controls[idRun]
	procEnableWindow.Call(button, 0)
	defer procEnableWindow.Call(button, 1)

	output, execErr := a.runner.Execute(context.Background(), cfg, op)
	if execErr != nil {
		a.setOutput("run failed: " + execErr.Error())
		return
	}
	a.setOutput(output)
}

func (a *app) showCLIPreview() {
	cfg, op, err := a.readConfig()
	if err != nil {
		a.setOutput("configuration error: " + err.Error())
		return
	}
	a.setOutput(winclient.BuildCLIPreview(cfg, op))
}

func (a *app) resetDefaults() {
	cfg := winclient.DefaultConfig()
	a.setText(idAddr, cfg.Addr)
	a.setText(idToken, cfg.Token)
	a.setText(idClientInstance, cfg.ClientInstanceID)
	a.setText(idLeaseSeconds, strconv.FormatUint(uint64(cfg.LeaseSeconds), 10))
	a.setText(idMountPoint, cfg.MountPoint)
	a.setText(idVolumePrefix, cfg.VolumePrefix)
	a.setText(idPath, cfg.Path)
	a.setText(idOffset, strconv.FormatInt(cfg.Offset, 10))
	a.setText(idLength, strconv.FormatUint(uint64(cfg.Length), 10))
	a.setText(idMaxEntries, strconv.FormatUint(uint64(cfg.MaxEntries), 10))
	procSendMessage.Call(a.controls[idOperation], cbSetCurSel, 2, 0)
}

func (a *app) readConfig() (winclient.Config, winclient.Operation, error) {
	cfg := winclient.Config{
		Addr:             a.text(idAddr),
		Token:            a.text(idToken),
		ClientInstanceID: a.text(idClientInstance),
		MountPoint:       a.text(idMountPoint),
		VolumePrefix:     a.text(idVolumePrefix),
		Path:             a.text(idPath),
	}
	lease, err := parseUint32Field("lease seconds", a.text(idLeaseSeconds))
	if err != nil {
		return winclient.Config{}, "", err
	}
	offset, err := parseInt64Field("offset", a.text(idOffset))
	if err != nil {
		return winclient.Config{}, "", err
	}
	length, err := parseUint32Field("read length", a.text(idLength))
	if err != nil {
		return winclient.Config{}, "", err
	}
	maxEntries, err := parseUint32Field("max entries", a.text(idMaxEntries))
	if err != nil {
		return winclient.Config{}, "", err
	}
	cfg.LeaseSeconds = lease
	cfg.Offset = offset
	cfg.Length = length
	cfg.MaxEntries = maxEntries

	index, _, _ := procSendMessage.Call(a.controls[idOperation], cbGetCurSel, 0, 0)
	if int(index) < 0 || int(index) >= len(a.operations) {
		return winclient.Config{}, "", fmt.Errorf("select an operation")
	}
	op := a.operations[int(index)]
	return cfg.Normalized(), op, nil
}

func parseUint32Field(name, value string) (uint32, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	n, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be an unsigned integer", name)
	}
	return uint32(n), nil
}

func parseInt64Field(name, value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return n, nil
}

func (a *app) addLabel(text string, x, y, w, h int) {
	a.createControl(0, "STATIC", text, wsChild|wsVisible, 0, x, y, w, h)
}

func (a *app) addEdit(id, x, y, w, h int) {
	a.controls[id] = a.createControl(id, "EDIT", "", wsChild|wsVisible|wsTabStop|wsBorder|esLeft|esAutoHScroll, wsExClientEdge, x, y, w, h)
}

func (a *app) addCombo(id, x, y, w, h int) {
	a.controls[id] = a.createControl(id, "COMBOBOX", "", wsChild|wsVisible|wsTabStop|cbsDropDownList|wsVScroll, wsExClientEdge, x, y, w, h)
}

func (a *app) addButton(id int, text string, x, y, w, h int) {
	a.controls[id] = a.createControl(id, "BUTTON", text, wsChild|wsVisible|wsTabStop|bsPushButton, 0, x, y, w, h)
}

func (a *app) addOutput(id, x, y, w, h int) {
	a.controls[id] = a.createControl(id, "EDIT", "", wsChild|wsVisible|wsTabStop|wsVScroll|wsHScroll|esLeft|esMultiline|esAutoVScroll|esAutoHScroll|esReadOnly|esWantReturn, wsExClientEdge, x, y, w, h)
}

func (a *app) createControl(id int, className, text string, style, exStyle uint32, x, y, w, h int) uintptr {
	hwnd, _, _ := procCreateWindowEx.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(className))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))),
		uintptr(style),
		uintptr(x),
		uintptr(y),
		uintptr(w),
		uintptr(h),
		a.hwnd,
		uintptr(id),
		a.hInstance,
		0,
	)
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

func (a *app) setOutput(text string) {
	a.setText(idOutput, strings.ReplaceAll(text, "\n", "\r\n"))
}
