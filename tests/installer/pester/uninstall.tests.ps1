#Requires -Modules Pester

BeforeAll {
    $script:Repo        = (Resolve-Path "$PSScriptRoot\..\..\..").Path
    $script:Installer   = Join-Path $Repo 'install.ps1'
    $script:Uninstaller = Join-Path $Repo 'uninstall.ps1'
    $script:InstallDir  = Join-Path $env:USERPROFILE '.local\bin'
    $script:Exe         = Join-Path $InstallDir 'spotnik.exe'
}

Describe 'uninstall.ps1' {

    It 'round-trip removes binary and PATH entry' {
        & $Installer -VersionArg 'v0.1.0-rc1'
        Test-Path $Exe | Should -BeTrue

        # SPOTNIK_KEEP_CONFIG=1 to skip the prompt branch in CI.
        $env:SPOTNIK_KEEP_CONFIG = '1'
        & $Uninstaller
        Test-Path $Exe | Should -BeFalse
    }
}
