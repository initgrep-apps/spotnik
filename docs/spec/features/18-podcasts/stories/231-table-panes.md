---
title: "Table panes (ShowEpisodes + FollowedShows + SavedEpisodes)"
feature: 18-podcasts
status: open
---

## Background

Three table-based panes for the podcasts page. Each embeds `*TableBasedPane`
(following the same pattern as PlaylistsPane, AlbumTracksPane, etc.) with
pane-specific columns, filtering, and pagination.

## Design

Common patterns (shared by all three panes):

- Embed `*TableBasedPane`
- Use `HandleFilterKey()` for filter routing
- `SetSize()` / `SetFocused()` propagated to table + filter
- `SetTheme()` rebuilds table with border token
- Filtering: `f` toggles filter bar, `Esc` clears + deactivates, `Enter` commits
- Navigation: `j`/`k` or `↓`/`↑` to move, `Enter` to select
- Playing indicator: `▶` replaces the first column value on currently-playing row
- Empty state: `EmptyState` component with primary text + hint
- Unplayable episodes: `⊘` glyph in icon column, dimmed text, `Enter` → toast

### ShowEpisodesPane (`internal/ui/panes/showepisodes.go` — new)

Shows episodes for the currently selected show. Reads show data from
`store.ShowEpisodes()` and metadata from `store.SelectedShowID()` /
`store.SelectedShow()`.

```go
type ShowEpisodesPane struct {
	*TableBasedPane
	store          state.StateReader
	theme          theme.Theme
	loadedEpisodes []domain.Episode
	episodesTotal  int
	hasNext        bool
	pendingOffset  int
}
```

**Pane metadata:**
- `ID()` → `layout.PaneShowEpisodes`
- `Title()` → dynamic: `"Show Name (N eps)"` when show selected, `"Show Episodes"` when no show
- `ToggleKey()` → `2`
- `Actions()` → `[f: filter]`

**Columns** (flex factor 18):

| Key | Header | Flex | Color | Notes |
|-----|--------|------|-------|-------|
| `"index"` | `"#"` | 1 | `ColumnIndex()` | 1-based; `▶` for currently-playing |
| `"title"` | `"Title"` | 9 | `ColumnPrimary()` | Episode name, truncated |
| `"released"` | `"Released"` | 4 | `ColumnSecondary()` | Date string, e.g. `"May 29"` |
| `"duration"` | `"Duration"` | 3 | `ColumnTertiary()` | Formatted `"m:ss"` or `"h:mm:ss"` |
| `"icon"` | `""` | 1 | `ColumnSecondary()` | `⊘` if `!is_playable`, `▶` if resume > 0 |

**Filtering**: `name` field, placeholder `"filter episodes..."`

**Pagination**: Embedded episodes from `GET /shows/{id}` provide first page
(up to 50). Additional pages fetched via `GET /shows/{id}/episodes?offset=N`
when cursor nears end. Prefetch within 5 rows of end.

**Interaction**: `Enter` on playable row → emit `PlayEpisodeMsg{EpisodeURI, PlaylistURI}`.
`Enter` on unplayable → direct toast via `a.alerts.NewAlertCmd(type, msg)`.

**Edge cases:**
- No current show + no last-selected show: empty state `"No show selected"` + hint
- Episodes from embedded list have no `show` field — fill from context
- `duration_ms: 0` → display `"—"` instead of `"0:00"`
- `is_playable: false` → `⊘` glyph, dimmed text, `Enter` shows toast

### FollowedShowsPane (`internal/ui/panes/followedshows.go` — new)

Lists the user's saved/followed shows. Reads from `store.FollowedShows()`.

```go
type FollowedShowsPane struct {
	*TableBasedPane
	store state.StateReader
	theme theme.Theme
}
```

**Pane metadata:**
- `ID()` → `layout.PaneFollowedShows`
- `Title()` → static `"Followed Shows"`
- `ToggleKey()` → `3`
- `Actions()` → `[f: filter]`

**Columns** (flex factor 21):

| Key | Header | Flex | Color | Notes |
|-----|--------|------|-------|-------|
| `"index"` | `"#"` | 1 | `ColumnIndex()` | 1-based |
| `"show"` | `"Show"` | 10 | `ColumnPrimary()` | `show.name` |
| `"publisher"` | `"Publisher"` | 6 | `ColumnSecondary()` | `show.publisher` |
| `"episodes"` | `"Eps"` | 3 | `ColumnTertiary()` | `total_episodes` as number |
| `"media"` | `""` | 1 | `ColumnSecondary()` | Glyph: `♫` audio, `🎬` mixed/video |

