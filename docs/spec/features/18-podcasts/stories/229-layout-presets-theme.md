---
title: "Layout, presets, theme border tokens"
feature: 18-podcasts
status: done
---

## Background

The podcasts page needs 4 new pane IDs, a new PagePodcasts page constant, 2
presets (Listening and Dashboard), an updated 3-page cycle, and 4 new border
color tokens across all 13 themes.

## Design

### Pane IDs + Page IDs (`internal/ui/layout/pane.go`)

Add to the `PaneID` iota:

```go
const (
	PaneNowPlaying     PaneID = iota // 0
	PaneQueue                        // 1
	PanePlaylists                    // 2
	PaneAlbums                       // 3
	PaneLikedSongs                   // 4
	PaneRecentlyPlayed               // 5
	PaneTopTracks                    // 6
	PaneTopArtists                   // 7
	PaneNetworkLog                   // 8
	PaneGatewayHealth                // 9
	PanePollingTraffic               // 10
	PaneGatewayLive                  // 11
	PanePodcastPlayback              // 12
	PaneShowEpisodes                 // 13
	PaneFollowedShows                // 14
	PaneSavedEpisodes                // 15
)
```

Add `PagePodcasts` to the `PageID` iota:

```go
const (
	PageMusic    PageID = iota // 0
	PageStats                  // 1
	PagePodcasts               // 2
)
```

### Presets (`internal/ui/layout/presets.go`)

Add two new presets and a page preset slice:

```go
var PresetPodcastListening = Preset{
	Name: "Listening",
	Visible: map[PaneID]bool{
		PanePodcastPlayback: true,
		PaneShowEpisodes:    true,
		PaneFollowedShows:   true,
	},
	Grid: []Row{
		{HeightWeight: 2, Cells: []Cell{{PaneID: PanePodcastPlayback, WidthWeight: 1}}},
		{HeightWeight: 3, Cells: []Cell{
			{PaneID: PaneShowEpisodes, WidthWeight: 55},
			{PaneID: PaneFollowedShows, WidthWeight: 45},
		}},
	},
}

var PresetPodcastDashboard = Preset{
	Name: "Dashboard",
	Visible: map[PaneID]bool{
		PanePodcastPlayback: true,
		PaneShowEpisodes:    true,
		PaneFollowedShows:   true,
		PaneSavedEpisodes:   true,
	},
	Grid: []Row{
		{HeightWeight: 2, Cells: []Cell{{PaneID: PanePodcastPlayback, WidthWeight: 1}}},
		{HeightWeight: 3, Cells: []Cell{
			{PaneID: PaneShowEpisodes, WidthWeight: 35},
			{PaneID: PaneFollowedShows, WidthWeight: 25},
			{PaneID: PaneSavedEpisodes, WidthWeight: 40},
		}},
	},
}

var PagePodcastsPresets = []Preset{PresetPodcastListening, PresetPodcastDashboard}
```

### Layout Manager (`internal/ui/layout/layout.go`)

In `NewManager()`, register podcasts page:

```go
presets: map[PageID][]Preset{
	PageMusic:    PageMusicPresets,
	PageStats:    PageStatsPresets,
	PagePodcasts: PagePodcastsPresets,
},
activePreset: map[PageID]int{
	PageMusic:    0,
	PageStats:    0,
	PagePodcasts: 0,
},
numberOfPages: 3, // update from 2 to 3
```

Update `TogglePage()` to 3-cycle:

```go
func (m *Manager) TogglePage() {
	switch m.activePage {
	case PageMusic:
		m.activePage = PagePodcasts
	case PagePodcasts:
		m.activePage = PageStats
	default:
		m.activePage = PageMusic
	}
	m.hidden = make(map[PaneID]bool)
	m.focusIndex = 0
	m.recompute()
}
```

Add `PagePodcasts` guard to `PaneAt()` focus rotation — order on podcasts page:
PodcastPlayback → ShowEpisodes → FollowedShows → SavedEpisodes

