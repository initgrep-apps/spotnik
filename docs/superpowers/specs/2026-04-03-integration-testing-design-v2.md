# Integration & Component Testing Design for Spotnik

**Date:** 2026-04-03
**Status:** Approved
**Supersedes:** `2026-04-02-integration-testing-design.md` (removed)
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

## 2. Testing Architecture

### 2.1 Three Test Layers

```
        /  Teatest  \          few — full workflows through real event loop
       /--------------\
      /   Catwalk       \      many — every pane's keybindings, rendering, states
     /--------------------\
    / Unit Tests (existing) \  foundation — keep as-is (1,074+ tests)
   /--------------------------\
```

[x/vt](https://github.com/charmbracelet/x/tree/main/vt) sits alongside as a utility used by both [catwalk](https://github.com/knz/catwalk) and [teatest](https://github.com/charmbracelet/x/tree/main/exp/teatest) for cleaner assertions and golden files.

| Layer | Tool | What It Proves | Speed | Volume |
|---|---|---|---|---|
| **Unit** (existing) | [`testing`](https://pkg.go.dev/testing) + [`testify`](https://github.com/stretchr/testify) + [`httptest`](https://pkg.go.dev/net/http/httptest) | Functions return correct values. API parsing works. State getters/setters are correct. | Fast (ms) | ~1,074 tests (keep) |
| **Component** (new) | [`catwalk`](https://github.com/knz/catwalk) + [`x/vt`](https://github.com/charmbracelet/x/tree/main/vt) | Each pane's keybindings work. Filter activates. Content renders. Scroll works. A single pane does what it claims. | Fast (ms) | Many — one script per pane per feature |
| **Integration** (new) | [`teatest`](https://github.com/charmbracelet/x/tree/main/exp/teatest) + [`x/vt`](https://github.com/charmbracelet/x/tree/main/vt) | The full app wires together. Overlay routing works. Cross-pane effects happen. Command chains fire through the real event loop. | Medium (100ms-3s) | Few — one per critical workflow |

### 2.2 Layer Relationships

**Rule:** If component tests fail, don't run integration tests. Fix the pane first.

**Analogy:**
- **Unit tests** = test the steering wheel mechanism on a bench
- **Component tests (catwalk)** = test the steering wheel alone — turn left, does it signal left?
- **Integration tests (teatest)** = drive the full car — turn the wheel, does the car actually turn?
- **x/vt** = the dashboard camera — tells you what the driver actually sees

### 2.3 Execution Order (Always)

```
1. make test            ← unit tests pass first
2. make test-component  ← pane behavior verified
3. make test-integration ← wiring verified
4. make lint            ← code quality
```

---

## 3. Tool Decisions

### 3.1 New Test-Only Dependencies

| Package | Import Path | Purpose | Production Impact |
|---|---|---|---|
| **teatest** | `github.com/charmbracelet/x/exp/teatest` | Headless Bubble Tea program runner | Zero — test files only |
| **catwalk** | `github.com/knz/catwalk` | Data-driven component test runner | Zero — test files only |
| **x/vt** | `github.com/charmbracelet/x/vt` | Virtual terminal emulator for screen parsing | Zero — test files only |

### 3.2 What Each Tool Does

**Catwalk** — You write a plain text file that says "send these inputs, expect this output." The test runner feeds inputs to your model's `Update()` and compares `View()` output against expected text. Run with `-rewrite` to regenerate expectations.

**Teatest** — Wraps your `tea.Model` in a real `tea.Program` running headlessly. You send keys and messages. Commands fire. Ticks tick. You wait for conditions on the output. Tests the real event loop.

**x/vt** — Not a testing tool itself. It's a screen parser. Both teatest and catwalk produce ANSI output. x/vt takes that raw output and gives you "here's what the user actually sees on screen" — plain text, cell positions, colors. Enables structured assertions instead of raw ANSI string matching.

### 3.3 Build Tag Separation

| Tag | What Runs | Command |
|---|---|---|
| (none) | Unit tests only (existing behavior, unchanged) | `go test ./...` |
| `component` | Catwalk component tests | `go test -tags=component ./...` |
| `integration` | Teatest integration tests | `go test -tags=integration ./...` |
| `component,integration` | Both new layers | `go test -tags=component,integration ./...` |

### 3.4 Makefile Additions

```makefile
test-component:    go test -tags=component ./... -race -count=1
test-integration:  go test -tags=integration ./... -race -count=1
test-all:          go test -tags=component,integration ./... -race -count=1
ci:                fmt-check tidy-check lint test-coverage test-component test-integration build
```

### 3.5 File Naming Conventions

| Layer | File Pattern | Build Tag |
|---|---|---|
| Unit | `*_test.go` | none |
| Component | `*_component_test.go` | `//go:build component` |
| Integration | `*_integration_test.go` | `//go:build integration` |
| Component data | `testdata/components/<pane>/*.txt` | N/A (data files) |
| Golden files | `testdata/golden/*.golden` | N/A (data files) |

---

## 4. Shared Test Utilities Package

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

## 5. Component Testing with Catwalk

### 5.1 Directory Structure

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

### 5.2 Script Format Example

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

### 5.3 Go Driver File (Minimal Boilerplate per Pane)

```go
//go:build component

func TestTopTracksComponent(t *testing.T) {
    s := state.New()
    s.SetTopTracks("short_term", testTopTracks())
    pane := NewTopTracksPane(s, theme.Load("black"), true)
    catwalk.RunAllTests(t, pane, "testdata/components/toptracks")
}
```

### 5.4 What Component Tests Catch

- Keybinding `f` actually activates the filter (not just shown in help bar)
- Typing in filter actually narrows results
- Escape actually closes the filter
- Time range `t` cycles through all values and wraps
- Enter plays the selected item
- Rendered output matches what users should see

### 5.5 What Component Tests Do NOT Catch

- Whether pressing `f` in the queue pane reaches the queue pane when the search overlay is open (that's routing — integration layer)
- Whether filtered results come from the real API (store is pre-populated)
- Cross-pane side effects

---

## 6. Integration Testing with Teatest

### 6.1 File Organization

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

### 6.2 Example: Search → Play → Now-Playing Updates

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

### 6.3 Example: Overlay Captures Input

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

### 6.4 Example: Golden File with x/vt

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

### 6.5 What Integration Tests Catch That Component Tests Miss

| Scenario | Why Catwalk Can't Catch It |
|---|---|
| Search overlay captures all keys | Requires root model routing |
| Play from search updates now-playing + queue | Cross-pane effect through message chain |
| 401 triggers token refresh then retries | Command chain through API gateway |
| Tab skips hidden panes | Layout manager + focus rotation interaction |
| Debounce actually fires after 300ms | Needs real event loop timing |
| Theme switch re-renders all panes | Cross-cutting effect through root model |

---

## 7. Decision Rules — When to Use What

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

### 7.1 For Story Implementation

Every story in `docs/spec/features/` has a `## Tests` section. When implementing a story:

1. **Write unit tests** for any new functions in `api/`, `state/`, `config/`
2. **Write catwalk scripts** for any pane keybinding or rendering behavior the story adds
3. **Write integration tests** only if the story involves cross-pane effects, overlay routing, or command chains
4. **Update golden files** if the story changes visible layout (`go test -tags=integration -update`)

### 7.2 What Does NOT Need New Tests

- Refactors that don't change behavior (existing tests cover it)
- Documentation changes
- Config key additions (unit test the config loader, not the pane)

### 7.3 Retroactive Coverage

All 17 completed features currently have unit tests only. When the integration/component testing infrastructure is set up (Story S1), subsequent stories will systematically add catwalk and teatest tests to every existing feature. Each feature gets its own story. This is not optional cleanup — it is the primary deliverable of this testing initiative. Existing unit tests stay untouched. The new layers are additive.

---

## 8. Skills for Test Writing

### 8.1 Skill: `component-test`

**Triggers when:** Agent is implementing a pane feature, keybinding, filter, or rendering change.

**Provides:**
- Catwalk script format and conventions
- Target directory: `testdata/components/<pane>/`
- Boilerplate Go driver file pattern
- Examples of common script patterns (filter toggle, key action, scroll, enter-plays)
- Reminder to pre-populate store with test data in the driver
- Command: `make test-component` and `-rewrite` for intentional output changes

### 8.2 Skill: `integration-test`

**Triggers when:** Agent is implementing overlay routing, cross-pane workflows, command chains, error recovery flows, or golden file updates.

**Provides:**
- Teatest pattern: `NewTestModel`, `Send`, `WaitFor`, `FinalModel`
- File placement: `internal/app/integration_*_test.go`
- `testutil.NewMockSpotifyServer` and `testutil.NewTestApp` usage
- Examples of common patterns (overlay open/close, cross-pane play, error recovery)
- x/vt golden file pattern for visual verification
- Build tag reminder: `//go:build integration`
- Command: `make test-integration`

### 8.3 Feature-Implementer Workflow (Updated)

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

## 9. Existing Test Inventory

### 9.1 Current State (Unchanged by This Design)

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

### 9.2 Test Data

- 16 JSON fixture files in `testdata/fixtures/`
- No golden files yet (will be added)
- No component test scripts yet (will be added)

### 9.3 Existing Patterns to Preserve

- Table-driven tests with `t.Run()` subtests
- `TestMain` for ANSI color profile setup (`lipgloss.SetColorProfile(termenv.TrueColor)`)
- `httptest.NewServer` for all API mocking
- Interface compliance checks (`var _ layout.Pane = &AlbumsPane{}`)
- `testify/assert` for non-fatal, `testify/require` for fatal assertions

---

## 10. Testable Workflows Catalog

### 10.1 Search

| # | Workflow | Layer | Priority |
|---|---|---|---|
| S-1 | Open search → type query → debounce fires → results appear | Integration | High |
| S-2 | Type query → results load → switch tab → results refresh for new type | Integration | High |
| S-3 | Type query → scroll down past 60% → next page prefetches (5-page batch) | Integration | High |
| S-4 | Switch tab mid-prefetch → stale pages discarded, fresh search fires | Integration | High |
| S-5 | Select track → Enter → track plays → overlay closes | Integration | High |
| S-6 | Select track → Ctrl+A → added to queue → confirmation toast | Integration | Medium |
| S-7 | Ctrl+U → input clears → results clear | Component | Medium |
| S-8 | Rapid typing (50ms between keys) → one search per 300ms debounce | Integration | Medium |
| S-9 | Fewer results than page size → no further prefetch | Component | Medium |
| S-10 | Esc → overlay closes → focus returns to previous pane | Integration | High |
| S-11 | Type `:songs ` → prefix locks → search fires with track type only | Component | Medium |
| S-12 | Type `:so` → prefix hints shown → Tab completes to `:songs ` | Component | Medium |
| S-13 | Query changed while fetching → stale SearchPageLoadedMsg discarded | Integration | High |
| S-14 | Offset reaches 1000 → prefetch stops, no error | Component | Low |
| S-15 | Play album/playlist result → plays as context (not track URI) | Integration | Medium |
| S-16 | Category badges render correctly (♫ track, ● artist, ◆ album, ☰ playlist) | Component | Low |
| S-17 | 80% overlay sizing → resize terminal → overlay adjusts | Integration | Low |

### 10.2 Playback (Now Playing)

| # | Workflow | Layer | Priority |
|---|---|---|---|
| P-1 | Space → playback toggles → UI reflects (optimistic update) | Component | High |
| P-2 | N or → → next track → now-playing updates | Integration | High |
| P-3 | Tick fires → progress bar advances via interpolation → clamps at duration | Component | High |
| P-4 | +/- → volume changes → UI reflects → clamped 0-100% | Component | Medium |
| P-5 | S → shuffle toggles → indicator updates (⇄ dim/bright) | Component | Medium |
| P-6 | R → repeat cycles (off → context → track → off) → indicator updates (↻/↻1) | Component | Medium |
| P-7 | V → visualizer pattern cycles (7 patterns) → preference persisted | Integration | Low |
| P-8 | Pause → polling widens (3s→10s) → resume → polling narrows (10s→3s) | Integration | Medium |
| P-9 | L → like/unlike current track → optimistic toggle | Component | Medium |
| P-10 | Nothing playing (204 response) → centered empty state message | Component | High |
| P-11 | Volume 403 VOLUME_CONTROL_DISALLOW → friendly toast message | Integration | Medium |
| P-12 | 5 consecutive playback poll errors → connection warning toast (once) | Integration | Low |
| P-13 | Height < 8 → compact mode (single-line strip, no visualizer) | Component | Medium |
| P-14 | Gradient seek bar renders with color interpolation (Gradient1→2→3) | Component | Low |
| P-15 | Transport controls render as Unicode (▷ ⏸ ⇄ ↻) not emoji | Component | Low |
| P-16 | ← → previous track → now-playing updates | Integration | Medium |

### 10.3 Queue

| # | Workflow | Layer | Priority |
|---|---|---|---|
| Q-1 | Tick fires → queue refreshes from API → table updates | Integration | High |
| Q-2 | F → filter activates → type → table filters → Esc closes | Component | Medium |
| Q-3 | Enter → track plays | Component | Medium |
| Q-4 | Play from search → queue reflects new tracks on refresh | Integration | High |
| Q-5 | Empty queue → centered "Queue is empty" message | Component | Medium |
| Q-6 | Queue with 20+ tracks → scroll indicators shown, j/k scrolls | Component | Medium |
| Q-7 | Currently playing track shows ▶ indicator in index column | Component | Low |

### 10.4 Playlists

| # | Workflow | Layer | Priority |
|---|---|---|---|
| PL-1 | Enter on playlist → track list loads → sub-view shown with title | Component | High |
| PL-2 | Shift+Up/Down → track reorders (optimistic) → API confirms | Integration | High |
| PL-3 | Shift+Up → API fails → rollback to original order | Integration | High |
| PL-4 | X → track removed from playlist (optimistic) | Component | Medium |
| PL-5 | Esc in track view → returns to playlist list | Component | Medium |
| PL-6 | F → filter playlists → results narrow | Component | Medium |
| PL-7 | Rapid reorders (Shift+Up 3x) → all applied in correct order | Integration | Medium |
| PL-8 | N → create new playlist → inline input → Enter confirms → API creates | Integration | Medium |
| PL-9 | R → rename selected playlist → inline input → Enter saves | Integration | Medium |
| PL-10 | X on track → API fails → track restored to original position | Integration | Medium |
| PL-11 | Empty playlist → track pane shows "No tracks in this playlist" | Component | Low |
| PL-12 | Reorder at boundary (first track up, last track down) → no-op | Component | Low |

### 10.5 Devices

| # | Workflow | Layer | Priority |
|---|---|---|---|
| D-1 | D → overlay opens → devices fetched → displayed | Integration | High |
| D-2 | Enter → playback transfers → overlay closes → now-playing reflects new device | Integration | High |
| D-3 | Select active device → "Already playing" feedback | Component | Medium |
| D-4 | Esc → overlay closes → focus returns to grid | Integration | Medium |
| D-5 | Open past TTL (5s) → close → reopen → fresh fetch fires | Integration | Low |
| D-6 | No devices connected → "No devices found" message | Component | Medium |
| D-7 | Active device shown in header (right-aligned, max 25 chars + truncation) | Component | Low |
| D-8 | Device with long name → truncated with ellipsis | Component | Low |

### 10.6 Library (Albums, Liked Songs)

| # | Workflow | Layer | Priority |
|---|---|---|---|
| L-1 | Pane loads → data fetched → table populated | Integration | High |
| L-2 | F → filter → type → Esc closes filter | Component | Medium |
| L-3 | Enter on track → track plays | Component | Medium |
| L-4 | I on Liked Songs → like/unlike toggles (optimistic) | Component | Medium |
| L-5 | Like from one pane → Liked Songs reflects on refresh | Integration | Medium |
| L-6 | Enter on album → plays album context (not single track) | Component | Medium |
| L-7 | A → add selected track to queue → confirmation toast | Integration | Medium |
| L-8 | Scroll near bottom → next page loads automatically (pagination) | Integration | Medium |
| L-9 | Empty library section → appropriate empty state message | Component | Low |
| L-10 | Filter with no matches → "No results" message | Component | Low |

### 10.7 Stats (Top Tracks, Top Artists, Recently Played)

| # | Workflow | Layer | Priority |
|---|---|---|---|
| ST-1 | Stats pane loads → short_term data fetched → table populated | Integration | High |
| ST-2 | T → time range cycles (4wk → 6mo → all → 4wk) → data refreshes | Component | Medium |
| ST-3 | Switch range → cached → switch back → cached data shown (no API call) | Integration | Medium |
| ST-4 | Top Tracks and Top Artists cycle time ranges independently | Integration | Low |
| ST-5 | Recently Played shows relative time (just now / Nm ago / Nh ago / Nd ago / date) | Component | Medium |
| ST-6 | A on Recently Played → add track to queue | Integration | Medium |
| ST-7 | Empty stats → "No listening data" message | Component | Low |
| ST-8 | Artist with no genres → shows `--` in genre column | Component | Low |

### 10.8 Overlay & Focus Routing

| # | Workflow | Layer | Priority |
|---|---|---|---|
| F-1 | / → search opens → keys route to search → Esc → keys route to grid | Integration | High |
| F-2 | D → devices open → keys route to devices → Esc → keys route to grid | Integration | High |
| F-3 | T → theme overlay → select → all panes re-render → overlay closes | Integration | Medium |
| F-4 | Overlay open → Q → does NOT quit (overlay captures input) | Integration | High |
| F-5 | Filter active → globals blocked → Esc → globals restored | Integration | High |
| F-6 | Only one overlay at a time (T while search open → blocked) | Integration | Medium |
| F-7 | Overlay close restores a.focus to a.prevFocus exactly | Integration | Medium |

### 10.9 Layout & Navigation

| # | Workflow | Layer | Priority |
|---|---|---|---|
| N-1 | Tab → focus rotates visible panes only (wraps at end) | Integration | High |
| N-2 | 1-8 → toggle pane → space redistributed → Tab skips hidden | Integration | Medium |
| N-3 | P → cycle preset (0→1→2→3→0) → panes rearrange → preset persisted | Integration | Medium |
| N-4 | 0 → toggle Page A/B → different pane set shown | Integration | Medium |
| N-5 | Terminal resize → all panes adjust → no gaps, no overlap | Integration | Medium |
| N-6 | Terminal below 120x30 → "too small" guard message | Integration | Low |
| N-7 | Mouse wheel scroll within pane → scrolls without changing focus | Integration | Low |

### 10.10 Error Recovery

| # | Workflow | Layer | Priority |
|---|---|---|---|
| E-1 | 429 → backoff → ratelimit toast → resume after Retry-After seconds | Integration | High |
| E-2 | 401 → token refresh → retry original request silently | Integration | High |
| E-3 | 403 playback → "Playback control not available" toast | Integration | Medium |
| E-4 | 403 non-playback → "Spotify Premium required" toast | Integration | Medium |
| E-5 | Multiple concurrent 401s → token refreshes only once (dedup) | Integration | High |
| E-6 | Network error → error toast → next poll succeeds → error auto-clears | Integration | Medium |
| E-7 | Action during backoff → rejected with feedback | Integration | Medium |
| E-8 | 5+ consecutive playback errors → connection warning toast (once, not repeated) | Integration | Low |
| E-9 | Proactive token refresh 5 min before expiry → seamless | Integration | Medium |
| E-10 | Token refresh fails → "Session expired. Run: spotnik auth" error toast | Integration | Medium |

### 10.11 Preferences

| # | Workflow | Layer | Priority |
|---|---|---|---|
| PR-1 | Change theme → flushed after 500ms debounce | Integration | Medium |
| PR-2 | Change theme + preset + visualizer rapidly → single flush writes all 3 | Integration | Medium |
| PR-3 | Restart → theme, preset, visualizer pattern restored from config | Integration | Medium |
| PR-4 | V → visualizer pattern changes → persisted via PreferenceStore | Integration | Medium |
| PR-5 | Flush failure → changes re-queued for next attempt | Integration | Low |

### 10.12 Startup & Auth

| # | Workflow | Layer | Priority |
|---|---|---|---|
| A-1 | Fresh launch → splash → auth → grid → data loads | Integration | High |
| A-2 | Cached token → skip auth → grid loads directly | Integration | High |
| A-3 | Expired token → auto-refresh → grid loads | Integration | Medium |
| A-4 | Terminal too small → "too small" message → resize → grid renders | Integration | Low |
| A-5 | Missing client_id (no embedded, no config) → error with setup instructions → exit 1 | Integration | Medium |
| A-6 | `spotnik auth logout` → clears keychain → next launch requires fresh auth | Integration | Medium |
| A-7 | First launch with no config → Bootstrap creates config file automatically | Integration | Medium |

### 10.13 Cross-Feature

| # | Workflow | Layer | Priority |
|---|---|---|---|
| X-1 | Search → play track → now-playing updates → queue updates on tick | Integration | High |
| X-2 | Search → add to queue → queue reflects on refresh | Integration | High |
| X-3 | Device transfer → playback re-fetched → now-playing shows new device | Integration | High |
| X-4 | Like track → search reflects updated state | Integration | Low |
| X-5 | Play from any pane (library, search, queue, playlist) → now-playing always updates | Integration | Medium |
| X-6 | Theme switch → all panes re-render with new colors including borders | Integration | Medium |
| X-7 | Playback keys (Space, n, +/-, s, r) route to NowPlaying regardless of pane focus | Integration | High |

### 10.14 Theme System

| # | Workflow | Layer | Priority |
|---|---|---|---|
| TH-1 | Load known theme ID → correct theme returned | Component | High |
| TH-2 | Load unknown theme ID → fallback to "black" without panic | Component | High |
| TH-3 | All 11 themes implement all 50 color tokens (non-empty) | Component | High |
| TH-4 | Theme switcher overlay → navigate → Enter applies → toast shown | Integration | Medium |
| TH-5 | Theme overlay → current theme marked with ◉, others with ○ | Component | Medium |
| TH-6 | Theme overlay shows 5 color swatches per theme | Component | Low |
| TH-7 | User TOML theme overrides built-in by ID | Component | Low |
| TH-8 | Malformed user theme TOML → skipped without panic | Component | Medium |
| TH-9 | Focused pane border uses bright accent color, unfocused uses dimmed | Component | Medium |

### 10.15 Golden Files / Visual Regression

| # | Workflow | Layer | Priority |
|---|---|---|---|
| G-1 | Default grid + default theme → matches golden | Integration | High |
| G-2 | Each of 11 themes → matches golden | Integration | Medium |
| G-3 | Search overlay over grid → matches golden | Integration | Medium |
| G-4 | Device overlay over grid → matches golden | Integration | Medium |
| G-5 | Theme overlay over grid → matches golden | Integration | Medium |
| G-6 | Each layout preset → matches golden | Integration | Low |
| G-7 | Various terminal sizes → matches golden | Integration | Low |

### 10.16 Nerd Status (Page B)

| # | Workflow | Layer | Priority |
|---|---|---|---|
| NB-1 | 0 → switch to Page B → Request Flow + Network Log shown | Integration | Medium |
| NB-2 | Network Log filter → F → type → narrows entries | Component | Medium |
| NB-3 | Request Flow event replay at 200ms tick → phases animate (entered → gateway → inflight → completed → done) | Component | Medium |
| NB-4 | 0 → back to Page A → original pane focus restored | Integration | Medium |
| NB-5 | Blocked request (429) shows in APP box but skips SPOTIFY box | Component | Medium |
| NB-6 | Decisions age out after 3s, completed requests after 5s | Component | Low |
| NB-7 | Network Log color-codes by status (2xx green, 429 yellow, 5xx red) | Component | Low |
| NB-8 | Staleness display shows stale domains with elapsed time | Component | Low |
| NB-9 | Request Flow width < 60 → flat fallback rendering | Component | Low |
| NB-10 | Multiple concurrent requests animate at staggered phases | Component | Low |

### 10.17 API Gateway Internals

| # | Workflow | Layer | Priority |
|---|---|---|---|
| GW-1 | Token bucket: burst up to 10, blocks when empty, refills at 10/sec | Integration | High |
| GW-2 | Concurrency cap: max 5 in-flight requests, 6th waits | Integration | High |
| GW-3 | Request dedup: 2 concurrent same-key requests → 1 HTTP call, both get result | Integration | High |
| GW-4 | Interactive priority requests bypass token bucket wait | Integration | Medium |
| GW-5 | Background requests rejected during 429 backoff (return RateLimitError) | Integration | Medium |
| GW-6 | Staleness: re-fetch after TTL (playlists 5m, devices 5s, stats 10m) | Integration | Medium |
| GW-7 | Adaptive idle polling: active+playing 3s/9s → idle+paused 30s/60s | Integration | Medium |
| GW-8 | KeyMsg resets tickCount → immediate fetch when returning from idle | Integration | Low |

---

## 11. Feature Test Roadmap

### 11.1 Story Sequencing

| Story | Scope | Depends On |
|---|---|---|
| **S1: Infrastructure** | Add teatest, catwalk, x/vt deps. Create `internal/testutil/`. Set up build tags, Makefile targets, `testdata/components/` and `testdata/golden/` dirs. Create both skills. | — |
| **S2: Now Playing** | Catwalk: P-1, P-3, P-4, P-5, P-6, P-9, P-10, P-13, P-14, P-15. Integration: P-2, P-7, P-8, P-11, P-12, P-16. | S1 |
| **S3: Queue** | Catwalk: Q-2, Q-3, Q-5, Q-6, Q-7. Integration: Q-1, Q-4. | S1 |
| **S4: Search** | Catwalk: S-7, S-9, S-11, S-12, S-14, S-16. Integration: S-1 through S-6, S-8, S-10, S-13, S-15, S-17. | S1 |
| **S5: Playlists** | Catwalk: PL-1, PL-4, PL-5, PL-6, PL-11, PL-12. Integration: PL-2, PL-3, PL-7, PL-8, PL-9, PL-10. | S1 |
| **S6: Library** | Catwalk: L-2, L-3, L-4, L-6, L-9, L-10. Integration: L-1, L-5, L-7, L-8. | S1 |
| **S7: Stats** | Catwalk: ST-2, ST-5, ST-7, ST-8. Integration: ST-1, ST-3, ST-4, ST-6. | S1 |
| **S8: Devices** | Catwalk: D-3, D-6, D-7, D-8. Integration: D-1, D-2, D-4, D-5. | S1 |
| **S9: Themes** | Catwalk: TH-1 through TH-9. Integration: TH-4, F-3. Golden: G-2. | S1 |
| **S10: Overlay & Focus** | Integration: F-1, F-2, F-4, F-5, F-6, F-7. | S1, S4, S8, S9 |
| **S11: Navigation & Layout** | Integration: N-1 through N-7. | S1 |
| **S12: Error Recovery** | Integration: E-1 through E-10. | S1 |
| **S13: Golden Files** | Golden: G-1, G-3 through G-7. | S1, S2-S9 |
| **S14: Nerd Status** | Catwalk: NB-2, NB-3, NB-5, NB-6, NB-7, NB-8, NB-9, NB-10. Integration: NB-1, NB-4. | S1 |
| **S15: Gateway** | Integration: GW-1 through GW-8. | S1 |
| **S16: Cross-Feature & Prefs** | Integration: X-1 through X-7, PR-1 through PR-5, A-1 through A-7. | S1, S2-S9 |

### 11.2 Priority Order

**S1** (infrastructure) → **S4** (search, most complex) → **S10** (overlay routing, biggest bug source) → **S2** (playback, most used) → **S12** (error recovery) → then S3-S16 in any order.

### 11.3 Each Story Produces

- Catwalk scripts in `testdata/components/`
- Integration tests in `internal/app/integration_*_test.go`
- Golden files in `testdata/golden/` (where applicable)
- Updated golden files via `-update` flag if layout changed

---

## 12. Pane & Keybinding Reference

### 12.1 Page A — Music Panes

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

### 12.2 Page B — Nerd Status

| Pane | Key Features | Filter | Scroll |
|---|---|---|---|
| Request Flow | Read-only visualization, 200ms animation tick | No | No |
| Network Log | f=filter, j/k=scroll, 200-entry ring buffer | Yes | Yes |

### 12.3 Overlays

| Overlay | Open | Close | Navigation | Action |
|---|---|---|---|---|
| Search | `/` | Esc | Tab/Shift+Tab=tabs, arrows=list, Ctrl+A=queue | Enter=play |
| Devices | `d` | Esc | j/k=list | Enter=transfer |
| Themes | `t` | Esc | j/k=list | Enter=apply |

### 12.4 Global Keys

| Key | Action | Scope |
|---|---|---|
| `q` | Quit | Always (blocked by overlays/filters) |
| `Tab`/`Shift+Tab` | Focus rotation | Visible panes |
| `0` | Toggle Page A/B | Always |
| `p` | Cycle preset | Current page |
| `1`-`8` | Toggle pane | Page A only |
| Space, n, +/-, s, r, v, arrows | Playback | Always route to Now Playing |

---

## 13. CI Integration

- `make ci` updated to: `fmt-check tidy-check lint test-coverage test-component test-integration build`
- Integration tests tagged with `//go:build integration`
- Component tests tagged with `//go:build component`
- Golden files committed to `testdata/golden/` and updated via `-update` flag
- No additional CI infrastructure needed (no FFmpeg, no browser, no external services)
- Unit test 80% coverage threshold unchanged
