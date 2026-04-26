# Page B Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign Page B (Nerd Status) into a pane-based keyboard-driven layout consistent with Page A; establish universal scroll/filter/Esc reset behavior across the whole app.

**Architecture:** Universal changes (GotoTop, Esc reset, keybindings) land first as a standalone commit group. Then the Page B panes replace RequestFlowPane with three focused panes (GatewayHealth, PollingTraffic, GatewayLive) while NetworkLog is fixed in-place. RequestFlow files are deleted only after all replacements are wired.

**Tech Stack:** Go 1.22+, Bubble Tea, Lip Gloss, evertras/bubble-table v0.19.2, uikit.ListRow, uikit.PaneChrome

**Spec:** `docs/superpowers/specs/2026-04-26-page-b-redesign-design.md`

---

## Part 1 — Universal Cross-Cutting Changes

### Task 1: Add GotoTop() and CurrentPage() to components.Table

**Files:**
- Modify: `internal/ui/components/table.go`
- Modify: `internal/ui/components/table_test.go`

- [ ] **Step 1: Write the failing test**

  In `table_test.go`, add a new test (after existing tests):

  ```go
  func TestTable_GotoTop_ResetsToFirstPage(t *testing.T) {
      th := theme.Load("black")
      cols := []ColumnDef{
          {Key: "k", Header: "K", FlexFactor: 1, Color: th.ColumnPrimary()},
      }
      tbl := NewTable(TableConfig{Columns: cols, Theme: th, PlayingIndex: -1, ShowHeader: true})
      tbl.SetSize(80, 5) // pageSize = 5-6 = -1 → clamped to 1; forces many pages

      rows := make([]map[string]string, 20)
      for i := range rows {
          rows[i] = map[string]string{"k": fmt.Sprintf("row-%d", i)}
      }
      tbl.SetRows(rows)
      assert.Equal(t, 1, tbl.CurrentPage(), "should start on page 1")

      // Navigate down 8 times to move past page 1.
      for range 8 {
          tbl.Update(tea.KeyMsg{Type: tea.KeyDown})
      }
      assert.Greater(t, tbl.CurrentPage(), 1, "should have scrolled past page 1")

      tbl.GotoTop()
      assert.Equal(t, 1, tbl.CurrentPage(), "GotoTop must reset to page 1")
  }
  ```

  Add `"fmt"` to imports if not present.

- [ ] **Step 2: Run test to verify it fails**

  ```bash
  go test ./internal/ui/components/... -run TestTable_GotoTop -v
  ```

  Expected: compile error — `tbl.CurrentPage undefined` and `tbl.GotoTop undefined`.

- [ ] **Step 3: Implement GotoTop() and CurrentPage() in table.go**

  Append after the `View()` method (line 231):

  ```go
  // GotoTop resets the table scroll position to the first page and first row.
  // Called by panes when the user presses Esc with no active filter.
  func (t *Table) GotoTop() {
      t.inner = t.inner.PageFirst()
  }

  // CurrentPage returns the 1-based current page number.
  // Used by tests to verify scroll-reset behaviour.
  func (t *Table) CurrentPage() int {
      return (&t.inner).CurrentPage()
  }
  ```

- [ ] **Step 4: Run tests to verify pass**

  ```bash
  go test ./internal/ui/components/... -v
  ```

  Expected: all green, including `TestTable_GotoTop_ResetsToFirstPage`.

- [ ] **Step 5: Commit**

  ```bash
  git add internal/ui/components/table.go internal/ui/components/table_test.go
  git commit -m "feat(uikit): add Table.GotoTop() and Table.CurrentPage() for Esc scroll-reset"
  ```

---

### Task 2: Add Esc scroll-reset to simple panes (Queue, TopTracks, TopArtists, LikedSongs, RecentlyPlayed, NetworkLog)

**Files:**
- Modify: `internal/ui/panes/queue.go`
- Modify: `internal/ui/panes/toptracks_pane.go`
- Modify: `internal/ui/panes/topartists_pane.go`
- Modify: `internal/ui/panes/likedsongs_pane.go`
- Modify: `internal/ui/panes/recentlyplayed_pane.go`
- Modify: `internal/ui/panes/networklog_pane.go`
- Modify: all six corresponding `_test.go` files

**Pattern:** In each pane's key-routing path (after the filter-active branch exits, before the `table.Update(keyMsg)` fallthrough), add:

```go
case keyMsg.Type == tea.KeyEscape:
    p.table.GotoTop()
    return p, nil
```

When filter IS active, Esc already works: `filter.Update(Esc)` deactivates the filter and the `if !p.filter.IsActive()` block re-focuses the table. The new Esc handling fires only when the filter is NOT active.

- [ ] **Step 1: Write failing tests — one per pane**

  Add to each pane's test file (use actual helper from that file; all follow `newTestXxxPane` pattern):

  **queue_test.go** — `TestQueuePane_Esc_ResetsScrollToPage1`:
  ```go
  func TestQueuePane_Esc_ResetsScrollToPage1(t *testing.T) {
      q := newTestQueuePane(t)
      q.SetSize(80, 5)
      q.SetFocused(true)
      rows := make([]domain.Track, 20)
      for i := range rows {
          rows[i] = domain.Track{Name: fmt.Sprintf("Track %d", i)}
      }
      // Inject tracks via store
      store := state.New()
      store.SetQueue(rows)
      q2 := panes.NewQueuePane(store, theme.Load("black"), false)
      q2.SetSize(80, 5)
      q2.SetFocused(true)
      for range 8 {
          q2.Update(tea.KeyMsg{Type: tea.KeyDown})
      }
      require.Greater(t, q2.TableCurrentPage(), 1, "need page > 1 to test reset")
      q2.Update(tea.KeyMsg{Type: tea.KeyEsc})
      assert.Equal(t, 1, q2.TableCurrentPage())
  }
  ```

  > **Note:** The test requires a `TableCurrentPage() int` accessor on each pane (see Step 3 below). This is the white-box test helper pattern from PANE-TEMPLATE.md.

  Add the same `_Esc_ResetsScrollToPage1` test to:
  - `toptracks_pane_test.go`
  - `topartists_pane_test.go`
  - `likedsongs_pane_test.go`
  - `recentlyplayed_pane_test.go`
  - `networklog_pane_test.go`

  Each test helper varies by the pane constructor and how rows are injected (via store or direct). Follow the pattern of the nearest existing test in each file.

- [ ] **Step 2: Run tests to verify they fail**

  ```bash
  go test ./internal/ui/panes/... -run "Esc_Resets" -v
  ```

  Expected: compile error — `TableCurrentPage undefined` on each pane type.

- [ ] **Step 3: Implement Esc handling and TableCurrentPage() accessor on each pane**

  **queue.go** — In `handleKey()`, after `if q.filter.IsActive()` block (around line 122), before `cmd := q.table.Update(msg)`:

  ```go
  // Esc with no active filter resets scroll to the top.
  if msg.Type == tea.KeyEscape {
      q.table.GotoTop()
      return q, nil
  }
  ```

  And add accessor at the bottom:
  ```go
  // TableCurrentPage returns the table's current page number. Exported for testing.
  func (q *QueuePane) TableCurrentPage() int { return q.table.CurrentPage() }
  ```

  **toptracks_pane.go** — In `Update()` switch, after the filter-active block exits, add new case in the `switch` block before `// Forward navigation to the table.`:

  ```go
  case keyMsg.Type == tea.KeyEscape:
      a.table.GotoTop()
      return a, nil
  ```

  Add accessor:
  ```go
  func (a *TopTracksPane) TableCurrentPage() int { return a.table.CurrentPage() }
  ```

  **topartists_pane.go** — Same as TopTracksPane. In the `switch` on `keyMsg`:

  ```go
  case keyMsg.Type == tea.KeyEscape:
      a.table.GotoTop()
      return a, nil
  ```

  Add accessor:
  ```go
  func (a *TopArtistsPane) TableCurrentPage() int { return a.table.CurrentPage() }
  ```

  **likedsongs_pane.go** — After the `filter.IsActive()` block, before `cmd := l.table.Update(msg)`:

  ```go
  if msg.Type == tea.KeyEscape {
      l.table.GotoTop()
      return l, nil
  }
  ```

  Add accessor:
  ```go
  func (l *LikedSongsPane) TableCurrentPage() int { return l.table.CurrentPage() }
  ```

  **recentlyplayed_pane.go** — Same pattern as likedsongs_pane.go (check surrounding code and use `r` receiver):

  ```go
  if msg.Type == tea.KeyEscape {
      r.table.GotoTop()
      return r, nil
  }
  ```

  Add accessor:
  ```go
  func (r *RecentlyPlayedPane) TableCurrentPage() int { return r.table.CurrentPage() }
  ```

  **networklog_pane.go** — In `handleKey()`, after `if p.filter.IsActive()` block, before `cmd := p.table.Update(m)`:

  ```go
  if m.Type == tea.KeyEscape {
      p.table.GotoTop()
      return p, nil
  }
  ```

  Add accessor:
  ```go
  func (p *NetworkLogPane) TableCurrentPage() int { return p.table.CurrentPage() }
  ```

- [ ] **Step 4: Run tests to verify pass**

  ```bash
  go test ./internal/ui/panes/... -run "Esc_Resets" -v
  ```

  Expected: all 6 new tests pass.

- [ ] **Step 5: Run full panes test suite to catch regressions**

  ```bash
  go test ./internal/ui/panes/... -v
  ```

  Expected: all green.

