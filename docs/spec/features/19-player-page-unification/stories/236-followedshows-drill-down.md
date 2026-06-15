---
title: "FollowedShows drill-down (absorb ShowEpisodes)"
feature: 19-player-page-unification
status: done
---

## Background

The `ShowEpisodesPane` (deleted in story 233) was a separate pane that displayed
episodes for the show selected in `FollowedShowsPane`. This story converts
`FollowedShowsPane` from a flat list to a two-level drill-down (same pattern as
PlaylistsPane), absorbing ShowEpisodes functionality directly.

## Design

### Level 1 — Show List (default view)

Unchanged from current FollowedShows behavior:

| Key | Header | FlexFactor | Color Token |
|-----|--------|-----------|-------------|
| `index` | `#` | 1 | `ColumnIndex()` |
| `show` | `Show` | 10 | `ColumnPrimary()` |
| `publisher` | `Publisher` | 6 | `ColumnSecondary()` |
| `episodes` | `Episodes` | 3 | `ColumnTertiary()` |
| `media` | `""` | 1 | `ColumnSecondary()` |

**Enter** on a show row:
- If `show.ID == selectedShowID` → no-op (same show)
- Set `selectedShowID`, `selectedShowName`
- Set `inEpisodeView = true`
- Clear `loadedEpisodes`, reset pagination state
- Set `episodesFetching = true`
- Emit `FetchShowEpisodesRequestMsg{ShowID: show.ID, Offset: 0}`
- Switch focus to episode table

### Level 2 — Episode List (sub-view)

| Key | Header | FlexFactor | Color Token |
|-----|--------|-----------|-------------|
| `index` | `#` | 1 | `ColumnIndex()` |
| `title` | `Title` | 9 | `ColumnPrimary()` |
| `released` | `Released` | 4 | `ColumnSecondary()` |
| `duration` | `Duration` | 3 | `ColumnTertiary()` |
| `icon` | `""` | 1 | `ColumnSecondary()` |

- Border title changes dynamically: `"Followed Shows"` → `"<Show Name> (498 eps)"` with `Esc ← back` hint
- `Enter` on playable episode → `PlayEpisodeMsg{EpisodeURI: ep.URI, PlaylistURI: "spotify:show:" + showID}` → auto-switch to Podcast preset (story 239)
- `Enter` on unplayable episode → toast: `"Episode not available in your market"`
- `Esc` → returns to Level 1, emits `FollowedShowsViewClosedMsg`

### State fields (new, mirror PlaylistsPane pattern)

```go
inEpisodeView    bool
selectedShowID   string
selectedShowName string
episodeTable     *components.Table
loadedEpisodes   []domain.Episode
episodesTotal    int
episodesOffset   int
hasMoreEpisodes  bool
episodesFetching bool
```

Drill-down state persists across preset switches (pane retains its sub-view
state when hidden and shown again). When user switches to a different show
(on Level 1), the episode list resets and reloads.

### Pagination

When cursor is within 5 rows of end AND `hasMoreEpisodes && !episodesFetching`:
emit `FetchShowEpisodesRequestMsg{ShowID: selectedShowID, Offset: episodesOffset}`.

### Title and Actions

```go
func (f *FollowedShowsPane) Title() string {
    if f.inEpisodeView {
        return fmt.Sprintf("%s ── %s (%d eps)", GlyphHRule, f.selectedShowName, f.episodesTotal)
    }
    return "Followed Shows"
}

func (f *FollowedShowsPane) Actions() []layout.Action {
    if f.inEpisodeView {
        return []layout.Action{{Key: "Esc", Label: "back"}}
    }
    return []layout.Action{f.BaseFilterAction()}
}
```

## Files

### Modify

- `internal/ui/panes/followedshows.go` — add episode sub-view state, Level 2
  table, Enter/Esc handlers, `Title()`, `Actions()`, pagination
- `internal/ui/panes/followedshows_test.go` — add drill-down tests
- `internal/ui/panes/messages.go` — add `FollowedShowsViewClosedMsg`
- `internal/app/handlers.go` — handle `FollowedShowsViewClosedMsg` (cancel
  in-flight fetches, clear staleness key)
