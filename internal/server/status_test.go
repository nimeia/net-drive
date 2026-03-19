package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSnapshotStatusUsesBackendRootAndUptime(t *testing.T) {
	root := t.TempDir()
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	startedAt := time.Date(2026, 3, 16, 8, 0, 0, 0, time.UTC)
	s := New("127.0.0.1:9000")
	s.RootPath = "/configured/root"
	s.Backend = backend
	s.JournalRetention = 123
	s.StartedAt = startedAt

	status := s.SnapshotStatus(startedAt.Add(90 * time.Second))
	if status.RootPath != backend.rootPath {
		t.Fatalf("status.RootPath = %q, want backend root %q", status.RootPath, backend.rootPath)
	}
	if status.Addr != s.Addr || status.JournalRetention != s.JournalRetention {
		t.Fatalf("status = %+v", status)
	}
	if status.UptimeSeconds != 90 {
		t.Fatalf("status.UptimeSeconds = %d, want 90", status.UptimeSeconds)
	}
	if status.StartedAt != startedAt.UTC().Format(time.RFC3339) {
		t.Fatalf("status.StartedAt = %q, want %q", status.StartedAt, startedAt.UTC().Format(time.RFC3339))
	}
	if len(status.Capabilities) == 0 {
		t.Fatalf("expected non-empty capabilities")
	}
}

func TestStatusHandlerHealthzStatusAndRuntimez(t *testing.T) {
	root := t.TempDir()
	s := New("127.0.0.1:9001")
	s.RootPath = root
	s.JournalRetention = 64
	s.StartedAt = time.Now().Add(-30 * time.Second)
	backend, err := newMetadataBackend(root)
	if err != nil {
		t.Fatalf("newMetadataBackend() error = %v", err)
	}
	s.Backend = backend
	s.Journal = newJournalBroker(backend, time.Now, s.JournalRetention)
	if _, err := backend.GetAttr(backend.RootNodeID()); err != nil {
		t.Fatalf("GetAttr(root) error = %v", err)
	}

	h := NewStatusHandler(s)

	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRec := httptest.NewRecorder()
	h.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("/healthz status code = %d, want %d", healthRec.Code, http.StatusOK)
	}
	var health map[string]any
	if err := json.Unmarshal(healthRec.Body.Bytes(), &health); err != nil {
		t.Fatalf("Unmarshal(/healthz) error = %v", err)
	}
	if ok, _ := health["ok"].(bool); !ok {
		t.Fatalf("/healthz body = %v, want ok=true", health)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/status", nil)
	statusRec := httptest.NewRecorder()
	h.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("/status status code = %d, want %d", statusRec.Code, http.StatusOK)
	}
	if got := statusRec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("/status content-type = %q, want application/json", got)
	}
	var status ServerStatus
	if err := json.Unmarshal(statusRec.Body.Bytes(), &status); err != nil {
		t.Fatalf("Unmarshal(/status) error = %v", err)
	}
	if status.Name != "devmount-server" || status.Addr != s.Addr || status.JournalRetention != s.JournalRetention {
		t.Fatalf("status = %+v", status)
	}

	runtimeReq := httptest.NewRequest(http.MethodGet, "/runtimez", nil)
	runtimeRec := httptest.NewRecorder()
	h.ServeHTTP(runtimeRec, runtimeReq)
	if runtimeRec.Code != http.StatusOK {
		t.Fatalf("/runtimez status code = %d, want %d", runtimeRec.Code, http.StatusOK)
	}
	var runtimeSnap RuntimeSnapshot
	if err := json.Unmarshal(runtimeRec.Body.Bytes(), &runtimeSnap); err != nil {
		t.Fatalf("Unmarshal(/runtimez) error = %v", err)
	}
	if runtimeSnap.At.IsZero() {
		t.Fatalf("runtime snapshot time is zero")
	}
	if runtimeSnap.Metadata.Locks.Read.Acquires == 0 {
		t.Fatalf("runtime snapshot locks = %+v, want acquisitions", runtimeSnap.Metadata.Locks)
	}
}
