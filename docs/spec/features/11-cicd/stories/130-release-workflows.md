---
title: "Release and release-please GitHub Actions Workflows"
feature: 11-cicd
status: open
---

## Background
Releasing is currently manual. This story adds two workflows: `release-please` automates
version bumps and changelog via Conventional Commits; the `release` workflow fires GoReleaser
when a `v*` tag is created. Depends on story 128 (GitHub Setup) for `RELEASE_PAT` secret.

## Design

### release-please workflow: `.github/workflows/release-please.yml`
**Triggers:** push to `main`
1. `googleapis/release-please-action@v4`
2. Config: `release-type: go`, `default-branch: main`
3. Analyzes Conventional Commits → creates/updates Release PR (version bump + CHANGELOG.md)
4. When maintainer merges Release PR → creates `v*` tag → triggers release workflow

### Release workflow: `.github/workflows/release.yml`
**Triggers:** push of tag matching `v*`
1. `actions/checkout@v4` with `fetch-depth: 0` (full history required by GoReleaser)
2. `actions/setup-go@v5`
3. `goreleaser/goreleaser-action@v6` — `goreleaser release --clean`
4. Uses `RELEASE_PAT` secret for cross-repo pushes to homebrew-tap and scoop-bucket

### Config files

**`.release-please-manifest.json`:**
```json
{ ".": "0.0.0" }
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

### Release flow
```
feat PR merged to main
  → release-please creates/updates Release PR
    → maintainer merges Release PR
      → v0.1.0 tag created automatically
        → Release workflow fires → GoReleaser
```

## Acceptance Criteria
- [ ] release-please workflow runs on push to `main`
- [ ] Release workflow triggers on `v*` tag push
- [ ] GoReleaser runs with `RELEASE_PAT` available
- [ ] `.release-please-manifest.json` and `release-please-config.json` present and valid JSON
- [ ] Both workflow YAMLs are valid

## Tasks
- [ ] Create `.github/workflows/release-please.yml`
      - test: YAML valid; push to main triggers workflow
- [ ] Create `.github/workflows/release.yml` with full git history checkout and GoReleaser action
      - test: YAML valid; workflow uses `RELEASE_PAT` secret
- [ ] Create `.release-please-manifest.json` starting at `0.0.0`
      - test: valid JSON; `jq . .release-please-manifest.json` succeeds
- [ ] Create `release-please-config.json` with go release type and changelog sections
      - test: valid JSON
