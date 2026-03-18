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
		t.Fatalf("Initialize() error = %v", err)
	}
	state := bridge.Snapshot()
	if !state.Initialized || state.VolumeName != "devmount" || state.CallCount["GetVolumeInfo"] == 0 || state.CallCount["GetFileInfo"] == 0 {
		t.Fatalf("unexpected bridge state after Initialize: %+v", state)
	}
	open, status := bridge.Open("/file.txt")
	if status != StatusSuccess {
		t.Fatalf("Open(file) status = 0x%08x", uint32(status))
	}
	if _, eof, status := bridge.Read(open.HandleID, 0, 4); status != StatusSuccess || !eof {
		t.Fatalf("Read() status=0x%08x eof=%v", uint32(status), eof)
	}
	if status := bridge.SetDeleteOnClose(open.HandleID, true); status != StatusAccessDenied {
		t.Fatalf("SetDeleteOnClose(handle) status = 0x%08x", uint32(status))
	}
	if status := bridge.CanDelete("/file.txt"); status != StatusAccessDenied {
		t.Fatalf("CanDelete(path) status = 0x%08x", uint32(status))
	}
	if _, status := bridge.GetSecurity(open.HandleID); status != StatusSuccess {
		t.Fatalf("GetSecurity(handle) status = 0x%08x", uint32(status))
	}
	if status := bridge.Flush(open.HandleID); status != StatusSuccess {
		t.Fatalf("Flush(file) status = 0x%08x", uint32(status))
	}
	if status := bridge.Cleanup(open.HandleID); status != StatusSuccess {
		t.Fatalf("Cleanup(file) status = 0x%08x", uint32(status))
	}
	if status := bridge.Close(open.HandleID); status != StatusSuccess {
		t.Fatalf("Close(file) status = 0x%08x", uint32(status))
	}
	state = bridge.Snapshot()
	if state.CallCount["Open"] == 0 || state.CallCount["Read"] == 0 || state.CallCount["Flush"] == 0 || state.CallCount["Cleanup"] == 0 || state.CallCount["GetSecurity"] == 0 || state.CallCount["CanDelete"] == 0 || state.CallCount["SetDeleteOnClose"] == 0 || state.CallCount["Close"] == 0 {
		t.Fatalf("call counts missing expected entries: %+v", state.CallCount)
	}
}
