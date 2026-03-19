package winfsp

import (
	"strings"
	"testing"
)

func TestBindingInfoSummary(t *testing.T) {
	if got := (BindingInfo{EffectiveBackend: "winfsp-native-preflight", PreflightOK: true}).Summary(); got != "winfsp-native-preflight (ready)" {
		t.Fatalf("Summary ready = %q", got)
	}
	if got := (BindingInfo{EffectiveBackend: "winfsp-native-preflight", Available: true}).Summary(); got != "winfsp-native-preflight (available)" {
		t.Fatalf("Summary available = %q", got)
	}
	if got := (BindingInfo{EffectiveBackend: "winfsp-native-preflight", PreflightError: "boom"}).Summary(); got != "winfsp-native-preflight (error: boom)" {
		t.Fatalf("Summary error = %q", got)
	}
	got := (BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1", DispatcherStatus: "dispatcher APIs ready", CallbackBridgeStatus: "callback bridge scaffold ready", ServiceLoopStatus: "dispatcher service loop scaffold ready"}).Summary()
	want := "winfsp-dispatcher-v1 (unavailable; dispatcher APIs ready; callback bridge scaffold ready; dispatcher service loop scaffold ready)"
	if got != want {
		t.Fatalf("Summary dispatcher = %q, want %q", got, want)
	}
}

func TestNTStatusHintIncludesMountPointGuidance(t *testing.T) {
	got := ntStatusHint(NTStatus(0xc0000033), "M:")
	if !strings.Contains(got, "STATUS_OBJECT_NAME_INVALID") {
		t.Fatalf("ntStatusHint missing status guidance: %q", got)
	}
	if !strings.Contains(got, `M:`) {
		t.Fatalf("ntStatusHint missing mount point: %q", got)
	}
}

func TestShouldSkipNativePreflightForDriveLetters(t *testing.T) {
	if !shouldSkipNativePreflight("Z:") {
		t.Fatal("shouldSkipNativePreflight(Z:) = false, want true")
	}
	if shouldSkipNativePreflight(`C:\mnt\devmount`) {
		t.Fatal("shouldSkipNativePreflight(directory) = true, want false")
	}
}

func TestBindingMountRuntimeSupportError(t *testing.T) {
	if err := (BindingInfo{EffectiveBackend: "winfsp-native-preflight"}).MountRuntimeSupportError(); err == nil {
		t.Fatal("preflight MountRuntimeSupportError = nil, want error")
	}
	if err := (BindingInfo{EffectiveBackend: "winfsp-dispatcher-v1"}).MountRuntimeSupportError(); err != nil {
		t.Fatalf("dispatcher MountRuntimeSupportError = %v, want nil", err)
	}
	if err := (BindingInfo{EffectiveBackend: ""}).MountRuntimeSupportError(); err != nil {
		t.Fatalf("empty backend MountRuntimeSupportError = %v, want nil", err)
	}
}
