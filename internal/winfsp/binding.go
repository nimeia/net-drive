package winfsp

import "fmt"

type BindingInfo struct {
	RequestedBackend string
	Backend          string
	EffectiveBackend string
	Available        bool
	DLLPath          string
	LauncherPath     string
	MountPoint       string
	PreflightOK      bool
	PreflightError   string
	DispatcherReady  bool
	DispatcherStatus string
	Note             string
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
	if b.DispatcherStatus != "" {
		return fmt.Sprintf("%s (%s; %s)", backend, status, b.DispatcherStatus)
	}
	return fmt.Sprintf("%s (%s)", backend, status)
}

func Probe(config HostConfig) (BindingInfo, error) { return probeBinding(config) }
