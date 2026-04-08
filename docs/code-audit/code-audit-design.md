# Spotnik Code Audit — 2026-04-08

**Scope:** Go best practices · design patterns · reusability · test infrastructure · documentation

**Verdict:** Architecture is sound. Elm purity is enforced, import boundaries are clean, the API gateway is sophisticated. The gaps are in onboarding, test infrastructure, file size, and a few fixable patterns.

**Action plan:** Three phased PRs — see implementation plans below.

---

## 1. Go Best Practices

### What is correct

| Area | Detail |
|------|--------|
| Error wrapping | `fmt.Errorf("context: %w", err)` used consistently at every layer |
| Typed errors | `UnauthorizedError`, `ForbiddenError`, `RateLimitError` with `errors.As()` assertions in tests |
| `atomic.Pointer` | Used for gateway injection into `BaseClient` — correct for thread-safe optional dependency |
| `sync.RWMutex` | Read-heavy store uses `RLock` for reads, `Lock` for writes consistently |
| No import cycles | `api/ → domain/`, `ui/ → domain/`, `state/ → domain/` — never circular |
| Compile-time assertions | `var _ PlayerAPI = (*Player)(nil)` in every interface file |
| Timer safety | `time.NewTimer` (not `time.After`) in token bucket to prevent goroutine leaks |
| Context propagation | All API calls thread `context.Context` through; `WithPriority` extends context values |

### What needs fixing

| Issue | Location | Severity | Fix (Phase) |
|-------|----------|----------|------------|
| `http.DefaultClient` bypass | `internal/api/auth.go:280` `postTokenRequest()` | Medium | Phase 3 |
| No `StateReader` interface | `internal/state/store.go` | Medium | Phase 2 |
| Inconsistent table-driven tests | `internal/api/player_test.go`, `library_test.go`, `playlists_test.go` | Low | Phase 2 |
| No `make test-integration` target | `Makefile` | Low | Phase 1 |

**`http.DefaultClient` detail:** `postTokenRequest` in `auth.go` calls `http.DefaultClient.Do(req)` directly, bypassing the injected `*http.Client`. This makes the token exchange untestable via `httptest.NewServer` and inconsistent with every other HTTP call in the codebase.

---

## 2. Design Patterns

### Existing patterns (working well)

| Pattern | Where used | Notes |
|---------|-----------|-------|
| **Elm Architecture** | Entire `internal/app/` | View is pure, Commands return data, Store mutated only in Update() |
| **Embedding** | All 6 API clients embed `BaseClient` | Avoids delegation boilerplate |
| **Interface per client** | `PlayerAPI`, `LibraryAPI`, etc. in `*_interfaces.go` | Enables mock injection |
| **Gateway dedup** | `internal/api/gateway.go` | Same `(Method, Path)` key → one HTTP call, all waiters share response |
| **Debounce with intent** | Panes, Search | Stale-tick detection via struct equality on intent snapshot |
| **Context priority** | `api.WithPriority(ctx, api.Interactive)` | Interactive requests bypass token bucket |
| **Generation counter** | `internal/app/app.go` preference flush | Discards stale timers without mutexes |

### Gaps / improvement opportunities

| Gap | Location | Fix (Phase) |
|-----|----------|------------|
| No `StateReader` interface | `internal/state/` | Phase 2 — panes depend on `*Store`, making unit tests heavier than needed |
| Pane boilerplate repeated | 8 panes in `internal/ui/panes/` | Phase 3 — `BasePane` embedding |
| `SetTheme()` 10-line pattern repeated | 8 panes | Phase 3 — `RebuildTableTheme()` helper |
| `gateway.go` 747 LOC does three things | `internal/api/gateway.go` | Phase 2 — split into gateway, gateway_bucket, gateway_dedup |
| `app.go` 1807 LOC does four things | `internal/app/app.go` | Phase 2 — split into app, handlers, prefs |

---

## 3. Reusability

### Well-reused components

- `components.Table` — used by all 8 Page A panes; consistent column-def + flex-factor pattern
- `components.Filter` — used by all filterable panes; routing guard via `FilterablePane.HasActiveFilter()`
- `components.Gradient`, `components.viz.*` — shared visualizer engine
- `internal/api/apitest/mock.go` — hand-written mocks for all 6 API interfaces; no external library
- `internal/api/pagination.go` — generic `fetchAll[T]` for paginated endpoints
- `layout.RenderPaneBorder` — shared border renderer used by all grid panes

### Duplicated patterns (to fix)

