---
title: "Like/Unlike Tracks — Core Infrastructure"
feature: 05-library
status: open
---

## Background

Spotnik can display liked tracks in the LikedSongsPane and the API client already has `LikeTrack`/`UnlikeTrack` methods, but there is no way for users to like or unlike a track from the UI. The `Track` domain struct has no `IsLiked` field — liked status is determined by whether a track ID exists in the store's `likedTracks` slice, but there is no O(1) lookup method. The store has no `AddLikedTrack` or `RemoveLikedTrack` methods for optimistic updates.

This story builds the core infrastructure: store methods for O(1) liked-status lookup and optimistic insert/remove, a new `GlyphLiked` glyph, message types for the like/unlike toggle flow, command factories, routing handlers, and the first two pane integrations (NowPlayingPane and LikedSongsPane). After this story, users can like/unlike from NowPlaying and unlike from LikedSongs.

Depends on: existing `LibraryClient.LikeTrack`/`UnlikeTrack` in `internal/api/library.go`, existing `LikedSongsPane` in `internal/ui/panes/likedsongs_pane.go`, existing `NowPlayingPane` in `internal/ui/panes/nowplaying.go`.

## Design

### Store changes (`internal/state/store.go`)

New field on `Store`:

```go
likedSet map[string]bool // O(1) lookup: trackID → liked
```

Initialized in `New()`. Rebuilt in `SetLikedTracks()`.

New methods:

```go
func (s *Store) IsTrackLiked(trackID string) bool
func (s *Store) AddLikedTrack(track domain.Track)
func (s *Store) RemoveLikedTrack(trackID string)
```

`AddLikedTrack` wraps the track in `domain.SavedTrack{AddedAt: time.Now().Format(time.RFC3339), Track: track}` and prepends to `likedTracks`. `RemoveLikedTrack` deletes from `likedSet` and removes from `likedTracks` slice. Both update `likedTotal`.

### StateReader changes (`internal/state/reader.go`)

Add to `StateReader` interface:

```go
IsTrackLiked(trackID string) bool
```

### Glyph changes (`internal/uikit/glyph.go`)

```go
GlyphLiked GlyphRole = "state.liked"
```

Table entry: `{"♥", "Y"}`

### Message types (`internal/ui/panes/messages.go`)

```go
type ToggleLikeRequestMsg struct {
	Track          domain.Track
	CurrentlyLiked bool // true = currently liked → unlike; false → like
}

type ToggleLikeResultMsg struct {
	TrackID       string
	Liked         bool   // true after successful like, false after unlike
	OriginalLiked bool   // state before toggle, for rollback
	Err           error
}
```

### Command factories (`internal/app/commands.go`)

```go
func (a *App) buildLikeTrackCmd(trackID, trackName string) tea.Cmd
func (a *App) buildUnlikeTrackCmd(trackID, trackName string) tea.Cmd
```

Pattern: snapshot `a.library` in Update() context, return closure calling `library.LikeTrack`/`library.UnlikeTrack` with `api.Interactive` priority, return `ToggleLikeResultMsg`. Handle 429 → `RateLimitedMsg`, 401 → `unauthorizedMsg`.

### Routing (`internal/app/routing.go`)

`ToggleLikeRequestMsg` handler:
1. Premium gate check (toast warning if not premium)
2. Optimistic store update: `AddLikedTrack` or `RemoveLikedTrack`
3. Dispatch `buildLikeTrackCmd` or `buildUnlikeTrackCmd`

`ToggleLikeResultMsg` handler:
1. On error: rollback optimistic update. If `OriginalLiked` was true (was liked before toggle), mark liked tracks stale to force re-fetch. If `OriginalLiked` was false, call `RemoveLikedTrack`. Show error toast.
2. On success: show toast ("♥ Liked" or "Unliked")

### NowPlayingPane (`internal/ui/panes/nowplaying.go`)

**Keybinding:** `l` key in `handleKey` method. Reads `store.PlaybackState().Item`, emits `ToggleLikeRequestMsg{Track: *item, CurrentlyLiked: store.IsTrackLiked(item.ID)}`.

**Heart display:** In `buildInfoLines()` or equivalent track-name rendering, prepend `"♥ "` to track name when `store.IsTrackLiked(track.ID)` is true.

### LikedSongsPane (`internal/ui/panes/likedsongs_pane.go`)

