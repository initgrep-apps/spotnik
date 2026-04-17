---
title: "Cleanup and LICENSE"
feature: 11-cicd
status: done
---

## Background
Two cleanup tasks alongside the release pipeline: remove the now-obsolete `## Versioning`
section from the docs overview (replaced by tag-based semver via release-please), and add an
MIT `LICENSE` file (required for Homebrew formula and Go module proxy). The hardcoded version
constant is already removed as part of story 57.

## Design

### Remove versioning section
Check `docs/spec/00-overview.md` for a `## Versioning` section that mapped informal version
numbers to features. Tag-based semver via release-please replaces it. If the section no longer
exists (may have been removed in the spec reorganization), this task is a no-op.

### MIT LICENSE
Standard MIT license text in project root. Copyright: `2026 Irshad Sheikh`.

## Acceptance Criteria
- [ ] `LICENSE` file exists in project root with MIT license text
- [ ] No `## Versioning` section in spec or docs overview files
- [ ] `make ci` passes

## Tasks
- [ ] Add MIT `LICENSE` file to project root
      - test: `cat LICENSE` contains "MIT License" and copyright year
- [ ] Check and remove `## Versioning` section if present in `docs/spec/00-overview.md`
      - test: `grep -n "## Versioning" docs/spec/00-overview.md` returns nothing
