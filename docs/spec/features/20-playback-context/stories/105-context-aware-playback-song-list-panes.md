---
title: "Context-aware playback for song-list panes"
feature: 20-playback-context
status: open
---

> **Prerequisite for the feature**: This story must be implemented before Stories 106
> and 107. It introduces `PlayContextMsg.OffsetURI` and the new `PlayTrackListMsg`
> type that those stories depend on. It also removes the now-obsolete `PlayTrackMsg`
> and `buildPlayTrackCmd`.

## Background

Every pane that plays a single track currently emits `PlayTrackMsg{TrackURI: uri}`,
which causes `app.go` to call `buildPlayTrackCmd(uri)`:

```go
// Current (wrong) call in buildPlayTrackCmd:
player.Play(ctx, domain.PlayOptions{URIs: []string{trackURI}})
```

This is the wrong API usage for panes that show a list of songs. When Spotify
receives a single `uris` entry with no `context_uri`, it clears the current
collection context and fills the queue with the same track repeated (~10 times).
This is Spotify's autoqueue behaviour when there is nothing else to play.

**Affected panes and their correct fix:**

| Pane | Root cause | Correct approach |
|---|---|---|
| Liked Songs (Enter handler in `likedsongs_pane.go`) | `PlayTrackMsg` with no context | `context_uri: "spotify:collection:tracks"` + `offset: {uri}` |
| Top Tracks (Enter handler in `toptracks_pane.go`) | `PlayTrackMsg` with no context | `uris: allTopTracks[selectedIdx:]` |
| Recently Played (Enter handler in `recentlyplayed_pane.go`) | `PlayTrackMsg` with no context | `uris: allRecentTracks[selectedIdx:]` |
| Search tracks (track-result Enter handler in `search.go`) | `PlayTrackMsg` with no context | `uris: allSearchResultURIs[selectedIdx:]` |

**Note on line number references:** Do not use line numbers — they are stale after any
edit. Locate the Enter handlers by searching for `PlayTrackMsg` in each pane file.

**Liked Songs** uses `spotify:collection:tracks` because Spotify owns that
collection and can fill the queue with your actual upcoming liked songs.

**Top Tracks, Recently Played, Search** have no Spotify collection URI — the
app fetched the list and holds it in memory. The correct approach is to pass
all track URIs starting from the selected one so Spotify queues the rest.

The **Queue pane** is excluded — playing a queued track is a skip-to operation,
not a context play, and its existing behaviour is correct.

**Queue self-correction:** After the context fix is in place, the queue pane will
automatically display the correct upcoming tracks within ~1000ms via its existing
tick-based polling. No additional work is required for the queue pane.

## Design

### 0. Remove obsolete `PlayTrackMsg` and `buildPlayTrackCmd`

`PlayTrackMsg` and `buildPlayTrackCmd` are the old single-URI play mechanism. After
this story, every pane that previously emitted `PlayTrackMsg` will emit either
`PlayContextMsg` (with `OffsetURI`) or `PlayTrackListMsg`. The old types become dead
code and must be deleted to keep the codebase clean.

**Delete from `internal/ui/panes/messages.go`:**
```go
// DELETE this entire type:
type PlayTrackMsg struct {
    TrackURI string
}
```

**Delete from `internal/app/commands.go`:**
```go
// DELETE this entire function:
func (a *App) buildPlayTrackCmd(trackURI string) tea.Cmd { ... }
```

**Delete from `internal/app/app.go`:**
```go
// DELETE this case from the Update() switch:
case panes.PlayTrackMsg:
    return a, a.buildPlayTrackCmd(m.TrackURI)
```

After deletion, the compiler will immediately flag any remaining `PlayTrackMsg`
usages (there should be none once all four panes are updated in this story).

---

### 1. Extend `domain.PlayOptions` with offset support

Add `PlayOffset` struct and `Offset` pointer field to `PlayOptions` in
`internal/domain/types.go`:

```go
// PlayOffset specifies where within a context to start playback.
type PlayOffset struct {
    // URI is the Spotify track URI to start from within the context.
    URI string `json:"uri,omitempty"`
}

// PlayOptions specifies what to play.
// Provide ContextURI + Offset for collections (albums, playlists, liked songs).
// Provide URIs for an explicit ordered track list with no collection context.
type PlayOptions struct {
    ContextURI string      `json:"context_uri,omitempty"`
    URIs       []string    `json:"uris,omitempty"`
    Offset     *PlayOffset `json:"offset,omitempty"`
}
```

### 2. Extend `PlayContextMsg` with optional `OffsetURI`

In `internal/ui/panes/messages.go`, add `OffsetURI` to `PlayContextMsg`:

```go
// PlayContextMsg is sent when the user selects a playlist, album, or collection
// to play. OffsetURI is optional — when set, playback starts at that track URI
// within the context rather than from the beginning.
type PlayContextMsg struct {
    ContextURI string
    OffsetURI  string // optional: start at this track URI within the context
}
```

