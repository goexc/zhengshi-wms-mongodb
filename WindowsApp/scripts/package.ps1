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
$buildArguments = @{
    Architecture = $Architecture
    Version = $Version
    TimestampServer = $TimestampServer
}
if (-not [string]::IsNullOrWhiteSpace($CertificateThumbprint)) {
    $buildArguments["CertificateThumbprint"] = $CertificateThumbprint
}
if ($RequireSignature) {
    $buildArguments["RequireSignature"] = $true
}
& (Join-Path $PSScriptRoot "build.ps1") @buildArguments

$compiler = Get-Command "ISCC.exe" -ErrorAction SilentlyContinue
if (-not $compiler) {
    $candidates = @(
        (Join-Path $env:ProgramFiles "Inno Setup 6\ISCC.exe"),
        (Join-Path ${env:ProgramFiles(x86)} "Inno Setup 6\ISCC.exe")
    )
    foreach ($candidate in $candidates) {
        if ($candidate -and (Test-Path -LiteralPath $candidate -PathType Leaf)) {
            $compiler = Get-Item -LiteralPath $candidate
            break
        }
    }
}
if (-not $compiler) {
    throw "ISCC.exe was not found; install Inno Setup 6 to create the installer"
}

$installerScript = Join-Path $projectRoot "build\installer.iss"
& $compiler.FullName "/DMyAppVersion=$Version" "/DMyAppArch=$Architecture" $installerScript
if ($LASTEXITCODE -ne 0) {
    throw "installer compilation failed with exit code $LASTEXITCODE"
}

$installer = Join-Path $projectRoot "dist\ZhengshiWMS-Setup-$Architecture-$Version.exe"
if (-not (Test-Path -LiteralPath $installer -PathType Leaf)) {
    throw "installer output was not created: $installer"
}
if (-not [string]::IsNullOrWhiteSpace($CertificateThumbprint)) {
    & (Join-Path $PSScriptRoot "sign.ps1") -Path $installer -CertificateThumbprint $CertificateThumbprint -TimestampServer $TimestampServer
}
$signature = Get-AuthenticodeSignature -LiteralPath $installer
if ($RequireSignature -and $signature.Status -ne "Valid") {
    throw "installer signature is not valid: $($signature.Status)"
}
$hash = Get-FileHash -LiteralPath $installer -Algorithm SHA256
$checksumFile = Join-Path $projectRoot "dist\SHA256SUMS-Setup-$Architecture.txt"
[System.IO.File]::WriteAllText($checksumFile, "$($hash.Hash) *$([System.IO.Path]::GetFileName($installer))`r`n", [System.Text.UTF8Encoding]::new($false))
Write-Output "Installer signature: $($signature.Status)"
$hash
