---
title: "Layout: reduce NowPlaying row weights, remove MinHeight, rename Dashboard preset"
feature: 17-album-art
status: open
---

## Background

Feature 17 set `HeightWeight: 2` and `MinHeight: 9` on the NowPlaying row in
Dashboard, Listening, Stats, and other presets. With album art rendering, the pane
now dominates 25–28% of the screen in presets meant for browsing — squeezing the
library and stats panes. The tier system that required MinHeight is also being
replaced (story 220), so MinHeight should be zeroed out on all NowPlaying rows.

Additionally `PresetDashboard.Name` reads `"Full Dashboard"` but the header shows
it as `"Music | Full Dashboard"` — verbose and inconsistent with the other preset
names.

## Design

### Target row weights

| Preset | NP HeightWeight | NP MinHeight | Other rows |
|--------|----------------|--------------|------------|
| Dashboard (0) | **1** (was 2) | **0** (was 9) | rows 1,2: weight **3** each |
| Listening (1) | 2 (unchanged) | **0** (was 9) | row 1: weight **3** (was 2) |
| Library (2) | 1 (unchanged) | **0** (was 9) | row 1: weight 4 (unchanged) |
| Discovery (3) | 1 (unchanged) | **0** (was 9) | rows 1,2: weight **3** each (was 2) |
| Stats | **1** (was 2) | **0** (was 9) | row 2 (NetworkLog): weight **3** (was 2) |

Resulting NowPlaying percentages at a typical terminal:

| Preset | NP% |
|--------|-----|
| Dashboard | 1/(1+3+3) = **14.3%** |
| Listening | 2/(2+3) = **40.0%** |
| Library | 1/(1+4) = **20.0%** |
| Discovery | 1/(1+3+3) = **14.3%** |
| Stats | 1/(1+3+3) = **14.3%** |

### Name rename

```go
// Before
var PresetDashboard = Preset{Name: "Full Dashboard", ...}

// After
var PresetDashboard = Preset{Name: "Dashboard", ...}
```

### Updated preset definitions

