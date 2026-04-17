---
title: "GitHub Actions CI Workflow"
feature: 11-cicd
status: open
---

## Background
No automated CI exists. Every PR currently requires manual `make ci` before merge. This story
adds a GitHub Actions workflow that runs the full `make ci` gate on every push and PR.

## Design

### File: `.github/workflows/ci.yml`

**Triggers:** push to any branch, pull_request targeting `main`

**Job: `ci` (ubuntu-latest)**
1. `actions/checkout@v4`
2. `actions/setup-go@v5` — version read from `go.mod`
3. Cache Go modules (`~/go/pkg/mod`, `~/.cache/go-build`, key: `go-${{ hashFiles('go.sum') }}`)
4. Install `golangci-lint` via `golangci-lint-action`
5. Run `make ci` (executes: fmt-check, tidy-check, lint, test-coverage, build)

Single job, single runner. `make ci` already orchestrates the full sequence.

## Acceptance Criteria
- [ ] Workflow triggers on push to any branch
- [ ] Workflow triggers on PRs targeting `main`
- [ ] `make ci` runs and fails the workflow if lint or coverage gate fails
- [ ] Go modules are cached between runs
- [ ] Workflow YAML is valid

## Tasks
- [ ] Create `.github/workflows/ci.yml` with checkout, Go setup, module cache, golangci-lint, `make ci`
      - test: `yamllint .github/workflows/ci.yml`; push branch and confirm Actions run
