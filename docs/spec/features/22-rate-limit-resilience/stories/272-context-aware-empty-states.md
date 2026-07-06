---
title: "Context-Aware Empty States"
feature: 22-rate-limit-resilience
status: done
---

## Background

All 9 table-based panes use identical empty-state logic in `View()`:

```go
if len(store.SomeData()) == 0 && !filter.IsActive() {
    return uikit.EmptyState{Text: "No X", ...}.Render()
}
```

This conflates four distinct states into one message:
1. **Never fetched** — preset was never active, no API call ever made
2. **Fetching** — first API call is in-flight
3. **Fetch failed** — API returned an error (network, 401, 403, 429, etc.)
4. **Genuinely empty** — API returned 200 with zero items

The store already tracks all the necessary state (error fields, fetching sentinels, throttle state) but:
- Podcast error/fetching accessors are missing from `StateReader` (panes can't read them)
- `EmptyState` has no concept of status — only static `Text` + `Hint`
- No pane checks anything beyond `len(store.Data()) == 0`

**Existing infrastructure to leverage:**
- `store.FollowedShowsFetching()`, `store.SavedEpisodesFetching()`, `store.ShowEpisodesFetching()` — exist on `*Store` but not on `StateReader`
- `store.FollowedShowsFetchErr()`, `store.SavedEpisodesFetchErr()`, `store.ShowEpisodesFetchErr()` — exist on `*Store` but not on `StateReader`
- `store.IsThrottled()`, `store.ThrottleRetryAfterSecs()` — already on `StateReader`
- `store.PlaylistsFetching()`, `store.AlbumsFetching()`, `store.LikedFetching()`, `store.RecentFetching()` — already on `StateReader`
- `store.PlaylistsFetchErr()`, `store.AlbumsFetchErr()`, `store.LikedTracksFetchErr()`, `store.RecentPlayedFetchErr()` — exist on `*Store` but not on `StateReader`
- `store.StatsFetching(timeRange)`, `store.StatsError()` — exist on `*Store` but not on `StateReader`

## Design

### 1. `internal/state/reader.go` — expose error + fetching accessors

Add to `StateReader` interface:

```go
// --- Fetch errors (read-only) ---

PlaylistsFetchErr() error
AlbumsFetchErr() error
LikedTracksFetchErr() error
RecentPlayedFetchErr() error
StatsError() error
FollowedShowsFetchErr() error
SavedEpisodesFetchErr() error
ShowEpisodesFetchErr() error

// --- Fetching sentinels (read-only) — podcast + stats ---

FollowedShowsFetching() bool
SavedEpisodesFetching() bool
ShowEpisodesFetching() bool
StatsFetching(timeRange string) bool
```

These methods already exist on `*Store` — just need to be added to the interface. The compile-time assertion `var _ StateReader = (*Store)(nil)` at `reader.go:153` will verify correctness.

### 2. `internal/uikit/empty_state.go` — add status-driven rendering

Add a `Status` field to `EmptyState`:

```go
// EmptyStatus classifies why a pane has no data to display.
type EmptyStatus int

const (
    // EmptyStatusNone means data is genuinely empty (API returned 200, zero items).
    EmptyStatusNone EmptyStatus = iota
    // EmptyStatusNeverFetched means no API call has ever completed for this data.
    EmptyStatusNeverFetched
    // EmptyStatusFetching means the first API call is in-flight.
    EmptyStatusFetching
    // EmptyStatusError means the last API call failed with a non-rate-limit error.
    EmptyStatusError
    // EmptyStatusRateLimited means the gateway is in 429 backoff.
    EmptyStatusRateLimited
)

type EmptyState struct {
    Text   string      // primary message (used as fallback when Status is None)
    Hint   string      // optional secondary text
    Status EmptyStatus // when set, overrides Text with status-driven message
    Width  int
    Height int
    Theme  theme.Theme
}
```

**`Render()` logic:**

When `Status` is set (non-zero), derive `Text` and `Hint` from status instead of using the struct fields directly:

| Status | Text | Hint |
|--------|------|------|
| `EmptyStatusNone` | Use `e.Text` as-is | Use `e.Hint` as-is |
| `EmptyStatusNeverFetched` | `"Loading {category}..."` | (none) |
| `EmptyStatusFetching` | `"Loading {category}..."` | (none) |
| `EmptyStatusError` | `"Unable to load {category}"` | `"Check your connection"` |
| `EmptyStatusRateLimited` | `"Unable to load {category}"` | `"Rate limited — retrying in Ns"` |

The `{category}` is derived from `e.Text` — strip the "No " prefix. E.g., "No followed shows" → "followed shows".

**Rate-limited hint:** Read `e.Theme` — but the retry-after seconds come from the store. Since `EmptyState` is a value type (no store reference), the pane must pass the retry-after into the `Hint` field when setting `Status = EmptyStatusRateLimited`. The pane reads `store.ThrottleRetryAfterSecs()` and formats the hint.

### 3. Pane `View()` updates — all 9 table panes

Each pane's `View()` must check status in this priority order:

```
1. Is filter active? → show filter bar + table (existing behavior, unchanged)
2. Is data non-empty? → show table (existing behavior, unchanged)
3. Is gateway throttled? → EmptyState{Status: EmptyStatusRateLimited, Hint: "Rate limited — retrying in Ns"}
4. Is fetch in-flight? → EmptyState{Status: EmptyStatusFetching}
5. Has data never been fetched? → EmptyState{Status: EmptyStatusNeverFetched}
6. Did last fetch error? → EmptyState{Status: EmptyStatusError}
7. Otherwise → EmptyState{Text: "No X", Hint: "..."} (existing behavior)
```

**Decision helper** — add a method to avoid repeating this logic 9 times:

```go
// internal/uikit/empty_state.go

// PaneEmptyStatus determines the EmptyStatus for a pane based on store state.
// category is the display name (e.g. "followed shows", "saved episodes").
// hasData is true when len(store.Data()) > 0.
// isFetching is the store's *Fetching() accessor result.
// fetchErr is the store's *FetchErr() accessor result.
// neverFetched is true when fetchedAt is zero time.
// isThrottled is store.IsThrottled().
// retryAfterSecs is store.ThrottleRetryAfterSecs().
func PaneEmptyStatus(category string, hasData, isFetching bool, fetchErr error,
    neverFetched, isThrottled bool, retryAfterSecs int) EmptyState {
    if isThrottled {
        return EmptyState{
            Status: EmptyStatusRateLimited,
            Text:   "No " + category,
            Hint:   fmt.Sprintf("Rate limited — retrying in %ds", retryAfterSecs),
            // Width/Height/Theme set by caller
        }
    }
    if isFetching {
        return EmptyState{Status: EmptyStatusFetching, Text: "No " + category}
    }
    if neverFetched {
        return EmptyState{Status: EmptyStatusNeverFetched, Text: "No " + category}
    }
    if fetchErr != nil {
        return EmptyState{Status: EmptyStatusError, Text: "No " + category}
    }
    return EmptyState{Status: EmptyStatusNone, Text: "No " + category}
}
```

Each pane calls this, then sets `Width`, `Height`, `Theme`, and `Hint` (for the genuinely-empty case) before calling `.Render()`.

### 4. Per-pane changes

| Pane | File | category | hasData | isFetching | fetchErr | neverFetched |
|------|------|----------|---------|------------|----------|--------------|
| FollowedShowsPane | `followedshows.go` | `"followed shows"` | `len(store.FollowedShows()) > 0` | `store.FollowedShowsFetching()` | `store.FollowedShowsFetchErr()` | `store.FollowedShowsFetchedAt().IsZero()` |
| SavedEpisodesPane | `savedepisodes.go` | `"saved episodes"` | `len(store.SavedEpisodes()) > 0` | `store.SavedEpisodesFetching()` | `store.SavedEpisodesFetchErr()` | `store.SavedEpisodesFetchedAt().IsZero()` |
| PlaylistsPane | `playlists_pane.go` | `"playlists"` | `len(store.Playlists()) > 0` | `store.PlaylistsFetching()` | `store.PlaylistsFetchErr()` | `store.PlaylistsFetchedAt().IsZero()` |
| AlbumsPane | `albums_pane.go` | `"saved albums"` | `len(store.SavedAlbums()) > 0` | `store.AlbumsFetching()` | `store.AlbumsFetchErr()` | `store.AlbumsFetchedAt().IsZero()` |
| LikedSongsPane | `likedsongs_pane.go` | `"liked songs"` | `len(store.LikedTracks()) > 0` | `store.LikedFetching()` | `store.LikedTracksFetchErr()` | `store.LikedTracksFetchedAt().IsZero()` |
| RecentlyPlayedPane | `recentlyplayed_pane.go` | `"recently played tracks"` | `len(store.RecentlyPlayed()) > 0` | `store.RecentFetching()` | `store.RecentPlayedFetchErr()` | `store.RecentPlayedFetchedAt().IsZero()` |
| TopTracksPane | `toptracks_pane.go` | `"top tracks"` | `len(store.TopTracks(timeRange)) > 0` | `store.StatsFetching(timeRange)` | `store.StatsError()` | `store.StatsFetchedAt(timeRange).IsZero()` |
| TopArtistsPane | `topartists_pane.go` | `"top artists"` | `len(store.TopArtists(timeRange)) > 0` | `store.StatsFetching(timeRange)` | `store.StatsError()` | `store.StatsFetchedAt(timeRange).IsZero()` |
| QueuePane | `queue.go` | `"queue"` | `len(store.Queue()) > 0` | N/A (queue is real-time polled, no sentinel) | N/A | N/A |

**QueuePane special case:** Queue has no fetching sentinel or error field on `StateReader`. It's polled every N seconds via `fetchQueueCmd`. For now, QueuePane keeps its existing simple empty state. A follow-up can add queue error tracking if needed.

**TopTracksPane / TopArtistsPane special case:** These read `timeRange` from their local state (cycled by user). The `StatsFetching` and `StatsError` accessors need the timeRange parameter.

### 5. `EmptyState.Render()` — status-driven text

When `Status != EmptyStatusNone`, `Render()` derives the display text:

```go
func (e EmptyState) Render() string {
    text := e.Text
    hint := e.Hint

    switch e.Status {
    case EmptyStatusNeverFetched, EmptyStatusFetching:
        category := strings.TrimPrefix(e.Text, "No ")
        text = "Loading " + category + "..."
        hint = ""
    case EmptyStatusError:
        category := strings.TrimPrefix(e.Text, "No ")
        text = "Unable to load " + category
        if hint == "" {
            hint = "Check your connection"
        }
    case EmptyStatusRateLimited:
        category := strings.TrimPrefix(e.Text, "No ")
        text = "Unable to load " + category
        // hint is set by caller with retry-after info
    }
    // ... existing centering logic unchanged ...
}
```

## Acceptance Criteria

- [ ] `StateReader` includes `FollowedShowsFetching()`, `SavedEpisodesFetching()`, `ShowEpisodesFetching()`, `StatsFetching(string)`
- [ ] `StateReader` includes `PlaylistsFetchErr()`, `AlbumsFetchErr()`, `LikedTracksFetchErr()`, `RecentPlayedFetchErr()`, `StatsError()`, `FollowedShowsFetchErr()`, `SavedEpisodesFetchErr()`, `ShowEpisodesFetchErr()`
- [ ] `EmptyStatus` type with 5 constants defined in `uikit/empty_state.go`
- [ ] `EmptyState.Status` field added; `Render()` derives text from status
- [ ] `PaneEmptyStatus()` helper function in `uikit/empty_state.go`
- [ ] FollowedShowsPane shows "Loading followed shows..." when fetching, "Unable to load followed shows" on error, "Unable to load followed shows — Rate limited" when throttled
- [ ] SavedEpisodesPane shows same pattern for saved episodes
- [ ] PlaylistsPane, AlbumsPane, LikedSongsPane, RecentlyPlayedPane show same pattern
- [ ] TopTracksPane, TopArtistsPane show same pattern with time-range-aware accessors
- [ ] QueuePane unchanged (no error/fetching tracking on StateReader)
- [ ] Existing "No X" messages unchanged when data is genuinely empty (status=None)
- [ ] Golden files regenerated for all affected panes
- [ ] `make ci` passes

## Tasks

- [ ] Add error + fetching accessors to `StateReader` in `internal/state/reader.go`
      - test: `go build ./...` (compile-time assertion verifies *Store implements StateReader)

- [ ] Add `EmptyStatus` type, constants, `Status` field to `EmptyState`; update `Render()` in `internal/uikit/empty_state.go`
      - test: `TestEmptyState_StatusNeverFetched`, `TestEmptyState_StatusFetching`, `TestEmptyState_StatusError`, `TestEmptyState_StatusRateLimited`, `TestEmptyState_StatusNone`

- [ ] Add `PaneEmptyStatus()` helper in `internal/uikit/empty_state.go`
      - test: `TestPaneEmptyStatus_Throttled`, `TestPaneEmptyStatus_Fetching`, `TestPaneEmptyStatus_NeverFetched`, `TestPaneEmptyStatus_Error`, `TestPaneEmptyStatus_HasData`

- [ ] Update FollowedShowsPane `View()` in `internal/ui/panes/followedshows.go`
      - test: `TestFollowedShowsPane_EmptyState_Throttled`, `TestFollowedShowsPane_EmptyState_Fetching`, `TestFollowedShowsPane_EmptyState_Error`

- [ ] Update SavedEpisodesPane `View()` in `internal/ui/panes/savedepisodes.go`
      - test: `TestSavedEpisodesPane_EmptyState_Throttled`, `TestSavedEpisodesPane_EmptyState_Fetching`, `TestSavedEpisodesPane_EmptyState_Error`

- [ ] Update PlaylistsPane `View()` in `internal/ui/panes/playlists_pane.go`
      - test: `TestPlaylistsPane_EmptyState_Throttled`, `TestPlaylistsPane_EmptyState_Fetching`, `TestPlaylistsPane_EmptyState_Error`

- [ ] Update AlbumsPane `View()` in `internal/ui/panes/albums_pane.go`
      - test: `TestAlbumsPane_EmptyState_Throttled`, `TestAlbumsPane_EmptyState_Fetching`, `TestAlbumsPane_EmptyState_Error`

- [ ] Update LikedSongsPane `View()` in `internal/ui/panes/likedsongs_pane.go`
      - test: `TestLikedSongsPane_EmptyState_Throttled`, `TestLikedSongsPane_EmptyState_Fetching`, `TestLikedSongsPane_EmptyState_Error`

- [ ] Update RecentlyPlayedPane `View()` in `internal/ui/panes/recentlyplayed_pane.go`
      - test: `TestRecentlyPlayedPane_EmptyState_Throttled`, `TestRecentlyPlayedPane_EmptyState_Fetching`, `TestRecentlyPlayedPane_EmptyState_Error`

- [ ] Update TopTracksPane `View()` in `internal/ui/panes/toptracks_pane.go`
      - test: `TestTopTracksPane_EmptyState_Throttled`, `TestTopTracksPane_EmptyState_Fetching`, `TestTopTracksPane_EmptyState_Error`

- [ ] Update TopArtistsPane `View()` in `internal/ui/panes/topartists_pane.go`
      - test: `TestTopArtistsPane_EmptyState_Throttled`, `TestTopArtistsPane_EmptyState_Fetching`, `TestTopArtistsPane_EmptyState_Error`

- [ ] Regenerate golden files: `go test ./... -update`
      - verify: `git diff --stat testdata/` shows only expected golden file changes

- [ ] `make ci` passes
