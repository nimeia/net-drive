package winclientlog

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecordAndTailStructuredEntry(t *testing.T) {
	logger := New(filepath.Join(t.TempDir(), "win32-client.log"))
	logger.now = func() time.Time { return time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC) }
	if err := logger.Info("startup"); err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if err := logger.Record(Entry{Level: LevelError, Code: "diag.export_failed", Component: "diagnostics", Message: "boom", Fields: map[string]string{"path": `C:\\temp\\diag.zip`}}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	tail, err := logger.Tail(0)
	if err != nil {
		t.Fatalf("Tail() error = %v", err)
	}
	for _, want := range []string{"level=INFO", `code="ui.info"`, `msg="startup"`, `code="diag.export_failed"`, `component="diagnostics"`, `path="C:\\\\temp\\\\diag.zip"`} {
		if !strings.Contains(tail, want) {
			t.Fatalf("Tail() = %q, missing %q", tail, want)
		}
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
