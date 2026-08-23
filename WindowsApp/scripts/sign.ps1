param(
    [Parameter(Mandatory = $true)]
    [string]$Path,
    [Parameter(Mandatory = $true)]
    [string]$CertificateThumbprint,
    [string]$TimestampServer = "http://timestamp.digicert.com"
)

$ErrorActionPreference = "Stop"
$resolvedPath = (Resolve-Path -LiteralPath $Path).Path
if (-not (Test-Path -LiteralPath $resolvedPath -PathType Leaf)) {
    throw "signing target is not a file: $resolvedPath"
}

$thumbprint = $CertificateThumbprint.Replace(" ", "").Trim()
if ($thumbprint -notmatch "^[0-9A-Fa-f]{40}$") {
    throw "CertificateThumbprint must be the 40-character SHA-1 certificate thumbprint used by signtool /sha1"
}

$signTool = Get-Command "signtool.exe" -ErrorAction SilentlyContinue
if (-not $signTool) {
    $kitsRoot = Join-Path ${env:ProgramFiles(x86)} "Windows Kits\10\bin"
    if (Test-Path -LiteralPath $kitsRoot -PathType Container) {
        $candidate = Get-ChildItem -LiteralPath $kitsRoot -Directory |
            Sort-Object Name -Descending |
            ForEach-Object { Join-Path $_.FullName "x64\signtool.exe" } |
            Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } |
            Select-Object -First 1
        if ($candidate) {
            $signTool = Get-Item -LiteralPath $candidate
        }
    }
}
if (-not $signTool) {
    throw "signtool.exe was not found; install the Windows SDK signing tools"
}

$signToolPath = if ($signTool.Source) { $signTool.Source } else { $signTool.FullName }
& $signToolPath sign /sha1 $thumbprint /fd SHA256 /tr $TimestampServer /td SHA256 $resolvedPath
if ($LASTEXITCODE -ne 0) {
    throw "signtool failed with exit code $LASTEXITCODE"
}

$signature = Get-AuthenticodeSignature -LiteralPath $resolvedPath
if ($signature.Status -ne "Valid") {
    throw "signed file verification failed: $($signature.Status)"
}
$signature
