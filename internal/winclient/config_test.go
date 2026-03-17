package winclient

import (
	"strings"
	"testing"
)

func TestConfigNormalizedAppliesDefaultsAndLeadingSlash(t *testing.T) {
	cfg := Config{Path: "README.md"}.Normalized()
	if cfg.Addr != "127.0.0.1:17890" {
		t.Fatalf("Addr = %q, want default", cfg.Addr)
	}
	if cfg.Path != "/README.md" {
		t.Fatalf("Path = %q, want /README.md", cfg.Path)
	}
	if cfg.Length == 0 || cfg.MaxEntries == 0 || cfg.LeaseSeconds == 0 {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
}

func TestConfigValidateRejectsUnsupportedOperation(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(Operation("mount")); err == nil {
		t.Fatal("Validate unsupported operation error = nil, want error")
	}
}

func TestConfigValidateNormalizesRelativePathBeforeValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Path = "relative/path"
	if err := cfg.Validate(OperationRead); err != nil {
		t.Fatalf("Validate relative path error = %v, want nil after normalization", err)
	}
	if got := cfg.Normalized().Path; got != "/relative/path" {
		t.Fatalf("Normalized path = %q, want /relative/path", got)
	}
}

func TestBuildCLIPreviewIncludesOperationSpecificFlags(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Path = "/docs/My File.txt"
	cfg.Length = 128
	cfg.MaxEntries = 9

	readCmd := BuildCLIPreview(cfg, OperationRead)
	if !strings.Contains(readCmd, `-op read`) {
		t.Fatalf("read preview missing op: %s", readCmd)
	}
	if !strings.Contains(readCmd, `-length 128`) {
		t.Fatalf("read preview missing length: %s", readCmd)
	}
	if !strings.Contains(readCmd, `"/docs/My File.txt"`) {
		t.Fatalf("read preview missing quoted path: %s", readCmd)
	}

	readdirCmd := BuildCLIPreview(cfg, OperationReadDir)
	if !strings.Contains(readdirCmd, `-max-entries 9`) {
		t.Fatalf("readdir preview missing max-entries: %s", readdirCmd)
	}
	if strings.Contains(readdirCmd, `-length 128`) {
		t.Fatalf("readdir preview should not include read length: %s", readdirCmd)
	}
}
