---
title: "PodcastPlayback pane"
feature: 18-podcasts
status: open
---

## Background

The PodcastPlayback pane replaces the visualizer-based layout used for music
with a 30/70 vertical split. The left panel shows episode info (title, show
name, transport controls, volume). The right panel shows episode metadata
(publish date, publisher, description) with a progress bar pinned to the
bottom.

## Design

### File: `internal/ui/panes/podcastplayback.go` (new)

```go
package panes

import (
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PodcastPlaybackPane struct {
	BasePane
	store          state.StateReader
	theme          theme.Theme
	localProgressMs int
	seekBar        *components.SeekBar
	volumeBar      *components.GradientVolumeBar
}
```

### Constructor

```go
func NewPodcastPlaybackPane(store state.StateReader, th theme.Theme, focused bool) *PodcastPlaybackPane {
	return &PodcastPlaybackPane{
		BasePane: NewBasePane(focused),
		store:    store,
		theme:    th,
	}
}
```

### Pane metadata

- `ID() layout.PaneID` → `layout.PanePodcastPlayback`
- `Title() string` → `"Now Playing"`
- `ToggleKey() int` → `1`
- `Actions() []layout.Action` → shuffle, repeat, play/pause, volume

### `SetSize(width, height int)`

Store dimensions. Recompute content layout:

```
contentWidth = max(width, 10)
infoWidth    = contentWidth * 30 / 100 (min 24)
detailsWidth = contentWidth - infoWidth - 1 (gap)
```

### `Update(msg tea.Msg) (tea.Model, tea.Cmd)`

Handle keypresses:

| Key | Action |
|-----|--------|
| Space | Play/pause → emit `PlaybackRequestMsg{Action: "toggle"}` |
| ← | Seek backward 5s → emit `SeekIntentMsg{Delta: -5000}` |
| → | Seek forward 5s → emit `SeekIntentMsg{Delta: 5000}` |
| Shift+← | Previous episode → emit `PlaybackRequestMsg{Action: "previous"}` |
| Shift+→ | Next episode → emit `PlaybackRequestMsg{Action: "next"}` |
| + | Volume up → emit `VolumeIntentMsg{Delta: 5}` |
| - | Volume down → emit `VolumeIntentMsg{Delta: -5}` |
| s | Toggle shuffle → emit `PlaybackRequestMsg{Action: "shuffle"}` |
| r | Cycle repeat → emit `PlaybackRequestMsg{Action: "repeat"}` |

Also handle `PlaybackStateFetchedMsg` to update `localProgressMs`.

### `View() string`

Two states:

**Empty state** (no playback or not an episode):

```
No podcast playing

Press / to search for shows
Or select a show from Followed Shows
```

Centered vertically and horizontally using existing `EmptyState` component
style (primary text + hint).

**Playing state** (30/70 split):

Left panel (30%, bordered with "Episode Info" title):
- Episode title (truncated, bold)
- Show name (dimmed/subtle)
- Blank separator
- Transport controls row (same layout as music NowPlaying controls component)
- Volume bar (`♪ ████▎□□□ 65%`)

Right panel (70%, no border):
- Metadata line: `Released: <date> · Duration: <Xm>`
- Publisher line: `Publisher: <show.publisher>`
- Blank separator
- Episode description (`item.description`, truncated to fit)
- Flex space (filling remaining space above progress bar)
- Progress bar pinned to bottom: `-- current_time ·· progress_bar ·· total_time --`

Progress bar characters: `█` for played portion, `░` for remaining.
Calculated from `progress_ms / duration_ms`. Updates every tick.

Source data: read from `p.store.PlaybackState()`:
- `CurrentlyPlayingType == "episode"` → use `Episode` field
- Otherwise → show empty state

### Components used

- `components.SeekBar` — same as music NowPlaying seek bar
- `components.GradientVolumeBar` — same as music NowPlaying volume bar
- `components.Controls` — same transport controls component
- `components.EmptyState` — for empty state rendering

