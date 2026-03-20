package server

import (
	"encoding/json"
	"net/http"
	"time"
)

type ServerStatus struct {
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	Addr             string   `json:"addr"`
	RootPath         string   `json:"root_path"`
	JournalRetention int      `json:"journal_retention"`
	StartedAt        string   `json:"started_at"`
	UptimeSeconds    int64    `json:"uptime_seconds"`
	Capabilities     []string `json:"capabilities"`
}

var DefaultStatusCapabilities = []string{"control", "metadata", "data", "events", "recovery", "workspace-profile", "small-file-cache", "journal-polling", "runtime-snapshot", "lock-wait-observer", "control-op-latency"}

func (s *Server) SnapshotStatus(now time.Time) ServerStatus {
	root := s.RootPath
	if s.Backend != nil && s.Backend.rootPath != "" {
		root = s.Backend.rootPath
	}
	return ServerStatus{Name: "devmount-server", Version: "0.8.0", Addr: s.Addr, RootPath: root, JournalRetention: s.JournalRetention, StartedAt: s.StartedAt.UTC().Format(time.RFC3339), UptimeSeconds: int64(now.Sub(s.StartedAt).Seconds()), Capabilities: DefaultStatusCapabilities}
}
func NewStatusHandler(s *Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.SnapshotStatus(time.Now()))
	})
	mux.HandleFunc("/runtimez", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.SnapshotRuntime(time.Now()))
	})
	return mux
}
