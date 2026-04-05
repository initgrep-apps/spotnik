# Search Overlay: Comprehensive Code Review
**Date:** 2026-04-05
**Reviewer:** Claude Code (Sonnet 4.6)
**Scope:** Feature 19 — Search Redesign, stories 97–104

---

## PART 1 — Design Document Analysis

### Stated Intent

The overlay solves the problem of searching Spotify without leaving the terminal. The redesign (stories 97–104) moved from a batch-prefetch, Store-coupled, debounce-fragmented architecture to a single-call-per-intent, overlay-owned-state, dual-layer-debounced design. The core user problem being fixed was: rapid tab switches triggering multiple 429-prone API calls, in-flight requests continuing after Esc, stale results from batching.

### Component Boundaries

| Owner | Responsibility |
|---|---|
| `SearchOverlay` | Display state: `results`, `total`, `loadingFirstPage`, `loadingNextPage`, `intent{query,tab,page}` |
| `App` | Session state: `searchQuery`, `searchPage`, `searchLoading`, `searchCancel` — staleness keys and HTTP lifecycle only |
| `Gateway` | Transport backstop: 100ms path-keyed debounce for `Interactive` requests |
| `Store` | Zero search state |

### Implicit State Machine

```
            keypress / Tab / Ctrl+Right / Ctrl+Left
              ↓
Empty ──keypress──→ Typing
  ↑                    │ debounce(query="")
  │                    ↓ no-op
  │              debounce(query!="")
  │                    ↓
  │              LoadingFirst ──results arrive──→ Results
  │                                                  │ Ctrl+Right/Left
  │                                                  ↓
  │                                            LoadingNext ──results arrive──→ Results
  │
  ←── Ctrl+U ──── (any state)
  ←── Esc ──────── (any state) → Closed (searchCancel())
```

### Required Invariants

1. At most one HTTP call live at any time (enforced by `searchCancel` + Gateway 100ms debounce)
2. No search state in Store — overlay owns display state
3. Error always clears loading flags — spinner never gets stuck
4. Only Esc closes the overlay
5. Stale ticks are discarded via `m.intent != o.intent` comparison
6. `searchCancel` is never nil — initialized to `func(){}`
7. `Reset()` is called on every open — clean slate guaranteed

### Ambiguities / Underspecified Behaviours

1. **Ctrl+U display state.** The design edge case table says "Return to empty state" but does not specify whether `o.results` and `o.total` must be explicitly zeroed, or whether the empty-query guard in `handleDebounce` is sufficient. The implementation assumed the `SearchClearedMsg` roundtrip would zero them; that assumption was never validated and is the source of the critical bug below.

2. **"No results" vs "type to search" distinction.** The design loading state table lists `"Type to search"` as the no-query state but doesn't specify what text appears for a zero-results response. The implementation reuses the no-query hint, which is misleading.

3. **`isFirst` semantics on tab switch.** The design says `IsFirstPage = true` when "results are nil." But a tab-switch can arrive when results from a previous tab are still showing (`results != nil`, `isFirst = false`). The design doesn't clarify whether a tab-switch with existing results should display spinner-only (new query) or spinner-above-old-results (page change). Current code conflates these two cases.

4. **`InfiniteScrolling` on the list.** Not mentioned in the design doc. The `list.Model` default without this flag is clamping, which is consistent with explicit pagination. The implementation added it without design justification.

---

## PART 2 — Code Implementation Review

### `NewSearchOverlay` (`search.go:222`)

**Purpose:** Constructs the overlay, wires theme, initializes all sub-components.
**Go semantics:** Correct — returns `*SearchOverlay`, initializes all fields, no zero-value hazards.
**BubbleTea:** list is correctly disabled of all built-in chrome. `InfiniteScrolling = true` is set — see Issue M3.
**Issues:** None in construction itself. `list.InfiniteScrolling = true` is architecturally problematic — see M3.

---

### `Init` (`search.go:436`)

**Purpose:** Start blink, spinner tick, placeholder tick; emit `SearchClearedMsg` for historical clean-state guarantee.
**Go semantics:** Returns `tea.Batch(...)`, correct.
**BubbleTea:** `searchSpinnerTick()` instead of `o.spinner.Tick` is correct — prevents cross-component `spinner.TickMsg` interference (story 94 fix).
**Issues:** `clearCmd = func() tea.Msg { return SearchClearedMsg{} }` is **dead on arrival** — `SearchClearedMsg` from `Init()` reaches `app.go`'s handler which `return a, nil` early, so it is never forwarded to the overlay. `Reset()` in `openSearch()` makes this redundant. **MINOR**.

