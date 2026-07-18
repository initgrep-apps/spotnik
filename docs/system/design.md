# Design — btop-Inspired UI Specification

> **Authoritative design spec for Spotnik.** Responsive pane-based grid inspired by btop.
> Agents: every pixel = hard constraint, not suggestion.
> Previous frozen 3-column layout fully replaced.

---

## 0. Authority

Layout mechanics (grid, pages, presets, keys 1–8, page switch) live here. Primitive rendering (PaneChrome, Toast, Panel, HeaderBar, StatusBar, overlay chrome, onboarding panels) lives in `tui.md`. Where both apply (pane borders), this doc describes pane identity (color token, toggle key, pane ID); `tui.md` describes exact rendering contract.

---

## Overview

Old UI mimicked Spotify web player: 3 fixed columns (Library | Player | Queue). Poor terminal fit — text overflow, wasted space, no scroll guidance, web-app transplant.

Redesign draws from **btop**: pane-based responsive grid filling every terminal cell, preset system for curated layouts, embedded border shortcuts, dense colorful aesthetic.

### What Changes

| Aspect | Previous | Current |
|---|---|---|
| Layout | Fixed 3-column (22/50/28%) | 3-row responsive grid, 14 PaneIDs across 2 pages |
| Panes | 3 fixed + 2 alternatives | 10 player + 4 stats pane impls (NowPlaying shared), toggleable |
| Pages | None | Player + Stats, cycled by `0` |
| Presets | None (1/2/3 view switch) | `p` cycles 6 Player presets + 1 Stats |
| Pane toggle | None | `1`-`8` hide/show (btop-style, context-aware) |
| Podcasts | Separate page (4 panes) | Integrated into Player: content-aware NowPlaying + FollowedShows drill-down |
| Shortcuts | All in status bar | Embedded in pane borders (btop-style) |
| Filtering | None | In-pane `f` filter on every list |
| Visuals | Mono cyan bars | Gradient bars, braille visualizer, multi-color columns |
| Borders | Same color all panes | Per-pane accent colors |
| Content | Overflows | Hard-capped with truncation |
| Min terminal | 100x24 | 120x30 |

### What Stays

- Rounded corners (`╭╮╰╯`) exclusively
- Theme system with token-based colors (no hardcoded hex)
- Elm architecture (messages, commands, Store)
- Overlays float above grid (`github.com/rmhubbert/bubbletea-overlay`)
- Toast notifications (bottom-right)
- Splash + onboarding screens (full-screen, transitional)
- All existing Spotify API integration
- `t` theme switch shortcut, `?` help overlay
- Every overlay centered in screen
- `tea.WithAltScreen()` full-screen rendering

Check `bubbletea` skill for available components before building new ones.

---

## 1. Design Philosophy

1. **Information density** — every terminal cell earns place. No decorative whitespace, no empty panes.
2. **Pane independence** — each data category owns pane. Show/hide/rearrange via presets without affecting others.
3. **Space awareness** — hidden pane → space redistributes to visible siblings. Hidden row → remaining rows expand.
4. **Embedded discoverability** — shortcuts visible in pane borders always. No memorization, no help-screen diving. Like btop's `proc` title `filter/reverse/tree`.
5. **Preset-driven layouts** — curated configs beat user chaos. 6 presets cover 95% use cases.
6. **Nerd aesthetic** — braille-dot graphics, gradient bars, dense aligned tables, per-pane border colors. Developer tool, not web-app skin.
7. **Content containment** — pane content never overflows rectangle. Truncation mandatory.

---

## 2. Pane Definitions

### Player page (10 panes across 6 presets)

| Pane | ID | API Source | Toggle Key | Border Accent |
|---|---|---|---|---|
| Now Playing | `PaneNowPlaying` | `GET /me/player` + episode/show state | `1` | `PlayingIndicator()` green |
| Queue | `PaneQueue` | `GET /me/player/queue` | `2` | `Warning()` yellow |
| Followed Shows | `PaneFollowedShows` | `GET /me/shows` | `3` (Podcast preset) | `PaneBorderFollowedShows()` teal |
| Saved Episodes | `PaneSavedEpisodes` | `GET /me/episodes` | `4` (PodcastDashboard preset) | `PaneBorderSavedEpisodes()` green |
| Playlists | `PanePlaylists` | `GET /me/playlists` | `3` (Dashboard/Library) | `SectionHeader()` blue |
| Albums | `PaneAlbums` | `GET /me/albums` | `4` (Dashboard/Library) | `SeekBar()` cyan |
| Liked Songs | `PaneLikedSongs` | `GET /me/tracks` | `5` | `Success()` green |
| Recently Played | `PaneRecentlyPlayed` | `GET /me/player/recently-played` | `6` | `DeviceActive()` teal |
| Top Tracks | `PaneTopTracks` | `GET /me/top/tracks` | `7` | `KeyHint()` purple |
| Top Artists | `PaneTopArtists` | `GET /me/top/artists` | `8` | `Error()` pink/red |

Toggle keys **context-aware** — adapt by preset. On Podcast preset key `3`=FollowedShows, key `4`=SavedEpisodes. On Dashboard/Library key `3`=Playlists, key `4`=Albums.

NowPlaying **content-aware**: track UI when playing track, episode UI (show name, description, `i` for Episode Details overlay) when playing podcast.

FollowedShows **drill-down**: `Enter` on show → episode sub-view in same pane. `Esc` returns to show list.

### Stats page (5 panes)

| Pane | ID | Data Source | Toggle Key | Border Accent |
|---|---|---|---|---|
| Now Playing | `PaneNowPlaying` | `GET /me/player` | — | `PlayingIndicator()` green |
| Gateway Health | `PaneGatewayHealth` | `store.ReadEventsFrom(cursor)` — token bucket, slots, backoff, dedup | `2` | `PaneBorderRequestFlow()` orange/amber |
| Polling Traffic | `PanePollingTraffic` | `PollingSnapshotMsg` + store TTL sentinels | `3` | `PaneBorderRequestFlow()` orange/amber |
| Gateway Live | `PaneGatewayLive` | `store.ReadEventsFrom(cursor)` — 500-entry event stream | `4` | `PaneBorderRequestFlow()` orange/amber |
| Network Log | `PaneNetworkLog` | `store.ReadEventsFrom(cursor)` — GatewayEventLog (200-entry buffer) | `5` | `PaneBorderNetworkLog()` warm grey |

### Key Notes

- Toggle keys **context-aware**: same key → different panes by preset
- Keys `2`–`5` toggle Stats pane visibility
- `0` cycles Player → Stats → Player (2-cycle)
- Playback keys (`Space`, `+`, `-`, `s`, `r`, `v`, `←`, `→`, `Shift+←`, `Shift+→`) route to NowPlaying regardless of focus
- `i` opens Episode Details overlay when podcast episode playing
- `A` add-to-queue in search overlay + list panes
- `l` like/unlike in 6 track-displaying panes (not NowPlaying)
- NowPlaying: btop-style horizontal split — InfoBox (~1/3 left) + viz.Engine (~2/3 right); seek bar inside right panel between viz rows

### Pane Interface

Every pane implements (`internal/ui/layout/pane.go:72`):

```go
type Pane interface {
    tea.Model
    SetSize(width, height int)
    SetFocused(focused bool)
    IsFocused() bool
    ID() PaneID
    Title() string
    ToggleKey() int
    Actions() []Action
    SetTheme(th theme.Theme)
}

type Action struct {
    Key   string
    Label string
}
```

Optional: `FilterablePane` (`pane.go:98`), `FilterQueryPane` (`pane.go:107`).

`SetTheme` — table panes must rebuild tables with new column colors (lipgloss column styles baked at creation).

---

## 3. Layout Grid System

### Grid Model

Row-based grid. Each row = cells (panes) with relative width weights. Rows have relative height weights.

```
Grid = []Row
Row  = {HeightWeight int, Cells []Cell}
Cell = {PaneID, WidthWeight int}
```

### Space Distribution

1. Filter hidden cells
2. Filter empty rows (all cells hidden)
3. Distribute height by `HeightWeight`
4. Distribute width per row by `WidthWeight`
5. Last cell/row absorbs remainder

### Reserved Space

```
Total height = terminal rows
  - Header:     1 line
  - Status bar: 1 line
  - Content:    terminal rows - 2
```

Each pane content area = `Rect.Width - 2` × `Rect.Height - 2` (borders consume 1 char each side).

---

## 4. Pages, Pane Toggling, Preset Layouts

### Page Switching

- `0` cycles Player → Stats → Player (2-cycle)
- Each page has own preset cycle
- Switching preserves pane state on both sides
- Hidden-map resets on page switch

### Pane Toggling (btop-style, context-aware)

**Player page:**
- Keys `1`–`8` toggle corresponding pane visibility
- Toggle key mapping changes per preset — key `3`=Playlists in Dashboard/Library, key `3`=FollowedShows in Podcast preset
- Key `4`=Albums in Dashboard/Library, key `4`=SavedEpisodes in PodcastDashboard preset

**Auto-switch rules:**

| Trigger | Action |
|---|---|
| Podcast episode starts playing | Auto-switch to PodcastDashboard preset (if not already) |
| Music track starts playing | Auto-switch to Listening preset (if on Podcast preset) |
| User manually changes preset | Override auto-switch until next content-type change |

**Stats page:**
- Keys `2`–`5` toggle Stats diagnostic panes
- Key `1` unused on Stats page

