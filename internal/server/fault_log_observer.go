package server

import (
	"errors"
	"io"
	"net"
	"strings"
	"sync/atomic"
)

type FaultLogRuntimeSnapshot struct {
	SuppressedNetClosed     uint64 `json:"suppressed_net_closed"`
	SuppressedEOF           uint64 `json:"suppressed_eof"`
	SuppressedUnexpectedEOF uint64 `json:"suppressed_unexpected_eof"`
	SuppressedBrokenPipe    uint64 `json:"suppressed_broken_pipe"`
	SuppressedConnReset     uint64 `json:"suppressed_conn_reset"`
	Logged                  uint64 `json:"logged"`
}

type faultLogObserver struct {
	suppressedNetClosed     atomic.Uint64
	suppressedEOF           atomic.Uint64
	suppressedUnexpectedEOF atomic.Uint64
	suppressedBrokenPipe    atomic.Uint64
	suppressedConnReset     atomic.Uint64
	logged                  atomic.Uint64
}

func (o *faultLogObserver) observeSuppressed(err error) bool {
	if o == nil || err == nil {
		return false
	}
	switch {
	case errors.Is(err, net.ErrClosed):
		o.suppressedNetClosed.Add(1)
		return true
	case errors.Is(err, io.EOF):
		o.suppressedEOF.Add(1)
		return true
	case errors.Is(err, io.ErrUnexpectedEOF):
		o.suppressedUnexpectedEOF.Add(1)
		return true
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "broken pipe") {
		o.suppressedBrokenPipe.Add(1)
		return true
	}
	if strings.Contains(lower, "connection reset by peer") || strings.Contains(lower, "forcibly closed by the remote host") {
		o.suppressedConnReset.Add(1)
		return true
	}
	return false
}

func (o *faultLogObserver) observeLogged() {
	if o == nil {
		return
	}
	o.logged.Add(1)
}

func (o *faultLogObserver) Snapshot() FaultLogRuntimeSnapshot {
	if o == nil {
		return FaultLogRuntimeSnapshot{}
	}
	return FaultLogRuntimeSnapshot{
		SuppressedNetClosed:     o.suppressedNetClosed.Load(),
		SuppressedEOF:           o.suppressedEOF.Load(),
		SuppressedUnexpectedEOF: o.suppressedUnexpectedEOF.Load(),
		SuppressedBrokenPipe:    o.suppressedBrokenPipe.Load(),
		SuppressedConnReset:     o.suppressedConnReset.Load(),
		Logged:                  o.logged.Load(),
	}
}