- [ ] **Step 6: Commit**

  ```bash
  git add internal/ui/panes/queue.go internal/ui/panes/queue_test.go \
          internal/ui/panes/toptracks_pane.go internal/ui/panes/toptracks_pane_test.go \
          internal/ui/panes/topartists_pane.go internal/ui/panes/topartists_pane_test.go \
          internal/ui/panes/likedsongs_pane.go internal/ui/panes/likedsongs_pane_test.go \
          internal/ui/panes/recentlyplayed_pane.go internal/ui/panes/recentlyplayed_pane_test.go \
          internal/ui/panes/networklog_pane.go internal/ui/panes/networklog_pane_test.go
  git commit -m "feat(panes): Esc scroll-reset on all simple table panes"
  ```

---

### Task 3: Add Esc scroll-reset to Albums and Playlists panes

**Files:**
- Modify: `internal/ui/panes/albums_pane.go`
- Modify: `internal/ui/panes/albums_pane_test.go`
- Modify: `internal/ui/panes/playlists_pane.go`
- Modify: `internal/ui/panes/playlists_pane_test.go`

These panes have a track sub-view. Esc in sub-view already closes it (unchanged). Esc in the main list view with no active filter resets scroll.

- [ ] **Step 1: Write failing tests**

  **albums_pane_test.go** — `TestAlbumsPane_Esc_ResetsScrollInMainListView`:
  ```go
  func TestAlbumsPane_Esc_ResetsScrollInMainListView(t *testing.T) {
      store := state.New()
      albums := make([]domain.SavedAlbum, 20)
      for i := range albums {
          albums[i] = domain.SavedAlbum{Album: domain.Album{
              ID: fmt.Sprintf("id-%d", i), Name: fmt.Sprintf("Album %d", i),
          }}
      }
      store.SetAlbums(albums)
      p := panes.NewAlbumsPane(store, theme.Load("black"), false)
      p.SetSize(80, 5)
      p.SetFocused(true)
      for range 8 {
          p.Update(tea.KeyMsg{Type: tea.KeyDown})
      }
      require.Greater(t, p.TableCurrentPage(), 1)
      p.Update(tea.KeyMsg{Type: tea.KeyEsc})
      assert.Equal(t, 1, p.TableCurrentPage())
  }
  ```

  **playlists_pane_test.go** — `TestPlaylistsPane_Esc_ResetsScrollInMainListView`:
  ```go
  func TestPlaylistsPane_Esc_ResetsScrollInMainListView(t *testing.T) {
      store := state.New()
      lists := make([]domain.SimplePlaylist, 20)
      for i := range lists {
          lists[i] = domain.SimplePlaylist{
              ID: fmt.Sprintf("pl-%d", i), Name: fmt.Sprintf("Playlist %d", i),
          }
      }
      store.SetPlaylists(lists)
      p := panes.NewPlaylistsPane(store, theme.Load("black"), false)
      p.SetSize(80, 5)
      p.SetFocused(true)
      for range 8 {
          p.Update(tea.KeyMsg{Type: tea.KeyDown})
      }
      require.Greater(t, p.TableCurrentPage(), 1)
      p.Update(tea.KeyMsg{Type: tea.KeyEsc})
      assert.Equal(t, 1, p.TableCurrentPage())
  }
  ```

- [ ] **Step 2: Run tests to verify they fail**

  ```bash
  go test ./internal/ui/panes/... -run "Esc_ResetScroll" -v
  ```

  Expected: compile error — `TableCurrentPage undefined`.

- [ ] **Step 3: Implement Esc handling and accessors**

  **albums_pane.go** — In `handleListViewKey()`, add a new `case` before the fallthrough `cmd := a.table.Update(keyMsg)`:

  ```go
  case keyMsg.Type == tea.KeyEscape:
      a.table.GotoTop()
      return a, nil
  ```

  Add accessor:
  ```go
  // TableCurrentPage returns the album list table's current page. Exported for testing.
  func (a *AlbumsPane) TableCurrentPage() int { return a.table.CurrentPage() }
  ```

  **playlists_pane.go** — In `handleListViewKey()`, add before `cmd := p.table.Update(key)`:

  ```go
  case key.Type == tea.KeyEscape:
      p.table.GotoTop()
      return p, nil
  ```

  Add accessor:
  ```go
  // TableCurrentPage returns the playlist list table's current page. Exported for testing.
  func (p *PlaylistsPane) TableCurrentPage() int { return p.table.CurrentPage() }
  ```

- [ ] **Step 4: Run tests to verify pass**

  ```bash
  go test ./internal/ui/panes/... -run "Esc_Resets" -v
  ```

  Expected: all 8 Esc reset tests green.

- [ ] **Step 5: Full panes suite**

  ```bash
  go test ./internal/ui/panes/... -v
  ```

- [ ] **Step 6: Commit**

  ```bash
  git add internal/ui/panes/albums_pane.go internal/ui/panes/albums_pane_test.go \
          internal/ui/panes/playlists_pane.go internal/ui/panes/playlists_pane_test.go
  git commit -m "feat(panes): Esc scroll-reset on Albums and Playlists list views"
  ```

---

### Task 4: Update keybinding documentation — SAME COMMIT for all three locations

**Files:**
- Modify: `internal/ui/panes/help_overlay.go`
- Modify: `docs/keybinding.md`
- Modify: `docs/DESIGN.md`

> **CLAUDE.md rule 15:** All three locations must update in the same commit. Do not split.

- [ ] **Step 1: Update help_overlay.go `helpContent` Navigation section**

  Current Navigation section (left column):
  ```go
  {title: "Navigation", bindings: []helpBinding{
      {"Tab", "Next pane"}, {"Shift+Tab", "Prev pane"},
      {"Esc", "Close overlay"},
  }},
  ```

  Replace with:
  ```go
  {title: "Navigation", bindings: []helpBinding{
      {"Tab", "Next pane"}, {"Shift+Tab", "Prev pane"},
      {"↑ / k", "Scroll up"}, {"↓ / j", "Scroll down"},
      {"Esc", "Close overlay · clear filter · scroll top"},
  }},
  ```

- [ ] **Step 2: Update docs/keybinding.md Navigation table**

  Current:
  ```
  | `Tab` | Next pane focus |
  | `Shift+Tab` | Previous pane focus |
  | `Esc` | Close overlay or filter |
  ```

  Replace with:
  ```
  | `Tab` | Next pane focus |
  | `Shift+Tab` | Previous pane focus |
  | `↑` / `k` | Scroll up |
  | `↓` / `j` | Scroll down |
  | `Esc` | Close overlay · clear filter · scroll top |
  ```

- [ ] **Step 3: Update docs/DESIGN.md §17 keybinding table**

  Find the Navigation section in §17 and add the same two rows:
  ```
  | `↑` / `k` | Scroll up |
  | `↓` / `j` | Scroll down |
  ```
  Update the Esc row label to:
  ```
  | `Esc` | Close overlay · clear filter · scroll top |
  ```

- [ ] **Step 4: Build to confirm no compile errors**

  ```bash
  go build ./...
  ```

- [ ] **Step 5: Commit all three files together**

  ```bash
  git add internal/ui/panes/help_overlay.go docs/keybinding.md docs/DESIGN.md
  git commit -m "docs(keybindings): add scroll j/k and Esc reset to all three keybinding locations"
  ```

---

## Part 2 — Page B Layout Rebuild

### Task 5: Add new PaneID constants and update TogglePane guard

**Files:**
- Modify: `internal/ui/layout/pane.go`
- Modify: `internal/ui/layout/layout.go`
- Modify: `internal/ui/layout/layout_test.go`

- [ ] **Step 1: Write a failing test for the new PaneID values**

  In `layout_test.go`, add:

  ```go
  func TestPaneIDs_PageBConstants_AreDistinct(t *testing.T) {
      ids := []layout.PaneID{
          layout.PaneNetworkLog,
          layout.PaneGatewayHealth,
          layout.PanePollingTraffic,
          layout.PaneGatewayLive,
      }
      seen := make(map[layout.PaneID]bool)
      for _, id := range ids {
          require.False(t, seen[id], "duplicate PaneID: %d", id)
          seen[id] = true
      }
      // Page B panes must all be >= PaneNetworkLog so TogglePane rejects them.
      for _, id := range ids {
          assert.GreaterOrEqual(t, int(id), int(layout.PaneNetworkLog),
              "Page B pane %d must be >= PaneNetworkLog", id)
      }
  }
  ```

- [ ] **Step 2: Run to verify fail**

  ```bash
  go test ./internal/ui/layout/... -run TestPaneIDs_PageBConstants -v
  ```

  Expected: compile error — `PaneGatewayHealth undefined`, etc.

- [ ] **Step 3: Update pane.go constants block**

  Replace the current `PaneRequestFlow` + `PaneNetworkLog` lines with:

  ```go
  PaneNetworkLog                   // Page B — not toggleable via number keys
  PaneGatewayHealth                // Page B — not toggleable via number keys
  PanePollingTraffic               // Page B — not toggleable via number keys
  PaneGatewayLive                  // Page B — not toggleable via number keys
  ```

  `PaneRequestFlow` is removed. `PaneNetworkLog` shifts from 9 to 8. The new panes occupy 9, 10, 11.

- [ ] **Step 4: Update TogglePane guard in layout.go**

  Find (around line 282):
  ```go
  if id >= PaneRequestFlow {
  ```

  Replace with:
  ```go
  if id >= PaneNetworkLog {
  ```

- [ ] **Step 5: Run tests to verify pass**

  ```bash
  go test ./internal/ui/layout/... -v
  ```

  Expected: all green. If any test references `PaneRequestFlow`, update that reference now.

