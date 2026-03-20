package server

import (
	"sync/atomic"
	"time"
)

type ControlOpLatencySnapshot struct {
	Count     uint64        `json:"count"`
	Errors    uint64        `json:"errors"`
	TotalWait time.Duration `json:"total_wait"`
	MaxWait   time.Duration `json:"max_wait"`
}

type ControlRuntimeSnapshot struct {
	Hello         ControlOpLatencySnapshot `json:"hello"`
	Auth          ControlOpLatencySnapshot `json:"auth"`
	CreateSession ControlOpLatencySnapshot `json:"create_session"`
	ResumeSession ControlOpLatencySnapshot `json:"resume_session"`
	Heartbeat     ControlOpLatencySnapshot `json:"heartbeat"`
}

type opLatencyCounters struct {
	count       atomic.Uint64
	errors      atomic.Uint64
	totalWaitNS atomic.Uint64
	maxWaitNS   atomic.Uint64
}

func (c *opLatencyCounters) observe(wait time.Duration, failed bool) {
	if wait < 0 {
		wait = 0
	}
	c.count.Add(1)
	if failed {
		c.errors.Add(1)
	}
	ns := uint64(wait)
	c.totalWaitNS.Add(ns)
	for {
		old := c.maxWaitNS.Load()
		if ns <= old || c.maxWaitNS.CompareAndSwap(old, ns) {
			break
		}
	}
}

func (c *opLatencyCounters) snapshot() ControlOpLatencySnapshot {
	return ControlOpLatencySnapshot{Count: c.count.Load(), Errors: c.errors.Load(), TotalWait: time.Duration(c.totalWaitNS.Load()), MaxWait: time.Duration(c.maxWaitNS.Load())}
}

type controlObserver struct {
	hello         opLatencyCounters
	auth          opLatencyCounters
	createSession opLatencyCounters
	resumeSession opLatencyCounters
	heartbeat     opLatencyCounters
}

func (o *controlObserver) observe(op string, wait time.Duration, failed bool) {
	if o == nil {
		return
	}
	switch op {
	case "hello":
		o.hello.observe(wait, failed)
	case "auth":
		o.auth.observe(wait, failed)
	case "create_session":
		o.createSession.observe(wait, failed)
	case "resume_session":
		o.resumeSession.observe(wait, failed)
	case "heartbeat":
		o.heartbeat.observe(wait, failed)
	}
}

func (o *controlObserver) Snapshot() ControlRuntimeSnapshot {
	if o == nil {
		return ControlRuntimeSnapshot{}
	}
	return ControlRuntimeSnapshot{Hello: o.hello.snapshot(), Auth: o.auth.snapshot(), CreateSession: o.createSession.snapshot(), ResumeSession: o.resumeSession.snapshot(), Heartbeat: o.heartbeat.snapshot()}
}
