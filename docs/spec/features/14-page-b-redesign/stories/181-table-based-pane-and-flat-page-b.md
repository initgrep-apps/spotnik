---
title: "Refactor: TableBasedPane consolidation, graded border filter label, flat Page B (delete RowSpan)"
feature: 14-page-b-redesign
status: open
---

## Background

Stories 173, 178, 179, and 180 each fixed a symptom of a deeper structural problem in
how filterable, table-based panes are composed. After three iterations of point fixes,
filter behaviour and the Page B layout still diverge from user expectations. This story
addresses the **structural** issues so the recurring symptoms stop being individually
patchable bugs.

There are four problems and they are interconnected:

### Problem 1 — Filter routing is duplicated across nine panes

`internal/ui/components/filter.go` exposes a `Filter` value object that holds a
`bubbles/textinput` plus a query string. The Filter is correct in isolation:
its `Query()` method returns the live in-progress text on every keystroke, which is
exactly what live-filtering UX requires (rows narrow as the user types).

The problem is that **every filterable pane re-implements the same Update routing
around Filter**:

```go
// queue.go, likedsongs_pane.go, toptracks_pane.go, topartists_pane.go,
// recentlyplayed_pane.go, networklog_pane.go, albums_pane.go, playlists_pane.go,
// gateway_live_pane.go — all repeat this pattern with minor variations:

if p.filter.IsActive() {
    cmd := p.filter.Update(msg)
    if !p.filter.IsActive() {
        p.table.SetFocused(true)
        p.resizeTable()
    }
    p.refreshRows()
    return p, cmd
}
if keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "f" {
    p.filter.Toggle()
    p.table.SetFocused(false)
    p.resizeTable()
    return p, nil
}
if keyMsg.Type == tea.KeyEscape {
    if p.filter.Query() != "" {
        p.filter.ClearQuery()
        p.refreshRows()
        return p, nil
    }
    p.table.GotoTop()
    return p, nil
}
```

This duplication is the **root cause of the divergence**. When story 178 fixed Esc
priority, it had to be implemented eight times. When story 173 added `Esc → GotoTop`,
it had to be wired into eight panes. When `GatewayLivePane` was authored, it took a
slightly different shape (committed `activeQuery` field, Enter-to-apply semantics)
and now behaves differently from its siblings. Each iteration creates new asymmetry
because the contract lives in nine places, not one.

### Problem 2 — `GatewayLivePane` uses a different filter contract

`gateway_live_pane.go` does not call `Filter.Query()` from `ActiveFilterQuery()`;
instead it has a separate `activeQuery string` field that is only set on Enter
(line 168). All other panes return `filter.Query()` directly. This is why the user
sees Gateway Live wait for Enter before showing the border label, while every other
pane shows the label as the user types.

The user has confirmed that the **live-filter behaviour is the desired one** for
all panes (rows narrow as you type, border updates as you type). Therefore Gateway
Live must be brought into alignment with the rest, not the other way around.

### Problem 3 — Border filter label is binary (full or dropped)

`internal/ui/layout/border.go:191-200` falls back from "render full filter notch"
straight to "drop right segment entirely" when the pane is too narrow. There is no
graded shrink. Result: when the pane is narrow, the entire `filtering: "query"`
label disappears. When an adjacent pane is toggled off and the pane resizes wider,
the label reappears. The user reports this as a "label sometimes vanishes" bug.

The user has also requested a shorter label form: `f(query)` instead of
`filtering: "query"`. This is both an improvement (reads cleaner, takes fewer
columns) and naturally enables a graded shrink by progressively trimming the query.

### Problem 4 — Page B layout uses RowSpan, which fights the user's mental model

Story 180 introduced `Cell.RowSpan` so `GatewayLive` could span two rows
vertically, with `GatewayHealth` and `PollingTraffic` stacked on the left. The
spanning cell's X coordinate is computed from **declared** widths (including
hidden cells) on purpose — the comment in `layout.go` step 3 calls this
"column alignment preserved when sibling cells are toggled off."

This intentional decision is in conflict with the user's expectation that hiding
a sibling pane causes other panes to fill the vacated space. Specifically:

- Press `2` to hide `GatewayHealth` → `PollingTraffic` should expand vertically
  into the vacated upper-left space. **Current:** the upper-left stays empty.
- Press `3` to hide `PollingTraffic` → `GatewayHealth` should expand vertically
  into the vacated lower-left space. **Current:** the lower-left stays empty.
- Press `2` and `3` to hide both → `GatewayLive` should expand to the full row
  width. **Current:** `GatewayLive` keeps its 70% column.
- Press `4` to hide `GatewayLive` → the two left panes should expand
  horizontally. **Current:** this works because the spanner is hidden, so the
  reserved interval disappears.

Patching `RowSpan` to redistribute weight when columns become empty is possible,
but it would compound the existing complexity (declared-vs-effective weights,
focus-order ordering, continuation rows). This is the third bug-fix iteration on
this path. Adding more conditional logic to `recompute()` is the trap.

The user has accepted a simpler design that **eliminates the need for RowSpan
entirely**: a flat single-row layout for the three diagnostic panes with weights
1:1:3 (Health, Traffic, Live). This delegates Page B's layout to the same flat
row+cell engine that Page A uses successfully. The toggle-redistribution behaviour
falls out of the existing engine for free.

---

## Goals

1. **Single source of truth for filter routing** — all table-based panes share one
   implementation. Adding/changing filter behaviour touches one file, not nine.
2. **Uniform live-filter UX across all nine panes** — including `GatewayLive`. Rows
   narrow as the user types; border label updates as the user types. Uniform
   `Esc` behaviour with three distinct modes: while filter input is active →
   cancel (clear query and close input); while filter input is closed and a
   query is preserved → clear the query; otherwise → reset table scroll to
   page 1.
3. **Graceful filter label shrink** — narrow panes show progressively shorter
   variants of the filter label rather than dropping it. Default form is `f(query)`.
4. **Simpler Page B layout** — flat row of three diagnostic panes (1:1:3). Toggling
   any pane redistributes width using the existing flat-row engine. RowSpan is
   deleted because it has no other consumer.

## Non-goals

- No new keybindings. No changes to `docs/keybinding.md`, `docs/DESIGN.md §17`, or
  `helpContent` in `help_overlay.go`.
- No changes to `BasePane` semantics — `TableBasedPane` is layered on top of it.
- No changes to Page A presets or behaviour. Their `Cell` literals continue to use
  the two-field named-form (`{PaneID: x, WidthWeight: y}`) and continue to work
  unchanged after `RowSpan` is deleted.
- No new `ListBasedPane` abstraction. All nine filterable panes today use
  `components.Table` (including `GatewayLivePane`, which uses a single-column
  no-header Table). YAGNI: when a real list-backed pane needs filtering, extract a
  shared interface then with two concrete examples in hand.
- No change to the `bubbles/textinput`-backed `Filter` component's existing public
  surface (`Toggle`, `Update`, `View`, `Query`, `ClearQuery`, `Matches`,
  `MatchesAny`, `IsActive`, `SetWidth`, `BorderLabel`). The component is correct;
  the duplication is in its callers.

---

## Design and Architecture

### Composition over inheritance for table panes

Go has no inheritance, but it has struct embedding. `BasePane` already uses this
pattern: it provides `store`, `theme`, `focused`, `width`, `height` plus default
`IsFocused()`, `SetFocused()`, `SetSize()`, `HasActiveFilter()` implementations.
Concrete panes embed `BasePane` and override what they need.

`TableBasedPane` extends this pattern one level: it embeds `BasePane` and adds the
two pieces every filterable table pane needs — a `*components.Table` and a
`*components.Filter`. It also exposes a single method, `HandleFilterKey`, that
contains the shared filter-routing block. Concrete panes call this method early in
their `Update` and let it consume the keys it knows about.

The choice to use a method that returns `(consumed bool, cmd tea.Cmd)` rather than
storing callbacks on the struct comes from a constraint of Go embedding:

- A pane embeds `*TableBasedPane`. Its `refreshRows()` and `resizeTable()` are
  pane-specific methods that read the pane's own fields (e.g. `q.store.Queue()`).
- If `TableBasedPane` stored callback fields, they would have to be set after
  construction (because the pane that owns them does not exist yet at
  `TableBasedPane` construction time). That's a mutable-field setter pattern,
  which is fragile (forgetting to set them is a runtime nil-deref).
- Passing the callbacks as parameters to `HandleFilterKey` makes the contract
  explicit at every call site, eliminates the nil-deref risk, and keeps
  `TableBasedPane` immutable after construction.

### Live-filter contract (codified)

After this story, the contract for every filterable table-based pane is:

| Event | Behaviour |
| --- | --- |
| `f` (filter not active) | Open filter input. Table loses focus. Table height shrinks by 1 row to accommodate the filter bar. Pane border `Actions()` returns empty (filter label takes the right segment). |
| any printable rune (filter active) | Append to query. Border label updates live (`f(query)`). Rows narrow to live matches. |
| `Backspace` (filter active) | Remove last char from query. Border label updates. Rows widen back. |
| `Enter` (filter active) | Close filter input, preserve query. Table regains focus. Table height grows. Border continues to render `f(query)` until query is cleared. |
| `Esc` (filter active) | Cancel filter — clear query, close input. Rows widen back. Table regains focus. Border returns to default action shortcut. |
| `Esc` (filter closed, query non-empty) | Clear committed query. Rows widen back. Border returns to default action shortcut. |
| `Esc` (filter closed, query empty) | `table.GotoTop()` — scroll to first page. |
| `Backspace` (filter closed, query non-empty) | No-op. (Filter is closed; `Backspace` belongs to pane navigation if any, otherwise unconsumed.) |

