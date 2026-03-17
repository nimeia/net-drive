package server

import (
	"encoding/json"
	"os"
)

type ServerConfig struct {
	Addr             string `json:"addr"`
	RootPath         string `json:"root_path"`
	AuthToken        string `json:"auth_token"`
	JournalRetention int    `json:"journal_retention"`
	StatusAddr       string `json:"status_addr"`
	AuditLogPath     string `json:"audit_log_path"`
}

func LoadServerConfig(path string) (ServerConfig, error) {
	var cfg ServerConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
func (c ServerConfig) ApplyToServer(s *Server) error {
	if c.Addr != "" {
		s.Addr = c.Addr
	}
	if c.RootPath != "" {
		s.RootPath = c.RootPath
	}
	if c.AuthToken != "" {
		s.AuthToken = c.AuthToken
	}
	if c.JournalRetention > 0 {
		s.JournalRetention = c.JournalRetention
	}
	if c.AuditLogPath != "" {
		audit, err := NewAuditLogger(c.AuditLogPath)
		if err != nil {
			return err
		}
		s.Audit = audit
	}
	return nil
}
