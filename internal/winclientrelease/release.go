package winclientrelease

import (
	"encoding/json"
	"fmt"
	"strings"

	"developer-mount/internal/winclientsmoke"
	"developer-mount/internal/winfsp"
)

type Artifact struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type Manifest struct {
	PackageName           string                    `json:"package_name"`
	Version               string                    `json:"version"`
	Artifacts             []Artifact                `json:"artifacts"`
	NativeCallbackSummary string                    `json:"native_callback_summary"`
	ExplorerMatrixSummary string                    `json:"explorer_matrix_summary"`
	ValidationTemplate    string                    `json:"validation_template,omitempty"`
	InstallerResults      []string                  `json:"installer_results,omitempty"`
	SmokeScenarios        []winclientsmoke.Scenario `json:"smoke_scenarios,omitempty"`
}

func NewManifest(version string, artifacts []Artifact, table winfsp.NativeCallbackTable, matrix winclientsmoke.RequestMatrix, smoke []winclientsmoke.Scenario) Manifest {
	return Manifest{PackageName: "developer-mount-windows-client", Version: strings.TrimSpace(version), Artifacts: artifacts, NativeCallbackSummary: table.Summary(), ExplorerMatrixSummary: matrix.Summary(), ValidationTemplate: "windows-host-validation-template.json", InstallerResults: []string{"msi-install", "msi-upgrade", "msi-uninstall", "exe-portable-launch"}, SmokeScenarios: smoke}
}
func (m Manifest) JSON() ([]byte, error) { return json.MarshalIndent(m, "", "  ") }
func (m Manifest) MarkdownChecklist() string {
	var b strings.Builder
	b.WriteString("# Windows Release Validation\n\n")
	b.WriteString(fmt.Sprintf("Package: %s\n\n", m.PackageName))
	b.WriteString(fmt.Sprintf("Version: %s\n\n", defaultValue(m.Version, "0.0.0-dev")))
	b.WriteString("Artifacts:\n")
	for _, a := range m.Artifacts {
		b.WriteString(fmt.Sprintf("- %s (%s): %s\n", a.Name, a.Kind, a.Path))
	}
	b.WriteString("\nNative callback summary: " + m.NativeCallbackSummary + "\n")
	b.WriteString("Explorer matrix summary: " + m.ExplorerMatrixSummary + "\n\n")
	b.WriteString("Validation steps:\n")
	b.WriteString("- Install WinFsp before starting dispatcher-v1 validation.\n")
	b.WriteString("- Run devmount-client-win32.exe -> Diagnostics -> Run Self-Check.\n")
	b.WriteString("- Confirm the native callback table and Explorer request matrix have no unexpected hot-path gaps.\n")
	b.WriteString("- Run the Windows Explorer smoke checklist on a Windows host.\n")
	b.WriteString("- Export diagnostics after smoke and archive the bundle with installer artifacts.\n")
	b.WriteString("- Backfill windows-host-validation-template.json with MSI install/upgrade/uninstall plus EXE launch results.\n")
	return b.String()
}
func defaultValue(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
