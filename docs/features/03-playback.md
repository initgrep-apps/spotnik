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

## Goal

Show what's currently playing and give the user full control over playback from the terminal.
This is the core feature — the one people open the app for. It must be rock-solid and responsive.

---

## Feature Acceptance Criteria

- [ ] Currently playing track (name, artist, album) visible within 1 second of app launch
- [ ] `Space` play/pause responds in under 200ms (optimistic update shown immediately)
- [ ] Seek bar updates every 1 second accurately via local interpolation
- [ ] Volume changes reflect immediately in UI (optimistic), confirmed by next poll
- [ ] Shuffle/repeat state accurately reflects Spotify state after each poll
- [ ] "Nothing playing" empty state shown cleanly when Spotify returns 204
- [ ] All API functions tested with httptest mocks; all pane Update() handlers tested
- [ ] No crashes on 204 (nothing playing), 429 (rate limited), or 503 (Spotify down)

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
│  Blinding Lights              │  ← Track name (bold, `TextPrimary()` token)
│  The Weeknd                   │  ← Artist (`TextSecondary()` token)
│  After Hours                  │  ← Album (`TextMuted()` token)
│                               │
│  ████████████████░░░░░░░░░░   │  ← Seek bar (`SeekBar()` token fill)
│  2:34 ──────────────── 4:12   │  ← Time row
│                               │
│  ⏮   ⏸   ⏭      🔀   🔁      │  ← Controls (`PlayingIndicator()` token when active)
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

> **Ownership:** Feature 03 owns the tick loop. See `docs/ARCHITECTURE.md` → "Polling Ownership"
> for rules on which features participate in the tick cycle.

- **Interval**: every 1000ms
- **Endpoint**: `GET /me/player` (includes full state: track, progress, device, shuffle, repeat, volume)
- **On 204 No Content**: Spotify returns 204 when nothing is playing — update state to empty
- **Nil state:** When `GET /me/player` returns 204, set `store.PlaybackState = nil` and
  `store.CurrentTrack = nil`. PlayerPane's `View()` checks for nil and renders the empty state.
- **On 429**: back off for `Retry-After` seconds, show status bar message
- **Progress interpolation**: between polls, increment `progressMs` by 1000ms each tick locally
  so the seek bar moves smoothly without hammering the API
- **Local interpolation state:** The PlayerPane model holds `localProgressMs int` as pane-local
  state (not in the Store). It increments by 1000 on each tick when playing, and resets to the
  server value when `playbackStateFetchedMsg` arrives.

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
| Shuffle off | `🔀` in `TextSecondary()` token color |
| Shuffle on | `🔀` in `PlayingIndicator()` token color |
| Repeat off | `🔁` in `TextSecondary()` token color |
| Repeat context | `🔁` in `PlayingIndicator()` token color |
| Repeat track | `🔂` in `PlayingIndicator()` token color |

---

## Keymap (Player Pane Focus)

> **Focus:** The root model passes a `focused bool` to each pane. Panes only handle
> key events when focused. The root model sets focus based on the active pane state.

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

---

## Task Breakdown

### Task 2.1 — API models

**Description:** Define Go structs that map Spotify's playback JSON responses to typed models.

**Files:** `internal/api/models.go`, `internal/api/models_test.go`

**Implementation steps:**
- [ ] Define `PlaybackState` struct (all fields from Spotify response)
- [ ] Define `Track`, `Artist`, `Album`, `SimpleAlbum` structs
- [ ] Write JSON unmarshaling tests using `testdata/fixtures/`

**Acceptance criteria:**
- `PlaybackState` struct maps all fields from Spotify response (is_playing, progress_ms, item, device, shuffle_state, repeat_state, volume)
- `Track` has ID, Name, URI, DurationMs, Artists, Album
- `Artist` has ID, Name
- `Album` has ID, Name
- All structs deserialize from Spotify JSON fixtures correctly

**Tests:**

*Unit tests:*
- `TestPlaybackState_Unmarshal` — parse fixture JSON into correct struct
- `TestPlaybackState_Unmarshal_NowPlaying` — is_playing=true populates correctly
- `TestTrack_Unmarshal` — parse track JSON with artists and album
- `TestPlaybackState_NilItem` — 204-like response with null item field

---

### Task 2.2 — Player API calls

**Description:** Implement HTTP client methods for all Spotify playback endpoints.

**Files:** `internal/api/player.go`, `internal/api/player_test.go`

**Implementation steps:**
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

