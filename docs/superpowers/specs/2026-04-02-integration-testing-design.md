# Integration Testing for Spotnik — Research & Decision Document

**Date:** 2026-04-02
**Status:** Approved
**Scope:** Research, tooling decision, and testable workflow catalog

---

## 1. Problem Statement

Spotnik has 74 test files with 80%+ coverage, all using synchronous unit tests: direct `Update(msg)` calls, assert on returned `Cmd`, check `View()` output. This approach verifies individual state transitions but cannot test:

- **Multi-step user workflows** — sequences of inputs that produce chained commands feeding back as messages
- **Async command chains** — prefetch pagination, debounce timing, polling intervals
- **Cross-pane effects** — playing a track from search updating now-playing and queue
- **Focus routing correctness** — overlay open/close with proper focus transfer and restoration
- **Time-dependent behavior** — debounce (300ms), adaptive polling (3s/10s/30s), rate-limit backoff
- **Visual regression** — rendered output changing unexpectedly after refactoring

Web UIs solve this with tools like Cypress and Playwright. TUI apps need an equivalent layer.

---

## 2. Current Testing Architecture

| Layer | Tool | What It Tests |
|---|---|---|
| API client | `httptest.NewServer` + table-driven tests | HTTP request/response parsing, error codes |
| State/Store | `testify` assertions | Getter/setter correctness, staleness gates |
| Config | `t.TempDir()` + TOML fixtures | Load/save, validation, defaults |
| Pane/Component | Direct `Update()` + `View()` calls | Single message handling, key bindings, rendering |
| App model | Direct `Update()` + store inspection | Message routing, command dispatch |
| Architecture | `elm_purity_test.go`, `command_safety_test.go` | Elm invariants (commands don't mutate store) |

**What's missing:** No test runs the actual Bubble Tea event loop. Commands are executed manually and their results fed back one at a time. There is no way to test that a sequence of user inputs produces the correct final state through the real Init → Update → View cycle.

---

## 3. Tool Evaluation

### 3.1 Teatest (`charmbracelet/x/exp/teatest`)

Official Bubble Tea testing package from the Charm team. Runs the real Bubble Tea event loop in a test harness.

**Key capabilities:**
- `NewTestModel(t, model)` — wraps any `tea.Model` in a test harness with configurable terminal size
- `Type(string)` — simulates keyboard typing into the running model
- `Send(tea.Msg)` — injects messages directly into the event loop
- `WaitFor(output, predicate)` — blocks until rendered output matches a condition
- `FinalModel()` / `FinalOutput()` — captures final state after quit
- `RequireEqualOutput()` — golden file snapshot comparison with `-update` flag
- Commands execute naturally through the event loop (debounce ticks fire, API callbacks arrive)

**Status:** Experimental (`charmbracelet/x/exp/`). API may change between versions but the package is actively maintained and used by the Charm team internally.

**Why teatest over alternatives:**

| Alternative | Why Not |
|---|---|
| Manual `Update()` chains | Already have this. Can't test async timing or command chains naturally. |
| Custom PTY harness (`creack/pty`) | Significant build effort, ANSI parsing complexity, extra dependency for marginal benefit over teatest. |
| VHS (`.tape` recordings) | Declarative only — no programmatic assertions. Requires FFmpeg. Good for demos, not for CI test suites. |
| Build from scratch | Would replicate what teatest already does. |

### 3.2 Decision

**Use teatest as the sole integration test driver.** Rationale:

1. **Single dependency** — handles pane-level, root-model, and full-app testing with one API
2. **Runs the real event loop** — commands, ticks, and messages flow naturally
3. **Golden file support** — built-in snapshot comparison for visual regression
4. **Spotnik's binary is trivial** — `main.go` is 5 lines, `cmd/root.go` creates the model and calls `tea.NewProgram()`. Testing the model directly catches 99% of what binary E2E would catch.
5. **Consistent with existing patterns** — still Go tests, still `go test ./...`, still table-driven where appropriate
6. **Low risk** — experimental status means possible API changes, but migration surface is test helpers only (no production code affected)

---

## 4. Testing Pyramid for Spotnik

```
           /  Full App  \        few tests, slow — complete app model with all panes + mocked API
          /--------------\
         / Root Model     \      core tests — cross-pane workflows, overlay routing, error recovery
        /------------------\
       / Pane-Level         \    many tests — isolated pane workflows with real event loop
      /----------------------\
     / Unit Tests (existing)  \  foundation — Update() + View() + httptest (keep as-is)
    /--------------------------\
```

Each layer up adds coverage for interactions the layer below cannot test. All layers coexist — unit tests are not replaced.

---

## 5. Testable Workflows Catalog

Below is every testable workflow organized by feature area. Each describes the user-visible flow, not implementation details.

### 5.1 Search

| # | Workflow | Priority |
|---|---|---|
| S-1 | Open search → type query → debounce fires → results appear | High |
| S-2 | Type query → results load → switch tab (Songs/Artists/Albums/Playlists) → results refresh for new type | High |
| S-3 | Type query → scroll down past 60% → next page prefetches automatically | High |
| S-4 | Switch tab mid-prefetch → stale pages discarded, fresh search fires | High |
| S-5 | Select track → press Enter → track plays → overlay closes | High |
| S-6 | Select track → press Ctrl+A → track added to queue → confirmation shown | Medium |
| S-7 | Press Ctrl+U → input clears → results clear | Medium |
| S-8 | Type rapidly (keystrokes every 50ms) → only one search per 300ms debounce window | Medium |
| S-9 | Search returns fewer results than page size → no further prefetch triggered | Medium |
| S-10 | Press Esc → overlay closes → focus returns to previously focused pane | High |

### 5.2 Playback (Now Playing)

| # | Workflow | Priority |
|---|---|---|
| P-1 | Press Space → playback toggles → UI reflects new state | High |
| P-2 | Press N → next track → now-playing updates with new track info | High |
| P-3 | Tick fires every second → progress bar advances → clamps at duration | High |
| P-4 | Press +/- → volume changes → UI reflects new volume | Medium |
| P-5 | Press S → shuffle toggles → indicator updates | Medium |
| P-6 | Press R → repeat cycles (off → context → track → off) → indicator updates | Medium |
| P-7 | Press V → visualizer pattern cycles → preference persisted | Low |
| P-8 | Pause playback → polling interval widens (3s → 10s) → resume → interval narrows back | Medium |

### 5.3 Queue

| # | Workflow | Priority |
|---|---|---|
| Q-1 | Queue pane visible → tick fires → queue refreshes from API → table updates | High |
| Q-2 | Press F → filter activates → type query → table filters in real-time | Medium |
| Q-3 | Select track → press Enter → track plays | Medium |
| Q-4 | Play track from search → queue pane reflects new upcoming tracks on next refresh | High |

### 5.4 Playlists

| # | Workflow | Priority |
|---|---|---|
| PL-1 | Select playlist → press Enter → track list loads → sub-view shown | High |
| PL-2 | In track view → press Shift+Up/Down → track reorders (optimistic) → API confirms | High |
| PL-3 | In track view → Shift+Up → API fails → local state rolls back to original order | High |
| PL-4 | In track view → press X → track removed from playlist | Medium |
| PL-5 | Press Esc in track view → returns to playlist list | Medium |
| PL-6 | Press F → filter playlists by name → results narrow in real-time | Medium |
| PL-7 | Rapid reorders (Shift+Up 3 times quickly) → all applied in correct order | Medium |

### 5.5 Devices

| # | Workflow | Priority |
|---|---|---|
| D-1 | Press D → overlay opens → device list fetched → devices displayed | High |
| D-2 | Select device → press Enter → playback transfers → overlay closes → now-playing reflects new device | High |
| D-3 | Select already-active device → feedback shown ("Already playing here") | Medium |
| D-4 | Press Esc → overlay closes → focus returns to grid | Medium |
| D-5 | Open overlay → wait past TTL (5s) → close and reopen → fresh fetch fires | Low |

### 5.6 Library Panes (Liked Songs, Albums, Recently Played)

| # | Workflow | Priority |
|---|---|---|
| L-1 | Pane loads on startup → data fetched → table populated | High |
| L-2 | Press F → filter activates → type query → table filters → Esc closes filter | Medium |
| L-3 | Select track → press Enter → track plays | Medium |
| L-4 | Press I on track in Liked Songs → like/unlike toggles → API confirms | Medium |
| L-5 | Liked Songs pane visible → like track from another pane → Liked Songs reflects change on refresh | Medium |

### 5.7 Stats (Top Tracks, Top Artists)

| # | Workflow | Priority |
|---|---|---|
| ST-1 | Stats pane loads → short_term data fetched → table populated | High |
| ST-2 | Press T → time range cycles (short → medium → long) → data refreshes | Medium |
| ST-3 | Switch time range → data cached → switch back → cached data shown (no API call) | Medium |
| ST-4 | Top Tracks and Top Artists cycle time ranges independently | Low |

### 5.8 Overlay & Focus Routing

| # | Workflow | Priority |
|---|---|---|
| F-1 | Press / → search overlay opens → all keys route to search → Esc closes → keys route to grid | High |
| F-2 | Press D → device overlay opens → all keys route to devices → Esc closes → keys route to grid | High |
| F-3 | Press T → theme overlay opens → select theme → all panes re-render with new theme → overlay closes | Medium |
| F-4 | Overlay open → press Q → does NOT quit (overlay captures input) | High |
| F-5 | Filter active in pane → global shortcuts blocked → Esc closes filter → shortcuts work again | High |
| F-6 | Only one overlay open at a time (no stacking) | Medium |

### 5.9 Layout & Navigation

| # | Workflow | Priority |
|---|---|---|
| N-1 | Press Tab → focus rotates through visible panes only | High |
| N-2 | Press 1-8 → toggle pane visibility → Tab skips hidden panes | Medium |
| N-3 | Press P → cycle preset → panes rearrange → focus on first visible | Medium |
| N-4 | Press 0 → toggle Page A/B → different pane set shown | Medium |
| N-5 | Resize terminal → all panes receive new dimensions → layout adjusts | Medium |

### 5.10 Error Recovery & Resilience

| # | Workflow | Priority |
|---|---|---|
| E-1 | API returns 429 → backoff activates → toast shown → after backoff expires → polling resumes | High |
| E-2 | API returns 401 → token refreshes → original request retries silently | High |
| E-3 | API returns 403 → "Spotify Premium required" toast shown | Medium |
| E-4 | Multiple concurrent 401s → token refreshes only once | High |
| E-5 | Network error on any fetch → error toast shown → user retries manually → succeeds | Medium |
| E-6 | User action during backoff → request queued or rejected with feedback | Medium |
| E-7 | 5+ consecutive playback fetch failures → connection warning toast (shown once) | Low |

### 5.11 Preference Persistence

| # | Workflow | Priority |
|---|---|---|
| PR-1 | Change theme → preference flushed to disk after 500ms debounce | Medium |
| PR-2 | Change theme then preset rapidly → single flush writes both | Medium |
| PR-3 | App restart → theme, preset, visualizer pattern restored from config | Medium |

### 5.12 Startup & Auth

| # | Workflow | Priority |
|---|---|---|
| A-1 | Fresh launch → splash screen → auth screen → complete auth → grid loads → initial data fetches fire | High |
| A-2 | Launch with cached token → skip auth → grid loads directly | High |
| A-3 | Launch with expired token → auto-refresh → grid loads | Medium |
| A-4 | Terminal too small → "too small" message → resize → grid renders | Low |

### 5.13 Cross-Feature Interactions

| # | Workflow | Priority |
|---|---|---|
| X-1 | Search → play track → now-playing updates → queue updates on next tick | High |
| X-2 | Search → add to queue → queue pane reflects new track on refresh | High |
| X-3 | Device transfer → playback state re-fetched → now-playing shows new device | High |
| X-4 | Like track in Liked Songs → search results reflect updated like state | Low |
| X-5 | Play from any pane (library, search, queue, playlist) → now-playing always updates | Medium |

### 5.14 Golden File / Visual Regression

| # | Workflow | Priority |
|---|---|---|
| G-1 | Grid with default theme + default preset → snapshot matches golden file | High |
| G-2 | Each of the 11 themes → snapshot matches golden file | Medium |
| G-3 | Search overlay rendered over dimmed grid → snapshot matches | Medium |
| G-4 | Device overlay rendered over dimmed grid → snapshot matches | Medium |
| G-5 | Each layout preset → snapshot matches | Low |
| G-6 | Terminal resize to various sizes → snapshots match | Low |

---

## 6. Story Sequencing

Implementation should follow this order:

| Story | Scope | Depends On |
|---|---|---|
| **S1: Infrastructure & helpers** | Test utilities, mock API factory, golden file conventions, teatest integration | — |
| **S2: Root model — cross-pane workflows** | Workflows from 5.8, 5.10, 5.13 (high priority items) | S1 |
| **S3: Root model — overlay & focus routing** | Workflows from 5.1 (S-1 through S-5, S-10), 5.5, 5.8 | S1 |
| **S4: Root model — error recovery** | Workflows from 5.10 (E-1 through E-4) | S1 |
| **S5: Pane-level — search** | Workflows from 5.1 (S-6 through S-9), debounce timing, prefetch chain | S1 |
| **S6: Pane-level — playlists** | Workflows from 5.4 | S1 |
| **S7: Pane-level — library, queue, stats** | Workflows from 5.2, 5.3, 5.6, 5.7 | S1 |
| **S8: Full app — golden files** | Workflows from 5.14 | S1, S2 |
| **S9: Full app — startup, polling, prefs** | Workflows from 5.11, 5.12, 5.9 | S1, S2 |

---

## 7. Dependency

Single new dependency:

```
github.com/charmbracelet/x/exp/teatest
```

- Experimental package, no backwards compatibility guarantee
- Used in test code only — zero impact on production binary
- If API changes, migration is confined to test helpers

---

## 8. CI Integration

- Integration tests tagged with `//go:build integration` (consistent with existing pattern)
- `make test-integration` runs integration suite separately
- `make ci` runs both unit and integration tests
- Golden files committed to `testdata/golden/` and updated via `-update` flag
- No additional CI infrastructure needed (no FFmpeg, no browser, no external services)
