package winfsp

import (
	"developer-mount/internal/mountcore"
	adapterpkg "developer-mount/internal/winfsp/adapter"
	"testing"
)

func TestDispatcherServiceStartStop(t *testing.T) {
	mount := mountcore.New(callbackFakeClient{}, mountcore.Options{RootNodeID: 1, ReadOnly: true, VolumeName: "devmount"})
	bridge := NewDispatcherBridge(NewCallbacks(adapterpkg.New(mount)))
	abi := NewDispatcherABI(bridge)
	svc := NewDispatcherService(dispatcherBindings{}, "M:", abi)
	if err := svc.Start("/"); err != nil {
		t.Fatalf("Start error=%v", err)
	}
	for _, want := range []string{"Create", "Write", "SetBasicInfo", "SetFileSize", "SetSecurity", "Rename", "Overwrite", "SetDeleteOnClose"} {
		if bridge.Snapshot().CallCount[want] == 0 {
			t.Fatalf("bridge call count %s missing", want)
		}
	}
	svc.Stop()
}
