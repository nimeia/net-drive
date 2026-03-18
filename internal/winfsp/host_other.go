//go:build !windows

package winfsp

import (
	"context"
	"fmt"
)

func runHost(ctx context.Context, h *Host) error {
	binding, _ := Probe(h.config)
	h.binding = populateBindingDerived(binding, h.dispatcher, h.abi, h.service)
	return fmt.Errorf("winfsp host is only supported on Windows")
}
