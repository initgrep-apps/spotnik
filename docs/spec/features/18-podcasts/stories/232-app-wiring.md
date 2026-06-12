---
title: "App wiring (commands, routing, handlers, docs)"
feature: 18-podcasts
status: open
---

## Background

The app needs command factories for podcast API calls, routing for the 3-page
cycle and podcast pane toggle keys, message handlers in the root Update loop,
search auto-navigation, polling entries for podcast data, and documentation
updates across multiple docs files.

## Design

### Commands (`internal/app/commands.go`)

Add 4 command factories following the same pattern as existing `build*Cmd`:

```go
func (a *App) buildFetchFollowedShowsCmd() tea.Cmd
```
- Guard: `a.podcastClient == nil` → return nil
- Snapshot store: check `store.FollowedShowsFetching()` sentinel, return nil if true
- Set `store.SetFollowedShowsFetching(true)`
- Return closure calling `a.podcastClient.FollowedShows(ctx, 50, 0)`
- On success: `FollowedShowsLoadedMsg{Items: items}`
- On error: `FollowedShowsLoadedMsg{Err: err}`
- Clear sentinel in handler, not here

```go
func (a *App) buildFetchSavedEpisodesCmd() tea.Cmd
```
Same pattern, calls `a.podcastClient.SavedEpisodes(ctx, 50, 0)`.

```go
func (a *App) buildFetchShowEpisodesCmd(ctx context.Context, showID string) tea.Cmd
```
Same pattern, calls `a.podcastClient.ShowEpisodes(ctx, showID, 50, 0)`.
Sets `store.SetShowEpisodesFetching(true)`.

```go
func (a *App) buildPlayEpisodeCmd(episodeURI, playlistURI string) tea.Cmd
```
Calls `a.client.Play(ctx, spotify.URIOption(episodeURI))`.
If `playlistURI != ""`, also set `spotify.ContextURIOption(playlistURI)`.
Returns `PlaybackRequestCompletedMsg` on success.
On error, captures via `a.alerts.NewAlertCmd("error", msg)`.

### App struct (`internal/app/app.go`)

Add fields:

```go
type App struct {
	// ... existing fields
	podcastClient *api.PodcastClient

	// Podcast panes
	podcastPlayback *panes.PodcastPlaybackPane
	showEpisodes    *panes.ShowEpisodesPane
	followedShows   *panes.FollowedShowsPane
	savedEpisodes   *panes.SavedEpisodesPane
}
```

In `New()`:
- Initialize `podcastClient`: `a.podcastClient = api.NewPodcastClient(a.config.APIBaseURL, accessToken)`
- Initialize 4 pane fields with store + theme
- Add to pane map:
  ```go
  layout.PanePodcastPlayback → a.podcastPlayback
  layout.PaneShowEpisodes    → a.showEpisodes
  layout.PaneFollowedShows   → a.followedShows
  layout.PaneSavedEpisodes   → a.savedEpisodes
  ```

### Routing (`internal/app/routing.go`)

**Page cycle**: The `0` key handler currently cycles Music → Stats → Music.
Change to cycle Music → Podcasts → Stats → Music by calling
`a.layout.TogglePage()` (which was updated in story 229 to 3-cycle).

**Pane toggle**: Handle superscript keys 1–4 on podcasts page:
```go
if a.layout.ActivePage() == layout.PagePodcasts {
    switch msg.Key.String() {
    case "1": a.layout.TogglePane(layout.PanePodcastPlayback)
    case "2": a.layout.TogglePane(layout.PaneShowEpisodes)
    case "3": a.layout.TogglePane(layout.PaneFollowedShows)
    case "4": a.layout.TogglePane(layout.PaneSavedEpisodes)
    }
}
```

**Playback keys**: In `isPlaybackKey()`:
- `Space`, `←`/`→`, `Shift+←`/`Shift+→`, `+`/`-`, `s`, `r` on podcasts page
  route to `a.podcastPlayback.Update(msg)` (same handler dispatch as music page
  sends to `a.nowPlaying.Update(msg)`)

### Handlers (`internal/app/handlers.go`)

Wire 7 new message types:

