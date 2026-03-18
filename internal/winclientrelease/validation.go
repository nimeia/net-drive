package winclientrelease

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
)

type ValidationStatus string

const (
	ValidationNotRun ValidationStatus = "not-run"
	ValidationPass   ValidationStatus = "pass"
	ValidationWarn   ValidationStatus = "warn"
	ValidationFail   ValidationStatus = "fail"
)

type ScenarioRecord struct {
	ScenarioID string           `json:"scenario_id"`
	Name       string           `json:"name"`
	Status     ValidationStatus `json:"status"`
	Notes      string           `json:"notes,omitempty"`
}

type ChecklistRecord struct {
	Item   string           `json:"item"`
	Status ValidationStatus `json:"status"`
	Notes  string           `json:"notes,omitempty"`
}

type HostValidationRecord struct {
	GeneratedAt           time.Time         `json:"generated_at"`
	Version               string            `json:"version,omitempty"`
	NativeCallbackSummary string            `json:"native_callback_summary"`
	ExplorerMatrixSummary string            `json:"explorer_matrix_summary"`
	ExplorerScenarios     []ScenarioRecord  `json:"explorer_scenarios"`
	InstallerChecklist    []ChecklistRecord `json:"installer_checklist"`
	RecoveryChecklist     []ChecklistRecord `json:"recovery_checklist"`
	Notes                 []string          `json:"notes,omitempty"`
}

func NewHostValidationRecord(version string, smoke []winclientsmoke.Scenario, table winfsp.NativeCallbackTable, matrix winclientsmoke.RequestMatrix) HostValidationRecord {
	record := HostValidationRecord{
		GeneratedAt:           time.Now().UTC(),
		Version:               strings.TrimSpace(version),
		NativeCallbackSummary: table.Summary(),
		ExplorerMatrixSummary: matrix.Summary(),
		Notes: []string{
			"Run this checklist on a real Windows host with WinFsp installed.",
			"Attach exported diagnostics and installer logs to the final validation record.",
		},
	}
	for _, s := range smoke {
		record.ExplorerScenarios = append(record.ExplorerScenarios, ScenarioRecord{ScenarioID: s.ID, Name: s.Name, Status: ValidationNotRun})
	}
	record.InstallerChecklist = []ChecklistRecord{
		{Item: "WinFsp installed and version captured", Status: ValidationNotRun},
		{Item: "MSI install succeeded", Status: ValidationNotRun},
		{Item: "EXE/portable bundle launch succeeded", Status: ValidationNotRun},
		{Item: "Shortcuts / tray / Dashboard available after install", Status: ValidationNotRun},
		{Item: "Upgrade path preserves config", Status: ValidationNotRun},
		{Item: "Uninstall removes binaries and leaves expected user data policy", Status: ValidationNotRun},
	}
	record.RecoveryChecklist = []ChecklistRecord{
		{Item: "Dirty-exit marker written during forced termination", Status: ValidationNotRun},
		{Item: "Next launch reports recovery warning", Status: ValidationNotRun},
		{Item: "Clean stop clears recovery marker state", Status: ValidationNotRun},
	}
	return record
}

func (r HostValidationRecord) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

func (r HostValidationRecord) Markdown() string {
	var b strings.Builder
	b.WriteString("# Windows Host Validation Record\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", r.GeneratedAt.Format(time.RFC3339)))
	if strings.TrimSpace(r.Version) != "" {
		b.WriteString(fmt.Sprintf("Version: %s\n\n", r.Version))
	}
	b.WriteString("Native callback summary: " + r.NativeCallbackSummary + "\n")
	b.WriteString("Explorer matrix summary: " + r.ExplorerMatrixSummary + "\n\n")
	b.WriteString("## Explorer scenarios\n")
	for _, s := range r.ExplorerScenarios {
		b.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", strings.ToUpper(string(s.Status)), s.Name, s.ScenarioID))
		if strings.TrimSpace(s.Notes) != "" {
			b.WriteString("  - notes: " + s.Notes + "\n")
		}
	}
	b.WriteString("\n## Installer checklist\n")
	for _, item := range r.InstallerChecklist {
		b.WriteString(fmt.Sprintf("- [%s] %s\n", strings.ToUpper(string(item.Status)), item.Item))
		if strings.TrimSpace(item.Notes) != "" {
			b.WriteString("  - notes: " + item.Notes + "\n")
		}
	}
	b.WriteString("\n## Recovery checklist\n")
	for _, item := range r.RecoveryChecklist {
		b.WriteString(fmt.Sprintf("- [%s] %s\n", strings.ToUpper(string(item.Status)), item.Item))
		if strings.TrimSpace(item.Notes) != "" {
			b.WriteString("  - notes: " + item.Notes + "\n")
		}
	}
	if len(r.Notes) > 0 {
		b.WriteString("\n## Notes\n")
		for _, note := range r.Notes {
			b.WriteString("- " + note + "\n")
		}
	}
	return b.String()
}
