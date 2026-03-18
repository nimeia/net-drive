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
  release_closure = "windows-release-closure-template.json"
  issue_list = "windows-pre-release-issues-template.json"
}
$manifest | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 (Join-Path $release "release-manifest.json")
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
- Regenerate closure and issue list outputs after each backfill.
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
- [ ] explorer-diagnostics
- [ ] explorer-unmount-cleanup

## Recovery
- [ ] dirty-exit marker observed
- [ ] relaunch warning captured
- [ ] clean stop recorded
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-host-validation-template.md")

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
- Fill this file together with windows-host-validation-template.json after running real Windows-host validation.
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-installer-results-template.md")

@"
{
  "version": "$Version",
  "msi": {"install": "not-run", "upgrade": "not-run", "uninstall": "not-run"},
  "exe": {"portable_launch": "not-run"},
  "notes": [
    "Backfill this file after running the real Windows host installer checks."
  ]
}
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-installer-results-template.json")

Copy-Item (Join-Path $release "windows-host-validation-template.json") (Join-Path $release "windows-host-validation-result-template.json") -Force
Copy-Item (Join-Path $release "windows-host-validation-template.md") (Join-Path $release "windows-host-validation-result-template.md") -Force

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
    {"scenario_id": "explorer-mount-visible", "status": "not-run", "notes": ""}
  ],
  "installer_checklist": [
    {"item": "MSI install succeeded", "status": "not-run", "notes": ""},
    {"item": "EXE/portable bundle launch succeeded", "status": "not-run", "notes": ""}
  ],
  "recovery_checklist": [
    {"item": "Next launch reports recovery warning", "status": "not-run", "notes": ""}
  ],
  "installer_runs": [
    {"channel": "msi", "action": "install", "status": "not-run", "version_to": "$Version", "log_path": ""},
    {"channel": "exe", "action": "portable-launch", "status": "not-run", "version_to": "$Version", "log_path": ""}
  ],
  "notes": [
    "Fill only the fields observed during the current Windows host validation round."
  ]
}
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-host-backfill-patch-template.json")
@"
# Windows Host Backfill Patch Template

Version: $Version

Use `windows-host-backfill-patch-template.json` to record the first-pass Windows host results observed on the test machine. Merge the patch into `windows-host-validation-result-template.json`, then regenerate closure and issue-list outputs.
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-host-backfill-patch-template.md")

@"
# Windows Release Closure Template

Version: $Version
State: NOT-READY

## Outstanding items
- [ ] Backfill windows-host-validation-result-template.json
- [ ] MSI install passed
- [ ] MSI upgrade passed
- [ ] MSI uninstall passed
- [ ] EXE portable launch passed
- [ ] Validation marked completed
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-release-closure-template.md")

@"
{
  "version": "$Version",
  "release_ready": false,
  "reasons": [
    "validation record still contains not-run checks",
    "installer run msi/install has not passed",
    "installer run msi/upgrade has not passed",
    "installer run msi/uninstall has not passed",
    "installer run exe/portable-launch has not passed",
    "validation record is not marked completed"
  ]
}
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-release-closure-template.json")

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

- [BLOCKER/OPEN] Mount becomes visible in Explorer
  - evidence: scenario not yet executed on a Windows host
  - remediation: Run the scenario and backfill the result.
"@ | Set-Content -Encoding UTF8 (Join-Path $release "windows-pre-release-issues-template.md")
Write-Host "Prepared Windows release assets at $release"