| Message | Handler |
|---------|---------|
| `FetchFollowedShowsRequestMsg` | If `!store.FollowedShowsStale()` → skip; else dispatch `buildFetchFollowedShowsCmd()` |
| `FetchSavedEpisodesRequestMsg` | Same pattern, `SavedEpisodesStale()` guard |
| `FetchShowEpisodesRequestMsg` | If `!store.ShowEpisodesStale()` and same showID → skip; else dispatch `buildFetchShowEpisodesCmd()` |
| `FollowedShowsLoadedMsg` | Call `store.SetFollowedShowsFetching(false)`; if err → `store.SetFollowedShowsFetchError(err)`, `alerts.NewAlertCmd`; else `store.SetFollowedShows(items)`, `store.ClearFollowedShowsFetchError()` |
| `SavedEpisodesLoadedMsg` | Same pattern for `store.SetSavedEpisodes()` |
| `ShowEpisodesLoadedMsg` | Same pattern for `store.SetShowEpisodes()`; also `store.SetShowEpisodesTotal(total)` |
| `SelectedShowChangedMsg` | `store.SetSelectedShowID(id)`, fetch show metadata via `buildFetchShowEpisodesCmd(ctx, id)` |
| `PlayEpisodeMsg` | Dispatch `buildPlayEpisodeCmd(episodeURI, playlistURI)` |

**Polling entries**: Add to the tick-driven polling loop:
- Followed shows: check `FollowedShowsStale()`, dispatch `FetchFollowedShowsRequestMsg` if stale
- Saved episodes: check `SavedEpisodesStale()`, dispatch `FetchSavedEpisodesRequestMsg` if stale
- Show episodes: check `ShowEpisodesStale()`, dispatch `FetchShowEpisodesRequestMsg` if stale

**Search auto-navigation**: In the search result handler, after a result is selected:
```go
if resultType == "show" || resultType == "episode" {
    a.layout.SetActivePage(layout.PagePodcasts)
}
```

### Appendices: docs updates

**`docs/system/design.md`:**

- **§1 Overview**: Change page count from 2 to 3, add Podcasts to cycle description
- **§2 Pane Definitions**: Add podcasts page pane table (4 panes with IDs and border tokens)
- **§3 Grid Definition**: Add podcasts page grid section (2-row layout)
- **§4 Pages/Presets**: Update page cycle, add podcast presets table
- **§8 Filterable Fields**: Add ShowEpisodes (name), FollowedShows (show.name, show.publisher), SavedEpisodes (episode.name, show.name)
- **§10 Border Colors**: Add 4 podcast border tokens with music-page analogies
- **§11 Theme Border Token Hex Values**: Add 4 new rows per theme
- **§16 Focus**: Update focus rotation for 3 pages
- **§17 Keybinding Table**: Append podcast keybindings (s, r, Space, +/-/, →/←, Shift+→/←, f, Esc, D)
- **§18**: Update border token count

**`docs/system/api-guide.md`:**
- **§1 Scopes**: Add `user-read-playback-position` to table
- **§2 Coverage**: Mark Shows and Episodes as Implemented
- **§6.4**: Add Saved Shows subsection (GET/PUT/DELETE /me/shows)
- **§14**: Add Shows API section (GET /shows/{id}, /shows/{id}/episodes, /me/shows)
- **§15**: Add Episodes API section (GET /episodes/{id}, GET /me/episodes)
- **§22**: Add `user-read-playback-position` to scope reference
- **§23**: Add Podcasts to feature matrix

**`README.md`**: Add Podcasts page section to Keybindings table

**No changes to:** `docs/system/architecture.md` (pattern-only doc, no literal file to update unless arch diagram is in it)

## Acceptance Criteria

- [ ] 4 command factories compile and dispatch correct API calls
- [ ] App struct has `podcastClient` + 4 pane fields; initialized in `New()`
- [ ] `0` key cycles Music → Podcasts → Stats → Music
- [ ] `1`–`4` toggle pane visibility on podcasts page
- [ ] Playback keys (Space, ←/→, etc.) route to `PodcastPlaybackPane` on podcasts page
- [ ] `FetchFollowedShowsRequestMsg`, `FetchSavedEpisodesRequestMsg`, `FetchShowEpisodesRequestMsg` trigger fetches
- [ ] `FollowedShowsLoadedMsg`, `SavedEpisodesLoadedMsg`, `ShowEpisodesLoadedMsg` write to store + clear errors
- [ ] `SelectedShowChangedMsg` triggers show episode fetch
- [ ] `PlayEpisodeMsg` triggers player playback command
- [ ] Polling loop checks podcast data staleness and dispatches fetch requests
- [ ] Search auto-navigation switches to PagePodcasts on show/episode selection
- [ ] `docs/system/design.md` §1–4, §8, §10–11, §16–18 updated
- [ ] `docs/system/api-guide.md` §1, §2, §6.4, §14, §15, §22, §23 updated
- [ ] `README.md` keybindings updated
- [ ] `make ci` passes

