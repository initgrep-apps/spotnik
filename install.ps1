#Requires -Version 5.1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# spotnik installer -- Windows (PowerShell 5.1+)
# Usage: powershell -c "irm https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.ps1 | iex"
# Env:   $env:SPOTNIK_VERSION = "v0.1.0"  pin a release (default: latest)

function Write-Banner  { Write-Host "`n  spotnik installer`n" -ForegroundColor White }
function Write-Success { param($msg) Write-Host "v $msg" -ForegroundColor Cyan }
function Write-Info    { param($msg) Write-Host ". $msg" -ForegroundColor DarkGray }
function Write-Warn    { param($msg) Write-Host "! $msg" -ForegroundColor Yellow }
function Write-Err     { param($msg) Write-Host "x $msg" -ForegroundColor Red }

Write-Banner

# Arch detection -- only amd64 is built
$cpuArch = $env:PROCESSOR_ARCHITECTURE
if ($cpuArch -ne 'AMD64') {
    Write-Err "Unsupported architecture: $cpuArch (only AMD64 supported)"
    exit 1
}
Write-Success "Arch: amd64"

# Version resolution
$version = $env:SPOTNIK_VERSION
if (-not $version) {
    Write-Info "Resolving latest version..."
    $release = Invoke-RestMethod -Uri 'https://api.github.com/repos/initgrep-apps/spotnik/releases/latest' -UseBasicParsing
    $version = $release.tag_name
}
if (-not $version) {
    Write-Err "Could not resolve version from GitHub API"
    exit 1
}
Write-Success "Version: $version"

# GoReleaser strips the leading 'v' from artifact names; tag keeps it.
$versionNum   = $version.TrimStart('v')
$zipName      = "spotnik_${versionNum}_windows_amd64.zip"
$checksumName = "spotnik_${versionNum}_checksums.txt"
$baseUrl      = "https://github.com/initgrep-apps/spotnik/releases/download/$version"

# Temp directory -- cleaned up in finally block
$tmpDir = Join-Path $env:TEMP "spotnik-install-$([System.IO.Path]::GetRandomFileName())"
New-Item -ItemType Directory -Path $tmpDir | Out-Null

try {
    # Download
    Write-Info "Downloading $zipName..."
    Invoke-WebRequest -Uri "$baseUrl/$zipName"      -OutFile "$tmpDir\$zipName"      -UseBasicParsing
    Invoke-WebRequest -Uri "$baseUrl/$checksumName" -OutFile "$tmpDir\$checksumName" -UseBasicParsing
    Write-Success "Downloaded"

    # Verify checksum
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

    # Extract
    Write-Info "Extracting..."
    Expand-Archive -Path "$tmpDir\$zipName" -DestinationPath $tmpDir -Force
    Write-Success "Extracted"

    # Install
    $installDir = Join-Path $env:USERPROFILE '.local\bin'
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

    # Update user PATH
    $userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    if ($userPath -notlike "*$installDir*") {
        [Environment]::SetEnvironmentVariable('PATH', "$installDir;$userPath", 'User')
        Write-Warn "Added $installDir to your PATH (restart shell to take effect)"
    }

    # Confirm
    $exePath = "$installDir\spotnik.exe"
    if (Test-Path $exePath) {
        $ver = & $exePath --version 2>$null
        Write-Host ""
        Write-Success $ver
        Write-Host "`n  Run: spotnik`n" -ForegroundColor White
    }
} finally {
    Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
}
