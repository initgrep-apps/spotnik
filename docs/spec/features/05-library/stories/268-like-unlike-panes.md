---
title: "Like/Unlike Tracks — Cross-Pane Wiring"
feature: 05-library
status: done
---

## Background

Story 267 built the core like/unlike infrastructure: store methods, glyph, messages, commands, routing, and the first two pane integrations (NowPlayingPane, LikedSongsPane). This story extends the `l` keybinding and heart indicator to all remaining panes that display individual tracks: Queue, TopTracks, RecentlyPlayed, Playlists (track sub-view), Albums (track sub-view), and Search results. It also updates the three keybinding documentation locations and the glyph catalogue.

After this story, users can like/unlike any track from any pane in the application.

Depends on: Story 267 (core infrastructure), existing pane implementations in `internal/ui/panes/`.

## Design

### QueuePane (`internal/ui/panes/queue.go`)

**Keybinding:** `l` key in `Update` method. Gets selected `QueueItem`, checks `IsTrack`, emits `ToggleLikeRequestMsg{Track: item.Track, CurrentlyLiked: store.IsTrackLiked(item.Track.ID)}`.

**Heart display:** In `refreshRows()`, prepend `"♥ "` to track name when `store.IsTrackLiked(track.ID)` is true.

### TopTracksPane (`internal/ui/panes/toptracks_pane.go`)

**Keybinding:** `l` key in `Update` method. Gets selected track, emits `ToggleLikeRequestMsg{Track: *track, CurrentlyLiked: store.IsTrackLiked(track.ID)}`.

**Heart display:** In `refreshRows()`, prepend `"♥ "` to track name when liked.

### RecentlyPlayedPane (`internal/ui/panes/recentlyplayed_pane.go`)

**Keybinding:** `l` key in `Update` method. Gets selected `PlayHistory` item, emits `ToggleLikeRequestMsg{Track: item.Track, CurrentlyLiked: store.IsTrackLiked(item.Track.ID)}`.

**Heart display:** In `refreshRows()`, prepend `"♥ "` to track name when liked.

### PlaylistsPane track sub-view (`internal/ui/panes/playlists_pane.go`)

**Keybinding:** `l` key in `handleTrackViewKey` method. Gets selected track from `loadedTracks`, emits `ToggleLikeRequestMsg{Track: track, CurrentlyLiked: store.IsTrackLiked(track.ID)}`.

**Heart display:** In track sub-view row rendering, prepend `"♥ "` to track name when liked.

### AlbumsPane track sub-view (`internal/ui/panes/albums_pane.go`)

**Keybinding:** `l` key in `handleTrackViewKey` method. Gets selected track from `loadedTracks`, emits `ToggleLikeRequestMsg{Track: track, CurrentlyLiked: store.IsTrackLiked(track.ID)}`.

**Heart display:** In track sub-view row rendering, prepend `"♥ "` to track name when liked.

### SearchOverlay (`internal/ui/panes/search.go`)

**Keybinding:** `l` key in `handleKey` method. Gets selected `SearchListItem`, checks `IsTrack`, converts to `domain.Track` (extract ID from URI, use Name, set Artists from secondary text), emits `ToggleLikeRequestMsg{Track: track, CurrentlyLiked: store.IsTrackLiked(track.ID)}`.

**Heart display:** In `SearchItemDelegate` render, prepend `"♥ "` to track name when liked. Note: search results don't have a full `domain.Track` — the delegate needs to call `store.IsTrackLiked(uri)` or extract the track ID from the URI.

### Documentation updates

Three keybinding locations (per AGENTS.md rule 17):
- `README.md` — add `l` to keybinding table
- `docs/system/design.md §17` — add `l` to keybinding table
- `internal/ui/panes/help_overlay.go` — add `l` to help content

Glyph catalogue:
- `docs/system/tui.md §4` — add `GlyphLiked` entry
- `docs/system/cli.md` — add `GlyphLiked` entry

## Files

### Modify

- `internal/ui/panes/queue.go` — add `l` keybinding, heart in track column
- `internal/ui/panes/toptracks_pane.go` — add `l` keybinding, heart in track column
- `internal/ui/panes/recentlyplayed_pane.go` — add `l` keybinding, heart in track column
- `internal/ui/panes/playlists_pane.go` — add `l` keybinding in track sub-view, heart in track column
- `internal/ui/panes/albums_pane.go` — add `l` keybinding in track sub-view, heart in track column
- `internal/ui/panes/search.go` — add `l` keybinding in search results
- `internal/ui/panes/search_delegate.go` — add heart in search result rendering
- `internal/ui/panes/help_overlay.go` — add `l` to help content
- `docs/system/design.md` — add `l` to keybinding table §17
- `docs/system/tui.md` — add `GlyphLiked` to glyph catalogue §4
- `docs/system/cli.md` — add `GlyphLiked` to glyph table
- `README.md` — add `l` to keybinding table

## Acceptance Criteria

- [ ] Pressing `l` in QueuePane emits `ToggleLikeRequestMsg` with correct `CurrentlyLiked` value
- [ ] Pressing `l` in TopTracksPane emits `ToggleLikeRequestMsg` with correct `CurrentlyLiked` value
- [ ] Pressing `l` in RecentlyPlayedPane emits `ToggleLikeRequestMsg` with correct `CurrentlyLiked` value
- [ ] Pressing `l` in PlaylistsPane track sub-view emits `ToggleLikeRequestMsg` with correct `CurrentlyLiked` value
- [ ] Pressing `l` in AlbumsPane track sub-view emits `ToggleLikeRequestMsg` with correct `CurrentlyLiked` value
- [ ] Pressing `l` in SearchOverlay on a track result emits `ToggleLikeRequestMsg` with correct `CurrentlyLiked` value
- [ ] All 6 panes show `♥` prefix on track name when liked, no icon when unliked
- [ ] `l` keybinding documented in README.md, design.md §17, and help_overlay.go
- [ ] `GlyphLiked` documented in tui.md §4 and cli.md
- [ ] `make ci` passes

## Tasks

- [ ] Add `l` keybinding and heart indicator to QueuePane
      - test: `TestQueuePane_L_EmitsToggleLikeRequest`, `TestQueuePane_View_ShowsHeartWhenLiked`
- [ ] Add `l` keybinding and heart indicator to TopTracksPane
      - test: `TestTopTracksPane_L_EmitsToggleLikeRequest`, `TestTopTracksPane_View_ShowsHeartWhenLiked`
- [ ] Add `l` keybinding and heart indicator to RecentlyPlayedPane
      - test: `TestRecentlyPlayedPane_L_EmitsToggleLikeRequest`, `TestRecentlyPlayedPane_View_ShowsHeartWhenLiked`
- [ ] Add `l` keybinding and heart indicator to PlaylistsPane track sub-view
      - test: `TestPlaylistsPane_TrackView_L_EmitsToggleLikeRequest`, `TestPlaylistsPane_TrackView_ShowsHeartWhenLiked`
- [ ] Add `l` keybinding and heart indicator to AlbumsPane track sub-view
      - test: `TestAlbumsPane_TrackView_L_EmitsToggleLikeRequest`, `TestAlbumsPane_TrackView_ShowsHeartWhenLiked`
- [ ] Add `l` keybinding and heart indicator to SearchOverlay
      - test: `TestSearchOverlay_L_OnTrack_EmitsToggleLikeRequest`, `TestSearchOverlay_View_ShowsHeartWhenLiked`
- [ ] Update help overlay, design.md, tui.md, cli.md, and README.md with `l` keybinding and `GlyphLiked`
      - test: verify help overlay renders `l` entry
