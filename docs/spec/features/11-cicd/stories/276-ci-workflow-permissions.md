---
title: "Add explicit GITHUB_TOKEN permissions to CI workflow"
feature: 11-cicd
status: done
---

## Background

GitHub CodeQL scanning (rule `actions/missing-workflow-permissions`) flagged `.github/workflows/ci.yml` because neither the workflow nor its jobs declare an explicit `permissions` block. When no permissions are set, the `GITHUB_TOKEN` inherits repository or organization defaults, which for older repositories is read-write. This violates least privilege and creates a fixed-permission attack surface if the workflow is copied or if defaults change.

The CI workflow only needs to:
- Check out the repository (`contents: read`).
- Run the Go toolchain, `golangci-lint`, and project `make` targets locally.

It does not open issues, write pull-request comments, push packages, or modify repository contents. Therefore the minimal required permission is `contents: read` at the workflow level.

This is a hardening fix for the existing CI/CD & Release feature (feature 11). It requires no Go code changes and no new dependencies.

## Design

Add a top-level `permissions` block to `.github/workflows/ci.yml` immediately after the `on:` section and before `jobs:`:

```yaml
permissions:
  contents: read
```

This applies to all jobs in the workflow because neither `ci` nor `tests-locale` defines its own `permissions` key. `contents: read` is sufficient for `actions/checkout` and for the workflow to execute the build/test commands.

Do **not** add job-level permissions; the workflow-level declaration keeps the file concise and is the recommended minimal fix from the CodeQL alert. The two jobs share identical needs (checkout + run commands), so a single root-level block is appropriate.

No changes to Go source, `Makefile`, golden files, or README are required.

## Files

### Modify

- `.github/workflows/ci.yml` — add `permissions: contents: read` at workflow root to satisfy least-privilege security scanning.

### Create

None.

### Delete

None.

## Acceptance Criteria

- [ ] `.github/workflows/ci.yml` contains a top-level `permissions` block with `contents: read`.
- [ ] No job in the workflow overrides or duplicates permissions unnecessarily.
- [ ] `make ci` still passes locally (workflow-only change; no code impact).
- [ ] The CodeQL `actions/missing-workflow-permissions` alert is resolved after merge.
- [ ] No new dependencies or Go source changes are introduced.

## Tasks

- [ ] Add workflow-level `permissions: contents: read` to `.github/workflows/ci.yml`.
  - verify: open `.github/workflows/ci.yml` and confirm the block appears between `on:` and `jobs:`
- [ ] Run `make ci` locally to ensure the change does not break the existing gate.
  - test: existing suite via `make ci` (lint, test-coverage, build, check-glyphs)
- [ ] Push the change and confirm the CodeQL security scanning alert closes.
