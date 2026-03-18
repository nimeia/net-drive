package winclientdiag

import (
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
	if len(report.Checks) < 7 {
		t.Fatalf("Checks length = %d", len(report.Checks))
	}
	if report.Summary.OverallSeverity == "" {
		t.Fatal("OverallSeverity empty")
	}
	for _, want := range []string{"Overall severity:", "Check summary:", "remediation:", "Recovery:"} {
		if !strings.Contains(report.Text(), want) {
			t.Fatalf("Text missing %q", want)
		}
	}
	zipPath, err := Export(filepath.Join(t.TempDir(), "diag.zip"), report)
	if err != nil {
		t.Fatalf("Export error = %v", err)
	}
	if filepath.Base(zipPath) != "diag.zip" {
		t.Fatalf("Export path = %q", zipPath)
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
