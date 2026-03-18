$ErrorActionPreference = "Stop"
param(
  [string]$ReleaseDir,
  [string]$Version = ""
)
if (-not (Test-Path $ReleaseDir)) { throw "Missing release dir: $ReleaseDir" }
$finalDir = Join-Path (Split-Path -Parent $ReleaseDir) "windows-final"
New-Item -ItemType Directory -Force -Path $finalDir | Out-Null
foreach ($name in @(
  'release-manifest.json',
  'windows-host-validation-result-template.json',
  'windows-validation-intake-report.json',
  'windows-release-closure.json',
  'windows-pre-release-issues.json',
  'windows-first-pass-fix-plan.json',
  'windows-release-candidate.json',
  'windows-final-release.json',
  'windows-final-release.md',
  'windows-final-signoff.md'
)) {
  $src = Join-Path $ReleaseDir $name
  if (Test-Path $src) { Copy-Item $src (Join-Path $finalDir $name) -Force }
}
if (-not $Version) {
  $manifestPath = Join-Path $ReleaseDir 'release-manifest.json'
  if (Test-Path $manifestPath) { $Version = (Get-Content $manifestPath -Raw | ConvertFrom-Json).version }
}
$finalPath = Join-Path $finalDir 'windows-final-release.json'
$finalStatus = 'unknown'
$publishReady = $false
if (Test-Path $finalPath) {
  $final = Get-Content $finalPath -Raw | ConvertFrom-Json -Depth 16
  $finalStatus = $final.final_status
  $publishReady = [bool]$final.publish_ready
}
@"
# Windows Final Release Package

Version: $Version
Final status: $finalStatus
Publish ready: $publishReady

This directory contains the finalized Windows release metadata generated from the latest Windows-host backfill:
- release-manifest.json
- windows-host-validation-result-template.json
- windows-validation-intake-report.json
- windows-release-closure.json
- windows-pre-release-issues.json
- windows-first-pass-fix-plan.json
- windows-release-candidate.json
- windows-final-release.json
- windows-final-signoff.md
"@ | Set-Content -Encoding UTF8 (Join-Path $finalDir 'final-package-notes.md')
Write-Host "Prepared Windows final release assets at $finalDir"
