$ErrorActionPreference = "Stop"
param([string]$Version = "0.1.0", [string]$Manufacturer = "OpenAI", [string]$UpgradeCode = "6A257F86-8B32-4F57-9A66-42F0F3456D5A")
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Split-Path -Parent $scriptDir
$dist = Join-Path $root "dist"
$release = Join-Path $dist "windows-release"
$msiRoot = Join-Path $release "msi"
$stage = Join-Path $msiRoot "stage"
$srcDir = Join-Path $msiRoot "wixsrc"
$outDir = Join-Path $msiRoot "out"
New-Item -ItemType Directory -Force -Path $stage,$srcDir,$outDir | Out-Null
$required = @("devmount-client-win32.exe","devmount-winfsp.exe","devmount-server.exe")
foreach ($name in $required) { $src = Join-Path $dist $name; if (-not (Test-Path $src)) { throw "Missing $src. Run scripts/build.ps1 first." }; Copy-Item $src (Join-Path $stage $name) -Force }
$client = [System.IO.Path]::Combine($stage,'devmount-client-win32.exe')
$winfsp = [System.IO.Path]::Combine($stage,'devmount-winfsp.exe')
$server = [System.IO.Path]::Combine($stage,'devmount-server.exe')
$wxs = @"
<?xml version="1.0" encoding="UTF-8"?>
<Wix xmlns="http://wixtoolset.org/schemas/v4/wxs">
  <Package Name="Developer Mount Windows Client" Manufacturer="$Manufacturer" Version="$Version" UpgradeCode="$UpgradeCode" Scope="perMachine">
    <MediaTemplate />
    <StandardDirectory Id="ProgramFiles64Folder">
      <Directory Id="INSTALLDIR" Name="DeveloperMount">
        <Component Id="MainFiles" Guid="*">
          <File Source="$client" />
          <File Source="$winfsp" />
          <File Source="$server" />
        </Component>
      </Directory>
    </StandardDirectory>
    <Feature Id="MainFeature" Title="Developer Mount Windows Client">
      <ComponentRef Id="MainFiles" />
    </Feature>
  </Package>
</Wix>
"@
$wxsPath = Join-Path $srcDir "DeveloperMount.wxs"
$wxs | Set-Content -Encoding UTF8 $wxsPath
$wix = Get-Command wix -ErrorAction SilentlyContinue
if ($wix) { & $wix.Source build $wxsPath -o (Join-Path $outDir "DeveloperMount-$Version.msi") }
Write-Host "Prepared MSI assets at $msiRoot"
