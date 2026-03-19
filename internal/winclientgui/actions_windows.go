//go:build windows

package winclientgui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"developer-mount/internal/winclient"
	"developer-mount/internal/winclientruntime"
)

func (a *app) handleCommand(wParam uintptr) {
	id := int(uint16(wParam & 0xffff))
	code := int(uint16((wParam >> 16) & 0xffff))
	switch id {
	case idPageSelect:
		if code == cbnSelChange {
			index, _, _ := procSendMessage.Call(a.controls[idPageSelect], cbGetCurSel, 0, 0)
			a.showPage(uiPage(index))
		}
	case idStartMount:
		if code == bnClicked {
			a.startMount()
		}
	case idStopMount:
		if code == bnClicked {
			a.stopMount()
		}
	case idMountPreview:
		if code == bnClicked {
			cfg, err := a.readConfigFields()
			if err != nil {
				a.setOutput("configuration error: " + err.Error())
				return
			}
			a.showPage(pageDiagnostics)
			a.setOutput(winclient.BuildCLIPreview(cfg, winclient.OperationMount))
		}
	case idChooseMountDir:
		if code == bnClicked {
			a.chooseMountDirectory()
		}
	case idRun:
		if code == bnClicked {
			a.runSelectedOperation()
		}
	case idPreview:
		if code == bnClicked {
			a.showCLIPreview()
		}
	case idRefreshDiagnostics:
		if code == bnClicked {
			a.refreshRuntimeViews()
		}
	case idRunSelfCheck:
		if code == bnClicked {
			a.runSelfCheck()
		}
	case idExportDiagnostics:
		if code == bnClicked {
			a.exportDiagnostics()
		}
	case idDefaults:
		if code == bnClicked {
			a.resetDefaults()
			a.refreshRuntimeViews()
		}
	case idClear:
		if code == bnClicked {
			a.setOutput("")
		}
	case idSaveProfile:
		if code == bnClicked {
			a.saveProfile()
		}
	case idLoadProfile:
		if code == bnClicked {
			a.loadSelectedProfile()
		}
	case idDeleteProfile:
		if code == bnClicked {
			a.deleteSelectedProfile()
		}
	case idSavedProfiles:
		if code == cbnSelChange {
			name := a.selectedComboText(idSavedProfiles)
			if name != "" {
				a.setText(idProfileName, name)
			}
		}
	case idOperation, idHostBackend:
		if code == cbnSelChange {
			a.refreshRuntimeViews()
		}
	}
}

func (a *app) startMount() {
	cfg, err := a.readConfigFields()
	if err != nil {
		a.setOutput("configuration error: " + err.Error())
		_ = a.logError("mount start config error: " + err.Error())
		return
	}
	if resolved, changed := winclient.ResolveMountPointForStart(cfg.MountPoint); changed {
		original := cfg.MountPoint
		cfg.MountPoint = resolved
		a.setText(idMountPoint, resolved)
		message := fmt.Sprintf("mount point %s is in use; switched to %s before start", original, resolved)
		a.setOutput(message)
		_ = a.logInfo(message)
	}
	if prepared, createdParent, err := winclient.PrepareDirectoryMountPoint(cfg.MountPoint, cfg.VolumePrefix); err != nil {
		a.setOutput("mount start failed: " + err.Error())
		_ = a.logError("mount directory prepare failed: " + err.Error())
		return
	} else {
		cfg.MountPoint = prepared
		a.setText(idMountPoint, prepared)
		if createdParent {
			message := fmt.Sprintf("created mount parent directory %s before start", filepath.Dir(prepared))
			a.setOutput(message)
			_ = a.logInfo(message)
		}
	}
	profile := strings.TrimSpace(a.text(idProfileName))
	if err := a.runtime.Start(cfg, profile); err != nil {
		a.setOutput("mount start failed: " + err.Error())
		_ = a.logError("mount start failed: " + err.Error())
		a.refreshRuntimeViews()
		return
	}
	if profile != "" {
		a.state.ActiveProfile = profile
		_ = a.store.Save(a.state)
	}
	a.recoveryState, _ = a.recovery.MarkStart(profile, cfg, a.runtime.Snapshot())
	a.showPage(pageDashboard)
	a.refreshRuntimeViews()
	message := fmt.Sprintf("mount start requested for %s using backend %s", cfg.MountPoint, cfg.HostBackend)
	a.setOutput(message)
	_ = a.logInfo(message)
}

