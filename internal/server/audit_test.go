package server

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestAuditLoggerNilAndCloseNoop(t *testing.T) {
	var logger *AuditLogger
	if err := logger.Log(AuditRecord{Category: "test"}); err != nil {
		t.Fatalf("nil logger Log() error = %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("nil logger Close() error = %v", err)
	}
}

func TestAuditLoggerAddsTimestampAndNewline(t *testing.T) {
	var buf bytes.Buffer
	logger := &AuditLogger{w: &buf}
	if err := logger.Log(AuditRecord{Category: "watch", Action: "poll", Outcome: "success", RequestID: 7}); err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	out := buf.String()
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("audit output = %q, want trailing newline", out)
	}
	var rec AuditRecord
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &rec); err != nil {
		t.Fatalf("Unmarshal(record) error = %v", err)
	}
	if rec.Timestamp == "" {
		t.Fatalf("expected timestamp to be populated")
	}
	if rec.Category != "watch" || rec.Action != "poll" || rec.Outcome != "success" || rec.RequestID != 7 {
		t.Fatalf("record = %+v", rec)
	}
}

func TestAuditLoggerMarshalError(t *testing.T) {
	var buf bytes.Buffer
	logger := &AuditLogger{w: &buf}
	err := logger.Log(AuditRecord{Category: "broken", Details: map[string]any{"bad": make(chan int)}})
	if err == nil {
		t.Fatalf("Log() error = nil, want marshal error")
	}
}