---

### `Reset` (`search.go:352`)

**Purpose:** Restore overlay to initial state on every `openSearch()` call.
**Go semantics:** Correct — sets all fields, calls `resultList.SetItems(nil)`.
**BubbleTea:** Correct — does NOT call `resizeList()` because terminal dimensions are unknown at reset time.
**Design alignment:** Correctly called in `openSearch()` before `Init()`. Satisfies story 96 invariant.
**Issues:** None.

---

### `handleKey` → `KeyCtrlU` (`search.go:603`) — CRITICAL BUG

**Purpose:** Clear the search input and return to the empty state.

`o.results` and `o.total` are NOT zeroed in this handler. The handler clears the input and intent but emits `SearchClearedMsg{}` — expecting the overlay's own handler (`search.go:465`) to zero `o.results` and `o.total`. **However, that handler is unreachable.** Routing analysis:

```go
// In app.go handleMsg():
case panes.SearchClearedMsg:
    return a, nil   // ← returns early from handleMsg()
```

The fallthrough `if a.searchOpen { a.searchPane.Update(msg) }` is after the switch statement — unreachable when a switch case returns. `SearchClearedMsg` from Ctrl+U is consumed by app.go and **never forwarded to the overlay**.

**Result:** After pressing Ctrl+U, the user sees an empty input field but `o.results` still contains the previous page's items, `o.total > 0`, `resultList.Items()` is non-empty. `renderResults()` renders old results. `renderPaginationBar()` renders a stale `page 1 of N`. The overlay does **not** show the `"Type to search..."` placeholder.

**This is the primary persistent bug.**

---

### `handleKey` → `KeyBackspace` (`search.go:625`) — MAJOR BUG

**Purpose:** Delete a character, re-parse prefix, or demote prefix tag if cursor at 0.

After `demoteFromPromptTag()` is called, `o.intent.tab` retains the previously-locked tab value (e.g., `TabSongs`). `o.prefixState` is reset to `PrefixNone` but `o.intent.tab` is NOT reset to `TabAll`.

If the user demotes and backspaces through the entire `:songs query` value — making input empty with `PrefixNone` — and then types a new normal query, `intent.tab` is still `TabSongs`. `handleDebounce` calls `searchTypesForTab(intent.tab)` → `["track"]`, sending a track-only search even though no prefix is visible and the tab bar shows "All".

**This is a silent wrong-API-type bug.**

---

### `handleAddToQueue` (`search.go:716`) — MAJOR BUG

**Purpose:** Add selected track to queue.

```go
return o, func() tea.Msg { return AddToQueueMsg{TrackURI: uri} }
```

`TrackName` field of `AddToQueueMsg` is left empty. The `SearchListItem` at this point has `si.Name` available. `buildAddToQueueCmd` threads `trackName` through to `AddToQueueResultMsg.TrackName` for the toast notification ("Added `<trackname>` to queue"). With an empty name, the toast reads "Added  to queue".

**Fix:** `return o, func() tea.Msg { return AddToQueueMsg{TrackURI: uri, TrackName: si.Name} }`

---

### `handleDebounce` (`search.go:529`)

**Purpose:** Fire `SearchRequestMsg` when intent snapshot matches current intent and query is valid.
**BubbleTea:** Correct — the stale-tick pattern via struct equality on `searchIntent` is clean and race-free.
**Issues:** `types := searchTypesForTab(o.intent.tab)` — affected by the stale `intent.tab` issue described above (KeyBackspace). The app.go handler has a fallback `if len(searchTypes) == 0 { searchTypes = all types }` — this is dead code since `TabToAPITypes` always returns ≥1 type. **MINOR dead code.**

---

### `cycleTabForward/Backward` (`search.go:736–758`)

**Purpose:** Advance/retreat active tab, reset page to 1, sync prefix tag, rebuild list, schedule debounce.

`rebuildListItems()` is called before the new search fires, re-rendering the old results under the new tab's label. With `loadingNextPage = true` (spinner above old results), the user sees incorrect results for the new tab while loading. Per the design this is intentional, but combined with the pagination bar showing the old total, it can be confusing.

