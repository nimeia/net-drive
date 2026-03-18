package winfsp

import (
	"strings"
	"testing"
)

func TestDefaultNativeCallbackTableDispatcherReady(t *testing.T) {
	table := DefaultNativeCallbackTable(BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	if !table.Active || !table.Finalized || table.Gaps != 0 || table.MissingHotPathCount() != 0 {
		t.Fatalf("unexpected table: %+v", table)
	}
	for _, want := range []string{"Create", "Write", "SetBasicInfo", "SetFileSize", "SetSecurity", "Rename", "Cleanup", "Flush", "GetSecurityByName", "GetSecurity", "CanDelete", "SetDeleteOnClose"} {
		if !strings.Contains(table.Markdown(), want) {
			t.Fatalf("markdown missing %q", want)
		}
	}
}
