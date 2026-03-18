package winclientsmoke

import (
	"strings"
	"testing"

	"developer-mount/internal/winfsp"
)

func TestDefaultExplorerRequestMatrix(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	matrix := DefaultExplorerRequestMatrix(table)
	if len(matrix.Entries) == 0 {
		t.Fatal("len(matrix.Entries) = 0, want > 0")
	}
	if matrix.Gaps != 0 {
		t.Fatalf("matrix.Gaps = %d, want 0", matrix.Gaps)
	}
	if !strings.Contains(matrix.Markdown(), "explorer-root-browse") {
		t.Fatal("markdown missing scenario")
	}
}
