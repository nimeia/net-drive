package winclientrelease

import (
	"strings"
	"testing"

	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
)

func TestHostValidationRecordTemplates(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	record := NewHostValidationRecord("1.2.3", winclientsmoke.DefaultExplorerSmoke(), table, matrix)
	if len(record.ExplorerScenarios) == 0 || len(record.InstallerChecklist) == 0 || len(record.RecoveryChecklist) == 0 {
		t.Fatalf("unexpected record sizes: %+v", record)
	}
	md := strings.ToLower(record.Markdown())
	for _, want := range []string{"windows host validation record", "explorer scenarios", "installer checklist", "recovery checklist", "1.2.3"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q", want)
		}
	}
	payload, err := record.JSON()
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	if !strings.Contains(string(payload), "explorer_scenarios") {
		t.Fatalf("json missing explorer_scenarios: %s", string(payload))
	}
}
