---
title: "Context-aware playback for song-list panes"
feature: 20-playback-context
status: open
---

## Background

Every pane that plays a single track currently sends only one URI to Spotify:

```go
player.Play(ctx, domain.PlayOptions{URIs: []string{trackURI}})
```

This is the wrong API usage for panes that show a list of songs. When Spotify
receives a single `uris` entry with no `context_uri`, it clears the current
collection context and fills the queue with the same track repeated (~10 times).
This is Spotify's autoqueue behaviour when there is nothing else to play.

**Affected panes and their correct fix:**

| Pane | Root cause | Correct approach |
|---|---|---|
| Liked Songs (`likedsongs_pane.go:153`) | `PlayTrackMsg` with no context | `context_uri: "spotify:collection:tracks"` + `offset: {uri}` |
| Top Tracks (`toptracks_pane.go:180`) | `PlayTrackMsg` with no context | `uris: allTopTracks[selectedIdx:]` |
| Recently Played (`recentlyplayed_pane.go:152`) | `PlayTrackMsg` with no context | `uris: allRecentTracks[selectedIdx:]` |
| Search tracks (`search.go:729`) | `PlayTrackMsg` with no context | `uris: allSearchResultURIs[selectedIdx:]` |

**Liked Songs** uses `spotify:collection:tracks` because Spotify owns that
collection and can fill the queue with your actual upcoming liked songs.

**Top Tracks, Recently Played, Search** have no Spotify collection URI — the
app fetched the list and holds it in memory. The correct approach is to pass
all track URIs starting from the selected one so Spotify queues the rest.

The **Queue pane** is excluded — playing a queued track is a skip-to operation,
not a context play, and its existing behaviour is correct.

## Design

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

- `internal/domain/types.go` — add `PlayOffset`, extend `PlayOptions`
- `internal/ui/panes/messages.go` — add `OffsetURI` to `PlayContextMsg`, add `PlayTrackListMsg`
- `internal/app/app.go` — update `PlayContextMsg` handler, add `PlayTrackListMsg` handler
- `internal/app/commands.go` — update `buildPlayContextCmd` signature, add `buildPlayTrackListCmd`
- `internal/ui/panes/likedsongs_pane.go` — emit `PlayContextMsg` with collection context
- `internal/ui/panes/toptracks_pane.go` — emit `PlayTrackListMsg` with URI slice
- `internal/ui/panes/recentlyplayed_pane.go` — emit `PlayTrackListMsg` with URI slice
- `internal/ui/panes/search.go` — emit `PlayTrackListMsg` for track results

## Acceptance Criteria

- [ ] Playing a track from Liked Songs sets `context_uri: "spotify:collection:tracks"` and `offset.uri` — queue fills with actual upcoming liked songs
- [ ] Playing a track from Top Tracks passes URIs from selected index onward — queue fills with remaining top tracks
- [ ] Playing a track from Recently Played passes URIs from selected index onward — queue fills with remaining recent tracks
- [ ] Playing a track from Search results passes URIs from selected index onward — queue fills with remaining results
- [ ] Playing a playlist/album via `PlayContextMsg` without `OffsetURI` is unchanged (no regression)
- [ ] `PlayOptions` with `Offset` marshals correctly: `{"context_uri":"...","offset":{"uri":"..."}}`
- [ ] `PlayOptions` without `Offset` omits the field (no regression in JSON output)
- [ ] Queue pane playback behaviour is unchanged

## Tasks

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
      `ContextURI: "spotify:collection:tracks"` and `OffsetURI: uri`
      - test: Enter on row N emits `PlayContextMsg{ContextURI:"spotify:collection:tracks", OffsetURI: tracks[N].URI}`

- [ ] Update `toptracks_pane.go` Enter handler to emit `PlayTrackListMsg` with
      URIs from selected index to end of `store.TopTracks(timeRange)` slice
      - test: Enter on row N emits `PlayTrackListMsg{URIs: topTrackURIs[N:]}`

- [ ] Update `recentlyplayed_pane.go` Enter handler to emit `PlayTrackListMsg`
      with URIs from selected index to end of `store.RecentlyPlayed()` slice
      - test: Enter on row N emits `PlayTrackListMsg{URIs: recentTrackURIs[N:]}`

- [ ] Update `search.go` track-result Enter handler to emit `PlayTrackListMsg`
      with URIs from selected index to end of current track results page
      - test: Enter on track row N emits `PlayTrackListMsg{URIs: searchTrackURIs[N:]}`