**Acceptance criteria:**
- Each method sends correct HTTP method + path + body to Spotify API
- Authorization header set with Bearer token
- 204 response returns nil state (not error) for GetPlaybackState
- 429 response returns error with Retry-After context
- Volume clamped to 0-100 before API call

**Tests:**

*Unit tests (all using httptest.NewServer):*
- `TestGetPlaybackState_Success` — returns parsed PlaybackState
- `TestGetPlaybackState_204` — returns nil, no error
- `TestGetPlaybackState_429` — returns error with retry info
- `TestPlay_SendsCorrectBody` — verifies context_uri or uris in request body
- `TestPause_SendsPUT` — correct method and path
- `TestNext_SendsPOST` — correct method and path
- `TestPrevious_SendsPOST` — correct method and path
- `TestSeek_SendsPositionMs` — query param contains position
- `TestSetVolume_ClampsRange` — volume > 100 sent as 100, < 0 as 0
- `TestSetShuffle_SendsState` — query param true/false
- `TestSetRepeat_SendsMode` — query param off/context/track

---

### Task 2.3 — Seek bar component

**Description:** Build a progress bar component that renders elapsed/total time with a visual fill bar.

**Files:** `internal/ui/components/progress.go`, `internal/ui/components/progress_test.go`

**Implementation steps:**
- [ ] `ProgressBar` struct with `Width`, `FilledColor`, `EmptyColor`
- [ ] `Render(progressMs, durationMs int) string`
- [ ] Time label row: `"2:34 ─────────────── 4:12"` — dynamic fill
- [ ] Test: various progress values, zero, full, over-full
- [ ] Test: correct width at different terminal widths

**Acceptance criteria:**
- Renders filled (`█`) and empty (`░`) characters proportional to progress/duration
- Time labels show elapsed (left) and total (right) in m:ss format
- Width adapts to available space (passed as parameter)
- Zero progress shows all empty; full progress shows all filled
- Colors come from theme tokens (`SeekBar()` for fill, `Surface()` for empty)

**Tests:**

*Unit tests:*
- `TestProgressBar_ZeroProgress` — all empty characters
- `TestProgressBar_HalfProgress` — roughly half filled
- `TestProgressBar_FullProgress` — all filled characters
- `TestProgressBar_TimeLabels` — "2:34" and "4:12" format
- `TestProgressBar_ZeroDuration` — handles division by zero gracefully
- `TestProgressBar_WidthAdapts` — different widths produce different bar lengths

---

### Task 2.4 — Volume bar component

**Description:** Build a volume indicator component with percentage label.

**Files:** `internal/ui/components/volume.go`, `internal/ui/components/volume_test.go`

**Implementation steps:**
- [ ] `VolumeBar` struct
- [ ] `Render(volume int) string`
- [ ] Clamp input: min 0, max 100
- [ ] Test: 0%, 50%, 100%, over-100 (clamped)

**Acceptance criteria:**
- Renders "VOL  ████████░░░░░░  65%" format
- Fixed 14-character bar width
- Volume clamped: min 0, max 100
- Percentage label shown on right

**Tests:**

*Unit tests:*
- `TestVolumeBar_Zero` — 0% shows all empty
- `TestVolumeBar_Fifty` — 50% shows roughly half filled
- `TestVolumeBar_Hundred` — 100% shows all filled
- `TestVolumeBar_OverHundred` — clamped to 100
- `TestVolumeBar_Negative` — clamped to 0

---

### Task 2.5 — Transport controls component

**Description:** Build the playback controls row showing play/pause, skip, shuffle, and repeat icons.

**Files:** `internal/ui/components/controls.go`, `internal/ui/components/controls_test.go`

**Implementation steps:**
- [ ] `Controls` struct with `IsPlaying`, `ShuffleOn`, `RepeatMode` fields
- [ ] `Render() string`
- [ ] Active states styled with `PlayingIndicator()` token color
- [ ] Inactive states styled with `TextSecondary()` token color
- [ ] Test: all state combinations render correct symbols/colors

**Acceptance criteria:**
- Renders ⏮ ⏸/▶ ⏭ 🔀 🔁/🔂 with correct spacing
- Playing state: shows ⏸ (pause); paused state: shows ▶ (play)
- Active shuffle/repeat: rendered in `PlayingIndicator()` token color
- Inactive: rendered in `TextSecondary()` token color
- Repeat modes: off=🔁 muted, context=🔁 active, track=🔂 active

**Tests:**

