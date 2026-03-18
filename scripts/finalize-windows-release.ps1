$ErrorActionPreference = "Stop"
param(
  [string]$ReleaseDir,
  [string]$ValidationResultJson,
  [string]$CompletedBy = ""
)
if (-not (Test-Path $ReleaseDir)) { throw "Missing release dir: $ReleaseDir" }
if (-not (Test-Path $ValidationResultJson)) { throw "Missing validation result json: $ValidationResultJson" }
$manifestPath = Join-Path $ReleaseDir "release-manifest.json"
if (-not (Test-Path $manifestPath)) { throw "Missing release manifest: $manifestPath" }
$manifest = Get-Content $manifestPath -Raw | ConvertFrom-Json -Depth 16
$validation = Get-Content $ValidationResultJson -Raw | ConvertFrom-Json -Depth 16
if ($CompletedBy) {
  $validation.completed_by = $CompletedBy
  $validation.completed_at = (Get-Date).ToUniversalTime().ToString('s') + 'Z'
}
$version = if ($manifest.version) { $manifest.version } else { $validation.version }
$reasons = New-Object System.Collections.Generic.List[string]
if (-not $validation.completed_at) { $reasons.Add('validation record is not marked completed') }
foreach ($item in @($validation.explorer_scenarios)) { if ($item.status -ne 'pass') { $reasons.Add("explorer scenario $($item.scenario_id) has status $($item.status)") } }
foreach ($item in @($validation.installer_checklist)) { if ($item.status -ne 'pass') { $reasons.Add("installer checklist '$($item.item)' has status $($item.status)") } }
foreach ($item in @($validation.recovery_checklist)) { if ($item.status -ne 'pass') { $reasons.Add("recovery checklist '$($item.item)' has status $($item.status)") } }
foreach ($item in @($validation.installer_runs)) { if ($item.status -ne 'pass') { $reasons.Add("installer run $($item.channel)/$($item.action) has status $($item.status)") } }
$ready = ($reasons.Count -eq 0)
$closure = [ordered]@{ generated_at = (Get-Date).ToUniversalTime().ToString('s') + 'Z'; version = $version; release_ready = $ready; reasons = @($reasons) }
$closure | ConvertTo-Json -Depth 16 | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir "windows-release-closure.json")
@"
# Windows Release Closure

Version: $version
Release ready: $ready

$(if ($reasons.Count -eq 0) { "No outstanding closure reasons." } else { ($reasons | ForEach-Object { "- " + $_ }) -join "`r`n" })
"@ | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir "windows-release-closure.md")

$issues = New-Object System.Collections.Generic.List[object]
foreach ($item in @($validation.explorer_scenarios)) {
  if ($item.status -ne 'pass') {
    $sev = if ($item.status -eq 'warn') { 'warning' } else { 'blocker' }
    $evidence = if ($item.notes) { $item.notes } else { 'scenario not yet executed on a Windows host' }
    $issues.Add([ordered]@{ id = "scenario-$($item.scenario_id)"; category = 'explorer'; title = $item.name; severity = $sev; status = 'open'; evidence = $evidence; remediation = 'Run the Explorer scenario and backfill the result.' })
  }
}
foreach ($item in @($validation.installer_checklist)) {
  if ($item.status -ne 'pass') {
    $sev = if ($item.status -eq 'warn') { 'warning' } else { 'blocker' }
    $evidence = if ($item.notes) { $item.notes } else { 'installer checklist item has not passed' }
    $issues.Add([ordered]@{ id = "installer-" + ($item.item -replace '[^A-Za-z0-9]+','-').ToLower().Trim('-'); category = 'installer'; title = $item.item; severity = $sev; status = 'open'; evidence = $evidence; remediation = 'Backfill the installer checklist item with a stable Windows-host result.' })
  }
}
foreach ($item in @($validation.recovery_checklist)) {
  if ($item.status -ne 'pass') {
    $sev = if ($item.status -eq 'warn') { 'warning' } else { 'blocker' }
    $evidence = if ($item.notes) { $item.notes } else { 'recovery checklist item has not passed' }
    $issues.Add([ordered]@{ id = "recovery-" + ($item.item -replace '[^A-Za-z0-9]+','-').ToLower().Trim('-'); category = 'recovery'; title = $item.item; severity = $sev; status = 'open'; evidence = $evidence; remediation = 'Backfill the recovery checklist item with a stable Windows-host result.' })
  }
}
foreach ($item in @($validation.installer_runs)) {
  if ($item.status -ne 'pass') {
    $sev = if ($item.status -eq 'warn') { 'warning' } else { 'blocker' }
    $evidence = if ($item.notes) { $item.notes } else { 'installer run has not passed' }
    $issues.Add([ordered]@{ id = "run-$($item.channel)-$($item.action)"; category = 'installer-run'; title = "$($item.channel)/$($item.action)"; severity = $sev; status = 'open'; evidence = $evidence; remediation = 'Run the installer scenario on a real Windows host and capture logs for sign-off.' })
  }
}
$issueList = [ordered]@{ generated_at = (Get-Date).ToUniversalTime().ToString('s') + 'Z'; version = $version; release_ready = $ready; open_count = $issues.Count; closed_count = 0; issues = @($issues) }
$issueList | ConvertTo-Json -Depth 16 | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir "windows-pre-release-issues.json")
@"
# Windows Pre-Release Issue List