**General:**
- Hidden pane → siblings expand
- All panes in row hidden → row collapses
- Hidden pane toggled back → reappears in original grid position
- Toggle state independent of presets — switching resets manual toggles

### Preset Cycling

`p` cycles preset layouts within current page. Each preset = hide/show bitmask. After last, wraps to first. Switching resets manual toggles.

### Player page Presets

#### Preset 0 — Dashboard (default)

8 panes visible across 3 rows. NowPlaying spans full width. No FollowedShows/SavedEpisodes.

```
╭─ ¹Now Playing ──────────────────╮ s shfl ╭─╮ r rpt ╭─╮ space play ╭─╮ +/- vol ╭─╮ v viz ╮  Row 1 (weight 1)
│ ╭─ Track Info ──────╮ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿              │
│ │ Martbaan          │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿              │
│ │ Samar Mehdi       │ ─── 1:41 ████████████████░░░░░░░░░░░░░░░  5:30 ──       │
│ │ ⇄  ▷  ≡  ↻        │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿              │
│ │ ♪ ███▎□□□ 65%     │ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿              │
│ ╰───────────────────╯ ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿              │
╰──────────────────────────────────────────────────────────────────────────────╯
╭─ ³Playlists ─────────╮╭─ ⁴Albums ────────────╮╭─ ⁵Liked Songs ──────╮  Row 2 (weight 3)
│  1  LoFi             ││  1  After Hours      ││  1  Blinding Lights  │
│  2  Soul             ││  2  OK Computer      ││  2  Save Your Tears  │
│  3  Workout          ││  3  In Rainbows      ││  3  Levitating       │
│  4  Best of Coke     ││  4  Blonde           ││  4  Peaches          │
│  ▼ more below        ││  ▼ more below        ││  ▼ more below        │
╰──────────────────────╯╰──────────────────────╯╰──────────────────────╯
╭─ ²Queue ──────╮╭─ ⁶Recent ─────╮╭─ ⁷Top Tracks ──╮╭─ ⁸Top Artists ─╮  Row 3 (weight 3)
│  1  Lil Boo   ││  1  Martbaan  ││  1  Blinding   ││  1  Weeknd     │
│  2  Street F  ││  2  Starboy   ││  2  Martbaan   ││  2  Drake      │
│  3  BIRDS     ││  3  Heat Wav  ││  3  Save Your  ││  3  Dua Lipa   │
│  ▼ more       ││  ▼ more       ││  ▼ more        ││  ▼ more        │
╰───────────────╯╰───────────────╯╰────────────────╯╰────────────────╯
```

**Grid:**
```
Row 1 (weight 1): [{NowPlaying, weight=1}]                              ← full width
Row 2 (weight 3): [{Playlists, weight=1}, {Albums, weight=1}, {LikedSongs, weight=1}]
Row 3 (weight 3): [{Queue, weight=1}, {RecentlyPlayed, weight=1}, {TopTracks, weight=1}, {TopArtists, weight=1}]
```

#### Preset 1 — Listening

NowPlaying expanded with large visualizer. Queue + RecentlyPlayed below.

```
╭─ ¹Now Playing ───────────────╮ s shfl ╭─╮ r rpt ╭─╮ space play ╭─╮ +/- vol ╭─╮ v viz ╮  Row 1 (weight 3)
│                                                                                  │
│ ╭─ Track Info ──────────────╮  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ │ Martbaan                  │  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ │ Samar Mehdi, June         │  ────────── 1:41 ████████░░░░░░░░░ 5:30 ───────    │
│ │ Martbaan (Album)          │  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ │ ⇄  ▷  ≡  ↻               │  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ │ ♪ █████▎□□□ 65%          │  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
│ ╰───────────────────────────╯  ⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿    │
╰──────────────────────────────────────────────────────────────────────────────────╯
╭─ ²Queue ──────────────────────╮╭─ ⁶Recently Played ────────────╮  Row 2 (weight 2)
│  #   Track          Artist    ││  #  Track          Played      │
│  1   Lil Boo Thang  P.Russell ││  1  Starboy        2m ago      │
│  2   Street Fighter Kamasi W  ││  2  Heat Waves     15m ago     │
│  3   BIRDS OF A     Billie E  ││  3  Levitating     1h ago      │
│  ▼ more below                 ││  ▼ more below                  │
╰───────────────────────────────╯╰────────────────────────────────╯
```

**Visible:** NowPlaying, Queue, RecentlyPlayed

#### Preset 2 — Podcast

NowPlaying with episode UI, FollowedShows, Queue.

```
╭─ ¹Now Playing ───────────────────────────────────────────────────────────╮  Row 1 (weight 2)
│  (content-aware: episode UI with show info, description, progress)         │
╰────────────────────────────────────────────────────────────────────────────╯
╭─ ³Followed Shows ───────────╮╭─ ²Queue ───────────────────────────────────╮  Row 2 (weight 3)
│  1  Show Name               ││  #   ◆/♪  Title          Artist/Show  Dur │
│  2  Another Show            ││  1   ◆    Episode Title  Show Name    34:12│
│  ▼ more below               ││  2   ♪    Track Name     Artist Name  3:45 │
╰─────────────────────────────╯╰────────────────────────────────────────────╯
```

**Visible:** NowPlaying, FollowedShows, Queue

#### Preset 3 — Library

NowPlaying compact strip (height < 8 → title-bar-embedded track info). Playlists, Albums, LikedSongs expanded.

```
╭─ ¹Now Playing ── Martbaan · Samar Mehdi ── ▶ 1:41/5:30 ──────────────╮  Row 1 (weight 1)
│  (height < 8: track info in title bar — compact title mode)            │
╰────────────────────────────────────────────────────────────────────────╯
╭─ ³Playlists ─────────╮╭─ ⁴Albums ────────────╮╭─ ⁵Liked Songs ──────╮  Row 2 (weight 4)
│  1  LoFi             ││  1  After Hours      ││  1  Blinding Lights  │
│  2  Best of Coke     ││  2  OK Computer      ││  2  Save Your Tears  │
│  3  Bosnia           ││  3  In Rainbows      ││  3  Levitating       │
│  4  Soul             ││  4  Blonde           ││  4  Peaches          │
│  5  Our soundtrack   ││  5  Random Access Mem││  5  Mood             │
│  6  Lofi Fruits      ││  6  The Dark Side    ││  6  Watermelon Sugar │
│  7  GT               ││  7  Currents         ││  7  Starboy          │
│  8  Running          ││  8  Rumours          ││  8  Positions        │
│  9  Lizzie Poole     ││  9  Abbey Road       ││  9  Heat Waves       │
│  10 My Playlist #21  ││  10 AM               ││  10 drivers license  │
│  ▼ more below        ││  ▼ more below        ││  ▼ more below        │
╰──────────────────────╯╰──────────────────────╯╰──────────────────────╯
```

**Visible:** NowPlaying (compact), Playlists, Albums, LikedSongs

#### Preset 4 — Discovery

NowPlaying compact strip. TopTracks, TopArtists, RecentlyPlayed expanded.

```
╭─ ¹Now Playing ── Martbaan · Samar Mehdi ── ▶ 1:41/5:30 ──────────────╮  Row 1 (weight 1)
│  (height < 8: track info in title bar — compact title mode)            │
╰────────────────────────────────────────────────────────────────────────╯
╭─ ⁷Top Tracks ────────────────╮╭─ ⁸Top Artists ───────────────────────╮  Row 2 (weight 2)
│  #  Track          Duration  ││  #  Artist          Popularity Flw   │
│  1  Blinding Ligh  4:12      ││  1  The Weeknd      ●●●●●  35M       │
│  2  Martbaan       5:30      ││  2  Drake           ●●●●○  22M       │
│  3  Save Your Te   3:35      ││  3  Dua Lipa        ●●●●○  12.5M     │
│  ▼ more below                ││  ▼ more below                        │
╰──────────────────────────────╯╰──────────────────────────────────────╯
╭─ ⁶Recently Played ──────────────────────────────────────────────────╮  Row 3 (weight 2)
│  #  Track                    Artist              Played             │
│  1  Starboy                  The Weeknd          2m ago             │
│  2  Heat Waves               Glass Animals       15m ago            │
│  3  Levitating               Dua Lipa            1h ago             │
│  ▼ more below                                                       │
╰──────────────────────────────────────────────────────────────────────╯
```

**Visible:** NowPlaying (compact), TopTracks, TopArtists, RecentlyPlayed

#### Preset 5 — PodcastDashboard

NowPlaying with episode UI, FollowedShows, SavedEpisodes, Queue.

```
╭─ ¹Now Playing ───────────────────────────────────────────────────────────╮  Row 1 (weight 2)
│  (content-aware: episode UI with show info, description, progress)         │
╰────────────────────────────────────────────────────────────────────────────╯
╭─ ³Followed Shows ──────────╮╭─ ⁴Saved Episodes ──────────────╮  Row 2 (weight 3)
│  1  Show Name              ││  1  Episode Title             │
│  2  Another Show           ││  2  Another Episode          │
│  ▼ more below              ││  ▼ more below                │
╰────────────────────────────╯╰──────────────────────────────╯
╭─ ²Queue ──────────────────────────────────────────────────────────────────╮  Row 3 (weight 3)
│  #   ◆/♪  Title                    Artist/Show          Duration         │
│  1   ◆    Episode Title            Show Name            34:12             │
│  2   ♪    Track Name               Artist Name          3:45              │
│  ▼ more below                                                             │
╰──────────────────────────────────────────────────────────────────────────╯
```

