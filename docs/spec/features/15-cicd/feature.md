---
title: "CI/CD & Release Pipeline"
status: open
---

## Description
Automated CI, cross-platform release builds, version management, and multi-channel distribution so every tagged release ships binaries to Homebrew, Scoop, DEB, RPM, go install, and GitHub Releases.

Spotnik currently has no CI/CD pipeline, no automated releases, no git tags, and a 2-line README placeholder. This feature sets up the full release pipeline so that every tag push automatically builds and ships binaries to 6 distribution channels. It also introduces release-please for automated versioning and changelog generation driven by Conventional Commits, replaces the hardcoded version with build-time injection, adds an MIT license, and rewrites the README with installation instructions for all channels.

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
- [ ] README includes installation instructions for all 6 channels
- [ ] make ci passes