Neither `cycleTabForward` nor `cycleTabBackward` updates `intent.query`. This is correct — `scheduleDebounce()` snapshots current intent, and `cleanQuery()` in `handleDebounce` reads the current clean input value. No divergence.

---

### `scheduleDebounce` / `handleDebounce` — global

**Correctness:** Excellent. The struct-snapshot stale-tick pattern correctly handles all four triggers with zero coordination. The 300ms timer is correct. The nil return from `buildSearchPageCmd` on cancelled contexts is correct.

---

### `renderTabBar` (`search.go:931`) — MINOR stale comment

```go
// When store.SearchLoading() is true, a spinner frame is appended to the right side so
// re-searches are visible even when existing results remain on screen.
```

`store.SearchLoading()` no longer exists (story 97 deleted all store search state). The spinner was moved to `renderResultsPanel` as a `loadingNextPage` spinner line. The NOTE comment below correctly describes current behaviour. The first comment is stale and contradicts it.

---

### `SearchClearedMsg` handler in `Update()` (`search.go:465`)

Dead code. Never reached because:
- From `Ctrl+U`: state should be cleared inline (not via roundtrip); the message goes to app.go (no-op), never forwarded to overlay.
- From `Init()`: same — app.go no-op, not forwarded.

Should either be made reachable (fix app.go routing) or the clearing inlined into `handleKey(KeyCtrlU)` and this case removed.

---

### `buildSearchPageCmd` (`commands.go:273`)

**Purpose:** Single-page HTTP fetch, nil return on cancel.
**Go semantics:** Correct. Pre-call and post-call `ctx.Err()` checks are correct. The Spotify 1000-offset guard is correct.
**BubbleTea:** `nil` return drops silently — correct.
**Issues:** None.

---

### `app.go` — `SearchRequestMsg` handler

**Purpose:** Cancel in-flight call, set staleness keys, emit `SearchLoadingMsg`, dispatch fetch.
**Go semantics:** `a.searchCancel()` called first — correct. `searchCancel` is never nil — correct.

```go
isFirst := len(a.searchPane.Results()) == 0
```

After Ctrl+U (with the clearing bug active), `Results()` returns the stale non-nil slice → `isFirst = false` → `SearchLoadingMsg{IsFirstPage: false}` → `loadingNextPage = true`. A brand-new query after Ctrl+U shows spinner-above-old-results instead of full-panel spinner. Secondary symptom of C1.

---

## PART 3 — End-to-End Flow Simulation

### Scenario 1 — Happy path: `/` → type → results → select → overlay stays open ✓

1. User presses `/` → `openSearch()` → `Reset()` + `Init()` → `searchOpen = true`
2. User types "beatles" → `handleKey(default)` → `intent.query = "beatles"` → `scheduleDebounce`
3. 300ms idle → `searchDebounceMsg` → snapshot matches → `SearchRequestMsg{Query:"beatles", Types:all, Page:1}`
4. App: cancel prior (no-op), `searchQuery="beatles"`, new ctx, `isFirst=true`, `SearchLoadingMsg{IsFirstPage:true}` + fetchCmd dispatched
5. Overlay: `loadingFirstPage = true` → full-panel spinner
6. HTTP response → `SearchPageLoadedMsg{Query:"beatles", Page:1, Results:[...], Total:247}`
7. App staleness check: passes → `searchLoading = false` → forwarded to overlay
8. Overlay: clears loading flags, `results = m.Results`, `total = 247`, `rebuildListItems()`, `resizeList()` → list + "page 1 of 25"
9. Enter → `handleEnter()` → `PlayTrackMsg` → app plays track, overlay stays open

**Matches design doc. ✓**

---

### Scenario 2 — Ctrl+U clears input ✗ MISMATCH

1. Results showing from "beatles" search. `o.results` = 10 items, `o.total = 247`.
2. User presses Ctrl+U → input cleared, `intent.query=""`, `intent.page=1`
3. `SearchClearedMsg{}` emitted as Cmd
4. `SearchClearedMsg` arrives at `app.go` → `case panes.SearchClearedMsg: return a, nil` — early return, NOT forwarded to overlay
5. Overlay's `SearchClearedMsg` handler NEVER called
6. `o.results` still has 10 items. `o.total = 247`. `resultList` still has 10 items.
7. View: input is empty, but `renderResults()` → `resultList.View()` → old results still displayed. Pagination bar still shows "page 1 of 25"

**Should happen:** Empty state — "Type to search..." hint.
**Actually happens:** Old results remain visible after Ctrl+U. **BUG. ✗**

