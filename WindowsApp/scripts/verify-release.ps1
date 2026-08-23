param(
    [ValidatePattern("^\d+\.\d+\.\d+$")]
    [string]$Version = "1.7.0",
    [switch]$OnlineReadOnly,
    [string]$CertificateThumbprint = "",
    [switch]$RequireSignature,
    [switch]$PackageInstaller
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$mutationVariables = @(
    "WMS_ONLINE_MUTATION",
    "WMS_ONLINE_OUTBOUND_MUTATION",
    "WMS_ONLINE_MASTER_MUTATION",
    "WMS_ONLINE_WAREHOUSE_MUTATION",
    "WMS_ONLINE_ADMIN_MUTATION",
    "WMS_ONLINE_FINANCE_MUTATION"
)

function Invoke-Checked {
    param([scriptblock]$Command, [string]$Failure)
    & $Command
    if ($LASTEXITCODE -ne 0) {
        throw "$Failure (exit code $LASTEXITCODE)"
    }
}

function Get-PEFacts {
    param([Parameter(Mandatory = $true)][string]$Path)
    $bytes = [System.IO.File]::ReadAllBytes($Path)
    if ($bytes.Length -lt 256 -or $bytes[0] -ne 0x4D -or $bytes[1] -ne 0x5A) {
        throw "not a PE file: $Path"
    }
    $peOffset = [BitConverter]::ToInt32($bytes, 0x3C)
    if ($peOffset -lt 0 -or $peOffset + 96 -ge $bytes.Length) {
        throw "invalid PE header offset: $Path"
    }
    if ([BitConverter]::ToUInt32($bytes, $peOffset) -ne 0x00004550) {
        throw "missing PE signature: $Path"
    }
    $machine = [BitConverter]::ToUInt16($bytes, $peOffset + 4)
    $optionalHeader = $peOffset + 24
    $subsystem = [BitConverter]::ToUInt16($bytes, $optionalHeader + 68)
    [pscustomobject]@{ Machine = $machine; Subsystem = $subsystem }
}

if ($RequireSignature -and [string]::IsNullOrWhiteSpace($CertificateThumbprint)) {
    throw "RequireSignature needs CertificateThumbprint"
}

foreach ($name in $mutationVariables) {
    if ([Environment]::GetEnvironmentVariable($name)) {
        throw "$name is set; release verification refuses to run while an online mutation gate is enabled"
    }
}

Push-Location $projectRoot
try {
    Invoke-Checked { go test -count=1 ./... } "go test failed"
    Invoke-Checked { go vet ./... } "go vet failed"
    Invoke-Checked { go mod tidy -diff } "go mod tidy -diff failed"

    if ($OnlineReadOnly) {
        foreach ($name in @("WMS_ONLINE_BASE_URL", "WMS_ONLINE_IMAGE_BASE_URL", "WMS_ONLINE_MOBILE", "WMS_ONLINE_PASSWORD")) {
            if ([string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($name))) {
                throw "$name is required for OnlineReadOnly verification"
            }
        }
        $oldOnlineTest = $env:WMS_ONLINE_TEST
        try {
            $env:WMS_ONLINE_TEST = "1"
            Invoke-Checked { go test -count=1 ./internal/api ./internal/ui -run Online.*ReadOnly } "online read-only tests failed"
        } finally {
            $env:WMS_ONLINE_TEST = $oldOnlineTest
        }
    }

    $expectedMachines = @{ amd64 = 0x8664; arm64 = 0xAA64 }
    $artifacts = @()
    foreach ($architecture in @("amd64", "arm64")) {
        $arguments = @{ Architecture = $architecture; Version = $Version }
        if (-not [string]::IsNullOrWhiteSpace($CertificateThumbprint)) {
            $arguments["CertificateThumbprint"] = $CertificateThumbprint
        }
        if ($RequireSignature) {
            $arguments["RequireSignature"] = $true
        }
        & (Join-Path $PSScriptRoot "build.ps1") @arguments
        if ($LASTEXITCODE -ne 0) { throw "$architecture build failed" }

        $path = Join-Path $projectRoot "dist\ZhengshiWMS-$architecture.exe"
        $facts = Get-PEFacts -Path $path
        if ($facts.Machine -ne $expectedMachines[$architecture]) {
            throw "$architecture PE machine mismatch: 0x$('{0:X4}' -f $facts.Machine)"
        }
        if ($facts.Subsystem -ne 2) {
            throw "$architecture PE subsystem is not Windows GUI: $($facts.Subsystem)"
        }
        $versionInfo = (Get-Item -LiteralPath $path).VersionInfo
        if (-not $versionInfo.FileVersion.StartsWith($Version) -or -not $versionInfo.ProductVersion.StartsWith($Version)) {
            throw "$architecture version resource mismatch: file=$($versionInfo.FileVersion) product=$($versionInfo.ProductVersion)"
        }
        $signature = Get-AuthenticodeSignature -LiteralPath $path
        if ($RequireSignature -and $signature.Status -ne "Valid") {
            throw "$architecture signature is not valid: $($signature.Status)"
        }
        $hash = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash
        $checksum = Get-Content -LiteralPath (Join-Path $projectRoot "dist\SHA256SUMS-$architecture.txt") -Raw
        if (-not $checksum.Contains($hash)) {
            throw "$architecture SHA256 manifest does not match the artifact"
        }
        $artifacts += [pscustomobject]@{
            Architecture = $architecture
            Path = $path
            SHA256 = $hash
            Signature = [string]$signature.Status
            FileVersion = $versionInfo.FileVersion
        }
    }

    if ($PackageInstaller) {
        foreach ($architecture in @("amd64", "arm64")) {
            $arguments = @{ Architecture = $architecture; Version = $Version }
            if (-not [string]::IsNullOrWhiteSpace($CertificateThumbprint)) {
                $arguments["CertificateThumbprint"] = $CertificateThumbprint
            }
            if ($RequireSignature) {
                $arguments["RequireSignature"] = $true
            }
            & (Join-Path $PSScriptRoot "package.ps1") @arguments
            if ($LASTEXITCODE -ne 0) { throw "$architecture installer packaging failed" }
        }
    }

    $artifacts | Format-Table -AutoSize
    Write-Output "Automated release verification passed. ARM64 real-device, DPI, high-contrast, printer and signed-installer acceptance remain separate manual gates."
} finally {
    Pop-Location
}