Version: $version
Release ready: $ready
Open issues: $($issues.Count)

$(if ($issues.Count -eq 0) { "No open pre-release issues." } else { ($issues | ForEach-Object { "- [" + $_.severity.ToUpper() + "/OPEN] " + $_.title + "`r`n  - evidence: " + $_.evidence + "`r`n  - remediation: " + $_.remediation }) -join "`r`n" })
"@ | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir "windows-pre-release-issues.md")

$fixes = New-Object System.Collections.Generic.List[object]
foreach ($issue in $issues) {
  $area = switch ($issue.category) {
    'explorer' { 'winfsp-native-callbacks' }
    'installer' { 'windows-release-packaging' }
    'installer-run' { 'windows-installer-validation' }
    'recovery' { 'winclient-recovery' }
    default { 'release-closure' }
  }
  $files = switch ($issue.category) {
    'explorer' { @('internal/winfsp/callbacks.go','internal/winclientsmoke/request_matrix.go') }
    'installer' { @('scripts/package-windows-release.ps1','scripts/package-windows-installer.ps1') }
    'installer-run' { @('scripts/backfill-windows-validation.ps1','scripts/finalize-windows-release.ps1') }
    'recovery' { @('internal/winclientrecovery/recovery.go','internal/winclientdiag/diag.go') }
    default { @('internal/winclientrelease/closure.go','internal/winclientrelease/issues.go') }
  }
  $fixes.Add([ordered]@{ id = $issue.id; severity = $issue.severity; category = $issue.category; title = $issue.title; evidence = $issue.evidence; remediation = $issue.remediation; suggested_area = $area; suggested_files = @($files) })
}
$fixPlan = [ordered]@{ generated_at = (Get-Date).ToUniversalTime().ToString('s') + 'Z'; version = $version; blockers = (@($issues | Where-Object severity -eq 'blocker')).Count; warnings = (@($issues | Where-Object severity -eq 'warning')).Count; infos = (@($issues | Where-Object severity -eq 'info')).Count; items = @($fixes) }
$fixPlan | ConvertTo-Json -Depth 16 | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir "windows-first-pass-fix-plan.json")
@"
# Windows First-Pass Fix Plan

Version: $version
Blockers: $((@($issues | Where-Object severity -eq 'blocker')).Count)
Warnings: $((@($issues | Where-Object severity -eq 'warning')).Count)
Infos: $((@($issues | Where-Object severity -eq 'info')).Count)