### Border token assignment (`internal/ui/layout/border.go`)

Add `PanePodcastPlayback`..`PaneSavedEpisodes` cases to `PaneBorderColor`:

```go
case PanePodcastPlayback:
	return t.PaneBorderPodcastPlayback()
case PaneShowEpisodes:
	return t.PaneBorderShowEpisodes()
case PaneFollowedShows:
	return t.PaneBorderFollowedShows()
case PaneSavedEpisodes:
	return t.PaneBorderSavedEpisodes()
```

### Theme interface (`internal/ui/theme/theme.go`)

Add 4 new methods to the `Theme` interface:

```go
PaneBorderPodcastPlayback() lipgloss.Color
PaneBorderShowEpisodes() lipgloss.Color
PaneBorderFollowedShows() lipgloss.Color
PaneBorderSavedEpisodes() lipgloss.Color
```

### ConfigTheme implementation (`internal/ui/theme/config_theme.go`)

Add to `paneBorderColors` struct:

```go
PodcastPlayback  string `toml:"podcast_playback"`
ShowEpisodes     string `toml:"show_episodes"`
FollowedShows    string `toml:"followed_shows"`
SavedEpisodes    string `toml:"saved_episodes"`
```

Add 4 methods:

```go
func (t *ConfigTheme) PaneBorderPodcastPlayback() lipgloss.Color { return lipgloss.Color(t.pb.PodcastPlayback) }
func (t *ConfigTheme) PaneBorderShowEpisodes() lipgloss.Color    { return lipgloss.Color(t.pb.ShowEpisodes) }
func (t *ConfigTheme) PaneBorderFollowedShows() lipgloss.Color   { return lipgloss.Color(t.pb.FollowedShows) }
func (t *ConfigTheme) PaneBorderSavedEpisodes() lipgloss.Color   { return lipgloss.Color(t.pb.SavedEpisodes) }
```

### Theme TOML files (13 files)

Add to the `[pane_border_colors]` section of each theme. The 4 new border tokens
follow the same hue mapping pattern as existing panes.

**Dark themes** (black, dracula, monokai, tokyonight, nord, catppuccin, rosepine, synthwave, gruvbox):

```toml
podcast_playback = "#ff6ac1"
show_episodes    = "#bd93f9"
followed_shows   = "#ff9f6e"
saved_episodes   = "#ff6e6e"
```

**Mono-dark**:

```toml
podcast_playback = "#aaaaaa"
show_episodes    = "#999999"
followed_shows   = "#888888"
saved_episodes   = "#777777"
```

**Light themes** (light, solarized):

```toml
podcast_playback = "#d63384"
show_episodes    = "#6f42c1"
followed_shows   = "#e86a33"
saved_episodes   = "#dc3545"
```

**Mono-light**:

```toml
podcast_playback = "#777777"
show_episodes    = "#888888"
followed_shows   = "#999999"
saved_episodes   = "#aaaaaa"
```

### Layout tests (`internal/ui/layout/layout_test.go`)

Update existing tests for the 3-page cycle:

- `TestTogglePage_SwitchesBetweenPages`: Music → Podcasts → Stats → Music
  (update expected order)
- `TestTogglePage_ClearsHiddenState`: Add third call to cycle back
- All Stats page tests (`TestTogglePane_StatsPage_*`, `TestLayoutManager_MinHeight`,
  `TestPresetStats_*`, `TestRecompute_StatsPage_*`): change `m.TogglePage()` to
  `m.TogglePage(); m.TogglePage()` to reach Stats through Podcasts
- `TestFullLifecycle`: Update expected page after toggles
- `TestRecompute_StatsPage_FocusOrder`: Expect Stats pane IDs after two toggles

## Acceptance Criteria

