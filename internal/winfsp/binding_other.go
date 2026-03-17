//go:build !windows

package winfsp

func probeBinding(config HostConfig) (BindingInfo, error) {
	return BindingInfo{
		Backend:        "winfsp-unavailable",
		Available:      false,
		MountPoint:     config.MountPoint,
		PreflightOK:    false,
		PreflightError: "WinFsp binding is available on Windows only",
		Note:           "Build and run on Windows with WinFsp installed to exercise the real host binding.",
	}, nil
}