### 3. Add `PlayTrackListMsg` for panes without a collection context

In `internal/ui/panes/messages.go`, add:

```go
// PlayTrackListMsg is sent when the user plays a track from a pane that has no
// Spotify collection context (Top Tracks, Recently Played, Search results).
// URIs is the ordered list of track URIs starting from the selected track —
// Spotify will play URIs[0] and queue the rest.
type PlayTrackListMsg struct {
    URIs []string
}
```

### 4. Update `buildPlayContextCmd` to pass the offset

In `internal/app/commands.go`, change signature and add offset logic:

```go
func (a *App) buildPlayContextCmd(contextURI, offsetURI string) tea.Cmd {
    player := a.player
    return func() tea.Msg {
        if player == nil {
            return panes.PlaybackCmdSentMsg{Err: errNilClient}
        }
        opts := domain.PlayOptions{ContextURI: contextURI}
        if offsetURI != "" {
            opts.Offset = &domain.PlayOffset{URI: offsetURI}
        }
        err := player.Play(api.WithPriority(context.Background(), api.Interactive), opts)
        if err != nil {
            if secs := parse429RetryAfter(err); secs > 0 {
                return panes.RateLimitedMsg{RetryAfterSecs: secs}
            }
            if isUnauthorizedError(err) {
                return unauthorizedMsg{}
            }
        }
        return panes.PlaybackCmdSentMsg{Err: err}
    }
}
```

### 5. Add `buildPlayTrackListCmd`

In `internal/app/commands.go`, add:

```go
// buildPlayTrackListCmd dispatches a play command for an ordered list of track
// URIs. Used by panes without a Spotify collection context (Top Tracks, Recently
// Played, Search). Spotify plays URIs[0] and queues the rest.
func (a *App) buildPlayTrackListCmd(uris []string) tea.Cmd {
    player := a.player
    return func() tea.Msg {
        if player == nil {
            return panes.PlaybackCmdSentMsg{Err: errNilClient}
        }
        err := player.Play(api.WithPriority(context.Background(), api.Interactive),
            domain.PlayOptions{URIs: uris})
        if err != nil {
            if secs := parse429RetryAfter(err); secs > 0 {
                return panes.RateLimitedMsg{RetryAfterSecs: secs}
            }
            if isUnauthorizedError(err) {
                return unauthorizedMsg{}
            }
        }
        return panes.PlaybackCmdSentMsg{Err: err}
    }
}
```

### 6. Wire `PlayContextMsg` and `PlayTrackListMsg` in `app.go`

Update the `PlayContextMsg` handler to forward `OffsetURI`:

```go
case panes.PlayContextMsg:
    return a, a.buildPlayContextCmd(m.ContextURI, m.OffsetURI)
```

Add a new handler for `PlayTrackListMsg`:

```go
case panes.PlayTrackListMsg:
    return a, a.buildPlayTrackListCmd(m.URIs)
```

### 7. Update each pane

**`internal/ui/panes/likedsongs_pane.go`** — Enter handler:
```go
return l, func() tea.Msg {
    return PlayContextMsg{
        ContextURI: "spotify:collection:tracks",
        OffsetURI:  uri,
    }
}
```

**`internal/ui/panes/toptracks_pane.go`** — Enter handler:
```go
tracks := t.store.TopTracks(t.timeRange)  // all top tracks, same slice used for rendering
uris := make([]string, 0, len(tracks)-idx)
for _, tr := range tracks[idx:] {
    uris = append(uris, tr.URI)
}
return t, func() tea.Msg { return PlayTrackListMsg{URIs: uris} }
```

**`internal/ui/panes/recentlyplayed_pane.go`** — Enter handler:
```go
tracks := r.store.RecentlyPlayed()
uris := make([]string, 0, len(tracks)-idx)
for _, tr := range tracks[idx:] {
    uris = append(uris, tr.Track.URI)
}
return r, func() tea.Msg { return PlayTrackListMsg{URIs: uris} }
```

**`internal/ui/panes/search.go`** — Enter handler for track results:
```go
// Build URI list from selected track to end of track results page.
results := o.store.SearchTracks()
uris := make([]string, 0, len(results)-idx)
for _, tr := range results[idx:] {
    uris = append(uris, tr.URI)
}
return o, func() tea.Msg { return PlayTrackListMsg{URIs: uris} }
```

### Files

