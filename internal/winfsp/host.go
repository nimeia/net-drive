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
	return populateBindingDerived(h.binding, h.dispatcher, h.abi, h.service)
}
func (h *Host) SetBinding(binding BindingInfo) {
	h.binding = populateBindingDerived(binding, h.dispatcher, h.abi, h.service)
}

func populateBindingDerived(binding BindingInfo, dispatcher *DispatcherBridge, abi *DispatcherABI, service *DispatcherService) BindingInfo {
	if dispatcher != nil && binding.EffectiveBackend == "winfsp-dispatcher-v1" {
		if binding.DispatcherStatus == "" {
			binding.DispatcherStatus = dispatcher.Snapshot().Summary()
		}
		if abi != nil && binding.CallbackBridgeStatus == "" {
			binding.CallbackBridgeStatus = abi.Snapshot().Summary()
		}
		if service != nil && binding.ServiceLoopStatus == "" {
			binding.ServiceLoopStatus = service.Snapshot().Summary()
		}
	}
	binding.NativeCallbackSummary = DefaultNativeCallbackTable(binding).Summary()
	return binding
}
func (h *Host) Run(ctx context.Context) error { return runHost(ctx, h) }
