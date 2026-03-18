package winclientrelease

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ReleaseCandidate struct {
	GeneratedAt       time.Time      `json:"generated_at"`
	Version           string         `json:"version,omitempty"`
	Channel           string         `json:"channel"`
	ReleaseReady      bool           `json:"release_ready"`
	ManifestPath      string         `json:"manifest_path,omitempty"`
	ValidationPath    string         `json:"validation_path,omitempty"`
	ClosurePath       string         `json:"closure_path,omitempty"`
	IssueListPath     string         `json:"issue_list_path,omitempty"`
	FixPlanPath       string         `json:"fix_plan_path,omitempty"`
	ArtifactCount     int            `json:"artifact_count"`
	OpenIssues        int            `json:"open_issues"`
	FinalStatus       string         `json:"final_status"`
	ClosureReasons    []string       `json:"closure_reasons,omitempty"`
	OutstandingIssues []ReleaseIssue `json:"outstanding_issues,omitempty"`
	Artifacts         []Artifact     `json:"artifacts,omitempty"`
}

func NewReleaseCandidate(manifest Manifest, validation HostValidationRecord, closure ReleaseClosure, issues PreReleaseIssueList) ReleaseCandidate {
	status := "blocked"
	if closure.ReleaseReady && issues.OpenCount == 0 && validation.Summary.Overall == ValidationPass {
		status = "rc-ready"
	} else if validation.Summary.Overall == ValidationWarn || issues.OpenCount > 0 {
		status = "needs-attention"
	}
	return ReleaseCandidate{
		GeneratedAt:       time.Now().UTC(),
		Version:           strings.TrimSpace(firstNonEmpty(closure.Version, validation.Version, manifest.Version)),
		Channel:           "rc",
		ReleaseReady:      closure.ReleaseReady,
		ManifestPath:      "release-manifest.json",
		ValidationPath:    defaultValue(manifest.ValidationResult, "windows-host-validation-result-template.json"),
		ClosurePath:       defaultValue(manifest.ReleaseClosure, "windows-release-closure.json"),
		IssueListPath:     defaultValue(manifest.IssueList, "windows-pre-release-issues.json"),
		FixPlanPath:       "windows-first-pass-fix-plan.json",
		ArtifactCount:     len(manifest.Artifacts),
		OpenIssues:        issues.OpenCount,
		FinalStatus:       status,
		ClosureReasons:    append([]string(nil), closure.Reasons...),
		OutstandingIssues: append([]ReleaseIssue(nil), issues.Issues...),
		Artifacts:         append([]Artifact(nil), manifest.Artifacts...),
	}
}

func (r ReleaseCandidate) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

func (r ReleaseCandidate) Markdown() string {
	var b strings.Builder
	b.WriteString("# Windows Release Candidate\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", r.GeneratedAt.Format(time.RFC3339)))
	if strings.TrimSpace(r.Version) != "" {
		b.WriteString(fmt.Sprintf("Version: %s\n\n", r.Version))
	}
	b.WriteString(fmt.Sprintf("Channel: %s\n", r.Channel))
	b.WriteString(fmt.Sprintf("Final status: %s\n", r.FinalStatus))
	b.WriteString(fmt.Sprintf("Release ready: %t\n", r.ReleaseReady))
	b.WriteString(fmt.Sprintf("Open issues: %d\n", r.OpenIssues))
	b.WriteString(fmt.Sprintf("Artifacts: %d\n\n", r.ArtifactCount))
	if len(r.ClosureReasons) > 0 {
		b.WriteString("## Closure reasons\n")
		for _, reason := range r.ClosureReasons {
			b.WriteString("- " + reason + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Output files\n")
	b.WriteString("- manifest: " + r.ManifestPath + "\n")
	b.WriteString("- validation: " + r.ValidationPath + "\n")
	b.WriteString("- closure: " + r.ClosurePath + "\n")
	b.WriteString("- issue list: " + r.IssueListPath + "\n")
	b.WriteString("- fix plan: " + r.FixPlanPath + "\n")
	return b.String()
}
