# AGENTS.md — Spotnik

> Primary guidance. Read before any code/decision.
> Conflict? This wins.

**Caveman mandatory** — respond terse, technical, no fluff. Agent must use caveman skill always. Only drop caveman when user says "explain" or "clarify". Code/commits written normal.

---

## What

Spotnik — terminal Spotify client for devs. Keyboard-driven, single binary.
Target: dev with Spotify Premium, lives in terminal.

---

## Reading Order

Read **this file** + **feature spec** first.

Reference docs — consult only when spec says so:
- `docs/system/architecture.md` — patterns, code examples
- `docs/system/design.md` — layout, keybindings, theme
- `docs/system/cli.md` — message types, glyphs, palette
- `docs/system/tui.md` — primitives, glyph catalogue, roles
- `docs/system/api-guide.md` — Spotify API endpoints & scope

---

## Tech Stack

| Layer | Choice |
|---|---|
| Language | Go 1.26+ |
| TUI | Bubble Tea v1.3+ |
| Styling | Lip Gloss |
| Components | Bubbles |
| Tables | `github.com/evertras/bubble-table` |
| Overlays | `github.com/rmhubbert/bubbletea-overlay` |
| HTTP | Go stdlib `net/http` |
| Config | `github.com/BurntSushi/toml` |
| Keychain | `github.com/zalando/go-keyring` |
| CLI | `github.com/spf13/cobra` |
| Testing | `testing` + `testify` |
| Linting | `golangci-lint` |

**No new deps without approval.** Ask: can stdlib do this in ~30 lines?

---

## Go Module

```
module github.com/initgrep-apps/spotnik
go 1.26
```

---

## Project Layout

Full structure in `docs/system/architecture.md`. Top-level:

```
spotnik/
├── main.go              ← entry point only
├── cmd/root.go          ← CLI flags, auth check, app launch
├── internal/
│   ├── app/             ← root Bubble Tea model
│   ├── api/             ← Spotify HTTP client, gateway, rate limiting
│   ├── domain/          ← shared types (PlaybackState, Track, etc.)
│   ├── ui/
│   │   ├── panes/       ← nowplaying, queue, playlists, search, devices, etc.
│   │   ├── components/  ← gradient, viz/, filter, table, controls, notifications
│   │   ├── layout/      ← LayoutManager, presets, focus rotation, PaneAt
│   │   └── theme/       ← Theme interface + 5 implementations
│   ├── state/           ← central Store (single source of truth)
│   ├── config/          ← config loading + defaults
│   └── keychain/        ← token storage
├── docs/
└── testdata/fixtures/   ← JSON fixtures for API mocks
```

---

## Spec Structure

Feature/issue specs in `docs/spec/`:

```
docs/spec/
├── 00-overview.md       ← master index (update on add/complete)
├── issues.md            ← untriaged issues
└── features/NN-name/
    ├── feature.md        ← title, status, description, AC
    └── stories/NN-story-name.md ← background, design, tasks, tests
```

YAML frontmatter: `title`, `status` (open/in-progress/done/closed), `feature` (stories only)

---

## Architecture Rules

- All API data in Store — never in pane structs
- Side effects only via Cmd (`tea.Cmd`) — never call API inside `Update()`
- `View()` pure: read state → string only. No ext calls, no heavy compute.
- Messages = typed structs. Never strings/constants as msg type.
- Panes isolated: communicate only through root-routed Msgs
- `ui/` never imports `api/` (and vice versa) — one-way dependency
- Cmds never mutate Store. Msg payloads carry `Data` + `Err`.
- All API errors → toast notifications via `a.alerts.NewAlertCmd`. Never inline error boxes in `View()`.

---

## API Rules

- Playback poll: **1000ms** via `tea.Tick` — never `time.Sleep`
- Search: **300ms debounce** after keypress
- `429` → `Retry-After` backoff + `"ratelimit"` toast
- `401` → refresh token, retry once
- `403` → `"warning"` toast "Spotify Premium required"
- Wrap errors: `fmt.Errorf("context: %w", err)`
- All requests via `BaseClient.doJSON`/`doNoContent` → `*Gateway` when attached. Never `http.Client.Do` directly.
- Priority: `api.WithPriority(ctx, api.Interactive)` for user-triggered, `Background` for polling/prefetch.