**Media type glyphs**: `♪` for `media_type == "audio"`, `♫` for `"mixed"`,
`🎬` for `"video"`. (Glyph catalogue: use `uikit.GlyphMusicNote`,
`uikit.GlyphShuffle`, `uikit.GlyphVideo` from the project's glyph system.)

**Filtering**: `show.name` + `show.publisher`, placeholder `"filter shows..."`

**Pagination**: Standard API pagination via `next` URL. Prefetch within 10 rows
of end. `showsFetching` boolean sentinel.

**Interaction**: `Enter` on a row → if `show.id != store.SelectedShowID()`,
emit `SelectedShowChangedMsg{ShowID: show.ID}`. No-op if same show.

**Empty state**: Text `"No followed shows"`, hint `"Search for shows with /"`.

### SavedEpisodesPane (`internal/ui/panes/savedepisodes.go` — new)

User's saved/bookmarked episodes. Reads from `store.SavedEpisodes()`.

```go
type SavedEpisodesPane struct {
	*TableBasedPane
	store state.StateReader
	theme theme.Theme
}
```

**Pane metadata:**
- `ID()` → `layout.PaneSavedEpisodes`
- `Title()` → static `"Saved Episodes"`
- `ToggleKey()` → `4`
- `Actions()` → `[f: filter]`

**Columns** (flex factor 23):

| Key | Header | Flex | Color | Notes |
|-----|--------|------|-------|-------|
| `"index"` | `"#"` | 1 | `ColumnIndex()` | 1-based; `▶` for currently-playing |
| `"episode"` | `"Episode"` | 9 | `ColumnPrimary()` | Episode `name`, truncated |
| `"show"` | `"Show"` | 6 | `ColumnSecondary()` | `show.name` |
| `"saved"` | `"Saved"` | 3 | `ColumnTertiary()` | Relative date from `added_at` |
| `"duration"` | `"Duration"` | 3 | `ColumnTertiary()` | Formatted `"m:ss"` or `"h:mm:ss"` |
| `"icon"` | `""` | 1 | `ColumnSecondary()` | `⊘` if `!is_playable`, `▶` if resume > 0 |

**Filtering**: `name` + `show.name`, placeholder `"filter episodes..."`

**Pagination**: Standard API pagination. Prefetch within 10 rows. `savedFetching`
boolean sentinel.

**Interaction**: `Enter` on playable → `PlayEpisodeMsg`. Unplayable → toast.

**Empty state**: Text `"No saved episodes"`, no hint.

### Shared table setup

Each pane's `buildRows()` method follows the same pattern as
`playlists_pane.go`:
1. Get data from store
2. Apply local filter string (case-insensitive substring)
3. Build `[]table.Row` with `RowData` keyed by column key
4. Apply style: `ColumnStyle(row)` for standard rows, dimmed styles for
   unplayable rows
5. Set playing indicator on currently-playing row

`SetTheme(th theme.Theme)` destroys and recreates the `table.Model` with new
border color and column styles.

### Tests

Each pane needs identity tests:

```go
func TestShowEpisodesPane_ID(t *testing.T)   // PaneShowEpisodes
func TestShowEpisodesPane_ToggleKey(t *testing.T) // 2
func TestFollowedShowsPane_ID(t *testing.T)  // PaneFollowedShows
func TestFollowedShowsPane_ToggleKey(t *testing.T) // 3
func TestSavedEpisodesPane_ID(t *testing.T)   // PaneSavedEpisodes
func TestSavedEpisodesPane_ToggleKey(t *testing.T) // 4
func TestSavedEpisodesPane_EnterOnPlayableEpisode(t *testing.T)
func TestSavedEpisodesPane_EnterOnUnplayableEpisode(t *testing.T)
```

## Acceptance Criteria

- [ ] ShowEpisodesPane compiles with ID/ToggleKey/Title/Actions
- [ ] FollowedShowsPane compiles with ID/ToggleKey/Title/Actions
- [ ] SavedEpisodesPane compiles with ID/ToggleKey/Title/Actions
- [ ] All three panes embed `*TableBasedPane` and propagate SetSize/SetFocused/SetTheme
- [ ] ShowEpisodes shows dynamic title with show name + episode count
- [ ] FollowedShows shows 5 columns with correct headers
- [ ] SavedEpisodes shows 6 columns with correct headers
- [ ] Filter bar activates/deactivates on `f`/`Esc`
- [ ] `Enter` on playable episode emits `PlayEpisodeMsg`
- [ ] `Enter` on unplayable episode renders toast (not playback)
- [ ] `Enter` on show in FollowedShows emits `SelectedShowChangedMsg`
- [ ] Playing indicator `▶` shows on currently-playing row
- [ ] `⊘` glyph and dimmed text on unplayable rows
- [ ] `duration_ms: 0` displays `"—"` not `"0:00"`
- [ ] Empty states render correctly
- [ ] All tests pass
- [ ] `go build ./internal/ui/panes/...` passes

## Tasks

**ShowEpisodesPane:**

- [ ] Create `internal/ui/panes/showepisodes.go` with `ShowEpisodesPane` embedding `*TableBasedPane`
- [ ] Add test: `TestShowEpisodesPane_ID` — verify ID == `PaneShowEpisodes` — in `showepisodes_test.go`
- [ ] Add test: `TestShowEpisodesPane_ToggleKey` — verify ToggleKey == 2 — in `showepisodes_test.go`
- [ ] Implement `Title()` returning `"Show Name (N eps)"` when show selected, `"Show Episodes"` when no show
- [ ] Add test: `TestShowEpisodesPane_Title_Dynamic` — verify title contains show name + count — in `showepisodes_test.go`
- [ ] Add test: `TestShowEpisodesPane_Title_Default` — verify fallback title — in `showepisodes_test.go`
- [ ] Implement `buildRows()` with 5 columns (index, title, released, duration, icon)
- [ ] Implement `Enter` handling: playable → emit `PlayEpisodeMsg`, unplayable → toast
- [ ] Add test: `TestShowEpisodesPane_EnterPlayable` — triggers `PlayEpisodeMsg` — in `showepisodes_test.go`
- [ ] Add test: `TestShowEpisodesPane_EnterUnplayable` — triggers toast, not playback — in `showepisodes_test.go`
- [ ] Implement filter: `f` toggles filter bar, `Esc` clears, placeholder `"filter episodes..."`
- [ ] Implement pagination: prefetch within 5 rows of end
- [ ] Handle empty state: "No show selected" with hint text

**FollowedShowsPane:**

- [ ] Create `internal/ui/panes/followedshows.go` with `FollowedShowsPane` embedding `*TableBasedPane`
- [ ] Add test: `TestFollowedShowsPane_ID` — verify ID == `PaneFollowedShows` — in `followedshows_test.go`
- [ ] Add test: `TestFollowedShowsPane_ToggleKey` — verify ToggleKey == 3 — in `followedshows_test.go`
- [ ] Implement `Title()` returning static `"Followed Shows"`
- [ ] Implement `buildRows()` with 5 columns (index, show, publisher, episodes, media glyph)
- [ ] Implement `Enter` handling: emit `SelectedShowChangedMsg{ShowID}` if different from current
- [ ] Add test: `TestFollowedShowsPane_EnterSelectsShow` — triggers `SelectedShowChangedMsg` — in `followedshows_test.go`
- [ ] Add test: `TestFollowedShowsPane_EnterSameShow` — no-op when same show — in `followedshows_test.go`
- [ ] Implement filter: `f` toggles, placeholder `"filter shows..."`
- [ ] Handle empty state: "No followed shows" + hint
- [ ] Verify media type glyphs: `♪` for audio, `♫` for mixed, `🎬` for video

**SavedEpisodesPane:**

- [ ] Create `internal/ui/panes/savedepisodes.go` with `SavedEpisodesPane` embedding `*TableBasedPane`
- [ ] Add test: `TestSavedEpisodesPane_ID` — verify ID == `PaneSavedEpisodes` — in `savedepisodes_test.go`
- [ ] Add test: `TestSavedEpisodesPane_ToggleKey` — verify ToggleKey == 4 — in `savedepisodes_test.go`
- [ ] Implement `Title()` returning static `"Saved Episodes"`
- [ ] Implement `buildRows()` with 6 columns (index, episode, show, saved, duration, icon)
- [ ] Implement `Enter` handling: same as ShowEpisodes (playable/unplayable)
- [ ] Add test: `TestSavedEpisodesPane_EnterPlayable` — triggers `PlayEpisodeMsg` — in `savedepisodes_test.go`
- [ ] Add test: `TestSavedEpisodesPane_EnterUnplayable` — triggers toast — in `savedepisodes_test.go`
- [ ] Implement filter: `f` toggles, placeholder `"filter episodes..."`
- [ ] Handle empty state: "No saved episodes"
- [ ] Handle `duration_ms: 0` → display `"—"`

**Shared edge cases:**

- [ ] Add test: `TestShowEpisodesPane_DurationZero` — displays `"—"` — in `showepisodes_test.go`
- [ ] Add test: `TestSavedEpisodesPane_DurationZero` — displays `"—"` — in `savedepisodes_test.go`
- [ ] Add test: `TestShowEpisodesPane_PlayingIndicator` — `▶` on currently-playing row — in shared test file
- [ ] Add test: `TestShowEpisodesPane_UnplayableGlyph` — `⊘` glyph, dimmed text — in shared test file
- [ ] Add test: `TestShowEpisodesPane_EmptyState` — verify empty state text — in each pane test
- [ ] Add test: `TestFollowedShowsPane_EmptyState` — verify empty state text
- [ ] Add test: `TestSavedEpisodesPane_EmptyState` — verify empty state text
- [ ] Add test: `TestShowEpisodesPane_FilterToggle` — `f` activates filter bar, `Esc` deactivates — in `showepisodes_test.go`
- [ ] Run `go test ./internal/ui/panes/...` — all pass