- [ ] **Step 6: Build full project to catch all references to PaneRequestFlow**

  ```bash
  go build ./...
  ```

  Expected: compile errors listing every remaining reference to `PaneRequestFlow`. Fix each:
  - `internal/ui/layout/presets.go` — will be replaced in Task 11 (leave for now; comment out the preset or make it a skeleton so the build passes)
  - `internal/ui/layout/border.go` — add a `// TODO` stub case for now, or temporarily use `default` to absorb all; will be cleaned up in Task 12
  - `internal/app/app.go` — add temporary stub new pane fields as `nil` so it compiles; will be properly wired in Task 12
  - `internal/app/handlers.go` — temporarily remove RequestFlowPane-specific handler blocks; these are replaced in Task 12

  > Strategy: make the build green with minimal temporary stubs. Do NOT delete the requestflow pane files yet — they still compile fine (they just export PaneRequestFlow which no longer exists). Instead, temporarily add `var _ = layout.PaneID(0)` placeholders or just assign `layout.PaneGatewayHealth` to the old pane slot.

  > Simplest path: In `presets.go`, temporarily replace `PaneRequestFlow` with `PaneGatewayHealth` in the visible map and grid so the file compiles. This will be overwritten in Task 11.

- [ ] **Step 7: Commit**

  ```bash
  git add internal/ui/layout/pane.go internal/ui/layout/layout.go internal/ui/layout/layout_test.go \
          internal/ui/layout/presets.go internal/ui/layout/border.go \
          internal/app/app.go internal/app/handlers.go
  git commit -m "feat(layout): add Page B PaneID constants (GatewayHealth, PollingTraffic, GatewayLive)"
  ```

---

### Task 6: Move PollingSnapshotMsg to messages.go

**Files:**
- Modify: `internal/ui/panes/messages.go`
- Modify: `internal/ui/panes/requestflow_pane.go` (remove the type definition)

- [ ] **Step 1: Move the type to messages.go**

  Cut from `requestflow_pane.go` (lines 18–27):

  ```go
  // PollingSnapshotMsg carries app-level polling state to the PollingTrafficPane.
  // The app sends this on each TickMsg so the pane can display polling diagnostics.
  type PollingSnapshotMsg struct {
      // TickIntervalMs is the current playback polling interval in milliseconds.
      TickIntervalMs int
      // IsIdle is true when the user has not pressed a key for idleThresholdSecs.
      IsIdle bool
      // IdleSecs is how long the user has been idle (0 when not idle).
      IdleSecs int
  }
  ```

  Paste into `messages.go` at the end (before final blank line). Update the doc comment to reference `PollingTrafficPane` instead of `RequestFlowPane`.

- [ ] **Step 2: Build to verify no duplicate type**

  ```bash
  go build ./internal/ui/panes/...
  ```

  Expected: clean compile (type is now defined once in messages.go; requestflow_pane.go still compiles because it never re-defined it after the move).

- [ ] **Step 3: Commit**

  ```bash
  git add internal/ui/panes/messages.go internal/ui/panes/requestflow_pane.go
  git commit -m "refactor(panes): move PollingSnapshotMsg to messages.go before RequestFlow deletion"
  ```

---

### Task 7: Create GatewayHealthPane

**Files:**
- Create: `internal/ui/panes/gateway_health_pane.go`
- Create: `internal/ui/panes/gateway_health_pane_test.go`

**Design:**
- No scroll, no filter
- 4-row 3-column fixed-width grid: icon · label (8 chars padded) · data
- Data source: `store.ReadEventsFrom(cursor)` — extract `event.Snapshot` from newest event

- [ ] **Step 1: Write failing tests**

  Create `gateway_health_pane_test.go`:

  ```go
  package panes_test

  import (
      "testing"

      tea "github.com/charmbracelet/bubbletea"
      "github.com/initgrep-apps/spotnik/internal/domain"
      "github.com/initgrep-apps/spotnik/internal/state"
      "github.com/initgrep-apps/spotnik/internal/ui/layout"
      "github.com/initgrep-apps/spotnik/internal/ui/panes"
      "github.com/initgrep-apps/spotnik/internal/ui/theme"
      "github.com/stretchr/testify/assert"
  )

  func newTestGatewayHealthPane(t *testing.T) *panes.GatewayHealthPane {
      t.Helper()
      return panes.NewGatewayHealthPane(state.New(), theme.Load("black"))
  }

  func TestGatewayHealthPane_ImplementsLayoutPane(t *testing.T) {
      var _ layout.Pane = newTestGatewayHealthPane(t)
  }

  func TestGatewayHealthPane_ID(t *testing.T) {
      assert.Equal(t, layout.PaneGatewayHealth, newTestGatewayHealthPane(t).ID())
  }

  func TestGatewayHealthPane_Title(t *testing.T) {
      assert.Equal(t, "Gateway Health", newTestGatewayHealthPane(t).Title())
  }

  func TestGatewayHealthPane_ToggleKey(t *testing.T) {
      assert.Equal(t, 2, newTestGatewayHealthPane(t).ToggleKey())
  }

  func TestGatewayHealthPane_View_EmptyBeforeResize(t *testing.T) {
      assert.Equal(t, "", newTestGatewayHealthPane(t).View())
  }

  func TestGatewayHealthPane_View_ContainsHealthRows(t *testing.T) {
      p := newTestGatewayHealthPane(t)
      p.SetSize(50, 10)
      view := p.View()
      assert.Contains(t, view, "Tokens")
      assert.Contains(t, view, "Slots")
      assert.Contains(t, view, "Backoff")
      assert.Contains(t, view, "Dedup")
  }

  func TestGatewayHealthPane_Update_DrainsCursor(t *testing.T) {
      store := state.New()
      p := panes.NewGatewayHealthPane(store, theme.Load("black"))
      p.SetSize(50, 10)

      // Emit a gateway event so the cursor advances.
      store.RecordEvent(domain.GatewayEvent{
          Kind: domain.EventTokenConsumed,
          Snapshot: domain.GatewayStateSnapshot{
              TokensAvailable: 7, TokensMax: 10,
          },
      })

      p.Update(TickMsg{})
      view := p.View()
      // After processing the event the snapshot is updated; tokens row reflects it.
      assert.Contains(t, view, "Tokens")
  }
  ```

- [ ] **Step 2: Run tests to verify they fail**

  ```bash
  go test ./internal/ui/panes/... -run TestGatewayHealth -v
  ```

  Expected: compile error — `panes.GatewayHealthPane undefined`.

- [ ] **Step 3: Implement gateway_health_pane.go**

  ```go
  package panes

  import (
      "fmt"
      "strings"

      tea "github.com/charmbracelet/bubbletea"
      "github.com/charmbracelet/lipgloss"
      "github.com/initgrep-apps/spotnik/internal/domain"
      "github.com/initgrep-apps/spotnik/internal/state"
      "github.com/initgrep-apps/spotnik/internal/ui/layout"
      "github.com/initgrep-apps/spotnik/internal/ui/theme"
      "github.com/initgrep-apps/spotnik/internal/uikit"
  )

  var _ layout.Pane = &GatewayHealthPane{}

  type GatewayHealthPane struct {
      store       state.StateReader
      theme       theme.Theme
      focused     bool
      width       int
      height      int
      eventCursor uint64
      snapshot    domain.GatewayStateSnapshot
  }

  func NewGatewayHealthPane(s state.StateReader, th theme.Theme) *GatewayHealthPane {
      return &GatewayHealthPane{
          store:    s,
          theme:    th,
          snapshot: domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5},
      }
  }

  func (p *GatewayHealthPane) ID() layout.PaneID        { return layout.PaneGatewayHealth }
  func (p *GatewayHealthPane) Title() string             { return "Gateway Health" }
  func (p *GatewayHealthPane) ToggleKey() int            { return 2 }
  func (p *GatewayHealthPane) Actions() []layout.Action  { return nil }
  func (p *GatewayHealthPane) IsFocused() bool           { return p.focused }
  func (p *GatewayHealthPane) SetFocused(f bool)         { p.focused = f }
  func (p *GatewayHealthPane) Init() tea.Cmd             { return nil }
  func (p *GatewayHealthPane) SetSize(w, h int)          { p.width = w; p.height = h }
  func (p *GatewayHealthPane) SetTheme(th theme.Theme)   { p.theme = th }

  func (p *GatewayHealthPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
      if _, ok := msg.(TickMsg); ok {
          p.drainEvents()
      }
      return p, nil
  }

  func (p *GatewayHealthPane) drainEvents() {
      if p.store == nil {
          return
      }
      newCursor, events := p.store.ReadEventsFrom(p.eventCursor)
      p.eventCursor = newCursor
      if len(events) > 0 {
          p.snapshot = events[len(events)-1].Snapshot
      }
  }

  func (p *GatewayHealthPane) View() string {
      if p.width == 0 || p.height == 0 {
          return ""
      }

      th := p.theme
      snap := p.snapshot
      mode := uikit.ActiveMode()
      const labelWidth = 8

      mutedStyle := lipgloss.NewStyle().Foreground(th.TextMuted())

      // Token row
      tokenColor := th.TextSecondary()
      if snap.TokensMax > 0 && snap.TokensAvailable <= 2 {
          tokenColor = th.Warning()
      }
      tokenStyle := lipgloss.NewStyle().Foreground(tokenColor)
      tokenIcon := tokenStyle.Render(uikit.GlyphFor(uikit.GlyphFilledDot, mode))
      tokenBar := p.renderDotBar(snap.TokensAvailable, snap.TokensMax,
          uikit.GlyphFilledDot, uikit.GlyphAvailable, tokenStyle, mutedStyle)
      tokenRow := p.renderRow(tokenIcon, "Tokens", tokenBar, labelWidth, mutedStyle)

      // Slot row
      slotColor := th.TextSecondary()
      if snap.ConcurrentMax > 0 && snap.ConcurrentActive >= snap.ConcurrentMax {
          slotColor = th.Warning()
      }
      slotStyle := lipgloss.NewStyle().Foreground(slotColor)
      slotIcon := slotStyle.Render(uikit.GlyphFor(uikit.GlyphFilledSquare, mode))
      slotBar := p.renderDotBar(snap.ConcurrentActive, snap.ConcurrentMax,
          uikit.GlyphFilledSquare, uikit.GlyphEmptySquare, slotStyle, mutedStyle)
      slotRow := p.renderRow(slotIcon, "Slots", slotBar, labelWidth, mutedStyle)

      // Backoff row
      backoffColor := th.TextMuted()
      backoffData := "none"
      if snap.BackoffRemaining > 0 {
          backoffColor = th.Error()
          backoffData = fmt.Sprintf("%.1fs", snap.BackoffRemaining)
      }
      backoffStyle := lipgloss.NewStyle().Foreground(backoffColor)
      backoffRow := p.renderRow(
          backoffStyle.Render(uikit.GlyphFor(uikit.GlyphDeadline, mode)),
          "Backoff", backoffStyle.Render(backoffData), labelWidth, mutedStyle)

      // Dedup row
      dedupColor := th.TextMuted()
      dedupData := "none"
      if snap.DedupWaiters > 0 {
          dedupColor = th.TextSecondary()
          dedupData = fmt.Sprintf("%d waiters", snap.DedupWaiters)
      }
      dedupStyle := lipgloss.NewStyle().Foreground(dedupColor)
      dedupRow := p.renderRow(
          dedupStyle.Render(uikit.GlyphFor(uikit.GlyphRateLimit, mode)),
          "Dedup", dedupStyle.Render(dedupData), labelWidth, mutedStyle)

      content := strings.Join([]string{tokenRow, slotRow, backoffRow, dedupRow}, "\n")
      return uikit.PaneChrome{
          Width: p.width, Height: p.height,
          Title: p.Title(), ToggleKey: p.ToggleKey(),
          AccentColor: layout.PaneBorderColor(p.ID(), th),
          Focused:     p.focused, Theme: th,
      }.Render(content)
  }

  func (p *GatewayHealthPane) renderRow(icon, label, data string, labelWidth int, labelStyle lipgloss.Style) string {
      return icon + "  " + labelStyle.Render(uikit.PadOrTruncate(label, labelWidth)) + "  " + data
  }

  func (p *GatewayHealthPane) renderDotBar(filled, total int,
      filledRole, emptyRole uikit.GlyphRole,
      filledStyle, emptyStyle lipgloss.Style) string {
      if total <= 0 {
          return ""
      }
      mode := uikit.ActiveMode()
      var sb strings.Builder
      for i := 0; i < total; i++ {
          if i < filled {
              sb.WriteString(filledStyle.Render(uikit.GlyphFor(filledRole, mode)))
          } else {
              sb.WriteString(emptyStyle.Render(uikit.GlyphFor(emptyRole, mode)))
          }
      }
      return sb.String()
  }
  ```

