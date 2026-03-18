package winfsp

import (
	"strings"
	"testing"
)

func TestDefaultNativeCallbackTableDispatcherReady(t *testing.T) {
	table := DefaultNativeCallbackTable(BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	if !table.Active {
		t.Fatal("table.Active = false, want true")
	}
	if table.Ready == 0 {
		t.Fatalf("table.Ready = %d, want > 0", table.Ready)
	}
	if table.Gaps != 0 {
		t.Fatalf("table.Gaps = %d, want 0", table.Gaps)
	}
	if table.MissingHotPathCount() != 0 {
		t.Fatalf("MissingHotPathCount = %d, want 0", table.MissingHotPathCount())
	}
	for _, want := range []string{"GetVolumeInfo", "Cleanup", "Flush", "GetSecurityByName", "GetSecurity", "CanDelete", "SetDeleteOnClose"} {
		if !strings.Contains(table.Markdown(), want) {
			t.Fatalf("markdown missing %q", want)
		}
	}
}
