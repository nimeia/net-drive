package server

import (
	"testing"
	"time"
)

func TestSessionManagerResumeMatrix(t *testing.T) {
	now := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)

	t.Run("success", func(t *testing.T) {
		mgr := NewSessionManager()
		s := mgr.Create("developer", "client-a", 30, now)
		got, found, active, matched := mgr.Resume(s.ID, "client-a", now.Add(10*time.Second))
		if !found || !active || !matched {
			t.Fatalf("Resume() = found:%v active:%v matched:%v, want true,true,true", found, active, matched)
		}
		if !got.ExpiresAt.After(s.ExpiresAt) {
			t.Fatalf("expected lease extension, old=%s new=%s", s.ExpiresAt, got.ExpiresAt)
		}
	})

	t.Run("client-instance-mismatch", func(t *testing.T) {
		mgr := NewSessionManager()
		s := mgr.Create("developer", "client-a", 30, now)
		_, found, active, matched := mgr.Resume(s.ID, "client-b", now.Add(10*time.Second))
		if !found || active || matched {
			t.Fatalf("Resume() = found:%v active:%v matched:%v, want true,false,false", found, active, matched)
		}
	})

	t.Run("expired", func(t *testing.T) {
		mgr := NewSessionManager()
		s := mgr.Create("developer", "client-a", 30, now)
		got, found, active, matched := mgr.Resume(s.ID, "client-a", now.Add(31*time.Second))
		if !found || active || !matched {
			t.Fatalf("Resume() = found:%v active:%v matched:%v, want true,false,true", found, active, matched)
		}
		if got.State != "expired" {
			t.Fatalf("state = %s, want expired", got.State)
		}
	})

	t.Run("missing", func(t *testing.T) {
		mgr := NewSessionManager()
		_, found, active, matched := mgr.Resume(9999, "client-a", now)
		if found || active || matched {
			t.Fatalf("Resume() = found:%v active:%v matched:%v, want false,false,false", found, active, matched)
		}
	})
}
