package winfsp

import "testing"

func TestBindingInfoSummary(t *testing.T) {
	if got := (BindingInfo{Backend: "winfsp-native-preflight", PreflightOK: true}).Summary(); got != "winfsp-native-preflight (ready)" {
		t.Fatalf("Summary ready = %q", got)
	}
	if got := (BindingInfo{Backend: "winfsp-native-preflight", Available: true}).Summary(); got != "winfsp-native-preflight (available)" {
		t.Fatalf("Summary available = %q", got)
	}
	if got := (BindingInfo{Backend: "winfsp-native-preflight", PreflightError: "boom"}).Summary(); got != "winfsp-native-preflight (error: boom)" {
		t.Fatalf("Summary error = %q", got)
	}
}