$(if ($fixes.Count -eq 0) { "No fixes queued." } else { ($fixes | ForEach-Object { "- [" + $_.severity.ToUpper() + "] " + $_.title + "`r`n  - area: " + $_.suggested_area + "`r`n  - evidence: " + $_.evidence + "`r`n  - remediation: " + $_.remediation + "`r`n  - suggested_files: " + ($_.suggested_files -join ', ') }) -join "`r`n" })
"@ | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir "windows-first-pass-fix-plan.md")

$status = if ($ready -and $issues.Count -eq 0 -and $validation.summary.overall -eq 'pass') { 'rc-ready' } elseif ($issues.Count -gt 0 -or $validation.summary.overall -eq 'warn') { 'needs-attention' } else { 'blocked' }
$rc = [ordered]@{ generated_at = (Get-Date).ToUniversalTime().ToString('s') + 'Z'; version = $version; channel = 'rc'; release_ready = $ready; manifest_path = 'release-manifest.json'; validation_path = 'windows-host-validation-result-template.json'; closure_path = 'windows-release-closure.json'; issue_list_path = 'windows-pre-release-issues.json'; fix_plan_path = 'windows-first-pass-fix-plan.json'; artifact_count = (@($manifest.artifacts)).Count; open_issues = $issues.Count; final_status = $status; closure_reasons = @($reasons); outstanding_issues = @($issues); artifacts = @($manifest.artifacts) }
$rc | ConvertTo-Json -Depth 16 | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir "windows-release-candidate.json")
@"
# Windows Release Candidate

Version: $version
Channel: rc
Final status: $status
Release ready: $ready
Open issues: $($issues.Count)
Artifacts: $((@($manifest.artifacts)).Count)
"@ | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir "windows-release-candidate.md")

$missingEvidence = New-Object System.Collections.Generic.List[object]
if (-not $validation.completed_at) { $missingEvidence.Add([ordered]@{ key = 'completed_at'; status = 'missing'; remediation = 'Mark the validation result completed after the Windows-host run is fully backfilled.' }) }
if (-not $validation.environment.machine) { $missingEvidence.Add([ordered]@{ key = 'environment.machine'; status = 'missing'; remediation = 'Capture the Windows host machine name used for validation.' }) }
if (-not $validation.environment.os_version) { $missingEvidence.Add([ordered]@{ key = 'environment.os_version'; status = 'missing'; remediation = 'Capture the exact Windows build used for validation.' }) }
if (-not $validation.environment.winfsp_version) { $missingEvidence.Add([ordered]@{ key = 'environment.winfsp_version'; status = 'missing'; remediation = 'Capture the WinFsp version from the validation host.' }) }
if (-not $validation.environment.diagnostics_bundle) { $missingEvidence.Add([ordered]@{ key = 'environment.diagnostics_bundle'; status = 'missing'; remediation = 'Attach the exported diagnostics ZIP produced after the Explorer and installer validation runs.' }) }
if (-not $validation.environment.installer_log_dir) { $missingEvidence.Add([ordered]@{ key = 'environment.installer_log_dir'; status = 'missing'; remediation = 'Archive the MSI/EXE installer logs and record the directory path.' }) }
foreach ($item in @($validation.installer_runs)) {
  if (-not $item.log_path) {
    $missingEvidence.Add([ordered]@{ key = "installer_runs.$($item.channel).$($item.action).log_path"; status = 'missing'; remediation = 'Capture the installer log path for each real Windows-host installer run.' })
  }
}
$pendingScenarios = @(@($validation.explorer_scenarios) | Where-Object status -ne 'pass' | ForEach-Object { "$($_.scenario_id) ($($_.status))" })
$pendingInstallerRuns = @(@($validation.installer_runs) | Where-Object status -ne 'pass' | ForEach-Object { "$($_.channel)/$($_.action) ($($_.status))" })
$pendingChecklist = @()
$pendingChecklist += @(@($validation.installer_checklist) | Where-Object status -ne 'pass' | ForEach-Object { "installer: $($_.item) ($($_.status))" })
$pendingChecklist += @(@($validation.recovery_checklist) | Where-Object status -ne 'pass' | ForEach-Object { "recovery: $($_.item) ($($_.status))" })
$intakeReady = ($missingEvidence.Count -eq 0 -and $validation.completed_at)
$intake = [ordered]@{
  generated_at = (Get-Date).ToUniversalTime().ToString('s') + 'Z'
  version = $version
  completed = [bool]$validation.completed_at
  validation_overall = $validation.summary.overall
  ready_for_targeted_fix = [bool]$intakeReady
  missing_evidence_count = $missingEvidence.Count
  open_scenario_count = $pendingScenarios.Count
  open_installer_runs = $pendingInstallerRuns.Count
  open_checklist_items = $pendingChecklist.Count
  evidence = @($missingEvidence)
  pending_scenarios = @($pendingScenarios)
  pending_installer_runs = @($pendingInstallerRuns)
  pending_checklist = @($pendingChecklist)
}
$intake | ConvertTo-Json -Depth 16 | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir 'windows-validation-intake-report.json')
@"
# Windows Validation Intake Report

