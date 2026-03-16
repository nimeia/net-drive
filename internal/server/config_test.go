package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadServerConfig(t *testing.T) {
	t.Run("valid-json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "devmount.json")
		if err := os.WriteFile(path, []byte(`{"addr":"127.0.0.1:1234","root_path":"/tmp/workspace","auth_token":"secret","journal_retention":99,"status_addr":"127.0.0.1:8080","audit_log_path":"audit.jsonl"}`), 0o644); err != nil {
			t.Fatalf("WriteFile(config) error = %v", err)
		}
		cfg, err := LoadServerConfig(path)
		if err != nil {
			t.Fatalf("LoadServerConfig() error = %v", err)
		}
		if cfg.Addr != "127.0.0.1:1234" || cfg.RootPath != "/tmp/workspace" || cfg.AuthToken != "secret" || cfg.JournalRetention != 99 || cfg.StatusAddr != "127.0.0.1:8080" || cfg.AuditLogPath != "audit.jsonl" {
			t.Fatalf("LoadServerConfig() = %+v", cfg)
		}
	})

	t.Run("invalid-json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "broken.json")
		if err := os.WriteFile(path, []byte(`{"addr":`), 0o644); err != nil {
			t.Fatalf("WriteFile(config) error = %v", err)
		}
		if _, err := LoadServerConfig(path); err == nil {
			t.Fatalf("LoadServerConfig() error = nil, want invalid json error")
		}
	})
}

func TestServerConfigApplyToServer(t *testing.T) {
	root := t.TempDir()
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	s := New("127.0.0.1:1111")
	s.RootPath = "/old/root"
	s.AuthToken = "old-token"
	s.JournalRetention = 10

	cfg := ServerConfig{
		Addr:             "127.0.0.1:2222",
		RootPath:         root,
		AuthToken:        "new-token",
		JournalRetention: 64,
		AuditLogPath:     auditPath,
	}
	if err := cfg.ApplyToServer(s); err != nil {
		t.Fatalf("ApplyToServer() error = %v", err)
	}
	defer s.Audit.Close()

	if s.Addr != cfg.Addr || s.RootPath != cfg.RootPath || s.AuthToken != cfg.AuthToken || s.JournalRetention != cfg.JournalRetention {
		t.Fatalf("server after ApplyToServer() = %+v", s)
	}
	if s.Audit == nil {
		t.Fatalf("expected audit logger to be created")
	}
	if err := s.Audit.Log(AuditRecord{Category: "test", Action: "apply-config", Outcome: "success"}); err != nil {
		t.Fatalf("Audit.Log() error = %v", err)
	}
	if data, err := os.ReadFile(auditPath); err != nil {
		t.Fatalf("ReadFile(audit) error = %v", err)
	} else if len(data) == 0 {
		t.Fatalf("audit log file is empty")
	}
}
