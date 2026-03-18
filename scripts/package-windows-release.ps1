$ErrorActionPreference = "Stop"
param([string]$Version = "0.1.0")
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Split-Path -Parent $scriptDir
$dist = Join-Path $root "dist"
$release = Join-Path $dist "windows-release"
New-Item -ItemType Directory -Force -Path $release | Out-Null
& (Join-Path $scriptDir "build.ps1")
if ($LASTEXITCODE -ne 0) { throw "build.ps1 failed" }
& (Join-Path $scriptDir "package-windows-msi.ps1") -Version $Version
& (Join-Path $scriptDir "package-windows-exe.ps1") -Version $Version
$manifest = [ordered]@{
  package_name = "developer-mount-windows-client"
  version = $Version
  generated_at = (Get-Date).ToString("s")
  artifacts = @(
    @{ name = "devmount-client-win32.exe"; kind = "exe"; path = "dist/devmount-client-win32.exe" },
    @{ name = "devmount-winfsp.exe"; kind = "exe"; path = "dist/devmount-winfsp.exe" },
    @{ name = "devmount-server.exe"; kind = "exe"; path = "dist/devmount-server.exe" },
    @{ name = "msi"; kind = "installer"; path = "dist/windows-release/msi" },
    @{ name = "portable-zip"; kind = "portable"; path = "dist/windows-release/exe" }
  )
  validation_template = "windows-host-validation-template.json"
  validation_result = "windows-host-validation-result-template.json"
  backfill_patch = "windows-host-backfill-patch-template.json"
  release_closure = "windows-release-closure.json"
  issue_list = "windows-pre-release-issues.json"
  fix_plan = "windows-first-pass-fix-plan.json"
  release_candidate = "windows-release-candidate.json"
  final_status = "needs-validation"
}
$manifest | ConvertTo-Json -Depth 16 | Set-Content -Encoding UTF8 (Join-Path $release "release-manifest.json")
@"
# Windows Release Validation

Version: $Version

## Artifacts
- dist/devmount-client-win32.exe
- dist/devmount-winfsp.exe
- dist/devmount-server.exe
- dist/windows-release/msi
- dist/windows-release/exe

## Validation
- Install WinFsp before dispatcher-v1 validation.
- Run devmount-client-win32.exe and Diagnostics -> Run Self-Check.
- Confirm the native callback table and Explorer request matrix summaries are present.
- Run the Explorer smoke checklist on a Windows host.
- Export diagnostics after smoke and archive the bundle.
- Apply windows-host-backfill-patch-template.json after each validation round.
- Merge windows-installer-results-template.json after each install/upgrade/uninstall round.
- Regenerate closure, issue list, fix plan, and RC outputs after each backfill.
"@ | Set-Content -Encoding UTF8 (Join-Path $release "release-validation.md")

@"
{
  "generated_at": "$(Get-Date -Format s)",
  "version": "$Version",
  "status": "not-run",
  "notes": [
    "Fill this record on a real Windows host after running MSI/EXE validation and Explorer smoke."
  ]
}
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-host-validation-template.json")
@"
# Windows Host Validation Template

Version: $Version
Status: NOT-RUN

## Installer
- [ ] WinFsp installed and version captured
- [ ] MSI install succeeded
- [ ] EXE/portable launch succeeded
- [ ] Upgrade path verified
- [ ] Uninstall path verified

## Explorer smoke
- [ ] explorer-mount-visible
- [ ] explorer-root-browse
- [ ] explorer-file-preview
- [ ] explorer-readonly-copy
- [ ] explorer-properties
- [ ] explorer-delete-denied
- [ ] explorer-create-denied
- [ ] explorer-write-denied
- [ ] explorer-rename-denied
- [ ] explorer-diagnostics
- [ ] explorer-unmount-cleanup

## Recovery
- [ ] dirty-exit marker observed
- [ ] relaunch warning captured
- [ ] clean stop recorded
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-host-validation-template.md")
Copy-Item (Join-Path $release "windows-host-validation-template.json") (Join-Path $release "windows-host-validation-result-template.json") -Force
Copy-Item (Join-Path $release "windows-host-validation-template.md") (Join-Path $release "windows-host-validation-result-template.md") -Force

@"
{
  "version": "$Version",
  "log_dir": "",
  "msi": {
    "install": {"status": "not-run", "version_to": "$Version", "log_path": "", "notes": ""},
    "upgrade": {"status": "not-run", "version_to": "$Version", "log_path": "", "notes": ""},
    "uninstall": {"status": "not-run", "version_to": "$Version", "log_path": "", "notes": ""}
  },
  "exe": {
    "portable_launch": {"status": "not-run", "version_to": "$Version", "log_path": "", "notes": ""}
  },
  "notes": [
    "Backfill this file after running the real Windows host installer checks."
  ]
}
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-installer-results-template.json")
@"
# Windows Installer Results Template

Version: $Version

## MSI
- [ ] install
- [ ] upgrade
- [ ] uninstall

## EXE
- [ ] portable launch

## Notes
- Fill this file together with windows-host-validation-result-template.json after running real Windows-host validation.
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-installer-results-template.md")

