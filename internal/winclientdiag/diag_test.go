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
	"developer-mount/internal/winclientruntime"
)

type fakeConn struct{ net.Conn }

func (fakeConn) Close() error { return nil }

func TestCheckerRunAndExport(t *testing.T) {
	checker := Checker{DialTimeout: time.Second, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) { return fakeConn{}, nil }, Now: func() time.Time { return time.Date(2026, 3, 18, 9, 30, 0, 0, time.UTC) }}
	report := checker.Run(context.Background(), winclient.DefaultConfig(), winclientruntime.Snapshot{Phase: winclientruntime.PhaseIdle}, filepath.Join(t.TempDir(), "store.json"), filepath.Join(t.TempDir(), "client.log"), "tail")
	if len(report.Checks) < 5 {
		t.Fatalf("Checks length = %d, want >= 5", len(report.Checks))
	}
	if report.Summary.OverallSeverity == "" {
		t.Fatal("OverallSeverity = empty, want populated")
	}
	for _, want := range []string{"Overall severity:", "Check summary:", "remediation:"} {
		if !strings.Contains(report.Text(), want) {
			t.Fatalf("Text() missing %q: %s", want, report.Text())
		}
	}
	zipPath, err := Export(filepath.Join(t.TempDir(), "diag.zip"), report)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if filepath.Base(zipPath) != "diag.zip" {
		t.Fatalf("Export path = %q, want diag.zip", zipPath)
	}
}
func TestCheckerRunHandlesDialFailure(t *testing.T) {
	checker := Checker{DialTimeout: time.Second, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, errors.New("dial failed")
	}, Now: time.Now}
	report := checker.Run(context.Background(), winclient.DefaultConfig(), winclientruntime.Snapshot{}, "", "", "")
	found := false
	for _, check := range report.Checks {
		if check.Name == "Server TCP connect" {
			found = true
			if check.Status != StatusFail || check.Code != CodeServerConnectFailed || check.Severity != SeverityError {
				t.Fatalf("server-connect = %+v, want fail/error with network code", check)
			}
		}
	}
	if !found {
		t.Fatal("server-connect check not found")
	}
}
func TestRuntimeErrorProducesWarningCheck(t *testing.T) {
	report := NewChecker().Run(context.Background(), winclient.DefaultConfig(), winclientruntime.Snapshot{Phase: winclientruntime.PhaseError, StatusText: "Mount runtime failed", LastError: "host crashed"}, "", "", "")
	for _, check := range report.Checks {
		if check.Code == CodeRuntimeError {
			if check.Status != StatusWarn || check.Severity != SeverityWarning {
				t.Fatalf("runtime check = %+v, want warn/warning", check)
			}
			return
		}
	}
	t.Fatal("runtime error check not found")
}