**Visible:** NowPlaying, FollowedShows, SavedEpisodes, Queue

### Player page Preset Summary

| Preset | Name | Visible Panes |
|---|---|---|
| 0 | Dashboard | NowPlaying, Queue, Playlists, Albums, LikedSongs, RecentlyPlayed, TopTracks, TopArtists (8) |
| 1 | Listening | NowPlaying, Queue, RecentlyPlayed |
| 2 | Podcast | NowPlaying, FollowedShows, Queue |
| 3 | Library | NowPlaying (compact), Playlists, Albums, LikedSongs |
| 4 | Discovery | NowPlaying (compact), TopTracks, TopArtists, RecentlyPlayed |
| 5 | PodcastDashboard | NowPlaying, FollowedShows, SavedEpisodes, Queue |

### Stats page Layout

5-pane, 3-row: NowPlaying compact strip (row 1) + 3 diagnostic panes side-by-side (row 2) + NetworkLog full-width (row 3).

```
╭─ ¹Now Playing ── Martbaan · Samar Mehdi ── ▶ 1:41/5:30 ──────────────╮  Row 1 (weight 1)
│  (height < 8: track info in title bar — compact title mode)            │
╰────────────────────────────────────────────────────────────────────────╯
╭─ ²Gateway Health ──────────╮╭─ ³Polling Traffic ──────────╮╭─ ⁴Gateway Live ──────────╮  Row 2 (weight 3, weights 1:1:3)
│  Tokens  ●●●●●●●●●●  10/10 ││  Playback  ▶ 1s · running  ││  event stream            │
│  Slots   ■□□□□  1/5        ││  Playlists  ◦ fresh        ││  (scrollable, filterable)│
│  Backoff none              ││  Albums     ⚠ 3m stale     ││                          │
│  Dedup   none              ││  Liked      ◦ fresh        ││                          │
│                            ││  Recent     ◦ fresh        ││                          │
╰────────────────────────────╯╰────────────────────────────╯╰──────────────────────────╯
╭─ ⁵Network Log ──────────────────────────────────────────────╮  Row 3 (weight 2)
│  Time      Method  Endpoint              Status  Latency  Priority  Decision │
│  12:03:45  GET     /me/player            200     45ms     ◷ bkgd    allowed │
│  (scrollable, filterable)                                                   │
╰──────────────────────────────────────────────────────────────────────────────╯
```

**Grid:**
```
Row 1 (weight 1): [{NowPlaying, weight=1}]                                  ← compact strip
Row 2 (weight 3): [{GatewayHealth, weight=1}, {PollingTraffic, weight=1}, {GatewayLive, weight=3}]  ← 1:1:3, GatewayLive dominant (#409)
Row 3 (weight 2): [{NetworkLog, weight=1}]                                  ← scrollable API log
```

### Preset/Toggle Behavior

On preset switch:
- Pane state (scroll, selection, filter text) preserved
- Focus moves to first visible pane if currently focused becomes hidden
- `renderGrid()` re-assembles immediately
- Manual pane toggles reset

---

## 5. Pane Border Chrome

See `tui.md §3.1` (PaneChrome) for full rendering contract: border anatomy, action-notch format, filter-mode preamble, glyph choices, roles, ASCII fallback.

