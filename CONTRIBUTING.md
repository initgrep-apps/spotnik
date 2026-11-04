# Contributing to Spotnik

Thanks for helping. Spotnik is a keyboard-driven terminal client with a strict,
spec-first workflow. Read this before opening a PR.

## Before you start

- **Bug or small fix?** Open an issue with a reproduction first.
- **New feature?** Every feature needs a spec under `docs/spec/features/`.
  Unspecced features are not merged.
- **Security issue?** Follow [`SECURITY.md`](SECURITY.md). Never open a public
  issue for a vulnerability.
- By contributing you agree to the [Code of Conduct](CODE_OF_CONDUCT.md) and to
  license your work under MIT.

## Requirements

- Go 1.26+
- [golangci-lint](https://golangci-lint.run/)
- A terminal that supports Unicode, or run with `ui.glyphs = "ascii"`

No new dependencies without maintainer approval. Ask first: can the standard
library do this in about 30 lines?

## Setup

```bash
git clone https://github.com/initgrep-apps/spotnik.git
cd spotnik
make ci      # fmt-check, tidy-check, lint, tests, coverage, glyph guards, build
make run     # build and run
```

## Feature workflow

1. Branch from `main`. Use `feat/NN-name` for a feature and `fix/NNN-story-name`
   for a post-merge fix.
2. Work through the stories in the feature spec. One story = one commit.
3. Run `make ci` until it passes.
4. Push the branch and open one PR per feature. Title it
   `feat(name): brief description`.
5. Do not merge your own PR unless you are the maintainer. Do not work on
   `main`.

The optional worktree layout is `../spotnik-feat-NN`.

## The gate

Every PR must pass the same gate CI runs:

```bash
make ci
```

This covers formatting, `go mod tidy` drift, lint, the full test suite with the
80% coverage floor, glyph fallback guards, and a build. A red gate is not
reviewed.

## Testing rules

- Table-driven tests.
- API mocks use `httptest.NewServer`; no mock libraries.
- Fixtures live in `testdata/fixtures/` with descriptive names.
- Every `api/`, `state/`, and `config/` function needs a test.

### Golden files

The TUI uses committed golden snapshots to catch rendering regressions. If you
intentionally change pane output:

```bash
go test ./... -update
```

Review the golden file diffs and commit them with the code. Never change
rendering output without regenerating the snapshots. `make test-golden-ascii`
regenerates them in ASCII glyph mode.

## Keybindings

Keybindings are frozen. If a change is approved, update all three locations in
the same commit:

- `README.md` → Keybindings
- `docs/system/design.md` §17
- `internal/ui/panes/help_overlay.go` → `helpContent`

## Documentation

- Add or update the feature spec and story files for behavior changes.
- Update `docs/system/sanity-tests.md` when critical coverage changes.
- New messages, glyphs, or palette entries belong in `docs/system/cli.md`.
- New primitives, glyphs, or roles belong in `docs/system/tui.md`.
- Keep `docs/system/architecture.md` and `design.md` in sync with the code.

## Code style

- `gofmt` always; `make lint` enforces it.
- Exported identifiers need a doc comment.
- Comments explain why, not what. Use `// NOTE:` for non-obvious decisions and
  `// TODO(feature-name):` for planned work.
- No `panic()` in production, no `time.Sleep()` (use `tea.Tick`), no synchronous
  API calls in `Update()` or `View()`.
- `ui/` must not import `api/`, and `api/` must not import `ui/`.

## Commits

Use Conventional Commits:

```
feat(search): add episode results
fix(auth): reject mismatched OAuth state
test(queue): cover empty queue
docs(system): sync design tokens
```

Never commit non-compiling code, failing tests, lint failures, secrets, or debug
prints.

## Pull requests

- One feature per branch and PR.
- Fill in the PR template: what changed, why, testing, and TUI screenshots.
- Keep the diff focused; separate refactors from behavior changes.
- Address review comments with new commits; do not force-push during review.
- Merges use the gate rows required by the `main-protection` ruleset.

## Release process

Releases are automated with `release-please`. Squash-merged Conventional Commit
PRs feed the next Release PR, which bumps the version and updates
`CHANGELOG.md`; merging it creates the tag and runs GoReleaser. Do not tag or
publish manually.
