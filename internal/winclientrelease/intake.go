package winclientrelease

import (
	"encoding/json"
	"fmt"
	"strings"
)

type InstallerActionResult struct {
	Status      ValidationStatus `json:"status"`
	VersionFrom string           `json:"version_from,omitempty"`
	VersionTo   string           `json:"version_to,omitempty"`
	LogPath     string           `json:"log_path,omitempty"`
	Notes       string           `json:"notes,omitempty"`
}

type InstallerResultSet struct {
	Version string `json:"version,omitempty"`
	LogDir  string `json:"log_dir,omitempty"`
	MSI     struct {
		Install   InstallerActionResult `json:"install"`
		Upgrade   InstallerActionResult `json:"upgrade"`
		Uninstall InstallerActionResult `json:"uninstall"`
	} `json:"msi"`
	EXE struct {
		PortableLaunch InstallerActionResult `json:"portable_launch"`
	} `json:"exe"`
	Notes []string `json:"notes,omitempty"`
}

func NewInstallerResultSetTemplate(version string) InstallerResultSet {
	set := InstallerResultSet{Version: strings.TrimSpace(version)}
	set.MSI.Install = InstallerActionResult{Status: ValidationNotRun, VersionTo: strings.TrimSpace(version)}
	set.MSI.Upgrade = InstallerActionResult{Status: ValidationNotRun, VersionTo: strings.TrimSpace(version)}
	set.MSI.Uninstall = InstallerActionResult{Status: ValidationNotRun, VersionTo: strings.TrimSpace(version)}
	set.EXE.PortableLaunch = InstallerActionResult{Status: ValidationNotRun, VersionTo: strings.TrimSpace(version)}
	set.Notes = []string{"Fill this file on the real Windows host after each installer validation round."}
	return set
}

func (s InstallerResultSet) JSON() ([]byte, error) { return json.MarshalIndent(s, "", "  ") }

func NewValidationPatchTemplate(validation HostValidationRecord) ValidationPatch {
	patch := ValidationPatch{
		Environment: HostEnvironment{Source: validation.Environment.Source, PackageChannel: validation.Environment.PackageChannel},
		Notes:       []string{"Fill only the fields observed during the current Windows host validation round."},
	}
	if strings.TrimSpace(patch.Environment.Source) == "" {
		patch.Environment.Source = "real-windows-host"
	}
	if strings.TrimSpace(patch.Environment.PackageChannel) == "" {
		patch.Environment.PackageChannel = "msi,exe"
	}
	patch.ExplorerScenarios = cloneScenarioRecords(validation.ExplorerScenarios)
	patch.InstallerChecklist = cloneChecklistRecords(validation.InstallerChecklist)
	patch.RecoveryChecklist = cloneChecklistRecords(validation.RecoveryChecklist)
	patch.InstallerRuns = cloneInstallerRunRecords(validation.InstallerRuns)
	for i := range patch.ExplorerScenarios {
		patch.ExplorerScenarios[i].Status = ValidationNotRun
		patch.ExplorerScenarios[i].Notes = ""
	}
	for i := range patch.InstallerChecklist {
		patch.InstallerChecklist[i].Status = ValidationNotRun
		patch.InstallerChecklist[i].Notes = ""
	}
	for i := range patch.RecoveryChecklist {
		patch.RecoveryChecklist[i].Status = ValidationNotRun
		patch.RecoveryChecklist[i].Notes = ""
	}
	for i := range patch.InstallerRuns {
		patch.InstallerRuns[i].Status = ValidationNotRun
		patch.InstallerRuns[i].Notes = ""
		patch.InstallerRuns[i].LogPath = ""
		patch.InstallerRuns[i].VersionFrom = ""
	}
	return patch
}

func (r *HostValidationRecord) ApplyInstallerResultSet(set InstallerResultSet) []string {
	warnings := []string{}
	if strings.TrimSpace(set.LogDir) != "" {
		r.Environment.InstallerLogDir = strings.TrimSpace(set.LogDir)
	}
	apply := func(channel, action string, result InstallerActionResult) {
		if !r.ApplyInstallerRun(channel, action, result.Status, result.Notes) {
			warnings = append(warnings, fmt.Sprintf("unknown installer run %q/%q", channel, action))
			return
		}
		for i := range r.InstallerRuns {
			if r.InstallerRuns[i].Channel == channel && r.InstallerRuns[i].Action == action {
				if strings.TrimSpace(result.VersionFrom) != "" {
					r.InstallerRuns[i].VersionFrom = strings.TrimSpace(result.VersionFrom)
				}
				if strings.TrimSpace(result.VersionTo) != "" {
					r.InstallerRuns[i].VersionTo = strings.TrimSpace(result.VersionTo)
				}
				if strings.TrimSpace(result.LogPath) != "" {
					r.InstallerRuns[i].LogPath = strings.TrimSpace(result.LogPath)
				}
			}
		}
	}
	apply("msi", "install", set.MSI.Install)
	apply("msi", "upgrade", set.MSI.Upgrade)
	apply("msi", "uninstall", set.MSI.Uninstall)
	apply("exe", "portable-launch", set.EXE.PortableLaunch)

	applyChecklistFromRun := func(item string, result InstallerActionResult) {
		if !r.ApplyChecklist("installer", item, result.Status, result.Notes) {
			warnings = append(warnings, fmt.Sprintf("unknown installer checklist item %q", item))
		}
	}
	applyChecklistFromRun("MSI install succeeded", set.MSI.Install)
	applyChecklistFromRun("EXE/portable bundle launch succeeded", set.EXE.PortableLaunch)
	for _, note := range set.Notes {
		r.AddNote(note)
	}
	if strings.TrimSpace(set.Version) != "" && strings.TrimSpace(r.Version) == "" {
		r.Version = strings.TrimSpace(set.Version)
	}
	r.RecomputeSummary()
	return warnings
}

func cloneScenarioRecords(src []ScenarioRecord) []ScenarioRecord {
	dst := make([]ScenarioRecord, len(src))
	copy(dst, src)
	return dst
}

func cloneChecklistRecords(src []ChecklistRecord) []ChecklistRecord {
	dst := make([]ChecklistRecord, len(src))
	copy(dst, src)
	return dst
}

func cloneInstallerRunRecords(src []InstallerRunRecord) []InstallerRunRecord {
	dst := make([]InstallerRunRecord, len(src))
	copy(dst, src)
	return dst
}
