# Spotnik — Issues / Follow-ups

> Placeholder for unresolved items captured during PR reviews and triage.
> Triage into feature stories when ready to fix.

---

## Release-please configuration before first public release

**Captured:** 2026-04-30
**Files:** `release-please-config.json`, `.release-please-manifest.json`

### Why this is open
The CI/CD pipeline (feature 11) was wired up before the project ever produced
a release. `bootstrap-sha` is currently set to `326d6d5`, which leaves ~462
commits between bootstrap and HEAD. If release-please is left as-is, the first
v0.1.0 CHANGELOG will be a wall of every feat/fix since that SHA.

### Decision needed before tagging v0.1.0
Pick one path:

1. **Move `bootstrap-sha` forward** to the merge commit of the public-release
   prep branch (or a similar deliberate point) so v0.1.0 contains only the
   commits you want surfaced.
2. **Hand-write v0.1.0** — manually `git tag v0.1.0`, push the tag, edit
   the GitHub Release notes by hand. Then update `bootstrap-sha` to that tag
   so release-please takes over for v0.1.1 onward.

### Other small follow-ups in the same area
- The `release-please.yml` workflow trigger has the `push: branches: main`
  block commented out — uncomment when ready for automated release PRs.
- Consider adding `{ "type": "docs", "section": "Documentation" }` to
  `changelog-sections` if doc-only commits should appear in CHANGELOG.
- After v0.1.0 ships, `bootstrap-sha` becomes irrelevant; can be removed
  from the config to reduce noise.

### Reference
Discussion in agent session 2026-04-30 covers the trade-offs in detail.

---

## Installer script polish (post-194 review)

**Captured:** 2026-04-30
**Source:** PR #251 Review
**Feature:** 11-cicd

### Why this is open
Five minor polish items from the round-1 PR review of story 194 installer scripts. Not blocking and can be triaged into a follow-up story when convenient.

### Items

1. `install.sh` and `install.ps1`: post-install `--version` check swallows binary execution failures (the `2>/dev/null || echo` pattern in bash and `2>$null` in PowerShell silently substitute the expected version string even if the freshly-installed binary fails to launch — defeating the post-install sanity check).
2. `install.ps1` cleanup uses `-ErrorAction SilentlyContinue` for the temp dir `Remove-Item` — repeated cleanup failures (e.g. AV scanner has zip locked) accumulate `%TEMP%\spotnik-install-*` directories forever with no signal.
3. Env var `SPOTNIK_NO_MODIFY_PATH` is misleading — the script never modifies PATH, only warns. Consider renaming to `SPOTNIK_NO_PATH_WARN` (or document the discrepancy with a NOTE comment).
4. `install.ps1` PATH update writes user PATH as `REG_SZ` rather than `REG_EXPAND_SZ` via `[Environment]::SetEnvironmentVariable` — any pre-existing `%USERPROFILE%`-style entries stop expanding. Rare in user PATH.
5. Spec story 194 mentions "Delete RELEASE_PAT secret (manual step)" but this lacks an explicit acceptance criterion or post-merge checklist task — easy to forget. Consider promoting to a tracked task.
