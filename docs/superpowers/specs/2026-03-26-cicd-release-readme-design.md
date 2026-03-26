# Spotnik CI/CD, Release Pipeline, and README — Design Spec

> **Date:** 2026-03-26
> **Status:** Draft
> **Owner:** irshad.mike@gmail.com

---

## Overview

Set up automated CI/CD, tag-based semver releases, multi-platform distribution, and a
proper README for Spotnik. The goal: every merge to `main` accumulates toward a release;
when the maintainer merges the Release PR, binaries ship to all channels automatically.

---

## 1. Version Injection

### Current State
- `internal/app/splash.go` has `const appVersion = "v1.1.0"` (hardcoded)
- `Makefile` injects `main.version` and `main.buildTime` via LDFLAGS, but the variables
  are never consumed in code

### Design

#### Version flow: `main.go` → `cmd` → `app` → `splash`

1. **`main.go`**: Add package-level vars injected by LDFLAGS:
   ```go
   var version = "dev"
   var buildTime = ""
   func main() { cmd.Execute(version, buildTime) }
   ```

2. **`cmd/root.go`**: Change `Execute()` signature to `Execute(version, buildTime string)`.
   Set `rootCmd.Version = version` (enables `spotnik --version`).
   Pass version into `AppOptions`:
   ```go
   func Execute(version, buildTime string) {
       rootCmd.Version = version
       // ... in runApp:
       opts := app.AppOptions{ ..., Version: version, BuildTime: buildTime }
   }
   ```

3. **`internal/app/app.go`**: Add `Version string` and `BuildTime string` fields to
   `AppOptions`. Store on the `App` struct.

4. **`internal/app/splash.go`**: Remove `const appVersion = "v1.1.0"`. Change
   `renderSplashView()` to accept version as a parameter (or read from `App.version`).

5. **`internal/app/splash_test.go`**: Update assertions — tests currently check for
   `appVersion` in rendered output. Change to check for the injected version value
   (e.g., pass `"dev"` or `"v0.1.0"` in test setup).

#### Build-time injection
- GoReleaser: `-X main.version={{.Version}} -X main.buildTime={{.Date}}`
- Makefile: `-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)` (already present)
- Local dev: `git describe --tags --always --dirty` or `"dev"` fallback

#### Initial version
`v0.1.0` — this is the first official tagged semver release. The version table previously
in `docs/features/00-overview.md` was informal tracking and is being removed (Section 5.1).
All prior version references were untagged.

---

## 2. GitHub Actions Workflows

### 2.1 CI Workflow (`.github/workflows/ci.yml`)

**Triggers:** Push to any branch, PRs targeting `main`

**Job: `ci`**
1. Checkout code
2. Set up Go (version extracted from `go.mod`)
3. Cache Go modules (`~/go/pkg/mod`, `~/.cache/go-build`)
4. Install `golangci-lint`
5. Run `make ci` (executes: `fmt-check`, `tidy-check`, `lint`, `test-coverage`, `build`)

Single job, single runner (`ubuntu-latest`). The Makefile `ci` target already
orchestrates the full check sequence.

### 2.2 Release Workflow (`.github/workflows/release.yml`)

**Triggers:** Push of tag matching `v*`

**Job: `release`**
1. Checkout code with full git history (`fetch-depth: 0`)
2. Set up Go
3. Run `goreleaser release --clean`
4. GoReleaser handles: cross-compilation, checksums, GitHub Release creation,
   Homebrew formula push, Scoop manifest push, DEB/RPM package generation

**Secrets required:** `RELEASE_PAT` (Personal Access Token with write access to
tap and bucket repos)

### 2.3 release-please Workflow (`.github/workflows/release-please.yml`)

**Triggers:** Push to `main`