@"
{
  "environment": {
    "source": "real-windows-host",
    "machine": "",
    "os_version": "",
    "winfsp_version": "",
    "package_channel": "msi,exe",
    "diagnostics_bundle": "",
    "installer_log_dir": ""
  },
  "explorer_scenarios": [
    {"scenario_id": "explorer-mount-visible", "status": "not-run", "notes": ""},
    {"scenario_id": "explorer-root-browse", "status": "not-run", "notes": ""},
    {"scenario_id": "explorer-file-preview", "status": "not-run", "notes": ""},
    {"scenario_id": "explorer-readonly-copy", "status": "not-run", "notes": ""},
    {"scenario_id": "explorer-properties", "status": "not-run", "notes": ""},
    {"scenario_id": "explorer-delete-denied", "status": "not-run", "notes": ""},
    {"scenario_id": "explorer-create-denied", "status": "not-run", "notes": ""},
    {"scenario_id": "explorer-write-denied", "status": "not-run", "notes": ""},
    {"scenario_id": "explorer-rename-denied", "status": "not-run", "notes": ""},
    {"scenario_id": "explorer-diagnostics", "status": "not-run", "notes": ""},
    {"scenario_id": "explorer-unmount-cleanup", "status": "not-run", "notes": ""}
  ],
  "installer_checklist": [
    {"item": "WinFsp installed and version captured", "status": "not-run", "notes": ""},
    {"item": "MSI install succeeded", "status": "not-run", "notes": ""},
    {"item": "EXE/portable bundle launch succeeded", "status": "not-run", "notes": ""},
    {"item": "Shortcuts / tray / Dashboard available after install", "status": "not-run", "notes": ""},
    {"item": "Upgrade path preserves config", "status": "not-run", "notes": ""},
    {"item": "Uninstall removes binaries and leaves expected user data policy", "status": "not-run", "notes": ""}
  ],
  "recovery_checklist": [
    {"item": "Dirty-exit marker written during forced termination", "status": "not-run", "notes": ""},
    {"item": "Next launch reports recovery warning", "status": "not-run", "notes": ""},
    {"item": "Clean stop clears recovery marker state", "status": "not-run", "notes": ""}
  ],
  "installer_runs": [
    {"channel": "msi", "action": "install", "status": "not-run", "version_to": "$Version", "log_path": "", "notes": ""},
    {"channel": "msi", "action": "upgrade", "status": "not-run", "version_to": "$Version", "log_path": "", "notes": ""},
    {"channel": "msi", "action": "uninstall", "status": "not-run", "version_to": "$Version", "log_path": "", "notes": ""},
    {"channel": "exe", "action": "portable-launch", "status": "not-run", "version_to": "$Version", "log_path": "", "notes": ""}
  ],
  "notes": [
    "Fill only the fields observed during the current Windows host validation round."
  ]
}
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-host-backfill-patch-template.json")
@"
# Windows Host Backfill Patch Template

Version: $Version

Use `windows-host-backfill-patch-template.json` to record the first-pass Windows host results observed on the test machine. Merge the patch into `windows-host-validation-result-template.json`, then regenerate closure, issue-list, fix-plan, and RC outputs.
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-host-backfill-patch-template.md")

@"
{
  "version": "$Version",
  "release_ready": false,
  "reasons": [
    "validation record still contains not-run checks"
  ]
}
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-release-closure-template.json")
@"
# Windows Release Closure Template

Version: $Version
State: NOT-READY
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-release-closure-template.md")

@"
{
  "version": "$Version",
  "release_ready": false,
  "open_count": 1,
  "closed_count": 0,
  "issues": [
    {
      "id": "scenario-explorer-mount-visible",
      "category": "explorer",
      "title": "Mount becomes visible in Explorer",
      "severity": "blocker",
      "status": "open",
      "evidence": "scenario not yet executed on a Windows host",
      "remediation": "Run the scenario and backfill the result."
    }
  ]
}
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-pre-release-issues-template.json")
@"
# Windows Pre-Release Issue List Template

Version: $Version
Release state: NOT READY
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-pre-release-issues-template.md")

@"
{
  "version": "$Version",
  "blockers": 1,
  "warnings": 0,
  "infos": 0,
  "items": [
    {
      "id": "scenario-explorer-mount-visible",
      "severity": "blocker",
      "category": "explorer",
      "title": "Mount becomes visible in Explorer",
      "evidence": "scenario not yet executed on a Windows host",
      "remediation": "Run the scenario and backfill the result.",
      "suggested_area": "winfsp-native-callbacks",
      "suggested_files": ["internal/winfsp/callbacks.go", "internal/winclientsmoke/request_matrix.go"]
    }
  ]
}
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-first-pass-fix-plan-template.json")
@"
# Windows First-Pass Fix Plan Template

Version: $Version
Blockers: 1
Warnings: 0
Infos: 0
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-first-pass-fix-plan-template.md")

@"
{
  "version": "$Version",
  "channel": "rc",
  "release_ready": false,
  "final_status": "needs-validation"
}
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-release-candidate-template.json")
@"
# Windows Release Candidate Template

Version: $Version
Channel: rc
Final status: needs-validation
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-release-candidate-template.md")
Write-Host "Prepared Windows release assets at $release"
