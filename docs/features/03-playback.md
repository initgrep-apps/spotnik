# Feature 03 — Playback Controls

> **Depends on:** Feature 02 (Auth) complete and committed.

## Implementation Context

### Store fields this feature uses
```go
// internal/state/store.go — fields you read/write via store methods
PlaybackState  *api.PlaybackState // nil when nothing active (204 response)
ActiveDevice   *api.Device        // nil when no device connected
```

### Polling pattern (tea.Tick — never time.Sleep)
```go
// PlayerPane.Init() starts the tick loop
func (m *PlayerPane) Init() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{} })
}
// On tickMsg: dispatch fetchPlaybackStateCmd
// fetchPlaybackStateCmd calls GET /me/player and returns playbackStateFetchedMsg
// Between polls: locally increment progressMs by 1000 per tick for smooth seek bar
```

### Message types for this feature
```go
type tickMsg                struct{}
type playbackStateFetchedMsg struct{ state *api.PlaybackState } // state=nil → 204
type playbackCmdSentMsg     struct{ err error }
```

### Design tokens used in this feature
`theme.SeekBar()` · `theme.VolumeBar()` · `theme.PlayingIndicator()` ·
`theme.TextPrimary()` · `theme.TextSecondary()` · `theme.TextMuted()`
Get the active theme via the `theme Theme` field injected into the pane by `app.New()`.

---

---

## Goal

Show what's currently playing and give the user full control over playback from the terminal.
This is the core feature — the one people open the app for. It must be rock-solid and responsive.

---

## User Stories

- **As a user**, I see the current track name, artist, album, and album art (ASCII if possible) in the center pane.
- **As a user**, I see a live-updating seek bar showing progress through the track.
- **As a user**, I press `Space` to play or pause without thinking.
- **As a user**, I press `n` to skip to the next track and `p` for previous.
- **As a user**, I press `+` or `-` to adjust volume in 5% increments.
- **As a user**, I press `s` to toggle shuffle and see its state change immediately.
- **As a user**, I press `r` to cycle through repeat modes (off → context → track).
- **As a user**, I see the active device name in the header bar.
- **As a user**, if nothing is playing, I see a clear "Nothing playing" state.

---

## Center Pane Layout (from DESIGN.md)

```
│  NOW PLAYING                  │
│  ───────────────────────────  │
│                               │
│  Blinding Lights              │  ← Track name (bold, Text color)
│  The Weeknd                   │  ← Artist (Subtext1 color)
│  After Hours                  │  ← Album (Subtext0, smaller)
│                               │
│  ████████████████░░░░░░░░░░   │  ← Seek bar (Peach fill)
│  2:34 ──────────────── 4:12   │  ← Time row
│                               │
│  ⏮   ⏸   ⏭      🔀   🔁      │  ← Controls (Green when active)
│  ───────────────────────────  │
│  VOL  ████████░░░░░░  65%     │  ← Volume bar
│                               │
```

**Empty state** (nothing playing):
```
│  NOW PLAYING                  │
│  ───────────────────────────  │
│                               │
│         Nothing playing       │
│                               │
│    Open Spotify on a device   │
│    and start playing music    │
│                               │
```

---

## Polling Architecture

The playback state must stay live. Use `tea.Tick` exclusively.

- **Interval**: every 1000ms
- **Endpoint**: `GET /me/player` (includes full state: track, progress, device, shuffle, repeat, volume)
- **On 204 No Content**: Spotify returns 204 when nothing is playing — update state to empty
- **On 429**: back off for `Retry-After` seconds, show status bar message
- **Progress interpolation**: between polls, increment `progressMs` by 1000ms each tick locally
  so the seek bar moves smoothly without hammering the API

---

## Seek Bar Behavior

The seek bar fills from left to right based on `progressMs / durationMs`.

- Width: full pane width minus 4 chars padding (2 each side)
- On `Seek` command: immediately update local `progressMs` to the seeked position,
  then the next poll confirms the actual server state
- Seeking UI: left/right arrow keys on seek bar move by 10s increments
  (only when Player pane is focused)

---

## Volume Control

- `+`: increase by 5, cap at 100
- `-`: decrease by 5, floor at 0
- Volume change fires `PUT /me/player/volume` immediately
- Optimistic update: update the local volume display before the API confirms
- If API returns error, revert to previous value

---

## Shuffle & Repeat Display

Shuffle and repeat state comes from `GET /me/player` response.

| State | Display |
|---|---|
| Shuffle off | `🔀` in `Subtext1` |
| Shuffle on | `🔀` in `Green` |
| Repeat off | `🔁` in `Subtext1` |
| Repeat context | `🔁` in `Green` |
| Repeat track | `🔂` in `Green` |