`Esc` is a global key with one rule across the whole app: **close the current
context** (overlay → close, filter → cancel/clear, otherwise → reset scroll).
That rule lives once in the help overlay (`?`); panes do not advertise `Esc`
in their borders.

The pane is free to handle additional keys (`Enter` to play a track, `j/k` for
navigation, `g`/`G` etc.) **after** `HandleFilterKey` returns `consumed=false`.

### Border graded shrink algorithm

The border renderer is the single point that decides how much of the filter label
fits on the top border line. When the right-segment budget is too small for the
default form, the renderer tries progressively shorter variants:

| Variant | Example (query = `jazz`) | Width (cols) |
| --- | --- | --- |
| 1. Full | `f(jazz)` | 7 |
| 2. Truncated | `f(ja…)` | 6 (varies) |
| 3. Minimal | `f(…)` | 5 |
| 4. Drop | (empty) | 0 |

The variant is chosen by the largest one that fits the budget. The truncation
variant trims the query rune-by-rune (with a single trailing `…`) until the result
fits or until only one rune remains, in which case it falls through to variant 3.

**No close-notch in filter mode.** Earlier iterations rendered a `╮ Esc close ╭`
notch alongside the label. This is removed: `Esc` is a global key documented once
in the help overlay (`?`), not repeated per-pane. The pane border in filter mode
is exclusively the label — that frees the entire right-segment budget for the
query text and removes the ~13-column overhead that caused narrow panes to drop
the label entirely. Same reasoning as overlays: pressing `Esc` closes the current
context (overlay, filter, modal) without per-context labelling.

### Layout: flat rows are sufficient

Removing `RowSpan` reduces `recompute()` from a multi-pass spanner-aware engine
(~280 LOC) to a straightforward two-loop layout (~50 LOC):

1. For each row in the active preset's grid, collect the visible cells.
2. If there are no visible cells in a row, drop the row entirely (live rows only).
3. Distribute the content height across live rows by `HeightWeight`.
4. For each row, distribute the row width across visible cells by `WidthWeight`.

The toggle-redistribution behaviour the user wants is the natural consequence of
step 4: when a cell is hidden, it is not counted in the row's total width weight,
so the remaining cells absorb its share proportionally.

Page A presets already work this way; no change is needed there. Page B's
`PresetNerdStatus` is reverted to a flat three-row layout that uses the same
algorithm.

---

## Implementation Specification

### Part A — `TableBasedPane` (new file)

**File:** `internal/ui/panes/table_based_pane.go` (new)

```go
// Package panes — TableBasedPane is the embedded base for every table-backed
// pane that supports in-pane text filtering. It owns the Filter and Table
// references and provides the shared filter-routing block (HandleFilterKey)
// that every filterable pane uses identically.
//
// Concrete panes:
//   - embed *TableBasedPane (pointer embedding so the pane can swap table/filter
//     references during SetTheme rebuild)
//   - construct it via NewTableBasedPane(store, theme, focused, table, filter)
//   - call tbp.HandleFilterKey(keyMsg, refreshRows, resizeTable) at the top of
//     their Update; if consumed=true, return cmd
//   - implement pane-specific keys after HandleFilterKey returns false
package panes

import (
    tea "github.com/charmbracelet/bubbletea"

    "github.com/initgrep-apps/spotnik/internal/state"
    "github.com/initgrep-apps/spotnik/internal/ui/components"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// TableBasedPane embeds BasePane and adds Table+Filter for table-backed panes.
//
// It satisfies layout.FilterablePane via HasActiveFilter (overrides BasePane's
// false default) and layout.FilterQueryPane via ActiveFilterQuery, so concrete
// panes that embed it inherit both interface implementations and do not need
// to declare them.
type TableBasedPane struct {
    BasePane
    table  *components.Table
    filter *components.Filter
}

// NewTableBasedPane constructs a TableBasedPane with the given dependencies.
// The caller is responsible for constructing the Table and Filter with the
// pane's specific column layout — the base does not impose a column shape.
func NewTableBasedPane(
    store state.StateReader,
    th theme.Theme,
    focused bool,
    table *components.Table,
    filter *components.Filter,
) *TableBasedPane {
    return &TableBasedPane{
        BasePane: BasePane{store: store, theme: th, focused: focused},
        table:    table,
        filter:   filter,
    }
}

// Table returns the embedded table reference. Used by panes for row updates,
// SetFocused/SetSize forwarding, and SetTheme rebuilds.
func (b *TableBasedPane) Table() *components.Table { return b.table }

// Filter returns the embedded filter reference. Used by panes for query
// inspection (filteredXxx helpers) and SetTheme rebuilds.
func (b *TableBasedPane) Filter() *components.Filter { return b.filter }

// SwapTableAndFilter replaces both references in one atomic call. Used by
// SetTheme implementations that rebuild Table and Filter together.
func (b *TableBasedPane) SwapTableAndFilter(t *components.Table, f *components.Filter) {
    b.table = t
    b.filter = f
}

// HasActiveFilter reports whether the filter input is currently capturing keys.
// Overrides BasePane's default-false implementation to satisfy
// layout.FilterablePane.
func (b *TableBasedPane) HasActiveFilter() bool { return b.filter.IsActive() }

// ActiveFilterQuery returns the current filter query for border display.
// Satisfies layout.FilterQueryPane. Returns the LIVE query (updates on every
// keystroke) — this is the intentional UX contract: the user sees their query
// in the border as they type it.
func (b *TableBasedPane) ActiveFilterQuery() string { return b.filter.Query() }

// HandleFilterKey processes the three filter-related key paths shared by every
// filterable table pane:
//
//   1. Filter is active → forward the key to filter.Update; if filter just
//      closed (Enter/Esc consumed it), refocus the table and call resizeTable;
//      always call refreshRows so live filtering takes effect.
//   2. Filter is inactive and the key is 'f' → toggle filter on, table loses
//      focus, call resizeTable.
//   3. Filter is inactive and the key is Esc → if a committed query exists,
//      ClearQuery + refreshRows; otherwise table.GotoTop().
//
// Returns (consumed=true) if the key was handled — the pane should return
// cmd without further processing. Returns (false, nil) if the key should
// fall through to pane-specific handling.
//
// Hooks (called after Filter state is updated, so callers reading
// p.filter.IsActive() / p.filter.Query() inside the hook see the new state):
//   - refreshRows: re-read the store with current query and update rows
//   - resizeTable: adjust table height for filter-bar visibility
//
// Both hooks must be non-nil. Passing nil is a programmer error and panics
// — fail loud, fail early.
func (b *TableBasedPane) HandleFilterKey(
    msg tea.KeyMsg,
    refreshRows func(),
    resizeTable func(),
) (consumed bool, cmd tea.Cmd) {
    if refreshRows == nil || resizeTable == nil {
        panic("TableBasedPane.HandleFilterKey: refreshRows and resizeTable must be non-nil")
    }

    // Path 1 — filter active: forward and refresh.
    if b.filter.IsActive() {
        cmd = b.filter.Update(msg)
        if !b.filter.IsActive() {
            // Filter closed via Enter or Esc — restore table focus + height.
            b.table.SetFocused(true)
            resizeTable()
        }
        // Always refresh: query may have changed (typing) or been committed/cancelled.
        refreshRows()
        return true, cmd
    }

    // Path 2 — 'f' opens the filter. Match exactly one rune to avoid swallowing
    // multi-rune key events (paste-bursts, IME, function keys with rune payloads).
    if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == 'f' {
        b.filter.Toggle()
        b.table.SetFocused(false)
        resizeTable()
        return true, nil
    }

    // Path 3 — Esc when filter is closed.
    if msg.Type == tea.KeyEscape {
        if b.filter.Query() != "" {
            b.filter.ClearQuery()
            refreshRows()
            return true, nil
        }
        b.table.GotoTop()
        return true, nil
    }

    return false, nil
}
```

**Why these specific decisions:**

- **Pointer embedding** (`*TableBasedPane`, not `TableBasedPane`): so the
  embedding pane can swap the underlying table/filter references after a theme
  rebuild without re-constructing the base. Value embedding would force a copy.
- **Hooks as parameters, not struct fields**: avoids nil-deref risk after
  construction; the contract is explicit at every call site.
- **Panic on nil hooks**: this is a programmer error, not a user-facing failure.
  Fail loud at the first call rather than silently no-op.
- **`len(msg.Runes) == 1` guard on `'f'`**: the existing pane code uses
  `string(keyMsg.Runes) == "f"` which silently accepts multi-rune sequences.
  Tightening the guard avoids a subtle class of bugs (paste of `"foo"`
  triggering filter mode).

### Part B — Migrate panes to `TableBasedPane`

For each of the nine filterable panes, the migration is:

1. Replace `BasePane` embedding with `*TableBasedPane` embedding.
2. Move `table` and `filter` fields from the pane struct into the
   `TableBasedPane` (accessed via `p.Table()`, `p.Filter()`).
3. Construct via `NewTableBasedPane(store, theme, focused, table, filter)`.
4. In `Update`, replace the duplicated filter routing with a single call:
   ```go
   if keyMsg, ok := msg.(tea.KeyMsg); ok {
       if consumed, cmd := p.HandleFilterKey(keyMsg, p.refreshRows, p.resizeTable); consumed {
           return p, cmd
       }
       // pane-specific keys (Enter, j/k that aren't covered by the table, etc.)
   }
   cmd := p.Table().Update(msg)
   return p, cmd
   ```
