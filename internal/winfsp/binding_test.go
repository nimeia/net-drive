package winfsp

import "testing"

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