*Unit tests:*
- `TestControls_Playing_ShowsPause` — ⏸ symbol when is_playing=true
- `TestControls_Paused_ShowsPlay` — ▶ symbol when is_playing=false
- `TestControls_ShuffleOn` — 🔀 rendered with active style
- `TestControls_ShuffleOff` — 🔀 rendered with muted style
- `TestControls_RepeatOff` — 🔁 muted
- `TestControls_RepeatContext` — 🔁 active
- `TestControls_RepeatTrack` — 🔂 active

---

### Task 2.6 — PlayerPane model

**Description:** Implement the central pane's Bubble Tea model that ties together all components and handles keybindings.

**Files:** `internal/ui/panes/player.go`, `internal/ui/panes/player_test.go`

**Implementation steps:**
- [ ] Implement `tea.Model`: `Init()`, `Update()`, `View()`
- [ ] `View()` reads from store: current track, state, volume
- [ ] `Update()` handles: `Space` → pause/play cmd, `n/p` → skip cmds, `+/-` → volume cmds
- [ ] Handle `playbackStateFetchedMsg` → update store
- [ ] Handle `204 No Content` → empty state display
- [ ] Test all key handlers return correct commands

**Acceptance criteria:**
- `View()` reads from store: current track, playback state, volume
- `View()` renders empty state when store has nil PlaybackState
- `Update()` handles Space→play/pause, n/p→skip, +/-→volume, s→shuffle, r→repeat
- Each key handler returns the correct tea.Cmd (never calls API directly)
- `localProgressMs` increments on tick when playing, resets on `playbackStateFetchedMsg`

**Tests:**

*Unit tests:*
- `TestPlayerPane_View_NowPlaying` — renders track name, artist, album
- `TestPlayerPane_View_EmptyState` — renders "Nothing playing" when nil
- `TestPlayerPane_Update_Space_WhenPlaying` — returns pause command
- `TestPlayerPane_Update_Space_WhenPaused` — returns play command
- `TestPlayerPane_Update_N_SkipsNext` — returns next command
- `TestPlayerPane_Update_P_SkipsPrev` — returns previous command
- `TestPlayerPane_Update_Plus_VolUp` — returns volume +5 command
- `TestPlayerPane_Update_Minus_VolDown` — returns volume -5 command
- `TestPlayerPane_Update_S_TogglesShuffle` — returns shuffle toggle command
- `TestPlayerPane_Update_R_CyclesRepeat` — cycles off→context→track
- `TestPlayerPane_Update_PlaybackFetched` — updates store, resets localProgressMs
- `TestPlayerPane_Update_IgnoresKeysWhenNotFocused` — returns nil cmd when focused=false

---

### Task 2.7 — Polling loop

**Description:** Set up the root model's tick-based polling that keeps playback state in sync with Spotify.

**Files:** `internal/app/app.go`, `internal/app/app_test.go`

**Implementation steps:**
- [ ] Root model `Init()` starts `tea.Tick(1s)`
- [ ] On `tickMsg`: dispatch `fetchPlaybackState` command
- [ ] Local progress interpolation between ticks
- [ ] Test: tick produces fetchPlaybackState command

**Acceptance criteria:**
- `Init()` returns batch: fetchPlaybackState + tickEvery(1s) + fetchLibrary
- On `tickMsg`: dispatches fetchPlaybackState and reschedules tick
- Local progress increments by 1000ms between polls when playing

**Tests:**

*Unit tests:*
- `TestApp_Init_ReturnsBatch` — Init returns non-nil batch command
- `TestApp_Update_TickMsg_DispatchesFetch` — tickMsg produces fetch command

*Integration tests:*
- `TestPollingLoop_FetchesAndUpdatesStore` — tick → fetch → playbackStateFetchedMsg → store updated with new track data

---

### Task 2.8 — Integration wiring

**Description:** Wire the PlayerPane into the root app model with header/status bar integration.

**Files:** `internal/app/app.go` (modify)

**Implementation steps:**
- [ ] Wire root app model to PlayerPane
- [ ] Header bar shows active device name
- [ ] Status bar shows player context keybindings when player pane is focused

**Acceptance criteria:**
- Root app model contains PlayerPane
- Header bar shows active device name from store
- Status bar shows player-context keybindings when player pane is focused

**Tests:**

*Integration tests:*
- `TestApp_PlayerPaneRouting` — key events routed to player pane when focused
- `TestApp_HeaderShowsDevice` — device name from playback state appears in header view

---

## Out of Scope

- Web Playback SDK (browser-based playback — not a terminal concern)
- Lyrics display
- Audio waveform visualization
- Crossfade control

---

*Last updated: 2026-03-22*