## Tasks

**Commands:**

- [ ] Add `buildFetchFollowedShowsCmd` to `internal/app/commands.go` — guards nil client, sets fetching sentinel, calls `FollowedShows(ctx, 50, 0)`
- [ ] Add `buildFetchSavedEpisodesCmd` — same pattern, calls `SavedEpisodes(ctx, 50, 0)`
- [ ] Add `buildFetchShowEpisodesCmd(ctx, showID)` — same pattern, calls `ShowEpisodes(ctx, showID, 50, 0)`
- [ ] Add `buildPlayEpisodeCmd(episodeURI, playlistURI)` — calls `client.Play` with URI option
- [ ] Add tests: verify each command factory returns a non-nil `tea.Cmd` closure — in `internal/app/app_test.go` (or new `commands_test.go`)
- [ ] Add tests: verify commands set fetching sentinel before dispatch — in `commands_test.go`

**App struct + init:**

- [ ] Add `podcastClient *api.PodcastClient` field to App struct in `internal/app/app.go`
- [ ] Add 4 pane fields: `podcastPlayback`, `showEpisodes`, `followedShows`, `savedEpisodes`
- [ ] Initialize `podcastClient` in `New()`: `api.NewPodcastClient(a.config.APIBaseURL, accessToken)`
- [ ] Initialize 4 panes in `New()` with store + theme
- [ ] Add 4 panes to pane map (`PanePodcastPlayback..PaneSavedEpisodes`)
- [ ] Add test: verify `New()` creates all 4 panes and `podcastClient` is non-nil — in `app_test.go`

**Routing:**

- [ ] Update `routing.go`: `0` key cycles Music → Podcasts → Stats → Music (via `a.layout.TogglePage()`)
- [ ] Add pane toggle keys `1`–`4` on podcasts page → `a.layout.TogglePane(PaneID)`
- [ ] Route playback keys (Space, ←/→, Shift+←/Shift+→, +/-, s, r) to `a.podcastPlayback.Update(msg)` on podcasts page
- [ ] Add test: verify `0` on podcasts page switches to Stats — in `routing_test.go`
- [ ] Add test: verify `1`–`4` toggle correct panes — in `routing_test.go`

**Handlers:**

- [ ] Handle `FetchFollowedShowsRequestMsg` — dispatch `buildFetchFollowedShowsCmd()` if stale
- [ ] Handle `FetchSavedEpisodesRequestMsg` — dispatch `buildFetchSavedEpisodesCmd()` if stale
- [ ] Handle `FetchShowEpisodesRequestMsg` — dispatch `buildFetchShowEpisodesCmd(id)` if stale or different show
- [ ] Handle `FollowedShowsLoadedMsg` — clear fetching sentinel, write to store (or set error + toast)
- [ ] Handle `SavedEpisodesLoadedMsg` — same pattern
- [ ] Handle `ShowEpisodesLoadedMsg` — same pattern, also set total
- [ ] Handle `SelectedShowChangedMsg` — set SelectedShowID, dispatch show episode fetch
- [ ] Handle `PlayEpisodeMsg` — dispatch `buildPlayEpisodeCmd`
- [ ] Add tests: verify each handler dispatches correct command or updates store — in `handlers_test.go`

**Polling + search auto-navigation:**

- [ ] Add polling entries: check `FollowedShowsStale()` → dispatch `FetchFollowedShowsRequestMsg`
- [ ] Add polling entries: check `SavedEpisodesStale()` → dispatch `FetchSavedEpisodesRequestMsg`
- [ ] Add polling entries: check `ShowEpisodesStale()` → dispatch `FetchShowEpisodesRequestMsg`
- [ ] Add search auto-navigation: in search result handler, if result is "show" or "episode", switch to `PagePodcasts`
- [ ] Add test: verify search auto-navigation switches page for shows/episodes — in `app_test.go`

**Docs:**

- [ ] Update `docs/system/design.md` §1 (3 pages), §2 (podcast pane table), §3 (podcast grid), §4 (presets), §8 (filterable fields), §10 (border tokens), §11 (hex values), §16 (focus rotation), §17 (keybindings), §18 (token count)
- [ ] Update `docs/system/api-guide.md` §1 (scope), §2 (coverage), §6.4 (saved shows), §14 (shows API), §15 (episodes API), §22 (scope reference), §23 (feature matrix)
- [ ] Update `README.md` keybindings with Podcasts page section
- [ ] Run `make ci` — all gates pass
