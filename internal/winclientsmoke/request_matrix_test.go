package winclientsmoke

import (
	"developer-mount/internal/winfsp"
	"testing"
)

func TestDefaultExplorerRequestMatrixIncludesReadOnlyMutationScenarios(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	matrix := DefaultExplorerRequestMatrix(table)
	need := map[string]bool{"explorer-create-denied": false, "explorer-write-denied": false, "explorer-rename-denied": false, "explorer-delete-denied": false}
	for _, entry := range matrix.Entries {
		if _, ok := need[entry.ScenarioID]; ok {
			need[entry.ScenarioID] = true
			if entry.Status == RequestStatusGap {
				t.Fatalf("scenario %s unexpectedly gap", entry.ScenarioID)
			}
		}
	}
	for id, found := range need {
		if !found {
			t.Fatalf("scenario %s missing", id)
		}
	}
	if !matrix.Finalized || matrix.Blocked == 0 {
		t.Fatalf("unexpected matrix: %+v", matrix)
	}
}
