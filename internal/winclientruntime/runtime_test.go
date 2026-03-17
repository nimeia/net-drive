package winclientruntime

import (
	"context"
	"errors"
	"testing"
	"time"

	"developer-mount/internal/winclient"
)

type fakeBuilder struct {
	session Session
	err     error
}

func (b fakeBuilder) Build(config winclient.Config) (Session, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.session, nil
}

type fakeSession struct {
	info SessionInfo
	run  func(ctx context.Context) error
}

func (s fakeSession) Info() SessionInfo {
	return s.info
}

func (s fakeSession) Run(ctx context.Context) error {
	if s.run != nil {
		return s.run(ctx)
	}
	<-ctx.Done()
	return ctx.Err()
}

func TestRuntimeStartStopLifecycle(t *testing.T) {
	now := time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC)
	r := NewWithClock(fakeBuilder{session: fakeSession{
		info: SessionInfo{ServerAddr: "127.0.0.1:17890", MountPoint: "M:", VolumePrefix: "devmount", RemotePath: "/", SessionID: 42},
	}}, func() time.Time { return now })

	if err := r.Start(winclient.DefaultConfig(), "default"); err != nil {
		t.Fatalf("Start error = %v", err)
	}
	snapshot := r.Snapshot()
	if snapshot.Phase != PhaseMounted {
		t.Fatalf("Phase = %s, want %s", snapshot.Phase, PhaseMounted)
	}
	if snapshot.SessionID != 42 {
		t.Fatalf("SessionID = %d, want 42", snapshot.SessionID)
	}

	if err := r.Stop(); err != nil {
		t.Fatalf("Stop error = %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if snap := r.Snapshot(); snap.Phase == PhaseIdle {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("runtime did not return to idle: %+v", r.Snapshot())
}

func TestRuntimeStartBuildErrorTransitionsToError(t *testing.T) {
	r := New(fakeBuilder{err: errors.New("dial failed")})
	err := r.Start(winclient.DefaultConfig(), "broken")
	if err == nil {
		t.Fatal("Start error = nil, want error")
	}
	snapshot := r.Snapshot()
	if snapshot.Phase != PhaseError {
		t.Fatalf("Phase = %s, want %s", snapshot.Phase, PhaseError)
	}
	if snapshot.LastError != "dial failed" {
		t.Fatalf("LastError = %q, want dial failed", snapshot.LastError)
	}
}

func TestRuntimeSessionErrorTransitionsToError(t *testing.T) {
	r := New(fakeBuilder{session: fakeSession{
		info: SessionInfo{ServerAddr: "127.0.0.1:17890", MountPoint: "M:", VolumePrefix: "devmount", RemotePath: "/"},
		run: func(ctx context.Context) error {
			return errors.New("host crashed")
		},
	}})
	if err := r.Start(winclient.DefaultConfig(), "default"); err != nil {
		t.Fatalf("Start error = %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if snap := r.Snapshot(); snap.Phase == PhaseError {
			if snap.LastError != "host crashed" {
				t.Fatalf("LastError = %q, want host crashed", snap.LastError)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("runtime did not transition to error: %+v", r.Snapshot())
}

func TestRuntimeRejectsConcurrentStart(t *testing.T) {
	blocker := make(chan struct{})
	r := New(fakeBuilder{session: fakeSession{
		info: SessionInfo{ServerAddr: "127.0.0.1:17890", MountPoint: "M:", VolumePrefix: "devmount", RemotePath: "/"},
		run: func(ctx context.Context) error {
			<-blocker
			<-ctx.Done()
			return ctx.Err()
		},
	}})
	if err := r.Start(winclient.DefaultConfig(), "default"); err != nil {
		t.Fatalf("Start error = %v", err)
	}
	if err := r.Start(winclient.DefaultConfig(), "default"); err == nil {
		t.Fatal("second Start error = nil, want busy error")
	}
	close(blocker)
	_ = r.Stop()
}
