---
title: "Fix: PollingTrafficPane missing Stats row"
feature: 14-page-b-redesign
status: done
---

## Background

Feature 15 introduced universal tick-driven polling for all data panes. The
`PollingTrafficPane` now shows an incomplete picture: it has rows for Playlists,
Albums, Liked, and Recent, but omits Stats — even though Stats is polled on the
same tick loop with its own `statsPoll` state and `StatsTTL` (10 min).

The user observes that all panes refresh but the traffic pane doesn't reflect it.

**Additional gap:** `StateReader` interface exposes `FetchedAt` timestamps for the
four library domains but not for Stats. `PollingTrafficPane` needs `StatsFetchedAt`
to render the Stats row without accessing the concrete `*Store` directly.

## Design

### `internal/state/reader.go` — add `StatsFetchedAt`

Add one method to the `StateReader` interface:

```go
// StatsFetchedAt returns the time stats for the given time range were last
// successfully fetched. Returns zero time if never fetched.
StatsFetchedAt(timeRange string) time.Time
```

The concrete implementation already exists in `store.go`
(`func (s *Store) StatsFetchedAt(timeRange string) time.Time`).
The compile-time assertion `var _ StateReader = (*Store)(nil)` will verify this.

### `internal/ui/panes/polling_traffic_pane.go` — add Stats row

Extend the `cacheRows` slice in `View()` with a Stats entry. Stats uses the
`short_term` range (the primary range polled by the tick loop):

```go
cacheRows := []cacheRow{
    {uikit.GlyphQueue,      "Playlists", p.store.PlaylistsFetchedAt(),              state.PlaylistsTTL},
    {uikit.GlyphDoubleNote, "Albums",    p.store.AlbumsFetchedAt(),                 state.AlbumsTTL},
    {uikit.GlyphPinned,     "Liked",     p.store.LikedTracksFetchedAt(),            state.LikedTracksTTL},
    {uikit.GlyphDeadline,   "Recent",    p.store.RecentPlayedFetchedAt(),           state.RecentlyPlayedTTL},
    {uikit.GlyphMusicNote,  "Stats",     p.store.StatsFetchedAt("short_term"),      state.StatsTTL},
}
```

Update the `renderedRows` capacity hint from `5` to `6`:

```go
renderedRows := make([]string, 0, 6)
```

No changes needed to `PollingSnapshotMsg` or the app-side dispatch.

### Tests

Update `TestPollingTrafficPane_View_ContainsAllRows` in
`polling_traffic_pane_test.go` to assert `"Stats"` appears in the rendered view.

Add `TestPollingTrafficPane_View_StatsRow_NeverFetched`: store returns zero time
for `StatsFetchedAt("short_term")` → view contains `"never fetched"` for the
Stats row.

Add `TestPollingTrafficPane_View_StatsRow_Fresh`: store returns `time.Now()` →
view contains `"fresh"` for Stats row.

## Acceptance Criteria

- [ ] `StateReader` interface includes `StatsFetchedAt(timeRange string) time.Time`
- [ ] `PollingTrafficPane.View()` renders a 6th row labelled "Stats"
- [ ] Stats row shows `never fetched`, `fresh`, or stale age — same logic as other cache rows
- [ ] `TestPollingTrafficPane_View_ContainsAllRows` asserts "Stats" in view
- [ ] New tests for never-fetched and fresh states pass
- [ ] `make ci` passes

## Tasks

- [ ] Add `StatsFetchedAt(timeRange string) time.Time` to `StateReader` interface
      — `internal/state/reader.go`
      - test: `go build ./...` → no compile error (concrete Store already implements it)

- [ ] Add Stats cache row to `cacheRows` slice in `PollingTrafficPane.View()`
      — `internal/ui/panes/polling_traffic_pane.go`
      - test: `go test ./internal/ui/panes/ -run TestPollingTraffic -v` → PASS

- [ ] Update `TestPollingTrafficPane_View_ContainsAllRows`; add never-fetched and
      fresh tests — `internal/ui/panes/polling_traffic_pane_test.go`
      - test: `go test ./internal/ui/panes/ -run TestPollingTraffic -v` → PASS

- [ ] `make ci` passes
