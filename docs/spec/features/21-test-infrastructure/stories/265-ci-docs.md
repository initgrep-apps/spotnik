---
title: "CI enforcement + docs"
feature: 21-test-infrastructure
status: open
---

## Background

Golden tests must run in CI and must be maintainable. This story wires golden tests into
`make ci`, documents the golden file protocol, updates AGENTS.md and sanity-tests.md, and
ensures the `go test -update` workflow is documented for developers.

## Design

### CI integration

Golden tests are standard Go tests — they already run with `go test ./...`. The golden
helper `AssertGolden` compares against committed files. No additional CI script needed.

Add to `Makefile` `ci` target — ensure golden tests are included in the test run (they
already are since they're `*_test.go` files).

Add a check that golden files are not stale: if tests pass with committed golden files,
CI is green. If a PR changes output without regenerating golden files, tests fail.

### `go test -update` protocol

Document in AGENTS.md and `docs/system/sanity-tests.md`:

```
When you intentionally change rendering output:
1. Run: go test ./... -update
2. Review golden file diffs in git
3. Commit regenerated golden files alongside code changes
```

### AGENTS.md updates

Add to Reading Order:
- Golden test protocol reference

Add to Never Do:
- "Change rendering output without regenerating golden files"

Add to Quick Commands:
- `go test ./... -update` — regenerate golden files

### `docs/system/sanity-tests.md` updates

Add section cross-referencing golden test coverage. Each behavioral test case gets a
note: "Golden snapshot: `TestXxxPane_View_Yyy`".

### `go.mod` — ensure teatest is direct dependency

```
require (
    github.com/charmbracelet/x/exp/teatest v0.x.x
)
```

## Files

### Modify

- `Makefile` — ensure golden tests included in `ci` target
- `AGENTS.md` — add golden test protocol, Reading Order entry, Never Do entry, Quick Commands
- `docs/system/sanity-tests.md` — add golden test cross-references
- `go.mod` / `go.sum` — teatest pinned (already added in story 256)

## Acceptance Criteria

- [ ] `make ci` runs golden tests and fails on mismatch
- [ ] `go test ./... -update` regenerates all golden files
- [ ] AGENTS.md has: Reading Order entry, Never Do entry, Quick Command for `-update`
- [ ] `docs/system/sanity-tests.md` updated with golden test cross-references
- [ ] No stale golden files in repo after final generation
- [ ] `make ci` passes

## Tasks

- [ ] Wire golden tests into `make ci`
      - test: `make ci` includes golden tests in test run
- [ ] Document golden test protocol in AGENTS.md
      - test: manual review of AGENTS.md
- [ ] Add golden test cross-references to sanity-tests.md
      - test: manual review
- [ ] Final golden file generation: `go test ./... -update`
      - test: all golden tests pass without `-update` flag
- [ ] Run full `make ci` and verify all passes
