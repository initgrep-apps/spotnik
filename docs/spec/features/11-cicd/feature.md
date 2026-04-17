---
title: "CI/CD & Release"
status: open
---

## Description

GitHub Actions pipeline and GoReleaser configuration for automated multi-platform distribution. The pipeline runs lint, tests, and coverage gate on every PR, then on merge to main builds release binaries for macOS (amd64/arm64) and Linux (amd64) via GoReleaser. Artifacts are attached to GitHub Releases.

## Acceptance Criteria

- [ ] `make ci` (lint + test + 80% coverage) runs cleanly on GitHub Actions for every PR
- [ ] Merge to main triggers GoReleaser build producing macOS and Linux binaries
- [ ] GitHub Release created automatically with binaries attached
- [ ] Pipeline fails fast on lint or coverage threshold violations
