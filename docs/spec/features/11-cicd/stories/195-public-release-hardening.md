---
title: "Public Release Hardening"
feature: 11-cicd
status: open
---

## Background

 Two pre-release gaps remain before we release a public release:

1. **Privacy/transparency** — spotnik requests 14 OAuth scopes for Spotify access.
   Users have no documentation answering "what can this read or change in my account?"
2. **Supply-chain integrity** — release artifacts are protected only by sha256
   checksums. The release pipeline uses movable action tags (`@v4`, `@v5`, `@v6`) which
   can be force-moved by a compromised upstream maintainer (cf. `tj-actions/changed-files`,
   March 2025). A compromise of the release workflow could swap binaries plus checksums
   and leave users with no way to detect tampering.

This story closes both gaps in one PR.

Full design: `docs/superpowers/specs/2026-05-01-public-release-hardening-design.md`.
Implementation plan: `docs/superpowers/plans/2026-05-01-public-release-hardening.md`.

## Design

### Threat model

| Layer | Catches |
|---|---|
| sha256 checksum (already shipping) | Network corruption, accidental swap |
| GitHub artifact attestation (`actions/attest@v4`) | Whole release page replaced; binary swapped but checksums kept consistent (attacker rebuilds matching checksums after editing source) |
| SHA-pinned actions | Compromised third-party action injecting code at build time |

GitHub's `actions/attest@v4` produces a Sigstore-signed SLSA Build Provenance v1.0
attestation per artifact, verifiable via `gh attestation verify`. This is the GitHub-
native, industry-standard path; it replaces both blob-signing of `checksums.txt` and
the older specialised `actions/attest-build-provenance@v2`.

### `docs/SCOPES.md`

Standard-tier content, grouped layout:

1. **Intro** — PKCE flow, 14 scopes, keychain storage, link to revoke section.
2. **Read-only access (9 scopes)** — sub-grouped by purpose: Playback state, Library,
   Playlists, Profile, Listening history, Following.
3. **Write actions (5 scopes)** — sub-grouped: Playback control, Library modification,
   Playlist editing.
4. **What spotnik does *not* do** — no telemetry, no third-party endpoints, no
   background activity. Lists exact outbound hosts (`accounts.spotify.com`,
   `api.spotify.com`, `127.0.0.1:8888` local).
5. **How to revoke** — link to https://www.spotify.com/account/apps with two-line steps.

### README link

Append a single line to the end of the existing `## Setup` section:

> Spotnik requests 14 Spotify OAuth scopes. See [docs/SCOPES.md](docs/SCOPES.md) for
> the full list and revocation steps.

### SHA-pin GitHub Actions

Across `.github/workflows/ci.yml`, `release.yml`, `release-please.yml`:

- `actions/checkout@v4` → `@<40-char SHA> # v4.x.y`
- `actions/setup-go@v5` → `@<40-char SHA> # v5.x.y`
- `goreleaser/goreleaser-action@v6` → `@<40-char SHA> # v7.x.y` *(also bumps the
  major version; v7 already requires the GoReleaser CLI v2 release line that this
  project uses via `version: "~> v2"`, so the bump is mechanical)*
- `googleapis/release-please-action@v4` → `@<40-char SHA> # v4.x.y`

Also remove the `master` reference in the inline `golangci-lint` installer URL.
Two options — implementer picks based on whichever produces a smaller, clearer diff:
- Replace `master` with a 40-char commit SHA from the `golangci/golangci-lint` repo
  matching the installed `v2.11.3` release, **or**
- Migrate to the official `golangci/golangci-lint-action`, SHA-pinned alongside the
  other actions.

### Build artifact attestations

`release.yml` — add `actions/attest@v4` step after GoReleaser, with `id-token: write`
and `attestations: write` permissions. Use GoReleaser's recommended call shape:

```yaml
- uses: actions/attest@v4
  with:
    subject-checksums: ./dist/checksums.txt
```

This reads the `checksums.txt` GoReleaser already produces and creates one Sigstore-
signed attestation per listed artifact, published to GitHub's attestation API.

### Verification gate (rc dry-run)