- [ ] **Step 4: Run tests to verify pass**

  ```bash
  go test ./internal/ui/panes/... -run TestGatewayHealth -v
  ```

  Expected: all green.

- [ ] **Step 5: Build to confirm no compile errors**

  ```bash
  go build ./...
  ```

- [ ] **Step 6: Commit**

  ```bash
  git add internal/ui/panes/gateway_health_pane.go internal/ui/panes/gateway_health_pane_test.go
  git commit -m "feat(panes): add GatewayHealthPane (Page B, toggle key 2)"
  ```

---

### Task 8: Create PollingTrafficPane

**Files:**
- Create: `internal/ui/panes/polling_traffic_pane.go`
- Create: `internal/ui/panes/polling_traffic_pane_test.go`

**Design:**
- No scroll, no filter
- Receives `PollingSnapshotMsg` for playback row; reads store TTL sentinels for library rows
- 5-row 3-column fixed-width grid: type icon · label (10 chars) · status glyph + value

- [ ] **Step 1: Write failing tests**

  Create `polling_traffic_pane_test.go`:

  ```go
  package panes_test

  import (
      "testing"

      tea "github.com/charmbracelet/bubbletea"
      "github.com/initgrep-apps/spotnik/internal/state"
      "github.com/initgrep-apps/spotnik/internal/ui/layout"
      "github.com/initgrep-apps/spotnik/internal/ui/panes"
      "github.com/initgrep-apps/spotnik/internal/ui/theme"
      "github.com/stretchr/testify/assert"
  )

  func newTestPollingTrafficPane(t *testing.T) *panes.PollingTrafficPane {
      t.Helper()
      return panes.NewPollingTrafficPane(state.New(), theme.Load("black"))
  }

  func TestPollingTrafficPane_ImplementsLayoutPane(t *testing.T) {
      var _ layout.Pane = newTestPollingTrafficPane(t)
  }

  func TestPollingTrafficPane_ID(t *testing.T) {
      assert.Equal(t, layout.PanePollingTraffic, newTestPollingTrafficPane(t).ID())
  }

  func TestPollingTrafficPane_Title(t *testing.T) {
      assert.Equal(t, "Polling Traffic", newTestPollingTrafficPane(t).Title())
  }

  func TestPollingTrafficPane_ToggleKey(t *testing.T) {
      assert.Equal(t, 3, newTestPollingTrafficPane(t).ToggleKey())
  }

  func TestPollingTrafficPane_View_EmptyBeforeResize(t *testing.T) {
      assert.Equal(t, "", newTestPollingTrafficPane(t).View())
  }

  func TestPollingTrafficPane_View_ContainsAllRows(t *testing.T) {
      p := newTestPollingTrafficPane(t)
      p.SetSize(50, 10)
      view := p.View()
      assert.Contains(t, view, "Playback")
      assert.Contains(t, view, "Playlists")
      assert.Contains(t, view, "Albums")
      assert.Contains(t, view, "Liked")
      assert.Contains(t, view, "Recent")
  }

  func TestPollingTrafficPane_Update_PollingSnapshotMsg(t *testing.T) {
      p := newTestPollingTrafficPane(t)
      p.SetSize(50, 10)

      model, cmd := p.Update(panes.PollingSnapshotMsg{
          TickIntervalMs: 1000,
          IsIdle:         false,
      })
      assert.Nil(t, cmd)
      view := model.(*panes.PollingTrafficPane).View()
      // Playback row reflects "running" state when not idle.
      assert.Contains(t, view, "running")
  }

  func TestPollingTrafficPane_Update_IdleSnapshot(t *testing.T) {
      p := newTestPollingTrafficPane(t)
      p.SetSize(50, 10)

      model, _ := p.Update(panes.PollingSnapshotMsg{
          TickIntervalMs: 10000,
          IsIdle:         true,
          IdleSecs:       90,
      })
      view := model.(*panes.PollingTrafficPane).View()
      assert.Contains(t, view, "idle")
  }
  ```

- [ ] **Step 2: Run tests to verify they fail**

  ```bash
  go test ./internal/ui/panes/... -run TestPollingTraffic -v
  ```

  Expected: compile error.

