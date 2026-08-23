param(
    [string]$Executable = ""
)

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms

$os = Get-CimInstance Win32_OperatingSystem
$screens = [System.Windows.Forms.Screen]::AllScreens | ForEach-Object {
    [pscustomobject]@{
        DeviceName = $_.DeviceName
        Primary = $_.Primary
        Bounds = $_.Bounds.ToString()
        WorkingArea = $_.WorkingArea.ToString()
    }
}
$desktop = Get-ItemProperty -LiteralPath "HKCU:\Control Panel\Desktop" -ErrorAction SilentlyContinue
$highContrast = Get-ItemProperty -LiteralPath "HKCU:\Control Panel\Accessibility\HighContrast" -ErrorAction SilentlyContinue
$result = [ordered]@{
    CapturedAt = [DateTime]::UtcNow.ToString("o")
    ComputerName = $env:COMPUTERNAME
    OS = $os.Caption
    OSVersion = $os.Version
    OSArchitecture = $os.OSArchitecture
    ProcessArchitecture = [System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString()
    LogPixels = $desktop.LogPixels
    Win8DpiScaling = $desktop.Win8DpiScaling
    HighContrastFlags = $highContrast.Flags
    Screens = @($screens)
}

if (-not [string]::IsNullOrWhiteSpace($Executable)) {
    $resolved = (Resolve-Path -LiteralPath $Executable).Path
    $item = Get-Item -LiteralPath $resolved
    $signature = Get-AuthenticodeSignature -LiteralPath $resolved
    $result["Executable"] = [ordered]@{
        Path = $resolved
        FileVersion = $item.VersionInfo.FileVersion
        ProductVersion = $item.VersionInfo.ProductVersion
        SHA256 = (Get-FileHash -LiteralPath $resolved -Algorithm SHA256).Hash
        Signature = [string]$signature.Status
    }
}

[pscustomobject]$result | ConvertTo-Json -Depth 6