**Job: `release-please`**
1. Runs `googleapis/release-please-action`
2. Configuration: `release-type: go`, `default-branch: main`
3. Analyzes Conventional Commits since last release
4. Creates/updates a Release PR with:
   - Version bump (based on `feat:` → minor, `fix:` → patch, `feat!:` → major)
   - Auto-generated `CHANGELOG.md`
5. When maintainer merges the Release PR → creates `v*` tag → triggers Release Workflow

### 2.4 release-please Configuration Files

**`.release-please-manifest.json`:**
```json
{
  ".": "0.0.0"
}
```
Starting at `0.0.0` so the first release-please PR bumps to `0.1.0`.

**`release-please-config.json`:**
```json
{
  "packages": {
    ".": {
      "release-type": "go",
      "bump-minor-pre-major": true,
      "bump-patch-for-minor-pre-major": true,
      "changelog-sections": [
        { "type": "feat", "section": "Features" },
        { "type": "fix", "section": "Bug Fixes" },
        { "type": "refactor", "section": "Refactoring" },
        { "type": "test", "section": "Tests" },
        { "type": "chore", "section": "Miscellaneous" }
      ]
    }
  }
}
```

### Release Flow

```
feature PR merged to main
  → release-please creates/updates Release PR (version + changelog)
    → maintainer merges Release PR when ready
      → tag v0.2.0 created automatically
        → Release workflow fires
          → GoReleaser builds + publishes to all channels
```

---

## 3. GoReleaser Configuration

File: `.goreleaser.yml` in project root.

### 3.1 Builds

| OS | Arch | Format |
|---|---|---|
| linux | amd64 | tar.gz |
| linux | arm64 | tar.gz |
| darwin | amd64 | tar.gz |
| darwin | arm64 | tar.gz |
| windows | amd64 | zip |

- Binary name: `spotnik`
- LDFLAGS: `-s -w -X main.version={{.Version}} -X main.buildTime={{.Date}}`
- Build flags: `-trimpath`
- CGO disabled

### 3.2 Archives

- Naming: `spotnik_{{.Version}}_{{.Os}}_{{.Arch}}`
- `.tar.gz` for Linux/macOS, `.zip` for Windows

### 3.3 Checksums

- SHA256 checksums file auto-generated

### 3.4 Homebrew Tap

- Target repo: `initgrep-apps/homebrew-tap`
- Formula name: `spotnik`
- Auto-pushed on release via `RELEASE_PAT`
- Includes: description, homepage URL, install block, test block

### 3.5 Scoop Bucket

- Target repo: `initgrep-apps/scoop-bucket`
- Manifest name: `spotnik`
- Auto-pushed on release via `RELEASE_PAT`

### 3.6 DEB/RPM Packages (nfpms)

- Package name: `spotnik`
- Maintainer: `irshad.mike@gmail.com`
- License: MIT
- Description: Terminal Spotify client for developers
- Installs to: `/usr/bin/spotnik`
- Formats: `deb`, `rpm`
- Uploaded as GitHub Release artifacts

### 3.7 Changelog

- Skipped in GoReleaser — release-please manages changelog via `CHANGELOG.md`
- GoReleaser config:
  ```yaml
  changelog:
    skip: true
  ```

---

## 4. README

Replace the current 2-line placeholder with a full README. Sections:

### 4.1 Header
- Project name and one-line description
- Badges: CI status, latest release, Go version, license

### 4.2 Screenshot
- Placeholder for terminal screenshot/GIF (to be added later)

### 4.3 Installation

Six channels, each with copy-paste commands:

1. **Homebrew (macOS/Linux):**
   ```
   brew install initgrep-apps/tap/spotnik
   ```

2. **Scoop (Windows):**
   ```
   scoop bucket add spotnik https://github.com/initgrep-apps/scoop-bucket
   scoop install spotnik
   ```

3. **DEB (Ubuntu/Debian):**
   ```
   wget https://github.com/initgrep-apps/spotnik/releases/latest/download/spotnik_<version>_linux_amd64.deb
   sudo dpkg -i spotnik_<version>_linux_amd64.deb
   ```

