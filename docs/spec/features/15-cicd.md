---
title: "CI/CD & Release Pipeline"
description: "Automated CI, cross-platform release builds, version management, and multi-channel distribution so every tagged release ships binaries to Homebrew, Scoop, DEB, RPM, go install, and GitHub Releases."
status: open
stories: [57]
---

# CI/CD & Release Pipeline

## Background

Spotnik currently has no CI/CD pipeline, no automated releases, no git tags, and a 2-line README placeholder. The version is hardcoded as `const appVersion = "v1.1.0"` in `internal/app/splash.go`. While the Makefile already injects `main.version` via LDFLAGS, the variables are unused. There is no way for users to install Spotnik through a package manager.

This feature sets up the full release pipeline so that every tag push automatically builds and ships binaries to 6 distribution channels: Homebrew, Scoop, DEB, RPM, `go install`, and direct binary download. It also introduces release-please for automated versioning and changelog generation driven by Conventional Commits, replaces the hardcoded version with build-time injection, adds an MIT license, and rewrites the README with installation instructions for all channels.

The release flow is: feature PR merged to main, release-please creates/updates a Release PR with version bump and changelog, maintainer merges the Release PR, a git tag is created automatically, the Release workflow fires GoReleaser which builds 5 platform binaries, creates a GitHub Release with checksums, and pushes package manager manifests to their respective repos.

---

## Story: CI/CD & Release Pipeline (spec 57)

### Background

Spotnik has no automated build, test, or release infrastructure. The version string is hardcoded, there are no GitHub Actions workflows, no GoReleaser configuration, no package manager distribution, and no proper README or license file. This story sets up the complete CI/CD and release pipeline from scratch, including version injection, GoReleaser cross-compilation, GitHub Actions for CI and release, release-please for automated versioning, an MIT license, a full README, and cleanup of the now-obsolete static versioning section in the feature overview.

A prerequisite is completing the manual GitHub setup steps in `docs/GITHUB-SETUP.md` before implementing this feature, as the release workflow requires repos and secrets that must be created manually.

### Acceptance Criteria

