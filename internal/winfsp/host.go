package winfsp

import (
	"context"
	adapterpkg "developer-mount/internal/winfsp/adapter"
)

type HostConfig struct {
	MountPoint, VolumePrefix, Backend string
	DebugLog                          bool
}

type Host struct {
	config     HostConfig
	callbacks  *Callbacks
	dispatcher *DispatcherBridge
	abi        *DispatcherABI
	service    *DispatcherService
	binding    BindingInfo
}

func NewHost(config HostConfig, adapter *adapterpkg.Adapter) *Host {
	callbacks := NewCallbacks(adapter)
	host := &Host{config: config, callbacks: callbacks}
	if normalizeRequestedBackend(config.Backend) == "dispatcher-v1" || normalizeRequestedBackend(config.Backend) == "auto" {
		host.dispatcher = NewDispatcherBridge(callbacks)
		host.abi = NewDispatcherABI(host.dispatcher)
		host.service = NewDispatcherService(dispatcherBindings{}, config.MountPoint, host.abi)
	}
	return host
}
func (h *Host) Config() HostConfig                    { return h.config }
func (h *Host) Callbacks() *Callbacks                 { return h.callbacks }
func (h *Host) DispatcherBridge() *DispatcherBridge   { return h.dispatcher }
func (h *Host) DispatcherABI() *DispatcherABI         { return h.abi }
func (h *Host) DispatcherService() *DispatcherService { return h.service }
func (h *Host) Binding() BindingInfo {
	binding := h.binding
	if h.dispatcher != nil && binding.EffectiveBackend == "winfsp-dispatcher-v1" {
		if binding.DispatcherStatus == "" {
			binding.DispatcherStatus = h.dispatcher.Snapshot().Summary()
		}
		if h.abi != nil && binding.CallbackBridgeStatus == "" {
			binding.CallbackBridgeStatus = h.abi.Snapshot().Summary()
		}
		if h.service != nil && binding.ServiceLoopStatus == "" {
			binding.ServiceLoopStatus = h.service.Snapshot().Summary()
		}
	}
	return binding
}
func (h *Host) SetBinding(binding BindingInfo) {
	if h.dispatcher != nil && binding.EffectiveBackend == "winfsp-dispatcher-v1" {
		if binding.DispatcherStatus == "" {
			binding.DispatcherStatus = h.dispatcher.Snapshot().Summary()
		}
		if h.abi != nil && binding.CallbackBridgeStatus == "" {
			binding.CallbackBridgeStatus = h.abi.Snapshot().Summary()
		}
		if h.service != nil && binding.ServiceLoopStatus == "" {
			binding.ServiceLoopStatus = h.service.Snapshot().Summary()
		}
	}
	h.binding = binding
}
func (h *Host) Run(ctx context.Context) error { return runHost(ctx, h) }
