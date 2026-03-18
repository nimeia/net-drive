package winclientrecovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"developer-mount/internal/winclient"
	"developer-mount/internal/winclientruntime"
)

const overrideEnvName = "DEVMOUNT_WINCLIENT_RECOVERY_PATH"

type State struct {
	Dirty         bool      `json:"dirty"`
	StartedAt     time.Time `json:"started_at,omitempty"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
	LastCleanExit time.Time `json:"last_clean_exit,omitempty"`
	ActiveProfile string    `json:"active_profile,omitempty"`
	MountPoint    string    `json:"mount_point,omitempty"`
	HostBackend   string    `json:"host_backend,omitempty"`
	LastPhase     string    `json:"last_phase,omitempty"`
	LastError     string    `json:"last_error,omitempty"`
}

func (s State) Summary() string {
	if s.Dirty {
		return fmt.Sprintf("unclean shutdown detected (profile=%s mount=%s phase=%s)", defaultText(s.ActiveProfile, "-"), defaultText(s.MountPoint, "-"), defaultText(s.LastPhase, "-"))
	}
	if !s.LastCleanExit.IsZero() {
		return fmt.Sprintf("last clean exit at %s", s.LastCleanExit.Format(time.RFC3339))
	}
	return "no previous recovery marker"
}

type Store struct {
	path string
	now  func() time.Time
}

func New(path string) Store { return Store{path: path, now: time.Now} }
func OpenDefault() (Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return Store{}, err
	}
	return New(path), nil
}
func DefaultPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv(overrideEnvName)); override != "" {
		return override, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "developer-mount", "win32-client-recovery.json"), nil
}
func (s Store) Path() string { return s.path }
func (s Store) Load() (State, error) {
	if strings.TrimSpace(s.path) == "" {
		return State{}, nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, fmt.Errorf("read recovery marker: %w", err)
	}
	var state State
	if len(data) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("decode recovery marker: %w", err)
	}
	return state, nil
}
func (s Store) MarkStart(profile string, cfg winclient.Config, snapshot winclientruntime.Snapshot) (State, error) {
	if s.now == nil {
		s.now = time.Now
	}
	state := State{Dirty: true, StartedAt: s.now(), UpdatedAt: s.now(), ActiveProfile: strings.TrimSpace(profile), MountPoint: cfg.MountPoint, HostBackend: cfg.HostBackend, LastPhase: string(snapshot.Phase), LastError: snapshot.LastError}
	return state, s.save(state)
}
func (s Store) Update(snapshot winclientruntime.Snapshot) (State, error) {
	if s.now == nil {
		s.now = time.Now
	}
	state, err := s.Load()
	if err != nil {
		return State{}, err
	}
	state.UpdatedAt = s.now()
	state.LastPhase = string(snapshot.Phase)
	state.LastError = snapshot.LastError
	if strings.TrimSpace(snapshot.MountPoint) != "" {
		state.MountPoint = snapshot.MountPoint
	}
	if strings.TrimSpace(snapshot.ActiveProfile) != "" {
		state.ActiveProfile = snapshot.ActiveProfile
	}
	if strings.TrimSpace(snapshot.RequestedBackend) != "" {
		state.HostBackend = snapshot.RequestedBackend
	}
	return state, s.save(state)
}
func (s Store) MarkCleanExit(snapshot winclientruntime.Snapshot) (State, error) {
	if s.now == nil {
		s.now = time.Now
	}
	state, err := s.Load()
	if err != nil {
		return State{}, err
	}
	state.Dirty = false
	state.LastCleanExit = s.now()
	state.UpdatedAt = state.LastCleanExit
	state.LastPhase = string(snapshot.Phase)
	state.LastError = snapshot.LastError
	return state, s.save(state)
}
func (s Store) save(state State) error {
	if strings.TrimSpace(s.path) == "" {
		return fmt.Errorf("recovery path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create recovery directory: %w", err)
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode recovery marker: %w", err)
	}
	payload = append(payload, '\n')
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return fmt.Errorf("write recovery marker: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("replace recovery marker: %w", err)
	}
	return nil
}
func defaultText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