func (a *app) chooseMountDirectory() {
	path, ok, err := chooseFolder(a.hwnd, "Select a parent directory for the mount point")
	if err != nil {
		a.setOutput("choose mount directory failed: " + err.Error())
		_ = a.logError("choose mount directory failed: " + err.Error())
		return
	}
	if !ok {
		return
	}
	suggested := winclient.SuggestDirectoryMountPoint(path, a.text(idVolumePrefix))
	a.setText(idMountPoint, suggested)
	message := fmt.Sprintf("selected mount parent %s; mount point set to %s", path, suggested)
	a.setOutput(message)
	_ = a.logInfo(message)
}

func (a *app) stopMount() {
	if err := a.runtime.Stop(); err != nil {
		a.setOutput("mount stop failed: " + err.Error())
		_ = a.logError("mount stop failed: " + err.Error())
		return
	}
	a.recoveryState, _ = a.recovery.Update(a.runtime.Snapshot())
	a.refreshRuntimeViews()
	a.setOutput("mount stop requested")
	_ = a.logInfo("mount stop requested")
}

func (a *app) runSelectedOperation() {
	cfg, op, err := a.readConfig()
	if err != nil {
		a.setOutput("configuration error: " + err.Error())
		_ = a.logError("diagnostics config error: " + err.Error())
		return
	}
	button := a.controls[idRun]
	procEnableWindow.Call(button, 0)
	defer procEnableWindow.Call(button, 1)
	output, execErr := a.runner.Execute(context.Background(), cfg, op)
	if execErr != nil {
		a.setOutput("run failed: " + execErr.Error())
		_ = a.logError(fmt.Sprintf("diagnostics %s failed: %v", op, execErr))
		return
	}
	if op == winclient.OperationMaterialize {
		output += "\nYou can now inspect the local directory with Explorer or an editor.\n"
	}
	a.setOutput(output)
	_ = a.logInfo(fmt.Sprintf("diagnostics %s completed", op))
	a.refreshRuntimeViews()
}

func (a *app) showCLIPreview() {
	cfg, op, err := a.readConfig()
	if err != nil {
		a.setOutput("configuration error: " + err.Error())
		return
	}
	a.setOutput(winclient.BuildCLIPreview(cfg, op))
}
func (a *app) refreshRuntimeViews() {
	snapshot := a.runtime.Snapshot()
	a.recoveryState, _ = a.recovery.Update(snapshot)
	a.setHeaderStatus(snapshot)
	a.maybeShowRuntimeError(snapshot)
	a.setText(idDashboardSummary, windowsText(a.dashboardSummary(snapshot)))
	a.setText(idDiagnosticsSummary, windowsText(a.diagnosticsSummary(snapshot)))
	a.syncTray(snapshot)
}
func (a *app) setHeaderStatus(snapshot winclientruntime.Snapshot) {
	status := snapshot.StatusText
	if snapshot.LastError != "" {
		status += " | error: " + summarizeRuntimeError(snapshot.LastError)
	}
	a.setText(idHeaderStatus, status)
}

func (a *app) maybeShowRuntimeError(snapshot winclientruntime.Snapshot) {
	if strings.TrimSpace(snapshot.LastError) == "" {
		a.lastShownError = ""
		return
	}
	if snapshot.LastError == a.lastShownError {
		return
	}
	a.lastShownError = snapshot.LastError
	a.setOutput("mount runtime error: " + snapshot.LastError)
}

