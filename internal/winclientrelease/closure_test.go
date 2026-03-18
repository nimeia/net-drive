package winclientrelease

import (
	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
	"strings"
	"testing"
	"time"
)

func TestReleaseClosureTracksOutstandingValidation(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	manifest := NewManifest("3.0.0", []Artifact{{Name: "client", Kind: "exe", Path: "dist/client.exe"}}, table, matrix, winclientsmoke.DefaultExplorerSmoke())
	record := NewHostValidationRecord("3.0.0", winclientsmoke.DefaultExplorerSmoke(), table, matrix)
	closure := NewReleaseClosure(manifest, record)
	if closure.ReleaseReady || len(closure.Reasons) == 0 || !strings.Contains(strings.ToLower(closure.Markdown()), "not ready") {
		t.Fatalf("unexpected closure: %+v", closure)
	}
}
func TestReleaseClosureBecomesReadyAfterBackfill(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	smoke := winclientsmoke.DefaultExplorerSmoke()
	manifest := NewManifest("3.0.0", []Artifact{{Name: "client", Kind: "exe", Path: "dist/client.exe"}}, table, matrix, smoke)
	record := NewHostValidationRecord("3.0.0", smoke, table, matrix)
	for _, s := range smoke {
		record.ApplyScenario(s.ID, ValidationPass, "ok")
	}
	for _, item := range record.InstallerChecklist {
		record.ApplyChecklist("installer", item.Item, ValidationPass, "ok")
	}
	for _, item := range record.RecoveryChecklist {
		record.ApplyChecklist("recovery", item.Item, ValidationPass, "ok")
	}
	for _, item := range []struct{ c, a string }{{"msi", "install"}, {"msi", "upgrade"}, {"msi", "uninstall"}, {"exe", "portable-launch"}} {
		record.ApplyInstallerRun(item.c, item.a, ValidationPass, "ok")
	}
	record.MarkCompleted("tester", time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC))
	closure := NewReleaseClosure(manifest, record)
	if !closure.ReleaseReady || closure.Manifest.FinalStatus != "ready" {
		t.Fatalf("unexpected ready closure: %+v", closure)
	}
}