| File | Change |
|------|--------|
| `internal/domain/types.go` | Add `PlayOffset` struct; add `Offset *PlayOffset` to `PlayOptions` |
| `internal/ui/panes/messages.go` | **Delete** `PlayTrackMsg`; add `OffsetURI` to `PlayContextMsg`; add `PlayTrackListMsg` |
| `internal/app/app.go` | **Delete** `PlayTrackMsg` handler; update `PlayContextMsg` handler to pass `m.OffsetURI`; add `PlayTrackListMsg` handler |
| `internal/app/commands.go` | **Delete** `buildPlayTrackCmd`; update `buildPlayContextCmd` signature; add `buildPlayTrackListCmd` |
| `internal/ui/panes/likedsongs_pane.go` | Replace `PlayTrackMsg` with `PlayContextMsg` (collection context + offset) |
| `internal/ui/panes/toptracks_pane.go` | Replace `PlayTrackMsg` with `PlayTrackListMsg` (URIs from selected index) |
| `internal/ui/panes/recentlyplayed_pane.go` | Replace `PlayTrackMsg` with `PlayTrackListMsg` (URIs from selected index) |
| `internal/ui/panes/search.go` | Replace `PlayTrackMsg` with `PlayTrackListMsg` for track results |

## Acceptance Criteria

- [ ] Playing a track from Liked Songs sets `context_uri: "spotify:collection:tracks"` and `offset.uri` — queue fills with actual upcoming liked songs
- [ ] Playing a track from Top Tracks passes URIs from selected index onward — queue fills with remaining top tracks
- [ ] Playing a track from Recently Played passes URIs from selected index onward — queue fills with remaining recent tracks
- [ ] Playing a track from Search results passes URIs from selected index onward — queue fills with remaining results
- [ ] Playing a playlist/album via `PlayContextMsg` without `OffsetURI` is unchanged (no regression)
- [ ] `PlayOptions` with `Offset` marshals correctly: `{"context_uri":"...","offset":{"uri":"..."}}`
- [ ] `PlayOptions` without `Offset` omits the field (no regression in JSON output)
- [ ] Queue pane playback behaviour is unchanged
- [ ] `PlayTrackMsg` and `buildPlayTrackCmd` are fully removed — no dead code remains
- [ ] Queue pane reflects correct upcoming tracks within ~1000ms after any play call

> **Stories 106 and 107 depend on this story.** `PlayContextMsg.OffsetURI` and
> `PlayTrackListMsg` must exist before those stories can be implemented.

## Tasks

- [ ] **Delete** `PlayTrackMsg` from `messages.go`, `buildPlayTrackCmd` from `commands.go`,
      and the `PlayTrackMsg` case from `app.go` Update switch. Do this first so the
      compiler immediately surfaces any remaining usages as you update the panes.
      - test: compilation succeeds after all panes are updated (no remaining references)

- [ ] Add `PlayOffset` struct and `Offset *PlayOffset` field to `domain.PlayOptions`
      - test: `PlayOptions{ContextURI:"x", Offset:&PlayOffset{URI:"y"}}` marshals to
        `{"context_uri":"x","offset":{"uri":"y"}}`; zero Offset is omitted from JSON

- [ ] Add `OffsetURI string` to `PlayContextMsg`; add `PlayTrackListMsg{URIs []string}`

- [ ] Update `buildPlayContextCmd(contextURI, offsetURI string)` to set `opts.Offset`
      when `offsetURI` is non-empty; update `PlayContextMsg` handler in `app.go` to
      pass `m.OffsetURI`
      - test: empty offsetURI → `PlayOptions.Offset` is nil; non-empty → `Offset.URI` set

- [ ] Add `buildPlayTrackListCmd` and `PlayTrackListMsg` handler in `app.go`
      - test: handler calls `buildPlayTrackListCmd` with correct URIs slice

- [ ] Update `likedsongs_pane.go` Enter handler to emit `PlayContextMsg` with
      `ContextURI: "spotify:collection:tracks"` and `OffsetURI: uri`.
      **Preserve the existing idx bounds guard** (`if idx >= 0 && idx < len(tracks)`).
      - test: Enter on row N emits `PlayContextMsg{ContextURI:"spotify:collection:tracks", OffsetURI: tracks[N].URI}`
      - test: Enter when list is empty (idx == -1) emits no command

- [ ] Update `toptracks_pane.go` Enter handler to emit `PlayTrackListMsg` with
      URIs from selected index to end of `store.TopTracks(timeRange)` slice.
      **Preserve the existing idx bounds guard**.
      - test: Enter on row N emits `PlayTrackListMsg{URIs: topTrackURIs[N:]}`
      - test: Enter on last row emits `PlayTrackListMsg{URIs: [lastTrackURI]}`

- [ ] Update `recentlyplayed_pane.go` Enter handler to emit `PlayTrackListMsg`
      with URIs from selected index to end of `store.RecentlyPlayed()` slice.
      **Preserve the existing idx bounds guard**.
      - test: Enter on row N emits `PlayTrackListMsg{URIs: recentTrackURIs[N:]}`

- [ ] Update `search.go` track-result Enter handler to emit `PlayTrackListMsg`
      with URIs from selected index to end of current track results page.
      **Preserve the existing idx bounds guard**.
      - test: Enter on track row N emits `PlayTrackListMsg{URIs: searchTrackURIs[N:]}`
