package winfsp

import (
	"context"
	adapterpkg "developer-mount/internal/winfsp/adapter"
)

type HostConfig struct {
	MountPoint   string
	VolumePrefix string
	DebugLog     bool
}
type Host struct {
	config    HostConfig
	callbacks *Callbacks
	binding   BindingInfo
}

func NewHost(config HostConfig, adapter *adapterpkg.Adapter) *Host {
	return &Host{config: config, callbacks: NewCallbacks(adapter)}
}
func (h *Host) Config() HostConfig            { return h.config }
func (h *Host) Callbacks() *Callbacks         { return h.callbacks }
func (h *Host) Binding() BindingInfo          { return h.binding }
func (h *Host) Run(ctx context.Context) error { return runHost(ctx, h) }
