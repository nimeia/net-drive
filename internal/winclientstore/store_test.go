package winclientstore

import (
	"path/filepath"
	"testing"

	"developer-mount/internal/winclient"
)

func TestLoadMissingStoreReturnsDefaultState(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "missing.json"))
	state, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if state.Version != SchemaVersion {
		t.Fatalf("Version = %d, want %d", state.Version, SchemaVersion)
	}
	if len(state.Profiles) != 0 {
		t.Fatalf("Profiles length = %d, want 0", len(state.Profiles))
	}
}

func TestSaveProfileRoundTripAndNormalize(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "win32-client.json"))
	cfg := winclient.Config{Addr: "127.0.0.1:17890", Path: "README.md"}
	state, err := store.SaveProfile("dev", cfg)
	if err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}
	if state.ActiveProfile != "dev" {
		t.Fatalf("ActiveProfile = %q, want dev", state.ActiveProfile)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := loaded.Profiles["dev"].Path; got != "/README.md" {
		t.Fatalf("loaded.Profiles[dev].Path = %q, want /README.md", got)
	}
	if got := loaded.Profiles["dev"].Token; got == "" {
		t.Fatal("expected default token to be applied during normalization")
	}
}

func TestDeleteProfilePromotesNextSortedActiveProfile(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "win32-client.json"))
	if _, err := store.SaveProfile("beta", winclient.DefaultConfig()); err != nil {
		t.Fatalf("SaveProfile(beta) error = %v", err)
	}
	if _, err := store.SaveProfile("alpha", winclient.DefaultConfig()); err != nil {
		t.Fatalf("SaveProfile(alpha) error = %v", err)
	}
	state, err := store.DeleteProfile("alpha")
	if err != nil {
		t.Fatalf("DeleteProfile(alpha) error = %v", err)
	}
	if state.ActiveProfile != "beta" {
		t.Fatalf("ActiveProfile = %q, want beta", state.ActiveProfile)
	}
	if _, ok := state.Profiles["alpha"]; ok {
		t.Fatal("alpha profile still present after delete")
	}
}

func TestDefaultPathRespectsOverride(t *testing.T) {
	t.Setenv(overrideEnvName, filepath.Join(t.TempDir(), "custom.json"))
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}
	if filepath.Base(path) != "custom.json" {
		t.Fatalf("DefaultPath() = %q, want override path", path)
	}
}
