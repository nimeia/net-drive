$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$outDir = if ($env:OUT_DIR) { $env:OUT_DIR } else { Join-Path $root 'dist/regression' }
$stressDir = if ($env:STRESS_DIR) { $env:STRESS_DIR } else { Join-Path $outDir 'stress' }
$soakDir = if ($env:SOAK_DIR) { $env:SOAK_DIR } else { Join-Path $outDir 'soak' }
$report = if ($env:REPORT) { $env:REPORT } else { Join-Path $outDir 'regression-compare-report.md' }
$runStress = if ($env:RUN_STRESS) { $env:RUN_STRESS } else { '1' }
$runSoak = if ($env:RUN_SOAK) { $env:RUN_SOAK } else { '1' }
$dryRun = if ($env:DRY_RUN) { $env:DRY_RUN } else { '0' }

New-Item -ItemType Directory -Force -Path $outDir, $stressDir, $soakDir | Out-Null

if ($dryRun -eq '1') {
  Write-Host "ROOT=$root"
  Write-Host "OUT_DIR=$outDir"
  Write-Host "STRESS_DIR=$stressDir"
  Write-Host "SOAK_DIR=$soakDir"
  Write-Host "REPORT=$report"
  Write-Host "RUN_STRESS=$runStress"
  Write-Host "RUN_SOAK=$runSoak"
  Write-Host "stress command: OUT_DIR=$stressDir ./scripts/run-stress-suite.ps1"
  Write-Host "soak command: OUT_DIR=$soakDir ./scripts/run-sampled-soak.ps1"
  Write-Host "report command: python ./scripts/render-regression-compare.py --root $outDir --output $report --repo-root $root"
  exit 0
}

if ($runStress -eq '1') {
  $env:OUT_DIR = $stressDir
  ./scripts/run-stress-suite.ps1
}

if ($runSoak -eq '1') {
  $env:OUT_DIR = $soakDir
  ./scripts/run-sampled-soak.ps1
}

python ./scripts/render-regression-compare.py --root $outDir --output $report --repo-root $root
Write-Host "Iter 48 regression compare complete. Report written to $report"