- `internal/app/commands.go` — ensure `buildFetchShowEpisodesCmd` works for
  FollowedShows drill-down

## Acceptance Criteria

- [ ] Enter on a show row enters episode sub-view (Level 2)
- [ ] Enter on the same show row again is a no-op
- [ ] Episode sub-view shows episodes with title, released date, duration, icon
- [ ] Enter on playable episode starts playback
- [ ] Enter on unplayable episode shows market-restriction toast
- [ ] Esc returns to show list (Level 1)
- [ ] `FollowedShowsViewClosedMsg` emitted on Esc, handler cancels in-flight fetch
- [ ] Pagination prefetches when cursor within 5 rows of end
- [ ] Drill-down state persists across preset switches
- [ ] Switching to a different show resets and reloads episode list
- [ ] Title dynamic: `"Followed Shows"` in Level 1, `"<Show Name> (N eps)"` in Level 2
- [ ] Actions dynamic: filter in Level 1, `Esc ← back` in Level 2
- [ ] `make ci` passes

## Tasks

- [ ] Add drill-down state fields to `FollowedShowsPane`
      - Modify `internal/ui/panes/followedshows.go`: add `inEpisodeView`, `selectedShowID`, `selectedShowName`, `episodeTable`, `loadedEpisodes`, `episodesTotal`, `episodesOffset`, `hasMoreEpisodes`, `episodesFetching` fields
      - test: `TestFollowedShowsPane_InitialState_Level1`
- [ ] Implement Level 1 Enter handler: entering episode sub-view
      - Modify `internal/ui/panes/followedshows.go`: Enter on show row sets `inEpisodeView`, `selectedShowID`, `selectedShowName`, clears episode state, emits `FetchShowEpisodesRequestMsg`
      - test: `TestFollowedShows_EnterShow_EntersEpisodeView`, `TestFollowedShows_EnterSameShow_NoOp`
- [ ] Implement Level 2 rendering: episode table with Title/Released/Duration/Icon columns
      - Modify `internal/ui/panes/followedshows.go`: render episode table in Level 2 with column spec from design
      - test: `TestFollowedShows_EpisodeView_ColumnHeaders`, `TestFollowedShows_EpisodeView_ShowsEpisodeData`
- [ ] Implement Level 2 Enter handler: playable episode starts playback, unplayable shows toast
      - Modify `internal/ui/panes/followedshows.go`: Enter on episode row emits `PlayEpisodeMsg` or shows market-restriction toast
      - test: `TestFollowedShows_EnterPlayableEpisode_Plays`, `TestFollowedShows_EnterUnplayableEpisode_Toasts`
- [ ] Implement Esc handler: return to Level 1
      - Modify `internal/ui/panes/followedshows.go`: Esc clears `inEpisodeView`, emits `FollowedShowsViewClosedMsg`
      - test: `TestFollowedShows_Esc_ReturnsToLevel1`
- [ ] Implement dynamic `Title()` and `Actions()` methods
      - Modify `internal/ui/panes/followedshows.go`: Level 1 returns `"Followed Shows"` + filter action, Level 2 returns show name + `(N eps)` + `Esc ← back`
      - test: `TestFollowedShows_Title_Level1`, `TestFollowedShows_Title_Level2`
- [ ] Implement pagination: prefetch when cursor within 5 rows of end
      - Modify `internal/ui/panes/followedshows.go`: check `hasMoreEpisodes && !episodesFetching` when cursor nears end, emit fetch command
      - test: `TestFollowedShows_Pagination_PrefetchNearEnd`, `TestFollowedShows_Pagination_NoPrefetchWhenAllLoaded`
- [ ] Handle `FetchShowEpisodesRequestMsg` and `FollowedShowsViewClosedMsg` in `handlers.go`
      - Modify `internal/app/handlers.go`: cancel in-flight fetches on close, update episode state on load
      - test: `TestHandler_FetchShowEpisodes_UpdatesState`, `TestHandler_FollowedShowsViewClosed_CancelsFetch`
- [ ] Ensure drill-down state persists across preset switches
      - test: `TestFollowedShows_DrillDownState_PersistsPresetSwitch`
- [ ] Run `make ci` — all lint, tests, and 80% coverage pass