**Border titles (#397 cleanup):**
- NowPlaying: "Now Playing" static (no episode info embedded)
- Drill-down panes: title `PaneName ── <drill-down name>` truncated (Albums: 30 chars, FollowedShows: 20 chars)
- All other panes: static title

---

## 6. Content Containment

**#1 rule: pane content never exceeds allocated rectangle.**

### Width Containment

- Every text line truncated to `paneWidth` chars
- `Truncate(text, maxWidth)` — rune-aware, `lipgloss.Width()` measurement, appends `…` when truncated
- `lipgloss.NewStyle().MaxWidth(paneWidth).MaxHeight(paneHeight)` wraps every pane `View()` as safety net
- `renderGrid()`: each cell wrapped in `lipgloss.NewStyle().Width(rect.Width).MaxWidth(rect.Width)` before `JoinHorizontal` — prevents cell pushing neighbors off-screen

### Vertical Containment

- Each pane computes `visibleItemCount` from allocated height
- Content beyond visible window accessible via `j`/`k` scrolling
- Scroll indicators (`▲` top, `▼` bottom) show when content extends
- `lipgloss.NewStyle().Height(paneHeight).MaxHeight(paneHeight)` enforces vertical cap
- Total `View()` output must equal exactly `terminalHeight` lines — pad if shorter, cap if taller

### Column Truncation (Dense Tables)

- Table columns get fixed proportions of pane width
- Each cell value individually truncated to column width
- Column widths recalculated on `SetSize()` — never hardcoded

### Empty Data Protocol (#406)

No blank rows. Table fills available height. Empty data sets render `uikit.EmptyState` via `PaneEmptyStatus()` factory (`uikit/empty_state.go:148`).

**5-status enum (`uikit/empty_state.go:14`):**

| Status | When | Message |
|---|---|---|
| `None` | Fetch succeeded, empty data | Action hint (e.g. "Nothing in queue · Press / to search") |
| `NeverFetched` | Initial state, no fetch yet | "Loading <category>..." |
| `Fetching` | In-flight | "Loading <category>..." |
| `Error` | Fetch returned error | "Unable to load <category>" |
| `RateLimited` | `IsThrottled=true` | Rate-limit hint with retry-after seconds |

`EmptyState` struct (`uikit/empty_state.go:36`): `Category`, `Text`, `Hint`, `Status`, `Width`, `Height`, `Theme`.

Panes using `PaneEmptyStatus` (Category field): `followedshows.go:258`, `topartists_pane.go:185`, `albums_pane.go:360`, `savedepisodes.go:118`, `playlists_pane.go:443`, `likedsongs_pane.go:159`, `recentlyplayed`, `toptracks`.

### Truncation Utility

`internal/ui/layout/truncate.go`:

```go
Truncate(s string, maxWidth int) string       // Truncate with "…" if too wide
PadRight(s string, width int) string          // Pad with spaces to exact width
TruncateOrPad(s string, width int) string     // Truncate or pad to exact width
```

All use `lipgloss.Width()` for rendered-width measurement, not `len()` or `utf8.RuneCountInString()`. Correctly handles CJK, combining marks, emoji.

---

## 7. Screen Stability

Terminal must never scroll. Entire UI renders within alt screen buffer.

- `tea.WithAltScreen()` — correct, must not change
- `View()` output exactly `terminalHeight` lines tall
- Grid content shorter → pad empty lines styled with `Base()` background
- Grid content overflow → height-capping prevents
- Every grid row height-capped to `Rect.Height`
- Grid + header + status bar sum to exactly `terminalHeight`

---

## 8. In-Pane Filtering

Every scrollable-list pane supports real-time filtering (btop process filter style).

### Behavior

1. `f` in focused pane toggles filter mode
2. Text input appears at top of pane content (below border)
3. Typing filters list in real-time (case-insensitive substring match)
4. `Esc` closes filter, restores full list
5. `Enter` selects first/current filtered result, closes filter
6. Filter state per-pane — each owns filter input + filtered items

### Filterable Fields

| Pane | Filter by |
|---|---|
| Playlists | Playlist name |
| Albums | Album name, artist name |
| Liked Songs | Track name, artist name |
| Queue | Track name, artist name |
| Recently Played | Track name, artist name |
| Top Tracks | Track name, artist name |
| Top Artists | Artist name |
| FollowedShows | Show name |
| SavedEpisodes | Episode name, show name |

### Visual Treatment

- Filter input: `TextPrimary()` text on `SurfaceAlt()` background
- Matching text: highlighted with `SelectedBg()` (optional, future)
- Filter active indicator in border: `filtering: "query"` replaces action shortcuts

### Filter Component

`internal/ui/components/filter.go`:

```go
type Filter struct {
    input    textinput.Model
    active   bool
    query    string
}

func (f *Filter) Toggle()
func (f *Filter) IsActive() bool
func (f *Filter) Query() string
func (f *Filter) Matches(text string) bool
func (f *Filter) Update(msg tea.Msg) tea.Cmd
func (f *Filter) View(width int) string
```

---

## 9. Dense Table Formatting

List panes render aligned columns with per-column colors.

### Column Layout (Queue example, #336)

```
  #   ◆/♪  Title                    Artist/Show          Dur
  1   ♪    Lil Boo Thang            Paul Russell         3:12
  2   ◆    Episode Title            Show Name            34:12
  3   ♪    BIRDS OF A FEATHER       Billie Eilish        3:30
```

Columns (FlexFactor `1:1:7:4:2`, `queue.go:35-41`):
- `#` index (FlexFactor 1)
- `type` — ◆ episode / ♪ track (FlexFactor 1, Priority 1)
- `title` (FlexFactor 7)
- `artist/show` (FlexFactor 4)
- `duration` (FlexFactor 2)

### Column Colors

| Column | Color Token | Purpose |
|---|---|---|
| Type (`◆`/`♪`) | `TextMuted()` | Episode/Track indicator |
| Title | `TextPrimary()` | Primary data — highest contrast |
| Artist/Show | `TextSecondary()` | Artist or show name |
| Duration/metadata | `TextMuted()` | Tertiary info |

**Selected row:** All columns override to `SelectedBg()` + `SelectedFg()`

### Column Ordering

Icon/glyph columns MUST be first data column. Order: `[Icon/Glyph] → [Primary Identifier] → [Secondary Info] → [Tertiary/Metadata]`

### Column Priority Thresholds

Columns have `Priority` field (1-3). Render-time filter by pane width:

| Priority | Label | Threshold | Behavior |
|---|---|---|---|
| 1 | Always | Any width | Always rendered |
| 2 | Default | ≥ 40 cols | Hidden when narrow |
| 3 | Wide-only | ≥ 60 cols | Hidden unless spacious |

Table wrapper (`components.Table`) applies in `rebuild()`. Width crossing threshold triggers rebuild.

### Column Header Guidelines

Short names: `Dur` (not `Duration`), `Pop` (not `Popularity`), `Pub` (not `Publisher`), `Eps` (not `Episodes`).

### Column Width Proportions

| Pane | Col 1 | Col 2 | Col 3 | Col 4 |
|---|---|---|---|---|
| Queue | # + ◆/♪ | Title 47% | Artist/Show 27% | Dur 13% |
| Playlists | Name 75% | Tracks 25% | — | — |
| Albums | Name 55% | Artist 30% | Year 15% | — |
| Liked Songs | Track 45% | Artist 40% | Dur 15% | — |
| Top Tracks | Track 45% | Artist 40% | Dur 15% | — |
| Top Artists | Name 55% | Pop 25% | Flw 20% | — |
| Recently Played | Track 45% | Artist 35% | Played 20% | — |

Column header row: `TableHeader()` color, not bold.

### Table Component

`internal/ui/components/table.go`:

```go
type Column struct {
    Header     string
    WeightPct  int
    Color      lipgloss.Color
}

type Table struct {
    columns  []Column
    width    int
}

func (t *Table) SetWidth(w int)
func (t *Table) RenderHeader() string
func (t *Table) RenderRow(values []string, selected bool, playing bool) string
```

---

## 10. Per-Pane Border Colors

Each pane has distinct border color providing visual identity without reading title.

| Pane | Focused Color | Unfocused Color |
|---|---|---|
| Now Playing | `PaneBorderNowPlaying()` (green) | Dimmed green |
| Queue | `PaneBorderQueue()` (yellow) | Dimmed yellow |
| Playlists | `PaneBorderPlaylists()` (blue) | Dimmed blue |
| Albums | `PaneBorderAlbums()` (cyan) | Dimmed cyan |
| Liked Songs | `PaneBorderLikedSongs()` (green) | Dimmed green |
| Recently Played | `PaneBorderRecentlyPlayed()` (teal) | Dimmed teal |
| Top Tracks | `PaneBorderTopTracks()` (purple) | Dimmed purple |
| Top Artists | `PaneBorderTopArtists()` (pink/red) | Dimmed pink |
| Followed Shows | `PaneBorderFollowedShows()` (teal) | Dimmed teal |
| Saved Episodes | `PaneBorderSavedEpisodes()` (green) | Dimmed green |
| Gateway Health | `PaneBorderRequestFlow()` (orange/amber) | Dimmed orange |
| Polling Traffic | `PaneBorderRequestFlow()` (orange/amber) | Dimmed orange |
| Gateway Live | `PaneBorderRequestFlow()` (orange/amber) | Dimmed orange |
| Network Log | `PaneBorderNetworkLog()` (warm grey) | Dimmed grey |

**Dimming strategy:** Unfocused borders = same hue at ~40% brightness. Achieved by separate unfocused tokens OR `lipgloss.NewStyle().Faint(true)` (simpler, theme-independent).

---

## 11. Visual Components

### Braille-Dot Audio Visualizer

Rendered in NowPlaying using Unicode braille (U+2800-U+28FF).

```
⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿⣿⣷⣿⣷⣿
```

**Behavior:**
- Simulated audio spectrum/waveform
- Animates on `tea.Tick` (200ms) when playing
- Static flat pattern when paused
- Width adapts to pane width
- Height: 2-4 lines depending on pane height (Preset 1 gets more rows)
- Colors: `VisualizerFg()` token, optional gradient

**Animation:**
- Component owns separate configurable `tea.Tick`, 200ms interval
- NowPlaying focused → arrow keys adjust tick speed (lower limit 200ms)
- `frameIndex int` counter, incremented per tick
- `View()` indexes precomputed frame table — no randomness, deterministic between ticks
- Paused: `frameIndex` stops, static flat-line
- Frame table: 30-50 pre-generated patterns looping smoothly

**3 animation patterns** (cycled via `v`):
- **Pattern 0 (Dual Sine Wave):** Two overlapping sines at different frequencies, flowing ocean motion. Default.
- **Pattern 1 (Standing Wave):** Counter-propagating wave interference, stationary nodes/antinodes, bars pulse in place.
- **Pattern 2 (Pulse/Ripple):** Gaussian peak travels left-to-right with trailing ripple, sonar ping sweep.

Pattern state local to pane (not in Store). `v` always routes to NowPlaying via `isPlaybackKey()`.

**Impl:** `internal/ui/components/visualizer.go`.

### Gradient-Colored Bars

Replace monochrome `SeekBar()` / `VolumeBar()` fills.

**Seek bar (#318 with debounce):**
```
████████████████░░░░░░░░░░░░░░░
```
- Fill gradient: `Gradient1()` → `Gradient2()` (left to right)
- Empty: `Surface()`
- Debounce: intent snapshot captured locally, seq-numbered. Stale ticks discarded on seq mismatch. `HasPending()` preserves local progress during debounce. `SeekAppliedMsg` confirm/cancel. Only final position sent to Spotify after 300ms idle.

**Volume bar (#269 with debounce):**
```
♪ ████▎□□□□□□□□□  65%
```
- Full cells: `█` (U+2588); fractional last cell: `▏▎▍▌▋▊▉` (1/8–7/8 fill, 8-step resolution)
- Empty cells: `□` in `Surface()`
- Low volume (0-33%): `Gradient1()` (green/cool)
- Mid volume (34-66%): `Gradient2()` (yellow/warm)
- High volume (67-100%): `Gradient3()` (red/hot)
- Volume = 0: `♪` in `TextMuted()`; all cells empty
- Volume > 0: `♪` in `Gradient1()`
- Debounce: `VolumeDebounceTickMsg` same protocol as seek. `+`/`-` keys return debounce cmd, not direct `PlaybackRequestMsg`.

**Impl:** `internal/ui/components/gradient.go`. Partial-block chars (`█▏▎▍▌▋▊▉□`) with fractional fill algorithm.

### NowPlaying Split Layout (btop-inspired)

Horizontal split inspired by btop's CPU pane.

**Proportions:**
- **Left (~1/3 width, min 28 chars):** InfoBox sub-pane — rounded border (`╭╮╰╯`), "Track Info" title:
  - Track name (bold, `TextPrimary()`)
  - Artist names (`TextSecondary()`)
  - Album name (`TextMuted()`)
  - Controls row (`⇄  ▷  ≡  ↻`)
  - Volume bar (`♪ ████▎□□□□□□□□□ 65%`)
- **Right (~2/3 width):** viz.Engine animated visualization with per-row color gradient
  - Top viz rows (top half of frame)
  - Gradient seek bar with time labels (1 row, `vizWidth` wide)
  - Bottom viz rows (bottom half of frame)

**Responsive:**
- `infoWidth = max(contentWidth/3, 28)` — min 28 ensures controls fit
- `vizWidth = contentWidth - infoWidth - 1` — gap; clamped min 1
- `vizHeight = bodyHeight - 1` — engine height excluding seek bar row
- height >= 8: full split layout with `lipgloss.Place` centering if content < available
- height < 8: title bar embeds track info (`Now Playing ── Track · Artist ── ▶ 1:41/5:30`)

**InfoBox border:** Standard rounded corners. Color follows `ActiveBorder()`/`InactiveBorder()` by focus state.

---

## 12. Notifications

See `tui.md §3.15` (Toast) for full rendering contract: intent types, default TTLs, glyph table, positioning per view-mode, content rules.

---

## 13. Search Overlay

Floating modal above grid. Two-panel: Search input (left ~30%) + Results (right ~70%).

See `tui.md §3.2` (OverlayChrome) for border rendering contract. Uses `OverlayChrome` with Accent border + notch-format action hints. **Border uses theme Active color (#411).** Textinput prompt tag renders themed prefix pill (`BuildPromptTag`).

### 7 Tabs

```
All → Songs → Artists → Albums → Playlists → Shows → Episodes → All
```

`TabLabels` (`search.go:41`): `TabAll`, `TabSongs`, `TabArtists`, `TabAlbums`, `TabPlaylists`, `TabShows`, `TabEpisodes`.

### Prefix Syntax (`search_prefix.go`)

States: `PrefixNone`, `PrefixTyping`, `Locked` (`search_prefix.go:13-23`).

| Prefix | Locks to |
|---|---|
| `:songs` | Songs tab |
| `:artists` | Artists tab |
| `:albums` | Albums tab |
| `:playlists` | Playlists tab |
| `:shows` | Shows tab |
| `:episodes` | Episodes tab |

Typing `:songs` + space → prefix locks, prompt tag → "Search Songs", subsequent typing filters within songs only. Backspace on empty query → prefix unlocks, tab returns to All.

### Tab Switch Behavior (#404)

Tab switch (Tab/Shift+Tab):
- Fires new `SearchRequestMsg` with new `TabToAPITypes` (`search.go:43-59`)
- Resets `page=1`
- Resets cursor
- Stale-response guard via `gen` counter (`search.go:503,599,631`) — only latest response displayed

### Pagination

PgDn/PgUp cycle pages. Prev arrow dimmed on page 1, next arrow dimmed on last page.

### Overlay Keys

| Key | Action |
|---|---|
| `Tab` | Cycle tab forward |
| `Shift+Tab` | Cycle tab backward |
| `Enter` | Play selected track (overlay stays open) |
| `Ctrl+A` / `A` | Add selected track to queue |
| `PgDn` | Next page |
| `PgUp` | Previous page |
| `Esc` | Close overlay, reset state |

Results panel border shows action notches: `ctrl+a, tab, pgdn, pgup`. No bottom keybar rendered.

**Impl:** `github.com/rmhubbert/bubbletea-overlay` for compositing; `internal/ui/panes/search.go`.

---

## 14. Device Switcher Overlay

`OverlayChrome` with `Devices` title. Positioned top-right via `btoverlay.Composite()`. Active device row uses `ListRow` with `◉` glyph (Success role); inactive rows use `○`.

See `tui.md §3.2` (OverlayChrome) + `§3.5` (ListRow) for rendering contracts.

**Impl:** `internal/ui/panes/devices.go:188,214`.

---

## 15. Global Header & Status Bar

See `tui.md §3.10` (HeaderBar) + `§3.11` (StatusBar) for rendering contracts: field roles, glyph choices, background token, ASCII fallback.

**Header (top line):** app name · page indicator · preset info · right-side device + profile chips. Uses `uikit.HeaderBar`.

**Status bar (bottom line):** global shortcuts only. Pane-specific shortcuts live in pane border notches. Uses `uikit.StatusBar` over `uikit.KeyBar`. Key labels in `KeyHint()`, descriptions in Muted role.

---

## 16. Focus & Navigation

### Focus Rotation

- `Tab` / `Shift+Tab`: rotate focus among **visible** panes only
- Order: top-left → top-right → second-row-left → ... → bottom-right → wrap
- Invisible panes skipped

### Pane Toggle (replaces Direct Pane Jump)

Toggle keys context-aware — adapt by page + preset. See §4 for auto-switch rules + per-preset mappings. Use `Tab`/`Shift+Tab` for focus navigation.

### Playback Keys (Always Route to NowPlaying)

| Key | Action |
|---|---|
| `Space` | Play/pause |
| `←` / `→` | Seek back/forward 5 s |
| `Shift+←` / `Shift+→` | Previous/next track |
| `+` / `-` | Volume up/down |
| `s` | Toggle shuffle |
| `r` | Cycle repeat |
| `v` | Cycle visualizer animation pattern |
| `i` | Show Episode Details overlay (when podcast episode playing) |

Route to `PaneNowPlaying` regardless of focus. NowPlaying content-aware — handles track + episode in single pane.

### Overlay Keys

| Key | Action |
|---|---|
| `/` | Open search overlay |
| `d` | Open device overlay |
| `u` | Open profile overlay |
| `t` | Open theme switcher overlay |
| `?` | Open help overlay |
| `Esc` | Close overlay / close filter |

Overlays intercept all keys while open. Focus saved + restored on close.

---

## 17. Keybinding Table (Complete)

| Key | Action | Scope |
|---|---|---|
| **Pages** | | |
| `0` | Cycle Player → Stats → Player | Global |
| **Pane Toggle (context-aware)** | | |
| `1`–`8` | Toggle pane visibility (mapping varies by preset) | Player page |
| `2`–`5` | Toggle pane visibility | Stats page |
| **Presets** | | |
| `p` | Cycle to next preset (6 Player, 1 Stats) | Per page |
| **Playback (always route to NowPlaying)** | | |
| `Space` | Play/pause | Always |
| `←` / `→` | Seek back/forward 5 s | Always |
| `Shift+←` / `Shift+→` | Previous/next track | Always |
| `+` / `-` | Volume up/down | Always |
| `s` | Toggle shuffle | Always |
| `r` | Cycle repeat | Always |
| `v` | Cycle visualizer animation pattern | Always |
| `i` | Show Episode Details overlay | Always (when episode playing) |
| **Navigation** | | |
| `Tab` | Next pane focus | Visible panes |
| `Shift+Tab` | Previous pane focus | Visible panes |
| `↑` / `k` | Scroll up | Focused pane |
| `↓` / `j` | Scroll down | Focused pane |
| `Enter` | Select/play · drill-down (FollowedShows) | Focused pane |
| `Esc` | Close overlay · clear filter · back from drill-down · scroll top | Context |
| **Pane Actions** | | |
| `f` | Toggle filter in focused pane | List panes |
| `g` | Cycle time range | TopTracks / TopArtists |
| `l` | Like / unlike selected track | LikedSongs / Queue / TopTracks / RecentlyPlayed / Playlists (track view) / Albums (track view) / Search |
| `A` | Add selected track to queue | Search overlay / list panes |
| **Playlist Management (Playlists pane)** | | |
| `Enter` | Open playlist tracks (sub-view) | Playlists pane |
| **Global Overlays** | | |
| `/` | Open search overlay | Global |
| `d` | Open device overlay | Global |
| `u` | Open user profile overlay | Global |
| `t` | Open theme switcher overlay | Global |
| `?` | Open help overlay | Global |
| `q` | Quit | Global |
| **Search Overlay (when open)** | | |
| `Tab` | Cycle result tab forward | Search overlay |
| `Shift+Tab` | Cycle result tab backward | Search overlay |
| `Enter` | Play selected track (overlay stays open) | Search overlay |
| `Ctrl+A` / `A` | Add selected track to queue | Search overlay |
| `PgDn` | Next results page | Search overlay |
| `PgUp` | Previous results page | Search overlay |
| **Profile Overlay** | | |
| `l` | Logout — ends session, keeps Client ID (press twice to confirm) | Profile overlay |
| `f` | Forget — removes session + Client ID (press twice to confirm) | Profile overlay |

**Three-location sync (AGENTS.md rule):** any keybinding change must update this table + `README.md` Keybindings + `internal/ui/panes/help_overlay.go` `helpContent` in same commit.

---

## 18. Theme Enhancements

### New Tokens

```go
// Gradient bars
Gradient1() lipgloss.Color     // Seek bar start / low volume
Gradient2() lipgloss.Color     // Seek bar end / mid volume
Gradient3() lipgloss.Color     // High volume (hot)

// Visualizer
VisualizerFg() lipgloss.Color  // Braille dot foreground
VizGradient1() lipgloss.Color  // Per-row viz gradient (Story 223)
VizGradient2() lipgloss.Color
VizGradient3() lipgloss.Color
VizGradient4() lipgloss.Color
VizGradient5() lipgloss.Color
VizGradient6() lipgloss.Color
VizGradient7() lipgloss.Color

// Tables
TableHeader() lipgloss.Color   // Column header text
ColumnIndex() lipgloss.Color   // # index column (Feature 70)
ColumnPrimary() lipgloss.Color // Title column
ColumnSecondary() lipgloss.Color // Artist column
ColumnTertiary() lipgloss.Color  // Duration column

// Status
PresetIndicator() lipgloss.Color  // Preset label in header
Accent() lipgloss.Color           // Generic accent (Feature 70)
HeaderChipFg() lipgloss.Color     // Header chip foreground

// Overlays
OverlayBackground() lipgloss.Color // Overlay dimmed background

// Per-pane borders
PaneBorderNowPlaying() lipgloss.Color
PaneBorderQueue() lipgloss.Color
PaneBorderPlaylists() lipgloss.Color
PaneBorderAlbums() lipgloss.Color
PaneBorderLikedSongs() lipgloss.Color
PaneBorderRecentlyPlayed() lipgloss.Color  // teal
PaneBorderTopTracks() lipgloss.Color       // purple
PaneBorderTopArtists() lipgloss.Color      // pink/red
PaneBorderFollowedShows() lipgloss.Color   // teal (podcast shows)
PaneBorderSavedEpisodes() lipgloss.Color   // green (saved episodes)
PaneBorderRequestFlow() lipgloss.Color     // orange/amber (flow visualization)
PaneBorderNetworkLog() lipgloss.Color      // warm grey (API log)

// Filter
// FilterInputBg dropped — use SurfaceAlt() instead (same value)
```

All 13 themes implement these tokens. Themes loaded dynamically from embedded TOML (`theme.go:128-135` `Available()`).

**13 themes:** `black`, `monokai`, `catppuccin`, `nord`, `light`, `dracula`, `gruvbox`, `rosepine`, `solarized`, `synthwave`, `tokyonight`, `mono-dark` (#288), `mono-light` (#288).

### Token Values — All 13 Themes

#### True Black (`black`) — Default

| Token | Hex | Usage |
|---|---|---|
| `Gradient1` | `#00ff88` | Green — seek start, low volume |
| `Gradient2` | `#ffcc00` | Yellow — seek end, mid volume |
| `Gradient3` | `#ff5555` | Red — high volume |
| `VisualizerFg` | `#00afff` | Ice blue — matches accent |
| `TableHeader` | `#666666` | Subtle header text |
| `PresetIndicator` | `#00afff` | Matches accent |
| `PaneBorderNowPlaying` | `#00ff88` | Green (playing) |
| `PaneBorderQueue` | `#ffcc00` | Yellow (warning) |
| `PaneBorderPlaylists` | `#00afff` | Blue (accent) |
| `PaneBorderAlbums` | `#00e5cc` | Cyan (teal) |
| `PaneBorderLikedSongs` | `#00ff88` | Green (success) |
| `PaneBorderRecentlyPlayed` | `#00ccaa` | Teal |
| `PaneBorderTopTracks` | `#bd93f9` | Purple |
| `PaneBorderTopArtists` | `#ff79c6` | Pink |
| `PaneBorderRequestFlow` | `#ffb86c` | Orange/amber |
| `PaneBorderNetworkLog` | `#8a8a8a` | Warm grey |
| ~~FilterInputBg~~ | — | Dropped: use `SurfaceAlt()` instead |

#### Monokai (`monokai`)

| Token | Hex | Notes |
|---|---|---|
| `Gradient1` | `#a6e22e` | Monokai green |
| `Gradient2` | `#e6db74` | Monokai yellow |
| `Gradient3` | `#f92672` | Monokai pink |
| `VisualizerFg` | `#66d9ef` | Monokai cyan |
| `TableHeader` | `#75715e` | Monokai comment grey |
| `PresetIndicator` | `#66d9ef` | Monokai cyan |
| `PaneBorderNowPlaying` | `#a6e22e` | Green |
| `PaneBorderQueue` | `#fd971f` | Orange |
| `PaneBorderPlaylists` | `#66d9ef` | Cyan |
| `PaneBorderAlbums` | `#e6db74` | Yellow |
| `PaneBorderLikedSongs` | `#a6e22e` | Green |
| `PaneBorderRecentlyPlayed` | `#4dc9b0` | Teal |
| `PaneBorderTopTracks` | `#ae81ff` | Purple |
| `PaneBorderTopArtists` | `#f92672` | Pink |
| `PaneBorderRequestFlow` | `#fd971f` | Orange |
| `PaneBorderNetworkLog` | `#75715e` | Monokai comment grey |

#### Catppuccin Mocha (`catppuccin`)

| Token | Hex | Notes |
|---|---|---|
| `Gradient1` | `#a6e3a1` | Green |
| `Gradient2` | `#f9e2af` | Yellow |
| `Gradient3` | `#f38ba8` | Red |
| `VisualizerFg` | `#89b4fa` | Blue |
| `TableHeader` | `#6c7086` | Overlay0 |
| `PresetIndicator` | `#89b4fa` | Blue |
| `PaneBorderNowPlaying` | `#a6e3a1` | Green |
| `PaneBorderQueue` | `#f9e2af` | Yellow |
| `PaneBorderPlaylists` | `#89b4fa` | Blue |
| `PaneBorderAlbums` | `#94e2d5` | Teal |
| `PaneBorderLikedSongs` | `#a6e3a1` | Green |
| `PaneBorderRecentlyPlayed` | `#94e2d5` | Teal |
| `PaneBorderTopTracks` | `#cba6f7` | Mauve |
| `PaneBorderTopArtists` | `#f38ba8` | Red/pink |
| `PaneBorderRequestFlow` | `#fab387` | Peach/orange |
| `PaneBorderNetworkLog` | `#6c7086` | Overlay0 grey |

#### Nord (`nord`)

| Token | Hex | Notes |
|---|---|---|
| `Gradient1` | `#a3be8c` | Nord green |
| `Gradient2` | `#ebcb8b` | Nord yellow |
| `Gradient3` | `#bf616a` | Nord red |
| `VisualizerFg` | `#88c0d0` | Nord frost |
| `TableHeader` | `#4c566a` | Nord grey |
| `PresetIndicator` | `#88c0d0` | Nord frost |
| `PaneBorderNowPlaying` | `#a3be8c` | Green |
| `PaneBorderQueue` | `#ebcb8b` | Yellow |
| `PaneBorderPlaylists` | `#88c0d0` | Frost |
| `PaneBorderAlbums` | `#8fbcbb` | Teal |
| `PaneBorderLikedSongs` | `#a3be8c` | Green |
| `PaneBorderRecentlyPlayed` | `#8fbcbb` | Teal |
| `PaneBorderTopTracks` | `#b48ead` | Purple |
| `PaneBorderTopArtists` | `#bf616a` | Red |
| `PaneBorderRequestFlow` | `#d08770` | Orange |
| `PaneBorderNetworkLog` | `#4c566a` | Nord grey |

#### Light — Catppuccin Latte (`light`)

| Token | Hex | Notes |
|---|---|---|
| `Gradient1` | `#40a02b` | Latte green |
| `Gradient2` | `#df8e1d` | Latte yellow |
| `Gradient3` | `#d20f39` | Latte red |
| `VisualizerFg` | `#1e66f5` | Latte blue |
| `TableHeader` | `#9ca0b0` | Latte overlay0 |
| `PresetIndicator` | `#1e66f5` | Latte blue |
| `PaneBorderNowPlaying` | `#40a02b` | Green |
| `PaneBorderQueue` | `#df8e1d` | Yellow |
| `PaneBorderPlaylists` | `#1e66f5` | Blue |
| `PaneBorderAlbums` | `#179299` | Teal |
| `PaneBorderLikedSongs` | `#40a02b` | Green |
| `PaneBorderRecentlyPlayed` | `#179299` | Teal |
| `PaneBorderTopTracks` | `#8839ef` | Mauve |
| `PaneBorderTopArtists` | `#d20f39` | Red |
| `PaneBorderRequestFlow` | `#fe640b` | Orange |
| `PaneBorderNetworkLog` | `#9ca0b0` | Latte overlay0 grey |

#### Dracula (`dracula`)

| Token | Hex | Notes |
|---|---|---|
| `Gradient1` | `#50FA7B` | Green |
| `Gradient2` | `#F1FA8C` | Yellow |
| `Gradient3` | `#FF5555` | Red |
| `VisualizerFg` | `#BD93F9` | Purple |
| `TableHeader` | `#6272A4` | Comment grey |
| `PresetIndicator` | `#BD93F9` | Purple |
| `PaneBorderNowPlaying` | `#50FA7B` | Green |
| `PaneBorderQueue` | `#F1FA8C` | Yellow |
| `PaneBorderPlaylists` | `#BD93F9` | Purple |
| `PaneBorderAlbums` | `#8BE9FD` | Cyan |
| `PaneBorderLikedSongs` | `#50FA7B` | Green |
| `PaneBorderRecentlyPlayed` | `#8BE9FD` | Cyan |
| `PaneBorderTopTracks` | `#FF79C6` | Pink |
| `PaneBorderTopArtists` | `#FF5555` | Red |
| `PaneBorderRequestFlow` | `#FFB86C` | Orange |
| `PaneBorderNetworkLog` | `#69ff47` | Bright green |

#### Gruvbox Dark (`gruvbox`)

| Token | Hex | Notes |
|---|---|---|
| `Gradient1` | `#b8bb26` | Gruvbox green |
| `Gradient2` | `#fabd2f` | Gruvbox yellow |
| `Gradient3` | `#fb4934` | Gruvbox red |
| `VisualizerFg` | `#fe8019` | Gruvbox orange |
| `TableHeader` | `#665c54` | Gruvbox grey |
| `PresetIndicator` | `#fe8019` | Orange |
| `PaneBorderNowPlaying` | `#b8bb26` | Green |
| `PaneBorderQueue` | `#fabd2f` | Yellow |
| `PaneBorderPlaylists` | `#83a598` | Teal/aqua |
| `PaneBorderAlbums` | `#8ec07c` | Bright green |
| `PaneBorderLikedSongs` | `#b8bb26` | Green |
| `PaneBorderRecentlyPlayed` | `#8ec07c` | Bright green |
| `PaneBorderTopTracks` | `#d3869b` | Purple |
| `PaneBorderTopArtists` | `#fb4934` | Red |
| `PaneBorderRequestFlow` | `#fe8019` | Orange |
| `PaneBorderNetworkLog` | `#458588` | Blue/teal |

#### Rose Pine (`rosepine`)

| Token | Hex | Notes |
|---|---|---|
| `Gradient1` | `#9ccfd8` | Foam (teal) |
| `Gradient2` | `#f6c177` | Gold |
| `Gradient3` | `#eb6f92` | Love (red/pink) |
| `VisualizerFg` | `#c4a7e7` | Iris (purple) |
| `TableHeader` | `#6e6a86` | Muted |
| `PresetIndicator` | `#c4a7e7` | Iris (purple) |
| `PaneBorderNowPlaying` | `#9ccfd8` | Foam (teal) |
| `PaneBorderQueue` | `#f6c177` | Gold |
| `PaneBorderPlaylists` | `#c4a7e7` | Iris (purple) |
| `PaneBorderAlbums` | `#31748f` | Pine (blue) |
| `PaneBorderLikedSongs` | `#ebbcba` | Rose |
| `PaneBorderRecentlyPlayed` | `#9ccfd8` | Foam (teal) |
| `PaneBorderTopTracks` | `#c4a7e7` | Iris (purple) |
| `PaneBorderTopArtists` | `#eb6f92` | Love (red/pink) |
| `PaneBorderRequestFlow` | `#f6c177` | Gold |
| `PaneBorderNetworkLog` | `#ff6e91` | Warm pink |

#### Solarized Dark (`solarized`)

| Token | Hex | Notes |
|---|---|---|
| `Gradient1` | `#859900` | Solarized green |
| `Gradient2` | `#b58900` | Solarized yellow |
| `Gradient3` | `#dc322f` | Solarized red |
| `VisualizerFg` | `#268bd2` | Solarized blue |
| `TableHeader` | `#586e75` | Base01 |
| `PresetIndicator` | `#268bd2` | Blue |
| `PaneBorderNowPlaying` | `#859900` | Green |
| `PaneBorderQueue` | `#b58900` | Yellow |
| `PaneBorderPlaylists` | `#268bd2` | Blue |
| `PaneBorderAlbums` | `#2aa198` | Cyan |
| `PaneBorderLikedSongs` | `#859900` | Green |
| `PaneBorderRecentlyPlayed` | `#2aa198` | Cyan |
| `PaneBorderTopTracks` | `#6c71c4` | Violet |
| `PaneBorderTopArtists` | `#d33682` | Magenta |
| `PaneBorderRequestFlow` | `#cb4b16` | Orange |
| `PaneBorderNetworkLog` | `#dc322f` | Red |

#### Synthwave '84 (`synthwave`)

| Token | Hex | Notes |
|---|---|---|
| `Gradient1` | `#72f1b8` | Mint green |
| `Gradient2` | `#fede5d` | Yellow |
| `Gradient3` | `#fe4450` | Neon red |
| `VisualizerFg` | `#ff7edb` | Pink |
| `TableHeader` | `#848bbd` | Muted blue |
| `PresetIndicator` | `#36f9f6` | Cyan |
| `PaneBorderNowPlaying` | `#72f1b8` | Mint green |
| `PaneBorderQueue` | `#fede5d` | Yellow |
| `PaneBorderPlaylists` | `#ff7edb` | Pink |
| `PaneBorderAlbums` | `#36f9f6` | Cyan |
| `PaneBorderLikedSongs` | `#72f1b8` | Mint green |
| `PaneBorderRecentlyPlayed` | `#36f9f6` | Cyan |
| `PaneBorderTopTracks` | `#ff7edb` | Pink |
| `PaneBorderTopArtists` | `#fe4450` | Neon red |
| `PaneBorderRequestFlow` | `#fede5d` | Yellow |
| `PaneBorderNetworkLog` | `#ff8b39` | Orange |

#### Tokyo Night (`tokyonight`)

| Token | Hex | Notes |
|---|---|---|
| `Gradient1` | `#9ece6a` | Green |
| `Gradient2` | `#e0af68` | Yellow/gold |
| `Gradient3` | `#f7768e` | Red |
| `VisualizerFg` | `#7aa2f7` | Blue |
| `TableHeader` | `#565f89` | Muted blue |
| `PresetIndicator` | `#7aa2f7` | Blue |
| `PaneBorderNowPlaying` | `#9ece6a` | Green |
| `PaneBorderQueue` | `#e0af68` | Yellow/gold |
| `PaneBorderPlaylists` | `#7aa2f7` | Blue |
| `PaneBorderAlbums` | `#73daca` | Teal |
| `PaneBorderLikedSongs` | `#9ece6a` | Green |
| `PaneBorderRecentlyPlayed` | `#73daca` | Teal |
| `PaneBorderTopTracks` | `#bb9af7` | Purple |
| `PaneBorderTopArtists` | `#f7768e` | Red |
| `PaneBorderRequestFlow` | `#ff9e64` | Orange |
| `PaneBorderNetworkLog` | `#7dcfff` | Light blue |

#### Mono Dark (`mono-dark`) — #288

Grayscale-only variant. No color tokens leak. All borders + bars use gray ramp.

| Token | Hex | Notes |
|---|---|---|
| `Gradient1` | `#cccccc` | Light gray |
| `Gradient2` | `#999999` | Mid gray |
| `Gradient3` | `#666666` | Dark gray |
| `VisualizerFg` | `#aaaaaa` | Gray |
| `TableHeader` | `#555555` | Dim gray |
| `PresetIndicator` | `#cccccc` | Light gray |
| `PaneBorderNowPlaying` | `#cccccc` | Light gray |
| `PaneBorderQueue` | `#999999` | Mid gray |
| `PaneBorderPlaylists` | `#aaaaaa` | Gray |
| `PaneBorderAlbums` | `#888888` | Gray |
| `PaneBorderLikedSongs` | `#cccccc` | Light gray |
| `PaneBorderRecentlyPlayed` | `#888888` | Gray |
| `PaneBorderTopTracks` | `#bbbbbb` | Gray |
| `PaneBorderTopArtists` | `#777777` | Gray |
| `PaneBorderRequestFlow` | `#999999` | Mid gray |
| `PaneBorderNetworkLog` | `#555555` | Dim gray |

#### Mono Light (`mono-light`) — #288

Grayscale variant with inverted background (light bg, dark fg). Same gray ramp for borders + bars.

| Token | Hex | Notes |
|---|---|---|
| `Gradient1` | `#333333` | Dark gray |
| `Gradient2` | `#666666` | Mid gray |
| `Gradient3` | `#999999` | Light gray |
| `VisualizerFg` | `#555555` | Gray |
| `TableHeader` | `#aaaaaa` | Light gray |
| `PresetIndicator` | `#333333` | Dark gray |
| `PaneBorderNowPlaying` | `#333333` | Dark gray |
| `PaneBorderQueue` | `#666666` | Mid gray |
| `PaneBorderPlaylists` | `#555555` | Gray |
| `PaneBorderAlbums` | `#777777` | Gray |
| `PaneBorderLikedSongs` | `#333333` | Dark gray |
| `PaneBorderRecentlyPlayed` | `#777777` | Gray |
| `PaneBorderTopTracks` | `#444444` | Gray |
| `PaneBorderTopArtists` | `#888888` | Gray |
| `PaneBorderRequestFlow` | `#666666` | Mid gray |
| `PaneBorderNetworkLog` | `#aaaaaa` | Light gray |

---

## 19. Stats page — Stats Specification

Live visibility into Spotnik's internal request pipeline. No Spotify API calls — all data from internal structures (`*Gateway`, `*Store`).

5 panes: NowPlaying compact strip at top + 3 diagnostic panes side-by-side (GatewayHealth, PollingTraffic, GatewayLive) + full-width NetworkLog row.

### Toggle Key Table (Stats page)

| Key | Pane |
|---|---|
| `2` | Gateway Health |
| `3` | Polling Traffic |
| `4` | Gateway Live |
| `5` | Network Log |

### Pane 1: Gateway Health (toggle key 2)

4-row fixed grid showing real-time gateway state:

```
╭─ ²Gateway Health ─────────────────────────╮
│  ●  Tokens    ●●●●●●●●●●  10/10           │
│  ■  Slots     ■□□□□  1/5                  │
│  ⏱  Backoff   none                        │
│  ≋  Dedup     none                        │
╰───────────────────────────────────────────╯
```

| Row | Data | Warning trigger |
|---|---|---|
| Tokens | Token bucket fill level (dot bar) | `Warning()` when ≤ 2 remaining |
| Slots | Concurrent semaphore (square bar) | `Warning()` when all slots full |
| Backoff | Countdown seconds (`Error()` colour) | Always `Error()` when > 0 |
| Dedup | Number of GET waiters | `TextSecondary()` when > 0 |

- **Data source:** `store.ReadEventsFrom(cursor)` — `GatewayStateSnapshot` from each event
- **Update:** Every 1s app tick

### Pane 2: Polling Traffic (toggle key 3)

5-row fixed grid showing playback poll cadence + library cache freshness:

```
╭─ ³Polling Traffic ────────────────────────╮
│  ♫  Playback    ▶ 1s · running            │
│  ☰  Playlists   ◦ fresh                   │
│  ♫♫ Albums      ⚠ 3m stale               │
│  📌 Liked       ◦ fresh                   │
│  ⏱  Recent      ◦ fresh                   │
╰───────────────────────────────────────────╯
```

- **Playback row:** `PollingSnapshotMsg` (tick interval + idle state)
- **Library rows:** `store.PlaylistsFetchedAt()`, `store.AlbumsFetchedAt()`, etc. + TTL constants
- **Stale colours:** `Warning()` for < 1h stale, `Error()` for ≥ 1h stale

### Pane 3: Gateway Live (toggle key 4)

500-entry reverse-chronological gateway event stream, scrollable + filterable:

```
╭─ ⁴Gateway Live ──────────────────────────── f filter ╭
│  12:03:45  → /me/player            allowed  200  45ms │
│  12:03:44  → /me/playlists         allowed  200 128ms │
│  12:03:43  ✗ /me/player            blocked            │
│  (scrollable with j/k; f opens filter; Enter commits) │
╰───────────────────────────────────────────────────────╯
```

- **Buffer:** 500 entries, newest at top
- **Filter:** `f` opens inline filter; `Enter` commits (shown in border); `Esc` clears committed query first, then resets scroll
- **Data source:** `store.ReadEventsFrom(cursor)` — every `domain.GatewayEvent`

### Pane 4: Network Log (toggle key 5)

Scrollable reverse-chronological log of completed API requests (200-entry buffer):

```
╭─ ⁵Network Log ──────────────────────────────────── f filter ╭
│  Time      Method  Endpoint           Status  Latency  Priority      Decision │
│  12:03:45  GET     /me/player         200     45ms     ◷ background  allowed  │
│  12:03:44  GET     /me/playlists      200     128ms    ◷ background  allowed  │
│  12:03:43  GET     /me/player         429     12ms     ⚡ interactive allowed  │
│  12:03:42  GET     /me/player/queue   0       —        ◷ background  blocked  │
╰──────────────────────────────────────────────────────────────────────────────╯
```

- **Scrollable:** `j`/`k` when focused; `Esc` resets scroll
- **Filterable:** `f` opens inline filter (by endpoint, status, priority, decision)
- **Color coding:** `Success()` 2xx, `Warning()` 429, `TextMuted()` other 4xx, `Error()` 5xx
- **Decision cross-tick:** `pendingDecisions` map persists decision events across ticks so Decision column populated when `EventHttpCompleted` arrives on later tick
- **Data source:** `store.ReadEventsFrom(cursor)` — `EventHttpCompleted` + `EventRequestBlocked`

### Tick Architecture

| Tick | Rate | Purpose |
|---|---|---|
| App tick | 1000ms | All 4 Stats panes refresh via `TickMsg`; `PollingSnapshotMsg` sent to PollingTrafficPane |
| Animation tick | 200ms | NowPlaying visualizer only — Stats panes do not consume `viz.TickMsg` |

### Data Sources (all internal — no API calls)

| Data | Source | Update |
|---|---|---|
| Token bucket state | `store.ReadEventsFrom` → `GatewayStateSnapshot` | Every app tick |
| Concurrent requests | `store.ReadEventsFrom` → `GatewayStateSnapshot` | Every app tick |
| Backoff / dedup | `store.ReadEventsFrom` → `GatewayStateSnapshot` | Every app tick |
| Polling state | `PollingSnapshotMsg` (tick interval + idle flag) | Every app tick |
| Library cache freshness | `store.*FetchedAt()` + TTL constants | Every app tick |
| Request log | `store.ReadEventsFrom(cursor)` — `EventHttpCompleted`, `EventRequestBlocked` | On each API response |
| Polling state | `tickCount`, `backoffTicks`, `isIdle()`, `pollIntervals()` | Every app tick |
| Store fetching | `Store.*Fetching()` sentinels | Every app tick |
| Store staleness | `Store.*FetchedAt()` + TTL constants | Every app tick |
| Request priority | `api.WithPriority(ctx, ...)` — `Interactive` vs `Background` | Per request |

### Traffic Shaping (#409)

Stats preset Row 2 weights `1:1:3` (GatewayHealth:PollingTraffic:GatewayLive). GatewayLive dominant — gives most screen real estate to live request replay. Reflects gateway traffic shaping priority: live observability > static health > polling diagnostics.

---

## 20. Mouse Scroll Support

Mouse scroll = scroll any pane without changing focus (btop behavior).

### Implementation

- `tea.EnableMouseCellMotion()` at app startup
- `tea.MouseMsg` with `MouseWheelUp`/`MouseWheelDown` scrolls pane under cursor
- Hit-test: check which pane `Rect` contains mouse position
- Pane doesn't need focus to scroll with mouse (btop style)
- Click pane to focus (optional, future)

### Architecture

`LayoutManager.PaneAt(x, y int) PaneID` (`layout.go`) hit-tests against computed `Rect` values. Mouse scroll routed to returned pane, bypassing focus system.

---

## 21. Responsive Behavior

### Minimum Terminal Size

| Preset | Min Width | Min Height |
|---|---|---|
| All | 120 columns | 30 rows |

Below minimum:
```
╭──────────────────────────────────────────╮
│  Spotnik needs more space                │
│                                          │
│  Current:  98 × 25                       │
│  Required: 120 × 30                      │
│                                          │
│  Please resize your terminal and retry.  │
╰──────────────────────────────────────────╯
```

### Future: Auto-Degrade

Not in initial impl. Future: auto-hide lower-priority panes (TopArtists/TopTracks first, then RecentlyPlayed, then library row) when terminal smaller than optimal but above minimum.

---

## 22. Architecture — LayoutManager

### Package: `internal/ui/layout/`

| File | Purpose |
|---|---|
| `layout.go` | `Manager` struct, `Resize()`, `recompute()`, `PaneRect()`, `PaneAt()`, `SetPreset()`, `CyclePreset()`, `TogglePage()`, `TogglePane()`, `RotateFocus()`, `FocusedPane()` |
| `pane.go` | `Pane` interface, `PaneID` enum (14 IDs), `PageID` enum (`PagePlayer`, `PageStats`), `Action` struct, `FilterablePane`, `FilterQueryPane` interfaces |
| `presets.go` | `PresetDashboard`, `PresetListening`, `PresetPodcast`, `PresetLibrary`, `PresetDiscovery`, `PresetPodcastDashboard`, `PresetStats` definitions |
| `border.go` | `RenderPaneBorder()` — custom border with btop-style title + actions |
| `truncate.go` | `Truncate()`, `PadRight()`, `TruncateOrPad()` — rune-aware text utilities |
| `*_test.go` | Full table-driven test coverage |

### Manager Struct

```go
type Manager struct {
    activePage   PageID           // PagePlayer or PageStats
    presets      map[PageID][]Preset
    activePreset map[PageID]int
    hidden       map[PaneID]bool
    rects        map[PaneID]Rect
    focusOrder   []PaneID         // visible panes in layout order
    focusIndex   int
    width        int
    height       int
    headerHeight int              // 1
    statusHeight int              // 1
}
```

### Integration with App

```go
// App struct (internal/app/app.go:160-181) — overlay fields
type App struct {
    layout *layout.Manager
    panes  map[layout.PaneID]layout.Pane

    // Floating overlays (7)
    searchPane                  *panes.SearchOverlay
    devicePane                  *panes.DeviceOverlay
    profileOverlay              *panes.ProfileOverlay
    profileOverlayOpen          bool
    showThemeSwitcher           bool
    themeOverlay                *panes.ThemeOverlay
    helpOpen                    bool
    helpOverlay                 *panes.HelpOverlay
    episodeDetailsOpen          bool
    episodeDetails              *panes.EpisodeDetailsOverlay
    onboardingPermissionsOverlay *panes.OnboardingPermissionsOverlay

    // Toast + error infra
    alerts    *uikit.ToastManager
    errMapper *uikit.ErrorMapper

    // Gateway
    gateway *api.Gateway
}
```

---

## 23. Episode Details Overlay (#334)

Floating overlay showing podcast episode details.

**Trigger:** `i` key when podcast episode playing. Silent no-op when track playing.

**Content:**
- Episode description
- Show name
- Release date
- Duration
- Resume point (if partially played)

**Chrome:** `uikit.OverlayChrome` with "Episode Details" title. Centered via `btoverlay.Composite()`.

**Close:** `Esc`.

**Impl:** `internal/ui/panes/episode_details_overlay.go:25`. Golden tests: `TestEpisodeDetailsOverlay_View_EpisodeInfo`, `TestEpisodeDetailsOverlay_View_Narrow`.

---

## 24. Onboarding Permissions Overlay (#268)

InfoBox overlay opened during onboarding Step 2 (OAuth).

**Trigger:** `v` key on Step 2 OAuth screen.

**Content:** `uikit.InfoBox` with title "Permissions Spotnik requests" + body listing Spotify OAuth scopes Spotnik uses.

**Chrome:** `uikit.OverlayChrome`. Narrow-width guard (`info_box_test.go:60`). Rendered in onboarding view branch (`render.go:254`).

**Close:** `Esc`.

**Impl:** `internal/ui/panes/onboarding_permissions_overlay.go:48`.

---

## 25. Theme Switcher / Profile / Help Overlays

### Theme Switcher

**Trigger:** `t` global key.

**Content:** All 13 themes listed. Currently active marked ✓. Select + Enter applies immediately, overlay closes.

**Chrome:** `uikit.OverlayChrome` with "Themes" title. Centered.

**Impl:** `internal/ui/panes/themes.go:28`. Golden: `TestThemeOverlay_View_ThemeList`, `TestThemeOverlay_View_Narrow`.

### Profile

**Trigger:** `u` global key.

**Content:** Display name, subscription tier (Premium/Free), country. Sub-actions: `l` logout (double-key confirm), `f` forget (double-key confirm).

**Chrome:** `uikit.OverlayChrome` with "Profile" title. Centered.

**Impl:** `internal/ui/panes/profile.go`. Golden: `TestProfileOverlay_View_*`.

### Help

**Trigger:** `?` global key.

**Content:** All keybindings grouped by category. Rendered from `helpContent` in `internal/ui/panes/help_overlay.go:42-81`.

**Chrome:** `uikit.OverlayChrome` with "Help" title. Centered.

**Impl:** `internal/ui/panes/help_overlay.go:85`. Golden: `TestHelpOverlay_View_Keybindings`, `TestHelpOverlay_View_Narrow`.

---

## 26. Box Drawing Reference

Rounded corners exclusively:

```
╭─────────────╮   Used for all pane borders + overlays
│             │
╰─────────────╯
```

`─` horizontal fills, `│` vertical borders. Never `┌┐└┘`.

---

## 27. Color System Rules

- All color values from `internal/ui/theme/`
- Never hardcode hex in component code
- Always reference tokens through `Theme` interface
- New components use new tokens (§18)
- Themes loaded from embedded TOML in `internal/ui/theme/themes/*.toml` (13 files)

---

## 28. Accessibility

- All state changes visible via color AND text/symbol — never color alone
- Per-pane border colors supplemented by pane titles (text identification)
- Filter state shown in border text, not just color
- Scroll indicators use text (`▲`/`▼`), not just position
- `?` help always available
- Mono themes (#288) provide colorless fallback for color-blind users

---

*Last updated: 2026-07-18*