---

## Testing Rules

- **80% coverage min** — `make test-coverage` enforces, CI fails below
- Every `api/`, `state/`, `config/` function needs a test
- Style: **table-driven**
- API mocks: `httptest.NewServer` (no mock libs)
- Fixtures: JSON in `testdata/fixtures/`, descriptive names

---

## Code Style

- `gofmt` always — enforced by `make lint`
- `golangci-lint` default rules
- Exports: doc comment required
- Comments explain *why*, not *what*
- `// NOTE:` for non-obvious decisions, `// TODO(feature-name):` for planned work
- No orphaned TODOs

---

## Design Rules

Full spec in `docs/system/design.md`. Hard rules:
- Grid via **LayoutManager** — 10 panes, 2 pages, presets
- Never hardcode hex — use `Theme` interface tokens
- Default theme: `black` (config key `theme = "black"`)
- Keybindings frozen — table in design.md §17
- Rounded corners only (`╭╮╰╯`, never `┌┐└┘`)
- Status bar always visible

---

## Commit Conventions

Conventional Commits format:
```
type(scope): description
```
Types: feat / fix / test / refactor / chore / docs.

Never commit: non-compiling code · failing tests · lint failures · secrets · debug prints

---

## Feature Dev Workflow

### Orchestrate (standard)
- Branch: `feat/NN-name`. Worktree: `../spotnik-feat-NN`. Stories = commits.
- Orchestrator creates worktree + branch. Feature-implementer works inside.
- PR per feature (not per story). Post-merge fix: `fix/NNN-story-name`.

### Standalone / manual
1. `git pull origin main && git checkout -b feat/NN-name`
2. Implement — one commit per task
3. `make ci` (lint + tests + 80% coverage — must pass)
4. `git push origin feat/NN-name`
5. Open PR: title = `feat(name): brief`, body = tasks + test summary
6. STOP — don't merge unless you're orchestrator
7. After merge: `git checkout main && git pull origin main`

**Hard rules:**
- Never work on `main`. Never merge own PR (unless orchestrator).
- Failing `make ci` blocks PR. One feature per branch.
- No sub-branches. Post-merge fix stories from main.

---

## Never Do

1. Store credentials/keys in tracked files
2. Bypass LayoutManager
3. Add unspecced features (no feature.md = no code)
4. Sync API calls in `View()` or `Update()`
5. Skip tests for `api/`/`state/`/`config/` code
6. Change keybindings without 3-file update (README + design.md §17 + help_overlay.go)
7. Use `panic()` in production
8. Use `time.Sleep()` — use `tea.Tick`
9. Cross-import `ui/`↔`api/`
10. Hardcode hex colors
11. Add theme without implementing full Theme interface
12. Work on main
13. Merge own PR (unless orchestrator)
14. Inline error boxes in `View()` — use `a.alerts.NewAlertCmd` toasts
15. Add msgs/glyphs without updating `docs/system/cli.md`
16. Add primitives/glyphs/roles without updating `docs/system/tui.md`
17. Modify keybindings without sync'ing all 3 locations

---

## Keybinding Maintenance

Three locations, same commit:
- `README.md` **Keybindings** — user-facing
- `docs/system/design.md §17` — spec table
- `internal/ui/panes/help_overlay.go` `helpContent` — in-app overlay

---

## Feature Order

```
01-theme → 02-auth → 03-playback → 04-library → 05-search
→ 06-queue → 07-devices → 08-stats → 09-playlists
→ 10-error-resilience → 11-api-gateway → 12-layout
→ 13-nowplaying → 14-nerd-status → 15-cicd
```

Don't start next feature until previous has passing tests + committed.

---

## Quick Commands

```
make build         compile → bin/spotnik
make run           build + run
make test          all tests
make lint          golangci-lint
make test-coverage coverage (min 80%)
make ci            full pre-commit check
```

---

## Delegation

Main agent = orchestrator. Stay lean (first 3-5 msgs at top).
Token-heavy tasks → delegate to subagent. Always synthesize results.

### When to Delegate

- Code exploration, multi-file search, large file reads
- Architecture design or code review
- Test debugging (2+ files) or feature implementation
- Anything requiring 5+ file interactions

### Subagent Decision Table