```go
var PresetDashboard = Preset{
    Name: "Dashboard",
    Visible: map[PaneID]bool{
        PaneNowPlaying: true, PaneQueue: true, PanePlaylists: true,
        PaneAlbums: true, PaneLikedSongs: true, PaneRecentlyPlayed: true,
        PaneTopTracks: true, PaneTopArtists: true,
    },
    Grid: []Row{
        {HeightWeight: 1, MinHeight: 0, Cells: []Cell{{PaneID: PaneNowPlaying, WidthWeight: 1}}},
        {HeightWeight: 3, Cells: []Cell{
            {PaneID: PanePlaylists, WidthWeight: 1},
            {PaneID: PaneAlbums, WidthWeight: 1},
            {PaneID: PaneLikedSongs, WidthWeight: 1},
        }},
        {HeightWeight: 3, Cells: []Cell{
            {PaneID: PaneQueue, WidthWeight: 1},
            {PaneID: PaneRecentlyPlayed, WidthWeight: 1},
            {PaneID: PaneTopTracks, WidthWeight: 1},
            {PaneID: PaneTopArtists, WidthWeight: 1},
        }},
    },
}

var PresetListening = Preset{
    Name: "Listening",
    Visible: map[PaneID]bool{
        PaneNowPlaying: true, PaneQueue: true, PaneRecentlyPlayed: true,
    },
    Grid: []Row{
        {HeightWeight: 2, MinHeight: 0, Cells: []Cell{{PaneID: PaneNowPlaying, WidthWeight: 1}}},
        {HeightWeight: 3, Cells: []Cell{
            {PaneID: PaneQueue, WidthWeight: 1},
            {PaneID: PaneRecentlyPlayed, WidthWeight: 1},
        }},
    },
}

var PresetLibrary = Preset{
    Name: "Library",
    Visible: map[PaneID]bool{
        PaneNowPlaying: true, PanePlaylists: true, PaneAlbums: true, PaneLikedSongs: true,
    },
    Grid: []Row{
        {HeightWeight: 1, MinHeight: 0, Cells: []Cell{{PaneID: PaneNowPlaying, WidthWeight: 1}}},
        {HeightWeight: 4, Cells: []Cell{
            {PaneID: PanePlaylists, WidthWeight: 1},
            {PaneID: PaneAlbums, WidthWeight: 1},
            {PaneID: PaneLikedSongs, WidthWeight: 1},
        }},
    },
}

var PresetDiscovery = Preset{
    Name: "Discovery",
    Visible: map[PaneID]bool{
        PaneNowPlaying: true, PaneTopTracks: true, PaneTopArtists: true, PaneRecentlyPlayed: true,
    },
    Grid: []Row{
        {HeightWeight: 1, MinHeight: 0, Cells: []Cell{{PaneID: PaneNowPlaying, WidthWeight: 1}}},
        {HeightWeight: 3, Cells: []Cell{
            {PaneID: PaneTopTracks, WidthWeight: 1},
            {PaneID: PaneTopArtists, WidthWeight: 1},
        }},
        {HeightWeight: 3, Cells: []Cell{{PaneID: PaneRecentlyPlayed, WidthWeight: 1}}},
    },
}

var PresetStats = Preset{
    Name: "Stats",
    Visible: map[PaneID]bool{
        PaneNowPlaying: true, PaneGatewayHealth: true,
        PanePollingTraffic: true, PaneGatewayLive: true, PaneNetworkLog: true,
    },
    Grid: []Row{
        {HeightWeight: 1, MinHeight: 0, Cells: []Cell{
            {PaneID: PaneNowPlaying, WidthWeight: 1},
        }},
        {HeightWeight: 3, Cells: []Cell{
            {PaneID: PaneGatewayHealth, WidthWeight: 1},
            {PaneID: PanePollingTraffic, WidthWeight: 1},
            {PaneID: PaneGatewayLive, WidthWeight: 3},
        }},
        {HeightWeight: 3, Cells: []Cell{
            {PaneID: PaneNetworkLog, WidthWeight: 1},
        }},
    },
}
```

## Acceptance Criteria

- [ ] `PresetDashboard.Name` is `"Dashboard"` (not `"Full Dashboard"`)
- [ ] NowPlaying row in Dashboard: HeightWeight=1, MinHeight=0
- [ ] NowPlaying row in Listening: HeightWeight=2, MinHeight=0
- [ ] NowPlaying row in Library: HeightWeight=1, MinHeight=0
- [ ] NowPlaying row in Discovery: HeightWeight=1, MinHeight=0
- [ ] NowPlaying row in Stats: HeightWeight=1, MinHeight=0
- [ ] Stats NetworkLog row: HeightWeight=3
- [ ] Listening second row (Queue/Recently): HeightWeight=3
- [ ] Discovery rows 1 and 2: HeightWeight=3 each
- [ ] `make ci` passes

## Tasks

- [ ] Update `internal/ui/layout/presets_test.go` to assert new HeightWeight, MinHeight,
      and Name values before touching `presets.go` (tests must fail first)
      - tests: `TestPresetDashboard` — assert Name=="Dashboard", Grid[0].HeightWeight==1,
        Grid[0].MinHeight==0, Grid[1].HeightWeight==3, Grid[2].HeightWeight==3
      - tests: `TestPresetStats_FlatThreeRows` — assert Grid[0].HeightWeight==1,
        Grid[2].HeightWeight==3
      - tests: `TestPresetStats_NowPlayingMinHeight` — assert Grid[0].MinHeight==0
      - tests: `TestPreset_MusicPage_NowPlayingMinHeight` — assert minHeight==0 for
        Dashboard, Listening, Library, Discovery

- [ ] Update `internal/ui/layout/presets.go` with the new preset definitions above
      - all five presets: Dashboard, Listening, Library, Discovery, Stats
      - run `rtk go test ./internal/ui/layout/... -v` — all tests must pass

- [ ] `make ci` passes