---

## Keymap (Player Pane Focus)

| Key | Action |
|---|---|
| `Space` | Toggle play/pause |
| `n` or `→` | Next track |
| `p` or `←` | Previous track |
| `+` | Volume +5 |
| `-` | Volume -5 |
| `s` | Toggle shuffle |
| `r` | Cycle repeat |
| `l` | Toggle like on current track |
| `←` (seek mode) | Seek back 10s |
| `→` (seek mode) | Seek forward 10s |

---

## Files to Create

| File | Purpose |
|---|---|
| `internal/api/player.go` | All playback API calls |
| `internal/api/player_test.go` | Tests with mock HTTP server |
| `internal/api/models.go` | PlaybackState, Track, Artist, Album structs |
| `internal/ui/panes/player.go` | PlayerPane model (Init/Update/View) |
| `internal/ui/panes/player_test.go` | Update tests for all keymap actions |
| `internal/ui/components/progress.go` | Seek bar component |
| `internal/ui/components/progress_test.go` | Progress bar render tests |
| `internal/ui/components/controls.go` | Transport controls row |
| `internal/ui/components/volume.go` | Volume bar component |
| `internal/app/app.go` | Root model (starts polling, routes messages) |

---

## Task Breakdown

### Task 2.1 — API models
- [ ] Define `PlaybackState` struct (all fields from Spotify response)
- [ ] Define `Track`, `Artist`, `Album`, `SimpleAlbum` structs
- [ ] Write JSON unmarshaling tests using `testdata/fixtures/`

### Task 2.2 — Player API calls
- [ ] `GetPlaybackState(ctx) (*PlaybackState, error)`
- [ ] `Play(ctx, opts PlayOptions) error` — supports context URI or track URI
- [ ] `Pause(ctx) error`
- [ ] `Next(ctx) error`
- [ ] `Previous(ctx) error`
- [ ] `Seek(ctx, positionMs int) error`
- [ ] `SetVolume(ctx, volume int) error`
- [ ] `SetShuffle(ctx, state bool) error`
- [ ] `SetRepeat(ctx, mode string) error`
- [ ] Test each with `httptest.NewServer` mock

### Task 2.3 — Seek bar component
- [ ] `ProgressBar` struct with `Width`, `FilledColor`, `EmptyColor`
- [ ] `Render(progressMs, durationMs int) string`
- [ ] Time label row: `"2:34 ─────────────── 4:12"` — dynamic fill
- [ ] Test: various progress values, zero, full, over-full
- [ ] Test: correct width at different terminal widths

### Task 2.4 — Volume bar component
- [ ] `VolumeBar` struct
- [ ] `Render(volume int) string`
- [ ] Clamp input: min 0, max 100
- [ ] Test: 0%, 50%, 100%, over-100 (clamped)

### Task 2.5 — Transport controls component
- [ ] `Controls` struct with `IsPlaying`, `ShuffleOn`, `RepeatMode` fields
- [ ] `Render() string`
- [ ] Active states styled with `Green`
- [ ] Test: all state combinations render correct symbols/colors

### Task 2.6 — PlayerPane model
- [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
- [ ] `View()` reads from store: current track, state, volume
- [ ] `Update()` handles: `Space` → pause/play cmd, `n/p` → skip cmds, `+/-` → volume cmds
- [ ] Handle `playbackStateFetchedMsg` → update store
- [ ] Handle `204 No Content` → empty state display
- [ ] Test all key handlers return correct commands

### Task 2.7 — Polling loop
- [ ] Root model `Init()` starts `tea.Tick(1s)`
- [ ] On `tickMsg`: dispatch `fetchPlaybackState` command
- [ ] Local progress interpolation between ticks
- [ ] Test: tick produces fetchPlaybackState command

### Task 2.8 — Integration wiring
- [ ] Wire root app model to PlayerPane
- [ ] Header bar shows active device name
- [ ] Status bar shows player context keybindings when player pane is focused

---

## Acceptance Criteria

- [ ] Currently playing track visible within 1 second of app launch
- [ ] `Space` play/pause responds in under 200ms (optimistic update shown immediately)
- [ ] Seek bar updates every 1 second accurately
- [ ] Volume changes reflect immediately in UI, confirmed by next poll
- [ ] Shuffle/repeat state accurately reflects Spotify state
- [ ] "Nothing playing" state shown cleanly when no active session
- [ ] All API functions tested; all pane Update() handlers tested
- [ ] No crashes on Spotify returning 204, 429, 503

---

## Out of Scope

- Web Playback SDK (browser-based playback — not a terminal concern)
- Lyrics display
- Audio waveform visualization
- Crossfade control

---

*Last updated: 2026-02-21*
