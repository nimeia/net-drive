package winclientlog

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendAndTail(t *testing.T) {
	logger := New(filepath.Join(t.TempDir(), "win32-client.log"))
	logger.now = func() time.Time { return time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC) }
	if err := logger.Info("startup"); err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if err := logger.Error("boom"); err != nil {
		t.Fatalf("Error() error = %v", err)
	}
	tail, err := logger.Tail(0)
	if err != nil {
		t.Fatalf("Tail() error = %v", err)
	}
	if !strings.Contains(tail, "[INFO] startup") || !strings.Contains(tail, "[ERROR] boom") {
		t.Fatalf("Tail() = %q, want both log lines", tail)
	}
}

func TestDefaultPathRespectsOverride(t *testing.T) {
	t.Setenv(overrideEnvName, filepath.Join(t.TempDir(), "custom.log"))
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}
	if filepath.Base(path) != "custom.log" {
		t.Fatalf("DefaultPath() = %q, want override path", path)
	}
}