Tag `v0.1.0-rc1` after merge. Per `.goreleaser.yml`'s `release.prerelease: auto`, this
publishes as a GitHub prerelease so `releases/latest` stays unaffected. Confirm:

- Build provenance attestations visible on the release page (Provenance section)
- `gh attestation verify` succeeds against at least one binary

Tag `v0.1.0` only after rc1 verification passes.

## Acceptance Criteria

- [ ] `docs/SCOPES.md` exists and lists all 14 scope strings from `internal/api/auth.go:25–30`
- [ ] README `## Setup` section links to `docs/SCOPES.md`
- [ ] No `@v4`/`@v5`/`@v6` movable tags remain in any workflow YAML; every `uses:` line pins a 40-char SHA with `# vX.Y.Z` trailing comment
- [ ] `golangci-lint` installer URL no longer references `master`
- [ ] `release.yml` declares `id-token: write` and `attestations: write` permissions
- [ ] `release.yml` runs `actions/attest@v4` with `subject-checksums: ./dist/checksums.txt` after the GoReleaser step
- [ ] `make ci` passes
- [ ] `v0.1.0-rc1` GitHub prerelease shows attestations on the release page
- [ ] `gh attestation verify` succeeds against at least one rc1 binary

## Tasks

- [ ] Create `docs/SCOPES.md` with Standard-tier content (intro, read-only group, write group, "what spotnik does not do", revoke section)
      - test: `grep -c "user-" docs/SCOPES.md` ≥ 14
      - test: every scope string from `internal/api/auth.go:25–30` appears in the doc
- [ ] Append link line to README `## Setup` section
      - test: `grep -F "[docs/SCOPES.md](docs/SCOPES.md)" README.md` returns one match
- [ ] Resolve current SHA + version for each pinned action (4 existing actions across 3 workflows + 1 new `actions/attest` + golangci-lint installer)
      - test: each pin has a 40-char hex SHA and a trailing `# vX.Y.Z` comment
- [ ] Update `ci.yml` to SHA-pin all `uses:` and the golangci-lint installer
      - test: `grep -E "uses: [^@]+@(v[0-9]+|main|master)" .github/workflows/ci.yml` returns no matches
      - test: `grep -F "golangci-lint/master" .github/workflows/ci.yml` returns no matches
- [ ] Update `release.yml` to SHA-pin all `uses:`, bump `goreleaser-action` from v6 to v7, add `actions/attest@v4` step with `subject-checksums: ./dist/checksums.txt`, set `id-token: write` and `attestations: write` permissions at the workflow level
      - test: `grep -E "uses: [^@]+@(v[0-9]+|main|master)" .github/workflows/release.yml` returns no matches
      - test: `release.yml` declares `id-token: write` and `attestations: write`
      - test: `release.yml` references `actions/attest@` and `subject-checksums: ./dist/checksums.txt`
- [ ] Update `release-please.yml` to SHA-pin the action
      - test: `grep -E "uses: [^@]+@(v[0-9]+|main|master)" .github/workflows/release-please.yml` returns no matches
- [ ] Tag `v0.1.0-rc1` from `main` after merge; let `release.yml` run end-to-end
      - test: GitHub prerelease for `v0.1.0-rc1` exists with build provenance visible on the release page
- [ ] Run `gh attestation verify` against at least one rc1 binary
      - test: command exits 0
- [ ] If rc1 verifies, tag `v0.1.0`; if not, fix the regression and tag `v0.1.0-rc2`

## Out of scope

- No `PRIVACY.md` or `SECURITY.md` (Private Vulnerability Reporting already enabled)
- No SBOM, Trivy, or CodeQL custom queries (default setup already enabled)
- No threat-model section in `docs/SCOPES.md` (Standard tier, not Full)
- No `CLAUDE.md` NEVER rule for scope drift (single-file constant; PR review catches it)
- No installer-script changes
- No cosign blob-signing of `checksums.txt`. Build attestations cover the same threat
  surface, are GitHub-native (no extra installer step, verifiable with `gh`), and
  align with SLSA Build Provenance v1.0. Trade-off: verification requires the `gh`
  CLI plus network access to Sigstore's transparency log — there is no offline
  verification path. If anyone later asks for air-gapped verification, cosign-on-
  checksums comes back as a follow-up story.
