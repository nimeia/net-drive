package server

import (
	"errors"
	"io"
	"net"
	"testing"
)

func TestFaultLogObserverSuppressesExpectedConnectionErrors(t *testing.T) {
	obs := &faultLogObserver{}
	cases := []error{
		net.ErrClosed,
		io.EOF,
		io.ErrUnexpectedEOF,
		errors.New("write tcp 127.0.0.1:1->127.0.0.1:2: write: broken pipe"),
		errors.New("read tcp 127.0.0.1:1->127.0.0.1:2: connection reset by peer"),
	}
	for _, err := range cases {
		if !obs.observeSuppressed(err) {
			t.Fatalf("observeSuppressed(%v) = false, want true", err)
		}
	}
	if obs.observeSuppressed(errors.New("boom")) {
		t.Fatal("observeSuppressed(unexpected) = true, want false")
	}
	obs.observeLogged()
	snap := obs.Snapshot()
	if snap.SuppressedNetClosed != 1 || snap.SuppressedEOF != 1 || snap.SuppressedUnexpectedEOF != 1 || snap.SuppressedBrokenPipe != 1 || snap.SuppressedConnReset != 1 || snap.Logged != 1 {
		t.Fatalf("snapshot = %+v", snap)
	}
}
