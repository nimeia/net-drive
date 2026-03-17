//go:build windows

package winclientgui

import (
	"context"
	"fmt"
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
	a.showPage(pageDashboard)
	a.refreshRuntimeViews()
	message := fmt.Sprintf("mount start requested for %s using backend %s", cfg.MountPoint, cfg.HostBackend)
	a.setOutput(message)
	_ = a.logInfo(message)
}

func (a *app) stopMount() {
	if err := a.runtime.Stop(); err != nil {
		a.setOutput("mount stop failed: " + err.Error())
		_ = a.logError("mount stop failed: " + err.Error())
		return
	}
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
	a.setHeaderStatus(snapshot)
	a.setText(idDashboardSummary, windowsText(a.dashboardSummary(snapshot)))
	a.setText(idDiagnosticsSummary, windowsText(a.diagnosticsSummary(snapshot)))
	a.syncTray(snapshot)
}
func (a *app) setHeaderStatus(snapshot winclientruntime.Snapshot) {
	status := snapshot.StatusText
	if snapshot.LastError != "" {
		status += " | error: " + snapshot.LastError
	}
	a.setText(idHeaderStatus, status)
}
func (a *app) dashboardSummary(snapshot winclientruntime.Snapshot) string {
	profile := strings.TrimSpace(a.text(idProfileName))
	if profile == "" {
		profile = "(unsaved draft)"
	}
	return fmt.Sprintf("Windows client shell\n\nPage: Dashboard\nCurrent profile: %s\nState: %s\nStatus: %s\nServer: %s\nMount point: %s\nVolume prefix: %s\nRemote path: %s\nClient instance: %s\nSession ID: %d\nPrincipal: %s\nServer version: %s %s\nLease expires: %s\nStore path: %s\nLog path: %s\nHost backend requested: %s\nHost backend effective: %s\nHost binding: %s\nDispatcher state: %s\nHost DLL: %s\nLauncher: %s\n\nUse Start Mount / Stop Mount to exercise the WinFsp host lifecycle.\nTray: close or minimize the window to keep the client running in the notification area.\nUse Profiles to edit connection and mount settings.\nUse Diagnostics to run volume/getattr/readdir/read/materialize, self-checks, and diagnostics export against the current profile.", profile, snapshot.Phase, snapshot.StatusText, emptyOrDraft(snapshot.ServerAddr, a.text(idAddr)), emptyOrDraft(snapshot.MountPoint, a.text(idMountPoint)), emptyOrDraft(snapshot.VolumePrefix, a.text(idVolumePrefix)), emptyOrDraft(snapshot.RemotePath, a.text(idPath)), emptyOrDraft(snapshot.ClientInstanceID, a.text(idClientInstance)), snapshot.SessionID, emptyOrDraft(snapshot.PrincipalID, "-"), emptyOrDraft(snapshot.ServerName, "-"), emptyOrDraft(snapshot.ServerVersion, "-"), emptyOrDraft(snapshot.ExpiresAt, "-"), a.store.Path(), a.logger.Path(), emptyOrDraft(snapshot.RequestedBackend, a.selectedHostBackend()), emptyOrDraft(snapshot.HostBackend, "-"), emptyOrDraft(snapshot.HostBindingStatus, "-"), emptyOrDraft(snapshot.HostDispatcherState, "-"), emptyOrDraft(snapshot.HostDLLPath, "-"), emptyOrDraft(snapshot.HostLauncherPath, "-"))
}
func (a *app) diagnosticsSummary(snapshot winclientruntime.Snapshot) string {
	cfg, cfgErr := a.readConfigFields()
	op, opErr := a.selectedOperation()
	lines := []string{"Diagnostics snapshot", "", fmt.Sprintf("Runtime phase: %s", snapshot.Phase), fmt.Sprintf("Runtime status: %s", snapshot.StatusText), fmt.Sprintf("Runtime error: %s", emptyOrDraft(snapshot.LastError, "-")), fmt.Sprintf("Requested backend: %s", emptyOrDraft(snapshot.RequestedBackend, a.selectedHostBackend())), fmt.Sprintf("Effective backend: %s", emptyOrDraft(snapshot.HostBackend, "-")), fmt.Sprintf("Host binding: %s", emptyOrDraft(snapshot.HostBindingStatus, "-")), fmt.Sprintf("Dispatcher state: %s", emptyOrDraft(snapshot.HostDispatcherState, "-")), fmt.Sprintf("Host DLL: %s", emptyOrDraft(snapshot.HostDLLPath, "-")), fmt.Sprintf("Launcher: %s", emptyOrDraft(snapshot.HostLauncherPath, "-")), fmt.Sprintf("Store path: %s", a.store.Path()), fmt.Sprintf("Log path: %s", a.logger.Path())}
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
	lines = append(lines, "", "Use Run Self-Check to validate the current profile and host binding.", "Use Export Diagnostics to generate a zip with report.txt/report.json/log-tail.txt.")
	return strings.Join(lines, "\n")
}
func emptyOrDraft(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
func windowsText(text string) string { return strings.ReplaceAll(text, "\n", "\r\n") }
