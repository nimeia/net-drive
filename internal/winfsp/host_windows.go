//go:build windows

package winfsp

import (
	"context"
	"fmt"
)

func runHost(ctx context.Context, h *Host) error {
	if h.config.MountPoint == "" {
		return fmt.Errorf("winfsp mount point is required")
	}
	binding, err := Probe(h.config)
	h.binding = binding
	if err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}
