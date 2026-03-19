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
	if cfg.LocalPath != "devmount-local" {
		t.Fatalf("LocalPath = %q, want default", cfg.LocalPath)
	}
	if cfg.HostBackend != HostBackendAuto {
		t.Fatalf("HostBackend = %q, want auto", cfg.HostBackend)
	}
	if cfg.MountPoint == "" {
		t.Fatalf("MountPoint = empty, want default drive letter")
	}
	if cfg.Length == 0 || cfg.MaxEntries == 0 || cfg.LeaseSeconds == 0 {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
}

func TestDefaultMountPointPicksFirstFreeCandidate(t *testing.T) {
	original := driveRootExists
	driveRootExists = func(root string) bool {
		return root == `Z:\` || root == `Y:\`
	}
	defer func() { driveRootExists = original }()

	if got := defaultMountPoint(); got != "X:" {
		t.Fatalf("defaultMountPoint() = %q, want X:", got)
	}
}

func TestDefaultMountPointFallsBackWhenCandidatesBusy(t *testing.T) {
	original := driveRootExists
	driveRootExists = func(root string) bool { return true }
	defer func() { driveRootExists = original }()

	if got := defaultMountPoint(); got != "M:" {
		t.Fatalf("defaultMountPoint() = %q, want M:", got)
	}
}

func TestResolveMountPointForStartKeepsFreeDrive(t *testing.T) {
	original := driveRootExists
	driveRootExists = func(root string) bool { return false }
	defer func() { driveRootExists = original }()

	got, changed := ResolveMountPointForStart("M:")
	if changed {
		t.Fatal("ResolveMountPointForStart(M:) changed = true, want false")
	}
	if got != "M:" {
		t.Fatalf("ResolveMountPointForStart(M:) = %q, want M:", got)
	}
}

func TestResolveMountPointForStartSwitchesBusyDrive(t *testing.T) {
	original := driveRootExists
	driveRootExists = func(root string) bool {
		return root == `M:\` || root == `L:\`
	}
	defer func() { driveRootExists = original }()

	got, changed := ResolveMountPointForStart("M:")
	if !changed {
		t.Fatal("ResolveMountPointForStart(M:) changed = false, want true")
	}
	if got != "K:" {
		t.Fatalf("ResolveMountPointForStart(M:) = %q, want K:", got)
	}
}

func TestResolveMountPointForStartLeavesDirectoryMountPoint(t *testing.T) {
	got, changed := ResolveMountPointForStart(`C:\mnt\devmount`)
	if changed {
		t.Fatal("ResolveMountPointForStart(directory) changed = true, want false")
	}
	if got != `C:\mnt\devmount` {
		t.Fatalf("ResolveMountPointForStart(directory) = %q", got)
	}
}

func TestConfigValidateRejectsUnsupportedOperation(t *testing.T) {
	if err := DefaultConfig().Validate(Operation("unsupported")); err == nil {
		t.Fatal("Validate unsupported operation error = nil, want error")
	}
}
func TestConfigValidateAllowsMountOperation(t *testing.T) {
	if err := DefaultConfig().Validate(OperationMount); err != nil {
		t.Fatalf("Validate mount error = %v, want nil", err)
	}
}

func TestNormalizeMountPointCanonicalizesDriveRoots(t *testing.T) {
	if got := NormalizeMountPoint("m:\\"); got != "M:" {
		t.Fatalf("NormalizeMountPoint(m:\\\\) = %q, want M:", got)
	}
	if got := NormalizeMountPoint("m:\\\\"); got != "M:" {
		t.Fatalf("NormalizeMountPoint(m:\\\\\\\\) = %q, want M:", got)
	}
	if got := NormalizeMountPoint(" m: "); got != "M:" {
		t.Fatalf("NormalizeMountPoint(\" m: \") = %q, want M:", got)
	}
}

func TestConfigValidateRejectsInvalidMountPoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MountPoint = "bad:mount"
	if err := cfg.Validate(OperationMount); err == nil {
		t.Fatal("Validate invalid mount point error = nil, want error")
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
func TestConfigValidateRejectsUnsupportedHostBackend(t *testing.T) {
	cfg := DefaultConfig()
	cfg.HostBackend = "boom"
	if err := cfg.Validate(OperationMount); err == nil {
		t.Fatal("Validate unsupported host backend error = nil, want error")
	}
}
func TestBuildCLIPreviewIncludesOperationSpecificFlags(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Path = "/docs/My File.txt"
	cfg.MountPoint = "m:\\"
	cfg.Length = 128
	cfg.MaxEntries = 9
	cfg.LocalPath = `C:\temp\dev mount`
	cfg.HostBackend = HostBackendDispatcherV1
	mountCmd := BuildCLIPreview(cfg, OperationMount)
	if !strings.Contains(mountCmd, `-op mount`) {
		t.Fatalf("mount preview missing op: %s", mountCmd)
	}
	if !strings.Contains(mountCmd, `-host-backend dispatcher-v1`) {
		t.Fatalf("mount preview missing host backend: %s", mountCmd)
	}
	if !strings.Contains(mountCmd, `-mount-point M:`) {
		t.Fatalf("mount preview missing normalized mount point: %s", mountCmd)
	}
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
	materializeCmd := BuildCLIPreview(cfg, OperationMaterialize)
	if !strings.Contains(materializeCmd, `-op materialize`) {
		t.Fatalf("materialize preview missing op: %s", materializeCmd)
	}
	if !strings.Contains(materializeCmd, `-local-path "C:\temp\dev mount"`) {
		t.Fatalf("materialize preview missing local path: %s", materializeCmd)
	}
	if !strings.Contains(materializeCmd, `-max-entries 9`) {
		t.Fatalf("materialize preview missing max-entries: %s", materializeCmd)
	}
}
