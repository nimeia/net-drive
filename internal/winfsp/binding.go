package winfsp

import (
	"fmt"
	"strings"
)

type BindingInfo struct {
	RequestedBackend      string
	Backend               string
	EffectiveBackend      string
	Available             bool
	DLLPath               string
	LauncherPath          string
	MountPoint            string
	PreflightOK           bool
	PreflightError        string
	DispatcherReady       bool
	DispatcherStatus      string
	CallbackBridgeReady   bool
	CallbackBridgeStatus  string
	ServiceLoopReady      bool
	ServiceLoopStatus     string
	NativeCallbackSummary string
	ExplorerRequestMatrix string
	Note                  string
}

func (b BindingInfo) Summary() string {
	status := "unavailable"
	if b.Available {
		status = "available"
	}
	if b.PreflightOK {
		status = "ready"
	}
	if b.PreflightError != "" {
		status = fmt.Sprintf("error: %s", b.PreflightError)
	}
	backend := b.EffectiveBackend
	if backend == "" {
		backend = b.Backend
	}
	if backend == "" {
		return status
	}
	parts := []string{status}
	if b.DispatcherStatus != "" {
		parts = append(parts, b.DispatcherStatus)
	}
	if b.CallbackBridgeStatus != "" {
		parts = append(parts, b.CallbackBridgeStatus)
	}
	if b.ServiceLoopStatus != "" {
		parts = append(parts, b.ServiceLoopStatus)
	}
	return fmt.Sprintf("%s (%s)", backend, strings.Join(parts, "; "))
}

func (b BindingInfo) MountRuntimeSupportError() error {
	switch b.EffectiveBackend {
	case "winfsp-native-preflight":
		return fmt.Errorf("backend %s validates WinFsp availability only and does not create an Explorer-visible mount point yet", b.EffectiveBackend)
	default:
		return nil
	}
}

func Probe(config HostConfig) (BindingInfo, error) { return probeBinding(config) }
