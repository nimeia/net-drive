package winclientsmoke

import (
	"testing"

	"developer-mount/internal/winfsp"
)

func TestDefaultExplorerRequestMatrixIncludesDeleteDenied(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	matrix := DefaultExplorerRequestMatrix(table)
	found := false
	for _, entry := range matrix.Entries {
		if entry.ScenarioID == "explorer-delete-denied" {
			found = true
			if entry.Status != RequestStatusBlocked && entry.Callback != "Cleanup" {
				t.Fatalf("delete-denied entry = %+v, want blocked for read-only callbacks", entry)
			}
		}
	}
	if !found {
		t.Fatal("explorer-delete-denied scenario not found in request matrix")
	}
	if matrix.Blocked == 0 {
		t.Fatalf("matrix.Blocked = %d, want > 0", matrix.Blocked)
	}
}