5. Delete the pane's own `ActiveFilterQuery()` and `HasActiveFilter()` methods —
   inherited from the base.
6. Keep the pane's `refreshRows`, `resizeTable`, `filteredXxx` helpers (their
   shape is per-pane; the base does not impose a row schema).
7. The `SetTheme` path uses `p.SwapTableAndFilter(newTable, newFilter)` after
   rebuilding both with the new theme.
8. **Drop the `Esc close` action branch from `Actions()`; consolidate the
   default into `TableBasedPane`.** The close-notch is no longer rendered, so
   the action that produced it is removed. The remaining `[{f, filter}]` hint
   is identical across every filterable pane, so it moves into the base.

   **Why no `if HasActiveFilter()` check is needed:** the border renderer
   evaluates `cfg.FilterQuery` before `cfg.Actions`. Whenever the query is
   non-empty (during typing or after Enter), the filter branch wins and
   `Actions()` is ignored. The only state where `Actions()` matters during
   filtering is the brief window between pressing `f` and typing the first
   character — query empty, filter active. In that window the hint
   `╮ f filter ╭` continues to render, which is fine: the user has not yet
   begun typing, the hint is consistent with the closed state, and once the
   first keystroke arrives the filter label takes over. Special-casing this
   window adds branching without UX value.

   **Default on the base:**

   ```go
   // table_based_pane.go
   //
   // Actions returns the default action shortcut for filterable table panes:
   // [{Key: "f", Label: "filter"}]. The filter-mode label (rendered by border.go
   // when FilterQuery != "") takes over the right segment automatically; the
   // hint is harmless in the brief active+empty-query window before typing.
   //
   // Panes that need additional actions (e.g. Albums list-view, Playlists
   // reorder hints) override this method and call back into the base via
   // BaseFilterAction() to compose their own slice.
   func (b *TableBasedPane) Actions() []layout.Action {
       return []layout.Action{{Key: "f", Label: "filter"}}
   }

   // BaseFilterAction returns the single {f, filter} action so subclasses can
   // compose their own Actions() without duplicating the literal.
   func (b *TableBasedPane) BaseFilterAction() layout.Action {
       return layout.Action{Key: "f", Label: "filter"}
   }
   ```

   **Per-pane consequence:**
   - Simple panes (`QueuePane`, `LikedSongsPane`, `TopTracksPane`,
     `TopArtistsPane`, `RecentlyPlayedPane`, `NetworkLogPane`,
     `GatewayLivePane`) **delete** their `Actions()` method entirely — the
     base default is correct.
   - Composite panes (`AlbumsPane`, `PlaylistsPane`) keep their override but
     compose using `b.BaseFilterAction()` and stop branching on
     `filter.IsActive()`:
     ```go
     // Before
     func (a *AlbumsPane) Actions() []layout.Action {
         if a.filter.IsActive() {
             return []layout.Action{{Key: "Esc", Label: "close"}}
         }
         return []layout.Action{
             {Key: "f", Label: "filter"},
             {Key: "Enter", Label: "open"},
         }
     }

     // After
     func (a *AlbumsPane) Actions() []layout.Action {
         return []layout.Action{
             a.BaseFilterAction(),
             {Key: "Enter", Label: "open"},
         }
     }
     ```

**Per-pane notes:**

| Pane | Receiver | Pane-specific keys after `HandleFilterKey` | Notes |
|---|---|---|---|
| `QueuePane` | `q` | `Enter` → emit `PlayTrackMsg` | `filteredQueue()` helper retained |
| `PlaylistsPane` | `p` | `Enter` → enter list view; `Shift+Up/Down` reorder; in-list `Esc` → back to grid | Two-mode pane (grid/list); filter only applies in list mode. `HandleFilterKey` is called only in list-view branch. |
| `AlbumsPane` | `a` | `Enter` → enter list view; in-list `Esc` → back to grid | Same two-mode shape as Playlists |
| `LikedSongsPane` | `l` | `Enter` → emit `PlayTrackMsg` | `filteredTracks()` retained |
| `RecentlyPlayedPane` | `r` | `Enter` → emit `PlayTrackMsg` | |
| `TopTracksPane` | `p` | `Enter` → emit `PlayTrackMsg` | |
| `TopArtistsPane` | `a` | `Enter` → drill into top artist tracks | |
| `NetworkLogPane` | `p` | none (read-only stream) | `pendingDecisions` field unchanged; `refreshRows` re-applies completedRequests through filter |
| `GatewayLivePane` | `p` | none (read-only stream) | **Behaviour change** — see below |

**`GatewayLivePane` specific changes** (most impactful behaviour shift):

- Delete the `activeQuery string` field.
- Delete the bespoke `handleKey` Enter/Esc logic that maintained `activeQuery`.
- `buildTableRows()` reads from `p.Filter().Query()` instead of `p.activeQuery`.
- **Hook wiring:** call `HandleFilterKey(keyMsg, p.buildTableRows, p.resizeTable)`.
  `buildTableRows` is the GatewayLive-specific name for what other panes call
  `refreshRows` — do not rename it for consistency, do not add a redundant
  `refreshRows` alias.
- Now the live event stream filters incrementally as the user types
  (matching the other eight panes). On `Enter`, the input closes but the query
  is preserved (live behaviour kept). On `Esc` while filter is open, query is
  cleared (cancel). On `Esc` while filter is closed and query is non-empty,
  the query is cleared (revealing the full event stream). On `Esc` again, the
  table scroll resets.
- `drainEvents()` (TickMsg path) keeps calling `buildTableRows()` so newly
  arrived events are filtered correctly against the current live query.
- The user-visible change: previously you had to press `Enter` to apply the
  filter. After this story, the filter applies as you type. This is the desired
  alignment with the other panes.

### Part C — Border graded shrink

**File:** `internal/ui/layout/border.go`

Add a new helper:

```go
// formatFilterLabel returns the most informative filter label that fits
// within the given column budget. Tries variants from widest to narrowest:
//
//   1. f(rock)       — preferred; full query in unquoted form
//   2. f(ro…)        — truncated; trims the query rune-by-rune with a trailing …
//   3. f(…)          — minimal; signals an active filter without showing query
//   4. ""            — drop; even f(…) does not fit
//
// The returned string is unstyled — the caller is responsible for applying
// theme colours via mutedStyle.
//
// budget is in terminal columns. The right segment in filter mode is exactly
// this label — no close-notch is rendered. Esc is a global key documented in
// the help overlay (`?`), not repeated per-pane.
func formatFilterLabel(query string, budget int) string {
    if query == "" || budget <= 0 {
        return ""
    }

    // Variant 1 — full unquoted form
    if v := "f(" + query + ")"; lipgloss.Width(v) <= budget {
        return v
    }
    // Variant 2 — progressively trim the query, append …
    runes := []rune(query)
    for i := len(runes) - 1; i >= 1; i-- {
        v := "f(" + string(runes[:i]) + "…)"
        if lipgloss.Width(v) <= budget {
            return v
        }
    }
    // Variant 3 — minimal indicator
    if v := "f(…)"; lipgloss.Width(v) <= budget {
        return v
    }
    // Variant 4 — drop
    return ""
}
```

Update `buildRightSegment` to use the helper. The segment currently renders the
filter mode as:

```text
filtering: "query" ─╮ Esc close ╭
```

After this story, filter mode renders as just the label:

```text
f(query)
```

The label progressively shrinks via `formatFilterLabel` when the right-segment
budget is small. There is no close-notch — `Esc` is a global keybinding
documented once in the help overlay (`?`). This is the same convention used by
overlay panes (Search, Profile, Help): `Esc` closes the current context without
per-context labelling.

**Refactored `buildRightSegment` shape (filter branch):**

```go
if cfg.FilterQuery != "" {
    // Filter mode: render the most informative label that fits the budget.
    // No close-notch — Esc is a global key documented in the help overlay.
    label := formatFilterLabel(cfg.FilterQuery, budget)
    if label == "" {
        return ""
    }
    return mutedStyle(label)
}
```

The action-mode branch (used when `cfg.FilterQuery == ""` and `cfg.Actions` is
non-empty) is unchanged — it continues to render the corner-notch action
shortcuts as today.

`buildRightSegment` must take a `budget int` parameter (passed from
`RenderPaneBorder`). The budget is computed as:

```go
const minDashes = 1 // at least one dash between leftInner and rightSegment
budget := outerWidth - 2 /* corners */ - lipgloss.Width(leftPrefix) -
          lipgloss.Width(leftInner) - minDashes
if budget < 0 { budget = 0 }
```

The existing `dashCount < 0` fallback in `RenderPaneBorder` becomes a guarantee
(by construction) rather than a runtime check. The truncate-title fallback for
extremely narrow panes is preserved.

### Part D — Flat Page B layout, delete `RowSpan`

**File:** `internal/ui/layout/pane.go`

Revert `Cell` to two fields:

```go
// Cell represents a pane slot in a row with its relative width.
type Cell struct {
    PaneID      PaneID
    WidthWeight int
}
```

Delete the `rowSpan() int` helper.

**File:** `internal/ui/layout/layout.go`

Replace the entire `recompute()` body (lines 47–391) with a flat two-loop layout:

