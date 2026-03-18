package winclientrelease

import (
	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
	"strings"
	"testing"
	"time"
)

func TestValidationPatchAppliesEnvironmentAndStatuses(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	record := NewHostValidationRecord("4.0.0", winclientsmoke.DefaultExplorerSmoke(), table, matrix)
	when := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)
	warnings := record.ApplyPatch(ValidationPatch{
		CompletedBy:       "tester",
		CompletedAt:       &when,
		Environment:       HostEnvironment{Machine: "WIN-01", OSVersion: "Windows 11 24H2", WinFspVersion: "2.0.0", DiagnosticsBundle: "diag.zip"},
		ExplorerScenarios: []ScenarioRecord{{ScenarioID: "explorer-mount-visible", Status: ValidationPass, Notes: "ok"}},
		InstallerRuns:     []InstallerRunRecord{{Channel: "msi", Action: "install", Status: ValidationPass, VersionTo: "4.0.0", LogPath: "logs/msi-install.log"}},
		Notes:             []string{"first-pass backfill"},
	})
	if len(warnings) != 0 {
		t.Fatalf("ApplyPatch warnings = %v, want none", warnings)
	}
	if record.CompletedAt == nil || record.CompletedBy != "tester" {
		t.Fatalf("completion not applied: %+v", record)
	}
	if record.Environment.Machine != "WIN-01" || record.Environment.DiagnosticsBundle != "diag.zip" {
		t.Fatalf("environment not applied: %+v", record.Environment)
	}
	if record.InstallerRuns[0].VersionTo != "4.0.0" || record.InstallerRuns[0].LogPath != "logs/msi-install.log" {
		t.Fatalf("installer run not updated: %+v", record.InstallerRuns[0])
	}
}

func TestPreReleaseIssueListTracksOutstandingValidation(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherReady: true, CallbackBridgeReady: true, ServiceLoopReady: true})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	smoke := winclientsmoke.DefaultExplorerSmoke()
	manifest := NewManifest("4.0.0", nil, table, matrix, smoke)
	record := NewHostValidationRecord("4.0.0", smoke, table, matrix)
	closure := NewReleaseClosure(manifest, record)
	issues := NewPreReleaseIssueList(manifest, record, closure)
	if issues.OpenCount == 0 || issues.ReleaseReady {
		t.Fatalf("unexpected issue list: %+v", issues)
	}
	md := strings.ToLower(issues.Markdown())
	for _, want := range []string{"windows pre-release issue list", "explorer", "installer", "closure"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q", want)
		}
	}
}
