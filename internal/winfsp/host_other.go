//go:build !windows

package winfsp

import (
	"context"
	"fmt"
)

func runHost(ctx context.Context, h *Host) error {
	binding, _ := Probe(h.config)
	h.binding = binding
	return fmt.Errorf("winfsp host is only supported on Windows")
}