- [ ] **Step 3: Implement polling_traffic_pane.go**

  ```go
  package panes

  import (
      "fmt"
      "strings"
      "time"

      tea "github.com/charmbracelet/bubbletea"
      "github.com/charmbracelet/lipgloss"
      "github.com/initgrep-apps/spotnik/internal/state"
      "github.com/initgrep-apps/spotnik/internal/ui/layout"
      "github.com/initgrep-apps/spotnik/internal/ui/theme"
      "github.com/initgrep-apps/spotnik/internal/uikit"
  )

  // Compile-time check.
  var _ layout.Pane = &PollingTrafficPane{}

  // PollingTrafficPane shows the current polling cadence for playback and library cache freshness.
  // Data comes from PollingSnapshotMsg (playback) and store TTL sentinel methods (library).
  type PollingTrafficPane struct {
      store        state.StateReader
      theme        theme.Theme
      focused      bool
      width        int
      height       int
      pollSnapshot PollingSnapshotMsg
  }

  // NewPollingTrafficPane creates a PollingTrafficPane.
  func NewPollingTrafficPane(s state.StateReader, th theme.Theme) *PollingTrafficPane {
      return &PollingTrafficPane{store: s, theme: th}
  }

  func (p *PollingTrafficPane) ID() layout.PaneID       { return layout.PanePollingTraffic }
  func (p *PollingTrafficPane) Title() string            { return "Polling Traffic" }
  func (p *PollingTrafficPane) ToggleKey() int           { return 3 }
  func (p *PollingTrafficPane) Actions() []layout.Action { return nil }
  func (p *PollingTrafficPane) IsFocused() bool          { return p.focused }
  func (p *PollingTrafficPane) SetFocused(f bool)        { p.focused = f }
  func (p *PollingTrafficPane) Init() tea.Cmd            { return nil }
  func (p *PollingTrafficPane) SetSize(w, h int)         { p.width = w; p.height = h }
  func (p *PollingTrafficPane) SetTheme(th theme.Theme)  { p.theme = th }

  func (p *PollingTrafficPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
      if m, ok := msg.(PollingSnapshotMsg); ok {
          p.pollSnapshot = m
      }
      return p, nil
  }

  func (p *PollingTrafficPane) View() string {
      if p.width == 0 || p.height == 0 {
          return ""
      }

      th := p.theme
      mode := uikit.ActiveMode()
      const labelWidth = 10

      mutedStyle := lipgloss.NewStyle().Foreground(th.TextMuted())

      renderTypeIcon := func(role uikit.GlyphRole) string {
          return mutedStyle.Render(uikit.GlyphFor(role, mode))
      }

      // Playback row
      var playRow string
      {
          icon := renderTypeIcon(uikit.GlyphMusicNote)
          label := mutedStyle.Render(uikit.PadOrTruncate("Playback", labelWidth))
          if p.pollSnapshot.IsIdle {
              glyph := lipgloss.NewStyle().Foreground(th.Warning()).Render(uikit.GlyphFor(uikit.GlyphPaused, mode))
              intervalStr := pollingHumanInterval(p.pollSnapshot.TickIntervalMs)
              status := lipgloss.NewStyle().Foreground(th.Warning()).Render(fmt.Sprintf("idle · %s", intervalStr))
              playRow = icon + "  " + label + "  " + glyph + " " + status
          } else {
              glyph := lipgloss.NewStyle().Foreground(th.Success()).Render(uikit.GlyphFor(uikit.GlyphPlaying, mode))
              intervalStr := pollingHumanInterval(p.pollSnapshot.TickIntervalMs)
              status := lipgloss.NewStyle().Foreground(th.Success()).Render(fmt.Sprintf("%s · running", intervalStr))
              playRow = icon + "  " + label + "  " + glyph + " " + status
          }
      }

      // Library cache rows: Playlists, Albums, Liked, Recent
      type cacheRow struct {
          typeRole   uikit.GlyphRole
          label      string
          fetchedAt  time.Time
          ttl        time.Duration
      }
      cacheRows := []cacheRow{
          {uikit.GlyphQueue,      "Playlists", p.store.PlaylistsFetchedAt(),    state.PlaylistsTTL},
          {uikit.GlyphDoubleNote, "Albums",    p.store.AlbumsFetchedAt(),       state.AlbumsTTL},
          {uikit.GlyphPinned,     "Liked",     p.store.LikedTracksFetchedAt(),  state.LikedTracksTTL},
          {uikit.GlyphDeadline,   "Recent",    p.store.RecentPlayedFetchedAt(), state.RecentlyPlayedTTL},
      }

      renderedRows := make([]string, 0, 5)
      renderedRows = append(renderedRows, playRow)

      for _, cr := range cacheRows {
          icon := renderTypeIcon(cr.typeRole)
          label := mutedStyle.Render(uikit.PadOrTruncate(cr.label, labelWidth))
          var statusStr string
          if cr.fetchedAt.IsZero() {
              statusStr = mutedStyle.Render("never fetched")
          } else if !state.IsStale(cr.fetchedAt, cr.ttl) {
              freshGlyph := lipgloss.NewStyle().Foreground(th.TextMuted()).Render(uikit.GlyphFor(uikit.GlyphAvailable, mode))
              statusStr = freshGlyph + " " + mutedStyle.Render("fresh")
          } else {
              age := cacheAge(cr.fetchedAt)
              d := time.Since(cr.fetchedAt)
              var staleColor lipgloss.Color
              if d >= time.Hour {
                  staleColor = th.Error()
              } else {
                  staleColor = th.Warning()
              }
              warnGlyph := lipgloss.NewStyle().Foreground(staleColor).Render(uikit.GlyphFor(uikit.GlyphWarning, mode))
              statusStr = warnGlyph + " " + lipgloss.NewStyle().Foreground(staleColor).Render(age+" stale")
          }
          renderedRows = append(renderedRows, icon+"  "+label+"  "+statusStr)
      }

      content := strings.Join(renderedRows, "\n")
      return uikit.PaneChrome{
          Width: p.width, Height: p.height,
          Title: p.Title(), ToggleKey: p.ToggleKey(),
          AccentColor: layout.PaneBorderColor(p.ID(), th),
          Focused:     p.focused, Theme: th,
      }.Render(content)
  }

  // pollingHumanInterval converts milliseconds to "Xs" or "Xms".
  func pollingHumanInterval(ms int) string {
      if ms <= 0 {
          return "?"
      }
      if ms >= 1000 {
          return fmt.Sprintf("%ds", ms/1000)
      }
      return fmt.Sprintf("%dms", ms)
  }
  ```

  > **Note on `cacheAge`:** `humanAge` in `requestflow_pane.go` returns `"2h 15m ago"` format.
  > The spec requires `"2h 55m stale"` — a bare duration with `" stale"` appended by the caller.
  > Define `cacheAge` in `polling_traffic_pane.go` with a different body (no "ago" suffix):
  >
  > ```go
  > // cacheAge returns a human-readable duration since t (e.g. "3m", "2h 15m").
  > // Unlike humanAge (requestflow_pane.go), this omits the "ago" suffix so the
  > // caller can append context-specific text such as " stale".
  > func cacheAge(t time.Time) string {
  >     d := time.Since(t)
  >     if d < time.Minute {
  >         return "just now"
  >     }
  >     if d < time.Hour {
  >         return fmt.Sprintf("%dm", int(d.Minutes()))
  >     }
  >     h := int(d.Hours())
  >     m := int(d.Minutes()) % 60
  >     if m == 0 {
  >         return fmt.Sprintf("%dh", h)
  >     }
  >     return fmt.Sprintf("%dh %dm", h, m)
  > }
  > ```
  >
  > Usage in `View()`: `age + " stale"` → `"2h 55m stale"` as required by the spec.

- [ ] **Step 4: Run tests to verify pass**

  ```bash
  go test ./internal/ui/panes/... -run TestPollingTraffic -v
  ```

- [ ] **Step 5: Commit**

  ```bash
  git add internal/ui/panes/polling_traffic_pane.go internal/ui/panes/polling_traffic_pane_test.go
  git commit -m "feat(panes): add PollingTrafficPane (Page B, toggle key 3)"
  ```

---

### Task 9: Create GatewayLivePane

**Files:**
- Create: `internal/ui/panes/gateway_live_pane.go`
- Create: `internal/ui/panes/gateway_live_pane_test.go`

**Design:**
- Scrollable + filterable (Enter-to-apply filter, `filter(query)` in border)
- 500-entry buffer, prepend-on-tick, reverse-chronological
- Each row is a `uikit.ListRow` (Glyph, Label, Intent — no Caption)
- Esc resets scroll to top (when filter not active)

- [ ] **Step 1: Write failing tests**

  Create `gateway_live_pane_test.go`:

  ```go
  package panes_test

  import (
      "fmt"
      "testing"

      tea "github.com/charmbracelet/bubbletea"
      "github.com/initgrep-apps/spotnik/internal/domain"
      "github.com/initgrep-apps/spotnik/internal/state"
      "github.com/initgrep-apps/spotnik/internal/ui/layout"
      "github.com/initgrep-apps/spotnik/internal/ui/panes"
      "github.com/initgrep-apps/spotnik/internal/ui/theme"
      "github.com/stretchr/testify/assert"
      "github.com/stretchr/testify/require"
  )

  func newTestGatewayLivePane(t *testing.T) *panes.GatewayLivePane {
      t.Helper()
      return panes.NewGatewayLivePane(state.New(), theme.Load("black"))
  }

  func TestGatewayLivePane_ImplementsLayoutPane(t *testing.T) {
      var _ layout.Pane = newTestGatewayLivePane(t)
  }

  func TestGatewayLivePane_ID(t *testing.T) {
      assert.Equal(t, layout.PaneGatewayLive, newTestGatewayLivePane(t).ID())
  }

  func TestGatewayLivePane_Title(t *testing.T) {
      assert.Equal(t, "Gateway Live", newTestGatewayLivePane(t).Title())
  }

  func TestGatewayLivePane_ToggleKey(t *testing.T) {
      assert.Equal(t, 4, newTestGatewayLivePane(t).ToggleKey())
  }

  func TestGatewayLivePane_View_EmptyBeforeResize(t *testing.T) {
      assert.Equal(t, "", newTestGatewayLivePane(t).View())
  }

  func TestGatewayLivePane_Update_DrainsCursorOnTick(t *testing.T) {
      store := state.New()
      p := panes.NewGatewayLivePane(store, theme.Load("black"))
      p.SetSize(80, 20)

      store.RecordEvent(domain.GatewayEvent{
          Kind:   domain.EventRequestEntered,
          Method: "GET", Path: "/v1/me/player",
          Priority: domain.PriorityInteractive,
      })

      p.Update(panes.TickMsg{})
      assert.Equal(t, 1, p.BufferedEventCount(), "expected 1 event after tick")
  }

  func TestGatewayLivePane_Buffer_CapsAt500(t *testing.T) {
      store := state.New()
      p := panes.NewGatewayLivePane(store, theme.Load("black"))
      p.SetSize(80, 20)

      for i := 0; i < 510; i++ {
          store.RecordEvent(domain.GatewayEvent{
              Kind: domain.EventRequestEntered, Method: "GET",
              Path: fmt.Sprintf("/v1/me/item/%d", i),
          })
      }
      p.Update(panes.TickMsg{})
      assert.LessOrEqual(t, p.BufferedEventCount(), 500, "buffer must not exceed 500")
  }

  func TestGatewayLivePane_Esc_ResetsScrollWhenFilterInactive(t *testing.T) {
      store := state.New()
      for i := 0; i < 60; i++ {
          store.RecordEvent(domain.GatewayEvent{
              Kind: domain.EventRequestEntered, Method: "GET",
              Path: fmt.Sprintf("/v1/item/%d", i),
          })
      }
      p := panes.NewGatewayLivePane(store, theme.Load("black"))
      p.SetSize(80, 5)
      p.SetFocused(true)
      p.Update(panes.TickMsg{}) // drain events

      for range 8 {
          p.Update(tea.KeyMsg{Type: tea.KeyDown})
      }
      require.Greater(t, p.TableCurrentPage(), 1)
      p.Update(tea.KeyMsg{Type: tea.KeyEsc})
      assert.Equal(t, 1, p.TableCurrentPage())
  }

  func TestGatewayLivePane_HasActiveFilter(t *testing.T) {
      p := newTestGatewayLivePane(t)
      p.SetSize(80, 20)
      p.SetFocused(true)
      assert.False(t, p.HasActiveFilter())
      p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
      assert.True(t, p.HasActiveFilter())
  }
  ```

