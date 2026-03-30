# Feature 38 — Notification & Staleness Hardening

> **Feature:** Fix robustness issues in the BubbleUp notification system and staleness
> tracking: assertion safety, Init() batching, fetchedAt nil guards, stats stamping,
> TOCTOU sentinel, and staleness gate data delivery.

## Context

PR reviews of Feature 31 (Notifications) and Feature 32 (Staleness) identified issues
in error handling and data integrity. These fixes harden both systems.

**Source:** `docs/issues.md` — PR #36 issues 2-4; PR #37 issues 6-8, 10

**Depends on:** Feature 36

---

## Task 1: Fix alerts.Update() type assertion and alerts.Init() return value

**Problem 1:** `app.go` line ~467 does `if am, ok := updatedAlerts.(bubbleup.AlertModel); ok`
but if the type assertion fails, alert state silently stops updating.

**Problem 2:** `app.go` line ~366 discards `alerts.Init()` return with `_ =`. A future
BubbleUp upgrade could return a setup command.

**Fix:**

1. For the type assertion, add a defensive comment explaining why this is safe
   (BubbleUp's AlertModel.Update always returns AlertModel). If assertion fails,
   it indicates a library bug — document this:
   ```go
   updatedAlerts, alertCmd := a.alerts.Update(msg)
   // BubbleUp.AlertModel.Update always returns AlertModel. If this assertion
   // fails, it indicates a BubbleUp library bug — alert state freezes but
   // the app continues working.
   if am, ok := updatedAlerts.(bubbleup.AlertModel); ok {
       a.alerts = am
   }
   ```

2. For `alerts.Init()`, batch the return value into commands:
   ```go
   alertsInitCmd := a.alerts.Init()
   cmds := []tea.Cmd{
       // ... existing commands ...
       alertsInitCmd,
   }
   return a, tea.Batch(cmds...)
   ```

**Files:**
- Modify: `internal/app/app.go` — Init() and Update() alerts wiring

**Tests:**
- Unit: verify Init() returns batched commands including alerts init
- Unit: verify Update() handles alerts update (existing test should cover)

**Commit:** `fix(app): batch alerts.Init() and document assertion safety`

---

## Task 2: Add alert type registration validation

**Problem:** `NewNotifications` in `internal/ui/components/notifications.go` (lines 51-55)
calls `RegisterNewAlertType` 5 times without checking return values. Invalid theme color
strings could cause silent registration failure.

**Fix:**

1. Check the BubbleUp API — if `RegisterNewAlertType` returns an error, handle it.
   If it doesn't return an error (void function), add a test that verifies all 5
   alert types are usable after registration by calling `NewAlertCmd` for each.

2. Add a test function:
   ```go
   func TestNewNotifications_AllAlertTypesRegistered(t *testing.T) {
       th := theme.NewBlackTheme()
       model := components.NewNotifications(th)

       alertTypes := []string{"success", "error", "warning", "info", "ratelimit"}
       for _, key := range alertTypes {
           cmd := model.NewAlertCmd(key, "test message")
           assert.NotNil(t, cmd, "alert type %q should produce a command", key)
       }
   }
   ```

**Files:**
- Modify: `internal/ui/components/notifications.go` — add validation if API supports it
- Modify: `internal/ui/components/notifications_test.go` — add registration test

**Tests:**
- Unit: all 5 alert types produce non-nil commands after registration

**Commit:** `test(ui): validate all alert type registrations`

---

## Task 3: Guard fetchedAt stamping on nil/empty data

**Problem:** All `Set*()` methods in `internal/state/store.go` unconditionally stamp
`fetchedAt = time.Now()` even when data is nil. When a nil-client fallback returns
empty data (Feature 36 adds errNilClient, but existing nil data paths remain),
the TTL prevents retries for the full duration.

**Fix:**

1. In each Set method, only stamp fetchedAt when data is non-nil/non-empty:
   ```go
   func (s *Store) SetPlaylists(playlists []domain.SimplePlaylist) {
       s.mu.Lock()
       defer s.mu.Unlock()
       s.playlists = playlists
       if len(playlists) > 0 {
           s.playlistsFetchedAt = time.Now()
       }
   }
   ```

2. Apply to: `SetPlaylists`, `SetSavedAlbums`, `SetLikedTracks`, `SetRecentlyPlayed`,
   `SetDevicesFetchedAt` (guard on the caller side)

3. **Exception:** `SetPlaybackState` — nil state is valid (204 = nothing playing),
   so it should still stamp fetchedAt. No change needed there.

**Files:**
- Modify: `internal/state/store.go` — guard 5 Set methods

**Tests:**
- Unit: verify SetPlaylists(nil) does NOT update fetchedAt
- Unit: verify SetPlaylists(validData) DOES update fetchedAt
- Unit: verify SetPlaybackState(nil) still stamps (204 is valid)

**Commit:** `fix(state): guard fetchedAt stamping on nil/empty data`

---

## Task 4: Fix stats double-stamping

**Problem:** Both `SetTopTracks()` and `SetTopArtists()` independently stamp
`statsFetchedAt[range]` in store.go. In the `StatsLoadedMsg` handler (app.go lines 601-603),
both are called sequentially. If only one call succeeds (partial data), the range appears
fresh despite incomplete data.

**Fix:**

1. Remove `statsFetchedAt` stamping from `SetTopTracks()` and `SetTopArtists()`
2. Add a new `StampStatsFetchedAt(timeRange string)` method:
   ```go
   func (s *Store) StampStatsFetchedAt(timeRange string) {
       s.mu.Lock()
       defer s.mu.Unlock()
       s.statsFetchedAt[timeRange] = time.Now()
   }
   ```
3. In the `StatsLoadedMsg` handler in `app.go`, stamp ONCE after both setters:
   ```go
   if m.TimeRange != "" {
       a.store.SetTopTracks(m.TimeRange, m.TopTracks)
       a.store.SetTopArtists(m.TimeRange, m.TopArtists)
       a.store.StampStatsFetchedAt(m.TimeRange)
   }
   ```

**Files:**
- Modify: `internal/state/store.go` — remove stamps from setters, add StampStatsFetchedAt
- Modify: `internal/app/app.go` — call StampStatsFetchedAt after both setters

**Tests:**
- Unit: verify SetTopTracks does NOT stamp statsFetchedAt
- Unit: verify StampStatsFetchedAt updates the timestamp
- Unit: verify StatsStale returns true before StampStatsFetchedAt is called

**Commit:** `fix(state): stamp statsFetchedAt once after both track and artist setters`

---

## Task 5: Add fetching sentinel for TOCTOU race

**Problem:** Between the staleness check and the async fetch completion, duplicate
`FetchPlaylistsRequestMsg` (and similar) can pass the staleness gate. No "fetching"
sentinel prevents this.

**Fix:**

1. Add boolean sentinel fields to Store:
   ```go
   playlistsFetching bool
   albumsFetching    bool
   likedFetching     bool
   recentFetching    bool
   statsFetching     map[string]bool  // keyed by time range
   devicesFetching   bool
   ```

2. Add methods `SetFetching(domain string, fetching bool)` or domain-specific:
   ```go
   func (s *Store) SetPlaylistsFetching(f bool) { s.mu.Lock(); s.playlistsFetching = f; s.mu.Unlock() }
   func (s *Store) PlaylistsFetching() bool { s.mu.RLock(); defer s.mu.RUnlock(); return s.playlistsFetching }
   ```

3. In Update() staleness gates, check fetching before dispatching:
   ```go
   case panes.FetchPlaylistsRequestMsg:
       if m.Offset == 0 && (!a.store.PlaylistsStale() || a.store.PlaylistsFetching()) {
           return a, nil
       }
       a.store.SetPlaylistsFetching(true)
       return a, a.buildFetchPlaylistsCmd(m.Offset)
   ```

4. Clear fetching flag in the loaded message handler:
   ```go
   case panes.LibraryLoadedMsg:
       a.store.SetPlaylistsFetching(false)
       // ... rest of handler
   ```

**Files:**
- Modify: `internal/state/store.go` — add fetching fields and methods
- Modify: `internal/app/app.go` — set/clear fetching in handlers

**Tests:**
- Unit: verify fetching=true prevents duplicate fetch dispatch
- Unit: verify fetching cleared on loaded message
- Unit: verify fetching does not block paginated fetches (Offset > 0)

**Commit:** `fix(state): add fetching sentinels to prevent TOCTOU duplicate fetches`

---

## Task 6: Send cached data when staleness gate blocks

**Problem:** When `FetchPlaylistsRequestMsg` is blocked by the staleness gate (TTL not
expired), `return a, nil` swallows the request. LibraryPane's `Init()` expects a
`LibraryLoadedMsg` to trigger auto-expand of sections.

**Fix:**

1. When the staleness gate blocks a request with Offset=0, send a synthetic loaded
   message with cached data so the pane can initialize:
   ```go
   case panes.FetchPlaylistsRequestMsg:
       if m.Offset == 0 && !a.store.PlaylistsStale() {
           // Data is fresh — send cached playlists so pane can initialize.
           cached := a.store.Playlists()
           return a, func() tea.Msg {
               return panes.LibraryLoadedMsg{Items: cached, Offset: 0}
           }
       }
       return a, a.buildFetchPlaylistsCmd(m.Offset)
   ```

2. Apply same pattern for albums, liked tracks, recently played, and stats if
   those staleness gates also block Init() requests.

**Files:**
- Modify: `internal/app/app.go` — staleness gate handlers

**Tests:**
- Unit: verify fresh playlists still send LibraryLoadedMsg with cached data
- Unit: verify stale playlists trigger API fetch

**Commit:** `fix(app): send cached data when staleness gate blocks pane Init()`

---

## Task 7: Update issues.md

**Fix:** Mark PR #36 issues 2-4 and PR #37 issues 6-8, 10 as resolved.

**Files:**
- Modify: `docs/issues.md`

**Commit:** `docs: mark notification and staleness hardening issues as resolved`

---

## Verification

```bash
# Fetching sentinels exist
grep -n 'Fetching' internal/state/store.go
# Expected: multiple matches

# Stats stamped once
grep -n 'statsFetchedAt\[' internal/state/store.go
# Expected: only in StampStatsFetchedAt method

# Cached data sent on fresh requests
grep -n 'Data is fresh' internal/app/app.go
# Expected: matches in staleness gate handlers

make ci
# Expected: Full pass
```

---

*Depends on: Feature 36*
*Blocks: Feature 39*