```go
// recompute recalculates all Rects from the active preset + hidden state.
// Called after Resize, SetPreset, CyclePreset, TogglePage, TogglePane.
//
// Algorithm:
//   1. Filter the preset's grid to live rows (rows with ≥1 visible cell).
//   2. Distribute terminal content height across live rows by HeightWeight.
//   3. For each row, distribute row width across visible cells by WidthWeight.
//
// When a cell is hidden (by TogglePane), its weight drops from the row's total
// and its row-mates absorb its share proportionally. When all cells in a row
// are hidden, the row drops out and remaining rows absorb its height share.
func (m *Manager) recompute() {
    m.rects = make(map[PaneID]Rect)

    presets := m.presets[m.activePage]
    presetIdx := m.activePreset[m.activePage]
    if presetIdx >= len(presets) {
        return
    }
    grid := presets[presetIdx]

    isVisible := func(id PaneID) bool {
        return grid.Visible[id] && !m.hidden[id]
    }

    // Step 1 — collect live rows.
    type liveCell struct {
        paneID      PaneID
        widthWeight int
    }
    type liveRow struct {
        heightWeight int
        cells        []liveCell
    }
    var liveRows []liveRow
    for _, row := range grid.Grid {
        var cells []liveCell
        for _, c := range row.Cells {
            if isVisible(c.PaneID) {
                cells = append(cells, liveCell{c.PaneID, c.WidthWeight})
            }
        }
        if len(cells) > 0 {
            liveRows = append(liveRows, liveRow{row.HeightWeight, cells})
        }
    }

    if len(liveRows) == 0 {
        m.focusOrder = nil
        m.clampFocusIndex()
        return
    }

    // Step 2 — distribute height.
    contentH := m.height - m.headerHeight - m.statusHeight
    if contentH < 0 {
        contentH = 0
    }
    totalHWeight := 0
    for _, r := range liveRows {
        totalHWeight += r.heightWeight
    }

    // Step 3 — place cells row-by-row.
    var newFocusOrder []PaneID
    y := 0
    for i, row := range liveRows {
        var h int
        switch {
        case totalHWeight == 0:
            h = 0
        case i == len(liveRows)-1:
            h = contentH - y // last row absorbs rounding remainder
        default:
            h = contentH * row.heightWeight / totalHWeight
        }
        if h < 0 {
            h = 0
        }

        totalWWeight := 0
        for _, c := range row.cells {
            totalWWeight += c.widthWeight
        }
        x := 0
        for j, c := range row.cells {
            var w int
            switch {
            case totalWWeight == 0:
                w = 0
            case j == len(row.cells)-1:
                w = m.width - x // last cell absorbs rounding remainder
            default:
                w = m.width * c.widthWeight / totalWWeight
            }
            if w < 0 {
                w = 0
            }
            m.rects[c.paneID] = Rect{X: x, Y: y, Width: w, Height: h}
            newFocusOrder = append(newFocusOrder, c.paneID)
            x += w
        }
        y += h
    }

    // Step 4 — restore focus to previously focused pane if still visible.
    prevFocused := m.currentFocusedPane()
    m.focusOrder = newFocusOrder
    m.restoreFocus(prevFocused)
}
```

This is ~70 LOC versus the current ~280 LOC. The behaviour for Page A presets is
identical (Page A never used RowSpan). The behaviour for Page B is now natural
toggle redistribution.

**File:** `internal/ui/layout/presets.go`

Update `PresetNerdStatus` to a flat three-row layout (1:1:3 weights):

```go
// PresetNerdStatus shows NowPlaying strip, three diagnostic panes side-by-side
// (Health, Traffic, Live with weights 1:1:3 → ~20%/20%/60%), and NetworkLog
// full-width below. All five panes are individually toggleable via keys 1-5.
var PresetNerdStatus = Preset{
    Name: "Nerd Status",
    Visible: map[PaneID]bool{
        PaneNowPlaying:     true,
        PaneGatewayHealth:  true,
        PanePollingTraffic: true,
        PaneGatewayLive:    true,
        PaneNetworkLog:     true,
    },
    Grid: []Row{
        {HeightWeight: 1, Cells: []Cell{
            {PaneID: PaneNowPlaying, WidthWeight: 1},
        }},
        {HeightWeight: 3, Cells: []Cell{
            {PaneID: PaneGatewayHealth, WidthWeight: 1},
            {PaneID: PanePollingTraffic, WidthWeight: 1},
            {PaneID: PaneGatewayLive, WidthWeight: 3},
        }},
        {HeightWeight: 2, Cells: []Cell{
            {PaneID: PaneNetworkLog, WidthWeight: 1},
        }},
    },
}
```

**Height ratio rationale (1:3:2):**

| Row | Pane(s) | Content shape | Why this weight |
|---|---|---|---|
| 1 | NowPlaying | Single track strip | `1` — slim header, ~1/6 of terminal height |
| 2 | Health · Traffic · Live | Health = 4 fixed rows; Traffic = 5 fixed rows; Live = scrollable stream | `3` — Live needs the room; Health/Traffic are short but their borders+padding fill any extra |
| 3 | NetworkLog | Scrollable request log | `2` — secondary surface, less emphasis than Live |

For a 200×50 terminal: content area = 50 − 1 (header) − 3 (status) = 46 rows.
At 1:3:2, rows compute to ~7, ~23, ~16. Health (4 fixed rows in a 23-row pane)
is generous — the empty interior is acceptable since the grid is rendered with
top alignment. If smoke testing shows too much empty space in Health/Traffic,
adjust to 1:2:2 (smaller middle, larger NetworkLog) or 1:3:1 (smaller
NetworkLog). These are tuning knobs; pick after running locally.

---

## Edge Cases

1. **Filter active + number key (`'1'`-`'5'`)**: must be forwarded to the filter
   input as part of the query, not consumed as a toggle. The existing routing in
   `internal/app/routing.go` checks `HasActiveFilter()` before routing toggle
   keys; this remains correct because `TableBasedPane.HasActiveFilter()`
   delegates to `filter.IsActive()`.

2. **Filter active + `Tab` (focus rotation)**: same — must be forwarded to the
   filter input, not consumed as Tab. Existing routing protects this.

3. **Hide all visible panes via toggles**: `TogglePane`'s "cannot hide last pane"
   guard remains. With flat layout, the guard is even simpler — no spanner
   special-casing needed.

4. **GatewayLive TickMsg arrives while filter is active**: `drainEvents()`
   prepends new events to the buffer and calls `buildTableRows()`. The filter's
   live `Query()` is read inside `buildTableRows()`, so new events are filtered
   against the current in-progress query. No interaction problem.

5. **Filter typed character that is special to bubbles/textinput** (`Backspace`,
   `Delete`, `Home`, `End`, arrow keys): forwarded to the textinput inside
   `Filter.Update`; the textinput handles them. No special-casing needed in
   `HandleFilterKey`.

6. **Pane is unfocused while filter is active**: invariant — **no code path
   moves focus away from a pane that has an active filter input.** The routing
   layer's `HasActiveFilter()` check protects against `Tab` rotation and
   number-key toggles. Other focus-changing operations must also respect this
   invariant:
   - `TogglePage` (key `0`): closes Page A's filter on departure (no Page B
     pane on Page A's preset, so `restoreFocus` lands on a different pane).
     Add: when leaving a page with an active filter, call `Filter.Toggle()` to
     deactivate before the focus shift, so no orphaned active filter persists.
   - `CyclePreset` (key `]`): same — call `Filter.Toggle()` on the previously
     focused pane before recompute.
   - `SetFocus(id)` (currently called only from tests): document that callers
     are responsible for deactivating any active filter first.

   **Test for the invariant:**
   ```go
   func TestQueuePane_FilterActive_PageToggleDeactivatesFilter(t *testing.T) {
       a := newTestApp(t)
       a.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
       // Open filter on Queue
       a.Layout().SetFocus(layout.PaneQueue)
       a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
       require.True(t, a.QueuePane().HasActiveFilter())
       // Toggle page
       a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
       assert.False(t, a.QueuePane().HasActiveFilter(),
           "page toggle must deactivate any pane's active filter")
   }
   ```

   If the invariant is violated, the filter pane is unfocused but `IsActive()`
   returns true. The pane's `Update` returns early on `!IsFocused()`, so keys
   are not consumed by the filter — they fall through to global routing. The
   filter input cursor blinks until focus returns. This is benign but
   confusing; the invariant prevents it.

7. **Border graded shrink + theme TextMuted**: the styled output uses
   `mutedStyle` which applies theme `TextMuted()` foreground. Verify the styled
   width equals the unstyled width (`lipgloss.Width` ignores ANSI). The graded
   shrink uses unstyled widths to compute fit, then applies styles to the
   chosen variant. `lipgloss.Width` is safe under these styles.

8. **Truncated query ends with combining mark or wide rune**: the rune-by-rune
   trim in variant 3 cuts on rune boundaries (Go ranges over runes). For
   East-Asian wide runes (`lipgloss.Width('字') == 2`), the loop respects the
   total width via `lipgloss.Width(v)`. No grapheme-cluster awareness — if a
   query contains combining marks, the trim may split the grapheme; this is a
   known limitation matching the rest of the codebase's existing rune-trimming
   helpers (`truncateToColumns` in border.go does the same).

9. **Theme rebuild during active filter**: `SetTheme` in each pane rebuilds
   table+filter and calls `SwapTableAndFilter`. If the rebuild creates a fresh
   Filter, the active state and query are lost. This is the existing behaviour
   (current panes also lose filter state on theme switch); preserving it is
   out of scope. Document in the per-pane SetTheme.

   **Cheap follow-up (not in this story):** with `SwapTableAndFilter` available,
   `SetTheme` could rebuild only the Table and retain the existing Filter
   reference (Filter holds a theme reference for placeholder/text styles —
   call `Filter.SetTheme(th)` if added later, or accept the slight visual
   inconsistency until the user re-opens the filter). Track as `docs/spec/issues.md`.

