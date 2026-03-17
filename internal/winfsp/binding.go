package winfsp

import "fmt"

type BindingInfo struct {
	Backend        string
	Available      bool
	DLLPath        string
	LauncherPath   string
	MountPoint     string
	PreflightOK    bool
	PreflightError string
	Note           string
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
	if b.Backend == "" {
		return status
	}
	return fmt.Sprintf("%s (%s)", b.Backend, status)
}

func Probe(config HostConfig) (BindingInfo, error) {
	return probeBinding(config)
}
