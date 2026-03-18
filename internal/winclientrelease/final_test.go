package winclientrelease

import (
	"strings"
	"testing"
	"time"

	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
)

func TestNewFinalReleaseReflectsOutstandingInputs(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	smoke := winclientsmoke.DefaultExplorerSmoke()
	manifest := NewManifest("0.1.0", nil, table, matrix, smoke)
	record := NewHostValidationRecord("0.1.0", smoke, table, matrix)
	closure := NewReleaseClosure(manifest, record)
	issues := NewPreReleaseIssueList(manifest, record, closure)
	intake := NewValidationIntakeReport(manifest, record)
	rc := NewReleaseCandidate(manifest, record, closure, issues)
	final := NewFinalRelease(manifest, record, intake, closure, issues, rc)
	if final.PublishReady {
		t.Fatal("PublishReady = true, want false")
	}
	if final.FinalStatus == "publish-ready" {
		t.Fatalf("FinalStatus = %s, want non-ready", final.FinalStatus)
	}
	if !strings.Contains(final.SignoffMarkdown(), "Windows Final Release Sign-Off") {
		t.Fatal("SignoffMarkdown missing title")
	}
}

func TestNewFinalReleaseBecomesPublishReady(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	smoke := winclientsmoke.DefaultExplorerSmoke()
	manifest := NewManifest("0.1.0", []Artifact{{Name: "a", Kind: "exe", Path: "dist/a.exe"}}, table, matrix, smoke)
	record := NewHostValidationRecord("0.1.0", smoke, table, matrix)
	now := time.Now().UTC()
	record.CompletedAt = &now
	record.CompletedBy = "tester"
	record.Environment.Machine = "WIN-01"
	record.Environment.OSVersion = "Windows 11"
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
	closure := NewReleaseClosure(manifest, record)
	issues := NewPreReleaseIssueList(manifest, record, closure)
	intake := NewValidationIntakeReport(manifest, record)
	rc := NewReleaseCandidate(manifest, record, closure, issues)
	final := NewFinalRelease(manifest, record, intake, closure, issues, rc)
	if !final.PublishReady {
		t.Fatal("PublishReady = false, want true")
	}
	if final.FinalStatus != "publish-ready" {
		t.Fatalf("FinalStatus = %s, want publish-ready", final.FinalStatus)
	}
}
