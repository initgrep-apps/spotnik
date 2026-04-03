# Integration & Component Testing Design for Spotnik

**Date:** 2026-04-03
**Status:** Approved
**Supersedes:** `2026-04-02-integration-testing-design.md` (research document, retained as reference)
**Scope:** Testing architecture, tool decisions, agent skills, and retroactive test roadmap

---

## 1. Problem Statement

Spotnik has 77 test files with 1,074+ test functions and 80%+ coverage. All tests use synchronous unit testing: direct `Update(msg)` calls, `View()` output inspection, and `httptest` mock servers.

**What unit tests cannot catch:**

- **Controls that don't work** — a keybinding shown in the help bar but not wired in `Update()`
- **Overlay focus routing** — keys leaking through overlays to the wrong pane
- **Cross-pane effects** — play from search not updating now-playing or queue
- **Command chains** — debounce → API call → response → state update → re-render
- **Visual regression** — layout changes after refactoring that go unnoticed
- **Regression after new features** — existing workflows breaking silently

**The cost:** After every feature implementation, every existing scenario must be manually re-tested. This is unsustainable with 17 completed features and growing.

---

## 2. Research Summary

Extensive research was conducted across the Go ecosystem, Bubble Tea ecosystem, and cross-framework TUI testing patterns (Textual/Python, Ratatui/Rust, Ink/Node.js). Key findings:

### What exists in Go

| Tool | Type | Maturity | Relevance |
|---|---|---|---|
| `testing` + `testify` + `httptest` | Unit testing | Production | Already in use (77 files) |
| `charmbracelet/x/exp/teatest` | Bubble Tea integration testing | Experimental | Primary integration tool |
| `knz/catwalk` | Data-driven Bubble Tea component testing | Stable (Apache-2.0) | Primary component tool |
| `charmbracelet/x/vt` | Virtual terminal emulator | Experimental | Screen parsing for assertions |
| `charmbracelet/x/exp/golden` | Golden file comparison | Experimental | Snapshot assertions |
| `charmbracelet/vhs` | Terminal recording | Production | Demos only, not assertions |
| Ginkgo/GoConvey | BDD frameworks | Production | Too heavy, not idiomatic |
| `creack/pty` | PTY automation | Production | Too low-level |

### Cross-ecosystem insights

- **Textual (Python)** — gold standard: headless execution + Pilot API + CSS-selector targeting + SVG snapshots
- **Ratatui (Rust)** — TestBackend + insta snapshots for visual regression
- **Lazygit** — custom integration framework: setup state → replay keystrokes → assert outcomes
- **Charm team's vision** — `teatest` + `x/vt` + `x/xpty` combined, building blocks exist but no integrated framework yet

### What the previous spec got right

- Teatest is the right integration tool
- VHS is for demos, not testing
- Direct `Update()`/`View()` can't catch wiring bugs
- The 60+ workflow catalog is comprehensive

### What the previous spec missed

- **`charmbracelet/x/vt`** — full virtual terminal emulator for structured screen assertions (not just raw ANSI string matching)
- **`knz/catwalk`** — data-driven component testing via plain text scripts (not considered at all)
- **Teatest has very low adoption** (~6 importers) — even Charm's own apps don't use it
- **No custom harness needed** — catwalk + teatest + x/vt used directly, no maintenance burden

---

## 3. Testing Architecture

### 3.1 Three Test Layers

```
        /  Teatest  \          few — full workflows through real event loop
       /--------------\
      /   Catwalk       \      many — every pane's keybindings, rendering, states
     /--------------------\
    / Unit Tests (existing) \  foundation — keep as-is (1,074+ tests)
   /--------------------------\
```

x/vt sits alongside as a utility used by both catwalk and teatest for cleaner assertions and golden files.

| Layer | Tool | What It Proves | Speed | Volume |
|---|---|---|---|---|
| **Unit** (existing) | `testing` + `testify` + `httptest` | Functions return correct values. API parsing works. State getters/setters are correct. | Fast (ms) | ~1,074 tests (keep) |
| **Component** (new) | `catwalk` + `x/vt` | Each pane's keybindings work. Filter activates. Content renders. Scroll works. A single pane does what it claims. | Fast (ms) | Many — one script per pane per feature |
| **Integration** (new) | `teatest` + `x/vt` | The full app wires together. Overlay routing works. Cross-pane effects happen. Command chains fire through the real event loop. | Medium (100ms-3s) | Few — one per critical workflow |