### Tests: `internal/ui/panes/podcastplayback_test.go`

```go
func TestPodcastPlaybackPane_ID(t *testing.T)       // ID == PanePodcastPlayback
func TestPodcastPlaybackPane_Title(t *testing.T)     // Title == "Now Playing"
func TestPodcastPlaybackPane_ToggleKey(t *testing.T) // ToggleKey == 1
func TestPodcastPlaybackPane_EmptyState(t *testing.T) // nil playback → "No podcast playing"
func TestPodcastPlaybackPane_EpisodeView(t *testing.T) // episode playback → contains title
func TestPodcastPlaybackPane_ProgressBar(t *testing.T) // progress bar format matches pattern
```

## Acceptance Criteria

- [ ] `PodcastPlaybackPane` compiles with `ID`, `Title`, `ToggleKey`, `Actions`, `SetSize`, `Update`, `View`
- [ ] Empty state shows "No podcast playing" with hint lines when no episode is active
- [ ] Playing state shows 30/70 left-right split
- [ ] Left panel title "Episode Info" rendered in border
- [ ] Left panel shows episode title (bold) and show name (dimmed)
- [ ] Transport controls rendered in left panel
- [ ] Volume bar rendered in left panel
- [ ] Right panel shows metadata line: `Released: <date> · Duration: <Xm>`
- [ ] Right panel shows `Publisher: <show.publisher>`
- [ ] Right panel shows episode description (truncated)
- [ ] Right panel shows progress bar at bottom
- [ ] Progress bar format: `-- 12:34 ·· ████████░░░░░░ ·· 45:00 --`
- [ ] Space, ←/→, Shift+←/Shift+→, +/-, s, r all emit correct messages
- [ ] All tests pass
- [ ] `go build ./internal/ui/panes/...` passes

## Tasks

- [ ] Create `internal/ui/panes/podcastplayback.go` with `PodcastPlaybackPane` struct embedding `BasePane`
- [ ] Implement `NewPodcastPlaybackPane` constructor with store + theme params
- [ ] Implement `ID()`, `Title()`, `ToggleKey()` metadata methods
- [ ] Add test: `TestPodcastPlaybackPane_ID` — verify ID == `PanePodcastPlayback` — in `podcastplayback_test.go`
- [ ] Add test: `TestPodcastPlaybackPane_Title` — verify Title == `"Now Playing"` — in `podcastplayback_test.go`
- [ ] Add test: `TestPodcastPlaybackPane_ToggleKey` — verify ToggleKey == 1 — in `podcastplayback_test.go`
- [ ] Implement `View()` empty state: centered "No podcast playing" + hint lines when playback is nil or `currently_playing_type != "episode"`
- [ ] Add test: `TestPodcastPlaybackPane_EmptyState` — set nil playback state, verify output contains "No podcast playing" — in `podcastplayback_test.go`
- [ ] Implement `View()` playing state: 30/70 left-right split with episode info (left) + details + progress bar (right)
- [ ] Add test: `TestPodcastPlaybackPane_EpisodeView` — populate store with episode playback, verify output contains episode title and show name — in `podcastplayback_test.go`
- [ ] Add test: `TestPodcastPlaybackPane_ProgressBar` — verify progress bar format matches `-- current_time ·· progress_bar ·· total_time --` — in `podcastplayback_test.go`
- [ ] Implement key handling in `Update()`: Space, ←/→, Shift+←/Shift+→, +/-, s, r → emit correct Msg types
- [ ] Add test: `TestPodcastPlaybackPane_KeyHandling` — send each keypress via `Update()`, verify correct Msg type returned — in `podcastplayback_test.go`
- [ ] Implement `SetSize()` computing 30/70 layout with min infoWidth of 24
- [ ] Run `go test ./internal/ui/panes/... -run "TestPodcastPlayback" -v` — all pass
