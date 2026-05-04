#Requires -Modules Pester

# Spotnik PowerShell installer tests. Run in Windows CI only.
# Tests pin to release v0.1.0-rc1.

BeforeAll {
    $script:Repo        = (Resolve-Path "$PSScriptRoot\..\..\..").Path
    $script:Installer   = Join-Path $Repo 'install.ps1'
    $script:Uninstaller = Join-Path $Repo 'uninstall.ps1'
    $script:InstallDir  = Join-Path $env:USERPROFILE '.local\bin'
    $script:Exe         = Join-Path $InstallDir 'spotnik.exe'
}

Describe 'install.ps1' {

    BeforeEach {
        if (Test-Path $Exe) { Remove-Item $Exe -Force }
        $env:SPOTNIK_VERSION = $null
    }

    It 'pinned via env var: downloads v0.1.0-rc1 asset (param-position regression)' {
        $env:SPOTNIK_VERSION = 'v0.1.0-rc1'
        & $Installer
        Test-Path $Exe | Should -BeTrue
        $out = & $Exe --version
        $out | Should -Match 'v0\.1\.0-rc1'
    }

    It 'pinned via positional param: downloads v0.1.0-rc1 asset' {
        & $Installer -VersionArg 'v0.1.0-rc1'
        Test-Path $Exe | Should -BeTrue
        $out = & $Exe --version
        $out | Should -Match 'v0\.1\.0-rc1'
    }

    It 'adds install dir to user PATH' {
        & $Installer -VersionArg 'v0.1.0-rc1'
        $userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
        ($userPath -split ';') | Should -Contain $InstallDir
    }

    It 'updates current-process PATH' {
        & $Installer -VersionArg 'v0.1.0-rc1'
        ($env:PATH -split ';') | Should -Contain $InstallDir
    }
}
