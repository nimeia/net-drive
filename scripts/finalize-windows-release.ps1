$ErrorActionPreference = "Stop"
param(
  [string]$ReleaseDir = (Join-Path (Join-Path (Split-Path -Parent $MyInvocation.MyCommand.Path) "..") "dist\windows-release"),
  [string]$ValidationResultJson = "",
  [string]$CompletedBy = ""
)
if (-not $ValidationResultJson) { $ValidationResultJson = Join-Path $ReleaseDir "windows-host-validation-result-template.json" }
if (-not (Test-Path $ValidationResultJson)) { throw "Missing validation result json: $ValidationResultJson" }
$validation = Get-Content $ValidationResultJson -Raw | ConvertFrom-Json -Depth 12
$reasons = New-Object System.Collections.Generic.List[string]
$all = @(); if ($validation.explorer_scenarios) { $all += $validation.explorer_scenarios }; if ($validation.installer_checklist) { $all += $validation.installer_checklist }; if ($validation.recovery_checklist) { $all += $validation.recovery_checklist }; if ($validation.installer_runs) { $all += $validation.installer_runs }
$notRun = ($all | Where-Object { $_.status -eq "not-run" }).Count
$fail = ($all | Where-Object { $_.status -eq "fail" }).Count
if ($fail -gt 0) { $reasons.Add("validation record still contains failed checks") }
if ($notRun -gt 0) { $reasons.Add("validation record still contains not-run checks") }
foreach ($pair in @(@("msi","install"),@("msi","upgrade"),@("msi","uninstall"),@("exe","portable-launch"))) {
  $hit = $validation.installer_runs | Where-Object { $_.channel -eq $pair[0] -and $_.action -eq $pair[1] -and $_.status -eq "pass" }
  if (-not $hit) { $reasons.Add("installer run $($pair[0])/$($pair[1]) has not passed") }
}
if (-not $validation.completed_at -or -not (($CompletedBy) -or $validation.completed_by)) { $reasons.Add("validation record is not marked completed") }
$ready = ($reasons.Count -eq 0)
$version = if ($validation.version) { $validation.version } else { "0.0.0-dev" }
$completedByValue = if ($CompletedBy) { $CompletedBy } else { $validation.completed_by }
$closure = [ordered]@{ generated_at = (Get-Date).ToUniversalTime().ToString("s") + "Z"; version = $version; release_ready = $ready; reasons = @($reasons); completed_by = $completedByValue; validation_result = [System.IO.Path]::GetFileName($ValidationResultJson) }
$closure | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir "windows-release-closure.json")
@"
# Windows Release Closure

Version: $version
Release ready: $ready
Completed by: $completedByValue
Validation result: $([System.IO.Path]::GetFileName($ValidationResultJson))

## Outstanding items
$(if ($reasons.Count -eq 0) { "- none" } else { ($reasons | ForEach-Object { "- " + $_ }) -join "`r`n" })
"@ | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir "windows-release-closure.md")

$issues = New-Object System.Collections.Generic.List[object]
foreach ($scenario in $validation.explorer_scenarios) {
  if ($scenario.status -ne "pass") {
    $sev = if ($scenario.status -eq "warn") { "warning" } else { "blocker" }
    $evidence = if ($scenario.notes) { $scenario.notes } else { "scenario not yet executed on a Windows host" }
    $issues.Add([ordered]@{ id = "scenario-$($scenario.scenario_id)"; category = "explorer"; title = $scenario.name; severity = $sev; status = "open"; evidence = $evidence; remediation = "Run the scenario and backfill the result." })
  }
}
foreach ($item in $validation.installer_checklist) {
  if ($item.status -ne "pass") {
    $sev = if ($item.status -eq "warn") { "warning" } else { "blocker" }
    $evidence = if ($item.notes) { $item.notes } else { "installer checklist item has not passed" }
    $issues.Add([ordered]@{ id = "installer-" + ($item.item -replace '[^A-Za-z0-9]+','-').ToLower().Trim('-'); category = "installer"; title = $item.item; severity = $sev; status = "open"; evidence = $evidence; remediation = "Backfill the installer checklist item with a stable Windows-host result." })
  }
}
foreach ($item in $validation.recovery_checklist) {
  if ($item.status -ne "pass") {
    $sev = if ($item.status -eq "warn") { "warning" } else { "blocker" }
    $evidence = if ($item.notes) { $item.notes } else { "recovery checklist item has not passed" }
    $issues.Add([ordered]@{ id = "recovery-" + ($item.item -replace '[^A-Za-z0-9]+','-').ToLower().Trim('-'); category = "recovery"; title = $item.item; severity = $sev; status = "open"; evidence = $evidence; remediation = "Backfill the recovery checklist item with a stable Windows-host result." })
  }
}
foreach ($item in $validation.installer_runs) {
  if ($item.status -ne "pass") {
    $sev = if ($item.status -eq "warn") { "warning" } else { "blocker" }
    $evidence = if ($item.notes) { $item.notes } else { "installer run has not passed" }
    $issues.Add([ordered]@{ id = "run-$($item.channel)-$($item.action)"; category = "installer-run"; title = "$($item.channel)/$($item.action)"; severity = $sev; status = "open"; evidence = $evidence; remediation = "Run the installer scenario on a real Windows host and capture logs for sign-off." })
  }
}
$issueList = [ordered]@{ generated_at = (Get-Date).ToUniversalTime().ToString("s") + "Z"; version = $version; release_ready = $ready; open_count = $issues.Count; closed_count = 0; issues = @($issues) }
$issueList | ConvertTo-Json -Depth 10 | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir "windows-pre-release-issues.json")
@"
# Windows Pre-Release Issue List

Version: $version
Release ready: $ready
Open issues: $($issues.Count)

$(if ($issues.Count -eq 0) { "No open pre-release issues." } else { ($issues | ForEach-Object { "- [" + $_.severity.ToUpper() + "/OPEN] " + $_.title + "`r`n  - evidence: " + $_.evidence + "`r`n  - remediation: " + $_.remediation }) -join "`r`n" })
"@ | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir "windows-pre-release-issues.md")

Write-Host "Wrote release closure and issue list to $ReleaseDir"
