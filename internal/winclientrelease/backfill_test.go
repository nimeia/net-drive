package winclientrelease

import (
	"testing"

	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
)

func TestHostValidationRecordBackfillSummary(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	record := NewHostValidationRecord("2.0.0", winclientsmoke.DefaultExplorerSmoke(), table, matrix)
	if !record.ApplyScenario("explorer-delete-denied", ValidationPass, "expected readonly denial") {
		t.Fatal("ApplyScenario(delete-denied) = false")
	}
	if !record.ApplyInstallerRun("msi", "install", ValidationPass, "install ok") {
		t.Fatal("ApplyInstallerRun(msi/install) = false")
	}
	if !record.ApplyInstallerRun("exe", "portable-launch", ValidationWarn, "signed but SmartScreen prompt shown") {
		t.Fatal("ApplyInstallerRun(exe/portable-launch) = false")
	}
	if !record.ApplyChecklist("recovery", "Clean stop clears recovery marker state", ValidationPass, "marker removed") {
		t.Fatal("ApplyChecklist(recovery) = false")
	}
	if record.Summary.Warn == 0 || record.Summary.Pass == 0 {
		t.Fatalf("summary not updated after backfill: %+v", record.Summary)
	}
}