### 3.2 Layer Relationships

**Rule:** If component tests fail, don't run integration tests. Fix the pane first.

**Analogy:**
- **Unit tests** = test the steering wheel mechanism on a bench
- **Component tests (catwalk)** = test the steering wheel alone — turn left, does it signal left?
- **Integration tests (teatest)** = drive the full car — turn the wheel, does the car actually turn?
- **x/vt** = the dashboard camera — tells you what the driver actually sees

### 3.3 Execution Order (Always)

```
1. make test            ← unit tests pass first
2. make test-component  ← pane behavior verified
3. make test-integration ← wiring verified
4. make lint            ← code quality
```

---

## 4. Tool Decisions

### 4.1 New Test-Only Dependencies

| Package | Import Path | Purpose | Production Impact |
|---|---|---|---|
| **teatest** | `github.com/charmbracelet/x/exp/teatest` | Headless Bubble Tea program runner | Zero — test files only |
| **catwalk** | `github.com/knz/catwalk` | Data-driven component test runner | Zero — test files only |
| **x/vt** | `github.com/charmbracelet/x/vt` | Virtual terminal emulator for screen parsing | Zero — test files only |

### 4.2 What Each Tool Does

**Catwalk** — You write a plain text file that says "send these inputs, expect this output." The test runner feeds inputs to your model's `Update()` and compares `View()` output against expected text. Run with `-rewrite` to regenerate expectations.

**Teatest** — Wraps your `tea.Model` in a real `tea.Program` running headlessly. You send keys and messages. Commands fire. Ticks tick. You wait for conditions on the output. Tests the real event loop.

**x/vt** — Not a testing tool itself. It's a screen parser. Both teatest and catwalk produce ANSI output. x/vt takes that raw output and gives you "here's what the user actually sees on screen" — plain text, cell positions, colors. Enables structured assertions instead of raw ANSI string matching.

### 4.3 Build Tag Separation

| Tag | What Runs | Command |
|---|---|---|
| (none) | Unit tests only (existing behavior, unchanged) | `go test ./...` |
| `component` | Catwalk component tests | `go test -tags=component ./...` |
| `integration` | Teatest integration tests | `go test -tags=integration ./...` |
| `component,integration` | Both new layers | `go test -tags=component,integration ./...` |

### 4.4 Makefile Additions

```makefile
test-component:    go test -tags=component ./... -race -count=1
test-integration:  go test -tags=integration ./... -race -count=1
test-all:          go test -tags=component,integration ./... -race -count=1
ci:                fmt-check tidy-check lint test-coverage test-component test-integration build
```

### 4.5 File Naming Conventions

| Layer | File Pattern | Build Tag |
|---|---|---|
| Unit | `*_test.go` | none |
| Component | `*_component_test.go` | `//go:build component` |
| Integration | `*_integration_test.go` | `//go:build integration` |
| Component data | `testdata/components/<pane>/*.txt` | N/A (data files) |
| Golden files | `testdata/golden/*.golden` | N/A (data files) |

---

## 5. Shared Test Utilities Package

**Problem:** 77 test files with duplicated helpers — `errorServer()`, `successServer()`, `testTheme()`, `sendKey()` scattered everywhere.

**Solution:** New package `internal/testutil/`:

```
internal/testutil/
├── mockserver.go    — reusable httptest factories (consolidate duplicated helpers)
├── fixtures.go      — fixture loading helpers
├── screen.go        — x/vt screen assertion helpers (AssertScreenContains, AssertCellColor)
└── keys.go          — key event construction helpers (consolidate sendKey patterns)
```

This replaces per-file helper duplication. Existing unit tests can optionally adopt these helpers but are not required to change.

---

## 6. Component Testing with Catwalk

### 6.1 Directory Structure

