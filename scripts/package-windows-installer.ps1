$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Split-Path -Parent $scriptDir
$dist = Join-Path $root "dist"
$stage = Join-Path $dist "windows-installer-stage"
$manifestPath = Join-Path $stage "install-manifest.json"
New-Item -ItemType Directory -Force -Path $stage | Out-Null
$required = @("devmount-client-win32.exe","devmount-winfsp.exe","devmount-server.exe")
foreach ($name in $required) {
  $src = Join-Path $dist $name
  if (-not (Test-Path $src)) { throw "Missing $src. Run scripts/build.ps1 first." }
  Copy-Item $src (Join-Path $stage $name) -Force
}
Copy-Item (Join-Path $root "README.md") (Join-Path $stage "README.md") -Force
Copy-Item (Join-Path $root "configs\devmount.example.json") (Join-Path $stage "devmount.example.json") -Force
$manifest = [ordered]@{ generated_at=(Get-Date).ToString("s"); package="developer-mount-windows-client"; binaries=$required; winfsp_required=$true; install_steps=@("Install WinFsp before starting the client in dispatcher-v1 mode.","Copy or unzip the staged files to the target machine.","Launch devmount-client-win32.exe and run Self-Check before Explorer smoke."); uninstall_steps=@("Stop all active mounts.","Close the tray application.","Delete the staged files or remove the installed directory.") }
$manifest | ConvertTo-Json -Depth 6 | Set-Content -Encoding UTF8 $manifestPath
@"
Write-Host "Developer Mount Windows client install bootstrap"
Write-Host "1. Ensure WinFsp is installed"
Write-Host "2. Run devmount-client-win32.exe"
Write-Host "3. Use Diagnostics -> Run Self-Check"
Write-Host "4. Run the Explorer smoke checklist from explorer-smoke.md"
"@ | Set-Content -Encoding UTF8 (Join-Path $stage "install.ps1")
@"
Write-Host "Developer Mount Windows client uninstall bootstrap"
Write-Host "1. Stop active mounts from the tray or Dashboard"
Write-Host "2. Exit the tray application"
Write-Host "3. Remove the installed files"
"@ | Set-Content -Encoding UTF8 (Join-Path $stage "uninstall.ps1")
Write-Host "Prepared Windows installer stage at $stage"

@"
# Installer validation template

## MSI
- [ ] install
- [ ] upgrade
- [ ] uninstall

## EXE
- [ ] portable launch

## Notes
- Attach installer logs and Windows host validation result JSON.
"@ | Set-Content -Encoding UTF8 (Join-Path $stage "windows-installer-results-template.md")

Copy-Item (Join-Path $stage "windows-installer-results-template.md") (Join-Path $stage "windows-host-validation-result-template.md") -Force

Copy-Item (Join-Path $root "dist\windows-release\windows-release-closure-template.md") (Join-Path $stage "windows-release-closure-template.md") -Force -ErrorAction SilentlyContinue
Copy-Item (Join-Path $root "dist\windows-release\windows-release-closure-template.json") (Join-Path $stage "windows-release-closure-template.json") -Force -ErrorAction SilentlyContinue
