package winclientrelease

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type FixPlanItem struct {
	ID             string        `json:"id"`
	Severity       IssueSeverity `json:"severity"`
	Category       string        `json:"category"`
	Title          string        `json:"title"`
	Evidence       string        `json:"evidence,omitempty"`
	Remediation    string        `json:"remediation,omitempty"`
	SuggestedArea  string        `json:"suggested_area,omitempty"`
	SuggestedFiles []string      `json:"suggested_files,omitempty"`
}

type FirstPassFixPlan struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Version     string        `json:"version,omitempty"`
	Blockers    int           `json:"blockers"`
	Warnings    int           `json:"warnings"`
	Infos       int           `json:"infos"`
	Items       []FixPlanItem `json:"items,omitempty"`
}

func NewFirstPassFixPlan(manifest Manifest, validation HostValidationRecord, issues PreReleaseIssueList) FirstPassFixPlan {
	plan := FirstPassFixPlan{GeneratedAt: time.Now().UTC(), Version: strings.TrimSpace(firstNonEmpty(validation.Version, manifest.Version, issues.Version))}
	for _, issue := range issues.Issues {
		item := FixPlanItem{
			ID:             issue.ID,
			Severity:       issue.Severity,
			Category:       issue.Category,
			Title:          issue.Title,
			Evidence:       issue.Evidence,
			Remediation:    issue.Remediation,
			SuggestedArea:  suggestedAreaForIssue(issue),
			SuggestedFiles: suggestedFilesForIssue(issue),
		}
		plan.Items = append(plan.Items, item)
		switch issue.Severity {
		case IssueSeverityBlocker:
			plan.Blockers++
		case IssueSeverityWarning:
			plan.Warnings++
		default:
			plan.Infos++
		}
	}
	return plan
}

func suggestedAreaForIssue(issue ReleaseIssue) string {
	switch issue.Category {
	case "explorer":
		return "winfsp-native-callbacks"
	case "installer", "installer-run":
		return "windows-release-packaging"
	case "recovery":
		return "winclient-recovery"
	case "closure":
		return "release-closure"
	default:
		return "windows-client-productization"
	}
}

func suggestedFilesForIssue(issue ReleaseIssue) []string {
	switch issue.Category {
	case "explorer":
		return []string{"internal/winfsp/callbacks.go", "internal/winfsp/dispatcher_bridge.go", "internal/winclientsmoke/request_matrix.go"}
	case "installer":
		return []string{"scripts/package-windows-release.ps1", "scripts/package-windows-installer.ps1", "internal/winclientrelease/release.go"}
	case "installer-run":
		return []string{"scripts/backfill-windows-validation.ps1", "scripts/finalize-windows-release.ps1", "internal/winclientrelease/intake.go"}
	case "recovery":
		return []string{"internal/winclientrecovery/recovery.go", "internal/winclientdiag/diag.go", "internal/winclientgui/diag_windows.go"}
	case "closure":
		return []string{"internal/winclientrelease/closure.go", "internal/winclientrelease/issues.go", "scripts/finalize-windows-release.ps1"}
	default:
		return nil
	}
}

func (p FirstPassFixPlan) JSON() ([]byte, error) { return json.MarshalIndent(p, "", "  ") }

func (p FirstPassFixPlan) Markdown() string {
	var b strings.Builder
	b.WriteString("# Windows First-Pass Fix Plan\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", p.GeneratedAt.Format(time.RFC3339)))
	if strings.TrimSpace(p.Version) != "" {
		b.WriteString(fmt.Sprintf("Version: %s\n\n", p.Version))
	}
	b.WriteString(fmt.Sprintf("Blockers: %d\nWarnings: %d\nInfos: %d\n\n", p.Blockers, p.Warnings, p.Infos))
	if len(p.Items) == 0 {
		b.WriteString("No fixes queued.\n")
		return b.String()
	}
	for _, item := range p.Items {
		b.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", strings.ToUpper(string(item.Severity)), item.Title, item.ID))
		if item.SuggestedArea != "" {
			b.WriteString("  - area: " + item.SuggestedArea + "\n")
		}
		if item.Evidence != "" {
			b.WriteString("  - evidence: " + item.Evidence + "\n")
		}
		if item.Remediation != "" {
			b.WriteString("  - remediation: " + item.Remediation + "\n")
		}
		if len(item.SuggestedFiles) > 0 {
			b.WriteString("  - suggested_files: " + strings.Join(item.SuggestedFiles, ", ") + "\n")
		}
	}
	return b.String()
}
