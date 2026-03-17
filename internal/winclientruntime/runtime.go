package winclientruntime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"developer-mount/internal/clientcore"
	"developer-mount/internal/mountcore"
	"developer-mount/internal/winclient"
	"developer-mount/internal/winfsp"
	adapterpkg "developer-mount/internal/winfsp/adapter"
)

type Phase string

const (
	PhaseIdle       Phase = "idle"
	PhaseConnecting Phase = "connecting"
	PhaseMounted    Phase = "mounted"
	PhaseStopping   Phase = "stopping"
	PhaseError      Phase = "error"
)

type Snapshot struct {
	Phase             Phase
	StatusText        string
	LastError         string
	ActiveProfile     string
	ServerAddr        string
	MountPoint        string
	VolumePrefix      string
	RemotePath        string
	ClientInstanceID  string
	SessionID         uint64
	PrincipalID       string
	ServerName        string
	ServerVersion     string
	ExpiresAt         string
	HostBackend       string
	HostDLLPath       string
	HostLauncherPath  string
	HostBindingStatus string
	StartedAt         time.Time
	UpdatedAt         time.Time
}

type SessionInfo struct {
	ServerAddr        string
	MountPoint        string
	VolumePrefix      string
	RemotePath        string
	ClientInstanceID  string
	SessionID         uint64
	PrincipalID       string
	ServerName        string
	ServerVersion     string
	ExpiresAt         string
	HostBackend       string
	HostDLLPath       string
	HostLauncherPath  string
	HostBindingStatus string
}

type Session interface {
	Info() SessionInfo
	Run(ctx context.Context) error
}

type Builder interface {
	Build(config winclient.Config) (Session, error)
}

type Host interface {
	Config() winfsp.HostConfig
	Run(ctx context.Context) error
}

type HostFactory func(config winfsp.HostConfig, adapter *adapterpkg.Adapter) Host

type Runtime struct {
	mu      sync.RWMutex
	builder Builder
	now     func() time.Time
	cancel  context.CancelFunc
	state   Snapshot
}

func New(builder Builder) *Runtime {
	return NewWithClock(builder, time.Now)
}

func NewWithClock(builder Builder, now func() time.Time) *Runtime {
	if builder == nil {
		builder = NewDefaultBuilder(nil)
	}
	if now == nil {
		now = time.Now
	}
	current := now()
	return &Runtime{
		builder: builder,
		now:     now,
		state: Snapshot{
			Phase:      PhaseIdle,
			StatusText: "Idle — no active mount session",
			StartedAt:  current,
			UpdatedAt:  current,
		},
	}
}

func (r *Runtime) Snapshot() Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

func (r *Runtime) Start(config winclient.Config, activeProfile string) error {
	config = config.Normalized()
	if err := config.Validate(winclient.OperationMount); err != nil {
		r.transitionToError(activeProfile, config, err)
		return err
	}

	r.mu.Lock()
	if r.cancel != nil || r.state.Phase == PhaseConnecting || r.state.Phase == PhaseMounted || r.state.Phase == PhaseStopping {
		current := r.state.Phase
		r.mu.Unlock()
		return fmt.Errorf("mount runtime is busy (%s)", current)
	}
	now := r.now()
	r.state = Snapshot{
		Phase:            PhaseConnecting,
		StatusText:       fmt.Sprintf("Connecting to %s and preparing mount at %s", config.Addr, config.MountPoint),
		ActiveProfile:    activeProfile,
		ServerAddr:       config.Addr,
		MountPoint:       config.MountPoint,
		VolumePrefix:     config.VolumePrefix,
		RemotePath:       config.Path,
		ClientInstanceID: config.ClientInstanceID,
		StartedAt:        now,
		UpdatedAt:        now,
	}
	r.mu.Unlock()

	session, err := r.builder.Build(config)
	if err != nil {
		r.transitionToError(activeProfile, config, err)
		return err
	}
	info := session.Info()
	ctx, cancel := context.WithCancel(context.Background())

	r.mu.Lock()
	r.cancel = cancel
	r.state = Snapshot{
		Phase:             PhaseMounted,
		StatusText:        fmt.Sprintf("Mounted %s at %s", info.ServerAddr, info.MountPoint),
		ActiveProfile:     activeProfile,
		ServerAddr:        info.ServerAddr,
		MountPoint:        info.MountPoint,
		VolumePrefix:      info.VolumePrefix,
		RemotePath:        info.RemotePath,
		ClientInstanceID:  info.ClientInstanceID,
		SessionID:         info.SessionID,
		PrincipalID:       info.PrincipalID,
		ServerName:        info.ServerName,
		ServerVersion:     info.ServerVersion,
		ExpiresAt:         info.ExpiresAt,
		HostBackend:       info.HostBackend,
		HostDLLPath:       info.HostDLLPath,
		HostLauncherPath:  info.HostLauncherPath,
		HostBindingStatus: info.HostBindingStatus,
		StartedAt:         r.state.StartedAt,
		UpdatedAt:         r.now(),
	}
	r.mu.Unlock()

	go r.waitForSession(ctx, session)
	return nil
}

