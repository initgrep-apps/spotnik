# Contributing to Spotnik

Thank you for your interest in contributing. This document covers everything you need to
know before opening a PR.

---

## Branch Naming

Branches must follow the `<type>/NN-short-description` pattern:

```
feat/57-search-pagination
fix/36-command-safety-errors
refactor/22-gateway-split
chore/109-onboarding-docs
```

Always branch from `main`. Never work directly on `main`.

---

## Commit Conventions

Spotnik uses [Conventional Commits](https://www.conventionalcommits.org/). Every commit
message must follow this format:

```
<type>(<scope>): <short description>

[optional body]
```

**Types:** `feat`, `fix`, `refactor`, `test`, `chore`, `docs`

**Scopes** (pick the nearest): `playback`, `auth`, `state`, `api`, `search`, `queue`,
`library`, `stats`, `playlists`, `devices`, `theme`, `layout`, `panes`, `gateway`, `ci`

**Examples:**

```
feat(playback): add seek bar with keyboard controls
fix(auth): handle token refresh race condition
test(library): add table tests for pagination
refactor(state): extract polling into ticker command
chore(deps): upgrade bubbletea to v0.27.1
```

**Rules:**
- Never commit non-compiling code, failing tests, lint failures, or debug prints
- One logical change per commit
- The description is lowercase and does not end with a period

---

## CI Gate

Before pushing, run:

```bash
make ci
```

This runs `fmt-check → tidy-check → lint → test-coverage → build` in order.
**All checks must pass.** A failing `make ci` blocks the PR step.

---

## Test Requirements

- **80% coverage minimum** — enforced by `make test-coverage`
- Every function in `api/`, `state/`, `config/` must have a test
- Use **table-driven tests** — see [docs/TESTING.md](docs/TESTING.md)
- Use `testhelpers.LoadFixture(t, "name.json")` to load fixtures from
  `testdata/fixtures/` — never inline `os.ReadFile` calls for fixtures
- Use `httptest.NewServer` for API client tests — no external mock libraries
- Integration tests use `//go:build integration` tag and live in `*_integration_test.go`
  files; run with `make test-integration`

---

## Architecture Rules

The codebase follows the Elm Architecture via Bubble Tea. Key rules that reviewers enforce:

- **All API data lives in the Store** — never in a pane struct
- **Side effects only via Commands** — never call API inside `Update()` directly
- **`View()` must be pure** — read state → string, no external calls
- **`ui/` never imports `api/`** — data flows through messages and store only
- **Commands must not mutate the Store** — return data in Msg payloads; only `Update()` writes to Store
- **All API errors route through toast notifications** — use `a.alerts.NewAlertCmd(type, msg)`
- **Never hardcode hex colour values** — always use `Theme` interface tokens
- **Adding a theme** requires implementing every method of the `Theme` interface

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for deeper context.

---

## Keybinding Changes

If you add, change, or remove any keybinding you **must** update all three locations in
the same commit:

1. `docs/keybinding.md` — human-readable reference
2. `docs/DESIGN.md §17` — spec-level keybinding table
3. `internal/ui/panes/help_overlay.go` `helpContent` var — in-app display

---

## PR Process

1. `git checkout main && git pull origin main`
2. Create a branch following the naming convention above
3. Implement the change with one commit per completed task (from the story spec)
4. Run `make ci` — must pass fully
5. Push: `git push -u origin <branch>`
6. Open a PR with title `feat(scope): description` or `fix(scope): description`
7. PR body should summarise tasks completed, test coverage achieved, and acceptance
   criteria checked

**Do not merge your own PR** unless you are the orchestrator agent after external review
passes.

---

## Adding a New Pane

See [docs/PANE-TEMPLATE.md](docs/PANE-TEMPLATE.md) for the step-by-step checklist.
