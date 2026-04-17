---
title: "CI/CD & Release Pipeline"
feature: 15-cicd
status: done
---

## Background
Spotnik has no automated build, test, or release infrastructure. The version string is hardcoded, there are no GitHub Actions workflows, no GoReleaser configuration, no package manager distribution, and no proper README or license file. This story sets up the complete CI/CD and release pipeline from scratch, including version injection, GoReleaser cross-compilation, GitHub Actions for CI and release, release-please for automated versioning, an MIT license, a full README, and cleanup of the now-obsolete static versioning section.

A prerequisite is completing the manual GitHub setup steps in `docs/GITHUB-SETUP.md` before implementing this feature, as the release workflow requires repos and secrets that must be created manually.

## Design

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

### Version Injection
Add `var version = "dev"` and `var buildTime = ""` in main.go. Change cmd.Execute() to accept (version, buildTime string). Set rootCmd.Version = version. Remove const appVersion from splash.go.

### GoReleaser
5 build targets: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64. LDFLAGS: `-s -w -X main.version={{.Version}} -X main.buildTime={{.Date}}`. Archives: .tar.gz for Linux/macOS, .zip for Windows. Homebrew tap, Scoop bucket, nfpms for DEB/RPM.

### GitHub Actions
- CI workflow: push to any branch, PRs targeting main. Runs make ci.
- Release workflow: push of v* tag. Runs goreleaser release --clean.
- release-please workflow: push to main. Manages version bumps and changelogs.

## Acceptance Criteria
- [ ] Version is injected at build time via LDFLAGS, not hardcoded
- [ ] spotnik --version prints the injected version string
- [ ] GoReleaser builds 5 platform targets
- [ ] CI workflow runs on push to any branch and PRs targeting main
- [ ] Release workflow triggers on v* tag push and runs GoReleaser
- [ ] release-please workflow runs on push to main
- [ ] Homebrew formula auto-pushed to initgrep-apps/homebrew-tap
- [ ] Scoop manifest auto-pushed to initgrep-apps/scoop-bucket
- [ ] DEB and RPM packages uploaded to GitHub Releases
- [ ] MIT license file exists in project root
- [ ] README includes installation instructions for all 6 channels, badges, prerequisites, keybindings, configuration, and contributing sections
- [ ] Static versioning section removed from docs/features/00-overview.md
- [ ] make ci passes with all changes

## Tasks
- [ ] Version injection -- replace hardcoded version with build-time LDFLAGS in main.go, cmd/root.go, app.go, splash.go, render.go
      - test: go test ./internal/app/... -run TestRenderSplash; make build verifies LDFLAGS
- [ ] GoReleaser configuration -- create .goreleaser.yml for 5 platforms, Homebrew, Scoop, nfpms
      - test: goreleaser check validates configuration
- [ ] GitHub Actions CI workflow -- .github/workflows/ci.yml
      - test: workflow YAML is valid
- [ ] GitHub Actions Release workflow -- .github/workflows/release.yml
      - test: workflow YAML is valid
- [ ] GitHub Actions release-please workflow -- .github/workflows/release-please.yml, manifest, config
      - test: configuration files are valid JSON
- [ ] LICENSE -- add MIT license file to project root
      - test: file exists
- [ ] README -- full rewrite with all sections
      - test: README contains installation, prerequisites, keybindings, contributing sections
- [ ] Cleanup -- remove obsolete static versioning section from docs/features/00-overview.md
      - test: section removed
- [ ] Full validation -- make ci passes, ./bin/spotnik --version prints version, all files committed
      - test: end-to-end verification