- [ ] **Step 2: Run tests to verify they fail**

  ```bash
  go test ./internal/ui/panes/... -run TestGatewayLive -v
  ```

  Expected: compile error.

- [ ] **Step 3: Implement gateway_live_pane.go**

  Key decisions:
  - Uses `components.Table` for scrollable display (rows built from `uikit.ListRow.Render()`)
  - Filter is `components.Filter` in Enter-to-apply mode: `f` opens filter → typing builds query → `Enter` commits (sets `activeQuery`)
  - Border shows `filter(query)` when filter is active via a custom `Actions()` return
  - Buffer: `[]gatewayLiveRow` slice, prepend batch then trim to 500; table built on each tick

  ```go
  package panes

  import (
      "fmt"
      "strings"
      "time"

      tea "github.com/charmbracelet/bubbletea"
      "github.com/initgrep-apps/spotnik/internal/domain"
      "github.com/initgrep-apps/spotnik/internal/state"
      "github.com/initgrep-apps/spotnik/internal/ui/components"
      "github.com/initgrep-apps/spotnik/internal/ui/layout"
      "github.com/initgrep-apps/spotnik/internal/ui/theme"
      "github.com/initgrep-apps/spotnik/internal/uikit"
  )

  const maxGatewayLiveRows = 500

  // gatewayLiveRow holds display data for one gateway event row.
  type gatewayLiveRow struct {
      glyphRole   uikit.GlyphRole
      intent      uikit.Role
      label       string // "HH:MM:SS  <event description>"
      matchString string // pre-built string for filter matching
  }

  // Compile-time check.
  var _ layout.Pane = &GatewayLivePane{}
  var _ layout.FilterablePane = &GatewayLivePane{}

  // GatewayLivePane displays a scrollable, filterable reverse-chronological stream
  // of gateway events. New events prepend at top on each tick.
  type GatewayLivePane struct {
      store       state.StateReader
      theme       theme.Theme
      focused     bool
      width       int
      height      int
      eventCursor uint64
      buffer      []gatewayLiveRow // newest-first; capped at maxGatewayLiveRows
      table       *components.Table
      filter      *components.Filter
      activeQuery string // committed filter query (Enter-to-apply)
  }

  // NewGatewayLivePane creates a GatewayLivePane.
  func NewGatewayLivePane(s state.StateReader, th theme.Theme) *GatewayLivePane {
      columns := []components.ColumnDef{
          {Key: "row", Header: "", FlexFactor: 1, Color: th.TextPrimary()},
      }
      t := components.NewTable(components.TableConfig{
          Columns: columns, Theme: th, PlayingIndex: -1, ShowHeader: false,
      })
      return &GatewayLivePane{
          store:  s,
          theme:  th,
          table:  t,
          filter: components.NewFilter(th),
      }
  }

  func (p *GatewayLivePane) ID() layout.PaneID    { return layout.PaneGatewayLive }
  func (p *GatewayLivePane) Title() string         { return "Gateway Live" }
  func (p *GatewayLivePane) ToggleKey() int        { return 4 }
  func (p *GatewayLivePane) IsFocused() bool       { return p.focused }
  func (p *GatewayLivePane) Init() tea.Cmd         { return nil }
  func (p *GatewayLivePane) SetTheme(th theme.Theme) {
      p.theme = th
      cols := []components.ColumnDef{
          {Key: "row", Header: "", FlexFactor: 1, Color: th.TextPrimary()},
      }
      p.table, p.filter = components.RebuildTableTheme(th, cols, p.table.Rows(), p.focused && !p.filter.IsActive())
      p.resizeTable()
  }

  // HasActiveFilter returns true when the filter input is open.
  func (p *GatewayLivePane) HasActiveFilter() bool { return p.filter.IsActive() }

  // BufferedEventCount returns how many events are in the display buffer. Exported for testing.
  func (p *GatewayLivePane) BufferedEventCount() int { return len(p.buffer) }

  // TableCurrentPage returns the table's current page number. Exported for testing.
  func (p *GatewayLivePane) TableCurrentPage() int { return p.table.CurrentPage() }

  func (p *GatewayLivePane) Actions() []layout.Action {
      if p.filter.IsActive() {
          return []layout.Action{{Key: "Esc", Label: "cancel"}}
      }
      return []layout.Action{{Key: "f", Label: "filter"}}
  }

  func (p *GatewayLivePane) SetFocused(focused bool) {
      p.focused = focused
      p.table.SetFocused(focused && !p.filter.IsActive())
  }

  func (p *GatewayLivePane) SetSize(width, height int) {
      p.width = width
      p.height = height
      p.filter.SetWidth(width)
      p.resizeTable()
  }

  func (p *GatewayLivePane) resizeTable() {
      tableHeight := p.height
      if p.filter.IsActive() {
          tableHeight--
      }
      if tableHeight < 0 {
          tableHeight = 0
      }
      p.table.SetSize(p.width, tableHeight)
  }

  func (p *GatewayLivePane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
      switch m := msg.(type) {
      case TickMsg:
          p.drainEvents()
          p.buildTableRows()
          return p, nil

      case tea.KeyMsg:
          if !p.focused {
              return p, nil
          }
          return p.handleKey(m)
      }
      return p, nil
  }

  func (p *GatewayLivePane) handleKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
      // When filter input is open, forward to filter.
      if p.filter.IsActive() {
          switch m.Type {
          case tea.KeyEnter:
              // Commit the filter query.
              p.activeQuery = p.filter.Query()
              p.filter.Toggle() // close the input
              p.table.SetFocused(true)
              p.resizeTable()
              p.buildTableRows()
              return p, nil
          case tea.KeyEscape:
              // Cancel — close without committing.
              p.filter.Toggle()
              p.table.SetFocused(true)
              p.resizeTable()
              return p, nil
          default:
              cmd := p.filter.Update(m)
              return p, cmd
          }
      }

      switch m.Type {
      case tea.KeyRunes:
          if string(m.Runes) == "f" {
              p.filter.Toggle()
              p.table.SetFocused(false)
              p.resizeTable()
              return p, nil
          }
      case tea.KeyEscape:
          if p.activeQuery != "" {
              // First Esc: clear committed filter.
              p.activeQuery = ""
              p.buildTableRows()
              return p, nil
          }
          // Second Esc (or no filter): reset scroll.
          p.table.GotoTop()
          return p, nil
      }

      cmd := p.table.Update(m)
      return p, cmd
  }

  func (p *GatewayLivePane) drainEvents() {
      if p.store == nil {
          return
      }
      newCursor, events := p.store.ReadEventsFrom(p.eventCursor)
      p.eventCursor = newCursor
      if len(events) == 0 {
          return
      }

      // Build new rows from freshly drained events.
      newRows := make([]gatewayLiveRow, 0, len(events))
      for _, e := range events {
          row, ok := buildGatewayLiveRow(e)
          if !ok {
              continue
          }
          newRows = append(newRows, row)
      }

      // Prepend new rows (newest at top), then trim to cap.
      // New events come in chronological order; reverse them so newest is first.
      for i, j := 0, len(newRows)-1; i < j; i, j = i+1, j-1 {
          newRows[i], newRows[j] = newRows[j], newRows[i]
      }
      p.buffer = append(newRows, p.buffer...)
      if len(p.buffer) > maxGatewayLiveRows {
          p.buffer = p.buffer[:maxGatewayLiveRows]
      }
  }

  // buildGatewayLiveRow maps a domain.GatewayEvent to a display row.
  // Returns (row, true) for known event kinds; (zero, false) for unknown.
  func buildGatewayLiveRow(e domain.GatewayEvent) (gatewayLiveRow, bool) {
      ts := e.Timestamp.Format("15:04:05")
      path := strings.TrimPrefix(e.Path, "/v1/me")

      switch e.Kind {
      case domain.EventRequestEntered:
          pri := "background"
          role := uikit.GlyphDeadline
          intent := uikit.RoleMuted
          matchStr := path + " " + pri
          if e.Priority == domain.PriorityInteractive {
              pri = "interactive"
              role = uikit.GlyphRunning
              intent = uikit.RolePlain
              matchStr = path + " " + pri
          }
          return gatewayLiveRow{role, intent, fmt.Sprintf("%s  %s %s", ts, e.Method, path), matchStr}, true

      case domain.EventTokenConsumed:
          return gatewayLiveRow{
              uikit.GlyphWarning, uikit.RoleWarning,
              fmt.Sprintf("%s  Token consumed → %d", ts, e.Snapshot.TokensAvailable),
              "token consumed",
          }, true

      case domain.EventTokenRefilled:
          return gatewayLiveRow{
              uikit.GlyphRepeatAll, uikit.RoleSuccess,
              fmt.Sprintf("%s  Tokens refilled → %d", ts, e.Snapshot.TokensAvailable),
              "token refilled",
          }, true

      case domain.EventSemaphoreAcquired:
          return gatewayLiveRow{
              uikit.GlyphFilledSquare, uikit.RoleInfo,
              fmt.Sprintf("%s  Semaphore acquired  %d/%d", ts, e.Snapshot.ConcurrentActive, e.Snapshot.ConcurrentMax),
              "semaphore acquired",
          }, true

      case domain.EventSemaphoreReleased:
          return gatewayLiveRow{
              uikit.GlyphEmptySquare, uikit.RoleMuted,
              fmt.Sprintf("%s  Semaphore released  %d/%d", ts, e.Snapshot.ConcurrentActive, e.Snapshot.ConcurrentMax),
              "semaphore released",
          }, true

      case domain.EventRequestAllowed:
          return gatewayLiveRow{
              uikit.GlyphSuccess, uikit.RoleSuccess,
              fmt.Sprintf("%s  %s %s  allowed", ts, e.Method, path),
              path + " allowed",
          }, true

      case domain.EventRequestBlocked:
          return gatewayLiveRow{
              uikit.GlyphError, uikit.RoleError,
              fmt.Sprintf("%s  %s %s  blocked", ts, e.Method, path),
              path + " blocked",
          }, true

      case domain.EventDedupJoined:
          return gatewayLiveRow{
              uikit.GlyphRateLimit, uikit.RoleInfo,
              fmt.Sprintf("%s  %s %s  dedup joined", ts, e.Method, path),
              path + " dedup",
          }, true

      case domain.EventDedupResolved:
          return gatewayLiveRow{
              uikit.GlyphSuccess, uikit.RoleSuccess,
              fmt.Sprintf("%s  Dedup resolved  %d", ts, e.StatusCode),
              "dedup resolved",
          }, true

      case domain.EventBackoffStarted:
          return gatewayLiveRow{
              uikit.GlyphBlocked, uikit.RoleError,
              fmt.Sprintf("%s  Backoff started  (retry in %.1fs)", ts, e.Snapshot.BackoffRemaining),
              "backoff",
          }, true

      case domain.EventHttpCompleted:
          return gatewayLiveRow{
              uikit.GlyphSuccess, uikit.RoleSuccess,
              fmt.Sprintf("%s  %d  %dms", ts, e.StatusCode, e.DurationMs),
              fmt.Sprintf("%d", e.StatusCode),
          }, true
      }
      return gatewayLiveRow{}, false
  }

  func (p *GatewayLivePane) buildTableRows() {
      query := p.activeQuery
      rows := make([]map[string]string, 0, len(p.buffer))
      for _, row := range p.buffer {
          if query != "" && !strings.Contains(strings.ToLower(row.matchString), strings.ToLower(query)) {
              continue
          }
          rendered := uikit.ListRow{
              Glyph:  row.glyphRole,
              Label:  row.label,
              Intent: row.intent,
              Theme:  p.theme,
          }.Render(p.width - 2) // -2 for border
          rows = append(rows, map[string]string{"row": rendered})
      }
      p.table.SetRows(rows)
  }

  func (p *GatewayLivePane) View() string {
      if p.width == 0 || p.height == 0 {
          return ""
      }
      var parts []string
      if p.filter.IsActive() {
          parts = append(parts, p.filter.View(p.width))
      }
      parts = append(parts, p.table.View())
      content := strings.Join(parts, "\n")
      return uikit.PaneChrome{
          Width: p.width, Height: p.height,
          Title: p.Title(), ToggleKey: p.ToggleKey(),
          Actions:     p.Actions(),
          FilterQuery: p.activeQuery,
          AccentColor: layout.PaneBorderColor(p.ID(), p.theme),
          Focused:     p.focused, Theme: p.theme,
      }.Render(content)
  }
  ```

  > **Note on filter matching with committed query:** `components.Filter.MatchesAny` uses the filter's live `Query()` state. Since we commit the query to `p.activeQuery` and close the filter, `filter.Query()` will be empty when the filter is closed. The `buildTableRows()` method must match against `p.activeQuery` directly — use `strings.Contains(strings.ToLower(row.matchString), strings.ToLower(query))` as shown above.