- [ ] Version is injected at build time via LDFLAGS, not hardcoded
- [ ] `spotnik --version` prints the injected version string
- [ ] GoReleaser builds 5 platform targets (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
- [ ] CI workflow runs on push to any branch and PRs targeting `main`
- [ ] Release workflow triggers on `v*` tag push and runs GoReleaser
- [ ] release-please workflow runs on push to `main` and manages version bumps and changelogs
- [ ] Homebrew formula is auto-pushed to `initgrep-apps/homebrew-tap`
- [ ] Scoop manifest is auto-pushed to `initgrep-apps/scoop-bucket`
- [ ] DEB and RPM packages are uploaded to GitHub Releases
- [ ] MIT license file exists in project root
- [ ] README includes installation instructions for all 6 channels, badges, prerequisites, keybindings, configuration, and contributing sections
- [ ] Static versioning section removed from `docs/features/00-overview.md`
- [ ] `make ci` passes with all changes

### Release Flow

```
feature PR merged to main
  -> release-please creates/updates Release PR (version + changelog)
    -> maintainer merges Release PR when ready
      -> tag v0.1.0 created automatically
        -> Release workflow fires (GoReleaser)
          -> Builds 5 platform binaries
          -> Creates GitHub Release with checksums
          -> Pushes Homebrew formula to initgrep-apps/homebrew-tap
          -> Pushes Scoop manifest to initgrep-apps/scoop-bucket
          -> Uploads DEB and RPM packages
```

### Distribution Channels

| Channel | Install Command | Managed By |
|---|---|---|
| Homebrew | `brew install initgrep-apps/tap/spotnik` | GoReleaser -> homebrew-tap repo |
| Scoop | `scoop install spotnik` | GoReleaser -> scoop-bucket repo |
| DEB | `sudo dpkg -i spotnik_*.deb` | GoReleaser -> GitHub Release |
| RPM | `sudo rpm -i spotnik_*.rpm` | GoReleaser -> GitHub Release |
| Go install | `go install github.com/initgrep-apps/spotnik@latest` | Automatic (public repo) |
| Binary | Download from Releases page | GoReleaser -> GitHub Release |

### Tasks

1. **Version injection** — Replace hardcoded version with build-time LDFLAGS injection
   - Add `var version = "dev"` and `var buildTime = ""` in `main.go`
   - Change `cmd.Execute()` to accept `(version, buildTime string)`
   - Set `rootCmd.Version = version` (enables `spotnik --version`)
   - Add `Version` and `BuildTime` fields to `AppOptions`
   - Add `version` field to `App` struct, set from `AppOptions.Version`
   - Change `renderSplashView()` signature to accept `version string` parameter
   - Remove `const appVersion` from `splash.go`
   - Update `render.go` to pass `a.version` to `renderSplashView()`
   - Update `splash_test.go` assertions to use explicit version strings
   - Files: `main.go`, `cmd/root.go`, `internal/app/app.go`, `internal/app/splash.go`, `internal/app/splash_test.go`, `internal/app/render.go`
   - Tests: Run `go test ./internal/app/... -run TestRenderSplash -v` and `make build` to verify LDFLAGS injection works

2. **GoReleaser configuration** — Create `.goreleaser.yml` for automated cross-platform builds
   - 5 build targets: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
   - LDFLAGS: `-s -w -X main.version={{.Version}} -X main.buildTime={{.Date}}`
   - Archives: `.tar.gz` for Linux/macOS, `.zip` for Windows
   - Homebrew tap: auto-push formula to `initgrep-apps/homebrew-tap`
   - Scoop bucket: auto-push manifest to `initgrep-apps/scoop-bucket`
   - nfpms: DEB and RPM packages
   - Changelog: skipped (release-please manages it)
   - Files: `.goreleaser.yml`

3. **GitHub Actions — CI workflow** — Automated CI on every push and PR
   - Triggers: push to any branch, PRs targeting `main`
   - Steps: checkout, setup Go, install golangci-lint, run `make ci` targets
   - Single job on `ubuntu-latest`
   - Files: `.github/workflows/ci.yml`

4. **GitHub Actions — Release workflow** — Automated release on tag push
   - Triggers: push of `v*` tag
   - Steps: checkout (full history), setup Go, run `goreleaser release --clean`
   - Secrets: `GITHUB_TOKEN` (auto), `RELEASE_PAT` (for tap/bucket repos)
   - Files: `.github/workflows/release.yml`

5. **GitHub Actions — release-please workflow** — Automated versioning and changelog
   - `.github/workflows/release-please.yml` — runs on push to `main`
   - `.release-please-manifest.json` — starts at `0.0.0` (first PR bumps to `0.1.0`)
   - `release-please-config.json` — `release-type: go`, Conventional Commits sections
   - Files: `.github/workflows/release-please.yml`, `.release-please-manifest.json`, `release-please-config.json`

6. **LICENSE** — Add MIT license file to project root
   - Files: `LICENSE`

7. **README** — Full rewrite with all sections
   - Header with CI/release/license badges
   - Installation (6 channels: Homebrew, Scoop, DEB, RPM, `go install`, binary)
   - Prerequisites (Spotify Premium, Developer App setup)
   - Quick Start
   - Keybindings (subset, link to full table in `docs/DESIGN.md`)
   - Configuration
   - Building from source
   - Contributing (Conventional Commits, `make ci`)
   - License
   - Files: `README.md`

8. **Cleanup** — Remove obsolete static versioning section
   - Remove `## Versioning` section from `docs/features/00-overview.md`
   - Tag-based semver replaces the static version-to-feature mapping
   - Files: `docs/features/00-overview.md`

9. **Full validation** — End-to-end verification
   - Run `make ci` — all checks must pass
   - Run `./bin/spotnik --version` — must print version string
   - Verify all new files are committed