10. **Existing `BorderLabel` method on `Filter`**: currently returns
    `filtering: "query"`. The render path does not call it (it builds the
    string directly in `border.go`). Two options: (a) leave `BorderLabel`
    untouched as dead code; (b) repoint it to `formatFilterLabel(query, math.MaxInt)`
    for callers who do not have a budget. Recommend (b) — minor consistency win.

---

## Tests

### Component tests — `TableBasedPane`

**File:** `internal/ui/panes/table_based_pane_test.go` (new)

These tests use a minimal fake `Table` and a real `Filter`, since the base does
not depend on a specific column layout.

```go
func TestTableBasedPane_HandleFilterKey_NotConsumedForOtherKeys(t *testing.T) {
    b := newTestTableBasedPane(t)
    consumed, cmd := b.HandleFilterKey(
        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")},
        func() {}, func() {},
    )
    assert.False(t, consumed)
    assert.Nil(t, cmd)
}

func TestTableBasedPane_HandleFilterKey_FActivatesFilter(t *testing.T) {
    b := newTestTableBasedPane(t)
    var resized int
    consumed, _ := b.HandleFilterKey(
        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")},
        func() {}, func() { resized++ },
    )
    assert.True(t, consumed)
    assert.True(t, b.HasActiveFilter())
    assert.Equal(t, 1, resized)
}

func TestTableBasedPane_HandleFilterKey_ForwardsToFilterWhenActive(t *testing.T) {
    b := newTestTableBasedPane(t)
    b.Filter().Toggle() // activate
    var refreshed int
    b.HandleFilterKey(
        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")},
        func() { refreshed++ }, func() {},
    )
    assert.Equal(t, "r", b.Filter().Query())
    assert.Equal(t, 1, refreshed)
}

func TestTableBasedPane_HandleFilterKey_EnterClosesFilterPreservesQuery(t *testing.T) {
    b := newTestTableBasedPane(t)
    b.Filter().Toggle()
    b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}, func() {}, func() {})
    b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}, func() {}, func() {})
    consumed, _ := b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyEnter}, func() {}, func() {})
    assert.True(t, consumed)
    assert.False(t, b.HasActiveFilter())
    assert.Equal(t, "ro", b.Filter().Query())
}

func TestTableBasedPane_HandleFilterKey_EscWhileActiveCancels(t *testing.T) {
    b := newTestTableBasedPane(t)
    b.Filter().Toggle()
    b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}, func() {}, func() {})
    consumed, _ := b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyEscape}, func() {}, func() {})
    assert.True(t, consumed)
    assert.False(t, b.HasActiveFilter())
    assert.Equal(t, "", b.Filter().Query())
}

func TestTableBasedPane_HandleFilterKey_EscWhenClosedClearsCommittedQuery(t *testing.T) {
    b := newTestTableBasedPane(t)
    b.Filter().Toggle()
    b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}, func() {}, func() {})
    b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyEnter}, func() {}, func() {})
    require.Equal(t, "r", b.Filter().Query())

    var refreshed int
    consumed, _ := b.HandleFilterKey(
        tea.KeyMsg{Type: tea.KeyEscape},
        func() { refreshed++ }, func() {},
    )
    assert.True(t, consumed)
    assert.Equal(t, "", b.Filter().Query())
    assert.Equal(t, 1, refreshed)
}

func TestTableBasedPane_HandleFilterKey_EscWhenClosedAndNoQueryGotoTop(t *testing.T) {
    b := newTestTableBasedPane(t)
    // ensure the table is on a non-zero page first (helper sets it up)
    b.Table().GotoPage(2)
    consumed, _ := b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyEscape}, func() {}, func() {})
    assert.True(t, consumed)
    assert.Equal(t, 1, b.Table().CurrentPage())
}

func TestTableBasedPane_HandleFilterKey_PanicsOnNilHooks(t *testing.T) {
    b := newTestTableBasedPane(t)
    assert.Panics(t, func() {
        b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyEscape}, nil, func() {})
    })
}

func TestTableBasedPane_ActiveFilterQuery_LiveValueWhileTyping(t *testing.T) {
    b := newTestTableBasedPane(t)
    b.Filter().Toggle()
    b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}, func() {}, func() {})
    assert.Equal(t, "r", b.ActiveFilterQuery())
    b.HandleFilterKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")}, func() {}, func() {})
    assert.Equal(t, "ro", b.ActiveFilterQuery())
}

// Pin the design decision that Actions() does NOT return nil while the filter
// is active with an empty query. Rationale: the border renderer evaluates
// FilterQuery before Actions; once the user types a character the filter
// label takes over the right segment. The brief active+empty-query window
// before the first keystroke harmlessly continues to render the {f, filter}
// hint — special-casing it adds branching without UX value.
func TestTableBasedPane_Actions_NotNilWhenFilterActiveWithEmptyQuery(t *testing.T) {
    b := newTestTableBasedPane(t)
    b.Filter().Toggle()
    require.True(t, b.HasActiveFilter())
    require.Equal(t, "", b.Filter().Query())

    actions := b.Actions()
    require.Len(t, actions, 1, "default Actions() returns the {f, filter} hint regardless of filter state")
    assert.Equal(t, "f", actions[0].Key)
    assert.Equal(t, "filter", actions[0].Label)
}
```

### Border tests — `formatFilterLabel`

**File:** `internal/ui/layout/border_test.go` (extend)

```go
func TestFormatFilterLabel(t *testing.T) {
    cases := []struct {
        name   string
        query  string
        budget int
        want   string
    }{
        // Variant 1 — full unquoted form fits
        {"full at large budget", "rock", 20, "f(rock)"},
        {"full at exact 7-col budget (rock)", "rock", 7, "f(rock)"},
        {"full at exact 12-col budget (rocknroll)", "rocknroll", 12, "f(rocknroll)"},
        // Variant 2 — truncation. "rocknroll" trimmed: f(rocknro…) w11, …,
        // f(roc…) w7, f(ro…) w6, f(r…) w5.
        {"truncated to f(rocknro…) at 11-col budget", "rocknroll", 11, "f(rocknro…)"},
        {"truncated to f(roc…) at 7-col budget", "rocknroll", 7, "f(roc…)"},
        {"truncated to f(ro…) at 6-col budget", "rocknroll", 6, "f(ro…)"},
        {"truncated to f(r…) at 5-col budget", "rocknroll", 5, "f(r…)"},
        // "rock" can also truncate when budget < 7
        {"rock truncated to f(ro…) at 6-col budget", "rock", 6, "f(ro…)"},
        {"rock truncated to f(r…) at 5-col budget", "rock", 5, "f(r…)"},
        // Variant 3 — minimal indicator
        {"falls back to f(…) at 4-col budget", "rocknroll", 4, "f(…)"},
        // Single-rune query: variant 2 loop is skipped (i starts at 0, condition i>=1 false)
        {"single-rune query fits f(x) at 4-col budget", "x", 4, "f(x)"},
        {"single-rune query drops at 3-col budget (no truncation possible)", "x", 3, ""},
        // Variant 4 — drop
        {"too narrow drops to empty", "rocknroll", 3, ""},
        {"empty query returns empty", "", 100, ""},
        {"zero budget returns empty", "rock", 0, ""},
        {"negative budget returns empty", "rock", -1, ""},
        // Wide-rune (CJK) queries: lipgloss.Width counts each as 2 columns;
        // formatFilterLabel relies on that for correct fit calculations.
        // Example: "字" → "f(字)" width 5. At budget 5 → returns "f(字)".
        // At budget 4 → variant 1 fails (5>4), variant 2 loop skipped
        // (1 rune), variant 3 "f(…)" width 4 → returns "f(…)".
        {"wide-rune fits f(字) at 5-col budget", "字", 5, "f(字)"},
        {"wide-rune falls back to f(…) at 4-col budget", "字", 4, "f(…)"},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := formatFilterLabel(tc.query, tc.budget)
            assert.Equal(t, tc.want, got)
            if got != "" {
                assert.LessOrEqual(t, lipgloss.Width(got), tc.budget,
                    "label width must fit within budget")
            }
        })
    }
}

func TestRenderPaneBorder_NarrowPane_FilterShrinksGracefully(t *testing.T) {
    cfg := layout.BorderConfig{
        Width: 30, Height: 5,
        Title: "Queue", ToggleKey: 2,
        AccentColor: lipgloss.Color("#888"),
        FilterQuery: "rock",
        Theme:       theme.Load("black"),
    }
    out := stripANSI(layout.RenderPaneBorder("", cfg))

    // Positive: filter indicator must be present in some form.
    assert.Contains(t, out, "f(", "narrow pane must still show filter indicator")

    // Negative: the close-notch is fully retired in filter mode. Verify by
    // asserting that no segment of the rendered output contains the literal
    // close-notch sequence "Esc close" (case-sensitive). The action-mode
    // notch with "filter" is also absent because cfg.FilterQuery is non-empty.
    assert.NotContains(t, out, "Esc close",
        "filter mode must not render the ╮ Esc close ╭ notch (Esc is global, see help overlay)")
}

func TestRenderPaneBorder_FilterModeRendersOnlyLabel(t *testing.T) {
    // Wide pane so f("rock") fits comfortably; assert the right segment is
    // exactly the label, no notch, no separator dash bar before "╮" except
    // the top-border fill dashes.
    cfg := layout.BorderConfig{
        Width: 80, Height: 5,
        Title: "Queue", ToggleKey: 2,
        AccentColor: lipgloss.Color("#888"),
        FilterQuery: "rock",
        Theme:       theme.Load("black"),
    }
    topLine := strings.Split(stripANSI(layout.RenderPaneBorder("", cfg)), "\n")[0]
    assert.Contains(t, topLine, "f(rock)")
    assert.NotContains(t, topLine, "Esc close")
    // The top-right notch corners are part of the border itself (╭...╮); the
    // CLOSE-NOTCH was an inner ╮...╭ pair around "Esc close". Confirm we have
    // exactly one ╮ on the line (the top-right border corner).
    assert.Equal(t, 1, strings.Count(topLine, "╮"),
        "filter mode should have exactly one ╮ (the top-right border corner)")
}
```

