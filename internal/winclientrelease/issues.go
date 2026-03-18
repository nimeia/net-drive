package winclientrelease

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type IssueSeverity string

type IssueStatus string

const (
	IssueSeverityBlocker IssueSeverity = "blocker"
	IssueSeverityWarning IssueSeverity = "warning"
	IssueSeverityInfo    IssueSeverity = "info"

	IssueStatusOpen   IssueStatus = "open"
	IssueStatusClosed IssueStatus = "closed"
)

type ReleaseIssue struct {
	ID          string        `json:"id"`
	Category    string        `json:"category"`
	Title       string        `json:"title"`
	Severity    IssueSeverity `json:"severity"`
	Status      IssueStatus   `json:"status"`
	Evidence    string        `json:"evidence,omitempty"`
	Remediation string        `json:"remediation,omitempty"`
}

type PreReleaseIssueList struct {
	GeneratedAt  time.Time      `json:"generated_at"`
	Version      string         `json:"version,omitempty"`
	ReleaseReady bool           `json:"release_ready"`
	OpenCount    int            `json:"open_count"`
	ClosedCount  int            `json:"closed_count"`
	Issues       []ReleaseIssue `json:"issues,omitempty"`
}

func NewPreReleaseIssueList(manifest Manifest, validation HostValidationRecord, closure ReleaseClosure) PreReleaseIssueList {
	list := PreReleaseIssueList{GeneratedAt: time.Now().UTC(), Version: strings.TrimSpace(firstNonEmpty(closure.Version, manifest.Version, validation.Version)), ReleaseReady: closure.ReleaseReady}
	for _, item := range validation.ExplorerScenarios {
		if issue, ok := issueFromScenario(item); ok {
			list.Issues = append(list.Issues, issue)
		}
	}
	for _, item := range validation.InstallerChecklist {
		if issue, ok := issueFromChecklist("installer", item); ok {
			list.Issues = append(list.Issues, issue)
		}
	}
	for _, item := range validation.RecoveryChecklist {
		if issue, ok := issueFromChecklist("recovery", item); ok {
			list.Issues = append(list.Issues, issue)
		}
	}
	for _, item := range validation.InstallerRuns {
		if issue, ok := issueFromInstallerRun(item); ok {
			list.Issues = append(list.Issues, issue)
		}
	}
	for idx, reason := range closure.Reasons {
		list.Issues = append(list.Issues, ReleaseIssue{ID: fmt.Sprintf("closure-%02d", idx+1), Category: "closure", Title: reason, Severity: IssueSeverityBlocker, Status: IssueStatusOpen, Evidence: "release closure evaluator", Remediation: "Update the Windows host validation result and regenerate the release closure."})
	}
	for _, issue := range list.Issues {
		if issue.Status == IssueStatusClosed {
			list.ClosedCount++
		} else {
			list.OpenCount++
		}
	}
	return list
}

func issueFromScenario(item ScenarioRecord) (ReleaseIssue, bool) {
	switch item.Status {
	case ValidationPass:
		return ReleaseIssue{}, false
	case ValidationWarn:
		return ReleaseIssue{ID: "scenario-" + item.ScenarioID, Category: "explorer", Title: item.Name, Severity: IssueSeverityWarning, Status: IssueStatusOpen, Evidence: strings.TrimSpace(item.Notes), Remediation: "Re-run the Explorer smoke scenario on a real Windows host and capture a stable pass or explicit release waiver."}, true
	case ValidationFail:
		return ReleaseIssue{ID: "scenario-" + item.ScenarioID, Category: "explorer", Title: item.Name, Severity: IssueSeverityBlocker, Status: IssueStatusOpen, Evidence: strings.TrimSpace(item.Notes), Remediation: "Fix the Explorer path or capture a release blocker waiver before shipping."}, true
	default:
		return ReleaseIssue{ID: "scenario-" + item.ScenarioID, Category: "explorer", Title: item.Name, Severity: IssueSeverityBlocker, Status: IssueStatusOpen, Evidence: "scenario not yet executed on a Windows host", Remediation: "Run this Explorer smoke scenario and backfill the result."}, true
	}
}

func issueFromChecklist(section string, item ChecklistRecord) (ReleaseIssue, bool) {
	if item.Status == ValidationPass {
		return ReleaseIssue{}, false
	}
	sev := IssueSeverityBlocker
	if item.Status == ValidationWarn {
		sev = IssueSeverityWarning
	}
	evidence := strings.TrimSpace(item.Notes)
	if evidence == "" {
		evidence = fmt.Sprintf("%s checklist item has status %s", section, item.Status)
	}
	return ReleaseIssue{ID: fmt.Sprintf("%s-%s", section, slug(item.Item)), Category: section, Title: item.Item, Severity: sev, Status: IssueStatusOpen, Evidence: evidence, Remediation: fmt.Sprintf("Backfill the %s checklist item with a stable Windows-host result.", section)}, true
}

func issueFromInstallerRun(item InstallerRunRecord) (ReleaseIssue, bool) {
	if item.Status == ValidationPass {
		return ReleaseIssue{}, false
	}
	sev := IssueSeverityBlocker
	if item.Status == ValidationWarn {
		sev = IssueSeverityWarning
	}
	evidence := strings.TrimSpace(item.Notes)
	if evidence == "" {
		evidence = fmt.Sprintf("installer run %s/%s has status %s", item.Channel, item.Action, item.Status)
	}
	return ReleaseIssue{ID: fmt.Sprintf("installer-%s-%s", slug(item.Channel), slug(item.Action)), Category: "installer", Title: strings.ToUpper(item.Channel) + "/" + item.Action, Severity: sev, Status: IssueStatusOpen, Evidence: evidence, Remediation: "Run the installer scenario on a real Windows host and capture logs/screenshots for sign-off."}, true
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	repl := strings.NewReplacer(" ", "-", "/", "-", "_", "-", ".", "-", ":", "-", "(", "", ")", "", "[", "", "]", "", "'", "")
	s = repl.Replace(s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

func (l PreReleaseIssueList) JSON() ([]byte, error) { return json.MarshalIndent(l, "", "  ") }

func (l PreReleaseIssueList) Markdown() string {
	var b strings.Builder
	b.WriteString("# Windows Pre-Release Issue List\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", l.GeneratedAt.Format(time.RFC3339)))
	if strings.TrimSpace(l.Version) != "" {
		b.WriteString(fmt.Sprintf("Version: %s\n\n", l.Version))
	}
	state := "NOT READY"
	if l.ReleaseReady {
		state = "READY"
	}
	b.WriteString(fmt.Sprintf("Release state: %s\n\n", state))
	b.WriteString(fmt.Sprintf("Open issues: %d\nClosed issues: %d\n\n", l.OpenCount, l.ClosedCount))
	if len(l.Issues) == 0 {
		b.WriteString("No open pre-release issues.\n")
		return b.String()
	}
	for _, issue := range l.Issues {
		b.WriteString(fmt.Sprintf("- [%s/%s] %s (%s)\n", strings.ToUpper(string(issue.Severity)), strings.ToUpper(string(issue.Status)), issue.Title, issue.ID))
		if issue.Evidence != "" {
			b.WriteString("  - evidence: " + issue.Evidence + "\n")
		}
		if issue.Remediation != "" {
			b.WriteString("  - remediation: " + issue.Remediation + "\n")
		}
	}
	return b.String()
}
