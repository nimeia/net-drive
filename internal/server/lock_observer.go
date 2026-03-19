package server

import (
	"sync"
	"sync/atomic"
	"time"
)

type LockWaitSnapshot struct {
	Acquires     uint64        `json:"acquires"`
	WaitOver50us uint64        `json:"wait_over_50us"`
	WaitOver1ms  uint64        `json:"wait_over_1ms"`
	TotalWait    time.Duration `json:"total_wait"`
	MaxWait      time.Duration `json:"max_wait"`
}

type RWLockWaitSnapshot struct {
	Read  LockWaitSnapshot `json:"read"`
	Write LockWaitSnapshot `json:"write"`
}

type lockWaitCounters struct {
	acquires     atomic.Uint64
	waitOver50us atomic.Uint64
	waitOver1ms  atomic.Uint64
	totalWaitNS  atomic.Uint64
	maxWaitNS    atomic.Uint64
}

func (c *lockWaitCounters) observe(wait time.Duration) {
	if wait < 0 {
		wait = 0
	}
	c.acquires.Add(1)
	ns := uint64(wait)
	c.totalWaitNS.Add(ns)
	if wait >= 50*time.Microsecond {
		c.waitOver50us.Add(1)
	}
	if wait >= time.Millisecond {
		c.waitOver1ms.Add(1)
	}
	for {
		old := c.maxWaitNS.Load()
		if ns <= old || c.maxWaitNS.CompareAndSwap(old, ns) {
			break
		}
	}
}

func (c *lockWaitCounters) snapshot() LockWaitSnapshot {
	return LockWaitSnapshot{
		Acquires:     c.acquires.Load(),
		WaitOver50us: c.waitOver50us.Load(),
		WaitOver1ms:  c.waitOver1ms.Load(),
		TotalWait:    time.Duration(c.totalWaitNS.Load()),
		MaxWait:      time.Duration(c.maxWaitNS.Load()),
	}
}

type observedRWMutex struct {
	mu    sync.RWMutex
	read  lockWaitCounters
	write lockWaitCounters
}

func (m *observedRWMutex) Lock() {
	start := time.Now()
	m.mu.Lock()
	m.write.observe(time.Since(start))
}

func (m *observedRWMutex) Unlock() { m.mu.Unlock() }

func (m *observedRWMutex) RLock() {
	start := time.Now()
	m.mu.RLock()
	m.read.observe(time.Since(start))
}

func (m *observedRWMutex) RUnlock() { m.mu.RUnlock() }

func (m *observedRWMutex) Snapshot() RWLockWaitSnapshot {
	return RWLockWaitSnapshot{Read: m.read.snapshot(), Write: m.write.snapshot()}
}
