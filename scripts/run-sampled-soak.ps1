$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$outDir = if ($env:OUT_DIR) { $env:OUT_DIR } else { Join-Path $root 'dist/stress' }
$duration = if ($env:DURATION) { $env:DURATION } else { '3m' }
$sampleInterval = if ($env:SAMPLE_INTERVAL) { $env:SAMPLE_INTERVAL } else { '1s' }
$browseWorkers = if ($env:BROWSE_WORKERS) { $env:BROWSE_WORKERS } else { '4' }
$saveWorkers = if ($env:SAVE_WORKERS) { $env:SAVE_WORKERS } else { '3' }
$heartbeatWorkers = if ($env:HEARTBEAT_WORKERS) { $env:HEARTBEAT_WORKERS } else { '2' }
$resumeWorkers = if ($env:RESUME_WORKERS) { $env:RESUME_WORKERS } else { '2' }
$faultSlowClient = if ($env:FAULT_SLOW_CLIENT) { $env:FAULT_SLOW_CLIENT } else { '1' }
$faultHalfClose = if ($env:FAULT_HALF_CLOSE) { $env:FAULT_HALF_CLOSE } else { '1' }
$faultDelayedWrite = if ($env:FAULT_DELAYED_WRITE) { $env:FAULT_DELAYED_WRITE } else { '1' }

New-Item -ItemType Directory -Force -Path $outDir | Out-Null

$csvPath = Join-Path $outDir 'sampled-soak-samples.csv'
$reportPath = Join-Path $outDir 'sampled-soak-report.md'
$runLog = Join-Path $outDir 'sampled-soak-run.log'
$stderrLog = Join-Path $outDir 'sampled-soak-run.stderr.log'

$goArgs = @(
  'run',
  './cmd/devmount-soak',
  '-duration', $duration,
  '-sample-interval', $sampleInterval,
  '-browse-workers', $browseWorkers,
  '-save-workers', $saveWorkers,
  '-heartbeat-workers', $heartbeatWorkers,
  '-resume-workers', $resumeWorkers,
  ('-fault-slow-client=' + $(if ($faultSlowClient -eq '1') { 'true' } else { 'false' })),
  ('-fault-half-close=' + $(if ($faultHalfClose -eq '1') { 'true' } else { 'false' })),
  ('-fault-delayed-write=' + $(if ($faultDelayedWrite -eq '1') { 'true' } else { 'false' })),
  '-csv', $csvPath,
  '-report', $reportPath
)

if (Test-Path $runLog) {
  Remove-Item $runLog -Force
}
if (Test-Path $stderrLog) {
  Remove-Item $stderrLog -Force
}

$goExe = (Get-Command go).Source
$proc = Start-Process -FilePath $goExe -ArgumentList $goArgs -WorkingDirectory $root -NoNewWindow -Wait -PassThru -RedirectStandardOutput $runLog -RedirectStandardError $stderrLog

if (Test-Path $stderrLog) {
  Get-Content $stderrLog | Add-Content $runLog
  Remove-Item $stderrLog -Force
}

Get-Content $runLog

if ($proc.ExitCode -ne 0) {
  throw "go run ./cmd/devmount-soak exited with code $($proc.ExitCode)"
}
