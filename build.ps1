$ErrorActionPreference = 'Stop'
$OutputEncoding = [Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$out = Join-Path $root 'dist\okf-cn.exe'
New-Item -ItemType Directory -Path (Split-Path -Parent $out) -Force | Out-Null

Push-Location $root
try {
    go test ./...
    if ($LASTEXITCODE -ne 0) {
        throw "go test failed with exit code $LASTEXITCODE"
    }
    go build -buildvcs=false -trimpath -ldflags '-s -w' -o $out ./cmd/okf
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE"
    }
    Write-Output "Built $out"
} finally {
    Pop-Location
}
