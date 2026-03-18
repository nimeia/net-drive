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
	if table.MissingHotPathCount() == 0 {
		t.Fatal("expected at least one hot-path gap for Cleanup/GetSecurityByName coverage")
	}
	if !strings.Contains(table.Markdown(), "GetVolumeInfo") {
		t.Fatal("markdown missing callback entry")
	}
}
