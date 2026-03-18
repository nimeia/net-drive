package winfsp

import (
	"fmt"
	"sync"
)

type DispatcherServiceState struct {
	Created, Running          bool
	MountPoint                string
	StartCount, StopCount     uint64
	LastOperation             string
	LastNTStatus              NTStatus
	LastError, CallbackBridge string
}

func (s DispatcherServiceState) Summary() string {
	summary := fmt.Sprintf("created=%v running=%v mount=%s starts=%d stops=%d", s.Created, s.Running, defaultDispatcherValue(s.MountPoint, "-"), s.StartCount, s.StopCount)
	if s.CallbackBridge != "" {
		summary += " callback=" + s.CallbackBridge
	}
	if s.LastOperation != "" {
		summary += " last_op=" + s.LastOperation
	}
	if s.LastError != "" {
		summary += " last_error=" + s.LastError
	}
	if s.LastNTStatus != 0 {
		summary += fmt.Sprintf(" ntstatus=0x%08x", uint32(s.LastNTStatus))
	}
	return summary
}

type DispatcherService struct {
	mu         sync.Mutex
	bindings   dispatcherBindings
	mountPoint string
	abi        *DispatcherABI
	state      DispatcherServiceState
}

func NewDispatcherService(bindings dispatcherBindings, mountPoint string, abi *DispatcherABI) *DispatcherService {
	return &DispatcherService{bindings: bindings, mountPoint: mountPoint, abi: abi, state: DispatcherServiceState{MountPoint: mountPoint}}
}
func (s *DispatcherService) Snapshot() DispatcherServiceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}
func (s *DispatcherService) Start(rootPath string) error {
	if s.abi == nil {
		return fmt.Errorf("dispatcher service ABI bridge is not initialized")
	}
	if err := s.abi.Initialize(rootPath); err != nil {
		s.record("start", StatusInternalError, err.Error())
		return err
	}
	if _, status := s.abi.GetVolumeInfo(); status != StatusSuccess {
		err := StatusError(status, fmt.Errorf("dispatcher service volume warmup failed"))
		s.record("warmup-volume", status, err.Error())
		return err
	}
	if _, status := s.abi.GetFileInfo(rootPath); status != StatusSuccess {
		err := StatusError(status, fmt.Errorf("dispatcher service root getattr warmup failed"))
		s.record("warmup-root", status, err.Error())
		return err
	}
	if handleID, _, status := s.abi.OpenDirectory(rootPath); status == StatusSuccess {
		_, _, _, _ = s.abi.ReadDirectory(handleID, 0, 32)
		_ = s.abi.Close(handleID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Created = true
	s.state.Running = true
	s.state.StartCount++
	s.state.LastOperation = "start"
	s.state.LastNTStatus = StatusSuccess
	s.state.LastError = ""
	s.state.CallbackBridge = s.abi.Snapshot().Summary()
	return nil
}
func (s *DispatcherService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Running = false
	s.state.StopCount++
	s.state.LastOperation = "stop"
	s.state.LastNTStatus = StatusSuccess
	s.state.CallbackBridge = s.abi.Snapshot().Summary()
}
func (s *DispatcherService) record(op string, status NTStatus, detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.LastOperation = op
	s.state.LastNTStatus = status
	s.state.LastError = detail
	if s.abi != nil {
		s.state.CallbackBridge = s.abi.Snapshot().Summary()
	}
}
