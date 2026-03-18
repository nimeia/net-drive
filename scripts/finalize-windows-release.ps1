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
Write-Host "Wrote release closure, issue list, fix plan, and RC outputs to $ReleaseDir"