---

### Scenario 3 — Esc → overlay closes → underlying view unchanged ✓

1. User presses Esc → `SearchClosedMsg{}`
2. App: `closeSearch()` → `searchCancel()`, all session fields cleared, `searchOpen = false`
3. Underlying grid renders normally

**Matches design doc. ✓**

---

### Scenario 4 — Press `/` while overlay is already open ✓

`handleKeyMsg` routes ALL keys to the overlay when `searchOpen`. `/` is forwarded to `handleKey(default)` → types a literal `/` into the input. Overlay does not re-open. Sensible. ✓

---

### Scenario 5 — Fast typing / debounce race ✓

Each keystroke updates `intent.query` and schedules a new debounce. Prior debounce ticks fire with stale snapshots → discarded by `m.intent != o.intent`. Only the last debounce (after 300ms idle) proceeds. One HTTP call. **Correct. ✓**

---

### Scenario 6 — Zero results ⚠

`SearchPageLoadedMsg{Results:[], Total:0}` arrives:
- `o.results = []` (empty slice, not nil)
- `resultList.SetItems([]list.Item{})` — empty list
- View: `len(resultList.Items()) == 0` → "Type to search tracks, artists, albums..."

The "no results found" case shows the same hint as "no query typed yet" — confusing. No crash. **MINOR UX issue. ⚠**

---

### Scenario 7 — Search returns an error ✓

`SearchPageLoadedMsg{Err: someErr}`:
- Staleness check passes → `searchLoading = false` → forwarded to overlay
- Overlay: both loading flags cleared, `resizeList()`, existing results preserved
- Toast via `a.alerts.NewAlertCmd`

**Matches design doc. ✓**

---

### Scenario 8 — Arrow keys past first/last item ⚠

`list.InfiniteScrolling = true` → cursor wraps from last item to first and vice versa within the same page. This contradicts the explicit-pagination model where there's no implicit scroll to the next page. **Design doc does not specify. MINOR gap. ⚠**

---

### Scenario 9 — Window resize while overlay is open ✓

`tea.WindowSizeMsg` → `a.propagateSizes()` → `searchPane.SetSize()` → `resizeList()`. Overlay resizes correctly. **✓**

---

### Scenario 10 — Stale message arrives after overlay closes ✓

After `closeSearch()`, `a.searchQuery = ""`. Staleness check on `SearchPageLoadedMsg`: `m.Query != ""` → discard. Overlay never touched. **✓**

---

### Scenario 11 — Multiple rapid `/` keypresses ✓

First `/` opens overlay. All subsequent route to overlay → type literal slashes. State stays consistent. **✓**

---

### Scenario 12 — Key routing when overlay is open ✓

`handleKeyMsg` short-circuits at `if a.searchOpen` — ALL keys go to overlay. No parent-level keys fire. Tab goes to `cycleTabForward` OR textinput suggestion acceptance (PrefixTyping). No routing conflicts by construction. **✓**

---

### Scenario 13 — Esc during in-flight search ✓

1. `loadingFirstPage = true`, HTTP in-flight
2. Esc → `SearchClosedMsg` → `closeSearch()` → `searchCancel()` → HTTP context cancelled
3. `buildSearchPageCmd`: `ctx.Err() != nil` → returns `nil`
4. BubbleTea drops nil silently — no stale `SearchPageLoadedMsg`

**Matches design doc. ✓**

---

## PART 4 — Architectural Assessment

**Component model.** `SearchOverlay` is a self-contained Elm model with `Init/Update/View`. It communicates with the parent via typed messages: emits `SearchRequestMsg`, `SearchClosedMsg`, `PlayTrackMsg`, `PlayContextMsg`, `AddToQueueMsg`; receives `SearchLoadingMsg`, `SearchPageLoadedMsg`, `SearchClearedMsg`. This is correct Bubble Tea overlay architecture.

**State ownership.** State ownership is clean with one exception: the `SearchClearedMsg` routing gap means the overlay's clearing logic depends on an unreachable handler. The pattern is sound by design; the implementation has a broken wire.

**Lifecycle modelling.** `loadingFirstPage` / `loadingNextPage` being independent booleans means an illegal state (`both true`) is representable. The `SearchLoadingMsg` handler correctly enforces mutual exclusivity. A `loadingState` enum (`loadingNone / loadingFirst / loadingNext`) would make illegal states unrepresentable — worthwhile but not urgent.

