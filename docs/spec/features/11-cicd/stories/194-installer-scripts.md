---
title: "Installer Scripts"
feature: 11-cicd
status: done
---

## Background

Homebrew tap and Scoop manifest distribution require maintaining separate repos
(`homebrew-tap`, `scoop-bucket`) and a `RELEASE_PAT` secret for cross-repo pushes.
Since no users exist yet there is zero migration cost. This story replaces both channels
with `install.sh` (macOS/Linux) and `install.ps1` (Windows) served directly from raw
GitHub, simplifying the GoReleaser config and release workflow in the process.

Full design: `docs/superpowers/specs/2026-04-30-installer-scripts-design.md`.
Implementation plan: `docs/superpowers/plans/2026-04-30-installer-scripts.md`.

## Design

### `install.sh` (macOS + Linux)

Single-file bash script (`set -euo pipefail`) in the repo root. ANSI-coloured output
(no gum dependency). Flow:

1. Detect OS (`uname -s` → `darwin` | `linux`; error points Windows users to PS1)
2. Detect arch (`uname -m` → `amd64` | `arm64`)
3. Resolve version — `$SPOTNIK_VERSION` if set, else GitHub Releases API `tag_name`
4. Download tarball + checksums to `$TMPDIR` (EXIT trap cleans up)
5. Verify SHA256 (`sha256sum --ignore-missing` on Linux, `shasum -a 256` on macOS)
6. Extract binary
7. Resolve install dir — `$SPOTNIK_INSTALL_DIR` override, else `~/.local/bin` (writable),
   else `/usr/local/bin` via sudo (reads `/dev/tty` — works in `curl|bash`)
8. `chmod +x`, move binary into place
9. Warn if install dir not in `$PATH` (suppressed by `SPOTNIK_NO_MODIFY_PATH=1`)
10. Run `spotnik --version` to confirm

GoReleaser strips the leading `v` from `{{.Version}}` in artifact filenames; the download
URL path uses the full tag. Scripts split accordingly:
- base URL: `.../releases/download/v0.1.0/`
- filename: `spotnik_0.1.0_darwin_arm64.tar.gz`

### `install.ps1` (Windows)

PowerShell 5.1+ script. Flow mirrors the bash script: arch detection, version resolution
via `Invoke-RestMethod`, download + `Get-FileHash` SHA256 check, `Expand-Archive`,
install to `%USERPROFILE%\.local\bin\`, persistent user PATH update via registry
(`Set-ItemProperty HKCU`), temp dir cleanup in `finally` block.

### GoReleaser + release workflow cleanup

Remove `brews` and `scoops` blocks from `.goreleaser.yml`. Remove the `RELEASE_PAT`
validation step and env var from `.github/workflows/release.yml` — `GITHUB_TOKEN`
suffices. Delete the `RELEASE_PAT` repo secret (manual step).

### README update

Replace the six-channel Installation section with: curl one-liner (primary), PS1
one-liner (primary), DEB/RPM (link to Releases page), Go install, manual download.
No brew or scoop references.

## Acceptance Criteria

- [ ] `curl -fsSL https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.sh | bash` installs `spotnik` on macOS arm64
- [ ] `curl -fsSL https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.sh | bash` installs `spotnik` on Linux amd64
- [ ] `SPOTNIK_VERSION=vX.Y.Z bash install.sh` downloads that exact release
- [ ] `SPOTNIK_INSTALL_DIR=/tmp/test bash install.sh` installs to the override path
- [ ] Tampered tarball produces checksum mismatch error and non-zero exit
- [ ] `install.ps1` on Windows amd64 installs to `%USERPROFILE%\.local\bin\spotnik.exe`
- [ ] `goreleaser check` passes with `brews`/`scoops` blocks removed
- [ ] No `RELEASE_PAT` references in `.github/` or `.goreleaser.yml`
- [ ] README Installation section contains no Homebrew or Scoop references
- [ ] `make ci` passes

## Tasks

- [ ] Write `install.sh` to repo root
      - test: `bash -n install.sh` exits 0 (syntax check)
      - test: `SPOTNIK_INSTALL_DIR=/tmp/spotnik-test bash install.sh` places binary at `/tmp/spotnik-test/spotnik`
      - test: corrupt the downloaded tarball in a temp copy; verify exit code is non-zero and error message contains "Checksum"
- [ ] Write `install.ps1` to repo root
      - test: PowerShell parser `ParseFile` returns no parse errors
- [ ] Remove `brews` and `scoops` blocks from `.goreleaser.yml`
      - test: `goreleaser check` exits 0
      - test: `grep -E "^brews:|^scoops:" .goreleaser.yml` returns nothing
- [ ] Remove `RELEASE_PAT` validation step and env var from `.github/workflows/release.yml`; update file comment
      - test: `grep "RELEASE_PAT" .github/workflows/release.yml` returns nothing
- [ ] Rewrite README Installation section (curl/PS1 primary; remove brew/scoop)
      - test: `grep -iE "homebrew|scoop|brew install|scoop install" README.md` returns nothing
      - test: README contains `install.sh` and `install.ps1` URLs
- [ ] Run `make ci` and confirm all checks pass
