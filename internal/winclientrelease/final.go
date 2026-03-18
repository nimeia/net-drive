package winclientrelease

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type FinalRelease struct {
	GeneratedAt          time.Time        `json:"generated_at"`
	Version              string           `json:"version,omitempty"`
	Channel              string           `json:"channel"`
	ReleaseReady         bool             `json:"release_ready"`
	PublishReady         bool             `json:"publish_ready"`
	FinalStatus          string           `json:"final_status"`
	ValidationOverall    ValidationStatus `json:"validation_overall"`
	OpenIssues           int              `json:"open_issues"`
	MissingEvidenceCount int              `json:"missing_evidence_count"`
	ManifestPath         string           `json:"manifest_path,omitempty"`
	ValidationPath       string           `json:"validation_path,omitempty"`
	IntakePath           string           `json:"intake_path,omitempty"`
	ClosurePath          string           `json:"closure_path,omitempty"`
	IssueListPath        string           `json:"issue_list_path,omitempty"`
	FixPlanPath          string           `json:"fix_plan_path,omitempty"`
	RCPath               string           `json:"rc_path,omitempty"`
	ClosureReasons       []string         `json:"closure_reasons,omitempty"`
	OutstandingNotes     []string         `json:"outstanding_notes,omitempty"`
}

func NewFinalRelease(manifest Manifest, validation HostValidationRecord, intake ValidationIntakeReport, closure ReleaseClosure, issues PreReleaseIssueList, rc ReleaseCandidate) FinalRelease {
	final := FinalRelease{
		GeneratedAt:          time.Now().UTC(),
		Version:              strings.TrimSpace(firstNonEmpty(rc.Version, validation.Version, manifest.Version)),
		Channel:              "stable",
		ReleaseReady:         closure.ReleaseReady,
		ValidationOverall:    validation.Summary.Overall,
		OpenIssues:           issues.OpenCount,
		MissingEvidenceCount: intake.MissingEvidenceCount,
		ManifestPath:         "release-manifest.json",
		ValidationPath:       defaultValue(manifest.ValidationResult, "windows-host-validation-result-template.json"),
		IntakePath:           defaultValue(manifest.ValidationIntake, "windows-validation-intake-report.json"),
		ClosurePath:          defaultValue(manifest.ReleaseClosure, "windows-release-closure.json"),
		IssueListPath:        defaultValue(manifest.IssueList, "windows-pre-release-issues.json"),
		FixPlanPath:          defaultValue(manifest.FixPlan, "windows-first-pass-fix-plan.json"),
		RCPath:               defaultValue(manifest.ReleaseCandidate, "windows-release-candidate.json"),
		ClosureReasons:       append([]string(nil), closure.Reasons...),
	}
	if !intake.ReadyForTargetedFix {
		final.OutstandingNotes = append(final.OutstandingNotes, "validation intake still misses required Windows-host evidence")
	}
	if issues.OpenCount > 0 {
		final.OutstandingNotes = append(final.OutstandingNotes, "pre-release issue list still contains open items")
	}
	if validation.Summary.Overall != ValidationPass {
		final.OutstandingNotes = append(final.OutstandingNotes, "validation summary is not fully pass")
	}
	if closure.ReleaseReady && rc.FinalStatus == "rc-ready" && intake.ReadyForTargetedFix && issues.OpenCount == 0 && validation.Summary.Overall == ValidationPass {
		final.PublishReady = true
		final.FinalStatus = "publish-ready"
	} else if validation.Summary.Overall == ValidationWarn || issues.OpenCount > 0 || intake.MissingEvidenceCount > 0 {
		final.FinalStatus = "needs-attention"
	} else {
		final.FinalStatus = "blocked"
	}
	return final
}

func (r FinalRelease) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

func (r FinalRelease) Markdown() string {
	var b strings.Builder
	b.WriteString("# Windows Final Release\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", r.GeneratedAt.Format(time.RFC3339)))
	if strings.TrimSpace(r.Version) != "" {
		b.WriteString(fmt.Sprintf("Version: %s\n\n", r.Version))
	}
	b.WriteString(fmt.Sprintf("Channel: %s\n", r.Channel))
	b.WriteString(fmt.Sprintf("Final status: %s\n", r.FinalStatus))
	b.WriteString(fmt.Sprintf("Release ready: %t\n", r.ReleaseReady))
	b.WriteString(fmt.Sprintf("Publish ready: %t\n", r.PublishReady))
	b.WriteString(fmt.Sprintf("Validation overall: %s\n", r.ValidationOverall))
	b.WriteString(fmt.Sprintf("Open issues: %d\n", r.OpenIssues))
	b.WriteString(fmt.Sprintf("Missing evidence: %d\n\n", r.MissingEvidenceCount))
	if len(r.ClosureReasons) > 0 {
		b.WriteString("## Closure reasons\n")
		for _, reason := range r.ClosureReasons {
			b.WriteString("- " + reason + "\n")
		}
		b.WriteString("\n")
	}
	if len(r.OutstandingNotes) > 0 {
		b.WriteString("## Outstanding notes\n")
		for _, note := range r.OutstandingNotes {
			b.WriteString("- " + note + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Output files\n")
	b.WriteString("- manifest: " + r.ManifestPath + "\n")
	b.WriteString("- validation: " + r.ValidationPath + "\n")
	b.WriteString("- intake: " + r.IntakePath + "\n")
	b.WriteString("- closure: " + r.ClosurePath + "\n")
	b.WriteString("- issue list: " + r.IssueListPath + "\n")
	b.WriteString("- fix plan: " + r.FixPlanPath + "\n")
	b.WriteString("- RC: " + r.RCPath + "\n")
	return b.String()
}

func (r FinalRelease) SignoffMarkdown() string {
	var b strings.Builder
	b.WriteString("# Windows Final Release Sign-Off\n\n")
	if strings.TrimSpace(r.Version) != "" {
		b.WriteString(fmt.Sprintf("Version: %s\n\n", r.Version))
	}
	b.WriteString(fmt.Sprintf("Final status: %s\n", r.FinalStatus))
	b.WriteString(fmt.Sprintf("Publish ready: %t\n\n", r.PublishReady))
	b.WriteString("## Release gates\n")
	b.WriteString(fmt.Sprintf("- [ ] Validation overall is PASS (%s)\n", r.ValidationOverall))
	b.WriteString(fmt.Sprintf("- [ ] Open issue count is zero (%d)\n", r.OpenIssues))
	b.WriteString(fmt.Sprintf("- [ ] Missing evidence count is zero (%d)\n", r.MissingEvidenceCount))
	b.WriteString(fmt.Sprintf("- [ ] Release closure is ready (%t)\n", r.ReleaseReady))
	b.WriteString(fmt.Sprintf("- [ ] Final release is publish-ready (%t)\n\n", r.PublishReady))
	b.WriteString("## Signatures\n")
	b.WriteString("- Engineering: ____________________  Date: __________\n")
	b.WriteString("- QA / Windows Host Validation: ____  Date: __________\n")
	b.WriteString("- Release / Packaging: _____________  Date: __________\n")
	return b.String()
}
