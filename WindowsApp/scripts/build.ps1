param(
    [ValidateSet("amd64", "386", "arm64")]
    [string]$Architecture = "amd64"
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$outputDirectory = Join-Path $projectRoot "dist"
$resourceFile = Join-Path $projectRoot "cmd\windowsapp\rsrc.syso"
$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH

try {
    Push-Location $projectRoot
    New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null

    go run ./tools/iconbuilder "logo.png" "build/windowsapp.ico"
    if ($LASTEXITCODE -ne 0) { throw "application icon generation failed" }

    $rsrc = Get-Command rsrc -ErrorAction SilentlyContinue
    if (-not $rsrc) {
        go install github.com/akavel/rsrc@latest
        $goPath = go env GOPATH
        $rsrcPath = Join-Path $goPath "bin\rsrc.exe"
    } else {
        $rsrcPath = $rsrc.Source
    }

    & $rsrcPath -manifest "build\windowsapp.manifest" -ico "build\windowsapp.ico" -o $resourceFile
    if ($LASTEXITCODE -ne 0) { throw "manifest resource generation failed" }

    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "go test failed" }

    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed" }

    $env:GOOS = "windows"
    $env:GOARCH = $Architecture
    go build -trimpath -ldflags="-s -w -H windowsgui" -o "dist\ZhengshiWMS-$Architecture.exe" ./cmd/windowsapp
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }

    Get-FileHash "dist\ZhengshiWMS-$Architecture.exe" -Algorithm SHA256
} finally {
    $env:GOOS = $oldGOOS
    $env:GOARCH = $oldGOARCH
    Pop-Location
}
