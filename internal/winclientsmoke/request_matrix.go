package winclientsmoke

import (
	"encoding/json"
	"fmt"
	"strings"

	"developer-mount/internal/winfsp"
)

type RequestStatus string

const (
	RequestStatusReady RequestStatus = "ready"
	RequestStatusGap   RequestStatus = "gap"
)

type RequestMatrixEntry struct {
	ScenarioID string        `json:"scenario_id"`
	Request    string        `json:"request"`
	Callback   string        `json:"callback"`
	Status     RequestStatus `json:"status"`
	Detail     string        `json:"detail,omitempty"`
}

type RequestMatrix struct {
	Ready   int                  `json:"ready"`
	Gaps    int                  `json:"gaps"`
	Entries []RequestMatrixEntry `json:"entries"`
}

func DefaultExplorerRequestMatrix(table winfsp.NativeCallbackTable) RequestMatrix {
	callbackState := map[string]winfsp.NativeCallback{}
	for _, cb := range table.Callbacks {
		callbackState[strings.ToLower(cb.Name)] = cb
	}
	entries := []RequestMatrixEntry{}
	add := func(scenarioID, request, callback string) {
		cb := callbackState[strings.ToLower(callback)]
		status := RequestStatusReady
		detail := cb.Detail
		if cb.State == winfsp.CallbackStateGap || cb.State == winfsp.CallbackStatePreflight {
			status = RequestStatusGap
		}
		entries = append(entries, RequestMatrixEntry{ScenarioID: scenarioID, Request: request, Callback: callback, Status: status, Detail: detail})
	}
	add("explorer-mount-visible", "Query volume label / capability flags", "GetVolumeInfo")
	add("explorer-root-browse", "Root getattr before browse", "GetFileInfo")
	add("explorer-root-browse", "Open root directory", "OpenDirectory")
	add("explorer-root-browse", "Enumerate directory entries", "ReadDirectory")
	add("explorer-file-preview", "Open preview target", "Open")
	add("explorer-file-preview", "Read preview bytes", "Read")
	add("explorer-file-preview", "Cleanup preview handle", "Cleanup")
	add("explorer-file-preview", "Close preview handle", "Close")
	add("explorer-readonly-copy", "Open copy source", "Open")
	add("explorer-readonly-copy", "Read copy source", "Read")
	add("explorer-readonly-copy", "Flush copy handle", "Flush")
	add("explorer-properties", "Query metadata before properties dialog", "GetFileInfo")
	add("explorer-properties", "Query security by name", "GetSecurityByName")
	add("explorer-properties", "Query security on open handle", "GetSecurity")
	add("explorer-security-query", "Read root security by name", "GetSecurityByName")
	add("explorer-unmount-cleanup", "Explorer cleanup after stop", "Cleanup")
	add("explorer-diagnostics", "Query diagnostics metadata", "GetVolumeInfo")
	m := RequestMatrix{Entries: entries}
	for _, e := range entries {
		if e.Status == RequestStatusGap {
			m.Gaps++
		} else {
			m.Ready++
		}
	}
	return m
}

func (m RequestMatrix) Summary() string {
	return fmt.Sprintf("entries=%d ready=%d gaps=%d", len(m.Entries), m.Ready, m.Gaps)
}

func (m RequestMatrix) JSON() ([]byte, error) { return json.MarshalIndent(m, "", "  ") }

func (m RequestMatrix) Markdown() string {
	var b strings.Builder
	b.WriteString("# Explorer Request Matrix\n\n")
	b.WriteString("Summary: " + m.Summary() + "\n\n")
	current := ""
	for _, entry := range m.Entries {
		if entry.ScenarioID != current {
			current = entry.ScenarioID
			b.WriteString("## " + current + "\n")
		}
		b.WriteString(fmt.Sprintf("- [%s] %s -> %s", strings.ToUpper(string(entry.Status)), entry.Request, entry.Callback))
		if entry.Detail != "" {
			b.WriteString(" — " + entry.Detail)
		}
		b.WriteString("\n")
	}
	return b.String()
}
