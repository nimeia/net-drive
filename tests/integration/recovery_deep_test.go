package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"developer-mount/internal/protocol"
)

func TestRecoveryWritableDeleteOnCloseRecoveredHandle(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	cli1, _ := newRecoveryEnv(t, root, 64, now, "recover-write-client")

	createResp, err := cli1.Create(cli1.RootNodeID, "recover-write.txt", false)
	if err != nil {
		t.Fatalf("Create(recover-write.txt) error = %v", err)
	}
	if _, err := cli1.Write(createResp.HandleID, 0, []byte("before-recovery")); err != nil {
		t.Fatalf("Write(before-recovery) error = %v", err)
	}
	if _, err := cli1.Flush(createResp.HandleID); err != nil {
		t.Fatalf("Flush(before-recovery) error = %v", err)
	}
	if _, err := cli1.CloseHandle(createResp.HandleID); err != nil {
		t.Fatalf("CloseHandle(before-recovery) error = %v", err)
	}

	sessionID := cli1.SessionID
	addr := cli1.Addr
	if err := cli1.Close(); err != nil {
		t.Fatalf("Close(cli1) error = %v", err)
	}

	cli2 := reconnectAndResume(t, addr, sessionID, "recover-write-client")
	recovered, err := cli2.RecoverHandles([]protocol.RecoverHandleSpec{{
		PreviousHandleID: createResp.HandleID,
		NodeID:           createResp.Entry.NodeID,
		Writable:         true,
		DeleteOnClose:    true,
	}})
	if err != nil {
		t.Fatalf("RecoverHandles() error = %v", err)
	}
	if len(recovered.Handles) != 1 || recovered.Handles[0].Error != "" {
		t.Fatalf("recovered handles = %+v", recovered.Handles)
	}
	if !recovered.Handles[0].DeleteOnClose {
		t.Fatalf("expected recovered handle to keep delete-on-close")
	}

	handleID := recovered.Handles[0].HandleID
	if _, err := cli2.Write(handleID, 0, []byte("after-recovery")); err != nil {
		t.Fatalf("Write(after-recovery) error = %v", err)
	}
	if _, err := cli2.CloseHandle(handleID); err != nil {
		t.Fatalf("CloseHandle(recovered) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "recover-write.txt")); !os.IsNotExist(err) {
		t.Fatalf("recover-write.txt should be deleted on close, stat err = %v", err)
	}
}

func TestRecoveryRevalidateAfterRenameAndUnknownResubscribe(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "before.txt"), []byte("rename-me"), 0o644); err != nil {
		t.Fatalf("WriteFile(before.txt) error = %v", err)
	}
	now := time.Date(2026, 3, 16, 13, 0, 0, 0, time.UTC)
	cli1, _ := newRecoveryEnv(t, root, 64, now, "recover-rename-client")

	beforeNode, err := cli1.Lookup(cli1.RootNodeID, "before.txt")
	if err != nil {
		t.Fatalf("Lookup(before.txt) error = %v", err)
	}
	if _, err := cli1.Rename(cli1.RootNodeID, "before.txt", cli1.RootNodeID, "after.txt", false); err != nil {
		t.Fatalf("Rename(before->after) error = %v", err)
	}

	sessionID := cli1.SessionID
	addr := cli1.Addr
	if err := cli1.Close(); err != nil {
		t.Fatalf("Close(cli1) error = %v", err)
	}

	cli2 := reconnectAndResume(t, addr, sessionID, "recover-rename-client")
	revalidated, err := cli2.RevalidateNodes([]uint64{beforeNode.Entry.NodeID})
	if err != nil {
		t.Fatalf("RevalidateNodes() error = %v", err)
	}
	if len(revalidated.Entries) != 1 {
		t.Fatalf("expected 1 revalidate entry, got %d", len(revalidated.Entries))
	}
	if !revalidated.Entries[0].Exists {
		t.Fatalf("expected renamed node to still exist")
	}
	if revalidated.Entries[0].Entry.Name != "after.txt" {
		t.Fatalf("revalidated entry name = %q, want %q", revalidated.Entries[0].Entry.Name, "after.txt")
	}

	resub, err := cli2.ResubscribeWatches([]protocol.ResubscribeSpec{{
		PreviousWatchID: 999999,
		NodeID:          cli2.RootNodeID,
		Recursive:       true,
		AfterSeq:        7,
	}})
	if err != nil {
		t.Fatalf("ResubscribeWatches() error = %v", err)
	}
	if len(resub.Watches) != 1 {
		t.Fatalf("expected 1 resubscribe result, got %d", len(resub.Watches))
	}
	if resub.Watches[0].Error != "" || resub.Watches[0].WatchID == 0 {
		t.Fatalf("resubscribe result = %+v", resub.Watches[0])
	}
	if resub.Watches[0].StartSeq != 7 || resub.Watches[0].AckedSeq != 7 {
		t.Fatalf("resubscribe result = %+v, want start/ack 7", resub.Watches[0])
	}
}
