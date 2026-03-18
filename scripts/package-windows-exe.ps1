$ErrorActionPreference = "Stop"
param([string]$Version = "0.1.0")
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Split-Path -Parent $scriptDir
$dist = Join-Path $root "dist"
$release = Join-Path $dist "windows-release"
$exeRoot = Join-Path $release "exe"
$bundle = Join-Path $exeRoot "bundle"
New-Item -ItemType Directory -Force -Path $bundle | Out-Null
$required = @("devmount-client-win32.exe","devmount-winfsp.exe","devmount-server.exe")
foreach ($name in $required) { $src = Join-Path $dist $name; if (-not (Test-Path $src)) { throw "Missing $src. Run scripts/build.ps1 first." }; Copy-Item $src (Join-Path $bundle $name) -Force }
@"
Write-Host "Developer Mount Windows Client portable bundle"
Write-Host "1. Ensure WinFsp is installed"
Write-Host "2. Run devmount-client-win32.exe"
Write-Host "3. Run Diagnostics -> Run Self-Check"
Write-Host "4. Run the Explorer smoke checklist and export diagnostics"
"@ | Set-Content -Encoding UTF8 (Join-Path $bundle "install.cmd")
$zipPath = Join-Path $exeRoot "DeveloperMount-$Version-portable.zip"
if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
Compress-Archive -Path (Join-Path $bundle '*') -DestinationPath $zipPath
Write-Host "Prepared EXE/portable bundle at $exeRoot"
