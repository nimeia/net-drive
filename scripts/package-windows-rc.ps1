$ErrorActionPreference = "Stop"
param(
  [string]$ReleaseDir,
  [string]$Version = ""
)
if (-not (Test-Path $ReleaseDir)) { throw "Missing release dir: $ReleaseDir" }
$rcDir = Join-Path (Split-Path -Parent $ReleaseDir) "windows-rc"
New-Item -ItemType Directory -Force -Path $rcDir | Out-Null
foreach ($name in @(
  'release-manifest.json',
  'windows-host-validation-result-template.json',
  'windows-release-closure.json',
  'windows-pre-release-issues.json',
  'windows-first-pass-fix-plan.json',
  'windows-release-candidate.json',
  'windows-release-candidate.md'
)) {
  $src = Join-Path $ReleaseDir $name
  if (Test-Path $src) { Copy-Item $src (Join-Path $rcDir $name) -Force }
}
if (-not $Version) {
  $manifestPath = Join-Path $ReleaseDir 'release-manifest.json'
  if (Test-Path $manifestPath) { $Version = (Get-Content $manifestPath -Raw | ConvertFrom-Json).version }
}
@"
# Windows RC Packaging

Version: $Version

This directory contains the finalized RC metadata generated from the latest Windows host backfill:
- release-manifest.json
- windows-host-validation-result-template.json
- windows-release-closure.json
- windows-pre-release-issues.json
- windows-first-pass-fix-plan.json
- windows-release-candidate.json
"@ | Set-Content -Encoding UTF8 (Join-Path $rcDir 'rc-package-notes.md')
Write-Host "Prepared Windows RC assets at $rcDir"
