package winclientrecovery

import (
	"path/filepath"
	"testing"
	"time"

	"developer-mount/internal/winclient"
	"developer-mount/internal/winclientruntime"
)

func TestRecoveryLifecycle(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "recovery.json"))
	store.now = func() time.Time { return time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC) }
	state, err := store.MarkStart("default", winclient.DefaultConfig(), winclientruntime.Snapshot{Phase: winclientruntime.PhaseConnecting})
	if err != nil {
		t.Fatalf("MarkStart() error = %v", err)
	}
	if !state.Dirty || state.ActiveProfile != "default" {
		t.Fatalf("unexpected start state: %+v", state)
	}
	store.now = func() time.Time { return time.Date(2026, 3, 18, 10, 5, 0, 0, time.UTC) }
	state, err = store.Update(winclientruntime.Snapshot{Phase: winclientruntime.PhaseMounted, MountPoint: "M:", RequestedBackend: winclient.HostBackendDispatcherV1})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if state.LastPhase != string(winclientruntime.PhaseMounted) || state.HostBackend != winclient.HostBackendDispatcherV1 {
		t.Fatalf("unexpected updated state: %+v", state)
	}
	state, err = store.MarkCleanExit(winclientruntime.Snapshot{Phase: winclientruntime.PhaseIdle})
	if err != nil {
		t.Fatalf("MarkCleanExit() error = %v", err)
	}
	if state.Dirty {
		t.Fatalf("clean exit should clear dirty marker: %+v", state)
	}
}
