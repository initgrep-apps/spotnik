# Feature 57 — CI/CD & Release Pipeline

> **Feature:** Set up GitHub Actions CI/CD, GoReleaser cross-compilation, release-please
> version management, multi-platform distribution (Homebrew, Scoop, DEB, RPM), version
> injection, README, and MIT license.

## Context

Spotnik has no CI/CD, no automated releases, no git tags, and a 2-line README placeholder.
The version is hardcoded as `const appVersion = "v1.1.0"` in `internal/app/splash.go`.
The Makefile already injects `main.version` via LDFLAGS but the variables are unused.

This feature sets up the full release pipeline so that every tag push automatically builds
and ships binaries to 6 distribution channels.

**Prerequisite:** Complete the manual GitHub setup steps in `docs/GITHUB-SETUP.md` before
implementing this feature. The release workflow requires repos and secrets that must be
created manually.

**Depends on:** Nothing — infrastructure only, no functional changes.

**Implementation plan:** `docs/superpowers/plans/2026-03-26-cicd-release-readme.md`

**Design spec:** `docs/superpowers/specs/2026-03-26-cicd-release-readme-design.md`

---

## Release Flow

```
feature PR merged to main
  → release-please creates/updates Release PR (version + changelog)
    → maintainer merges Release PR when ready
      → tag v0.1.0 created automatically
        → Release workflow fires (GoReleaser)
          → Builds 5 platform binaries
          → Creates GitHub Release with checksums
          → Pushes Homebrew formula to initgrep-apps/homebrew-tap
          → Pushes Scoop manifest to initgrep-apps/scoop-bucket
          → Uploads DEB and RPM packages
```

---

## Task 1: Version injection

**Problem:** Version is hardcoded as `const appVersion = "v1.1.0"` in `splash.go`.

**Fix:**
- Add `var version = "dev"` and `var buildTime = ""` in `main.go`
- Change `cmd.Execute()` to accept `(version, buildTime string)`
- Set `rootCmd.Version = version` (enables `spotnik --version`)
- Add `Version` and `BuildTime` fields to `AppOptions`
- Add `version` field to `App` struct, set from `AppOptions.Version`
- Change `renderSplashView()` signature to accept `version string` parameter
- Remove `const appVersion` from `splash.go`
- Update `render.go` to pass `a.version` to `renderSplashView()`
- Update `splash_test.go` assertions to use explicit version strings

**Files:**
- `main.go`
- `cmd/root.go`
- `internal/app/app.go`
- `internal/app/splash.go`
- `internal/app/splash_test.go`
- `internal/app/render.go`

**Tests:** Run `go test ./internal/app/... -run TestRenderSplash -v` and `make build`
to verify LDFLAGS injection works.

---

## Task 2: GoReleaser configuration

**Problem:** No automated release build pipeline exists.

**Fix:** Create `.goreleaser.yml` with:
- 5 build targets: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- LDFLAGS: `-s -w -X main.version={{.Version}} -X main.buildTime={{.Date}}`
- Archives: `.tar.gz` for Linux/macOS, `.zip` for Windows
- Homebrew tap: auto-push formula to `initgrep-apps/homebrew-tap`
- Scoop bucket: auto-push manifest to `initgrep-apps/scoop-bucket`
- nfpms: DEB and RPM packages
- Changelog: skipped (release-please manages it)

**Files:**
- `.goreleaser.yml`

---

## Task 3: GitHub Actions — CI workflow

**Problem:** No automated CI pipeline exists.

**Fix:** Create `.github/workflows/ci.yml`:
- Triggers: push to any branch, PRs targeting `main`
- Steps: checkout, setup Go, install golangci-lint, run `make ci` targets
- Single job on `ubuntu-latest`

**Files:**
- `.github/workflows/ci.yml`

---

## Task 4: GitHub Actions — Release workflow

**Problem:** No automated release workflow exists.

**Fix:** Create `.github/workflows/release.yml`:
- Triggers: push of `v*` tag
- Steps: checkout (full history), setup Go, run `goreleaser release --clean`
- Secrets: `GITHUB_TOKEN` (auto), `RELEASE_PAT` (for tap/bucket repos)

**Files:**
- `.github/workflows/release.yml`

---

## Task 5: GitHub Actions — release-please workflow

**Problem:** No automated versioning or changelog generation.

**Fix:** Create workflow and config files:
- `.github/workflows/release-please.yml` — runs on push to `main`
- `.release-please-manifest.json` — starts at `0.0.0` (first PR bumps to `0.1.0`)
- `release-please-config.json` — `release-type: go`, Conventional Commits sections

**Files:**
- `.github/workflows/release-please.yml`
- `.release-please-manifest.json`
- `release-please-config.json`

---

## Task 6: LICENSE

**Fix:** Create MIT license file in project root.

**Files:**
- `LICENSE`

---

## Task 7: README

**Problem:** README is a 2-line placeholder.

**Fix:** Full rewrite with sections:
- Header with CI/release/license badges
- Installation (6 channels: Homebrew, Scoop, DEB, RPM, `go install`, binary)
- Prerequisites (Spotify Premium, Developer App setup)
- Quick Start
- Keybindings (subset, link to full table in `docs/DESIGN.md`)
- Configuration
- Building from source
- Contributing (Conventional Commits, `make ci`)
- License

**Files:**
- `README.md`

---

## Task 8: Cleanup

**Fix:**
- Remove `## Versioning` section from `docs/features/00-overview.md`
- Tag-based semver replaces the static version-to-feature mapping

**Files:**
- `docs/features/00-overview.md`

---

## Task 9: Full validation

- Run `make ci` — all checks must pass
- Run `./bin/spotnik --version` — must print version string
- Verify all new files are committed

---

## Distribution Channels

| Channel | Install Command | Managed By |
|---|---|---|
| Homebrew | `brew install initgrep-apps/tap/spotnik` | GoReleaser → homebrew-tap repo |
| Scoop | `scoop install spotnik` | GoReleaser → scoop-bucket repo |
| DEB | `sudo dpkg -i spotnik_*.deb` | GoReleaser → GitHub Release |
| RPM | `sudo rpm -i spotnik_*.rpm` | GoReleaser → GitHub Release |
| Go install | `go install github.com/initgrep-apps/spotnik@latest` | Automatic (public repo) |
| Binary | Download from Releases page | GoReleaser → GitHub Release |