**Trigger/dismiss pattern.** Standard and correct: parent opens overlay, overlay emits close message to parent. The routing guard `if a.searchOpen { ... return }` in `handleKeyMsg` correctly prevents parent-level key processing while the overlay is active.

**Coupling.** `app.go` reads `a.searchPane.Results()` to determine `isFirst` — tight coupling for one field. Acceptable and contained.

---

## PART 5 — Prioritised Findings and Recommendations

---

### CRITICAL ISSUES

---

**[C1] Ctrl+U does not clear results — `SearchClearedMsg` routing is broken**

- **Location:** `search.go:603` (`handleKey(KeyCtrlU)`) + `app.go` (`case panes.SearchClearedMsg:`)
- **Description:** `handleKey(KeyCtrlU)` clears the input and intent but does NOT zero `o.results`, `o.total`, or `resultList`. It emits `SearchClearedMsg{}` expecting the overlay's own handler (`search.go:465`) to do the clearing. But `app.go`'s handler returns early before the overlay-forwarding fallthrough is reached. The overlay's `SearchClearedMsg` handler is dead code.
- **Evidence:**
  - `app.go`: `case panes.SearchClearedMsg: return a, nil` — early return
  - `routing.go`: `if a.searchOpen { a.searchPane.Update(msg) }` is after the switch — unreachable when a case returns
  - `search.go:465`: `SearchClearedMsg` handler that zeroes `o.results` — never executed
- **Fix:** Zero display state directly in `handleKey(KeyCtrlU)` — do not rely on the `SearchClearedMsg` roundtrip:

```go
case tea.KeyCtrlU:
    o.input.Prompt = "> "
    o.input.SetValue("")
    o.lockedPrefix = ""
    o.prefixState = PrefixNone
    o.intent.page = 1
    o.intent.query = ""
    // Clear results immediately — do not rely on SearchClearedMsg roundtrip
    o.results = nil
    o.total = 0
    o.loadingFirstPage = false
    o.loadingNextPage = false
    o.resultList.SetItems(nil)
    o.resizeList()
    return o, tea.Batch(
        func() tea.Msg { return SearchClearedMsg{} },
        searchPlaceholderTick(),
    )
```

Then remove the dead `case SearchClearedMsg:` block from overlay's `Update()`.

---

### MAJOR ISSUES

---

**[M1] `handleAddToQueue` omits `TrackName` — queue toast shows empty name**

- **Location:** `search.go:727`
- **Description:** `AddToQueueMsg` is emitted with `TrackURI` only. `TrackName` is empty. `buildAddToQueueCmd` threads the name through to `AddToQueueResultMsg.TrackName` for the status-bar toast. Toast renders as "Added  to queue".
- **Evidence:** `return o, func() tea.Msg { return AddToQueueMsg{TrackURI: uri} }` — `si.Name` is available and unused.
- **Fix:**

```go
name := si.Name
return o, func() tea.Msg { return AddToQueueMsg{TrackURI: uri, TrackName: name} }
```

---

**[M2] `intent.tab` not reset after `demoteFromPromptTag()` — stale type filter on re-typed queries**

- **Location:** `search.go:266` (`demoteFromPromptTag`)
- **Description:** When the user presses Backspace at cursor 0 with a locked prefix, `demoteFromPromptTag()` resets `prefixState = PrefixNone` and `lockedPrefix = ""` but leaves `intent.tab = TabSongs` (or whichever was active). If the user then backspaces through the entire `:prefix query` value and types a new normal query, `handleDebounce` calls `searchTypesForTab(intent.tab)` → e.g. `["track"]` only — silently restricting what appears to be an all-types search.
- **Evidence:** `demoteFromPromptTag()` sets `o.prefixState = PrefixNone` and `o.lockedPrefix = ""` but no line resets `o.intent.tab`.
- **Fix:** Add `o.intent.tab = TabAll` to `demoteFromPromptTag()`.

---

**[M3] `list.InfiniteScrolling = true` violates explicit pagination model**

- **Location:** `search.go:262`
- **Description:** Infinite scrolling within a page allows `↑` at item 0 to jump to item 9 — the last item on the same page. This contradicts the design's explicit Ctrl+Right/Left pagination contract and creates a false impression that all results are available locally. Not in the design doc.
- **Fix:** `rl.InfiniteScrolling = false` so the cursor clamps at page boundaries, consistent with explicit pagination.