func (r *Runtime) Stop() error {
	r.mu.Lock()
	cancel := r.cancel
	if cancel == nil {
		r.mu.Unlock()
		return fmt.Errorf("no active mount session")
	}
	r.state.Phase = PhaseStopping
	r.state.StatusText = fmt.Sprintf("Stopping mount at %s", r.state.MountPoint)
	r.state.UpdatedAt = r.now()
	r.mu.Unlock()
	cancel()
	return nil
}

func (r *Runtime) waitForSession(ctx context.Context, session Session) {
	err := session.Run(ctx)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cancel = nil
	updated := r.now()
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		r.state.Phase = PhaseIdle
		r.state.StatusText = fmt.Sprintf("Idle — last mount %s stopped", r.state.MountPoint)
		r.state.LastError = ""
		r.state.UpdatedAt = updated
		return
	}
	r.state.Phase = PhaseError
	r.state.StatusText = fmt.Sprintf("Mount runtime failed at %s", r.state.MountPoint)
	r.state.LastError = err.Error()
	r.state.UpdatedAt = updated
}

func (r *Runtime) transitionToError(activeProfile string, config winclient.Config, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	r.state = Snapshot{
		Phase:            PhaseError,
		StatusText:       fmt.Sprintf("Mount runtime failed for %s", config.MountPoint),
		LastError:        err.Error(),
		ActiveProfile:    activeProfile,
		ServerAddr:       config.Addr,
		MountPoint:       config.MountPoint,
		VolumePrefix:     config.VolumePrefix,
		RemotePath:       config.Path,
		ClientInstanceID: config.ClientInstanceID,
		StartedAt:        r.state.StartedAt,
		UpdatedAt:        now,
	}
	if r.state.StartedAt.IsZero() {
		r.state.StartedAt = now
	}
}

type DefaultBuilder struct {
	newHost HostFactory
}

func NewDefaultBuilder(factory HostFactory) DefaultBuilder {
	if factory == nil {
		factory = func(config winfsp.HostConfig, adapter *adapterpkg.Adapter) Host {
			return winfsp.NewHost(config, adapter)
		}
	}
	return DefaultBuilder{newHost: factory}
}

func (b DefaultBuilder) Build(config winclient.Config) (Session, error) {
	config = config.Normalized()
	cli := clientcore.New(config.Addr)
	if err := cli.Connect(); err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = cli.Close()
		}
	}()

	helloResp, err := cli.Hello()
	if err != nil {
		return nil, err
	}
	authResp, err := cli.Auth(config.Token)
	if err != nil {
		return nil, err
	}
	sessionResp, err := cli.CreateSession(config.ClientInstanceID, config.LeaseSeconds)
	if err != nil {
		return nil, err
	}
	if _, err := cli.Heartbeat(); err != nil {
		return nil, err
	}

	mount := mountcore.New(cli, mountcore.Options{
		RootNodeID: cli.RootNodeID,
		VolumeName: config.VolumePrefix,
		ReadOnly:   true,
	})
	adapter := adapterpkg.New(mount)
	host := b.newHost(winfsp.HostConfig{
		MountPoint:   config.MountPoint,
		VolumePrefix: config.VolumePrefix,
	}, adapter)
	binding, err := winfsp.Probe(host.Config())
	if err != nil {
		return nil, err
	}
	cleanup = false
	return defaultSession{
		client: cli,
		host:   host,
		info: SessionInfo{
			ServerAddr:        config.Addr,
			MountPoint:        config.MountPoint,
			VolumePrefix:      config.VolumePrefix,
			RemotePath:        config.Path,
			ClientInstanceID:  config.ClientInstanceID,
			SessionID:         sessionResp.SessionID,
			PrincipalID:       authResp.PrincipalID,
			ServerName:        helloResp.ServerName,
			ServerVersion:     helloResp.ServerVersion,
			ExpiresAt:         sessionResp.ExpiresAt,
			HostBackend:       binding.Backend,
			HostDLLPath:       binding.DLLPath,
			HostLauncherPath:  binding.LauncherPath,
			HostBindingStatus: binding.Summary(),
		},
	}, nil
}

type defaultSession struct {
	client *clientcore.Client
	host   Host
	info   SessionInfo
}

func (s defaultSession) Info() SessionInfo {
	return s.info
}

func (s defaultSession) Run(ctx context.Context) error {
	defer s.client.Close()
	return s.host.Run(ctx)
}
