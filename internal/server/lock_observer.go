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

type DurationSnapshot struct {
	Count    uint64        `json:"count"`
	Over50us uint64        `json:"over_50us"`
	Over1ms  uint64        `json:"over_1ms"`
	Over10ms uint64        `json:"over_10ms"`
	Total    time.Duration `json:"total"`
	Max      time.Duration `json:"max"`
}

type RWLockWaitSnapshot struct {
	Read      LockWaitSnapshot `json:"read"`
	Write     LockWaitSnapshot `json:"write"`
	WriteHold DurationSnapshot `json:"write_hold"`
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

type durationCounters struct {
	count    atomic.Uint64
	over50us atomic.Uint64
	over1ms  atomic.Uint64
	over10ms atomic.Uint64
	totalNS  atomic.Uint64
	maxNS    atomic.Uint64
}

func (c *durationCounters) observe(d time.Duration) {
	if d < 0 {
		d = 0
	}
	c.count.Add(1)
	ns := uint64(d)
	c.totalNS.Add(ns)
	if d >= 50*time.Microsecond {
		c.over50us.Add(1)
	}
	if d >= time.Millisecond {
		c.over1ms.Add(1)
	}
	if d >= 10*time.Millisecond {
		c.over10ms.Add(1)
	}
	for {
		old := c.maxNS.Load()
		if ns <= old || c.maxNS.CompareAndSwap(old, ns) {
			break
		}
	}
}

func (c *durationCounters) snapshot() DurationSnapshot {
	return DurationSnapshot{
		Count:    c.count.Load(),
		Over50us: c.over50us.Load(),
		Over1ms:  c.over1ms.Load(),
		Over10ms: c.over10ms.Load(),
		Total:    time.Duration(c.totalNS.Load()),
		Max:      time.Duration(c.maxNS.Load()),
	}
}

type observedRWMutex struct {
	mu          sync.RWMutex
	read        lockWaitCounters
	write       lockWaitCounters
	writeHold   durationCounters
	writeHeldAt atomic.Int64
}

func (m *observedRWMutex) Lock() {
	start := time.Now()
	m.mu.Lock()
	m.write.observe(time.Since(start))
	m.writeHeldAt.Store(time.Now().UnixNano())
}

func (m *observedRWMutex) Unlock() {
	heldAt := m.writeHeldAt.Swap(0)
	if heldAt > 0 {
		m.writeHold.observe(time.Since(time.Unix(0, heldAt)))
	}
	m.mu.Unlock()
}

func (m *observedRWMutex) RLock() {
	start := time.Now()
	m.mu.RLock()
	m.read.observe(time.Since(start))
}

func (m *observedRWMutex) RUnlock() { m.mu.RUnlock() }

func (m *observedRWMutex) Snapshot() RWLockWaitSnapshot {
	return RWLockWaitSnapshot{Read: m.read.snapshot(), Write: m.write.snapshot(), WriteHold: m.writeHold.snapshot()}
}
