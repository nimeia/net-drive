package winfsp

import (
	"testing"

	"developer-mount/internal/mountcore"
	adapterpkg "developer-mount/internal/winfsp/adapter"
)

func TestDispatcherServiceStartStop(t *testing.T) {
	mount := mountcore.New(callbackFakeClient{}, mountcore.Options{RootNodeID: 1, ReadOnly: true, VolumeName: "devmount"})
	bridge := NewDispatcherBridge(NewCallbacks(adapterpkg.New(mount)))
	abi := NewDispatcherABI(bridge)
	svc := NewDispatcherService(dispatcherBindings{}, "M:", abi)
	if err := svc.Start("/"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	snap := svc.Snapshot()
	if !snap.Created || !snap.Running || snap.StartCount == 0 {
		t.Fatalf("unexpected service snapshot: %+v", snap)
	}
	abiSnap := abi.Snapshot()
	if abiSnap.LastOperation == "" || abiSnap.Requests == 0 {
		t.Fatalf("unexpected abi snapshot: %+v", abiSnap)
	}
	svc.Stop()
	snap = svc.Snapshot()
	if snap.Running || snap.StopCount == 0 {
		t.Fatalf("unexpected service stop snapshot: %+v", snap)
	}
}