4. **RPM (Fedora/RHEL):**
   ```
   wget https://github.com/initgrep-apps/spotnik/releases/latest/download/spotnik_<version>_linux_amd64.rpm
   sudo rpm -i spotnik_<version>_linux_amd64.rpm
   ```

5. **Go install:**
   ```
   go install github.com/initgrep-apps/spotnik@latest
   ```

6. **Binary download:** Link to GitHub Releases page

### 4.4 Prerequisites
- Spotify Premium account
- Spotify Developer app (with redirect URI setup for PKCE auth)

### 4.5 Quick Start
- Run `spotnik` → auth flow opens browser → start playing

### 4.6 Keybindings
- Show a subset: navigation (Tab, Shift+Tab, Page1/Page2), playback (space, n, p, +/-),
  search (/), queue (a), devices (d), quit (q/Ctrl+C)
- Link to `docs/DESIGN.md` for the full table

### 4.7 Configuration
- Config file location (`~/.config/spotnik/config.toml`)
- Key options: theme, default device

### 4.8 Building from Source
- Note: requires Go version as specified in `go.mod` (currently 1.26.1+)
```
git clone https://github.com/initgrep-apps/spotnik.git
cd spotnik
make build
./bin/spotnik
```

### 4.9 Contributing
- Conventional Commits required
- Run `make ci` before pushing
- PR process: one feature per branch, never merge your own PR

### 4.10 License
- MIT

---

## 5. Cleanup

### 5.1 Remove Versioning Table
- Delete the `## Versioning` section from `docs/features/00-overview.md`
- Tag-based semver replaces static version-to-feature mapping

### 5.2 Remove Hardcoded Version
- Delete `const appVersion = "v1.1.0"` from `internal/app/splash.go`
- Replaced by injected `version` variable (Section 1)

### 5.3 Add LICENSE File
- MIT license in project root (`LICENSE`)

---

## 6. GitHub Setup (Manual Steps)

### 6.1 Create Homebrew Tap Repo
1. Go to `https://github.com/organizations/initgrep-apps/repositories/new`
2. Name: `homebrew-tap`
3. Public repo, initialize with README
4. No other config needed — GoReleaser pushes the formula

### 6.2 Create Scoop Bucket Repo
1. Same location
2. Name: `scoop-bucket`
3. Public repo, initialize with README

### 6.3 Create Personal Access Token
1. Go to `https://github.com/settings/personal-access-tokens/new`
2. Fine-grained token, scoped to `initgrep-apps` organization
3. Repository access: select `homebrew-tap` and `scoop-bucket`
4. Permissions: `Contents: Read and write`
5. Generate and copy

### 6.4 Add Repository Secret
1. Go to `https://github.com/initgrep-apps/spotnik/settings/secrets/actions`
2. New repository secret
3. Name: `RELEASE_PAT`
4. Value: the PAT from step 6.3

---

## 7. File Summary

| File | Action |
|---|---|
| `main.go` | Add `version` and `buildTime` vars, pass to `cmd.Execute()` |
| `cmd/root.go` | Accept version params, forward to app |
| `internal/app/app.go` | Add `Version` and `BuildTime` fields to `AppOptions` and `App` |
| `internal/app/splash.go` | Remove `const appVersion`, use injected version from `App` |
| `internal/app/splash_test.go` | Update assertions to use injected version value |
| `.github/workflows/ci.yml` | Create — CI pipeline |
| `.github/workflows/release.yml` | Create — GoReleaser release pipeline |
| `.github/workflows/release-please.yml` | Create — release-please automation |
| `.goreleaser.yml` | Create — GoReleaser configuration |
| `README.md` | Rewrite — full documentation |
| `LICENSE` | Create — MIT license |
| `docs/features/00-overview.md` | Remove versioning section |
| `.release-please-manifest.json` | Create — tracks current version |
| `release-please-config.json` | Create — release-please configuration |
