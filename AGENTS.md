# AGENTS.md — Agent Guidance for Spotnik

> **Primary guidance file for all agents working on this codebase.**
> Read this completely before writing any code or making any decision.
> This file wins if any conflict arises with other sources.

---

## What We Are Building

**Spotnik** — a terminal Spotify client for developers. Keyboard-driven, single binary,
beautiful in a terminal. Not a Spotify clone — a developer-first music environment.
Target user: developer with Spotify Premium who lives in the terminal all day.

---

## Reading Order

Read **this file** and **your feature spec**. That's it.

`docs/ARCHITECTURE.md` and `docs/DESIGN.md` are reference docs — consult them only when
the feature spec explicitly points you to a pattern or layout you need to look up.

---

## Tech Stack (Non-Negotiable)

| Layer | Choice | Notes |
|---|---|---|
| Language | Go 1.22+ | Single binary, no runtime |
| TUI | Bubble Tea v0.27+ | Elm architecture |
| Styling | Lip Gloss | Token-based, via Theme interface |
| Components | Bubbles | Lists, inputs, spinners |
| Tables | `github.com/evertras/bubble-table` | Dense sortable tables in panes |
| Overlays | `github.com/rmhubbert/bubbletea-overlay` | Modal overlay compositing |
| HTTP | Go stdlib `net/http` | No extra HTTP lib |
| Config | `github.com/BurntSushi/toml` | |
| Keychain | `github.com/zalando/go-keyring` | Token storage |
| CLI | `github.com/spf13/cobra` | Commands + subcommands |
| Testing | `testing` + `testify` | Table-driven |
| Linting | `golangci-lint` | Required gate |

**No new dependencies without explicit approval.** Ask first: can stdlib do this in ~30 lines?

---

## Go Module

```
module github.com/initgrep-apps/spotnik
go 1.22
```

---

## Project Layout

Full structure with annotations is in `docs/ARCHITECTURE.md`. Top-level overview:

```
spotnik/
├── main.go              ← entry point only, no logic
├── cmd/root.go          ← CLI flags, auth check, app launch
├── internal/
│   ├── app/             ← root Bubble Tea model
│   ├── api/             ← Spotify HTTP client, gateway, rate limiting
│   ├── domain/          ← shared types bridging api/ and ui/ (PlaybackState, Track, etc.)
│   ├── ui/
│   │   ├── panes/       ← nowplaying, queue, playlists, albums, likedsongs, toptracks,
│   │   │                   topartists, recentlyplayed, search, devices, requestflow,
│   │   │                   networklog
│   │   ├── components/  ← gradient, visualizer (viz/), filter, table, controls,
│   │   │                   notifications, infobox, timeutil
│   │   ├── layout/      ← LayoutManager, preset system, focus rotation, PaneAt hit-test
│   │   └── theme/       ← Theme interface + 5 implementations
│   ├── state/           ← central Store (single source of truth)
│   ├── config/          ← config loading + defaults
│   └── keychain/        ← token storage abstraction
├── docs/
└── testdata/fixtures/   ← JSON fixtures for API mock tests
```

---

## Spec Structure

Feature and issue specs live in `docs/spec/`:

```
docs/spec/
├── 00-overview.md              ← index of all features with status and story counts
├── issues.md                   ← quick dump for unresolved issues from PR reviews
└── features/
    └── NN-name/                ← one directory per feature
        ├── feature.md          ← high-level: title, status, description, acceptance criteria
        └── stories/
            └── NN-story-name.md  ← individual story with background, design, tasks, tests
```

- **feature.md** has YAML frontmatter: `title`, `status` (open/in-progress/done/closed)
- **Story files** have YAML frontmatter: `title`, `feature` (parent directory name), `status`
- **issues.md** collects untriaged issues — triage into feature stories when ready to fix
- **00-overview.md** is the master index — update it when adding or completing features

---

## Architecture Rules

Full patterns and code examples are in `docs/ARCHITECTURE.md`. These are the non-negotiables:

- **All API data lives in the Store** — never in a pane struct
- **Side effects only via Commands** — never call API inside `Update()` directly
- **`View()` must be pure** — no external calls, no heavy computation, just read state → string
- **Messages are typed structs** — never strings or constants as message types
- **Panes never talk to each other** — only through messages routed via root model
- **`ui/` never imports `api/`** — data flows through messages and store only
- **`api/` never imports `ui/`** — one-way dependency enforced
- **Commands must not mutate the Store** — return data in Msg payloads; only `Update()` writes to Store. Msg types carry `Data` + `Err error` fields. See `docs/ARCHITECTURE.md` "Data-Carrying Messages" section for before/after examples.
- **All API errors route through toast notifications** — use `a.alerts.NewAlertCmd(type, msg)` from `app.go` Update handlers. Pane `View()` methods must never render inline error boxes or read store error fields for display.

---

## API Rules

