#Requires -Version 7.2
<#
.SYNOPSIS
Compile WMS Web or Android resources with an installed HBuilderX toolchain.
.DESCRIPTION
Uses the Node and uni CLI bundled with HBuilderX. Preserves process environment
variables and the working directory. Does not clean caches or remove files.
Android resource compilation is not a complete Kotlin, APK, or device check.
.EXAMPLE
pwsh -File ./scripts/build-hbuilderx.ps1 -HBuilderXPath 'D:\Program Files\HBuilderX5.24' -Platform web
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$HBuilderXPath,

    [ValidateSet('web', 'app-android')]
    [string]$Platform = 'web'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
# Native stderr must be logged and diagnosed, not terminate the tee pipeline.
$PSNativeCommandUseErrorActionPreference = $false

function Get-CompilerDiagnostics {
    param([Parameter(Mandatory = $true)][string]$LogPath)

    $ansiPattern = '\x1B\[[0-?]*[ -/]*[@-~]'
    $errorPattern = '(?i)(?:\berror\s+(?:TS|UTS|KOTLIN)\d*\b|(?:^|[\s\]])(?:error|fatal error)\s*[:：]|\[(?:error|fatal)\]|\b(?:compilation|compile|build)\s+failed\b|\bFAILURE:\s*Build failed\b|\bBUILD FAILED\b|(?:^|\s)e:\s+(?:file://|.*\.kt(?:[:\s]|$)|.*(?:Unresolved reference|Type mismatch|Cannot infer|Expecting))|(?:UTS|Kotlin|原生|资源|项目|源码|代码)[^\r\n]{0,24}(?:编译失败|编译错误)|(?:^|\s)编译失败)'
    Get-Content -LiteralPath $LogPath | ForEach-Object {
        $plain = [regex]::Replace([string]$_, $ansiPattern, '')
        if ($plain -match $errorPattern) { $plain }
    }
}

$hxRoot = (Resolve-Path -LiteralPath $HBuilderXPath).ProviderPath
$appRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).ProviderPath
$pluginsRoot = Join-Path $hxRoot 'plugins'
$nodePath = Join-Path $pluginsRoot 'node/node.exe'
$nodeModules = Join-Path $pluginsRoot 'uniapp-cli-vite/node_modules'
$uniCliPath = Join-Path $nodeModules '@dcloudio/vite-plugin-uni/bin/uni.js'

foreach ($requiredFile in @($nodePath, $uniCliPath, (Join-Path $appRoot 'manifest.json'), (Join-Path $appRoot 'package.json'))) {
    if (-not (Test-Path -LiteralPath $requiredFile -PathType Leaf)) {
        throw "Required file was not found: $requiredFile. Check the HBuilderX installation and project dependencies."
    }
}

$outputPath = Join-Path $appRoot "unpackage/dist/build/$Platform"
$logDirectory = Join-Path $appRoot 'unpackage/verification'
New-Item -ItemType Directory -Path $logDirectory -Force | Out-Null
$runStamp = Get-Date -Format 'yyyyMMdd-HHmmss-fff'
$logPath = Join-Path $logDirectory "hbuilderx-$Platform-$runStamp-$PID.log"

$buildEnvironment = @{
    HX_APP_ROOT = $hxRoot
    UNI_HBUILDERX_PLUGINS = $pluginsRoot
    UNI_INPUT_DIR = $appRoot
    UNI_OUTPUT_DIR = $outputPath
    UNI_APP_X = 'true'
    NODE_PATH = $nodeModules
}
$originalEnvironment = @{}
$originalEnvironmentExists = @{}
foreach ($name in $buildEnvironment.Keys) {
    $originalEnvironmentExists[$name] = Test-Path -LiteralPath "Env:$name"
    $originalEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}

$nativeExitCode = 1
$locationPushed = $false
try {
    foreach ($name in $buildEnvironment.Keys) {
        [Environment]::SetEnvironmentVariable($name, $buildEnvironment[$name], 'Process')
    }
    Push-Location -LiteralPath $appRoot
    $locationPushed = $true
    @(
        "Started: $([DateTime]::Now.ToString('o'))"
        "HBuilderX: $hxRoot"
        "Platform: $Platform"
        "Input: $appRoot"
        "Output: $outputPath"
        "Log: $logPath"
        'Android mode compiles resources; complete Kotlin/APK/device validation is separate.'
    ) | Tee-Object -FilePath $logPath

    & $nodePath $uniCliPath build -p $Platform 2>&1 |
        ForEach-Object { [string]$_ } |
        Tee-Object -FilePath $logPath -Append
    $nativeExitCode = $LASTEXITCODE
} catch {
    "Build invocation failed: $($_.Exception.Message)" | Tee-Object -FilePath $logPath -Append
    $nativeExitCode = 1
} finally {
    if ($locationPushed) { Pop-Location }
    foreach ($name in $originalEnvironment.Keys) {
        if ($originalEnvironmentExists[$name]) {
            [Environment]::SetEnvironmentVariable($name, $originalEnvironment[$name], 'Process')
        } else {
            Remove-Item -LiteralPath "Env:$name" -ErrorAction SilentlyContinue
        }
    }
}

$diagnostics = @(Get-CompilerDiagnostics -LogPath $logPath)
$scriptExitCode = $nativeExitCode
if ($diagnostics.Count -gt 0 -and $scriptExitCode -eq 0) { $scriptExitCode = 1 }
if ($diagnostics.Count -gt 0) {
    Write-Host "Compiler errors detected in $($diagnostics.Count) log line(s):" -ForegroundColor Red
    $diagnostics | ForEach-Object { Write-Host $_ }
}
"Finished: $([DateTime]::Now.ToString('o')); native exit: $nativeExitCode; error lines: $($diagnostics.Count); result: $scriptExitCode" |
    Tee-Object -FilePath $logPath -Append
Write-Host "Build log: $logPath"
exit $scriptExitCode