| Trigger | Subagent | Prompt Recipe | Handle Output |
|---------|----------|---------------|---------------|
| Code exploration: "how X works", "find Y", "trace Z" | `explore` | thoroughness (medium/very) + scope + pattern | Compact file:line + brief summary |
| Multi-file search: "grep X across N files" | `explore` | pattern + dir + file types | Compact file list |
| Read large file (>100 lines) | `Task(read+summarize)` | path + what to extract | Minimal relevant summary |
| Architecture: "design X component" | `code-architect` | existing patterns + constraints | Files to create/modify, data flow |
| Code review: "review diff/branch" | `code-reviewer` or `pr-*` suite | diff + focus areas | ≥80 confidence findings |
| Test debug: failures in 2+ files | `Task` per file (parallel) | error msg + test name per file | Root cause + fix per file |
| Feature: "implement story N" | `feature-implementer` | story spec + branch + worktree | CI-passing commits |
| Bug: "is there a bug in X?" | `code-reviewer` / `cavecrew-reviewer` | file + suspicion | ≥80 confidence findings |

### Delegation Rules

1. **One subagent per independent domain** — never bundle unrelated work in one agent.
2. **Self-contained prompts** — include all context needed, never rely on session history.
3. **Parallel dispatch** — for independent work (e.g. 3 failing test files), dispatch simultaneously.
4. **Review then integrate** — after subagent returns: read summary, check conflicts, merge.
5. **Always synthesize** — return 2-3 sentence summary to user, never raw subagent dump.
6. **Verify after edits** — if subagent touched files, run `make ci` before claiming done.

---

<!-- rtk-instructions v2 -->
# RTK — Token-Optimized Commands

**Always prefix commands with `rtk`.** If RTK has a dedicated filter, it uses it. If not, passes through unchanged. Always safe.

Even in `&&` chains: `rtk git add . && rtk git commit -m "msg" && rtk git push`

### Commands by Workflow

#### Build & Compile (80-90% savings)
```
rtk cargo build / check / clippy
rtk tsc
rtk lint
rtk prettier --check
rtk next build
```

#### Test (60-99% savings)
```
rtk cargo test     rtk go test     rtk jest
rtk vitest         rtk playwright  rtk pytest
rtk rspec          rtk test <cmd>
```

#### Git (59-80% savings)
```
rtk git status     rtk git log      rtk git diff
rtk git show       rtk git add      rtk git commit
rtk git push       rtk git pull     rtk git branch
rtk git stash      rtk git worktree
```
Git passthrough works for ALL subcommands, even unlisted.

#### GitHub (26-87% savings)
```
rtk gh pr view <num>    rtk gh pr checks
rtk gh run list         rtk gh issue list
rtk gh api
```

#### JS/TS (70-90% savings)
```
rtk pnpm list / outdated / install
rtk npm run <script>
rtk npx <cmd>
rtk prisma
```

#### Files & Search (60-75% savings)
```
rtk ls <path>           rtk read <file>
rtk grep <pattern>      rtk find <pattern>
```
Format flags (-c, -l, -L, -o, -Z) on `grep` run raw.

#### Analysis & Debug (70-90% savings)
```
rtk err <cmd>           rtk log <file>
rtk json <file>         rtk deps
rtk env                 rtk summary <cmd>
rtk diff
```

#### Infra (85% savings)
```
rtk docker ps / images / logs <c>
rtk kubectl get / logs
```

#### Network (65-70% savings)
```
rtk curl <url>          rtk wget <url>
```

#### Meta
```
rtk gain            rtk gain --history
rtk discover        rtk proxy <cmd>
rtk init            rtk init --global
```

### Savings Reference

| Category | Commands | Savings |
|----------|----------|---------|
| Test | vitest, playwright, cargo test | 90-99% |
| Build | next, tsc, lint, prettier | 70-87% |
| Git | status, log, diff, add, commit | 59-80% |
| GitHub | gh pr, gh run, gh issue | 26-87% |
| JS/TS | pnpm, npm, npx | 70-90% |
| Files | ls, read, grep, find | 60-75% |
| Debug | err, log, json, deps, env, diff | 70-90% |
| Infra | docker, kubectl | 85% |
| Network | curl, wget | 65-70% |

Overall: **60-90% token reduction** on common operations.
<!-- /rtk-instructions -->