Version: $version
Completed: $([bool]$validation.completed_at)
Validation overall: $($validation.summary.overall)
Ready for targeted fix: $([bool]$intakeReady)
Missing evidence: $($missingEvidence.Count)
Open explorer scenarios: $($pendingScenarios.Count)
Open installer runs: $($pendingInstallerRuns.Count)
Open checklist items: $($pendingChecklist.Count)
"@ | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir 'windows-validation-intake-report.md')

$finalStatus = if ($ready -and $issues.Count -eq 0 -and $validation.summary.overall -eq 'pass' -and $intakeReady -and $status -eq 'rc-ready') { 'publish-ready' } elseif ($issues.Count -gt 0 -or $validation.summary.overall -eq 'warn' -or $missingEvidence.Count -gt 0) { 'needs-attention' } else { 'blocked' }
$publishReady = ($finalStatus -eq 'publish-ready')
$finalRelease = [ordered]@{
  generated_at = (Get-Date).ToUniversalTime().ToString('s') + 'Z'
  version = $version
  channel = 'stable'
  release_ready = $ready
  publish_ready = $publishReady
  final_status = $finalStatus
  validation_overall = $validation.summary.overall
  open_issues = $issues.Count
  missing_evidence_count = $missingEvidence.Count
  manifest_path = 'release-manifest.json'
  validation_path = 'windows-host-validation-result-template.json'
  intake_path = 'windows-validation-intake-report.json'
  closure_path = 'windows-release-closure.json'
  issue_list_path = 'windows-pre-release-issues.json'
  fix_plan_path = 'windows-first-pass-fix-plan.json'
  rc_path = 'windows-release-candidate.json'
  closure_reasons = @($reasons)
}
$finalRelease | ConvertTo-Json -Depth 16 | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir 'windows-final-release.json')
@"
# Windows Final Release

Version: $version
Channel: stable
Final status: $finalStatus
Release ready: $ready
Publish ready: $publishReady
Validation overall: $($validation.summary.overall)
Open issues: $($issues.Count)
Missing evidence: $($missingEvidence.Count)
"@ | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir 'windows-final-release.md')
@"
# Windows Final Release Sign-Off

Version: $version
Final status: $finalStatus
Publish ready: $publishReady

## Release gates
- [ ] Validation overall is PASS ($($validation.summary.overall))
- [ ] Open issue count is zero ($($issues.Count))
- [ ] Missing evidence count is zero ($($missingEvidence.Count))
- [ ] Release closure is ready ($ready)
- [ ] Final release is publish-ready ($publishReady)

## Signatures
- Engineering: ____________________  Date: __________
- QA / Windows Host Validation: ____  Date: __________
- Release / Packaging: _____________  Date: __________
"@ | Set-Content -Encoding UTF8 (Join-Path $ReleaseDir 'windows-final-signoff.md')
Write-Host "Wrote release closure, intake report, issue list, fix plan, RC, and final release outputs to $ReleaseDir"
