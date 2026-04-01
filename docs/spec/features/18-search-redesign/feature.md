---
title: "Search Overlay Redesign"
status: open
---

## Description

Redesign the search overlay from a narrow stacked-section list into a wide tabbed interface
with rich metadata columns. The overlay becomes the dominant screen element when open,
showing one result section at a time with full table-style presentation and per-section
counts in the tab bar.

Key changes from the current design:
- **Wider overlay** — `min(90, 80% terminal)` instead of `min(50, 60% terminal)`
- **Tabbed sections** — horizontal tab bar replaces stacked section headers; only one
  section visible at a time; Tab/Shift+Tab cycles tabs
- **Tab counts** — each tab shows the total result count (e.g. "Tracks 10")
- **Per-tab colors** — each tab has a distinct color via existing `PaneBorder*` theme
  tokens (Tracks=purple, Artists=pink, Albums=cyan, Playlists=blue in default theme).
  Zero TOML changes — all 11 themes get distinct tab colors automatically.
- **Column colors** — table data uses `ColumnIndex/Primary/Secondary/Tertiary` theme
  tokens, matching how all table panes (Queue, LikedSongs, TopTracks, etc.) render.
  Column headers use the active tab's color for visual identity reinforcement.
- **Richer metadata** — tracks show Album + Duration; albums show Year + Track Count;
  playlists show Track Count; carried through from the API response
- **More results** — 10 results per section (up from 5), matching the Feb 2026 API max of 10 per type
- **Column headers** — each section has its own column header row for clarity
- **Contextual help bar** — bottom row shows available keybindings for the active section
- **Graceful degradation** — Tracks tab drops Album column on narrow terminals (`< 60` chars)

## Visual Design

Color key: Each tab uses its `PaneBorder*` token. Column data uses `Column*` tokens.
Column headers inherit the active tab's color. Selected row uses `SelectedBg/Fg`.

```
╭─ Search ───────────────────────────────────────────────────────── Enter play  Esc close ──╮
│ > alpha                                                                                    │
│ ··························································································│
│  ▪ Tracks 10     Artists 5     Albums 8     Playlists 12                                   │
│    [TopTracks]   [TextMuted]   [TextMuted]  [TextMuted]                                    │
│ ─────────────────────────────────────────────────────────────────────────────────────────── │
│                                                                                            │
│  #  Track                          Artist                Album              Duration       │
│  [TabColor headers — PaneBorderTopTracks in this case]                                     │
│  ─  ─────                          ──────                ─────              ────────       │
│  ▶  Alpha                          Kingside              Alpha EP           3:42           │
│  [SelectedBg/Fg — overrides column colors for selected row]                                │
│  2  Forever Young                  Alphaville            Forever Young      3:44           │
│  [ColIndex] [ColPrimary]           [ColSecondary]        [ColSecondary]     [ColTertiary]  │
│  3  Alpha's Goodbye                King                  Alpha's Goodbye    4:12           │
│  4  3 Hour Study Focus Music       SPIRIT ENERGY VIBE    Study Collection   2:01:30        │
│  5  Mozart Effect                  Study Music Project   Classical Focus    1:45:00        │
│  6  Alpha Male                     T.I.                  No Mercy           4:33           │
│  7  Alpha                          Charlotte Cardin      Phoenix            3:18           │
│  8  Alpha Omega                    Machine Head          Bloodstone & D...  7:52           │
│  9  Alpha Dog                      Fall Out Boy          Infinity on High   3:24           │
│ 10  Alphaville                     Dreamland             Alphaville         4:01           │
│                                                                                            │
│                                                                                            │
│ ─────────────────────────────────────────────────────────────────────────────────────────── │
│  Tab next section  ↑↓ navigate  Enter play  Ctrl+A queue  Esc close                       │
╰──────────────────────────────────────────────────────────────────────────────────────────────╯
```

Artists tab:
```
│    Tracks 10    ▪ Artists 5     Albums 8     Playlists 12                                  │
│ ─────────────────────────────────────────────────────────────────────────────────────────── │
│                                                                                            │
│  #  Artist                                                                                 │
│  ─  ──────                                                                                 │
│  ▶  Alphaville                                                                             │
│  2  Alpha Blondy                                                                           │
│  3  Alpha Portal                                                                           │
│  4  Alpha Wann                                                                             │
│  5  LNGSHOT                                                                                │
```

Albums tab:
```
│    Tracks 10     Artists 5    ▪ Albums 8     Playlists 12                                  │
│ ─────────────────────────────────────────────────────────────────────────────────────────── │
│                                                                                            │
│  #  Album                          Artist                Year    Tracks                    │
│  ─  ─────                          ──────                ────    ──────                    │
│  ▶  Happy Patel – Khatarnak        Vir Das               2023    12                       │
│  2  Alpha's Goodbye                King                  2024    8                         │
│  3  They Call Him OG               Thaman S              2024    15                        │
```

Playlists tab:
```
│    Tracks 10     Artists 5     Albums 8    ▪ Playlists 12                                 │
│ ─────────────────────────────────────────────────────────────────────────────────────────── │
│                                                                                            │
│  #  Playlist                       Owner                 Tracks                            │
│  ─  ────────                       ─────                 ──────                            │
│  ▶  Alpha Male Songs               Am I real?            45                               │
│  2  Alpha songs                    SIDAN                 23                                │
│  3  ALPHA MOVIE ALL SONGS          Me Time               120                              │
```

## Acceptance Criteria

- [ ] Overlay width is `min(90, 80% terminal)` — nearly double the current 50-char max
- [ ] Overlay height is `max(26, 75% terminal)` — taller to fit more results
- [ ] Tab bar renders all 4 sections with result counts (e.g. "Tracks 10")
- [ ] Active tab is visually distinct (▪ marker + bold + tab-specific color)
- [ ] Each tab has a distinct color via `PaneBorder*` theme tokens — zero TOML changes
- [ ] Inactive tabs use `TextMuted()` color
- [ ] Only the active section's results are shown (not all 4 stacked)
- [ ] Tab/Shift+Tab cycles through tabs (same keys as current section cycling)
- [ ] Tracks tab shows 5 columns: #, Track, Artist, Album, Duration
- [ ] Albums tab shows 5 columns: #, Album, Artist, Year, Tracks
- [ ] Playlists tab shows 4 columns: #, Playlist, Owner, Tracks
- [ ] Artists tab shows 2 columns: #, Artist
- [ ] Column headers styled with active tab's color (from `tabColorForSection`)
- [ ] Column data uses `ColumnIndex/Primary/Secondary/Tertiary` theme tokens
- [ ] Selected row uses `SelectedBg()/SelectedFg()` (overrides column colors)
- [ ] Tracks tab drops Album column when terminal is narrow (`contentWidth < 60`)
- [ ] Up to 10 results per section (API max since Feb 2026, increased from 5)
- [ ] Column headers render below the tab bar with underline separators
- [ ] Bottom help bar shows contextual keybindings
- [ ] Enriched data types carry Album name + Duration for tracks, Year + TrackCount for albums, TrackCount for playlists
- [ ] Existing keybindings preserved: Enter play, Ctrl+A queue, Esc close, ↑↓ navigate
- [ ] Theme switching (`t`) updates all tab and column colors correctly
- [ ] `make ci` passes
