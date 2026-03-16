$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Split-Path -Parent $scriptDir
$dist = Join-Path $root "dist"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go is not available in PATH. Install Go or add it to PATH before running this script."
}

New-Item -ItemType Directory -Path $dist -Force | Out-Null

$targets = @(
    @{ Name = "devmount-server.exe"; Package = "./cmd/devmount-server" },
    @{ Name = "devmount-client.exe"; Package = "./cmd/devmount-client" },
    @{ Name = "devmount-winfsp.exe"; Package = "./cmd/devmount-winfsp" }
)

Push-Location $root
try {
    foreach ($target in $targets) {
        $output = Join-Path $dist $target.Name
        go build -o $output $target.Package
        if ($LASTEXITCODE -ne 0) {
            throw "go build failed for $($target.Package)"
        }
    }
}
finally {
    Pop-Location
}

Write-Host "Built binaries into $dist"
