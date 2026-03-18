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
	Backend       string           `json:"backend"`
	Active        bool             `json:"active"`
	Finalized     bool             `json:"finalized"`
	Ready         int              `json:"ready"`
	Gaps          int              `json:"gaps"`
	ReadOnly      int              `json:"read_only"`
	Preflight     int              `json:"preflight_only"`
	Callbacks     []NativeCallback `json:"callbacks"`
	ClosureDetail string           `json:"closure_detail,omitempty"`
}

func DefaultNativeCallbackTable(binding BindingInfo) NativeCallbackTable {
	backend := binding.EffectiveBackend
	if strings.TrimSpace(backend) == "" {
		backend = binding.Backend
	}
	dispatcher := strings.Contains(strings.ToLower(backend), "dispatcher")
	active := dispatcher && binding.DispatcherReady && binding.CallbackBridgeReady && binding.ServiceLoopReady
	readyState := CallbackStatePreflight
	if active {
		readyState = CallbackStateReady
	}
	callbacks := []NativeCallback{
		{Name: "GetVolumeInfo", State: readyState, ExplorerHot: true, Detail: "Volume label and capability metadata are wired through mountcore.", RequiredFor: []string{"explorer-mount-visible", "explorer-root-browse"}},
		{Name: "GetFileInfo", State: readyState, ExplorerHot: true, Detail: "Path metadata lookup is bridged through adapter -> mountcore.", RequiredFor: []string{"explorer-root-browse", "explorer-file-preview", "explorer-properties"}},
		{Name: "Create", State: CallbackStateReadOnly, ExplorerHot: true, Detail: "Create is explicitly denied in read-only mode so Explorer receives a clean create denial instead of a gap.", RequiredFor: []string{"explorer-create-denied"}},
		{Name: "Open", State: readyState, ExplorerHot: true, Detail: "File open is mapped to read-only handles.", RequiredFor: []string{"explorer-file-preview", "explorer-readonly-copy", "explorer-delete-denied", "explorer-write-denied", "explorer-rename-denied"}},
		{Name: "OpenDirectory", State: readyState, ExplorerHot: true, Detail: "Directory open is bridged through mountcore.", RequiredFor: []string{"explorer-root-browse"}},
		{Name: "ReadDirectory", State: readyState, ExplorerHot: true, Detail: "Directory enumeration is available for Explorer browse and refresh.", RequiredFor: []string{"explorer-root-browse"}},
		{Name: "Read", State: readyState, ExplorerHot: true, Detail: "Read requests support preview/copy for small and medium files.", RequiredFor: []string{"explorer-file-preview", "explorer-readonly-copy"}},
		{Name: "Write", State: CallbackStateReadOnly, ExplorerHot: true, Detail: "Write is explicitly denied on open handles in read-only mode.", RequiredFor: []string{"explorer-write-denied"}},
		{Name: "Overwrite", State: CallbackStateReadOnly, ExplorerHot: false, Detail: "Overwrite/supersede is explicitly denied in read-only mode.", RequiredFor: []string{"explorer-write-denied"}},
		{Name: "Cleanup", State: readyState, ExplorerHot: true, Detail: "Cleanup tracks handle-finalization state and preserves delete-on-close denial semantics until close.", RequiredFor: []string{"explorer-unmount-cleanup", "explorer-file-preview", "explorer-delete-denied"}},
		{Name: "Close", State: readyState, ExplorerHot: true, Detail: "Handle close is propagated back to mountcore.", RequiredFor: []string{"explorer-file-preview", "explorer-unmount-cleanup", "explorer-delete-denied"}},
		{Name: "Flush", State: readyState, ExplorerHot: false, Detail: "Flush is bridged as a read-only success path for Explorer/editor compatibility.", RequiredFor: []string{"explorer-readonly-copy", "explorer-write-denied"}},
		{Name: "GetSecurityByName", State: readyState, ExplorerHot: true, Detail: "Security descriptor probing returns a stable read-only descriptor with owner/group/access details.", RequiredFor: []string{"explorer-properties", "explorer-security-query"}},
		{Name: "GetSecurity", State: readyState, ExplorerHot: true, Detail: "Per-handle security query includes cleanup/flush state and delete-on-close intent.", RequiredFor: []string{"explorer-properties", "explorer-security-query", "explorer-delete-denied"}},
		{Name: "SetSecurity", State: CallbackStateReadOnly, ExplorerHot: false, Detail: "Write-side security mutation is explicitly denied for the read-only client.", RequiredFor: []string{"explorer-write-denied"}},
		{Name: "SetBasicInfo", State: CallbackStateReadOnly, ExplorerHot: false, Detail: "Timestamp/attribute mutation is explicitly denied for read-only mode.", RequiredFor: []string{"explorer-write-denied"}},
		{Name: "SetFileSize", State: CallbackStateReadOnly, ExplorerHot: false, Detail: "Resize is explicitly denied for read-only mode.", RequiredFor: []string{"explorer-write-denied"}},
		{Name: "CanDelete", State: CallbackStateReadOnly, ExplorerHot: true, Detail: "Delete checks are explicitly denied in read-only mode, which lets Explorer surface a clean no-delete result.", RequiredFor: []string{"explorer-delete-denied"}},
		{Name: "SetDeleteOnClose", State: CallbackStateReadOnly, ExplorerHot: true, Detail: "Delete-on-close is tracked for diagnostics but denied for the read-only client.", RequiredFor: []string{"explorer-delete-denied"}},
		{Name: "Rename", State: CallbackStateReadOnly, ExplorerHot: true, Detail: "Rename is explicitly denied in read-only mode.", RequiredFor: []string{"explorer-rename-denied"}},
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
	table.Finalized = table.Gaps == 0
	if table.Finalized {
		table.ClosureDetail = "all native callbacks in the current matrix are either dispatcher-ready or explicitly denied in read-only mode"
	} else {
		table.ClosureDetail = "callback matrix still contains implementation gaps"
	}
	return table
}
func (t NativeCallbackTable) Summary() string {
	return fmt.Sprintf("backend=%s active=%v finalized=%v ready=%d gaps=%d read_only=%d preflight_only=%d", defaultDispatcherValue(t.Backend, "-"), t.Active, t.Finalized, t.Ready, t.Gaps, t.ReadOnly, t.Preflight)
}
func (t NativeCallbackTable) MissingHotPathCount() int {
	missing := 0
	for _, cb := range t.Callbacks {
		if cb.ExplorerHot && (cb.State == CallbackStateGap || cb.State == CallbackStatePreflight) {
			missing++
		}
	}
	return missing
}
func (t NativeCallbackTable) JSON() ([]byte, error) { return json.MarshalIndent(t, "", "  ") }
func (t NativeCallbackTable) Markdown() string {
	var b strings.Builder
	b.WriteString("# WinFsp Native Callback Table\n\n")
	b.WriteString("Summary: " + t.Summary() + "\n")
	if t.ClosureDetail != "" {
		b.WriteString("Closure: " + t.ClosureDetail + "\n")
	}
	b.WriteString("\n")
	for _, cb := range t.Callbacks {
		b.WriteString(fmt.Sprintf("- **%s** [%s]", cb.Name, strings.ToUpper(string(cb.State))))
		if cb.ExplorerHot {
			b.WriteString(" *(Explorer hot-path)*")
		}
		if cb.Detail != "" {
			b.WriteString(" — " + cb.Detail)
		}
		b.WriteString("\n")
		if len(cb.RequiredFor) > 0 {
			b.WriteString("  - required for: " + strings.Join(cb.RequiredFor, ", ") + "\n")
		}
		if cb.Remediation != "" {
			b.WriteString("  - remediation: " + cb.Remediation + "\n")
		}
	}
	return b.String()
}