```
testdata/components/
├── queue/
│   ├── filter_toggle.txt
│   ├── filter_narrows_results.txt
│   ├── enter_plays_track.txt
│   └── scroll_navigation.txt
├── search/
│   ├── type_query.txt
│   ├── tab_switch.txt
│   ├── prefix_autocomplete.txt
│   ├── ctrl_u_clears.txt
│   └── enter_plays_result.txt
├── playlists/
│   ├── enter_opens_tracks.txt
│   ├── shift_up_reorders.txt
│   ├── x_removes_track.txt
│   ├── esc_returns_to_list.txt
│   └── filter_playlists.txt
├── likedsongs/
│   ├── filter_toggle.txt
│   ├── i_toggles_like.txt
│   └── enter_plays.txt
├── albums/
│   ├── filter_toggle.txt
│   └── enter_plays_album.txt
├── recentlyplayed/
│   ├── filter_toggle.txt
│   └── enter_plays.txt
├── toptracks/
│   ├── time_range_cycle.txt
│   ├── filter_toggle.txt
│   └── enter_plays.txt
├── topartists/
│   ├── time_range_cycle.txt
│   └── filter_toggle.txt
├── devices/
│   ├── navigate_list.txt
│   ├── enter_selects.txt
│   └── esc_closes.txt
├── themes/
│   ├── navigate_list.txt
│   ├── enter_applies.txt
│   └── esc_closes.txt
├── nowplaying/
│   ├── space_toggles.txt
│   ├── volume_controls.txt
│   └── shuffle_repeat.txt
└── networklog/
    ├── filter_toggle.txt
    └── scroll.txt
```

### 6.2 Script Format Example

File: `testdata/components/toptracks/time_range_cycle.txt`
```
# Setup: TopTracks pane with short_term data loaded
# Verify t key cycles through time ranges

run
----
-- view:
 # │ Track          │ Artist        │ Pop
 1 │ Blinding Lights│ The Weeknd    │ 95
                                    4wk

run
key t
----
-- view:
 # │ Track          │ Artist        │ Pop
 1 │ Blinding Lights│ The Weeknd    │ 95
                                    6mo

run
key t
----
-- view:
 # │ Track          │ Artist        │ Pop
 1 │ Blinding Lights│ The Weeknd    │ 95
                                    all

run
key t
----
-- view:
 # │ Track          │ Artist        │ Pop
 1 │ Blinding Lights│ The Weeknd    │ 95
                                    4wk
```

### 6.3 Go Driver File (Minimal Boilerplate per Pane)

```go
//go:build component

func TestTopTracksComponent(t *testing.T) {
    s := state.New()
    s.SetTopTracks("short_term", testTopTracks())
    pane := NewTopTracksPane(s, theme.Load("black"), true)
    catwalk.RunAllTests(t, pane, "testdata/components/toptracks")
}
```

### 6.4 What Component Tests Catch

- Keybinding `f` actually activates the filter (not just shown in help bar)
- Typing in filter actually narrows results
- Escape actually closes the filter
- Time range `t` cycles through all values and wraps
- Enter plays the selected item
- Rendered output matches what users should see

### 6.5 What Component Tests Do NOT Catch

