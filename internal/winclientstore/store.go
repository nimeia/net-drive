package winclientstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"developer-mount/internal/winclient"
)

const (
	SchemaVersion   = 1
	overrideEnvName = "DEVMOUNT_WINCLIENT_STORE_PATH"
)

type State struct {
	Version       int                         `json:"version"`
	ActiveProfile string                      `json:"active_profile,omitempty"`
	Profiles      map[string]winclient.Config `json:"profiles,omitempty"`
}

type Store struct {
	path string
}

func New(path string) Store {
	return Store{path: path}
}

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

func (s Store) Path() string {
	return s.path
}

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
	name = strings.TrimSpace(name)
	if name == "" {
		return State{}, fmt.Errorf("profile name is required")
	}
	state, err := s.Load()
	if err != nil {
		return State{}, err
	}
	state.Profiles[name] = config.Normalized()
	state.ActiveProfile = name
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
	if state.ActiveProfile == name {
		state.ActiveProfile = ""
		for _, next := range SortedProfileNames(state) {
			state.ActiveProfile = next
			break
		}
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
	return State{
		Version:  SchemaVersion,
		Profiles: map[string]winclient.Config{},
	}
}

func normalizeState(state State) State {
	if state.Version == 0 {
		state.Version = SchemaVersion
	}
	if state.Profiles == nil {
		state.Profiles = map[string]winclient.Config{}
	}
	normalized := make(map[string]winclient.Config, len(state.Profiles))
	for name, config := range state.Profiles {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		normalized[trimmed] = config.Normalized()
	}
	state.Profiles = normalized
	if _, ok := state.Profiles[state.ActiveProfile]; !ok {
		state.ActiveProfile = ""
	}
	return state
}
