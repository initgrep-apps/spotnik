# Installer tests

Bats-based tests for `install.sh` / `uninstall.sh` and Pester tests for
`install.ps1` / `uninstall.ps1`.

## Local run (Linux only)

    make test-installer

This builds Docker images for Ubuntu (bash/zsh/fish), Debian, Arch, and Fedora,
runs the bats suite in each, and reports pass/fail per cell.

Limit the matrix:

    bash tests/installer/run.sh ubuntu-bash debian

## CI

`.github/workflows/installer-tests.yml` runs the Linux Docker matrix on
`ubuntu-latest`, the bats suite directly on `macos-latest`, and the Pester
suite on `windows-latest`. Triggered only on changes under
`install.sh`, `install.ps1`, `uninstall.sh`, `uninstall.ps1`, or
`tests/installer/**`.

Tests pin to release `v0.1.0-rc1` (immutable). The "latest" code path is
covered only by a smoke case that asserts a binary is produced.

## Lifetime

These tests are intentionally short-lived. Once the installer redesign has
shipped clean across two releases, delete `tests/installer/`, the Makefile
target, and the workflow.
