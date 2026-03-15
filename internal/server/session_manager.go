package server

import (
	"sync"
	"sync/atomic"
	"time"
)

type Session struct {
	ID               uint64
	PrincipalID      string
	ClientInstanceID string
	LeaseSeconds     uint32
	ExpiresAt        time.Time
	State            string
}

type SessionManager struct {
	nextID   atomic.Uint64
	mu       sync.RWMutex
	sessions map[uint64]Session
}

func NewSessionManager() *SessionManager {
	mgr := &SessionManager{sessions: make(map[uint64]Session)}
	mgr.nextID.Store(1000)
	return mgr
}

func (m *SessionManager) Create(principalID, clientInstanceID string, leaseSeconds uint32, now time.Time) Session {
	id := m.nextID.Add(1)
	s := Session{
		ID:               id,
		PrincipalID:      principalID,
		ClientInstanceID: clientInstanceID,
		LeaseSeconds:     leaseSeconds,
		ExpiresAt:        now.Add(time.Duration(leaseSeconds) * time.Second),
		State:            "active",
	}
	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()
	return s
}

func (m *SessionManager) Heartbeat(id uint64, now time.Time) (Session, bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return Session{}, false, false
	}
	if now.After(s.ExpiresAt) {
		s.State = "expired"
		m.sessions[id] = s
		return s, true, false
	}
	s.ExpiresAt = now.Add(time.Duration(s.LeaseSeconds) * time.Second)
	s.State = "active"
	m.sessions[id] = s
	return s, true, true
}

func (m *SessionManager) ValidateActive(id uint64, now time.Time) (Session, bool, bool) {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return Session{}, false, false
	}
	if now.After(s.ExpiresAt) || s.State != "active" {
		return s, true, false
	}
	return s, true, true
}

func (m *SessionManager) Resume(id uint64, clientInstanceID string, now time.Time) (Session, bool, bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return Session{}, false, false, false
	}
	if s.ClientInstanceID != clientInstanceID {
		return s, true, false, false
	}
	if now.After(s.ExpiresAt) || s.State != "active" {
		s.State = "expired"
		m.sessions[id] = s
		return s, true, false, true
	}
	s.ExpiresAt = now.Add(time.Duration(s.LeaseSeconds) * time.Second)
	s.State = "active"
	m.sessions[id] = s
	return s, true, true, true
}
