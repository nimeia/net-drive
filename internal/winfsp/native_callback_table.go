package winfsp

import (
	"encoding/json"
	"fmt"
	"strings"
)

type NativeCallbackState string

const (
	CallbackStateReady     NativeCallbackState = "ready"
	CallbackStateGap       NativeCallbackState = "gap"
	CallbackStateReadOnly  NativeCallbackState = "read-only"
	CallbackStatePreflight NativeCallbackState = "preflight-only"
)

type NativeCallback struct {
	Name        string              `json:"name"`
	State       NativeCallbackState `json:"state"`
	ExplorerHot bool                `json:"explorer_hot"`
	Detail      string              `json:"detail,omitempty"`
	Remediation string              `json:"remediation,omitempty"`
	RequiredFor []string            `json:"required_for,omitempty"`
}

type NativeCallbackTable struct {
	Backend   string           `json:"backend"`
	Active    bool             `json:"active"`
	Ready     int              `json:"ready"`
	Gaps      int              `json:"gaps"`
	ReadOnly  int              `json:"read_only"`
	Preflight int              `json:"preflight_only"`
	Callbacks []NativeCallback `json:"callbacks"`
}

func DefaultNativeCallbackTable(binding BindingInfo) NativeCallbackTable {
	backend := binding.EffectiveBackend
	if strings.TrimSpace(backend) == "" {
		backend = binding.Backend
	}
	dispatcher := strings.Contains(strings.ToLower(backend), "dispatcher")
	active := dispatcher && binding.DispatcherReady && binding.CallbackBridgeReady && binding.ServiceLoopReady
	state := CallbackStatePreflight
	if active {
		state = CallbackStateReady
	}
	callbacks := []NativeCallback{
		{Name: "GetVolumeInfo", State: state, ExplorerHot: true, Detail: "Volume label and capability metadata are wired through mountcore.", RequiredFor: []string{"explorer-mount-visible", "explorer-root-browse"}},
		{Name: "GetFileInfo", State: state, ExplorerHot: true, Detail: "Path metadata lookup is bridged through adapter -> mountcore.", RequiredFor: []string{"explorer-root-browse", "explorer-file-preview", "explorer-properties"}},
		{Name: "Open", State: state, ExplorerHot: true, Detail: "File open is mapped to read-only handles.", RequiredFor: []string{"explorer-file-preview", "explorer-readonly-copy"}},
		{Name: "OpenDirectory", State: state, ExplorerHot: true, Detail: "Directory open is bridged through mountcore.", RequiredFor: []string{"explorer-root-browse"}},
		{Name: "ReadDirectory", State: state, ExplorerHot: true, Detail: "Directory enumeration is available for Explorer browse and refresh.", RequiredFor: []string{"explorer-root-browse"}},
		{Name: "Read", State: state, ExplorerHot: true, Detail: "Read requests support preview/copy for small and medium files.", RequiredFor: []string{"explorer-file-preview", "explorer-readonly-copy"}},
		{Name: "Close", State: state, ExplorerHot: true, Detail: "Handle close is propagated back to mountcore.", RequiredFor: []string{"explorer-file-preview", "explorer-unmount-cleanup"}},
		{Name: "Cleanup", State: state, ExplorerHot: true, Detail: "Cleanup is bridged as a harmless handle finalization step for read-only mode.", RequiredFor: []string{"explorer-unmount-cleanup", "explorer-file-preview"}},
		{Name: "Flush", State: state, ExplorerHot: false, Detail: "Flush is bridged as a read-only success path for Explorer/editor compatibility.", RequiredFor: []string{"explorer-readonly-copy"}},
		{Name: "GetSecurityByName", State: state, ExplorerHot: true, Detail: "Security descriptor probing is served with a minimal read-only descriptor.", RequiredFor: []string{"explorer-properties", "explorer-security-query"}},
		{Name: "GetSecurity", State: state, ExplorerHot: true, Detail: "Per-handle security query is available after open/open-directory.", RequiredFor: []string{"explorer-properties", "explorer-security-query"}},
		{Name: "SetSecurity", State: CallbackStateReadOnly, ExplorerHot: false, Detail: "Write-side security mutation is intentionally blocked for the read-only client."},
		{Name: "SetBasicInfo", State: CallbackStateReadOnly, ExplorerHot: false, Detail: "Timestamp/attribute mutation is intentionally blocked for read-only mode."},
		{Name: "SetFileSize", State: CallbackStateReadOnly, ExplorerHot: false, Detail: "Resize is intentionally blocked for read-only mode."},
		{Name: "Write", State: CallbackStateReadOnly, ExplorerHot: false, Detail: "Write callback remains unavailable in read-only mode."},
		{Name: "CanDelete", State: CallbackStateReadOnly, ExplorerHot: false, Detail: "Delete checks are intentionally denied in read-only mode."},
		{Name: "Rename", State: CallbackStateReadOnly, ExplorerHot: false, Detail: "Rename is intentionally denied in read-only mode."},
		{Name: "Overwrite", State: CallbackStateReadOnly, ExplorerHot: false, Detail: "Overwrite is intentionally denied in read-only mode."},
	}
	table := NativeCallbackTable{Backend: backend, Active: active, Callbacks: callbacks}
	for _, cb := range callbacks {
		switch cb.State {
		case CallbackStateReady:
			table.Ready++
		case CallbackStateGap:
			table.Gaps++
		case CallbackStateReadOnly:
			table.ReadOnly++
		case CallbackStatePreflight:
			table.Preflight++
		}
	}
	return table
}

func (t NativeCallbackTable) Summary() string {
	return fmt.Sprintf("backend=%s active=%v ready=%d gaps=%d read_only=%d preflight_only=%d", defaultDispatcherValue(t.Backend, "-"), t.Active, t.Ready, t.Gaps, t.ReadOnly, t.Preflight)
}

func (t NativeCallbackTable) MissingHotPathCount() int {
	total := 0
	for _, cb := range t.Callbacks {
		if cb.ExplorerHot && cb.State == CallbackStateGap {
			total++
		}
	}
	return total
}

func (t NativeCallbackTable) JSON() ([]byte, error) { return json.MarshalIndent(t, "", "  ") }

func (t NativeCallbackTable) Markdown() string {
	var b strings.Builder
	b.WriteString("# WinFsp Native Callback Table\n\n")
	b.WriteString(fmt.Sprintf("Summary: %s\n\n", t.Summary()))
	for _, cb := range t.Callbacks {
		b.WriteString("## " + cb.Name + "\n")
		b.WriteString(fmt.Sprintf("- state: %s\n", cb.State))
		b.WriteString(fmt.Sprintf("- explorer_hot: %v\n", cb.ExplorerHot))
		if cb.Detail != "" {
			b.WriteString("- detail: " + cb.Detail + "\n")
		}
		if len(cb.RequiredFor) > 0 {
			b.WriteString("- required_for: " + strings.Join(cb.RequiredFor, ", ") + "\n")
		}
		if cb.Remediation != "" {
			b.WriteString("- remediation: " + cb.Remediation + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}