- [ ] `PanePodcastPlayback`..`PaneSavedEpisodes` compile as PaneID values
- [ ] `PagePodcasts` = 2 in the PageID iota
- [ ] `PresetPodcastListening` has correct 55/45 split for row 2
- [ ] `PresetPodcastDashboard` has correct 35/25/40 split for row 2
- [ ] `NewManager` registers `PagePodcastsPresets` and initializes `activePreset[PagePodcasts] = 0`
- [ ] `TogglePage()` cycles Music → Podcasts → Stats → Music
- [ ] `PaneAt()` on podcasts page returns podcast panes in correct focus order
- [ ] `PaneBorderColor` returns correct token for all 4 new PaneIDs
- [ ] `Theme` interface has 4 new methods; `ConfigTheme` implements them
- [ ] All 13 theme TOML files have 4 new border color entries and load without error
- [ ] All layout tests pass with 3-page cycle
- [ ] `go build ./internal/ui/layout/... ./internal/ui/theme/...` passes

## Tasks

- [ ] Add 4 pane IDs (`PanePodcastPlayback`..`PaneSavedEpisodes`) to PaneID iota in `internal/ui/layout/pane.go`
- [ ] Add `PagePodcasts` to PageID iota (order: Music=0, Stats=1, Podcasts=2)
- [ ] Add test: verify `PagePodcasts == 2` — in `internal/ui/layout/layout_test.go`
- [ ] Add `PresetPodcastListening` and `PresetPodcastDashboard` to `internal/ui/layout/presets.go`
- [ ] Add `PagePodcastsPresets` slice
- [ ] Add tests: verify `PresetPodcastListening` row 2 split is 55/45, `PresetPodcastDashboard` row 2 split is 35/25/40 — in `internal/ui/layout/layout_test.go`
- [ ] Register PagePodcasts in `NewManager` in `internal/ui/layout/layout.go` (presets + activePreset)
- [ ] Add test: verify `NewManager` has `PagePodcastsPresets` registered and `activePreset[PagePodcasts] == 0` — in `layout_test.go`
- [ ] Update `TogglePage()` to 3-cycle (Music → Podcasts → Stats → Music)
- [ ] Update `TestTogglePage_SwitchesBetweenPages` to expect Music → Podcasts → Stats → Music
- [ ] Update `TestTogglePage_ClearsHiddenState` to call TogglePage 3 times
- [ ] Update all Stats page tests to reach Stats via two TogglePage calls (Music → Podcasts → Stats): `TestTogglePane_StatsPage_*`, `TestLayoutManager_MinHeight`, `TestPresetStats_*`, `TestRecompute_StatsPage_*`, `TestFullLifecycle`, `TestRecompute_StatsPage_FocusOrder`
- [ ] Update `PaneAt()` focus rotation for podcasts page (PodcastPlayback → ShowEpisodes → FollowedShows → SavedEpisodes)
- [ ] Add test: verify `PaneAt()` on podcasts page returns correct focus order — in `layout_test.go`
- [ ] Add 4 `PanePodcastPlayback`..`PaneSavedEpisodes` cases to `PaneBorderColor` in `internal/ui/layout/border.go`
- [ ] Add test: verify `PaneBorderColor` returns distinct tokens for each new PaneID — in `border_test.go`
- [ ] Add 4 methods to `Theme` interface in `internal/ui/theme/theme.go`
- [ ] Verify compilation: all existing theme implementations still satisfy `Theme` interface
- [ ] Add 4 TOML fields to `paneBorderColors` struct in `internal/ui/theme/config_theme.go`
- [ ] Add 4 `ConfigTheme` methods implementing new interface methods
- [ ] Add test: verify `ConfigTheme` implements new border methods — in `config_theme_test.go`
- [ ] Update all 13 theme TOML files with 4 new border color entries
- [ ] Add test: verify all 13 themes load without error and produce valid colors — in `theme_test.go`
- [ ] Run `go test ./internal/ui/layout/... ./internal/ui/theme/...` — all pass