### Layout tests — flat Page B + redistribution

**File:** `internal/ui/layout/layout_test.go` (extend)

```go
func TestRecompute_PageBFlat_ThreeRows(t *testing.T) {
    m := layout.NewManager()
    m.Resize(200, 50)
    m.TogglePage()

    np := m.PaneRect(layout.PaneNowPlaying)
    h  := m.PaneRect(layout.PaneGatewayHealth)
    pt := m.PaneRect(layout.PanePollingTraffic)
    gl := m.PaneRect(layout.PaneGatewayLive)
    nl := m.PaneRect(layout.PaneNetworkLog)

    // Three rows: NowPlaying / [Health Traffic Live] / NetworkLog
    assert.Equal(t, 0, np.X)
    assert.Equal(t, 200, np.Width)
    assert.Equal(t, np.Y+np.Height, h.Y, "row 2 starts where row 1 ends")
    assert.Equal(t, h.Y, pt.Y, "Health and Traffic share row")
    assert.Equal(t, h.Y, gl.Y, "Live shares row with Health/Traffic")
    assert.Equal(t, h.Y+h.Height, nl.Y, "NetworkLog starts where row 2 ends")

    // 1:1:3 width split
    assert.Equal(t, h.Width, pt.Width, "Health and Traffic equal width")
    assert.InDelta(t, 3*h.Width, gl.Width, 1, "Live ≈ 3× Health (rounding ±1)")
}

func TestTogglePane_PageB_HealthHidden_TrafficLiveExpand(t *testing.T) {
    m := layout.NewManager()
    m.Resize(200, 50)
    m.TogglePage()
    pre := m.PaneRect(layout.PanePollingTraffic).Width

    m.TogglePane(layout.PaneGatewayHealth)
    assert.False(t, m.IsPaneVisible(layout.PaneGatewayHealth))

    pt := m.PaneRect(layout.PanePollingTraffic)
    gl := m.PaneRect(layout.PaneGatewayLive)
    assert.Equal(t, 0, pt.X, "Traffic now starts at x=0")
    assert.Greater(t, pt.Width, pre, "Traffic must absorb Health's column")
    assert.Equal(t, pt.Y, gl.Y)
    // Combined widths fill the row
    assert.Equal(t, 200, pt.Width+gl.Width)
}

func TestTogglePane_PageB_HealthAndTrafficHidden_LiveFullRow(t *testing.T) {
    m := layout.NewManager()
    m.Resize(200, 50)
    m.TogglePage()
    m.TogglePane(layout.PaneGatewayHealth)
    m.TogglePane(layout.PanePollingTraffic)

    gl := m.PaneRect(layout.PaneGatewayLive)
    assert.Equal(t, 0, gl.X)
    assert.Equal(t, 200, gl.Width, "Live fills full row when both siblings hidden")
}

func TestTogglePane_PageB_LiveHidden_HealthTrafficExpand(t *testing.T) {
    m := layout.NewManager()
    m.Resize(200, 50)
    m.TogglePage()
    m.TogglePane(layout.PaneGatewayLive)

    h  := m.PaneRect(layout.PaneGatewayHealth)
    pt := m.PaneRect(layout.PanePollingTraffic)
    assert.Equal(t, 0, h.X)
    assert.Equal(t, h.Width, pt.Width, "Health and Traffic split equally")
    assert.Equal(t, 200, h.Width+pt.Width, "they fill the full row")
}
```

**Delete** the following layout tests (no longer applicable):

- `TestRecompute_RowSpan_GatewayLive` in `layout_test.go`
- `TestPresetNerdStatus_GridHasFourRows` in `presets_test.go`
- Any other test that asserts spanner geometry, RowSpan field presence, or
  three-field Cell literal access.

**Add** to `presets_test.go`:

```go
func TestPresetNerdStatus_FlatThreeRows(t *testing.T) {
    require.Len(t, layout.PresetNerdStatus.Grid, 3)
    require.Len(t, layout.PresetNerdStatus.Grid[1].Cells, 3)

    middle := layout.PresetNerdStatus.Grid[1]
    assert.Equal(t, layout.PaneGatewayHealth,  middle.Cells[0].PaneID)
    assert.Equal(t, layout.PanePollingTraffic, middle.Cells[1].PaneID)
    assert.Equal(t, layout.PaneGatewayLive,    middle.Cells[2].PaneID)
    assert.Equal(t, 1, middle.Cells[0].WidthWeight)
    assert.Equal(t, 1, middle.Cells[1].WidthWeight)
    assert.Equal(t, 3, middle.Cells[2].WidthWeight)
}
```

### Routing tests — protect filter input from focus shifts and key toggles

**File:** `internal/app/routing_test.go` (extend)

```go
func TestRouting_FilterActive_TabDoesNotRotateFocus(t *testing.T) {
    a := newTestApp(t)
    a.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
    a.Layout().SetFocus(layout.PaneQueue)
    require.Equal(t, layout.PaneQueue, a.Layout().FocusedPane())

    // Open filter on Queue
    a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
    require.True(t, a.QueuePane().HasActiveFilter())

    // Press Tab — must NOT rotate focus
    a.Update(tea.KeyMsg{Type: tea.KeyTab})
    assert.Equal(t, layout.PaneQueue, a.Layout().FocusedPane(),
        "Tab must not rotate focus while a filter is active")
    assert.True(t, a.QueuePane().HasActiveFilter(),
        "Tab must not deactivate the filter")
}

func TestRouting_FilterActive_NumberKeysDoNotToggle(t *testing.T) {
    a := newTestApp(t)
    a.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
    a.Layout().SetFocus(layout.PaneQueue)
    a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
    require.True(t, a.QueuePane().HasActiveFilter())

    // Press '2' — must be consumed by the filter input, not toggle a pane
    a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
    assert.Equal(t, "2", a.QueuePane().Filter().Query())
    assert.True(t, a.Layout().IsPaneVisible(layout.PaneNowPlaying),
        "number key while filter active must not toggle a pane")
}
```

> An equivalent test already exists for the pre-refactor code
> (`TestFilterActive_NumberKeys_DoNotTogglePanes` in `routing_test.go`).
> Verify it still passes after migration; add the Tab variant if missing.

### Pane tests — keep existing, verify after migration

The existing per-pane tests (`_ActiveFilterQuery_ReturnsCommittedQuery`,
`_Esc_ClearsCommittedFilter`, `_Esc_ResetsScrollToPage1`,
`_Esc_ResetsScrollInMainListView`) remain valuable — they verify behaviour from
the user-facing pane API, not the internal routing. After migration each pane
delegates routing to the base, but all existing tests must still pass.

For `GatewayLivePane` specifically, the existing test
`TestGatewayLivePane_FilterEnter_AppliesQuery` likely asserts the
"Enter-to-apply" behaviour. **It must be rewritten** to assert the new
live-filter behaviour:

```go
func TestGatewayLivePane_LiveFilter_NarrowsRowsOnKeystroke(t *testing.T) {
    s := newStoreWithEvents(t /* ... */)
    p := panes.NewGatewayLivePane(s, theme.Load("black"))
    p.SetSize(80, 20)
    p.SetFocused(true)

    // emit some events to populate the buffer
    p.Update(panes.TickMsg{})
    require.Greater(t, p.BufferedEventCount(), 0)

    // Activate filter and type "rate"
    p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
    for _, r := range "rate" {
        p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
    }
    // Border label updates live (ActiveFilterQuery returns the live query)
    assert.Equal(t, "rate", p.ActiveFilterQuery())
    // Rows have already narrowed (live filter)
    // ... assert via a helper that inspects buildTableRows output
}
```

### Manual smoke tests (post-implementation)

These are not automated — run them once before merging:

1. Run `make run`. Resize the terminal to 200 columns × 50 rows
   (`stty cols 200 rows 50` from a separate session, or drag the window). Press
   `0` to switch to Page B. Verify three-row layout: NowPlaying strip /
   Health · Traffic · Live (1:1:3 width) / NetworkLog strip.
2. Press `2`. Health hides; Traffic moves to x=0, Live keeps right edge, both
   absorb Health's column.
3. Press `2` again. Health returns. Press `3`. Traffic hides; Health and Live
   absorb its column.
4. Press `2` and `3` together. Both hide. Live fills the full middle row.
5. Press `4`. Live hides. Health and Traffic each take 50% of the row.
6. Press `f` on Queue (Page A). Type "jazz". Border shows `f(jazz)` (unquoted,
   no `╮ Esc close ╭` notch). Rows narrow live as you type.
7. Press `Enter`. Filter input closes; border continues to render `f(jazz)`.
   Rows remain narrowed. Table regains focus.
8. Resize the terminal narrower (drag width down). Border label progressively
   shrinks: `f(jazz)` → `f(jaz…)` → `f(ja…)` → `f(j…)` → `f(…)` → drops. No
   flicker, no crash.
