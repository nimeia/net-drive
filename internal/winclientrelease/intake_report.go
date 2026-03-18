package winclientrelease

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type IntakeEvidenceStatus string

const (
	IntakeEvidencePresent IntakeEvidenceStatus = "present"
	IntakeEvidenceMissing IntakeEvidenceStatus = "missing"
)

type IntakeEvidence struct {
	Key         string               `json:"key"`
	Status      IntakeEvidenceStatus `json:"status"`
	Detail      string               `json:"detail,omitempty"`
	Remediation string               `json:"remediation,omitempty"`
}

type ValidationIntakeReport struct {
	GeneratedAt          time.Time        `json:"generated_at"`
	Version              string           `json:"version,omitempty"`
	Completed            bool             `json:"completed"`
	ValidationOverall    ValidationStatus `json:"validation_overall"`
	ReadyForTargetedFix  bool             `json:"ready_for_targeted_fix"`
	MissingEvidenceCount int              `json:"missing_evidence_count"`
	OpenScenarioCount    int              `json:"open_scenario_count"`
	OpenInstallerRuns    int              `json:"open_installer_runs"`
	OpenChecklistItems   int              `json:"open_checklist_items"`
	Evidence             []IntakeEvidence `json:"evidence,omitempty"`
	PendingScenarios     []string         `json:"pending_scenarios,omitempty"`
	PendingInstallerRuns []string         `json:"pending_installer_runs,omitempty"`
	PendingChecklist     []string         `json:"pending_checklist,omitempty"`
	Notes                []string         `json:"notes,omitempty"`
}

func NewValidationIntakeReport(manifest Manifest, validation HostValidationRecord) ValidationIntakeReport {
	report := ValidationIntakeReport{
		GeneratedAt:       time.Now().UTC(),
		Version:           strings.TrimSpace(firstNonEmpty(validation.Version, manifest.Version)),
		Completed:         validation.CompletedAt != nil,
		ValidationOverall: validation.Summary.Overall,
	}
	addEvidence := func(key, value, remediation string) {
		status := IntakeEvidencePresent
		detail := strings.TrimSpace(value)
		if detail == "" {
			status = IntakeEvidenceMissing
			report.MissingEvidenceCount++
		}
		report.Evidence = append(report.Evidence, IntakeEvidence{Key: key, Status: status, Detail: detail, Remediation: remediation})
	}
	addEvidence("completed_at", func() string {
		if validation.CompletedAt == nil {
			return ""
		}
		return validation.CompletedAt.UTC().Format(time.RFC3339)
	}(), "Mark the validation result completed after the first Windows-host run is fully backfilled.")
	addEvidence("environment.machine", validation.Environment.Machine, "Record the Windows host machine name used for the first-pass validation.")
	addEvidence("environment.os_version", validation.Environment.OSVersion, "Capture the exact Windows build used for validation.")
	addEvidence("environment.winfsp_version", validation.Environment.WinFspVersion, "Capture the WinFsp version from the validation host.")
	addEvidence("environment.diagnostics_bundle", validation.Environment.DiagnosticsBundle, "Attach the exported diagnostics ZIP produced after the Explorer and installer validation runs.")
	addEvidence("environment.installer_log_dir", validation.Environment.InstallerLogDir, "Archive the MSI/EXE installer logs and record the directory path.")

	for _, scenario := range validation.ExplorerScenarios {
		if scenario.Status != ValidationPass {
			report.OpenScenarioCount++
			report.PendingScenarios = append(report.PendingScenarios, fmt.Sprintf("%s (%s)", scenario.ScenarioID, scenario.Status))
		}
	}
	for _, run := range validation.InstallerRuns {
		if run.Status != ValidationPass {
			report.OpenInstallerRuns++
			report.PendingInstallerRuns = append(report.PendingInstallerRuns, fmt.Sprintf("%s/%s (%s)", run.Channel, run.Action, run.Status))
		}
		if strings.TrimSpace(run.LogPath) == "" {
			report.MissingEvidenceCount++
			report.Evidence = append(report.Evidence, IntakeEvidence{
				Key:         fmt.Sprintf("installer_runs.%s.%s.log_path", run.Channel, run.Action),
				Status:      IntakeEvidenceMissing,
				Remediation: "Capture the installer log path for each real Windows-host installer run.",
			})
		}
	}
	appendChecklist := func(prefix string, items []ChecklistRecord) {
		for _, item := range items {
			if item.Status != ValidationPass {
				report.OpenChecklistItems++
				report.PendingChecklist = append(report.PendingChecklist, fmt.Sprintf("%s: %s (%s)", prefix, item.Item, item.Status))
			}
		}
	}
	appendChecklist("installer", validation.InstallerChecklist)
	appendChecklist("recovery", validation.RecoveryChecklist)

	report.ReadyForTargetedFix = report.Completed && report.MissingEvidenceCount == 0
	if !report.Completed {
		report.Notes = append(report.Notes, "The validation result has not been marked completed yet.")
	}
	if report.MissingEvidenceCount > 0 {
		report.Notes = append(report.Notes, "Capture the missing diagnostics bundle, installer logs, and host metadata before claiming first-pass fixes are grounded in real Windows-host evidence.")
	}
	if report.OpenScenarioCount+report.OpenInstallerRuns+report.OpenChecklistItems == 0 {
		report.Notes = append(report.Notes, "All first-pass Windows-host checks are recorded as pass.")
	}
	return report
}

