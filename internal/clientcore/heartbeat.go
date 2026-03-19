package clientcore

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func (c *Client) RunHeartbeatLoop(ctx context.Context) error {
	interval := heartbeatInterval(c.SnapshotState().LeaseSeconds)
	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
		if _, err := c.Heartbeat(); err != nil {
			return err
		}
		timer.Reset(heartbeatInterval(c.SnapshotState().LeaseSeconds))
	}
}

func (c *Client) RunWithHeartbeat(ctx context.Context, run func(context.Context) error) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	runCh := make(chan error, 1)
	hbCh := make(chan error, 1)
	go func() { runCh <- run(runCtx) }()
	go func() { hbCh <- c.RunHeartbeatLoop(runCtx) }()

	var runErr error
	var hbErr error
	for runCh != nil || hbCh != nil {
		select {
		case err := <-runCh:
			runErr = err
			runCh = nil
			cancel()
		case err := <-hbCh:
			hbErr = err
			hbCh = nil
			if err != nil && !errors.Is(err, context.Canceled) {
				cancel()
			}
		}
	}

	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		return runErr
	}
	if hbErr != nil && !errors.Is(hbErr, context.Canceled) {
		return fmt.Errorf("session heartbeat failed: %w", hbErr)
	}
	return nil
}

func heartbeatInterval(leaseSeconds uint32) time.Duration {
	if leaseSeconds == 0 {
		leaseSeconds = 30
	}
	interval := time.Duration(leaseSeconds) * time.Second / 3
	if interval < 5*time.Second {
		return 5 * time.Second
	}
	return interval
}
