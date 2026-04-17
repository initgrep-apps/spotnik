---
title: "CI/CD & Release"
status: open
---

## Description

GitHub Actions pipeline and GoReleaser configuration for automated multi-platform distribution. The pipeline runs lint, tests, and coverage gate on every PR. On merge to main, release-please manages version bumps and changelogs via Conventional Commits; when the maintainer merges the Release PR, GoReleaser builds binaries for macOS (amd64/arm64), Linux (amd64/arm64), and Windows (amd64), distributes via Homebrew, Scoop, DEB, and RPM, and attaches artifacts to GitHub Releases. Includes version injection, full README, and MIT license.

## Acceptance Criteria

- [ ] `make ci` (lint + test + 80% coverage) runs cleanly on GitHub Actions for every PR
- [ ] release-please creates/updates Release PR on every merge to `main`
- [ ] Merging Release PR triggers GoReleaser producing 5 platform binaries
- [ ] GitHub Release created automatically with binaries and checksums attached
- [ ] Homebrew formula pushed to `initgrep-apps/homebrew-tap`
- [ ] Scoop manifest pushed to `initgrep-apps/scoop-bucket`
- [ ] DEB and RPM packages uploaded to GitHub Releases
- [ ] `spotnik --version` prints injected version string
- [ ] MIT `LICENSE` file present in project root
- [ ] Full README with all 6 install channels present
- [ ] Pipeline fails fast on lint or coverage threshold violations
