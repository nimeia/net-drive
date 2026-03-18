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

type InstallerRunRecord struct {
	Channel     string           `json:"channel"`
	Action      string           `json:"action"`
	Status      ValidationStatus `json:"status"`
	VersionFrom string           `json:"version_from,omitempty"`
	VersionTo   string           `json:"version_to,omitempty"`
	LogPath     string           `json:"log_path,omitempty"`
	Notes       string           `json:"notes,omitempty"`
}

type ValidationSummary struct {
	NotRun  int              `json:"not_run"`
	Pass    int              `json:"pass"`
	Warn    int              `json:"warn"`
	Fail    int              `json:"fail"`
	Overall ValidationStatus `json:"overall"`
}

type HostEnvironment struct {
	Source            string   `json:"source,omitempty"`
	Machine           string   `json:"machine,omitempty"`
	OSVersion         string   `json:"os_version,omitempty"`
	WinFspVersion     string   `json:"winfsp_version,omitempty"`
	PackageChannel    string   `json:"package_channel,omitempty"`
	DiagnosticsBundle string   `json:"diagnostics_bundle,omitempty"`
	InstallerLogDir   string   `json:"installer_log_dir,omitempty"`
	Notes             []string `json:"notes,omitempty"`
}

type ValidationPatch struct {
	CompletedBy        string               `json:"completed_by,omitempty"`
	CompletedAt        *time.Time           `json:"completed_at,omitempty"`
	Environment        HostEnvironment      `json:"environment,omitempty"`
	ExplorerScenarios  []ScenarioRecord     `json:"explorer_scenarios,omitempty"`
	InstallerChecklist []ChecklistRecord    `json:"installer_checklist,omitempty"`
	RecoveryChecklist  []ChecklistRecord    `json:"recovery_checklist,omitempty"`
	InstallerRuns      []InstallerRunRecord `json:"installer_runs,omitempty"`
	Notes              []string             `json:"notes,omitempty"`
}

type HostValidationRecord struct {
	GeneratedAt           time.Time            `json:"generated_at"`
	CompletedAt           *time.Time           `json:"completed_at,omitempty"`
	CompletedBy           string               `json:"completed_by,omitempty"`
	Version               string               `json:"version,omitempty"`
	Environment           HostEnvironment      `json:"environment,omitempty"`
	NativeCallbackSummary string               `json:"native_callback_summary"`
	ExplorerMatrixSummary string               `json:"explorer_matrix_summary"`
	ExplorerScenarios     []ScenarioRecord     `json:"explorer_scenarios"`
	InstallerChecklist    []ChecklistRecord    `json:"installer_checklist"`
	RecoveryChecklist     []ChecklistRecord    `json:"recovery_checklist"`
	InstallerRuns         []InstallerRunRecord `json:"installer_runs,omitempty"`
	Summary               ValidationSummary    `json:"summary"`
	Notes                 []string             `json:"notes,omitempty"`
}

