package winclientrelease

import (
	"strings"
	"testing"

	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
)

func TestNewFirstPassFixPlanBuildsItems(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	record := NewHostValidationRecord("0.1.0", winclientsmoke.DefaultExplorerSmoke(), table, matrix)
	_ = record.ApplyScenario("explorer-mount-visible", ValidationFail, "mount missing in Explorer")
	manifest := NewManifest("0.1.0", nil, table, matrix, winclientsmoke.DefaultExplorerSmoke())
	closure := NewReleaseClosure(manifest, record)
	issues := NewPreReleaseIssueList(manifest, record, closure)
	plan := NewFirstPassFixPlan(manifest, record, issues)
	if plan.Blockers == 0 {
		t.Fatalf("Blockers = %d, want > 0", plan.Blockers)
	}
	if len(plan.Items) == 0 {
		t.Fatal("Items empty")
	}
	if !strings.Contains(plan.Markdown(), "Windows First-Pass Fix Plan") {
		t.Fatal("Markdown missing title")
	}
}
