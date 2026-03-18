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
		{ID: "explorer-root-browse", Name: "Browse root directory", Goal: "Verify Explorer can enumerate the root directory and issue the common getattr/opendir/readdir sequence.", Steps: []string{"Open the mounted root.", "Sort by name and refresh once.", "Observe Diagnostics for callback and request-matrix summaries after refresh."}, Expected: []string{"Explorer lists entries without hanging.", "Refresh does not produce a binding/runtime error."}},
		{ID: "explorer-file-preview", Name: "Open a small file", Goal: "Verify a small file can be opened through the mounted view.", Steps: []string{"Double-click a known small text file.", "Close the viewer/editor after preview."}, Expected: []string{"The file contents load.", "The runtime remains mounted after close."}},
		{ID: "explorer-readonly-copy", Name: "Copy out from read-only mount", Goal: "Verify users can copy files from the mounted view.", Steps: []string{"Copy a file from the mount to Desktop or a temp folder.", "Open the copied file locally."}, Expected: []string{"Copy succeeds.", "Local copy matches expected contents."}},
		{ID: "explorer-properties", Name: "Open Explorer properties/security", Goal: "Verify Explorer can query metadata and security descriptors for a mounted item.", Steps: []string{"Right-click a mounted file or directory and open Properties.", "Browse to the General and Security tabs if available."}, Expected: []string{"Explorer does not hang while loading properties.", "Diagnostics still report the security callbacks as ready."}},
		{ID: "explorer-create-denied", Name: "Create gesture is denied cleanly", Goal: "Verify Explorer receives a clean read-only denial when attempting to create a new file or folder.", Steps: []string{"From Explorer, try New > Text Document or create a folder inside the mount.", "Dismiss any retry prompt after the first denial."}, Expected: []string{"Explorer shows a read-only/access denied style result.", "Diagnostics show Create as explicitly blocked, not a gap."}},
		{ID: "explorer-write-denied", Name: "Write and metadata mutation are denied cleanly", Goal: "Verify write-side callbacks are explicitly blocked in read-only mode.", Steps: []string{"Open a mounted file in a text editor and try to save changes.", "Try to change metadata or trigger an overwrite-style save path."}, Expected: []string{"Save fails with a read-only/access denied style result.", "Diagnostics show Write/SetFileSize/SetBasicInfo/SetSecurity/Overwrite as blocked callbacks, not gaps."}},
		{ID: "explorer-rename-denied", Name: "Rename gesture is denied cleanly", Goal: "Verify Explorer receives a clean read-only denial for rename.", Steps: []string{"Select a mounted file or directory and press F2 or use Rename.", "Confirm the rename attempt once."}, Expected: []string{"Explorer shows an access denied/read-only style failure instead of hanging.", "Diagnostics show Rename as explicitly blocked, not a gap."}},
		{ID: "explorer-delete-denied", Name: "Delete gesture is denied cleanly", Goal: "Verify Explorer receives a clean read-only denial for delete and delete-on-close flows.", Steps: []string{"Select a mounted file and press Delete or use the context menu.", "If Explorer opens a preview/editor first, close the handle and retry once."}, Expected: []string{"Explorer shows an access denied/read-only style failure instead of hanging.", "Diagnostics still show CanDelete and SetDeleteOnClose as explicit read-only callbacks, not gaps."}},
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
