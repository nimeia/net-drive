package winclientrelease

import (
	"strings"
	"testing"
	"time"

	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
)

func TestValidationIntakeReportDetectsMissingEvidence(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	smoke := winclientsmoke.DefaultExplorerSmoke()
	manifest := NewManifest("0.1.0", nil, table, matrix, smoke)
	record := NewHostValidationRecord("0.1.0", smoke, table, matrix)
	report := NewValidationIntakeReport(manifest, record)
	if report.ReadyForTargetedFix {
		t.Fatal("ReadyForTargetedFix = true, want false")
	}
	if report.MissingEvidenceCount == 0 {
		t.Fatal("MissingEvidenceCount = 0, want > 0")
	}
	if !strings.Contains(report.Markdown(), "Windows Validation Intake Report") {
		t.Fatal("Markdown missing title")
	}
}

func TestValidationIntakeReportBecomesReady(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	smoke := winclientsmoke.DefaultExplorerSmoke()
	manifest := NewManifest("0.1.0", nil, table, matrix, smoke)
	record := NewHostValidationRecord("0.1.0", smoke, table, matrix)
	now := time.Now().UTC()
	record.CompletedAt = &now
	record.CompletedBy = "tester"
	record.Environment.Machine = "WIN-01"
	record.Environment.OSVersion = "Windows 11 24H2"
	record.Environment.WinFspVersion = "2.0"
	record.Environment.DiagnosticsBundle = "diag.zip"
	record.Environment.InstallerLogDir = "C:/logs"
	for i := range record.ExplorerScenarios {
		record.ExplorerScenarios[i].Status = ValidationPass
	}
	for i := range record.InstallerChecklist {
		record.InstallerChecklist[i].Status = ValidationPass
	}
	for i := range record.RecoveryChecklist {
		record.RecoveryChecklist[i].Status = ValidationPass
	}
	for i := range record.InstallerRuns {
		record.InstallerRuns[i].Status = ValidationPass
		record.InstallerRuns[i].LogPath = "C:/logs/run.log"
	}
	record.RecomputeSummary()
	report := NewValidationIntakeReport(manifest, record)
	if !report.ReadyForTargetedFix {
		t.Fatal("ReadyForTargetedFix = false, want true")
	}
	if report.MissingEvidenceCount != 0 {
		t.Fatalf("MissingEvidenceCount = %d, want 0", report.MissingEvidenceCount)
	}
}
