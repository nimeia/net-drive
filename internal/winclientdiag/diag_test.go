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
	if len(report.Checks) < 4 {
		t.Fatalf("Checks length = %d, want >= 4", len(report.Checks))
	}
	if !strings.Contains(report.Text(), "Checks:") {
		t.Fatalf("Text() missing checks section: %s", report.Text())
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
		if check.Name == "server-connect" {
			found = true
			if check.Status != StatusFail {
				t.Fatalf("server-connect status = %s, want fail", check.Status)
			}
		}
	}
	if !found {
		t.Fatal("server-connect check not found")
	}
}