func (a *app) dashboardSummary(snapshot winclientruntime.Snapshot) string {
	profile := strings.TrimSpace(a.text(idProfileName))
	if profile == "" {
		profile = "(unsaved draft)"
	}
	return fmt.Sprintf(`Windows client shell

Page: Dashboard
Current profile: %s
State: %s
Status: %s
Recovery: %s
Server: %s
Mount point: %s
Volume prefix: %s
Remote path: %s
Client instance: %s
Session ID: %d
Principal: %s
Server version: %s %s
Lease expires: %s
Store path: %s
Log path: %s
Recovery path: %s
Host backend requested: %s
Host backend effective: %s
Host binding: %s
Dispatcher state: %s
Host DLL: %s
Launcher: %s

Use Start Mount / Stop Mount to exercise the WinFsp host lifecycle.
Tray: close or minimize the window to keep the client running in the notification area.
Use Profiles to edit connection and mount settings.
Use Diagnostics to run volume/getattr/readdir/read/materialize, self-checks, diagnostics export, and Windows Explorer smoke against the current profile.`, profile, snapshot.Phase, snapshot.StatusText, a.recoveryState.Summary(), emptyOrDraft(snapshot.ServerAddr, a.text(idAddr)), emptyOrDraft(snapshot.MountPoint, a.text(idMountPoint)), emptyOrDraft(snapshot.VolumePrefix, a.text(idVolumePrefix)), emptyOrDraft(snapshot.RemotePath, a.text(idPath)), emptyOrDraft(snapshot.ClientInstanceID, a.text(idClientInstance)), snapshot.SessionID, emptyOrDraft(snapshot.PrincipalID, "-"), emptyOrDraft(snapshot.ServerName, "-"), emptyOrDraft(snapshot.ServerVersion, "-"), emptyOrDraft(snapshot.ExpiresAt, "-"), a.store.Path(), a.logger.Path(), a.recovery.Path(), emptyOrDraft(snapshot.RequestedBackend, a.selectedHostBackend()), emptyOrDraft(snapshot.HostBackend, "-"), emptyOrDraft(snapshot.HostBindingStatus, "-"), emptyOrDraft(snapshot.HostDispatcherState, "-"), emptyOrDraft(snapshot.HostDLLPath, "-"), emptyOrDraft(snapshot.HostLauncherPath, "-"))
}
func (a *app) diagnosticsSummary(snapshot winclientruntime.Snapshot) string {
	cfg, cfgErr := a.readConfigFields()
	op, opErr := a.selectedOperation()
	lines := []string{"Diagnostics snapshot", "", fmt.Sprintf("Runtime phase: %s", snapshot.Phase), fmt.Sprintf("Runtime status: %s", snapshot.StatusText), fmt.Sprintf("Runtime error: %s", emptyOrDraft(snapshot.LastError, "-")), fmt.Sprintf("Recovery state: %s", a.recoveryState.Summary()), fmt.Sprintf("Requested backend: %s", emptyOrDraft(snapshot.RequestedBackend, a.selectedHostBackend())), fmt.Sprintf("Effective backend: %s", emptyOrDraft(snapshot.HostBackend, "-")), fmt.Sprintf("Host binding: %s", emptyOrDraft(snapshot.HostBindingStatus, "-")), fmt.Sprintf("Dispatcher state: %s", emptyOrDraft(snapshot.HostDispatcherState, "-")), fmt.Sprintf("Host DLL: %s", emptyOrDraft(snapshot.HostDLLPath, "-")), fmt.Sprintf("Launcher: %s", emptyOrDraft(snapshot.HostLauncherPath, "-")), fmt.Sprintf("Store path: %s", a.store.Path()), fmt.Sprintf("Log path: %s", a.logger.Path()), fmt.Sprintf("Recovery path: %s", a.recovery.Path())}
	if cfgErr == nil {
		lines = append(lines, fmt.Sprintf("Current profile: %s", emptyOrDraft(strings.TrimSpace(a.text(idProfileName)), "(unsaved draft)")), fmt.Sprintf("Mount CLI: %s", winclient.BuildCLIPreview(cfg, winclient.OperationMount)))
	} else {
		lines = append(lines, "Current config error: "+cfgErr.Error())
	}
	if cfgErr == nil && opErr == nil {
		lines = append(lines, fmt.Sprintf("Selected op CLI: %s", winclient.BuildCLIPreview(cfg, op)))
	} else if opErr != nil {
		lines = append(lines, "Selected operation error: "+opErr.Error())
	}
	lines = append(lines, "", "Use Run Self-Check to validate the current profile, native callback table, and Explorer request matrix.", "Use Export Diagnostics to generate a zip with report.txt/report.json/log-tail.txt/explorer-smoke.md/explorer-request-matrix.md/winfsp-native-callbacks.md/recovery.json.")
	return strings.Join(lines, "\n")
}
func emptyOrDraft(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func summarizeRuntimeError(err string) string {
	err = strings.TrimSpace(err)
	if err == "" {
		return ""
	}
	if strings.Contains(err, "invalid mount point syntax") {
		return "invalid mount point; use X: or an existing absolute directory"
	}
	if strings.Contains(err, "already exists; WinFsp expects a new mount leaf") {
		return "directory mount leaf already exists; choose a new child path"
	}
	return err
}

func windowsText(text string) string { return strings.ReplaceAll(text, "\n", "\r\n") }
