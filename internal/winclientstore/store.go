package winclientstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"developer-mount/internal/winclient"
)

const (
	SchemaVersion   = 2
	overrideEnvName = "DEVMOUNT_WINCLIENT_STORE_PATH"
)

type Settings struct {
	DefaultWorkspace string `json:"default_workspace,omitempty"`
	AutoReconnect    bool   `json:"auto_reconnect,omitempty"`
	LaunchOnLogin    bool   `json:"launch_on_login,omitempty"`
}

type WorkspaceMeta struct {
	DisplayName string `json:"display_name,omitempty"`
	AutoMount   bool   `json:"auto_mount,omitempty"`
	LastUsedAt  string `json:"last_used_at,omitempty"`
}

type State struct {
	Version       int                         `json:"version"`
	ActiveProfile string                      `json:"active_profile,omitempty"`
	Profiles      map[string]winclient.Config `json:"profiles,omitempty"`
	Settings      Settings                    `json:"settings,omitempty"`
	WorkspaceMeta map[string]WorkspaceMeta    `json:"workspace_meta,omitempty"`
}

type Store struct{ path string }

func New(path string) Store { return Store{path: path} }
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
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "developer-mount", "win32-client.json"), nil
}
func (s Store) Path() string { return s.path }
func (s Store) Load() (State, error) {
	state := DefaultState()
	if strings.TrimSpace(s.path) == "" {
		return state, nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return State{}, fmt.Errorf("read store %s: %w", s.path, err)
	}
	if len(data) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("decode store %s: %w", s.path, err)
	}
	return normalizeState(state), nil
}
func (s Store) Save(state State) error {
	if strings.TrimSpace(s.path) == "" {
		return fmt.Errorf("store path is empty")
	}
	state = normalizeState(state)
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create store directory: %w", err)
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode store: %w", err)
	}
	payload = append(payload, '\n')
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("write temp store: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace store: %w", err)
	}
	return nil
}
func (s Store) SaveProfile(name string, config winclient.Config) (State, error) {
	return s.SaveWorkspace(name, config, WorkspaceMeta{})
}
func (s Store) SaveWorkspace(name string, config winclient.Config, meta WorkspaceMeta) (State, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return State{}, fmt.Errorf("profile name is required")
	}
	state, err := s.Load()
	if err != nil {
		return State{}, err
	}
	state.Profiles[name] = config.Normalized()
	state.WorkspaceMeta[name] = normalizeWorkspaceMeta(meta)
	state.ActiveProfile = name
	if strings.TrimSpace(state.Settings.DefaultWorkspace) == "" {
		state.Settings.DefaultWorkspace = name
	}
	if err := s.Save(state); err != nil {
		return State{}, err
	}
	return state, nil
}
func (s Store) DeleteProfile(name string) (State, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return State{}, fmt.Errorf("profile name is required")
	}
	state, err := s.Load()
	if err != nil {
		return State{}, err
	}
	delete(state.Profiles, name)
	delete(state.WorkspaceMeta, name)
	if state.ActiveProfile == name {
		state.ActiveProfile = ""
		for _, next := range SortedProfileNames(state) {
			state.ActiveProfile = next
			break
		}
	}
	if state.Settings.DefaultWorkspace == name {
		state.Settings.DefaultWorkspace = state.ActiveProfile
	}
	if err := s.Save(state); err != nil {
		return State{}, err
	}
	return state, nil
}
func SortedProfileNames(state State) []string {
	names := make([]string, 0, len(state.Profiles))
	for name := range state.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
func DefaultState() State {
	return State{Version: SchemaVersion, Profiles: map[string]winclient.Config{}, WorkspaceMeta: map[string]WorkspaceMeta{}}
}
func normalizeState(state State) State {
	if state.Version == 0 || state.Version < SchemaVersion {
		state.Version = SchemaVersion
	}
	if state.Profiles == nil {
		state.Profiles = map[string]winclient.Config{}
	}
	if state.WorkspaceMeta == nil {
		state.WorkspaceMeta = map[string]WorkspaceMeta{}
	}
	state.Settings.DefaultWorkspace = strings.TrimSpace(state.Settings.DefaultWorkspace)
	normalizedProfiles := make(map[string]winclient.Config, len(state.Profiles))
	normalizedMeta := make(map[string]WorkspaceMeta, len(state.WorkspaceMeta))
	for name, cfg := range state.Profiles {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		normalizedProfiles[trimmed] = cfg.Normalized()
		normalizedMeta[trimmed] = normalizeWorkspaceMeta(state.WorkspaceMeta[trimmed])
	}
	state.Profiles = normalizedProfiles
	state.WorkspaceMeta = normalizedMeta
	if _, ok := state.Profiles[state.ActiveProfile]; !ok {
		state.ActiveProfile = ""
	}
	if state.Settings.DefaultWorkspace == "" {
		if state.ActiveProfile != "" {
			state.Settings.DefaultWorkspace = state.ActiveProfile
		} else {
			for _, name := range SortedProfileNames(state) {
				state.Settings.DefaultWorkspace = name
				break
			}
		}
	}
	return state
}
func normalizeWorkspaceMeta(meta WorkspaceMeta) WorkspaceMeta {
	meta.DisplayName = strings.TrimSpace(meta.DisplayName)
	meta.LastUsedAt = normalizeTimestamp(meta.LastUsedAt)
	return meta
}
func normalizeTimestamp(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return ""
	}
	return parsed.UTC().Format(time.RFC3339)
}
