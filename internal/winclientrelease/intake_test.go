package winclientrelease

import (
	"testing"

	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
)

func testValidationRecord() HostValidationRecord {
	return NewHostValidationRecord("0.1.0", winclientsmoke.DefaultExplorerSmoke(), winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{}), winclientsmoke.DefaultExplorerRequestMatrix(winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{})))
}

func TestNewValidationPatchTemplateCopiesAllSections(t *testing.T) {
	record := testValidationRecord()
	patch := NewValidationPatchTemplate(record)
	if len(patch.ExplorerScenarios) != len(record.ExplorerScenarios) {
		t.Fatalf("ExplorerScenarios len = %d, want %d", len(patch.ExplorerScenarios), len(record.ExplorerScenarios))
	}
	if len(patch.InstallerChecklist) != len(record.InstallerChecklist) || len(patch.RecoveryChecklist) != len(record.RecoveryChecklist) || len(patch.InstallerRuns) != len(record.InstallerRuns) {
		t.Fatalf("patch sections mismatch: %+v", patch)
	}
	for _, item := range patch.ExplorerScenarios {
		if item.Status != ValidationNotRun {
			t.Fatalf("scenario status = %s, want not-run", item.Status)
		}
	}
}

func TestApplyInstallerResultSetUpdatesRunsAndChecklist(t *testing.T) {
	record := testValidationRecord()
	set := NewInstallerResultSetTemplate("0.1.0")
	set.LogDir = `C:\logs`
	set.MSI.Install.Status = ValidationPass
	set.MSI.Install.LogPath = `C:\logs\install.log`
	set.EXE.PortableLaunch.Status = ValidationWarn
	set.EXE.PortableLaunch.Notes = "manual warning"
	warnings := record.ApplyInstallerResultSet(set)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if record.Environment.InstallerLogDir != `C:\logs` {
		t.Fatalf("InstallerLogDir = %q", record.Environment.InstallerLogDir)
	}
	if got := record.InstallerRuns[0].Status; got != ValidationPass {
		t.Fatalf("MSI install status = %s, want pass", got)
	}
	if got := record.InstallerRuns[3].Status; got != ValidationWarn {
		t.Fatalf("EXE launch status = %s, want warn", got)
	}
	foundChecklist := false
	for _, item := range record.InstallerChecklist {
		if item.Item == "MSI install succeeded" && item.Status == ValidationPass {
			foundChecklist = true
		}
	}
	if !foundChecklist {
		t.Fatal("installer checklist not updated from installer result set")
	}
}
