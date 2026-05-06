#Requires -Version 5.1
param(
    [string]$VersionArg
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# spotnik installer for Windows (PowerShell 5.1+).
#
# Usage:
#   irm https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.ps1 | iex
#   $env:SPOTNIK_VERSION="v0.1.0"; irm https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.ps1 | iex
#
# Env:
#   $env:SPOTNIK_VERSION         pin a release tag (default: latest stable)
#   $env:SPOTNIK_INSTALL_DIR     override install destination
#   $env:SPOTNIK_NO_MODIFY_PATH  skip user/process PATH update

function Write-Banner  { Write-Host "`n  spotnik installer`n" -ForegroundColor White }
function Write-Success { param($msg) Write-Host "v $msg" -ForegroundColor Cyan }
function Write-Info    { param($msg) Write-Host ". $msg" -ForegroundColor DarkGray }
function Write-Warn    { param($msg) Write-Host "! $msg" -ForegroundColor Yellow }
function Write-Err     { param($msg) Write-Host "x $msg" -ForegroundColor Red }

Write-Banner

$cpuArch = $env:PROCESSOR_ARCHITECTURE
if ($cpuArch -ne 'AMD64') {
    Write-Err "Unsupported architecture: $cpuArch (only AMD64 supported)"
    exit 1
}
Write-Success "Arch: amd64"

$version = $VersionArg
if (-not $version) { $version = $env:SPOTNIK_VERSION }
if (-not $version) {
    Write-Info "Resolving latest version..."
    try {
        $release = Invoke-RestMethod -Uri 'https://api.github.com/repos/initgrep-apps/spotnik/releases/latest' -UseBasicParsing
        $version = $release.tag_name
    } catch {
        Write-Err "Failed to query GitHub API: $_"
        Write-Info 'Workaround: pin a version, e.g. $env:SPOTNIK_VERSION="v0.1.0"; irm ... | iex'
        exit 1
    }
}
if (-not $version) {
    Write-Err "Could not resolve version from GitHub API"
    exit 1
}
Write-Success "Version: $version"

# GoReleaser strips the leading 'v' from artifact names; tag keeps it.
$versionNum   = $version.TrimStart('v')
$zipName      = "spotnik_${versionNum}_windows_amd64.zip"
$checksumName = "checksums.txt"
$baseUrl      = "https://github.com/initgrep-apps/spotnik/releases/download/$version"

$tmpDir = Join-Path $env:TEMP "spotnik-install-$([System.IO.Path]::GetRandomFileName())"
New-Item -ItemType Directory -Path $tmpDir | Out-Null

try {
    Write-Info "Downloading $zipName..."
    try {
        Invoke-WebRequest -Uri "$baseUrl/$zipName"      -OutFile "$tmpDir\$zipName"      -UseBasicParsing
        Invoke-WebRequest -Uri "$baseUrl/$checksumName" -OutFile "$tmpDir\$checksumName" -UseBasicParsing
    } catch {
        $status = $null
        if ($_.Exception -is [System.Net.WebException] -and $_.Exception.Response) {
            $status = [int]$_.Exception.Response.StatusCode
        }
        if ($status -eq 404) {
            Write-Err "Release $version not found (404). Check https://github.com/initgrep-apps/spotnik/releases for available versions."
        } else {
            Write-Err "Download failed: $_"
        }
        exit 1
    }
    Write-Success "Downloaded"

    Write-Info "Verifying checksum..."
    $checksumLine = Get-Content "$tmpDir\$checksumName" | Where-Object { $_ -match [regex]::Escape($zipName) }
    if (-not $checksumLine) {
        Write-Err "Checksum entry for $zipName not found in checksums file"
        exit 1
    }
    $expectedHash = ($checksumLine -split '\s+')[0]
    $actualHash   = (Get-FileHash -Path "$tmpDir\$zipName" -Algorithm SHA256).Hash
    if ($actualHash.ToLower() -ne $expectedHash.ToLower()) {
        Write-Err "Checksum mismatch -- aborting"
        exit 1
    }
    Write-Success "Checksum OK"

    Write-Info "Extracting..."
    Expand-Archive -Path "$tmpDir\$zipName" -DestinationPath $tmpDir -Force
    Write-Success "Extracted"

    if ($env:SPOTNIK_INSTALL_DIR) {
        $installDir = $env:SPOTNIK_INSTALL_DIR
    } else {
        $installDir = Join-Path $env:LOCALAPPDATA 'Programs\spotnik'
    }
    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir | Out-Null
    }
    $src = Get-ChildItem -Path $tmpDir -Filter 'spotnik.exe' -Recurse | Select-Object -First 1
    if (-not $src) {
        Write-Err "spotnik.exe not found in extracted archive"
        exit 1
    }
    Copy-Item -Path $src.FullName -Destination "$installDir\spotnik.exe" -Force
    Write-Success "Installed $installDir\spotnik.exe"

    $pathOk = $true
    if ($env:SPOTNIK_NO_MODIFY_PATH -eq '1') {
        Write-Warn "Skipping PATH update (`$env:SPOTNIK_NO_MODIFY_PATH=1)"
        $pathOk = $false
    } else {
        $userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
        if (-not $userPath) { $userPath = '' }
        $pathEntries = $userPath -split ';' | Where-Object { $_ -ne '' }
        if ($pathEntries -notcontains $installDir) {
            $newPath = (@($installDir) + $pathEntries) -join ';'
            if ($newPath.Length -gt 2047) {
                Write-Warn "User PATH would exceed safe length ($($newPath.Length) chars). Add manually: $installDir"
                $pathOk = $false
            } else {
                try {
                    [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
                    Write-Warn "Added $installDir to user PATH (new shells inherit automatically)"
                } catch {
                    Write-Warn "Could not update PATH automatically: $_"
                    Write-Warn "Add manually to user PATH: $installDir"
                    $pathOk = $false
                }
            }
        }
        if (($env:PATH -split ';') -notcontains $installDir) {
            $env:PATH = "$installDir;$env:PATH"
        }
    }

    $exePath = Join-Path $installDir 'spotnik.exe'
    if (Test-Path $exePath) {
        $global:LASTEXITCODE = 0
        $ver = & $exePath --version 2>&1
        if ($LASTEXITCODE -eq 0 -and $ver) {
            Write-Host ""
            Write-Success $ver
            if ($pathOk) {
                Write-Host "`n  Run: spotnik`n" -ForegroundColor White
            } else {
                Write-Host "`n  Run with full path until PATH is fixed:" -ForegroundColor White
                Write-Host "    & '$exePath'`n" -ForegroundColor Yellow
            }
        } else {
            Write-Warn "Installed binary failed to run (exit $LASTEXITCODE): $ver"
            Write-Info "Possible causes: missing VC++ redistributable, Defender quarantine, wrong arch."
        }
    }
} finally {
    Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
}
