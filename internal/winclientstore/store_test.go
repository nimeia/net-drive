package winclientstore

import (
	"path/filepath"
	"testing"

	"developer-mount/internal/winclient"
)

func TestSaveWorkspaceRoundTrip(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "win32-client.json"))
	state, err := store.SaveWorkspace("dev", winclient.Config{Addr: "127.0.0.1:17890", Path: "/"}, WorkspaceMeta{DisplayName: "Engineering"})
	if err != nil {
		t.Fatalf("SaveWorkspace() error = %v", err)
	}
	if state.Settings.DefaultWorkspace != "dev" {
		t.Fatalf("DefaultWorkspace = %q", state.Settings.DefaultWorkspace)
	}
}