9. Resize back wider. Label expands back to the widest variant that fits.
10. Press `Esc`. Border label clears, all rows return.
11. Press `Esc` again. Table scrolls to page 1 (no other state to clear).
12. Repeat steps 6-11 on Page B's GatewayLive — verify identical live-filter
    behaviour (was previously Enter-to-apply).
13. **Tab interception while filter active.** On Queue, press `f`, then press
    `Tab`. Verify Tab is inserted into the filter input as a literal tab
    character (textinput's default behaviour) and **does not** rotate focus to
    the next pane. Press `Esc` to cancel; press `Tab` — focus rotates as
    normal. This verifies `routing.go`'s `HasActiveFilter()` guard via
    `TableBasedPane.HasActiveFilter()` after migration.
14. **Number key interception while filter active.** On Queue, press `f`, then
    press `2`. Verify `2` appears in the filter input — it must not be
    consumed as a pane toggle. (On Page B, repeat with key `3` to confirm the
    Page B routing also defers to `HasActiveFilter()`.)

---

## Code to Delete

### Layout (engine and presets)

- `internal/ui/layout/pane.go`:
  - `Cell.RowSpan int` field (line 9–10)
  - `func (c Cell) rowSpan() int` helper (lines 13–18)
  - **Update (not delete):** the `FilterQueryPane` interface doc comment
    (lines 100-105) currently reads "exposes a committed filter query for
    display in the pane border. ... shows filtering: \"query\" in the
    top-right corner." Change to: "exposes the current filter query (live
    while typing, preserved after Enter) for display in the pane border. When
    `ActiveFilterQuery()` returns a non-empty string, the border renderer shows
    `f(query)` in the top-right corner, with progressive shrink for narrow
    panes. See `formatFilterLabel` in `border.go`."

- `internal/ui/layout/border.go`:
  - **Update (not delete):** `BorderConfig.FilterQuery` doc comment (line 35)
    currently reads "FilterQuery is non-empty when filter mode is active. When
    set, replaces the action shortcuts with: filtering: \"query\" ─╮ Esc close ╭".
    Change to: "FilterQuery is the live filter query (updates on every
    keystroke; preserved after Enter). When non-empty, the right segment
    renders `f(query)` via `formatFilterLabel`, replacing any action shortcuts
    in the same segment."

- `internal/ui/layout/layout.go`:
  - The entire spanning pass (Step 3, lines ~188–274)
  - The reserved-interval logic in Step 4 (lines ~276–384) — replaced by simple
    width-by-weight loop
  - All `spanner*`, `rowIdxByOrig`, `spannerCoverageByRow`, `cellSpec`,
    `liveRow.spannerCoverage`, `rowLayout.spannerCoverage` structures

- `internal/ui/layout/presets.go`:
  - `RowSpan: 2` line in `PresetNerdStatus.Grid[1].Cells[1]`
  - The `// GatewayLive continuation — no cell here; recompute() handles the span`
    comment and its enclosing third row

- `internal/ui/layout/layout_test.go`:
  - `TestRecompute_RowSpan_GatewayLive`
  - Any other test referencing `RowSpan` field or spanner geometry

- `internal/ui/layout/presets_test.go`:
  - `TestPresetNerdStatus_GridHasFourRows`

### Border (file)

- `internal/ui/layout/border.go`:
  - The hardcoded `filtering: "..."` string format in `buildRightSegment`
    (replaced by `formatFilterLabel` helper that produces `f(query)`)
  - The close-notch construction in filter mode
    (`borderChar(trCorner) + " " + keyHintStyle("Esc") + " " + mutedStyle("close") + " " + borderChar(tlCorner)`) —
    deleted entirely. Filter mode renders the label only.
  - The all-or-nothing `dashCount < 0` fallback that drops the entire right
    segment (replaced by graded shrink computed via budget)

### Panes (filter routing and Esc-close action)

- `internal/ui/panes/gateway_live_pane.go`:
  - `activeQuery string` field (line 46)
  - All `p.activeQuery` reads/writes in `handleKey`, `buildTableRows`, etc.
  - The bespoke Enter/Esc/`f` handling in `handleKey` (lines ~161–215) —
    replaced by `HandleFilterKey` call

- All nine filterable panes (`queue.go`, `playlists_pane.go`, `albums_pane.go`,
  `likedsongs_pane.go`, `recentlyplayed_pane.go`, `toptracks_pane.go`,
  `topartists_pane.go`, `networklog_pane.go`, `gateway_live_pane.go`):
  - The duplicated filter routing block (`if p.filter.IsActive() { ... }`,
    `if msg == 'f' { ... }`, `if Esc { ... }`) — replaced by single
    `HandleFilterKey` call
  - The `if filter.IsActive() return [{Esc, close}]` branch inside each pane's
    `Actions()` method — deleted. `TableBasedPane.Actions()` provides the
    default `[{f, filter}]`. Composite panes (`AlbumsPane`, `PlaylistsPane`)
    compose their slice via `BaseFilterAction()`.
  - For simple panes (Queue, LikedSongs, TopTracks, TopArtists, RecentlyPlayed,
    NetworkLog, GatewayLive): the entire pane-level `Actions()` method is
    deleted — base default is sufficient.
  - Per-pane `func (p *X) ActiveFilterQuery() string` — inherited from base
  - Per-pane `func (p *X) HasActiveFilter() bool` — inherited from base
  - Per-pane `var _ layout.FilterQueryPane = &X{}` interface assertions —
    **kept** (cheap compile-time guard against accidental method-name shadowing
    on the concrete pane). Add the same assertion on `TableBasedPane` itself
    so a misconfigured base also fails at compile time.

Estimated total deletion: ~280 LOC layout (RowSpan engine) + ~25 LOC × 9 panes
filter routing + ~30 LOC GatewayLive `activeQuery` plumbing + ~10 LOC × 9
panes deleted/composed `Actions()` ≈ **~625 LOC removed**.
Estimated additions: ~140 LOC `TableBasedPane` (including default `Actions()`
and `BaseFilterAction()`) + ~50 LOC `formatFilterLabel` + ~70 LOC simplified
`recompute` + ~10 LOC focus-invariant guards in `TogglePage`/`CyclePreset` +
~220 LOC new tests (table-based-pane + border + layout + routing + invariant)
≈ **~490 LOC added**.
**Net: ~135 LOC smaller** with one source of truth for filter routing.

---

## Implementation Approach

The work is sequenced so each step compiles and passes tests before the next
begins. Commit per step.

### Step 1 — `formatFilterLabel` helper + tests (border)

Pure function, no callers yet. Add the helper to `border.go`, add table-driven
tests, run them. This isolates the label-shrink logic before it's wired up.

### Step 2 — Wire `formatFilterLabel` into `buildRightSegment`

Refactor `buildRightSegment` to take a `budget int` parameter. Refactor
`RenderPaneBorder` to compute the budget and pass it. Update existing border
tests that may have asserted the old `filtering: "..."` format. Add the
narrow-pane integration test. `make ci` must still pass — Page A panes should
look identical except for the filter label format change (which currently is
not exercised by any Page A test fixture).

### Step 3 — `TableBasedPane` skeleton + tests (no migration yet)

Create `table_based_pane.go` with the type, constructor, accessors, and
`HandleFilterKey`. Add the unit tests using a fake pane scaffolded just for
this purpose. `make ci` passes — base is unused but compiles.

### Step 4 — Migrate panes one at a time

Order from simplest to most complex:

1. `NetworkLogPane` (simple, no Enter, no list view)
2. `RecentlyPlayedPane`
3. `TopTracksPane`
4. `TopArtistsPane`
5. `LikedSongsPane`
6. `QueuePane` (has Enter→play)
7. `AlbumsPane` (has list-view sub-mode)
8. `PlaylistsPane` (has list-view sub-mode)
9. `GatewayLivePane` (has Tick handling and `activeQuery` field deletion)

For each pane: replace embed, replace Update routing, delete inherited methods,
run pane tests. Commit per pane. Run `make ci` after each to confirm no
regression.

### Step 5 — Layout: delete `RowSpan` and simplify `recompute`

Order:
1. Rewrite `PresetNerdStatus` to the flat 3-row layout (drops the `RowSpan: 2`
   literal as part of the rewrite; the third row that previously held the
   spanner continuation is removed entirely).
2. Delete `Cell.RowSpan` field and `rowSpan()` helper from `pane.go` — compile
   error sweep across `layout.go` and `presets.go` (only `recompute` references
   the field).
3. Replace `recompute` with the flat two-loop implementation.
4. Update `FilterQueryPane` interface doc comment in `pane.go` and
   `BorderConfig.FilterQuery` doc comment in `border.go` to reflect the
   live-query semantics and `f(query)` border format (see Code to Delete →
   "Update (not delete)" entries).
5. Delete obsolete tests, add new flat-layout tests.
6. `make ci`.

### Step 6 — Final verification

- `make ci` green.
- Manual smoke test (the 9-step list above).
- Update `docs/spec/features/14-page-b-redesign/feature.md` post-implementation
  section with a note pointing to PR # for this story.

---

## Acceptance Criteria

### Filter UX (cross-pane)

- [ ] All nine filterable panes implement live-filter behaviour: typing a
      character narrows rows immediately; the border label updates immediately.
- [ ] `f` toggles filter input on every filterable pane via the same code path.
- [ ] `Esc` while filter is open cancels filter (clears query, closes input)
      on every pane.
- [ ] `Esc` while filter is closed and query is non-empty clears the committed
      query on every pane.
- [ ] `Esc` while filter is closed and query is empty resets table scroll to
      page 1 on every pane.
