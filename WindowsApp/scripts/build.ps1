param(
    [ValidateSet("amd64", "arm64")]
    [string]$Architecture = "amd64",
    [ValidatePattern("^\d+\.\d+\.\d+$")]
    [string]$Version = "0.6.0"
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

    go run github.com/akavel/rsrc@v0.10.2 -arch $Architecture -manifest "build\windowsapp.manifest" -ico "build\windowsapp.ico" -o $resourceFile
    if ($LASTEXITCODE -ne 0) { throw "manifest resource generation failed" }

    go test ./...
    if ($LASTEXITCODE -ne 0) { throw "go test failed" }

    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed" }

    $env:GOOS = "windows"
    $env:GOARCH = $Architecture
    $buildTimestamp = [DateTime]::UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ")
    $gitCommit = (git rev-parse --short=12 HEAD 2>$null)
    if (-not $gitCommit) { $gitCommit = "unknown" }
    $linkerFlags = "-s -w -H windowsgui -X zhengshi-wms-windowsapp/internal/ui.clientVersion=$Version -X zhengshi-wms-windowsapp/internal/ui.buildTime=$buildTimestamp -X zhengshi-wms-windowsapp/internal/ui.gitCommit=$gitCommit"
    go build -trimpath -ldflags $linkerFlags -o "dist\ZhengshiWMS-$Architecture.exe" ./cmd/windowsapp
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }

    Write-Output "Version: $Version"
    Write-Output "Build time: $buildTimestamp"
    Write-Output "Git commit: $gitCommit"
    Get-FileHash "dist\ZhengshiWMS-$Architecture.exe" -Algorithm SHA256
} finally {
    $env:GOOS = $oldGOOS
    $env:GOARCH = $oldGOARCH
    Pop-Location
}