- Whether pressing `f` in the queue pane reaches the queue pane when the search overlay is open (that's routing — integration layer)
- Whether filtered results come from the real API (store is pre-populated)
- Cross-pane side effects

---

## 7. Integration Testing with Teatest

### 7.1 File Organization

```
internal/app/
├── integration_search_test.go       — search workflows
├── integration_overlay_test.go      — overlay focus routing
├── integration_playback_test.go     — playback through event loop
├── integration_crossfeature_test.go — cross-pane effects
├── integration_error_test.go        — 401/429/403 recovery
├── integration_navigation_test.go   — Tab, pane toggle, page switch
├── integration_playlists_test.go    — reorder, remove, sub-view
└── integration_golden_test.go       — visual snapshots via x/vt
```

### 7.2 Example: Search → Play → Now-Playing Updates

```go
//go:build integration

func TestSearchPlayUpdatesNowPlaying(t *testing.T) {
    srv := testutil.NewMockSpotifyServer(t)
    app := testutil.NewTestApp(t, srv.URL)
    tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

    // Open search
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("Search"))
    }, teatest.WithDuration(2*time.Second))

    // Type query — debounce fires through real event loop
    tm.Type("arctic monkeys")
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("Do I Wanna Know"))
    }, teatest.WithDuration(3*time.Second))

    // Play the track
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

    // Overlay closes, now-playing updates — cross-pane effect
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("Do I Wanna Know")) &&
            !bytes.Contains(bts, []byte("Search"))
    }, teatest.WithDuration(3*time.Second))

    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
    tm.WaitFinished(t)
}
```

### 7.3 Example: Overlay Captures Input

```go
//go:build integration

func TestOverlayBlocksQuit(t *testing.T) {
    srv := testutil.NewMockSpotifyServer(t)
    app := testutil.NewTestApp(t, srv.URL)
    tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

    // Open search overlay
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("Search"))
    })

    // Press q — overlay should capture, app stays alive
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

    // Verify still running — close overlay, then quit for real
    tm.Send(tea.KeyMsg{Type: tea.KeyEscape})
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
    tm.WaitFinished(t)
}
```

### 7.4 Example: Golden File with x/vt

```go
//go:build integration

func TestGridDefaultLayoutGolden(t *testing.T) {
    srv := testutil.NewMockSpotifyServer(t)
    app := testutil.NewTestApp(t, srv.URL)
    tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("Now Playing"))
    }, teatest.WithDuration(5*time.Second))

    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
    raw := tm.FinalOutput(t)
    buf := new(bytes.Buffer)
    buf.ReadFrom(raw)

    em := vt.NewEmulator(120, 40)
    em.WriteString(buf.String())

    // Golden file compares parsed screen — immune to ANSI sequence changes
    golden.RequireEqual(t, []byte(em.String()))
}
```

### 7.5 What Integration Tests Catch That Component Tests Miss

| Scenario | Why Catwalk Can't Catch It |
|---|---|
| Search overlay captures all keys | Requires root model routing |
| Play from search updates now-playing + queue | Cross-pane effect through message chain |
| 401 triggers token refresh then retries | Command chain through API gateway |
| Tab skips hidden panes | Layout manager + focus rotation interaction |
| Debounce actually fires after 300ms | Needs real event loop timing |
| Theme switch re-renders all panes | Cross-cutting effect through root model |

---

## 8. Decision Rules — When to Use What

```
Is it a new function in api/, state/, config/, domain/?
  → Unit test (existing pattern, testify + httptest)

Is it a pane keybinding, filter, rendering, or UI behavior?
  → Component test (catwalk script in testdata/components/<pane>/)

Does it involve two or more panes talking to each other?
  → Integration test (teatest)

Does it involve overlay open/close/focus routing?
  → Integration test (teatest)

Does it involve command chains (key → API call → response → state update → re-render)?
  → Integration test (teatest)

Does it involve visual correctness (layout, colors, golden file)?
  → Golden file test (x/vt + golden, can be either layer)
```

### 8.1 For Story Implementation

Every story in `docs/spec/features/` has a `## Tests` section. When implementing a story:

1. **Write unit tests** for any new functions in `api/`, `state/`, `config/`
2. **Write catwalk scripts** for any pane keybinding or rendering behavior the story adds
3. **Write integration tests** only if the story involves cross-pane effects, overlay routing, or command chains
4. **Update golden files** if the story changes visible layout (`go test -tags=integration -update`)

### 8.2 What Does NOT Need New Tests

- Refactors that don't change behavior (existing tests cover it)
- Documentation changes
- Config key additions (unit test the config loader, not the pane)

### 8.3 Retroactive Coverage

All 17 completed features currently have unit tests only. When the integration/component testing infrastructure is set up (Story S1), subsequent stories will systematically add catwalk and teatest tests to every existing feature. Each feature gets its own story. This is not optional cleanup — it is the primary deliverable of this testing initiative. Existing unit tests stay untouched. The new layers are additive.

---

## 9. Skills & Agents for Test Writing

### 9.1 Why Skills, Not Agents

The feature-implementer agent already has the full feature context (story spec, implementation code, pane behavior). A skill teaches it the testing patterns. A separate agent would duplicate context and produce disconnected tests.

### 9.2 Skill: `component-test`

**Triggers when:** Agent is implementing a pane feature, keybinding, filter, or rendering change.

**Provides:**
- Catwalk script format and conventions
- Target directory: `testdata/components/<pane>/`
- Boilerplate Go driver file pattern
- Examples of common script patterns (filter toggle, key action, scroll, enter-plays)
- Reminder to pre-populate store with test data in the driver
- Command: `make test-component` and `-rewrite` for intentional output changes

### 9.3 Skill: `integration-test`

**Triggers when:** Agent is implementing overlay routing, cross-pane workflows, command chains, error recovery flows, or golden file updates.

**Provides:**
- Teatest pattern: `NewTestModel`, `Send`, `WaitFor`, `FinalModel`
- File placement: `internal/app/integration_*_test.go`
- `testutil.NewMockSpotifyServer` and `testutil.NewTestApp` usage
- Examples of common patterns (overlay open/close, cross-pane play, error recovery)
- x/vt golden file pattern for visual verification
- Build tag reminder: `//go:build integration`
- Command: `make test-integration`

### 9.4 Feature-Implementer Workflow (Updated)

```
1. Read story spec → understand tasks and tests needed
2. Write implementation code
3. Write unit tests (existing pattern — no skill needed)
4. Invoke component-test skill → write catwalk scripts for pane behavior
5. Invoke integration-test skill → write teatest tests if story has cross-pane effects
6. Run make test-all → verify everything passes
7. Commit
```

---

## 10. Existing Test Inventory

### 10.1 Current State (Unchanged by This Design)

| Package | Test Files | Test Functions | Approach |
|---|---|---|---|
| `internal/api/` | 14 | 126 | `httptest` mock servers, table-driven, JSON fixtures |
| `internal/app/` | 18 | 379 | Direct `Update()`/`View()`, Elm purity checks, mock servers |
| `internal/state/` | 2 | 87 | Direct getter/setter assertions |
| `internal/config/` | 1 | 20 | `t.TempDir()`, TOML fixtures, permission checks |
| `internal/prefs/` | 1 | 12 | In-memory + disk flush, TOML round-trip |
| `internal/keychain/` | 2 | 20 | In-memory store + macOS keychain integration |
| `internal/domain/` | 2 | 11 | JSON marshal/unmarshal, type assertions |
| `internal/ui/components/` | 11 | 95 | Direct component construction, frame comparison |
| `internal/ui/layout/` | 5 | 88 | Rect computation, border drawing, ANSI extraction |
| `internal/ui/panes/` | 16+ | 540 | Direct pane `Update()`/`View()`, store manipulation |
| `internal/ui/theme/` | 1 | 46 | Theme loading, color method coverage |
| `cmd/` | 1 | 24 | CLI initialization, flag parsing |
| **Total** | **77** | **1,074+** | |

### 10.2 Test Data

- 16 JSON fixture files in `testdata/fixtures/`
- No golden files yet (will be added)
- No component test scripts yet (will be added)

### 10.3 Existing Patterns to Preserve

- Table-driven tests with `t.Run()` subtests
- `TestMain` for ANSI color profile setup (`lipgloss.SetColorProfile(termenv.TrueColor)`)
- `httptest.NewServer` for all API mocking
- Interface compliance checks (`var _ layout.Pane = &AlbumsPane{}`)
- `testify/assert` for non-fatal, `testify/require` for fatal assertions

---

## 11. Testable Workflows Catalog

### 11.1 Search

| # | Workflow | Layer | Priority |
|---|---|---|---|
| S-1 | Open search → type query → debounce fires → results appear | Integration | High |
| S-2 | Type query → results load → switch tab → results refresh for new type | Integration | High |
| S-3 | Type query → scroll down past 60% → next page prefetches | Integration | High |
| S-4 | Switch tab mid-prefetch → stale pages discarded | Integration | High |
| S-5 | Select track → Enter → track plays → overlay closes | Integration | High |
| S-6 | Select track → Ctrl+A → added to queue → confirmation shown | Integration | Medium |
| S-7 | Ctrl+U → input clears → results clear | Component | Medium |
| S-8 | Rapid typing (50ms between keys) → one search per 300ms debounce | Integration | Medium |
| S-9 | Fewer results than page size → no further prefetch | Component | Medium |
| S-10 | Esc → overlay closes → focus returns to previous pane | Integration | High |

### 11.2 Playback (Now Playing)

| # | Workflow | Layer | Priority |
|---|---|---|---|
| P-1 | Space → playback toggles → UI reflects | Component | High |
| P-2 | N → next track → now-playing updates | Integration | High |
| P-3 | Tick fires → progress bar advances → clamps at duration | Component | High |
| P-4 | +/- → volume changes → UI reflects | Component | Medium |
| P-5 | S → shuffle toggles → indicator updates | Component | Medium |
| P-6 | R → repeat cycles (off → context → track → off) | Component | Medium |
| P-7 | V → visualizer pattern cycles → preference persisted | Integration | Low |
| P-8 | Pause → polling widens → resume → polling narrows | Integration | Medium |

### 11.3 Queue

| # | Workflow | Layer | Priority |
|---|---|---|---|
| Q-1 | Tick fires → queue refreshes from API → table updates | Integration | High |
| Q-2 | F → filter activates → type → table filters | Component | Medium |
| Q-3 | Enter → track plays | Component | Medium |
| Q-4 | Play from search → queue reflects new tracks on refresh | Integration | High |

### 11.4 Playlists

| # | Workflow | Layer | Priority |
|---|---|---|---|
| PL-1 | Enter on playlist → track list loads → sub-view shown | Component | High |
| PL-2 | Shift+Up/Down → track reorders (optimistic) → API confirms | Integration | High |
| PL-3 | Shift+Up → API fails → rollback to original order | Integration | High |
| PL-4 | X → track removed from playlist | Component | Medium |
| PL-5 | Esc in track view → returns to playlist list | Component | Medium |
| PL-6 | F → filter playlists → results narrow | Component | Medium |
| PL-7 | Rapid reorders (Shift+Up 3x) → all applied in correct order | Integration | Medium |

### 11.5 Devices

| # | Workflow | Layer | Priority |
|---|---|---|---|
| D-1 | D → overlay opens → devices fetched → displayed | Integration | High |
| D-2 | Enter → playback transfers → overlay closes → now-playing reflects | Integration | High |
| D-3 | Select active device → "Already playing" feedback | Component | Medium |
| D-4 | Esc → overlay closes → focus returns | Integration | Medium |
| D-5 | Open past TTL → close → reopen → fresh fetch | Integration | Low |

### 11.6 Library (Liked Songs, Albums, Recently Played)

| # | Workflow | Layer | Priority |
|---|---|---|---|
| L-1 | Pane loads → data fetched → table populated | Integration | High |
| L-2 | F → filter → type → Esc closes | Component | Medium |
| L-3 | Enter → track plays | Component | Medium |
| L-4 | I on Liked Songs → like/unlike toggles | Component | Medium |
| L-5 | Like from one pane → Liked Songs reflects on refresh | Integration | Medium |

### 11.7 Stats (Top Tracks, Top Artists)

| # | Workflow | Layer | Priority |
|---|---|---|---|
| ST-1 | Stats pane loads → short_term data fetched | Integration | High |
| ST-2 | T → time range cycles → data refreshes | Component | Medium |
| ST-3 | Switch range → cached → switch back → cached shown | Integration | Medium |
| ST-4 | Top Tracks and Top Artists cycle independently | Integration | Low |

### 11.8 Overlay & Focus Routing

| # | Workflow | Layer | Priority |
|---|---|---|---|
| F-1 | / → search opens → keys route to search → Esc → keys route to grid | Integration | High |
| F-2 | D → devices open → keys route to devices → Esc → keys route to grid | Integration | High |
| F-3 | T → theme overlay → select → all panes re-render → overlay closes | Integration | Medium |
| F-4 | Overlay open → Q → does NOT quit | Integration | High |
| F-5 | Filter active → globals blocked → Esc → globals restored | Integration | High |
| F-6 | Only one overlay at a time | Integration | Medium |

### 11.9 Layout & Navigation

| # | Workflow | Layer | Priority |
|---|---|---|---|
| N-1 | Tab → focus rotates visible panes only | Integration | High |
| N-2 | 1-8 → toggle pane → Tab skips hidden | Integration | Medium |
| N-3 | P → cycle preset → panes rearrange | Integration | Medium |
| N-4 | 0 → toggle Page A/B | Integration | Medium |
| N-5 | Terminal resize → all panes adjust | Integration | Medium |

### 11.10 Error Recovery

| # | Workflow | Layer | Priority |
|---|---|---|---|
| E-1 | 429 → backoff → toast → resume after backoff | Integration | High |
| E-2 | 401 → token refresh → retry silently | Integration | High |
| E-3 | 403 → "Premium required" toast | Integration | Medium |
| E-4 | Multiple concurrent 401s → single refresh | Integration | High |
| E-5 | Network error → toast → manual retry → success | Integration | Medium |
| E-6 | Action during backoff → queued or rejected with feedback | Integration | Medium |
| E-7 | 5+ consecutive failures → connection warning toast | Integration | Low |

### 11.11 Preferences

| # | Workflow | Layer | Priority |
|---|---|---|---|
| PR-1 | Change theme → flushed after 500ms debounce | Integration | Medium |
| PR-2 | Change theme + preset rapidly → single flush | Integration | Medium |
| PR-3 | Restart → theme, preset, visualizer restored | Integration | Medium |

### 11.12 Startup & Auth

| # | Workflow | Layer | Priority |
|---|---|---|---|
| A-1 | Fresh launch → splash → auth → grid → data loads | Integration | High |
| A-2 | Cached token → skip auth → grid loads | Integration | High |
| A-3 | Expired token → auto-refresh → grid loads | Integration | Medium |
| A-4 | Terminal too small → message → resize → grid renders | Integration | Low |

### 11.13 Cross-Feature

| # | Workflow | Layer | Priority |
|---|---|---|---|
| X-1 | Search → play → now-playing updates → queue updates on tick | Integration | High |
| X-2 | Search → add to queue → queue reflects on refresh | Integration | High |
| X-3 | Device transfer → playback re-fetched → now-playing shows new device | Integration | High |
| X-4 | Like track → search reflects updated state | Integration | Low |
| X-5 | Play from any pane → now-playing always updates | Integration | Medium |

### 11.14 Golden Files / Visual Regression

| # | Workflow | Layer | Priority |
|---|---|---|---|
| G-1 | Default grid + default theme → matches golden | Integration | High |
| G-2 | Each of 11 themes → matches golden | Integration | Medium |
| G-3 | Search overlay over grid → matches golden | Integration | Medium |
| G-4 | Device overlay over grid → matches golden | Integration | Medium |
| G-5 | Each layout preset → matches golden | Integration | Low |
| G-6 | Various terminal sizes → matches golden | Integration | Low |

### 11.15 Nerd Status (Page B)

| # | Workflow | Layer | Priority |
|---|---|---|---|
| NB-1 | 0 → switch to Page B → Request Flow + Network Log shown | Integration | Medium |
| NB-2 | Network Log filter → F → type → narrows | Component | Medium |
| NB-3 | Request Flow animation renders at 200ms tick | Component | Low |
| NB-4 | 0 → back to Page A → original pane focus restored | Integration | Medium |

---

## 12. Feature Test Roadmap

### 12.1 Story Sequencing

| Story | Scope | Depends On |
|---|---|---|
| **S1: Infrastructure** | Add teatest, catwalk, x/vt deps. Create `internal/testutil/`. Set up build tags, Makefile targets, `testdata/components/` and `testdata/golden/` dirs. Create both skills. | — |
| **S2: Now Playing** | Catwalk: P-1, P-3, P-4, P-5, P-6. Integration: P-2, P-7, P-8. | S1 |
| **S3: Queue** | Catwalk: Q-2, Q-3. Integration: Q-1, Q-4. | S1 |
| **S4: Search** | Catwalk: S-7, S-9. Integration: S-1 through S-6, S-8, S-10. | S1 |
| **S5: Playlists** | Catwalk: PL-1, PL-4, PL-5, PL-6. Integration: PL-2, PL-3, PL-7. | S1 |
| **S6: Library** | Catwalk: L-2, L-3, L-4. Integration: L-1, L-5. | S1 |
| **S7: Stats** | Catwalk: ST-2. Integration: ST-1, ST-3, ST-4. | S1 |
| **S8: Devices** | Catwalk: D-3. Integration: D-1, D-2, D-4, D-5. | S1 |
| **S9: Themes** | Catwalk: theme list nav, enter applies, esc closes. Integration: F-3. Golden: G-2. | S1 |
| **S10: Overlay & Focus** | Integration: F-1, F-2, F-4, F-5, F-6. | S1, S4, S8, S9 |
| **S11: Navigation & Layout** | Integration: N-1 through N-5. | S1 |
| **S12: Error Recovery** | Integration: E-1 through E-7. | S1 |
| **S13: Golden Files** | Golden: G-1, G-3 through G-6. | S1, S2-S9 |
| **S14: Nerd Status** | Catwalk: NB-2, NB-3. Integration: NB-1, NB-4. | S1 |

### 12.2 Priority Order

**S1** (infrastructure) → **S4** (search, most complex) → **S10** (overlay routing, biggest bug source) → **S2** (playback, most used) → then S3-S14 in any order.

### 12.3 Each Story Produces

- Catwalk scripts in `testdata/components/`
- Integration tests in `internal/app/integration_*_test.go`
- Golden files in `testdata/golden/` (where applicable)
- Updated golden files via `-update` flag if layout changed

---

## 13. Pane & Keybinding Reference

### 13.1 Page A — Music Panes

| # | Pane | Toggle | Key Features | Filter | Scroll |
|---|---|---|---|---|---|
| 1 | Now Playing | `1` | Space, n, +/-, s, r, v, arrows. Visualizer, gradient bars. | No | No |
| 2 | Queue | `2` | f=filter, Enter=play, j/k=scroll | Yes | Yes |
| 3 | Playlists | `3` | f=filter, Enter=open tracks, Shift+arrows=reorder, x=remove, Esc=back | Yes | Yes |
| 4 | Albums | `4` | f=filter, Enter=play album | Yes | Yes |
| 5 | Liked Songs | `5` | f=filter, i=toggle like, Enter=play | Yes | Yes |
| 6 | Recently Played | `6` | f=filter, Enter=play | Yes | Yes |
| 7 | Top Tracks | `7` | f=filter, t=time range, Enter=play | Yes | Yes |
| 8 | Top Artists | `8` | f=filter, t=time range, Enter=play artist | Yes | Yes |

### 13.2 Page B — Nerd Status

| Pane | Key Features | Filter | Scroll |
|---|---|---|---|
| Request Flow | Read-only visualization, 200ms animation tick | No | No |
| Network Log | f=filter, j/k=scroll, 200-entry ring buffer | Yes | Yes |

### 13.3 Overlays

| Overlay | Open | Close | Navigation | Action |
|---|---|---|---|---|
| Search | `/` | Esc | Tab/Shift+Tab=tabs, arrows=list, Ctrl+A=queue | Enter=play |
| Devices | `d` | Esc | j/k=list | Enter=transfer |
| Themes | `t` | Esc | j/k=list | Enter=apply |

### 13.4 Global Keys

| Key | Action | Scope |
|---|---|---|
| `q` | Quit | Always (blocked by overlays/filters) |
| `Tab`/`Shift+Tab` | Focus rotation | Visible panes |
| `0` | Toggle Page A/B | Always |
| `p` | Cycle preset | Current page |
| `1`-`8` | Toggle pane | Page A only |
| Space, n, +/-, s, r, v, arrows | Playback | Always route to Now Playing |

---

## 14. CI Integration

- `make ci` updated to: `fmt-check tidy-check lint test-coverage test-component test-integration build`
- Integration tests tagged with `//go:build integration`
- Component tests tagged with `//go:build component`
- Golden files committed to `testdata/golden/` and updated via `-update` flag
- No additional CI infrastructure needed (no FFmpeg, no browser, no external services)
- Unit test 80% coverage threshold unchanged
