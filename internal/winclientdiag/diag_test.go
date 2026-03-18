package winclientdiag

import (
	"archive/zip"
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"developer-mount/internal/winclient"
	"developer-mount/internal/winclientrecovery"
	"developer-mount/internal/winclientruntime"
	"developer-mount/internal/winclientsmoke"
)

type fakeConn struct{ net.Conn }

func (fakeConn) Close() error { return nil }
func TestCheckerRunAndExport(t *testing.T) {
	checker := Checker{DialTimeout: time.Second, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) { return fakeConn{}, nil }, Now: func() time.Time { return time.Date(2026, 3, 18, 9, 30, 0, 0, time.UTC) }}
	report := checker.Run(context.Background(), winclient.DefaultConfig(), winclientruntime.Snapshot{Phase: winclientruntime.PhaseIdle}, filepath.Join(t.TempDir(), "store.json"), filepath.Join(t.TempDir(), "client.log"), filepath.Join(t.TempDir(), "recovery.json"), winclientrecovery.State{Dirty: true, ActiveProfile: "default"}, "tail", winclientsmoke.DefaultExplorerSmoke())
	if len(report.Checks) < 9 || len(report.CallbackTable.Callbacks) == 0 || len(report.ExplorerMatrix.Entries) == 0 {
		t.Fatalf("unexpected report sizes: checks=%d callbacks=%d matrix=%d", len(report.Checks), len(report.CallbackTable.Callbacks), len(report.ExplorerMatrix.Entries))
	}
	for _, want := range []string{"Overall severity:", "Check summary:", "remediation:", "Recovery:", "Native callback table:", "Explorer request matrix:"} {
		if !strings.Contains(report.Text(), want) {
			t.Fatalf("Text missing %q", want)
		}
	}
	zipPath, err := Export(filepath.Join(t.TempDir(), "diag.zip"), report)
	if err != nil {
		t.Fatalf("Export error = %v", err)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("OpenReader error = %v", err)
	}
	defer zr.Close()
	have := map[string]bool{}
	for _, f := range zr.File {
		have[f.Name] = true
	}
	for _, want := range []string{"report.txt", "report.json", "explorer-smoke.md", "explorer-smoke.json", "explorer-request-matrix.md", "explorer-request-matrix.json", "winfsp-native-callbacks.md", "winfsp-native-callbacks.json", "recovery.json", "log-tail.txt", "windows-host-validation-template.md", "windows-host-validation-template.json", "windows-host-validation-result-template.md", "windows-host-validation-result-template.json", "windows-host-backfill-patch-template.md", "windows-host-backfill-patch-template.json", "windows-release-closure-template.md", "windows-release-closure-template.json", "windows-pre-release-issues.md", "windows-pre-release-issues.json", "windows-first-pass-fix-plan.md", "windows-first-pass-fix-plan.json", "windows-release-candidate.md", "windows-release-candidate.json"} {
		if !have[want] {
			t.Fatalf("zip missing %q", want)
		}
	}
}
func TestCheckerRunHandlesDialFailure(t *testing.T) {
	checker := Checker{DialTimeout: time.Second, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, errors.New("dial failed")
	}, Now: time.Now}
	report := checker.Run(context.Background(), winclient.DefaultConfig(), winclientruntime.Snapshot{}, "", "", "", winclientrecovery.State{}, "", nil)
	found := false
	for _, check := range report.Checks {
		if check.Name == "Server TCP connect" {
			found = true
			if check.Status != StatusFail || check.Code != CodeServerConnectFailed || check.Severity != SeverityError {
				t.Fatalf("server-connect = %+v", check)
			}
		}
	}
	if !found {
		t.Fatal("server-connect check not found")
	}
}
func TestRuntimeErrorProducesWarningCheck(t *testing.T) {
	report := NewChecker().Run(context.Background(), winclient.DefaultConfig(), winclientruntime.Snapshot{Phase: winclientruntime.PhaseError, StatusText: "Mount runtime failed", LastError: "host crashed"}, "", "", "", winclientrecovery.State{}, "", nil)
	for _, check := range report.Checks {
		if check.Code == CodeRuntimeError {
			if check.Status != StatusWarn || check.Severity != SeverityWarning {
				t.Fatalf("runtime check = %+v", check)
			}
			return
		}
	}
	t.Fatal("runtime error check not found")
}