- [ ] **Step 4: Run tests to verify pass**

  ```bash
  go test ./internal/ui/panes/... -run TestGatewayLive -v
  ```

- [ ] **Step 5: Commit**

  ```bash
  git add internal/ui/panes/gateway_live_pane.go internal/ui/panes/gateway_live_pane_test.go
  git commit -m "feat(panes): add GatewayLivePane (Page B, toggle key 4, scrollable event stream)"
  ```

---

### Task 10: Fix NetworkLogPane

**Files:**
- Modify: `internal/ui/panes/networklog_pane.go`
- Modify: `internal/ui/panes/networklog_pane_test.go`

**Changes:**
1. Promote `decisions` local map → `pendingDecisions` persistent struct field (decision cross-tick bug)
2. Column headers to Title Case: `"TIME"` → `"Time"`, `"METHOD"` → `"Method"`, etc.
3. Esc scroll-reset was already added in Task 2 for the simple panes — NetworkLog was included there

- [ ] **Step 1: Write failing test for the decision cross-tick fix**

  Add to `networklog_pane_test.go`:

  ```go
  func TestNetworkLogPane_Decision_PersistedAcrossTicks(t *testing.T) {
      store := state.New()
      reqID := uint64(42)

      // Tick 1: RequestAllowed event arrives (no paired HttpCompleted yet)
      store.RecordEvent(domain.GatewayEvent{
          Kind:      domain.EventRequestAllowed,
          RequestID: reqID,
          Method:    "GET", Path: "/v1/me/player",
      })

      p := panes.NewNetworkLogPane(store, theme.Load("black"))
      p.SetSize(120, 20)
      p.Update(panes.TickMsg{})

      // Tick 2: HttpCompleted arrives — decision must be "allowed", not ""
      store.RecordEvent(domain.GatewayEvent{
          Kind:       domain.EventHttpCompleted,
          RequestID:  reqID,
          StatusCode: 200,
          Method:     "GET", Path: "/v1/me/player",
      })
      p.Update(panes.TickMsg{})

      view := p.View()
      assert.Contains(t, view, "Allowed", "decision should survive across ticks")
  }
  ```

- [ ] **Step 2: Run test to verify it fails**

  ```bash
  go test ./internal/ui/panes/... -run TestNetworkLogPane_Decision_PersistedAcrossTicks -v
  ```

  Expected: FAIL — decision shows empty because the local map is discarded each tick.

- [ ] **Step 3: Implement pendingDecisions fix**

  In `NetworkLogPane` struct, add:
  ```go
  pendingDecisions map[uint64]domain.EventKind
  ```

  In `NewNetworkLogPane(...)`, initialize:
  ```go
  pendingDecisions: make(map[uint64]domain.EventKind),
  ```

  In `refreshRows()`, replace the local `decisions := make(...)` block with:

  ```go
  // Accumulate decision events into the persistent map.
  for _, e := range events {
      switch e.Kind {
      case domain.EventRequestAllowed, domain.EventRequestBlocked, domain.EventDedupJoined:
          p.pendingDecisions[e.RequestID] = e.Kind
      }
  }

  // Process completed requests, consume the decision from the persistent map.
  for _, e := range events {
      switch e.Kind {
      case domain.EventHttpCompleted:
          row := networkLogRow{
              // ... existing fields ...
              decision: p.pendingDecisions[e.RequestID],
          }
          delete(p.pendingDecisions, e.RequestID) // prevent unbounded growth
          p.completedRequests = append(p.completedRequests, row)

      case domain.EventRequestBlocked:
          // ... existing blocked row logic, unchanged ...
          delete(p.pendingDecisions, e.RequestID)
      }
  }
  ```

- [ ] **Step 4: Update ToggleKey and column headers**

  In `NetworkLogPane`, update `ToggleKey()` to return `5` (spec assigns key 5 to Network Log on Page B):

  ```go
  func (p *NetworkLogPane) ToggleKey() int { return 5 }
  ```

  In `NewNetworkLogPane`, change the column definitions to Title Case:

  ```go
  columns := []components.ColumnDef{
      {Key: "time",     Header: "Time",     FlexFactor: 3, Color: th.ColumnIndex()},
      {Key: "method",   Header: "Method",   FlexFactor: 2, Color: th.ColumnSecondary()},
      {Key: "endpoint", Header: "Endpoint", FlexFactor: 7, Color: th.ColumnPrimary()},
      {Key: "status",   Header: "Status",   FlexFactor: 2, Color: th.ColumnTertiary()},
      {Key: "latency",  Header: "Latency",  FlexFactor: 2, Color: th.ColumnTertiary()},
      {Key: "priority", Header: "Priority", FlexFactor: 3, Color: th.ColumnIndex()},
      {Key: "decision", Header: "Decision", FlexFactor: 3, Color: th.ColumnSecondary()},
  }
  ```

  Also update any existing tests that assert on the `ToggleKey()` return value or exact uppercase header strings.

- [ ] **Step 5: Run all NetworkLog tests**

  ```bash
  go test ./internal/ui/panes/... -run TestNetworkLogPane -v
  ```

  Expected: all green including the new cross-tick decision test.

- [ ] **Step 6: Commit**

  ```bash
  git add internal/ui/panes/networklog_pane.go internal/ui/panes/networklog_pane_test.go
  git commit -m "fix(networklog): pendingDecisions cross-tick fix; Title Case column headers"
  ```

---

### Task 11: Update Page B preset grid

**Files:**
- Modify: `internal/ui/layout/presets.go`
- Modify: `internal/ui/layout/presets_test.go`

- [ ] **Step 1: Write failing test**

  In `presets_test.go`, add:

  ```go
  func TestPresetNerdStatus_HasFivePanes(t *testing.T) {
      preset := layout.PresetNerdStatus
      require.Len(t, preset.Visible, 5, "Page B should have 5 visible panes")
      assert.True(t, preset.Visible[layout.PaneNowPlaying])
      assert.True(t, preset.Visible[layout.PaneGatewayHealth])
      assert.True(t, preset.Visible[layout.PanePollingTraffic])
      assert.True(t, preset.Visible[layout.PaneGatewayLive])
      assert.True(t, preset.Visible[layout.PaneNetworkLog])
  }

  func TestPresetNerdStatus_GridHasThreeRows(t *testing.T) {
      require.Len(t, layout.PresetNerdStatus.Grid, 3)
      // Row 2: three panes with weights 1, 1, 2
      row2 := layout.PresetNerdStatus.Grid[1]
      require.Len(t, row2.Cells, 3)
      assert.Equal(t, layout.PaneGatewayHealth, row2.Cells[0].PaneID)
      assert.Equal(t, layout.PanePollingTraffic, row2.Cells[1].PaneID)
      assert.Equal(t, layout.PaneGatewayLive, row2.Cells[2].PaneID)
      assert.Equal(t, 2, row2.Cells[2].WidthWeight)
  }
  ```

