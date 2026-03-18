//go:build windows

package winfsp

import (
	"context"
	"fmt"
	"syscall"
)

type dispatcherBindings struct {
	dll              *syscall.DLL
	create           *syscall.Proc
	setMountPoint    *syscall.Proc
	removeMountPoint *syscall.Proc
	startDispatcher  *syscall.Proc
	stopDispatcher   *syscall.Proc
	deleteFS         *syscall.Proc
	addDirInfo       *syscall.Proc
}

func probeDispatcherBindings(dllPath string) (dispatcherBindings, error) {
	dll, err := syscall.LoadDLL(dllPath)
	if err != nil {
		return dispatcherBindings{}, fmt.Errorf("load %s: %w", dllPath, err)
	}
	load := func(name string) (*syscall.Proc, error) {
		proc, err := dll.FindProc(name)
		if err != nil {
			return nil, fmt.Errorf("find %s: %w", name, err)
		}
		return proc, nil
	}
	create, err := load("FspFileSystemCreate")
	if err != nil {
		return dispatcherBindings{}, err
	}
	setMountPoint, err := load("FspFileSystemSetMountPoint")
	if err != nil {
		return dispatcherBindings{}, err
	}
	removeMountPoint, err := load("FspFileSystemRemoveMountPoint")
	if err != nil {
		return dispatcherBindings{}, err
	}
	startDispatcher, err := load("FspFileSystemStartDispatcher")
	if err != nil {
		return dispatcherBindings{}, err
	}
	stopDispatcher, err := load("FspFileSystemStopDispatcher")
	if err != nil {
		return dispatcherBindings{}, err
	}
	deleteFS, err := load("FspFileSystemDelete")
	if err != nil {
		return dispatcherBindings{}, err
	}
	addDirInfo, err := load("FspFileSystemAddDirInfo")
	if err != nil {
		return dispatcherBindings{}, err
	}
	return dispatcherBindings{dll: dll, create: create, setMountPoint: setMountPoint, removeMountPoint: removeMountPoint, startDispatcher: startDispatcher, stopDispatcher: stopDispatcher, deleteFS: deleteFS, addDirInfo: addDirInfo}, nil
}
func runDispatcherHostV1(ctx context.Context, h *Host) error {
	if !h.binding.DispatcherReady {
		return fmt.Errorf("dispatcher-v1 requested but dispatcher APIs are not ready")
	}
	bridge := h.DispatcherBridge()
	abi := h.DispatcherABI()
	service := h.DispatcherService()
	if bridge == nil || abi == nil || service == nil {
		return fmt.Errorf("dispatcher-v1 requested but dispatcher bridge/ABI/service loop is not initialized")
	}
	native, err := newNativeFileSystem(h.binding.DLLPath, h.config, h.callbacks)
	if err != nil {
		return err
	}
	defer native.Close()
	if err := service.Start("/"); err != nil {
		binding := h.Binding()
		binding.DispatcherStatus = bridge.Snapshot().Summary()
		binding.CallbackBridgeStatus = abi.Snapshot().Summary()
		binding.ServiceLoopStatus = service.Snapshot().Summary()
		h.SetBinding(binding)
		return err
	}
	if err := native.Start(); err != nil {
		service.Stop()
		return err
	}
	binding := h.Binding()
	binding.DispatcherStatus = bridge.Snapshot().Summary()
	binding.CallbackBridgeReady = true
	binding.CallbackBridgeStatus = abi.Snapshot().Summary()
	binding.ServiceLoopReady = true
	binding.ServiceLoopStatus = service.Snapshot().Summary()
	h.SetBinding(binding)
	<-ctx.Done()
	native.Stop()
	service.Stop()
	binding = h.Binding()
	binding.CallbackBridgeStatus = abi.Snapshot().Summary()
	binding.ServiceLoopStatus = service.Snapshot().Summary() + " stopped"
	h.SetBinding(binding)
	return nil
}
