package server

import (
	"developer-mount/internal/protocol"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

type AuditRecord struct {
	Timestamp string         `json:"timestamp"`
	Category  string         `json:"category"`
	Action    string         `json:"action"`
	Outcome   string         `json:"outcome"`
	RequestID uint64         `json:"request_id,omitempty"`
	SessionID uint64         `json:"session_id,omitempty"`
	NodeID    uint64         `json:"node_id,omitempty"`
	Path      string         `json:"path,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}
type AuditLogger struct {
	mu     sync.Mutex
	w      io.Writer
	closer io.Closer
}

func NewAuditLogger(path string) (*AuditLogger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &AuditLogger{w: f, closer: f}, nil
}
func (a *AuditLogger) Log(rec AuditRecord) error {
	if a == nil || a.w == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if rec.Timestamp == "" {
		rec.Timestamp = protocol.NowRFC3339(time.Now().UTC())
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = a.w.Write(append(data, byte('\n')))
	return err
}
func (a *AuditLogger) Close() error {
	if a == nil || a.closer == nil {
		return nil
	}
	return a.closer.Close()
}