**Keybinding:** `l` key in `Update` method. Gets selected track, emits `ToggleLikeRequestMsg{Track: *track, CurrentlyLiked: true}` (all tracks in LikedSongs are liked, so this always unlikes).

**Heart display:** All rows always show `"♥ "` prepended to track name (all are liked).

## Files

### Modify

- `internal/state/store.go` — add `likedSet` field, `IsTrackLiked`, `AddLikedTrack`, `RemoveLikedTrack`, update `SetLikedTracks` and `New()`
- `internal/state/reader.go` — add `IsTrackLiked` to `StateReader`
- `internal/uikit/glyph.go` — add `GlyphLiked` constant + table entry
- `internal/ui/panes/messages.go` — add `ToggleLikeRequestMsg`, `ToggleLikeResultMsg`
- `internal/app/commands.go` — add `buildLikeTrackCmd`, `buildUnlikeTrackCmd`
- `internal/app/routing.go` — add handlers for `ToggleLikeRequestMsg`, `ToggleLikeResultMsg`
- `internal/ui/panes/nowplaying.go` — add `l` keybinding, heart in InfoBox
- `internal/ui/panes/likedsongs_pane.go` — add `l` keybinding for unlike, heart on all rows
- `internal/state/store_test.go` — tests for new store methods

## Acceptance Criteria

- [ ] `Store.IsTrackLiked(id)` returns true for tracks in `likedTracks`, false otherwise
- [ ] `Store.AddLikedTrack(track)` prepends to `likedTracks` and updates `likedSet`
- [ ] `Store.RemoveLikedTrack(id)` removes from `likedTracks` and `likedSet`
- [ ] `SetLikedTracks` rebuilds `likedSet` from incoming tracks
- [ ] Pressing `l` in NowPlayingPane emits `ToggleLikeRequestMsg` with correct `CurrentlyLiked` value
- [ ] Pressing `l` in LikedSongsPane emits `ToggleLikeRequestMsg` with `CurrentlyLiked: true`
- [ ] Routing handler does premium gate check before dispatching API command
- [ ] Optimistic store update happens before API call, rollback on error
- [ ] NowPlayingPane shows `♥` prefix on track name when liked
- [ ] LikedSongsPane shows `♥` prefix on all track names
- [ ] Toast notifications fire on success and error
- [ ] `make ci` passes

## Tasks

- [ ] Add `likedSet` field, initialize in `New()`, rebuild in `SetLikedTracks()`, add `IsTrackLiked`, `AddLikedTrack`, `RemoveLikedTrack` methods to Store
      - test: `TestStore_IsTrackLiked`, `TestStore_AddLikedTrack`, `TestStore_RemoveLikedTrack`, `TestStore_SetLikedTracksRebuildsLikedSet`
- [ ] Add `IsTrackLiked` to `StateReader` interface
      - test: compile-time assertion already exists, verify no compile error
- [ ] Add `GlyphLiked` constant and table entry to `glyph.go`
      - test: existing `AllGlyphRoles` test covers new entry
- [ ] Add `ToggleLikeRequestMsg` and `ToggleLikeResultMsg` to `messages.go`
      - test: compile-time check only
- [ ] Add `buildLikeTrackCmd` and `buildUnlikeTrackCmd` to `commands.go`
      - test: `TestBuildLikeTrackCmd_Success`, `TestBuildLikeTrackCmd_NilClient`, `TestBuildUnlikeTrackCmd_Success`, `TestBuildUnlikeTrackCmd_NilClient`
- [ ] Wire `ToggleLikeRequestMsg` and `ToggleLikeResultMsg` in `routing.go` with premium gate, optimistic update, rollback, and toasts
      - test: `TestRouting_ToggleLikeRequest_Likes`, `TestRouting_ToggleLikeRequest_Unlikes`, `TestRouting_ToggleLikeRequest_PremiumGate`, `TestRouting_ToggleLikeResult_Success`, `TestRouting_ToggleLikeResult_ErrorRollback`
- [ ] Add `l` keybinding and heart indicator to NowPlayingPane
      - test: `TestNowPlayingPane_L_EmitsToggleLikeRequest`, `TestNowPlayingPane_View_ShowsHeartWhenLiked`, `TestNowPlayingPane_View_NoHeartWhenUnliked`
- [ ] Add `l` keybinding and heart indicator to LikedSongsPane
      - test: `TestLikedSongsPane_L_EmitsToggleLikeRequest`, `TestLikedSongsPane_View_ShowsHeartOnAllRows`