- Playback state: poll every **1000ms** via `tea.Tick` — never `time.Sleep`
- Search: **300ms debounce** after last keypress — never fire on every keystroke
- On `429`: back off for `Retry-After` seconds, emit `"ratelimit"` toast via `a.alerts.NewAlertCmd`
- On `401`: refresh token immediately, retry once
- On `403`: emit `"warning"` toast "Spotify Premium required"
- Always wrap errors with context: `fmt.Errorf("getting track: %w", err)`
- **All requests go through the API Gateway** — `BaseClient.doJSON`/`doNoContent` route through `*Gateway` when attached; never bypass with `http.Client.Do` directly in API methods
- Use `api.WithPriority(ctx, api.Interactive)` for user-triggered commands; `Background` is the default for polling and prefetch

---

## Testing Rules

- **80% coverage minimum** — `make test-coverage` enforces this, CI fails below threshold
- Every function in `api/`, `state/`, `config/` needs a test
- Style: **table-driven** — see `docs/ARCHITECTURE.md` for the pattern
- API mocks: use `httptest.NewServer` — no external mock libraries
- Fixtures: JSON responses in `testdata/fixtures/` named descriptively

---

## Code Style

- `gofmt` always — non-negotiable, enforced by `make lint`
- `golangci-lint` uses default rules — no custom `.golangci.yml` needed
- Exported types/funcs/consts: doc comment required
- Comments explain *why*, not *what*
- `// NOTE:` for non-obvious decisions, `// TODO(feature-name):` for planned work
- No orphaned TODOs

---

## Design Rules

Full spec is in `docs/DESIGN.md` — read it before any UI work. Hard rules:

- **Grid layout managed by LayoutManager** — 10 panes across 2 pages, configured via presets; see `docs/DESIGN.md` for full spec
- **Never hardcode hex values** — always use `Theme` interface tokens
- **Default theme is `black`** — config key `theme = "black"`
- **Keybindings are frozen** — full table in `docs/DESIGN.md` §17, update there first if changing
- **Rounded corners only** — `╭╮╰╯`, never `┌┐└┘`
- **Status bar always visible** — never hide or remove it

---

## Commit Conventions

Conventional Commits format:
```
feat(playback): add seek bar with keyboard controls
fix(auth): handle token refresh race condition
test(library): add table tests for pagination
refactor(state): extract polling into ticker command
chore(deps): upgrade bubbletea to v0.27.1
```

Never commit: non-compiling code · failing tests · lint failures · hardcoded secrets · debug prints

---

## Feature Development Workflow

Follow this sequence exactly for every feature — no shortcuts.

```
1. git checkout main && git pull origin main
2. git checkout -b feat/NN-feature-name   (e.g. feat/03-playback)
3. Implement tasks from the feature spec, one commit per completed task
4. make ci   ← must pass fully (lint + tests + 80% coverage)
5. git push origin feat/NN-feature-name
6. Open a PR: title = "feat(name): brief description"  body = tasks completed + test summary
7. STOP — do not merge unless you are the orchestrator agent running `/orchestrate`.
8. After merge confirmed: git checkout main && git pull origin main
```

**Hard rules:**
- Never work directly on `main`
- Never merge your own PR — unless you are the orchestrator after external review passes
- A failing `make ci` blocks the PR step — fix before pushing
- One feature per branch — never mix features in a branch

---

## What Agents Must NEVER Do

1. Store credentials or secrets in tracked files
2. Bypass the LayoutManager — all pane geometry flows through it
3. Add a feature not in `docs/spec/features/` without creating a spec first
4. Call API synchronously from `View()` or `Update()`
5. Skip writing tests for new `api/`, `state/`, `config/` code
6. Change keybindings without updating `docs/DESIGN.md`
7. Use `panic()` in production code paths
8. Use `time.Sleep()` — use `tea.Tick`
9. Import `ui/` from `api/` or `api/` from `ui/`
10. Hardcode hex colour values in component code
11. Add a theme without implementing every method of the `Theme` interface
12. Work directly on `main` — always use a feature branch
13. Merge a PR — unless you are the orchestrator agent after external review passes
14. Render inline error boxes in pane `View()` methods — all API errors go through toast notifications via `a.alerts.NewAlertCmd`
15. Add, change, or remove a keybinding without updating all three locations in the
    same commit: `docs/keybinding.md`, `docs/DESIGN.md §17`, and the `helpContent` var in
    `internal/ui/panes/help_overlay.go`.

---

## Keybinding Maintenance

All keybindings are documented in three places that must stay in sync:
- `docs/keybinding.md` — human-readable reference (canonical for external readers)
- `docs/DESIGN.md §17` — spec-level keybinding table
- `internal/ui/panes/help_overlay.go` `helpContent` var — in-app help overlay display

When adding, changing, or removing any keybinding, update all three in the same commit.

---

## Feature Order

See `docs/spec/00-overview.md` for the full map. Short version:
`01-theme` → `02-auth` → `03-playback` → `04-library` → `05-search`
→ `06-queue` → `07-devices` → `08-stats` → `09-playlists` → `10-error-resilience`
→ `11-api-gateway` → `12-layout` → `13-nowplaying` → `14-nerd-status` → `15-cicd`

Do not start a feature until the previous one has passing tests and is committed.

---

## Quick Commands

```bash
make build     # compile → bin/spotnik
make run       # build + run
make test      # all tests
make lint      # golangci-lint
make test-coverage  # coverage report (min 80%)
make ci        # full pre-commit check
```

