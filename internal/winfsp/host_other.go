//go:build !windows

package winfsp

import (
	"context"
	"fmt"
)

func runHost(ctx context.Context, h *Host) error {
	_ = ctx
	_ = h
	return fmt.Errorf("winfsp host is only available on windows")
}
