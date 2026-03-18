package winfsp

import (
	"context"
	adapterpkg "developer-mount/internal/winfsp/adapter"
)

type HostConfig struct {
	MountPoint   string
	VolumePrefix string
	Backend      string
	DebugLog     bool
}

type Host struct {
	config     HostConfig
	callbacks  *Callbacks
	dispatcher *DispatcherBridge
	binding    BindingInfo
}

func NewHost(config HostConfig, adapter *adapterpkg.Adapter) *Host {
	callbacks := NewCallbacks(adapter)
	host := &Host{config: config, callbacks: callbacks}
	if normalizeRequestedBackend(config.Backend) == "dispatcher-v1" || normalizeRequestedBackend(config.Backend) == "auto" {
		host.dispatcher = NewDispatcherBridge(callbacks)
	}
	return host
}
func (h *Host) Config() HostConfig    { return h.config }
func (h *Host) Callbacks() *Callbacks { return h.callbacks }
func (h *Host) Binding() BindingInfo {
	binding := h.binding
	if h.dispatcher != nil && binding.EffectiveBackend == "winfsp-dispatcher-v1" {
		state := h.dispatcher.Snapshot()
		if binding.DispatcherStatus == "" {
			binding.DispatcherStatus = state.Summary()
		}
	}
	return binding
}
func (h *Host) SetBinding(binding BindingInfo) {
	if h.dispatcher != nil && binding.EffectiveBackend == "winfsp-dispatcher-v1" && binding.DispatcherStatus == "" {
		binding.DispatcherStatus = h.dispatcher.Snapshot().Summary()
	}
	h.binding = binding
}
func (h *Host) DispatcherBridge() *DispatcherBridge { return h.dispatcher }
func (h *Host) Run(ctx context.Context) error       { return runHost(ctx, h) }
