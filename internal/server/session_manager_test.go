package server

import (
	"testing"
	"time"
)

func TestSessionManagerHeartbeatExtendsLease(t *testing.T) {
	now := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	mgr := NewSessionManager()
	session := mgr.Create("developer", "client-1", 30, now)
	updated, found, active := mgr.Heartbeat(session.ID, now.Add(10*time.Second))
	if !found || !active {
		t.Fatalf("Heartbeat() expected active session, got found=%v active=%v", found, active)
	}
	if !updated.ExpiresAt.After(session.ExpiresAt) {
		t.Fatalf("expected expiry to move forward: old=%s new=%s", session.ExpiresAt, updated.ExpiresAt)
	}
}