func NewHostValidationRecord(version string, smoke []winclientsmoke.Scenario, table winfsp.NativeCallbackTable, matrix winclientsmoke.RequestMatrix) HostValidationRecord {
	record := HostValidationRecord{
		GeneratedAt:           time.Now().UTC(),
		Version:               strings.TrimSpace(version),
		Environment:           HostEnvironment{Source: "real-windows-host"},
		NativeCallbackSummary: table.Summary(),
		ExplorerMatrixSummary: matrix.Summary(),
		Notes: []string{
			"Run this checklist on a real Windows host with WinFsp installed.",
			"Attach exported diagnostics and installer logs to the final validation record.",
			"Record MSI install/upgrade/uninstall and EXE portable launch results in installer_runs for release sign-off.",
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
	record.InstallerRuns = []InstallerRunRecord{
		{Channel: "msi", Action: "install", Status: ValidationNotRun},
		{Channel: "msi", Action: "upgrade", Status: ValidationNotRun},
		{Channel: "msi", Action: "uninstall", Status: ValidationNotRun},
		{Channel: "exe", Action: "portable-launch", Status: ValidationNotRun},
	}
	record.RecomputeSummary()
	return record
}

func (r *HostValidationRecord) RecomputeSummary() {
	s := ValidationSummary{Overall: ValidationNotRun}
	count := func(status ValidationStatus) {
		switch status {
		case ValidationPass:
			s.Pass++
		case ValidationWarn:
			s.Warn++
		case ValidationFail:
			s.Fail++
		default:
			s.NotRun++
		}
	}
	for _, item := range r.ExplorerScenarios {
		count(item.Status)
	}
	for _, item := range r.InstallerChecklist {
		count(item.Status)
	}
	for _, item := range r.RecoveryChecklist {
		count(item.Status)
	}
	for _, item := range r.InstallerRuns {
		count(item.Status)
	}
	s.Overall = ValidationNotRun
	if s.Fail > 0 {
		s.Overall = ValidationFail
	} else if s.Warn > 0 {
		s.Overall = ValidationWarn
	} else if s.Pass > 0 && s.NotRun == 0 {
		s.Overall = ValidationPass
	} else if s.Pass > 0 {
		s.Overall = ValidationWarn
	}
	r.Summary = s
}

func (r *HostValidationRecord) ApplyScenario(id string, status ValidationStatus, notes string) bool {
	for i := range r.ExplorerScenarios {
		if r.ExplorerScenarios[i].ScenarioID == id {
			r.ExplorerScenarios[i].Status = status
			r.ExplorerScenarios[i].Notes = strings.TrimSpace(notes)
			r.RecomputeSummary()
			return true
		}
	}
	return false
}

func (r *HostValidationRecord) ApplyChecklist(section, item string, status ValidationStatus, notes string) bool {
	section = strings.ToLower(strings.TrimSpace(section))
	var list *[]ChecklistRecord
	switch section {
	case "installer":
		list = &r.InstallerChecklist
	case "recovery":
		list = &r.RecoveryChecklist
	default:
		return false
	}
	for i := range *list {
		if (*list)[i].Item == item {
			(*list)[i].Status = status
			(*list)[i].Notes = strings.TrimSpace(notes)
			r.RecomputeSummary()
			return true
		}
	}
	return false
}

func (r *HostValidationRecord) ApplyInstallerRun(channel, action string, status ValidationStatus, notes string) bool {
	for i := range r.InstallerRuns {
		if r.InstallerRuns[i].Channel == channel && r.InstallerRuns[i].Action == action {
			r.InstallerRuns[i].Status = status
			r.InstallerRuns[i].Notes = strings.TrimSpace(notes)
			r.RecomputeSummary()
			return true
		}
	}
	return false
}

func (r *HostValidationRecord) ApplyEnvironment(env HostEnvironment) {
	if strings.TrimSpace(env.Source) != "" {
		r.Environment.Source = strings.TrimSpace(env.Source)
	}
	if strings.TrimSpace(env.Machine) != "" {
		r.Environment.Machine = strings.TrimSpace(env.Machine)
	}
	if strings.TrimSpace(env.OSVersion) != "" {
		r.Environment.OSVersion = strings.TrimSpace(env.OSVersion)
	}
	if strings.TrimSpace(env.WinFspVersion) != "" {
		r.Environment.WinFspVersion = strings.TrimSpace(env.WinFspVersion)
	}
	if strings.TrimSpace(env.PackageChannel) != "" {
		r.Environment.PackageChannel = strings.TrimSpace(env.PackageChannel)
	}
	if strings.TrimSpace(env.DiagnosticsBundle) != "" {
		r.Environment.DiagnosticsBundle = strings.TrimSpace(env.DiagnosticsBundle)
	}
	if strings.TrimSpace(env.InstallerLogDir) != "" {
		r.Environment.InstallerLogDir = strings.TrimSpace(env.InstallerLogDir)
	}
	if len(env.Notes) > 0 {
		r.Environment.Notes = append(r.Environment.Notes, env.Notes...)
	}
}

func (r *HostValidationRecord) AddNote(note string) {
	note = strings.TrimSpace(note)
	if note == "" {
		return
	}
	r.Notes = append(r.Notes, note)
}

func (r *HostValidationRecord) ApplyPatch(patch ValidationPatch) []string {
	warnings := []string{}
	r.ApplyEnvironment(patch.Environment)
	for _, item := range patch.ExplorerScenarios {
		if !r.ApplyScenario(item.ScenarioID, item.Status, item.Notes) {
			warnings = append(warnings, fmt.Sprintf("unknown explorer scenario %q", item.ScenarioID))
		}
	}
	for _, item := range patch.InstallerChecklist {
		if !r.ApplyChecklist("installer", item.Item, item.Status, item.Notes) {
			warnings = append(warnings, fmt.Sprintf("unknown installer checklist item %q", item.Item))
		}
	}
	for _, item := range patch.RecoveryChecklist {
		if !r.ApplyChecklist("recovery", item.Item, item.Status, item.Notes) {
			warnings = append(warnings, fmt.Sprintf("unknown recovery checklist item %q", item.Item))
		}
	}
	for _, item := range patch.InstallerRuns {
		updated := false
		for i := range r.InstallerRuns {
			if r.InstallerRuns[i].Channel == item.Channel && r.InstallerRuns[i].Action == item.Action {
				r.InstallerRuns[i].Status = item.Status
				r.InstallerRuns[i].Notes = strings.TrimSpace(item.Notes)
				if strings.TrimSpace(item.VersionFrom) != "" {
					r.InstallerRuns[i].VersionFrom = strings.TrimSpace(item.VersionFrom)
				}
				if strings.TrimSpace(item.VersionTo) != "" {
					r.InstallerRuns[i].VersionTo = strings.TrimSpace(item.VersionTo)
				}
				if strings.TrimSpace(item.LogPath) != "" {
					r.InstallerRuns[i].LogPath = strings.TrimSpace(item.LogPath)
				}
				updated = true
			}
		}
		if !updated {
			warnings = append(warnings, fmt.Sprintf("unknown installer run %q/%q", item.Channel, item.Action))
		}
	}
	for _, note := range patch.Notes {
		r.AddNote(note)
	}
	completedBy := strings.TrimSpace(patch.CompletedBy)
	if patch.CompletedAt != nil || completedBy != "" {
		when := time.Now().UTC()
		if patch.CompletedAt != nil {
			when = patch.CompletedAt.UTC()
		}
		if completedBy == "" {
			completedBy = r.CompletedBy
		}
		r.MarkCompleted(completedBy, when)
	}
	r.RecomputeSummary()
	return warnings
}

func (r *HostValidationRecord) MarkCompleted(by string, when time.Time) {
	when = when.UTC()
	r.CompletedBy = strings.TrimSpace(by)
	r.CompletedAt = &when
}

func (r HostValidationRecord) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

func hasEnvironmentData(env HostEnvironment) bool {
	return strings.TrimSpace(env.Source) != "" || strings.TrimSpace(env.Machine) != "" || strings.TrimSpace(env.OSVersion) != "" || strings.TrimSpace(env.WinFspVersion) != "" || strings.TrimSpace(env.PackageChannel) != "" || strings.TrimSpace(env.DiagnosticsBundle) != "" || strings.TrimSpace(env.InstallerLogDir) != "" || len(env.Notes) > 0
}
func (r HostValidationRecord) Markdown() string {
	var b strings.Builder
	b.WriteString("# Windows Host Validation Record\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", r.GeneratedAt.Format(time.RFC3339)))
	if r.CompletedAt != nil {
		b.WriteString(fmt.Sprintf("Completed: %s\n\n", r.CompletedAt.Format(time.RFC3339)))
	}
	if strings.TrimSpace(r.CompletedBy) != "" {
		b.WriteString(fmt.Sprintf("Completed by: %s\n\n", r.CompletedBy))
	}
	if strings.TrimSpace(r.Version) != "" {
		b.WriteString(fmt.Sprintf("Version: %s\n\n", r.Version))
	}
	b.WriteString(fmt.Sprintf("Summary: overall=%s pass=%d warn=%d fail=%d not-run=%d\n", strings.ToUpper(string(r.Summary.Overall)), r.Summary.Pass, r.Summary.Warn, r.Summary.Fail, r.Summary.NotRun))
	b.WriteString("Native callback summary: " + r.NativeCallbackSummary + "\n")
	b.WriteString("Explorer matrix summary: " + r.ExplorerMatrixSummary + "\n\n")
	if hasEnvironmentData(r.Environment) {
		b.WriteString("## Environment\n")
		if strings.TrimSpace(r.Environment.Source) != "" {
			b.WriteString("- source: " + r.Environment.Source + "\n")
		}
		if strings.TrimSpace(r.Environment.Machine) != "" {
			b.WriteString("- machine: " + r.Environment.Machine + "\n")
		}
		if strings.TrimSpace(r.Environment.OSVersion) != "" {
			b.WriteString("- os_version: " + r.Environment.OSVersion + "\n")
		}
		if strings.TrimSpace(r.Environment.WinFspVersion) != "" {
			b.WriteString("- winfsp_version: " + r.Environment.WinFspVersion + "\n")
		}
		if strings.TrimSpace(r.Environment.PackageChannel) != "" {
			b.WriteString("- package_channel: " + r.Environment.PackageChannel + "\n")
		}
		if strings.TrimSpace(r.Environment.DiagnosticsBundle) != "" {
			b.WriteString("- diagnostics_bundle: " + r.Environment.DiagnosticsBundle + "\n")
		}
		if strings.TrimSpace(r.Environment.InstallerLogDir) != "" {
			b.WriteString("- installer_log_dir: " + r.Environment.InstallerLogDir + "\n")
		}
		for _, note := range r.Environment.Notes {
			b.WriteString("- note: " + note + "\n")
		}
		b.WriteString("\n")
	}
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
	if len(r.InstallerRuns) > 0 {
		b.WriteString("\n## Installer runs\n")
		for _, item := range r.InstallerRuns {
			b.WriteString(fmt.Sprintf("- [%s] %s/%s", strings.ToUpper(string(item.Status)), strings.ToUpper(item.Channel), item.Action))
			if item.VersionFrom != "" || item.VersionTo != "" {
				b.WriteString(fmt.Sprintf(" (%s -> %s)", defaultValue(item.VersionFrom, "-"), defaultValue(item.VersionTo, "-")))
			}
			b.WriteString("\n")
			if strings.TrimSpace(item.LogPath) != "" {
				b.WriteString("  - log: " + item.LogPath + "\n")
			}
			if strings.TrimSpace(item.Notes) != "" {
				b.WriteString("  - notes: " + item.Notes + "\n")
			}
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
