package winfsp

import (
	"developer-mount/internal/mountcore"
	adapterpkg "developer-mount/internal/winfsp/adapter"
	"testing"
)

func TestDispatcherBridgeInitializesAndRoutesCallbacks(t *testing.T) {
	mount := mountcore.New(callbackFakeClient{}, mountcore.Options{RootNodeID: 1, ReadOnly: true, VolumeName: "devmount"})
	bridge := NewDispatcherBridge(NewCallbacks(adapterpkg.New(mount)))
	if err := bridge.Initialize("/"); err != nil {
		t.Fatalf("Initialize error=%v", err)
	}
	_ = bridge.Create("/new.txt", false)
	open, status := bridge.Open("/file.txt")
	if status != StatusSuccess {
		t.Fatalf("Open status=0x%08x", uint32(status))
	}
	_, _, _ = bridge.Read(open.HandleID, 0, 4)
	_, _ = bridge.Write(open.HandleID, 0, []byte("x"), false)
	_ = bridge.SetBasicInfo(open.HandleID, 0)
	_ = bridge.SetFileSize(open.HandleID, 0, false)
	_ = bridge.SetSecurity(open.HandleID, "x")
	_ = bridge.Rename(open.HandleID, "/renamed.txt", false)
	_ = bridge.Overwrite(open.HandleID, 0, 0, false)
	_ = bridge.SetDeleteOnClose(open.HandleID, true)
	_ = bridge.CanDelete("/file.txt")
	_, _ = bridge.GetSecurity(open.HandleID)
	_ = bridge.Flush(open.HandleID)
	_ = bridge.Cleanup(open.HandleID)
	_ = bridge.Close(open.HandleID)
	state := bridge.Snapshot()
	for _, want := range []string{"Create", "Open", "Read", "Write", "SetBasicInfo", "SetFileSize", "SetSecurity", "Rename", "Overwrite", "Flush", "Cleanup", "GetSecurity", "CanDelete", "SetDeleteOnClose", "Close"} {
		if state.CallCount[want] == 0 {
			t.Fatalf("call count %s missing", want)
		}
	}
}
