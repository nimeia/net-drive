package winclientrelease

import (
	"strings"
	"testing"

	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
)

func TestNewReleaseCandidateReflectsOpenIssues(t *testing.T) {
	table := winfsp.DefaultNativeCallbackTable(winfsp.BindingInfo{})
	matrix := winclientsmoke.DefaultExplorerRequestMatrix(table)
	smoke := winclientsmoke.DefaultExplorerSmoke()
	manifest := NewManifest("0.1.0", []Artifact{{Name: "a", Kind: "exe", Path: "dist/a.exe"}}, table, matrix, smoke)
	record := NewHostValidationRecord("0.1.0", smoke, table, matrix)
	closure := NewReleaseClosure(manifest, record)
	issues := NewPreReleaseIssueList(manifest, record, closure)
	rc := NewReleaseCandidate(manifest, record, closure, issues)
	if rc.FinalStatus == "rc-ready" {
		t.Fatalf("FinalStatus = %s, want non-ready", rc.FinalStatus)
	}
	if rc.OpenIssues == 0 {
		t.Fatalf("OpenIssues = %d, want > 0", rc.OpenIssues)
	}
	if !strings.Contains(rc.Markdown(), "Windows Release Candidate") {
		t.Fatal("Markdown missing title")
	}
}