func (r ValidationIntakeReport) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

func (r ValidationIntakeReport) Markdown() string {
	var b strings.Builder
	b.WriteString("# Windows Validation Intake Report\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", r.GeneratedAt.Format(time.RFC3339)))
	if strings.TrimSpace(r.Version) != "" {
		b.WriteString(fmt.Sprintf("Version: %s\n\n", r.Version))
	}
	b.WriteString(fmt.Sprintf("Completed: %t\n", r.Completed))
	b.WriteString(fmt.Sprintf("Validation overall: %s\n", r.ValidationOverall))
	b.WriteString(fmt.Sprintf("Ready for targeted fix: %t\n", r.ReadyForTargetedFix))
	b.WriteString(fmt.Sprintf("Missing evidence: %d\n", r.MissingEvidenceCount))
	b.WriteString(fmt.Sprintf("Open explorer scenarios: %d\n", r.OpenScenarioCount))
	b.WriteString(fmt.Sprintf("Open installer runs: %d\n", r.OpenInstallerRuns))
	b.WriteString(fmt.Sprintf("Open checklist items: %d\n\n", r.OpenChecklistItems))
	if len(r.Evidence) > 0 {
		b.WriteString("## Evidence\n")
		for _, item := range r.Evidence {
			b.WriteString(fmt.Sprintf("- [%s] %s", strings.ToUpper(string(item.Status)), item.Key))
			if item.Detail != "" {
				b.WriteString(": " + item.Detail)
			}
			b.WriteString("\n")
			if item.Remediation != "" {
				b.WriteString("  - remediation: " + item.Remediation + "\n")
			}
		}
		b.WriteString("\n")
	}
	writeList := func(title string, values []string) {
		if len(values) == 0 {
			return
		}
		b.WriteString("## " + title + "\n")
		for _, v := range values {
			b.WriteString("- " + v + "\n")
		}
		b.WriteString("\n")
	}
	writeList("Pending Explorer Scenarios", r.PendingScenarios)
	writeList("Pending Installer Runs", r.PendingInstallerRuns)
	writeList("Pending Checklist Items", r.PendingChecklist)
	if len(r.Notes) > 0 {
		b.WriteString("## Notes\n")
		for _, note := range r.Notes {
			b.WriteString("- " + note + "\n")
		}
	}
	return b.String()
}
