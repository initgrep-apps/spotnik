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
