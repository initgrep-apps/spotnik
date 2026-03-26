# GitHub Setup — Manual Prerequisites for CI/CD

> **Do these steps before implementing Feature 57 (CI/CD & Release Pipeline).**
> These are manual actions in the GitHub UI that cannot be automated.

---

## 1. Create Homebrew Tap Repository

1. Go to [github.com/organizations/initgrep-apps/repositories/new](https://github.com/organizations/initgrep-apps/repositories/new)
2. Repository name: `homebrew-tap`
3. Visibility: **Public** (required for `brew install` to work)
4. Check "Add a README file"
5. Click "Create repository"

GoReleaser will auto-push the Homebrew formula to this repo on each release.
Users will install via: `brew install initgrep-apps/tap/spotnik`

---

## 2. Create Scoop Bucket Repository

1. Same location: [github.com/organizations/initgrep-apps/repositories/new](https://github.com/organizations/initgrep-apps/repositories/new)
2. Repository name: `scoop-bucket`
3. Visibility: **Public**
4. Check "Add a README file"
5. Click "Create repository"

GoReleaser will auto-push the Scoop manifest here on each release.
Users will install via: `scoop bucket add spotnik https://github.com/initgrep-apps/scoop-bucket && scoop install spotnik`

---

## 3. Create a Fine-Grained Personal Access Token

1. Go to [github.com/settings/personal-access-tokens/new](https://github.com/settings/personal-access-tokens/new)
2. Token name: `spotnik-release`
3. Expiration: choose a reasonable expiry (e.g. 1 year)
4. Resource owner: `initgrep-apps`
5. Repository access: **Only select repositories** → select `homebrew-tap` and `scoop-bucket`
6. Permissions → Repository permissions → **Contents: Read and write**
7. Click "Generate token"
8. **Copy the token** — you won't see it again

This PAT allows GoReleaser to push formula/manifest updates to the tap and bucket repos.

---

## 4. Add Repository Secret to Spotnik

1. Go to [github.com/initgrep-apps/spotnik/settings/secrets/actions](https://github.com/initgrep-apps/spotnik/settings/secrets/actions)
2. Click "New repository secret"
3. Name: `RELEASE_PAT`
4. Value: paste the PAT from step 3
5. Click "Add secret"

The release workflow uses this secret to authenticate cross-repo pushes.

---

## Verification

After completing all 4 steps, verify:

- [ ] `initgrep-apps/homebrew-tap` repo exists and is public
- [ ] `initgrep-apps/scoop-bucket` repo exists and is public
- [ ] `RELEASE_PAT` secret exists in spotnik repo settings
- [ ] PAT has `Contents: Read and write` on both tap and bucket repos

Once verified, Feature 57 can be implemented.

---

**Related:**
- Feature spec: `docs/features/57-cicd-release-pipeline.md`
- Implementation plan: `docs/superpowers/plans/2026-03-26-cicd-release-readme.md`
- Design spec: `docs/superpowers/specs/2026-03-26-cicd-release-readme-design.md`
