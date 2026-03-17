//go:build windows

package winclientgui

import (
	"fmt"

	"developer-mount/internal/winclientdiag"
)

func (a *app) runSelfCheck() {
	report, err := a.currentDiagnosticsReport()
	if err != nil {
		a.setOutput("self-check failed: " + err.Error())
		_ = a.logError("self-check failed: " + err.Error())
		return
	}
	a.setOutput(report.Text())
	_ = a.logInfo("self-check completed")
	a.showTrayNotification("Self-check completed", "Diagnostics summary refreshed in the Diagnostics page output panel.")
}

func (a *app) exportDiagnostics() {
	report, err := a.currentDiagnosticsReport()
	if err != nil {
		a.setOutput("export diagnostics failed: " + err.Error())
		_ = a.logError("export diagnostics failed: " + err.Error())
		return
	}
	path, err := winclientdiag.Export("", report)
	if err != nil {
		a.setOutput("export diagnostics failed: " + err.Error())
		_ = a.logError("export diagnostics failed: " + err.Error())
		return
	}
	message := fmt.Sprintf("diagnostics exported to %s", path)
	a.setOutput(message)
	_ = a.logInfo(message)
	a.showTrayNotification("Diagnostics exported", path)
}
