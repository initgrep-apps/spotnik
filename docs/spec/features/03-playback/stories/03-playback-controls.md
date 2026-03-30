---
title: "Playback Controls"
feature: 03-playback
status: done
---

## Background
This story built the entire playback subsystem: API models for Spotify's playback JSON, an HTTP client with methods for every playback endpoint, three UI components (seek bar, volume bar, transport controls), the PlayerPane Bubble Tea model, and the root-level tick polling loop. It also wired the PlayerPane into the root app model with header and status bar integration.

## Design

### Store fields this feature uses
```go
// internal/state/store.go -- fields you read/write via store methods
PlaybackState  *api.PlaybackState // nil when nothing active (204 response)
ActiveDevice   *api.Device        // nil when no device connected
```

### Polling pattern (tea.Tick -- never time.Sleep)
```go
// PlayerPane.Init() starts the tick loop
func (m *PlayerPane) Init() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{} })
}
// On tickMsg: dispatch fetchPlaybackStateCmd
// fetchPlaybackStateCmd calls GET /me/player and returns playbackStateFetchedMsg
// Between polls: locally increment progressMs by 1000 per tick for smooth seek bar
```

### Message types
```go
type tickMsg                struct{}
type playbackStateFetchedMsg struct{ state *api.PlaybackState } // state=nil -> 204
type playbackCmdSentMsg     struct{ err error }
```

### Design tokens used
`theme.SeekBar()`, `theme.VolumeBar()`, `theme.PlayingIndicator()`, `theme.TextPrimary()`, `theme.TextSecondary()`, `theme.TextMuted()`

### Center Pane Layout
```
|  NOW PLAYING                  |
|  ---------------------------  |
|                               |
|  Blinding Lights              |  <- Track name (bold, TextPrimary() token)
|  The Weeknd                   |  <- Artist (TextSecondary() token)
|  After Hours                  |  <- Album (TextMuted() token)
|                               |
|  ________________             |  <- Seek bar (SeekBar() token fill)
|  2:34 ________________ 4:12   |  <- Time row
|                               |
|  |<   ||   >|      ~   =>    |  <- Controls (PlayingIndicator() token when active)
|  ---------------------------  |
|  VOL  ________  65%           |  <- Volume bar
|                               |
```

### Empty state (nothing playing)
```
|  NOW PLAYING                  |
|  ---------------------------  |
|                               |
|         Nothing playing       |
|                               |
|    Open Spotify on a device   |
|    and start playing music    |
|                               |
```

### Polling architecture
- **Interval**: every 1000ms
- **Endpoint**: `GET /me/player`
- **On 204 No Content**: set `store.PlaybackState = nil` and `store.CurrentTrack = nil`
- **On 429**: back off for `Retry-After` seconds, show status bar message
- **Progress interpolation**: between polls, increment `progressMs` by 1000ms each tick locally
- **Local interpolation state:** `localProgressMs int` as pane-local state (not in Store)

### Keymap (Player Pane Focus)

| Key | Action |
|---|---|
| `Space` | Toggle play/pause |
| `n` or `->` | Next track |
| `p` or `<-` | Previous track |
| `+` | Volume +5 |
| `-` | Volume -5 |
| `s` | Toggle shuffle |
| `r` | Cycle repeat |
| `l` | Toggle like on current track |
| `<-` (seek mode) | Seek back 10s |
| `->` (seek mode) | Seek forward 10s |

### Files

| File | Purpose |
|---|---|
| `internal/api/player.go` | All playback API calls |
| `internal/api/player_test.go` | Tests with mock HTTP server |
| `internal/api/models.go` | PlaybackState, Track, Artist, Album structs |
| `internal/api/models_test.go` | JSON unmarshaling tests |
| `internal/ui/panes/player.go` | PlayerPane model (Init/Update/View) |
| `internal/ui/panes/player_test.go` | Update tests for all keymap actions |
| `internal/ui/components/progress.go` | Seek bar component |
| `internal/ui/components/progress_test.go` | Progress bar render tests |
| `internal/ui/components/controls.go` | Transport controls row |
| `internal/ui/components/controls_test.go` | Controls render tests |
| `internal/ui/components/volume.go` | Volume bar component |
| `internal/ui/components/volume_test.go` | Volume bar render tests |
| `internal/app/app.go` | Root model (starts polling, routes messages) |
| `internal/app/app_test.go` | Polling and routing tests |

