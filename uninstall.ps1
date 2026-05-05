#Requires -Version 5.1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# spotnik uninstaller -- Windows (PowerShell 5.1+)
# Usage:
#   irm https://raw.githubusercontent.com/initgrep-apps/spotnik/main/uninstall.ps1 | iex
# Env:
#   $env:SPOTNIK_PURGE_CONFIG = "1"   also delete %APPDATA%\spotnik (default: prompt)
#   $env:SPOTNIK_KEEP_CONFIG  = "1"   skip config deletion (default: prompt)

function Write-Banner  { Write-Host "`n  spotnik uninstaller`n" -ForegroundColor White }
function Write-Success { param($msg) Write-Host "v $msg" -ForegroundColor Cyan }
function Write-Info    { param($msg) Write-Host ". $msg" -ForegroundColor DarkGray }
function Write-Warn    { param($msg) Write-Host "! $msg" -ForegroundColor Yellow }
function Write-Err     { param($msg) Write-Host "x $msg" -ForegroundColor Red }

Write-Banner

# Resolve binary location
$exePath = $null
$cmd = Get-Command spotnik -ErrorAction SilentlyContinue
if ($cmd) {
    $exePath = $cmd.Source
} else {
    foreach ($candidate in @(
        (Join-Path $env:USERPROFILE '.local\bin\spotnik.exe'),
        (Join-Path $env:USERPROFILE 'bin\spotnik.exe'),
        (Join-Path $env:LOCALAPPDATA 'Programs\spotnik\spotnik.exe')
    )) {
        if (Test-Path $candidate) { $exePath = $candidate; break }
    }
}

if (-not $exePath) {
    Write-Warn "spotnik binary not found in PATH or common install locations"
    Write-Info "Nothing to uninstall."
} else {
    Write-Success "Found: $exePath"

    # Forget credentials (best-effort)
    Write-Info "Wiping tokens and client ID from Windows Credential Manager (spotnik auth forget)..."
    try {
        & $exePath auth forget | Out-Null
        Write-Success "Credentials wiped"
    } catch {
        Write-Warn "spotnik auth forget exited non-zero (already forgotten?). Continuing."
    }

    # Remove binary
    Write-Info "Removing $exePath..."
    try {
        Remove-Item -Path $exePath -Force
        Write-Success "Removed $exePath"
    } catch {
        Write-Err "Failed to remove ${exePath}: $_"
        exit 1
    }

    # Remove install dir from user PATH if it now contains nothing meaningful
    $installDir = Split-Path -Parent $exePath
    $userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    if ($userPath -and $userPath -split ';' -contains $installDir -and -not (Test-Path $installDir -PathType Container -ErrorAction SilentlyContinue)) {
        $newPath = ($userPath -split ';' | Where-Object { $_ -ne $installDir -and $_ -ne '' }) -join ';'
        try {
            [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
            Write-Info "Removed $installDir from user PATH"
        } catch {
            Write-Warn "Could not update PATH: $_"
        }
    }
}

# Config dir handling
$configDir = Join-Path $env:APPDATA 'spotnik'
if (-not (Test-Path $configDir)) {
    $configDir = Join-Path $env:USERPROFILE '.config\spotnik'
}

if (Test-Path $configDir) {
    if ($env:SPOTNIK_KEEP_CONFIG -eq '1') {
        Write-Info "Keeping config dir: $configDir"
    } elseif ($env:SPOTNIK_PURGE_CONFIG -eq '1') {
        Remove-Item -Recurse -Force $configDir
        Write-Success "Removed $configDir"
    } else {
        $ans = Read-Host "  Also remove $configDir? [y/N]"
        if ($ans -match '^[yY]') {
            Remove-Item -Recurse -Force $configDir
            Write-Success "Removed $configDir"
        } else {
            Write-Info "Kept $configDir"
        }
    }
}

Write-Host ""
Write-Success "Uninstall complete."
