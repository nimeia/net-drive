package winclientrelease

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ReleaseClosure struct {
	GeneratedAt  time.Time            `json:"generated_at"`
	Version      string               `json:"version"`
	Manifest     Manifest             `json:"manifest"`
	Validation   HostValidationRecord `json:"validation"`
	ReleaseReady bool                 `json:"release_ready"`
	Reasons      []string             `json:"reasons,omitempty"`
}

func NewReleaseClosure(manifest Manifest, validation HostValidationRecord) ReleaseClosure {
	closure := ReleaseClosure{GeneratedAt: time.Now().UTC(), Version: strings.TrimSpace(manifest.Version), Manifest: manifest, Validation: validation}
	closure.evaluate()
	return closure
}
func (c *ReleaseClosure) evaluate() {
	var reasons []string
	if c.Validation.Summary.Fail > 0 {
		reasons = append(reasons, "validation record still contains failed checks")
	}
	if c.Validation.Summary.NotRun > 0 {
		reasons = append(reasons, "validation record still contains not-run checks")
	}
	mustPass := [][2]string{{"msi", "install"}, {"msi", "upgrade"}, {"msi", "uninstall"}, {"exe", "portable-launch"}}
	for _, item := range mustPass {
		if !hasInstallerRunStatus(c.Validation, item[0], item[1], ValidationPass) {
			reasons = append(reasons, fmt.Sprintf("installer run %s/%s has not passed", item[0], item[1]))
		}
	}
	if c.Validation.CompletedAt == nil || strings.TrimSpace(c.Validation.CompletedBy) == "" {
		reasons = append(reasons, "validation record is not marked completed")
	}
	c.Reasons = reasons
	c.ReleaseReady = len(reasons) == 0
	if c.ReleaseReady {
		c.Manifest.FinalStatus = "ready"
	} else {
		c.Manifest.FinalStatus = "needs-validation"
	}
}
func hasInstallerRunStatus(record HostValidationRecord, channel, action string, status ValidationStatus) bool {
	for _, item := range record.InstallerRuns {
		if item.Channel == channel && item.Action == action && item.Status == status {
			return true
		}
	}
	return false
}
func (c ReleaseClosure) JSON() ([]byte, error) { return json.MarshalIndent(c, "", "  ") }
func (c ReleaseClosure) Markdown() string {
	var b strings.Builder
	b.WriteString("# Windows Release Closure\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", c.GeneratedAt.Format(time.RFC3339)))
	if strings.TrimSpace(c.Version) != "" {
		b.WriteString(fmt.Sprintf("Version: %s\n\n", c.Version))
	}
	state := "NOT READY"
	if c.ReleaseReady {
		state = "READY"
	}
	b.WriteString("Release state: " + state + "\n\n")
	b.WriteString("Validation summary: overall=" + string(c.Validation.Summary.Overall) + " pass=" + fmt.Sprint(c.Validation.Summary.Pass) + " warn=" + fmt.Sprint(c.Validation.Summary.Warn) + " fail=" + fmt.Sprint(c.Validation.Summary.Fail) + " not-run=" + fmt.Sprint(c.Validation.Summary.NotRun) + "\n\n")
	if len(c.Reasons) > 0 {
		b.WriteString("## Outstanding items\n")
		for _, reason := range c.Reasons {
			b.WriteString("- " + reason + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Artifacts\n")
	for _, a := range c.Manifest.Artifacts {
		b.WriteString(fmt.Sprintf("- %s (%s): %s\n", a.Name, a.Kind, a.Path))
	}
	b.WriteString("\n## Finalization\n- Backfill windows-host-validation-result-template.json on a real Windows host.\n- Regenerate this closure after MSI install/upgrade/uninstall and EXE portable launch all pass.\n")
	return b.String()
}
