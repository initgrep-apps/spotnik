---
title: "GoReleaser Configuration"
feature: 11-cicd
status: done
---

## Background
No `.goreleaser.yml` exists. GoReleaser is referenced in the release workflow (story 130) but
has no config. This story creates the full configuration: 5 build targets, archives, checksums,
Homebrew tap, Scoop bucket, and DEB/RPM packages. Depends on story 128 for `homebrew-tap` and
`scoop-bucket` repos to exist.

## Design

### `.goreleaser.yml`

**Builds** (5 targets):

| OS | Arch | Archive |
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

**Archives:** `spotnik_{{.Version}}_{{.Os}}_{{.Arch}}` — `.tar.gz` (linux/darwin), `.zip` (windows)

**Checksums:** SHA256, filename `spotnik_{{.Version}}_checksums.txt`

**Homebrew tap:**
- `github.com/initgrep-apps/homebrew-tap` — formula name: `spotnik`
- Pushed via `RELEASE_PAT`

**Scoop bucket:**
- `github.com/initgrep-apps/scoop-bucket` — manifest name: `spotnik`
- Pushed via `RELEASE_PAT`

**nfpms (DEB/RPM):**
- Package name: `spotnik`, maintainer: `irshad.mike@gmail.com`
- License: MIT, installs to `/usr/bin/spotnik`
- Formats: `deb`, `rpm`

**Changelog:** skipped — release-please owns `CHANGELOG.md`

## Acceptance Criteria
- [ ] `goreleaser check` passes with no errors
- [ ] 5 build targets defined
- [ ] Homebrew and Scoop publishers configured with correct repo targets
- [ ] DEB and RPM nfpm configs present
- [ ] Changelog generation disabled

## Tasks
- [ ] Create `.goreleaser.yml` with builds, archives, checksums, homebrew, scoop, nfpms sections
      - test: `goreleaser check` exits 0; config covers all 5 platforms
