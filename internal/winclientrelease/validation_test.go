package winclientrelease

import (
	"strings"
	"testing"
	"time"

	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
)

func TestHostValidationRecordTemplates(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	record := NewHostValidationRecord("1.2.3", winclientsmoke.DefaultExplorerSmoke(), table, matrix)
	if len(record.ExplorerScenarios) == 0 || len(record.InstallerChecklist) == 0 || len(record.RecoveryChecklist) == 0 || len(record.InstallerRuns) == 0 {
		t.Fatalf("unexpected record sizes: %+v", record)
	}
	if !record.ApplyScenario("explorer-mount-visible", ValidationPass, "ok") {
		t.Fatal("ApplyScenario = false, want true")
	}
	if !record.ApplyChecklist("installer", "MSI install succeeded", ValidationPass, "msi ok") {
		t.Fatal("ApplyChecklist installer = false, want true")
	}
	if !record.ApplyInstallerRun("msi", "upgrade", ValidationWarn, "manual restart required") {
		t.Fatal("ApplyInstallerRun = false, want true")
	}
	record.MarkCompleted("tester", time.Date(2026, 3, 18, 8, 0, 0, 0, time.UTC))
	md := strings.ToLower(record.Markdown())
	for _, want := range []string{"windows host validation record", "explorer scenarios", "installer checklist", "recovery checklist", "installer runs", "overall=", "completed by: tester", "1.2.3"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q", want)
		}
	}
	payload, err := record.JSON()
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	jsonText := string(payload)
	for _, want := range []string{"explorer_scenarios", "installer_runs", "summary", "completed_by"} {
		if !strings.Contains(jsonText, want) {
			t.Fatalf("json missing %q: %s", want, jsonText)
		}
	}
	if record.Summary.Pass == 0 || record.Summary.Warn == 0 {
		t.Fatalf("summary not updated: %+v", record.Summary)
	}
}