### Out of Scope
- Web Playback SDK (browser-based playback)
- Lyrics display
- Audio waveform visualization
- Crossfade control

## Acceptance Criteria
- [ ] Currently playing track (name, artist, album) visible within 1 second of app launch
- [ ] `Space` play/pause responds in under 200ms (optimistic update shown immediately)
- [ ] Seek bar updates every 1 second accurately via local interpolation
- [ ] Volume changes reflect immediately in UI (optimistic), confirmed by next poll
- [ ] Shuffle/repeat state accurately reflects Spotify state after each poll
- [ ] "Nothing playing" empty state shown cleanly when Spotify returns 204
- [ ] All API functions tested with httptest mocks; all pane Update() handlers tested
- [ ] No crashes on 204 (nothing playing), 429 (rate limited), or 503 (Spotify down)

## Tasks
- [ ] API models -- Define PlaybackState, Track, Artist, Album structs with JSON unmarshaling
      - test: `TestPlaybackState_Unmarshal`, `TestPlaybackState_Unmarshal_NowPlaying`, `TestTrack_Unmarshal`, `TestPlaybackState_NilItem`
- [ ] Player API calls -- Implement HTTP client methods for all playback endpoints
      - test: `TestGetPlaybackState_Success`, `TestGetPlaybackState_204`, `TestGetPlaybackState_429`, `TestPlay_SendsCorrectBody`, `TestPause_SendsPUT`, `TestNext_SendsPOST`, `TestPrevious_SendsPOST`, `TestSeek_SendsPositionMs`, `TestSetVolume_ClampsRange`, `TestSetShuffle_SendsState`, `TestSetRepeat_SendsMode`
- [ ] Seek bar component -- Build progress bar with elapsed/total time and visual fill
      - test: `TestProgressBar_ZeroProgress`, `TestProgressBar_HalfProgress`, `TestProgressBar_FullProgress`, `TestProgressBar_TimeLabels`, `TestProgressBar_ZeroDuration`, `TestProgressBar_WidthAdapts`
- [ ] Volume bar component -- Build volume indicator with percentage label
      - test: `TestVolumeBar_Zero`, `TestVolumeBar_Fifty`, `TestVolumeBar_Hundred`, `TestVolumeBar_OverHundred`, `TestVolumeBar_Negative`
- [ ] Transport controls component -- Build playback controls row with state-dependent icons
      - test: `TestControls_Playing_ShowsPause`, `TestControls_Paused_ShowsPlay`, `TestControls_ShuffleOn`, `TestControls_ShuffleOff`, `TestControls_RepeatOff`, `TestControls_RepeatContext`, `TestControls_RepeatTrack`
- [ ] PlayerPane model -- Implement central pane Bubble Tea model with all keybindings
      - test: `TestPlayerPane_View_NowPlaying`, `TestPlayerPane_View_EmptyState`, `TestPlayerPane_Update_Space_WhenPlaying`, `TestPlayerPane_Update_Space_WhenPaused`, `TestPlayerPane_Update_N_SkipsNext`, `TestPlayerPane_Update_P_SkipsPrev`, `TestPlayerPane_Update_Plus_VolUp`, `TestPlayerPane_Update_Minus_VolDown`, `TestPlayerPane_Update_S_TogglesShuffle`, `TestPlayerPane_Update_R_CyclesRepeat`, `TestPlayerPane_Update_PlaybackFetched`, `TestPlayerPane_Update_IgnoresKeysWhenNotFocused`
- [ ] Polling loop -- Set up root model tick-based polling for playback state sync
      - test: `TestApp_Init_ReturnsBatch`, `TestApp_Update_TickMsg_DispatchesFetch`, `TestPollingLoop_FetchesAndUpdatesStore`
- [ ] Integration wiring -- Wire PlayerPane into root app with header/status bar integration
      - test: `TestApp_PlayerPaneRouting`, `TestApp_HeaderShowsDevice`
