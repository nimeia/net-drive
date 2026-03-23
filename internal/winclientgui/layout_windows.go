//go:build windows

package winclientgui

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"developer-mount/internal/winclient"
)

func (a *app) initControls() {
	pageLabelX := 16
	pageInputX := 84
	headerY := 16
	rowH := 24
	a.addLabelCommon("Page", pageLabelX, headerY, 56, rowH)
	a.addComboCommon(idPageSelect, pageInputX, headerY, 160, 120)
	a.addStaticCommon(idHeaderStatus, "Runtime idle", 264, headerY, 824, rowH)
	for _, page := range []uiPage{pageDashboard, pageProfiles, pageDiagnostics} {
		procSendMessage.Call(a.controls[idPageSelect], cbAddString, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(page.title()))))
	}
	procSendMessage.Call(a.controls[idPageSelect], cbSetCurSel, 0, 0)
	a.initDashboardControls(headerY + rowH + 20)
	a.initProfileControls(headerY + rowH + 20)
	a.initDiagnosticsControls(headerY + rowH + 20)
}
func (a *app) initDashboardControls(y int) {
	a.addLabel(pageDashboard, "Dashboard", 16, y, 220, 28)
	a.addLabel(pageDashboard, "Mount runtime status, support diagnostics, and quick actions", 16, y+28, 420, 20)
	a.addButton(pageDashboard, idStartMount, "Start Mount", 16, y+60, 120, 30)
	a.addButton(pageDashboard, idStopMount, "Stop Mount", 148, y+60, 120, 30)
	a.addButton(pageDashboard, idMountPreview, "Show Mount CLI", 280, y+60, 140, 30)
	a.addButton(pageDashboard, idExportDiagnostics, "Export Diagnostics", 432, y+60, 160, 30)
	a.addOutput(pageDashboard, idDashboardSummary, 16, y+108, 1070, 720)
}
func (a *app) initProfileControls(y int) {
	xLabel := 16
	xInput := 160
	rightLabelX := 560
	rightInputX := 700
	rowH := 24
	gap := 8
	inputW := 320
	rightInputW := 220
	a.addLabel(pageProfiles, "Profiles", xLabel, y, 220, 28)
	a.addLabel(pageProfiles, "Named support/test profiles and connection defaults", xLabel, y+28, 360, 20)
	y += 56
	a.addLabel(pageProfiles, "Profile Name", xLabel, y, 120, rowH)
	a.addEdit(pageProfiles, idProfileName, xInput, y, 220, rowH)
	a.addLabel(pageProfiles, "Saved Profiles", rightLabelX, y, 120, rowH)
	a.addCombo(pageProfiles, idSavedProfiles, rightInputX, y, 180, 160)
	a.addButton(pageProfiles, idSaveProfile, "Save", 844, y-2, 72, 28)
	a.addButton(pageProfiles, idLoadProfile, "Load", 924, y-2, 72, 28)
	a.addButton(pageProfiles, idDeleteProfile, "Delete", 1004, y-2, 72, 28)
	y += rowH + gap
	a.addLabel(pageProfiles, "Server Addr", xLabel, y, 120, rowH)
	a.addEdit(pageProfiles, idAddr, xInput, y, inputW, rowH)
	a.addLabel(pageProfiles, "Lease Seconds", rightLabelX, y, 120, rowH)
	a.addEdit(pageProfiles, idLeaseSeconds, rightInputX, y, rightInputW, rowH)
	y += rowH + gap
	a.addLabel(pageProfiles, "Token", xLabel, y, 120, rowH)
	a.addEdit(pageProfiles, idToken, xInput, y, 880, rowH)
	y += rowH + gap
	a.addLabel(pageProfiles, "Client Instance", xLabel, y, 120, rowH)
	a.addEdit(pageProfiles, idClientInstance, xInput, y, inputW, rowH)
	a.addLabel(pageProfiles, "Mount Point", rightLabelX, y, 120, rowH)
	a.addEdit(pageProfiles, idMountPoint, rightInputX, y, rightInputW-104, rowH)
	a.addButton(pageProfiles, idChooseMountDir, "Choose Parent", rightInputX+rightInputW-112, y-2, 112, 28)
	y += rowH + gap
	a.addLabel(pageProfiles, "Volume Prefix", xLabel, y, 120, rowH)
	a.addEdit(pageProfiles, idVolumePrefix, xInput, y, inputW, rowH)
	a.addLabel(pageProfiles, "Remote Path", rightLabelX, y, 120, rowH)
	a.addEdit(pageProfiles, idPath, rightInputX, y, rightInputW+120, rowH)
	y += rowH + gap
	a.addLabel(pageProfiles, "Local Path", xLabel, y, 120, rowH)
	a.addEdit(pageProfiles, idLocalPath, xInput, y, 320, rowH)
	a.addLabel(pageProfiles, "Host Backend", rightLabelX, y, 120, rowH)
	a.addCombo(pageProfiles, idHostBackend, rightInputX, y, 220, 160)
	for _, backend := range winclient.HostBackendOptions() {
		procSendMessage.Call(a.controls[idHostBackend], cbAddString, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(backend))))
	}
	procSendMessage.Call(a.controls[idHostBackend], cbSetCurSel, 0, 0)
	y += rowH + gap
	a.addLabel(pageProfiles, "Offset", xLabel, y, 120, rowH)
	a.addEdit(pageProfiles, idOffset, xInput, y, inputW, rowH)
	a.addLabel(pageProfiles, "Read Length", rightLabelX, y, 120, rowH)
	a.addEdit(pageProfiles, idLength, rightInputX, y, rightInputW, rowH)
	y += rowH + gap
	a.addLabel(pageProfiles, "Max Entries", xLabel, y, 120, rowH)
	a.addEdit(pageProfiles, idMaxEntries, xInput, y, inputW, rowH)
	a.addButton(pageProfiles, idDefaults, "Restore Defaults", rightLabelX, y-2, 160, 28)
}
func (a *app) initDiagnosticsControls(y int) {
	a.addLabel(pageDiagnostics, "Diagnostics", 16, y, 220, 28)
	a.addLabel(pageDiagnostics, "Advanced smoke actions, self-checks, CLI preview, and diagnostics export over the current in-memory profile", 16, y+28, 620, 20)
	y += 56
	a.addLabel(pageDiagnostics, "Operation", 16, y, 100, 24)
	a.addCombo(pageDiagnostics, idOperation, 100, y, 220, 160)
	for _, op := range a.operations {
		procSendMessage.Call(a.controls[idOperation], cbAddString, 0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(string(op)))))
	}
	procSendMessage.Call(a.controls[idOperation], cbSetCurSel, uintptr(len(a.operations)-1), 0)
	a.addButton(pageDiagnostics, idRun, "Run", 340, y-2, 80, 28)
	a.addButton(pageDiagnostics, idPreview, "Show CLI", 428, y-2, 100, 28)
	a.addButton(pageDiagnostics, idRefreshDiagnostics, "Refresh", 536, y-2, 100, 28)
	a.addButton(pageDiagnostics, idRunSelfCheck, "Run Self-Check", 644, y-2, 140, 28)
	a.addButton(pageDiagnostics, idExportDiagnostics, "Export Diagnostics", 792, y-2, 150, 28)
	a.addButton(pageDiagnostics, idClear, "Clear Output", 950, y-2, 110, 28)
	y += 40
	a.addOutput(pageDiagnostics, idDiagnosticsSummary, 16, y, 1070, 200)
	y += 216
	a.addOutput(pageDiagnostics, idOutput, 16, y, 1070, 500)
}
func (a *app) showPage(page uiPage) {
	for current, handles := range a.pageControls {
		show := swHide
		if current == page {
			show = swShow
		}
		for _, hwnd := range handles {
			procShowWindow.Call(hwnd, uintptr(show))
		}
	}
	a.currentPage = page
	procSendMessage.Call(a.controls[idPageSelect], cbSetCurSel, uintptr(page), 0)
	a.refreshRuntimeViews()
}
func (a *app) registerControl(page uiPage, id int, hwnd uintptr) uintptr {
	if id != 0 {
		a.controls[id] = hwnd
	}
	a.pageControls[page] = append(a.pageControls[page], hwnd)
	return hwnd
}
func (a *app) addLabelCommon(text string, x, y, w, h int) {
	a.createControl(0, "STATIC", text, wsChild|wsVisible, 0, x, y, w, h)
}
func (a *app) addStaticCommon(id int, text string, x, y, w, h int) {
	hwnd := a.createControl(id, "STATIC", text, wsChild|wsVisible, 0, x, y, w, h)
	a.controls[id] = hwnd
}
func (a *app) addComboCommon(id, x, y, w, h int) {
	hwnd := a.createControl(id, "COMBOBOX", "", wsChild|wsVisible|wsTabStop|cbsDropDownList|wsVScroll, wsExClientEdge, x, y, w, h)
	a.controls[id] = hwnd
}
func (a *app) addLabel(page uiPage, text string, x, y, w, h int) {
	a.registerControl(page, 0, a.createControl(0, "STATIC", text, wsChild|wsVisible, 0, x, y, w, h))
}
func (a *app) addEdit(page uiPage, id, x, y, w, h int) {
	a.registerControl(page, id, a.createControl(id, "EDIT", "", wsChild|wsVisible|wsTabStop|wsBorder|esLeft|esAutoHScroll, wsExClientEdge, x, y, w, h))
}
func (a *app) addCombo(page uiPage, id, x, y, w, h int) {
	a.registerControl(page, id, a.createControl(id, "COMBOBOX", "", wsChild|wsVisible|wsTabStop|cbsDropDownList|wsVScroll, wsExClientEdge, x, y, w, h))
}
func (a *app) addButton(page uiPage, id int, text string, x, y, w, h int) {
	a.registerControl(page, id, a.createControl(id, "BUTTON", text, wsChild|wsVisible|wsTabStop|bsPushButton, 0, x, y, w, h))
}
func (a *app) addOutput(page uiPage, id, x, y, w, h int) {
	a.registerControl(page, id, a.createControl(id, "EDIT", "", wsChild|wsVisible|wsTabStop|wsVScroll|wsHScroll|esLeft|esMultiline|esAutoVScroll|esAutoHScroll|esReadOnly|esWantReturn, wsExClientEdge, x, y, w, h))
}
func (a *app) createControl(id int, className, text string, style, exStyle uint32, x, y, w, h int) uintptr {
	hwnd, _, _ := procCreateWindowEx.Call(uintptr(exStyle), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(className))), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))), uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), a.hwnd, uintptr(id), a.hInstance, 0)
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
func (a *app) selectedComboText(id int) string {
	combo := a.controls[id]
	index, _, _ := procSendMessage.Call(combo, cbGetCurSel, 0, 0)
	if int(index) < 0 {
		return ""
	}
	length, _, _ := procSendMessage.Call(combo, cbGetLBTEXTLen, index, 0)
	buf := make([]uint16, int(length)+1)
	procSendMessage.Call(combo, cbGetLBTEXT, index, uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}
func (a *app) setComboSelection(id int, value string) {
	combo := a.controls[id]
	for idx := 0; idx < 32; idx++ {
		length, _, _ := procSendMessage.Call(combo, cbGetLBTEXTLen, uintptr(idx), 0)
		if int(length) < 0 {
			break
		}
		buf := make([]uint16, int(length)+1)
		procSendMessage.Call(combo, cbGetLBTEXT, uintptr(idx), uintptr(unsafe.Pointer(&buf[0])))
		if strings.EqualFold(syscall.UTF16ToString(buf), value) {
			procSendMessage.Call(combo, cbSetCurSel, uintptr(idx), 0)
			return
		}
	}
	procSendMessage.Call(combo, cbSetCurSel, 0, 0)
}
func (a *app) setOutput(text string) { a.setText(idOutput, strings.ReplaceAll(text, "\n", "\r\n")) }
func (page uiPage) title() string {
	switch page {
	case pageDashboard:
		return "Dashboard"
	case pageProfiles:
		return "Profiles"
	case pageDiagnostics:
		return "Diagnostics"
	default:
		return fmt.Sprintf("Page-%d", page)
	}
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
