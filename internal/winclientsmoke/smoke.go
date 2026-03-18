package winclientsmoke

import (
	"encoding/json"
	"strings"
)

type Scenario struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Goal     string   `json:"goal"`
	Steps    []string `json:"steps"`
	Expected []string `json:"expected"`
}

func DefaultExplorerSmoke() []Scenario {
	return []Scenario{
		{ID: "explorer-mount-visible", Name: "Explorer sees mount point", Goal: "Verify the mounted drive or directory appears in Explorer.", Steps: []string{"Start the mount from the Dashboard or tray menu.", "Open Explorer and locate the configured mount point."}, Expected: []string{"The mount point is visible.", "The volume label uses the configured prefix."}},
		{ID: "explorer-root-browse", Name: "Browse root directory", Goal: "Verify Explorer can enumerate the root directory and issue the common getattr/opendir/readdir sequence.", Steps: []string{"Open the mounted root.", "Sort by name and refresh once.", "Observe Diagnostics for callback and request-matrix summaries after refresh."}, Expected: []string{"Explorer lists entries without hanging.", "Refresh does not produce a binding/runtime error.", "The request matrix still shows only the known remaining gaps."}},
		{ID: "explorer-file-preview", Name: "Open a small file", Goal: "Verify a small file can be opened through the mounted view.", Steps: []string{"Double-click a known small text file.", "Close the viewer/editor after preview."}, Expected: []string{"The file contents load.", "The runtime remains mounted after close."}},
		{ID: "explorer-readonly-copy", Name: "Copy out from read-only mount", Goal: "Verify users can copy files from the mounted view.", Steps: []string{"Copy a file from the mount to Desktop or a temp folder.", "Open the copied file locally."}, Expected: []string{"Copy succeeds.", "Local copy matches expected contents."}},
		{ID: "explorer-diagnostics", Name: "Export diagnostics after smoke", Goal: "Verify diagnostics export works after Explorer interactions.", Steps: []string{"Run Export Diagnostics from the tray or Diagnostics page.", "Inspect the generated zip."}, Expected: []string{"The zip contains report.txt and report.json.", "The report includes binding/runtime summaries."}},
		{ID: "explorer-unmount-cleanup", Name: "Stop mount and verify cleanup", Goal: "Verify the mount disappears cleanly after stop.", Steps: []string{"Stop the mount from the Dashboard or tray.", "Refresh Explorer."}, Expected: []string{"The mount point is no longer active.", "The runtime returns to idle without an unhandled error."}},
	}
}
func Markdown(scenarios []Scenario) string {
	var b strings.Builder
	b.WriteString("# Windows Explorer Smoke\n\n")
	for _, s := range scenarios {
		b.WriteString("## " + s.ID + " — " + s.Name + "\n")
		b.WriteString(s.Goal + "\n\nSteps:\n")
		for _, step := range s.Steps {
			b.WriteString("- " + step + "\n")
		}
		b.WriteString("Expected:\n")
		for _, ex := range s.Expected {
			b.WriteString("- " + ex + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}
func JSON(scenarios []Scenario) ([]byte, error) { return json.MarshalIndent(scenarios, "", "  ") }
