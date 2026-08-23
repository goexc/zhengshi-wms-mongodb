param(
    [ValidateSet("amd64", "arm64")]
    [string]$Architecture = "amd64",
    [ValidatePattern("^\d+\.\d+\.\d+$")]
    [string]$Version = "1.7.0",
    [string]$CertificateThumbprint = "",
    [string]$TimestampServer = "http://timestamp.digicert.com",
    [switch]$RequireSignature
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$outputDirectory = Join-Path $projectRoot "dist"
$resourceFile = Join-Path $projectRoot "cmd\windowsapp\rsrc.syso"
$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH

if ($RequireSignature -and [string]::IsNullOrWhiteSpace($CertificateThumbprint)) {
    throw "RequireSignature needs CertificateThumbprint"
}

try {
    Push-Location $projectRoot
    New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null

    go run ./tools/iconbuilder "logo.png" "build/windowsapp.ico"
    if ($LASTEXITCODE -ne 0) { throw "application icon generation failed" }

    $versionParts = $Version.Split('.')
    $resourceArguments = @(
        "run", "github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0",
        "-64=true",
        "-ver-major", $versionParts[0], "-ver-minor", $versionParts[1], "-ver-patch", $versionParts[2], "-ver-build", "0",
        "-product-ver-major", $versionParts[0], "-product-ver-minor", $versionParts[1], "-product-ver-patch", $versionParts[2], "-product-ver-build", "0",
        "-file-version", $Version, "-product-version", $Version,
        "-company", "Zhengshi", "-description", "Zhengshi WMS Windows Client", "-product-name", "Zhengshi WMS",
        "-internal-name", "ZhengshiWMS", "-original-name", "ZhengshiWMS-$Architecture.exe",
        "-manifest", "build\windowsapp.manifest", "-icon", "build\windowsapp.ico", "-application-icon", "build\windowsapp.ico",
        "-o", $resourceFile
    )
    if ($Architecture -eq "arm64") {
        $resourceArguments += "-arm=true"
    }
    $resourceArguments += "build\versioninfo.json"
    & go @resourceArguments
    if ($LASTEXITCODE -ne 0) { throw "Windows icon, manifest and version resource generation failed" }

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
    $outputFile = Join-Path $outputDirectory "ZhengshiWMS-$Architecture.exe"
    go build -trimpath -ldflags $linkerFlags -o $outputFile ./cmd/windowsapp
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }

    if (-not [string]::IsNullOrWhiteSpace($CertificateThumbprint)) {
        & (Join-Path $PSScriptRoot "sign.ps1") -Path $outputFile -CertificateThumbprint $CertificateThumbprint -TimestampServer $TimestampServer
        if ($LASTEXITCODE -ne 0) { throw "application signing failed" }
    }

    $signature = Get-AuthenticodeSignature -LiteralPath $outputFile
    if ($RequireSignature -and $signature.Status -ne "Valid") {
        throw "application signature is not valid: $($signature.Status)"
    }

    $hash = Get-FileHash -LiteralPath $outputFile -Algorithm SHA256
    $checksumFile = Join-Path $outputDirectory "SHA256SUMS-$Architecture.txt"
    [System.IO.File]::WriteAllText($checksumFile, "$($hash.Hash) *$([System.IO.Path]::GetFileName($outputFile))`r`n", [System.Text.UTF8Encoding]::new($false))

    Write-Output "Version: $Version"
    Write-Output "Build time: $buildTimestamp"
    Write-Output "Git commit: $gitCommit"
    Write-Output "Signature: $($signature.Status)"
    $hash
} finally {
    $env:GOOS = $oldGOOS
    $env:GOARCH = $oldGOARCH
    Pop-Location
}
