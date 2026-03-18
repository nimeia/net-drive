//go:build !windows

package winfsp

import (
	"context"
	"fmt"
)

type dispatcherBindings struct{}

func probeDispatcherBindings(dllPath string) (dispatcherBindings, error) {
	return dispatcherBindings{}, fmt.Errorf("dispatcher bindings are only available on Windows")
}

func runDispatcherHostV1(ctx context.Context, h *Host) error {
	_ = ctx
	_ = h
	return fmt.Errorf("dispatcher-v1 host is only supported on Windows")
}
