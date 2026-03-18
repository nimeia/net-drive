$ErrorActionPreference = "Stop"
param(
  [string]$ReleaseDir = (Join-Path (Join-Path (Split-Path -Parent $MyInvocation.MyCommand.Path) "..") "dist\windows-release"),
  [string]$ValidationResultJson = "",
  [string]$CompletedBy = ""
)
if (-not $ValidationResultJson) { $ValidationResultJson = Join-Path $ReleaseDir "windows-host-validation-result-template.json" }
if (-not (Test-Path $ValidationResultJson)) { throw "Missing validation result json: $ValidationResultJson" }
$validation = Get-Content $ValidationResultJson -Raw | ConvertFrom-Json
$reasons = New-Object System.Collections.Generic.List[string]
$all = @(); if ($validation.explorer_scenarios) { $all += $validation.explorer_scenarios }; if ($validation.installer_checklist) { $all += $validation.installer_checklist }; if ($validation.recovery_checklist) { $all += $validation.recovery_checklist }; if ($validation.installer_runs) { $all += $validation.installer_runs }
$notRun = ($all | Where-Object { $_.status -eq "not-run" }).Count; $fail = ($all | Where-Object { $_.status -eq "fail" }).Count
if ($fail -gt 0) { $reasons.Add("validation record still contains failed checks") }
if ($notRun -gt 0) { $reasons.Add("validation record still contains not-run checks") }
foreach ($pair in @(@("msi","install"),@("msi","upgrade"),@("msi","uninstall"),@("exe","portable-launch"))) { $hit = $validation.installer_runs | Where-Object { $_.channel -eq $pair[0] -and $_.action -eq $pair[1] -and $_.status -eq "pass" }; if (-not $hit) { $reasons.Add("installer run $($pair[0])/$($pair[1]) has not passed") } }
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
Write-Host "Wrote release closure to $ReleaseDir"
