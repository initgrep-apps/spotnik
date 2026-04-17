---
title: "GitHub Setup — Manual Prerequisite"
feature: 11-cicd
status: open
---

## Background
The release workflow requires two public GitHub repos (`homebrew-tap`, `scoop-bucket`) and a
fine-grained PAT with write access to both. These cannot be created by automation — they are
one-time manual steps that must be completed before stories 130 and 131 can be end-to-end
validated. Full instructions are already in `docs/GITHUB-SETUP.md`.

## Design
No code changes. This is a prerequisite verification gate.

### Steps (see `docs/GITHUB-SETUP.md` for full detail)
1. Create `initgrep-apps/homebrew-tap` repo — public, initialize with README
2. Create `initgrep-apps/scoop-bucket` repo — public, initialize with README
3. Create fine-grained PAT `spotnik-release` scoped to `initgrep-apps`, repo access:
   `homebrew-tap` and `scoop-bucket`, permission: `Contents: Read and write`
4. Add `RELEASE_PAT` secret to `initgrep-apps/spotnik` repo settings

## Acceptance Criteria
- [ ] `initgrep-apps/homebrew-tap` exists and is public
- [ ] `initgrep-apps/scoop-bucket` exists and is public
- [ ] `RELEASE_PAT` secret present in spotnik repo settings
- [ ] PAT has `Contents: Read and write` on both tap and bucket repos

## Tasks
- [ ] Create `initgrep-apps/homebrew-tap` repo (manual GitHub UI)
      - test: repo exists and is public
- [ ] Create `initgrep-apps/scoop-bucket` repo (manual GitHub UI)
      - test: repo exists and is public
- [ ] Create fine-grained PAT with correct scope (manual GitHub UI)
      - test: token generated and copied
- [ ] Add `RELEASE_PAT` secret to spotnik repo (manual GitHub UI)
      - test: secret visible in repo settings > Actions > Secrets
