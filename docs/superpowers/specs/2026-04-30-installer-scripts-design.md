# Installer Scripts Design

**Date:** 2026-04-30
**Status:** Approved

## Summary

Add `install.sh` (macOS/Linux) and `install.ps1` (Windows) to the repo root so spotnik
can be installed with a single command. Simultaneously drop Homebrew and Scoop distribution
from GoReleaser — no users exist yet so there is zero migration cost, and the one-liner
covers both audiences with less infrastructure.

---

## One-Liner Commands

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.sh | bash

# macOS / Linux — pinned version
SPOTNIK_VERSION=v0.2.0 curl -fsSL https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.sh | bash

# Windows (PowerShell)
powershell -c "irm https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.ps1 | iex"

# Windows — pinned version
$env:SPOTNIK_VERSION="v0.2.0"; irm https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.ps1 | iex
```

Scripts live on `main` and are always current. Release artifacts are already uploaded to
GitHub Releases by GoReleaser — no additional GoReleaser changes required for the
download step.

---

## Files Changed

| File | Change |
|---|---|
| `install.sh` | New — macOS + Linux installer |
| `install.ps1` | New — Windows installer |
| `.goreleaser.yml` | Remove `brews` and `scoops` blocks |
| `.github/workflows/release.yml` | Remove `RELEASE_PAT` validation step |
| `README.md` | Rewrite Installation section (one-liner primary, brew/scoop removed) |
| `docs/spec/features/11-cicd/feature.md` | Update description + acceptance criteria |
| `docs/spec/features/11-cicd/stories/134-installer-scripts.md` | New story |

---

## `install.sh` Design

### Style
ANSI-coloured output matching the openclaw installer's plain fallback mode (no gum
dependency). Uses `SUCCESS`, `WARN`, `ERROR`, `MUTED`, `INFO`, `BOLD`, `NC` colour vars.

### Flow

```
1.  Print banner
2.  Detect OS         uname -s → "darwin" | "linux" | error
                      (error message points Windows users to install.ps1)
3.  Detect arch       uname -m → "amd64" | "arm64" | error (unsupported)
4.  Resolve version   if $SPOTNIK_VERSION set → use it
                      else → curl GitHub Releases API, grep tag_name
5.  Build names       tarball:   spotnik_${VERSION}_${OS}_${ARCH}.tar.gz
                      checksums: spotnik_${VERSION}_checksums.txt
                      base URL:  https://github.com/initgrep-apps/spotnik/releases/download/${VERSION}
6.  Download          tarball + checksums → $TMPDIR (EXIT trap cleans up)
7.  Verify checksum   sha256sum --ignore-missing (Linux)
                      shasum -a 256 --ignore-missing (macOS)
                      exit 1 on mismatch
8.  Extract           tar -xzf → binary to tmp dir
9.  Resolve install   $SPOTNIK_INSTALL_DIR set → use it directly
                      else:
                      a) $HOME/.local/bin writable + on $PATH  → install there
                      b) $HOME/.local/bin writable, NOT on $PATH → install there + warn
                      c) not writable → offer /usr/local/bin via sudo
                         (reads from /dev/tty — works in curl|bash context)
10. Install binary    chmod +x, mv into place
11. Confirm           run `spotnik --version`, print success
```

### Env-var config

| Variable | Default | Purpose |
|---|---|---|
| `SPOTNIK_VERSION` | (latest via API) | Pin a specific release tag |
| `SPOTNIK_INSTALL_DIR` | (auto-detect) | Override install destination |
| `SPOTNIK_NO_MODIFY_PATH` | `0` | Skip PATH warning / modification |

### OS / arch matrix

| `uname -s` | `uname -m` | Artifact |
|---|---|---|
| Darwin | x86_64 | `spotnik_vX_darwin_amd64.tar.gz` |
| Darwin | arm64 | `spotnik_vX_darwin_arm64.tar.gz` |
| Linux | x86_64 | `spotnik_vX_linux_amd64.tar.gz` |
| Linux | aarch64 | `spotnik_vX_linux_arm64.tar.gz` |
| Other | any | Error — unsupported |

---

## `install.ps1` Design

### Flow

```
1.  Print banner
2.  Detect arch       $env:PROCESSOR_ARCHITECTURE → amd64 only
                      (arm64 Windows not built by GoReleaser)
3.  Resolve version   if $env:SPOTNIK_VERSION set → use it
                      else → Invoke-RestMethod GitHub Releases API, read .tag_name
4.  Build names       zip:       spotnik_${VERSION}_windows_amd64.zip
                      checksums: spotnik_${VERSION}_checksums.txt
5.  Download          zip + checksums → $env:TEMP
6.  Verify checksum   Get-FileHash -Algorithm SHA256, compare against checksums file
                      Throw on mismatch
7.  Extract           Expand-Archive → tmp dir
8.  Install           $env:USERPROFILE\.local\bin\ (create if missing)
9.  Update PATH       if dir not in user PATH → Set-ItemProperty (HKCU registry)
                      Print reminder to restart shell
10. Confirm           & spotnik.exe --version, print success
```

### Requirements
- PowerShell 5.1+ (ships with Windows 10+)
- No additional dependencies

---

## GoReleaser Changes

Remove the `brews` and `scoops` top-level blocks from `.goreleaser.yml`. No other
GoReleaser changes — builds, archives, checksums, nfpms, and GitHub Release upload are
all unchanged.

```yaml
# Remove these two blocks entirely:
brews:
  - ...
scoops:
  - ...
```

---

## Release Workflow Changes

Remove the `RELEASE_PAT` validation step from `.github/workflows/release.yml` — it was
only needed for cross-repo Homebrew/Scoop pushes. The `GITHUB_TOKEN` built-in is
sufficient for everything that remains. The `RELEASE_PAT` secret can be deleted from the
repo settings.

---

## README Installation Section

Replace the current six-channel Installation section with:

```
## Installation

### macOS / Linux
curl -fsSL https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.sh | bash

### Windows
powershell -c "irm https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.ps1 | iex"

### Linux packages (DEB / RPM)
Download from the Releases page.

### Go install
go install github.com/initgrep-apps/spotnik@latest

### Manual
Download a pre-built binary from the Releases page.
```

---

## Spec Changes

**`docs/spec/features/11-cicd/feature.md`**
- Update description: replace "distributes via Homebrew, Scoop, DEB, and RPM" with
  "distributes via curl/PS1 one-liner installer, DEB, and RPM"
- Update acceptance criteria: remove Homebrew + Scoop checks; add one-liner install check

**New story:** `docs/spec/features/11-cicd/stories/134-installer-scripts.md`
- Tasks: write `install.sh`, write `install.ps1`, remove brews/scoops from GoReleaser,
  update release workflow, update README, update feature.md

---

## Out of Scope

- Update awareness / version checking (future story)
- Custom domain URL shortening (future, if a domain is acquired)
- Gum / interactive spinner UI (Go binary installs fast enough without it)
- Homebrew core submission (future, once user base warrants it)