---

### MINOR ISSUES

---

**[m1] Dead `SearchClearedMsg` handler in overlay (`search.go:465`)**

After applying C1's fix (inline clearing in Ctrl+U), this case is permanently unreachable. Remove it.

---

**[m2] Stale comment in `renderTabBar` (`search.go:929`)**

The first comment block references `store.SearchLoading()` (deleted in story 97) and describes spinner-in-tab-bar behaviour that was moved to `renderResultsPanel`. Delete the first comment; keep only the NOTE.

---

**[m3] Stale comment on `SearchClearedMsg` in `messages.go:311`**

```go
// Story 99 will wire the root app to clear overlay-local search state in response.
```

Story 99 concluded the opposite: the overlay self-manages its clear; the app handler is a no-op. Replace with: "Emitted by SearchOverlay on Ctrl+U. App handler is a no-op — the overlay manages its own display-state clear inline."

---

**[m4] Unreachable fallback in `SearchRequestMsg` handler (`app.go:884`)**

```go
if len(searchTypes) == 0 {
    searchTypes = []string{"track", "artist", "album", "playlist"}
}
```

`TabToAPITypes` always returns ≥1 type; `m.Types` is always non-empty. Dead code — remove.

---

**[m5] Zero-results hint text is misleading (`search.go:1019`)**

After a search returns 0 results, `len(resultList.Items()) == 0` → "Type to search tracks, artists, albums..." — same as the no-query state. Users cannot distinguish "no results" from "hasn't searched yet."

**Fix:**

```go
func (o *SearchOverlay) renderResults(_ int) string {
    if o.results != nil && len(o.resultList.Items()) == 0 {
        return lipgloss.NewStyle().Foreground(o.theme.TextMuted()).Render("No results found.")
    }
    if len(o.resultList.Items()) == 0 {
        return lipgloss.NewStyle().Foreground(o.theme.TextMuted()).Render("Type to search tracks, artists, albums...")
    }
    return o.resultList.View()
}
```

---

### SUGGESTIONS

**[S1] Remove `clearCmd` from `Init()`.** Since `Reset()` is always called before `Init()`, the `SearchClearedMsg` from `Init()` is redundant. After C1 is fixed, removing it eliminates a confusing dead message.

**[S2] Replace dual `loadingFirstPage`/`loadingNextPage` booleans with a `loadingState` enum.** Makes illegal states (`both true`) unrepresentable. Improves switch-based rendering logic.

**[S3] Export `searchDebounceMsg` for integration testing.** Currently there is no way to drive debounce ticks in tests without real `tea.Tick` timing. Exporting the type allows tests to inject the message directly for deterministic coverage.

---

### DESIGN DOCUMENT GAPS

1. **Ctrl+U display state.** "Return to empty state" is underspecified — does not state that `o.results` and `o.total` must be explicitly zeroed. Implementation assumed the `SearchClearedMsg` roundtrip would handle it; that assumption was never validated.

2. **"No results" vs "type to search" distinction.** The design loading state table does not specify distinct text for a zero-results response vs no-query state. Implementation reuses the no-query hint.

3. **`isFirst` semantics on tab switch.** The design does not clarify whether a tab-switch with existing results should trigger `loadingFirstPage` (spinner-only) or `loadingNextPage` (spinner-above-old-results). Current code uses `len(Results()) == 0` which conflates new-query and new-tab scenarios.

4. **`InfiniteScrolling` on the list.** Not mentioned in the design doc. Clamping (the default) is more consistent with explicit pagination.

---

## Verdict

The architecture is fundamentally sound. The Elm purity is maintained, the staleness/cancellation model is well-designed, and the dual-layer debounce pattern is correct and elegant. The codebase is well-structured and well-commented throughout.

**However, one critical wiring bug ([C1]) causes Ctrl+U to appear to work but leave stale results visible** — this is the bug that has been "repeatedly not fixed" across iterations. The fix has been attempted at the overlay-handler level without recognising that the handler is unreachable: `case panes.SearchClearedMsg: return a, nil` in app.go returns before the overlay-forwarding fallthrough. The actual fix is 6 lines in `handleKey(KeyCtrlU)`, not a routing change. Additionally, **M1** (missing TrackName in queue toast), **M2** (stale `intent.tab` after prefix demotion), and **M3** (`InfiniteScrolling` violating the pagination contract) are genuine functional issues. Fix those four items and the implementation is shippable.