- [ ] `GatewayLivePane` no longer has an `activeQuery` field; its filter
      behaviour is identical to the other eight panes.

### Border (label rendering)

- [ ] Filter label uses `f(query)` form by default (no quotes around the query),
      not `filtering: "query"`.
- [ ] When the pane is too narrow for `f(query)`, the label shrinks
      progressively: `f(query)` → `f(qu…)` → `f(…)` → drops.
- [ ] When an adjacent pane is hidden and the current pane resizes wider, the
      label expands back to the widest variant that fits.
- [ ] The right segment in filter mode is **only** the label — no
      `╮ Esc close ╭` notch is rendered. `Esc` is documented as a global key in
      the help overlay (`?`) and is not repeated per-pane.
- [ ] The action-mode right segment (e.g. `╮ f filter ╭` when no filter is
      active) is unchanged — only the filter-mode segment dropped its notch.

### Layout

- [ ] `Cell.RowSpan` field is removed; `Cell` is two-field `{PaneID, WidthWeight}`.
- [ ] `recompute` is the simple flat-row implementation; the spanning pass and
      reserved-interval logic are deleted.
- [ ] `PresetNerdStatus` has three rows: NowPlaying / [Health Traffic Live with
      weights 1:1:3] / NetworkLog.
- [ ] Hiding `GatewayHealth` causes `PollingTraffic` and `GatewayLive` to absorb
      its column proportionally.
- [ ] Hiding both `GatewayHealth` and `PollingTraffic` causes `GatewayLive` to
      fill the full middle row.
- [ ] Hiding `GatewayLive` causes `GatewayHealth` and `PollingTraffic` to split
      the row 1:1.
- [ ] All Page A presets and behaviour are unchanged.

### Code health

- [ ] `TableBasedPane` exists in `internal/ui/panes/table_based_pane.go`.
- [ ] No filterable pane contains the `filter.IsActive() / 'f' / Esc` routing
      block — each pane delegates to `HandleFilterKey`.
- [ ] No filterable pane declares its own `ActiveFilterQuery` or
      `HasActiveFilter` — both are inherited from `TableBasedPane`.
- [ ] `var _ layout.FilterablePane = &TableBasedPane{}` and
      `var _ layout.FilterQueryPane = &TableBasedPane{}` compile-time
      assertions exist on `TableBasedPane`. Per-pane assertions are retained
      (cheap shadow guard).
- [ ] `FilterQueryPane` interface and `BorderConfig.FilterQuery` doc comments
      reflect live-query semantics and `f(query)` border format.
- [ ] Focus invariant — no code path moves focus away from a pane with an
      active filter input. Enforced in `TogglePage` and `CyclePreset` and
      verified by tests.
- [ ] Total LOC deletion ≥ total LOC additions (rough sanity — measure with
      `git diff --shortstat`).

### Tests (coverage)

- [ ] All new component tests for `TableBasedPane` pass — including
      `TestTableBasedPane_Actions_NotNilWhenFilterActiveWithEmptyQuery` (pins
      the design decision against future "fix me" temptation).
- [ ] All new border tests for `formatFilterLabel` and graded shrink pass —
      including `TestRenderPaneBorder_FilterModeRendersOnlyLabel` which
      asserts no close-notch is present.
- [ ] All new layout tests for flat Page B and toggle redistribution pass.
- [ ] All new routing tests pass (Tab + number-key interception while filter
      active).
- [ ] All new focus-invariant tests pass (`TogglePage`/`CyclePreset` deactivate
      any active filter).
- [ ] All existing per-pane filter and Esc tests pass without modification
      (except `GatewayLivePane` Enter-to-apply test, which is rewritten for
      live behaviour).
- [ ] Any test that asserted the `╮ Esc close ╭` notch in filter mode is
      updated to assert the label-only right segment.
- [ ] `make ci` passes.

---

## Risks and Rollback

### Risks

1. **Cell struct literal sweep** — the existing `presets.go` already uses
   named-field form (`{PaneID: x, WidthWeight: y}`), so removing `RowSpan` is a
   one-line delete in `PresetNerdStatus`. No mass update is required. Risk is low.

2. **GatewayLive behaviour change is user-visible** — previously `Enter`
   committed the filter; after this story `Enter` only closes the input
   (query was already applied live). Document in commit message and feature.md
   post-implementation note. Risk is acceptable; the user has confirmed this
   alignment is desired.

3. **Border graded shrink may differ subtly from current output at certain
   widths** — pin behaviour with table-driven tests at the boundaries (budgets
   4, 5, 6, 7, 8, 9, 100). Risk is low if test coverage is adequate.

4. **SetTheme rebuild ordering** — each pane's `SetTheme` rebuilds table+filter
   independently. After migration, the rebuild flow uses
   `p.SwapTableAndFilter(newT, newF)`. Verify each migrated pane's SetTheme
   test still passes. Risk is low.

5. **Focus rotation order** — with RowSpan deleted, focus order is purely
   top-to-bottom, left-to-right. The current code's focus order with RowSpan
   was hand-tuned in story 180 to put spanners after their row siblings. After
   deletion this is moot. Verify Tab traversal on Page B matches the user's
   visual expectation (Health → Traffic → Live → NetworkLog).

### Rollback

Each step is a self-contained commit. Rollback is `git revert <commit>`. The
most invasive commits are step 5 (RowSpan deletion) and step 4-pane-9
(GatewayLive migration). Either can be reverted independently — they have no
hard ordering dependency on each other.

If rollback is needed mid-merge, prefer reverting in reverse order of the
implementation sequence (step 6 → 5 → 4-pane-by-pane → 3 → 2 → 1).

---

## Tasks

- [ ] Step 1 — Add `formatFilterLabel(query, budget)` helper + table-driven
      tests in `internal/ui/layout/border.go` and `border_test.go`
- [ ] Step 2 — Wire `formatFilterLabel` into `buildRightSegment` (add `budget`
      parameter); delete the close-notch construction in filter mode; update
      `RenderPaneBorder` to compute budget; add narrow-pane integration test;
      update any existing tests that asserted `filtering: "..."` format or the
      `╮ Esc close ╭` notch
- [ ] Step 3 — Create `internal/ui/panes/table_based_pane.go` with
      `TableBasedPane`, constructor, accessors, `SwapTableAndFilter`,
      `HasActiveFilter`, `ActiveFilterQuery`, `HandleFilterKey`; add
      `table_based_pane_test.go` with the unit tests listed above
- [ ] Step 4a — Migrate `NetworkLogPane`
- [ ] Step 4b — Migrate `RecentlyPlayedPane`
- [ ] Step 4c — Migrate `TopTracksPane`
- [ ] Step 4d — Migrate `TopArtistsPane`
- [ ] Step 4e — Migrate `LikedSongsPane`
- [ ] Step 4f — Migrate `QueuePane`
- [ ] Step 4g — Migrate `AlbumsPane`
- [ ] Step 4h — Migrate `PlaylistsPane`
- [ ] Step 4i — Migrate `GatewayLivePane`; delete `activeQuery` field; rewrite
      `TestGatewayLivePane_FilterEnter_AppliesQuery` as
      `TestGatewayLivePane_LiveFilter_NarrowsRowsOnKeystroke`
- [ ] Step 4j — Add default `Actions()` method on `TableBasedPane` returning
      `[{Key: "f", Label: "filter"}]` plus a `BaseFilterAction()` helper. In
      simple panes (Queue, LikedSongs, TopTracks, TopArtists, RecentlyPlayed,
      NetworkLog, GatewayLive) delete the entire pane-level `Actions()` method.
      In composite panes (Albums, Playlists) drop the `if filter.IsActive()`
      branch and compose the slice via `BaseFilterAction()`. Update or remove
      tests asserting the old `Esc close` action; existing
      action-shortcut-rendering tests for `f filter` continue to pass.
- [ ] Step 5a — Update `PresetNerdStatus` to flat three-row 1:1:3 layout
- [ ] Step 5b — Delete `Cell.RowSpan` field and `rowSpan()` helper from
      `internal/ui/layout/pane.go`; sweep compile errors
- [ ] Step 5c — Replace `recompute()` body with the flat two-loop implementation
- [ ] Step 5d — Delete obsolete layout tests
      (`TestRecompute_RowSpan_GatewayLive`,
      `TestPresetNerdStatus_GridHasFourRows`); add new tests
      (`TestRecompute_PageBFlat_ThreeRows`,
      `TestTogglePane_PageB_HealthHidden_TrafficLiveExpand`,
      `TestTogglePane_PageB_HealthAndTrafficHidden_LiveFullRow`,
      `TestTogglePane_PageB_LiveHidden_HealthTrafficExpand`,
      `TestPresetNerdStatus_FlatThreeRows`)
- [ ] Step 5e — Update doc comments on `layout.FilterQueryPane` interface and
      `layout.BorderConfig.FilterQuery` field to reflect live-query semantics
      and `f(query)` border format
- [ ] Step 5f — Add focus-invariant enforcement in `TogglePage` and
      `CyclePreset`: deactivate any active pane filter before recompute. Add
      `TestQueuePane_FilterActive_PageToggleDeactivatesFilter` and a parallel
      `_PresetCycleDeactivatesFilter` test
- [ ] Step 5g — Extend `internal/app/routing_test.go` with
      `TestRouting_FilterActive_TabDoesNotRotateFocus` and confirm
      `TestFilterActive_NumberKeys_DoNotTogglePanes` still passes after migration
- [ ] Step 6 — `make ci` green; manual smoke test (9-step list); update
      `docs/spec/features/14-page-b-redesign/feature.md` post-implementation
      section with PR reference
