$ErrorActionPreference = "Stop"
param([string]$Version = "0.1.0")
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Split-Path -Parent $scriptDir
$dist = Join-Path $root "dist"
$release = Join-Path $dist "windows-release"
New-Item -ItemType Directory -Force -Path $release | Out-Null
& (Join-Path $scriptDir "build.ps1")
if ($LASTEXITCODE -ne 0) { throw "build.ps1 failed" }
& (Join-Path $scriptDir "package-windows-msi.ps1") -Version $Version
& (Join-Path $scriptDir "package-windows-exe.ps1") -Version $Version
$manifest = [ordered]@{ package_name = "developer-mount-windows-client"; version = $Version; generated_at = (Get-Date).ToString("s"); artifacts = @(@{ name = "devmount-client-win32.exe"; kind = "exe"; path = "dist/devmount-client-win32.exe" }, @{ name = "devmount-winfsp.exe"; kind = "exe"; path = "dist/devmount-winfsp.exe" }, @{ name = "devmount-server.exe"; kind = "exe"; path = "dist/devmount-server.exe" }, @{ name = "msi"; kind = "installer"; path = "dist/windows-release/msi" }, @{ name = "portable-zip"; kind = "portable"; path = "dist/windows-release/exe" }) }
$manifest | ConvertTo-Json -Depth 6 | Set-Content -Encoding UTF8 (Join-Path $release "release-manifest.json")
@"
# Windows Release Validation

Version: $Version

## Artifacts
- dist/devmount-client-win32.exe
- dist/devmount-winfsp.exe
- dist/devmount-server.exe
- dist/windows-release/msi
- dist/windows-release/exe

## Validation
- Install WinFsp before dispatcher-v1 validation.
- Run devmount-client-win32.exe and Diagnostics -> Run Self-Check.
- Confirm the native callback table and Explorer request matrix summaries are present.
- Run the Explorer smoke checklist on a Windows host.
- Export diagnostics after smoke and archive the bundle.
"@ | Set-Content -Encoding UTF8 (Join-Path $release "release-validation.md")
Write-Host "Prepared Windows release assets at $release"