| Pattern | Duplication | Fix |
|---------|------------|-----|
| `os.ReadFile + json.Unmarshal` for fixtures | 15+ times in `internal/api/*_test.go` | Phase 1: `testhelpers.LoadFixture` |
| Pane struct fields `store, theme, focused, width, height` | 8 panes | Phase 3: `BasePane` |
| `SetTheme()` rebuild pattern | 8 panes | Phase 3: `RebuildTableTheme` |
| `IsFocused() bool`, `HasActiveFilter() bool` | 8 panes | Phase 3: `BasePane` |

---

## 4. Test Infrastructure

### What is working well

- 84 test files across 13 packages, all passing
- `internal/api/apitest/mock.go` — clean field-injection mocks, no external library
- `httptest.NewServer` consistently for API tests
- Dedicated purity tests: `elm_purity_test.go`, `command_safety_test.go`
- 5 integration tests with `//go:build integration` tag
- Concurrent gateway tests with channel-based release coordination
- 80% coverage gate enforced by `make ci`

### Gaps

| Gap | Impact | Fix (Phase) |
|-----|--------|------------|
| No centralised fixture loader | 15+ duplicated `os.ReadFile` calls | Phase 1: `internal/testhelpers/fixtures.go` |
| No `make test-integration` target | Integration tests are undiscoverable | Phase 1: add to Makefile |
| Inconsistent test style in `api/` | Some tests use ad-hoc structure, not tables | Phase 2: table-driven refactor |
| No `StateReader` interface | Pane tests must construct full `*Store` | Phase 2: `StateReader` makes pane tests lighter |

### Fixture loader design

`internal/testhelpers/fixtures.go` — `LoadFixture(t *testing.T, name string) []byte`

Uses `runtime.Caller(0)` to resolve the path relative to the helper file itself,
so callers in any package don't need to know the relative depth.

---

## 5. Documentation

### Accurate and complete

| Doc | Status |
|-----|--------|
| `docs/ARCHITECTURE.md` | Accurate; missing 6 sections (see below) |
| `docs/DESIGN.md` | Mostly accurate; 3 sections outdated (see below) |
| `docs/spec/` | 107 spec files; comprehensive feature planning |
| `Makefile` | Well-documented with `make help` |

### Missing sections in ARCHITECTURE.md

1. **PreferenceStore** — `internal/prefs/prefs.go` is a first-class component not mentioned anywhere
2. **Page/Preset/Toggle system** — `TogglePage()`, `CyclePreset()`, `TogglePane()` not documented
3. **View lifecycle** — `viewSplash → viewAuth → viewGrid` transitions not documented
4. **SetTheme in Pane interface** — method missing from the interface description
5. **Overlay routing precedence** — the guard order (theme > device > search > filter > grid) not documented
6. **`http.DefaultClient` known issue** — should be noted until Phase 3 fixes it

### Outdated sections in DESIGN.md

1. **§2 Pane interface** — `SetTheme(th theme.Theme)` missing from method table
2. **§17 Keybindings** — `?` (Help) listed as implemented; it is PLANNED/not-yet-implemented
3. **§18 Themes** — lists "5 existing themes"; code has 11 (dracula, gruvbox, rosepine, solarized, synthwave, tokyonight are undocumented)

### Missing docs for new contributors (all Phase 1)

| Doc | Blocks what |
|-----|-------------|
| `README.md` (currently 3 lines) | First impression; no features, no install, no quick start |
| `CONTRIBUTING.md` | PR process, commit conventions, code style, test expectations |
| `docs/DEV-SETUP.md` | Spotify credentials setup, `.env` configuration, local dev workflow |
| `docs/TESTING.md` | How to write tests, fixture conventions, mock patterns, integration tag |
| `docs/PANE-TEMPLATE.md` | Step-by-step guide for adding a new pane |

---

## Implementation Plans

| Phase | Branch | Plan file |
|-------|--------|-----------|
| Phase 1 — Docs + Test Infra | `chore/audit-phase1-docs-testinfra` | `docs/superpowers/plans/2026-04-08-phase1-docs-testinfra.md` |
| Phase 2 — Structural Refactors | `refactor/audit-phase2-structure` | `docs/superpowers/plans/2026-04-08-phase2-structural-refactors.md` |
| Phase 3 — Design Patterns | `refactor/audit-phase3-patterns` | `docs/superpowers/plans/2026-04-08-phase3-design-patterns.md` |

Each phase must pass `make ci` before the next begins.