- [ ] **Step 2: Run test to verify it fails**

  ```bash
  go test ./internal/ui/layout/... -run TestPresetNerdStatus -v
  ```

- [ ] **Step 3: Update PresetNerdStatus in presets.go**

  Replace the existing `PresetNerdStatus` variable:

  ```go
  // PresetNerdStatus shows NowPlaying compact strip with gateway diagnostics below.
  var PresetNerdStatus = Preset{
      Name: "Nerd Status",
      Visible: map[PaneID]bool{
          PaneNowPlaying: true, PaneGatewayHealth: true,
          PanePollingTraffic: true, PaneGatewayLive: true, PaneNetworkLog: true,
      },
      Grid: []Row{
          {HeightWeight: 1, Cells: []Cell{{PaneNowPlaying, 1}}},
          {HeightWeight: 3, Cells: []Cell{
              {PaneGatewayHealth, 1}, {PanePollingTraffic, 1}, {PaneGatewayLive, 2},
          }},
          {HeightWeight: 2, Cells: []Cell{{PaneNetworkLog, 1}}},
      },
  }
  ```

- [ ] **Step 4: Run tests to verify pass**

  ```bash
  go test ./internal/ui/layout/... -v
  ```

- [ ] **Step 5: Commit**

  ```bash
  git add internal/ui/layout/presets.go internal/ui/layout/presets_test.go
  git commit -m "feat(layout): update Page B preset — 4 panes replacing RequestFlow"
  ```

---

### Task 12: Wire new panes in app.go, update border.go, delete RequestFlow files

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/handlers.go`
- Modify: `internal/ui/layout/border.go`
- Delete: `internal/ui/panes/requestflow_pane.go`
- Delete: `internal/ui/panes/requestflow_pane_test.go`
- Delete: `internal/ui/panes/requestflow_boxed.go`
- Delete: `internal/ui/panes/requestflow_boxed_test.go`
- Delete: `internal/ui/panes/requestflow_replay.go`
- Delete: `internal/ui/panes/requestflow_replay_test.go`

- [ ] **Step 1: Update border.go — replace PaneRequestFlow case**

  In `PaneBorderColor()` switch, remove:
  ```go
  case PaneRequestFlow:
      return t.PaneBorderRequestFlow()
  ```

  Add cases for the three new panes:
  ```go
  case PaneGatewayHealth, PanePollingTraffic, PaneGatewayLive:
      return t.PaneBorderRequestFlow()
  ```

- [ ] **Step 2: Update app.go struct — remove RequestFlow, add three new panes**

  In the `App` struct, remove any `requestFlowPane` field if it exists as a typed field (it may be accessed only via the `panes` map and the `RequestFlowPane()` accessor). Add nothing to the struct itself — the panes are wired via the `panes` map.

  In `New()` (or wherever the pane map is built), replace:
  ```go
  requestFlowPane := panes.NewRequestFlowPane(s, t)
  ```

  With:
  ```go
  gatewayHealthPane := panes.NewGatewayHealthPane(s, t)
  pollingTrafficPane := panes.NewPollingTrafficPane(s, t)
  gatewayLivePane := panes.NewGatewayLivePane(s, t)
  ```

  In the pane map initialization, replace:
  ```go
  layout.PaneRequestFlow: requestFlowPane,
  ```

  With:
  ```go
  layout.PaneGatewayHealth:   gatewayHealthPane,
  layout.PanePollingTraffic:  pollingTrafficPane,
  layout.PaneGatewayLive:     gatewayLivePane,
  ```

- [ ] **Step 3: Update handlers.go — reroute PollingSnapshotMsg**

  Find the block (around line 389–407) that sends `PollingSnapshotMsg` to `RequestFlowPane`:
  ```go
  if rfp := a.RequestFlowPane(); rfp != nil {
      updated, _ := rfp.Update(pollingSnapshot)
      if p, ok := updated.(*panes.RequestFlowPane); ok {
          a.panes[layout.PaneRequestFlow] = p
      }
  }
  ```

  Replace with:
  ```go
  if ptp, ok := a.panes[layout.PanePollingTraffic]; ok {
      updated, _ := ptp.Update(pollingSnapshot)
      a.panes[layout.PanePollingTraffic] = updated
  }
  ```

  Remove any other blocks in handlers.go that reference `PaneRequestFlow` or `RequestFlowPane()`. These include the TickMsg forwarding to RequestFlowPane and the viz.TickMsg handler if present.

- [ ] **Step 4: Remove the RequestFlowPane() accessor from app.go**

  Delete the exported accessor `func (a *App) RequestFlowPane() *panes.RequestFlowPane { ... }` (around line 802). Also delete `func (a *App) NetworkLogPane() *panes.NetworkLogPane { ... }` only if no tests use it. If test files import it, keep it or update tests.

  > If existing app tests call `a.RequestFlowPane()`, update those tests to use `a.GatewayLivePane()` or remove the test if it tested RequestFlow-specific behaviour.

  Add new accessors for testing:
  ```go
  // GatewayHealthPane returns the GatewayHealthPane from the panes map.
  func (a *App) GatewayHealthPane() *panes.GatewayHealthPane {
      p, ok := a.panes[layout.PaneGatewayHealth]
      if !ok {
          return nil
      }
      if ghp, ok := p.(*panes.GatewayHealthPane); ok {
          return ghp
      }
      return nil
  }

  // PollingTrafficPane returns the PollingTrafficPane from the panes map.
  func (a *App) PollingTrafficPane() *panes.PollingTrafficPane {
      p, ok := a.panes[layout.PanePollingTraffic]
      if !ok {
          return nil
      }
      if ptp, ok := p.(*panes.PollingTrafficPane); ok {
          return ptp
      }
      return nil
  }

  // GatewayLivePane returns the GatewayLivePane from the panes map.
  func (a *App) GatewayLivePane() *panes.GatewayLivePane {
      p, ok := a.panes[layout.PaneGatewayLive]
      if !ok {
          return nil
      }
      if glp, ok := p.(*panes.GatewayLivePane); ok {
          return glp
      }
      return nil
  }
  ```

- [ ] **Step 5: Delete the six RequestFlow files**

  ```bash
  git rm internal/ui/panes/requestflow_pane.go \
         internal/ui/panes/requestflow_pane_test.go \
         internal/ui/panes/requestflow_boxed.go \
         internal/ui/panes/requestflow_boxed_test.go \
         internal/ui/panes/requestflow_replay.go \
         internal/ui/panes/requestflow_replay_test.go
  ```

  After deletion, `humanAge` and `humanInterval` are gone with the file — no callers remain.
  `cacheAge` in `polling_traffic_pane.go` stays as-is; do **not** rename it to `humanAge`
  because the two functions have different output formats (`cacheAge` omits the "ago" suffix).

- [ ] **Step 6: Build to verify clean compile**

  ```bash
  go build ./...
  ```

  Expected: no errors. Fix any remaining reference to `PaneRequestFlow` or `RequestFlowPane`.

- [ ] **Step 7: Run full test suite**

  ```bash
  go test ./...
  ```

  Expected: all green. Any tests that tested `RequestFlowPane` behaviour now also pass due to the file deletions.

- [ ] **Step 8: Commit**

  ```bash
  git add internal/app/app.go internal/app/handlers.go internal/ui/layout/border.go
  git commit -m "feat(app): wire GatewayHealth/PollingTraffic/GatewayLive; delete RequestFlowPane"
  ```

---

### Task 13: Full CI gate

- [ ] **Step 1: Run the full CI gate**

  ```bash
  make ci
  ```

  Expected: lint + tests + 80% coverage — all green.

- [ ] **Step 2: Fix any coverage gap**

  If coverage drops below 80%, identify uncovered lines with:

  ```bash
  make test-coverage
  ```

  Add focused tests for uncovered paths in the new pane files. Common gaps:
  - `GatewayHealthPane.View()` paths for Warning state (≤2 tokens, all slots full)
  - `PollingTrafficPane.View()` path for `fetchedAt.IsZero()` (never fetched)
  - `GatewayLivePane.handleKey()` branches for Esc with active filter vs. with committed query

- [ ] **Step 3: Update DESIGN.md Page B section**

  In `docs/DESIGN.md`, update the Page B section to reflect the new 4-pane layout. Remove references to `RequestFlowPane`. Add toggle key table for Page B.

- [ ] **Step 4: Final commit**

  ```bash
  git add docs/DESIGN.md
  git commit -m "docs(design): update Page B spec — 4-pane layout, remove RequestFlowPane"
  ```

---

## Self-Review Checklist

- [ ] `PollingSnapshotMsg` defined once in `messages.go`, not in any deleted file
- [ ] `humanAge` / `pollingHumanInterval` live in exactly one file after RequestFlow deletion
- [ ] All three keybinding locations updated in Task 4 commit
- [ ] `TogglePane` guard uses `PaneNetworkLog` (not `PaneRequestFlow`)
- [ ] `border.go` switch handles all new PaneIDs
- [ ] `PresetNerdStatus` has no reference to `PaneRequestFlow`
- [ ] `GatewayLivePane` Esc priority: committed filter cleared first, then scroll reset
- [ ] `NetworkLogPane.pendingDecisions` initialized in constructor, `delete()` called after use
- [ ] `make ci` passes
