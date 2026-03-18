$ErrorActionPreference = "Stop"
param(
  [string]$ValidationResultJson,
  [string]$PatchJson,
  [string]$OutJson = "",
  [string]$OutMarkdown = "",
  [string]$CompletedBy = ""
)
if (-not (Test-Path $ValidationResultJson)) { throw "Missing validation result json: $ValidationResultJson" }
if (-not (Test-Path $PatchJson)) { throw "Missing patch json: $PatchJson" }
$validation = Get-Content $ValidationResultJson -Raw | ConvertFrom-Json -Depth 12
$patch = Get-Content $PatchJson -Raw | ConvertFrom-Json -Depth 12
if (-not $OutJson) { $OutJson = $ValidationResultJson }
if (-not $OutMarkdown) { $OutMarkdown = [System.IO.Path]::ChangeExtension($OutJson, '.md') }
if ($patch.environment) {
  foreach ($prop in @('source','machine','os_version','winfsp_version','package_channel','diagnostics_bundle','installer_log_dir')) {
    if ($patch.environment.$prop) { $validation.environment.$prop = $patch.environment.$prop }
  }
  if ($patch.environment.notes) {
    if (-not $validation.environment.notes) { $validation.environment | Add-Member -NotePropertyName notes -NotePropertyValue @() -Force }
    $validation.environment.notes += $patch.environment.notes
  }
}
foreach ($item in @($patch.explorer_scenarios)) {
  $hit = $validation.explorer_scenarios | Where-Object { $_.scenario_id -eq $item.scenario_id } | Select-Object -First 1
  if ($hit) { $hit.status = $item.status; $hit.notes = $item.notes }
}
foreach ($item in @($patch.installer_checklist)) {
  $hit = $validation.installer_checklist | Where-Object { $_.item -eq $item.item } | Select-Object -First 1
  if ($hit) { $hit.status = $item.status; $hit.notes = $item.notes }
}
foreach ($item in @($patch.recovery_checklist)) {
  $hit = $validation.recovery_checklist | Where-Object { $_.item -eq $item.item } | Select-Object -First 1
  if ($hit) { $hit.status = $item.status; $hit.notes = $item.notes }
}
foreach ($item in @($patch.installer_runs)) {
  $hit = $validation.installer_runs | Where-Object { $_.channel -eq $item.channel -and $_.action -eq $item.action } | Select-Object -First 1
  if ($hit) {
    $hit.status = $item.status; $hit.notes = $item.notes
    if ($item.version_from) { $hit.version_from = $item.version_from }
    if ($item.version_to) { $hit.version_to = $item.version_to }
    if ($item.log_path) { $hit.log_path = $item.log_path }
  }
}
if ($patch.notes) {
  if (-not $validation.notes) { $validation | Add-Member -NotePropertyName notes -NotePropertyValue @() -Force }
  $validation.notes += $patch.notes
}
$all = @(); if ($validation.explorer_scenarios) { $all += $validation.explorer_scenarios }; if ($validation.installer_checklist) { $all += $validation.installer_checklist }; if ($validation.recovery_checklist) { $all += $validation.recovery_checklist }; if ($validation.installer_runs) { $all += $validation.installer_runs }
$summary = [ordered]@{ not_run = ($all | Where-Object { $_.status -eq 'not-run' }).Count; pass = ($all | Where-Object { $_.status -eq 'pass' }).Count; warn = ($all | Where-Object { $_.status -eq 'warn' }).Count; fail = ($all | Where-Object { $_.status -eq 'fail' }).Count; overall = 'not-run' }
if ($summary.fail -gt 0) { $summary.overall = 'fail' } elseif ($summary.warn -gt 0) { $summary.overall = 'warn' } elseif ($summary.pass -gt 0 -and $summary.not_run -eq 0) { $summary.overall = 'pass' } elseif ($summary.pass -gt 0) { $summary.overall = 'warn' }
$validation.summary = $summary
$completedByValue = if ($CompletedBy) { $CompletedBy } elseif ($patch.completed_by) { $patch.completed_by } else { $validation.completed_by }
if ($completedByValue) { $validation.completed_by = $completedByValue; $validation.completed_at = (Get-Date).ToUniversalTime().ToString('s') + 'Z' }
$validation | ConvertTo-Json -Depth 12 | Set-Content -Encoding UTF8 $OutJson
@"
# Windows Host Validation Record

Version: $($validation.version)
Completed by: $($validation.completed_by)
Summary: overall=$($validation.summary.overall) pass=$($validation.summary.pass) warn=$($validation.summary.warn) fail=$($validation.summary.fail) not-run=$($validation.summary.not_run)

Use finalize-windows-release.ps1 to regenerate closure and issue-list outputs.
"@ | Set-Content -Encoding UTF8 $OutMarkdown
Write-Host "Updated validation record: $OutJson"
