---
title: "Search Results: Rich Metadata Display"
feature: 19-search-redesign
status: open
---

## Background

The current search delegate (`SearchItemDelegate` in `search_delegate.go`) renders each result as two lines: a category badge + name, and a sparse subtitle (first artist name for tracks/albums, owner for playlists, empty for artists). The Spotify search API returns significantly more data per item that we're not surfacing:

- **Tracks**: duration, album name, explicit flag, all artists (not just first)
- **Artists**: genres, follower count, popularity — currently the subtitle is *empty*
- **Albums**: release date, total tracks, album type (single/album/compilation), all artists
- **Playlists**: track count, description — track count exists in domain but isn't shown

The user wants every result item to look "nerdy and rich" — information-dense, colorful, with per-field theme styling. Think btop/htop aesthetic: dense data, distinct colors per data type, everything visible at a glance.

This story has two layers:
1. **Domain + API**: Expand `SearchArtist` to capture genres, followers, popularity from the API response (these fields exist in the JSON but are currently dropped during unmarshaling)
2. **Delegate rendering**: Redesign `SearchItemDelegate.Render()` to show all available metadata with theme-colored styling per field

## Design

### Layer 1: Domain Type Expansion

**File: `internal/domain/search.go`**

Expand `SearchArtist` to capture the three additional fields the search API returns:

```go
type SearchArtist struct {
    ID         string   `json:"id"`
    Name       string   `json:"name"`
    URI        string   `json:"uri"`
    Genres     []string `json:"genres"`     // e.g. ["art rock", "alternative rock"]
    Followers  int      `json:"-"`          // custom unmarshal from nested followers.total
    Popularity int      `json:"popularity"` // 0-100
}
```

`Followers` requires custom `UnmarshalJSON` because the API nests it as `followers.total` (same pattern used by `SearchPlaylist.TrackCount`):

```go
func (a *SearchArtist) UnmarshalJSON(data []byte) error {
    raw := &struct {
        ID         string   `json:"id"`
        Name       string   `json:"name"`
        URI        string   `json:"uri"`
        Genres     []string `json:"genres"`
        Popularity int      `json:"popularity"`
        Followers  struct {
            Total int `json:"total"`
        } `json:"followers"`
    }{}
    if err := json.Unmarshal(data, raw); err != nil {
        return err
    }
    a.ID = raw.ID
    a.Name = raw.Name
    a.URI = raw.URI
    a.Genres = raw.Genres
    a.Popularity = raw.Popularity
    a.Followers = raw.Followers.Total
    return nil
}
```

**Expand `SearchPlaylist`** to capture `description`:

Add `Description string \`json:"-"\`` to `SearchPlaylist` and extract it in the existing `UnmarshalJSON`.

**Track and Album domain types already have** `DurationMs`, `Album.Name`, `Artists []Artist`, `ReleaseDate`, `TotalTracks` — no domain changes needed for those.

The **Explicit** flag on tracks is not yet in `domain.Track`. Add it:

```go
type Track struct {
    // ... existing fields ...
    Explicit bool `json:"explicit"` // true if track has explicit content
}
```

### Layer 2: SearchListItem Expansion

**File: `internal/ui/panes/search_delegate.go`**

Expand `SearchListItem` with additional metadata fields. The existing `Subtitle` field is **kept** — it satisfies the `list.Item.Description()` interface and is used by the list's built-in filtering. The conversion helpers populate `Subtitle` with the rich description string during construction.

```go
type SearchListItem struct {
    Category string // "track", "artist", "album", "playlist"
    Name     string // Primary display name
    Subtitle string // Rich description line (kept — satisfies list.Item.Description())
    URI      string // Spotify URI
    IsTrack  bool   // true for tracks

    // Track-specific metadata
    ArtistNames string // All artists joined: "Artist1, Artist2"
    AlbumName   string // Album name
    Duration    string // Formatted "3:42"
    Explicit    bool   // Explicit content flag

    // Artist-specific metadata
    Genres     string // Top 2-3 genres joined: "art rock, grunge"
    Followers  string // Formatted: "12.4M followers" or "3.2K followers"
    Popularity int    // 0-100

    // Album-specific metadata
    AlbumType    string // "Album", "Single", "Compilation"
    ReleaseYear  string // "2020" (extracted from release_date)
    TrackCount   string // "13 tracks"
    AlbumArtists string // All album artists joined

    // Playlist-specific metadata
    Owner          string // Owner display name
    PlaylistDesc   string // Playlist description (truncated)
    PlaylistTracks string // "245 tracks"
}
```

`Title()`, `Description()`, and `FilterValue()` remain unchanged:

```go
func (i SearchListItem) Title() string       { return i.Name }
func (i SearchListItem) Description() string  { return i.Subtitle }
func (i SearchListItem) FilterValue() string  { return i.Name }
```

The conversion helpers (Layer 4) populate `Subtitle` with the rich description string at construction time. This keeps the interface contract intact and ensures list filtering works against the description.

### Layer 3: Per-Category Visual Design

**File: `internal/ui/panes/search_delegate.go`**

Each category type has a distinct two-line layout. Line 1 carries the badge + name + right-aligned metadata. Line 2 carries the description with per-field theme colors. The delegate's `Render()` method switches on category to produce the correct layout.

#### Track — `♪`

```
╭──────────────────────────────────────────────────────────────────────╮
│                                                                      │
│  ♪ Pirates of the Caribbean Theme                        [E]  3:42   │
│    Hans Zimmer, Klaus Badelt  ·  At World's End                      │
│                                                                      │
│  ♪ He's a Pirate — Epic Version                               2:15   │
│    Hans Zimmer  ·  Pirates of the Caribbean                          │
│                                                                      │
╰──────────────────────────────────────────────────────────────────────╯

Color breakdown:
  ♪           → badgeColor("track") = Success()         (green)
  Name        → TextPrimary()  /  SelectedBg()+SelectedFg() when highlighted
  [E]         → Warning()                                (yellow/amber — only if Explicit==true)
  3:42        → TextMuted()                              (dim, right-aligned)
  Artist list → ColumnSecondary()                        (supporting color)
  ·           → TextMuted()                              (dim separator)
  Album name  → ColumnTertiary()                         (metadata color)
```

**Line 1**: `badge` `name` .............. `[E]` `duration`
- Badge: `♪` in `Success()`, bold
- Name: `TextPrimary()` (or `SelectedBg()`/`SelectedFg()` when selected)
- `[E]`: Only rendered when `Explicit == true`. `Warning()` color, bold. Placed before duration.
- Duration: `TextMuted()`, right-aligned. Format: `m:ss`

**Line 2**: `  ` `artists` ` · ` `album`
- 2-space indent (aligns with name, past the badge)
- Artist names: `ColumnSecondary()`. All artists joined with `, `
- Dot separator: `TextMuted()`
- Album name: `ColumnTertiary()`

#### Artist — `★`

```
╭──────────────────────────────────────────────────────────────────────╮
│                                                                      │
│  ★ Hans Zimmer                                                       │
│    Film score, Soundtrack  ·  12.4M followers                        │
│                                                                      │
│  ★ Klaus Badelt                                                      │
│    Film score  ·  847 followers                                      │
│                                                                      │
╰──────────────────────────────────────────────────────────────────────╯

Color breakdown:
  ★           → badgeColor("artist") = KeyHint()         (purple)
  Name        → TextPrimary()  /  SelectedBg()+SelectedFg() when highlighted
  Genres      → ColumnSecondary()                        (supporting color)
  ·           → TextMuted()                              (dim separator)
  Followers   → TextMuted()                              (dim)
```

**Line 1**: `badge` `name`
- Badge: `★` in `KeyHint()`, bold
- Name: `TextPrimary()` (or selected styles)
- No right-aligned metadata on line 1 for artists

**Line 2**: `  ` `genres` ` · ` `followers`
- Genres: `ColumnSecondary()`. Top 3 joined with `, `. If empty, omit the genre segment.
- Dot separator: `TextMuted()` (only rendered if both genres and followers are present)
- Followers: `TextMuted()`. Formatted as "12.4M followers", "3.2K followers", or "847 followers"

#### Album — `◎`

```
╭──────────────────────────────────────────────────────────────────────╮
│                                                                      │
│  ◎ At World's End (Original Soundtrack)              Album  ·  2007  │
│    Hans Zimmer  ·  13 tracks                                         │
│                                                                      │
│  ◎ Interstellar                                     Single  ·  2014  │
│    Hans Zimmer  ·  1 tracks                                          │
│                                                                      │
╰──────────────────────────────────────────────────────────────────────╯

Color breakdown:
  ◎           → badgeColor("album") = SeekBar()          (cyan)
  Name        → TextPrimary()  /  SelectedBg()+SelectedFg() when highlighted
  Album type  → Info()                                   (info blue — "Album", "Single", "Compilation")
  ·           → TextMuted()                              (dim separator)
  Year        → TextMuted()                              (dim, right-aligned with type)
  Artists     → ColumnSecondary()                        (supporting color)
  Track count → TextMuted()                              (dim)
```

**Line 1**: `badge` `name` .............. `albumType` ` · ` `year`
- Badge: `◎` in `SeekBar()`, bold
- Name: `TextPrimary()` (or selected styles)
- Album type: `Info()`. Capitalized: "Album", "Single", "Compilation". Right-aligned.
- Dot separator: `TextMuted()`
- Year: `TextMuted()`. First 4 chars of `release_date`. Right-aligned alongside type.

**Line 2**: `  ` `artists` ` · ` `trackCount`
- Artist names: `ColumnSecondary()`. All artists joined with `, `
- Dot separator: `TextMuted()`
- Track count: `TextMuted()`. Format: "13 tracks"

#### Playlist — `▤`

```
╭──────────────────────────────────────────────────────────────────────╮
│                                                                      │
│  ▤ Pirates Movie Soundtracks                            245 tracks   │
│    by john_doe                                                       │
│                                                                      │
│  ▤ Epic Film Scores Collection                         1.2K tracks   │
│    by spotifyuser42  ·  The best orchestral movie scores             │
│                                                                      │
╰──────────────────────────────────────────────────────────────────────╯

Color breakdown:
  ▤            → badgeColor("playlist") = SectionHeader() (blue)
  Name         → TextPrimary()  /  SelectedBg()+SelectedFg() when highlighted
  Track count  → TextMuted()                              (dim, right-aligned)
  "by" + owner → ColumnSecondary()                        (supporting color)
  ·            → TextMuted()                              (dim separator)
  Description  → TextMuted() + Italic                     (dim, italic — only if non-empty)
```

**Line 1**: `badge` `name` .............. `trackCount`
- Badge: `▤` in `SectionHeader()`, bold
- Name: `TextPrimary()` (or selected styles)
- Track count: `TextMuted()`. Format: "245 tracks". Right-aligned.

**Line 2**: `  ` `by owner` [` · ` `description`]
- "by" prefix + owner name: `ColumnSecondary()`
- If `PlaylistDesc` is non-empty: dot separator in `TextMuted()` + description in `TextMuted()` italic, truncated to fit available width
- If `PlaylistDesc` is empty: just "by owner", no separator

### Layer 3b: Delegate Render Implementation

**File: `internal/ui/panes/search_delegate.go`**

The `Render()` method dispatches to per-category render helpers. Each helper constructs the two lines with the layout described above.

```go
func (d SearchItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
    si, ok := item.(SearchListItem)
    if !ok { return }

    isSelected := index == m.Index()
    width := m.Width()

    switch si.Category {
    case "track":
        d.renderTrack(w, si, isSelected, width)
    case "artist":
        d.renderArtist(w, si, isSelected, width)
    case "album":
        d.renderAlbum(w, si, isSelected, width)
    case "playlist":
        d.renderPlaylist(w, si, isSelected, width)
    default:
        d.renderDefault(w, si, isSelected, width)
    }
}
```

Each `render*` helper follows this structure:

1. Render the badge symbol with `badgeColor()` + Bold
2. Render the name with `TextPrimary()` or `SelectedBg()`/`SelectedFg()`
3. Render right-aligned metadata (if any) with appropriate token
4. Compose line 1: badge + name + padding + right meta
5. Render line 2 segments with per-field colors joined by `TextMuted()` dot separators
6. Write both lines to `w` via `fmt.Fprintf`

**Right-alignment strategy**: Calculate `nameMaxWidth = width - badgeWidth - 1 - rightMetaWidth - 1`. Truncate name to fit. Use `lipgloss.PlaceHorizontal(width, lipgloss.Left, leftPart)` for line 1, then overlay the right-aligned portion.

**Shared render helpers** (private to delegate):

```go
func (d SearchItemDelegate) styledBadge(category string) string
func (d SearchItemDelegate) styledName(name string, selected bool, maxW int) string
func (d SearchItemDelegate) styledDot() string  // " · " in TextMuted
func (d SearchItemDelegate) rightAlign(left, right string, width int) string
```

### Layer 4: Conversion Helper Updates

**File: `internal/ui/panes/search_delegate.go`**

Update all conversion helpers to populate the new fields:

**`tracksToListItems`**: Extract all artist names joined by ", ", album name, formatted duration, explicit flag. Build `Subtitle` from these.

```go
func tracksToListItems(tracks []domain.Track) []list.Item {
    items := make([]list.Item, len(tracks))
    for i, t := range tracks {
        artists := joinArtistNames(t.Artists)
        dur := formatSearchDuration(t.DurationMs)
        items[i] = SearchListItem{
            Category:    "track",
            Name:        t.Name,
            Subtitle:    artists + " · " + t.Album.Name + " · " + dur,
            URI:         t.URI,
            IsTrack:     true,
            ArtistNames: artists,
            AlbumName:   t.Album.Name,
            Duration:    dur,
            Explicit:    t.Explicit,
        }
    }
    return items
}
```

**`artistsToListItems`**: Extract top 3 genres joined by ", ", formatted follower count, popularity. Build `Subtitle`.

```go
func artistsToListItems(artists []domain.SearchArtist) []list.Item {
    items := make([]list.Item, len(artists))
    for i, a := range artists {
        genres := joinGenres(a.Genres, 3)
        followers := formatFollowers(a.Followers)
        subtitle := genres
        if genres != "" && followers != "" {
            subtitle += " · " + followers
        } else if followers != "" {
            subtitle = followers
        }
        items[i] = SearchListItem{
            Category:   "artist",
            Name:       a.Name,
            Subtitle:   subtitle,
            URI:        a.URI,
            IsTrack:    false,
            Genres:     genres,
            Followers:  followers,
            Popularity: a.Popularity,
        }
    }
    return items
}
```

**`albumsToListItems`**: Extract album type label, release year, track count, all artist names. Build `Subtitle`.

```go
func albumsToListItems(albums []domain.SearchAlbum) []list.Item {
    items := make([]list.Item, len(albums))
    for i, al := range albums {
        artists := joinArtistNames(al.Artists)
        year := extractYear(al.ReleaseDate)
        tc := fmt.Sprintf("%d tracks", al.TotalTracks)
        items[i] = SearchListItem{
            Category:     "album",
            Name:         al.Name,
            Subtitle:     artists + " · " + year + " · " + tc,
            URI:          al.URI,
            IsTrack:      false,
            AlbumType:    formatAlbumType(al.AlbumType),
            ReleaseYear:  year,
            TrackCount:   tc,
            AlbumArtists: artists,
        }
    }
    return items
}
```

**`playlistsToListItems`**: Extract owner, track count, description (truncated to 60 chars). Build `Subtitle`.

```go
func playlistsToListItems(playlists []domain.SearchPlaylist) []list.Item {
    items := make([]list.Item, len(playlists))
    for i, p := range playlists {
        tc := fmt.Sprintf("%d tracks", p.TrackCount)
        subtitle := "by " + p.Owner.DisplayName + " · " + tc
        desc := truncateString(p.Description, 60)
        if desc != "" {
            subtitle += " · " + desc
        }
        items[i] = SearchListItem{
            Category:       "playlist",
            Name:           p.Name,
            Subtitle:       subtitle,
            URI:            p.URI,
            IsTrack:        false,
            Owner:          p.Owner.DisplayName,
            PlaylistTracks: tc,
            PlaylistDesc:   desc,
        }
    }
    return items
}
```

### Layer 5: Formatting Helpers

Add to `search_delegate.go`:

```go
// joinArtistNames joins artist names with ", ".
func joinArtistNames(artists []domain.Artist) string {
    names := make([]string, len(artists))
    for i, a := range artists { names[i] = a.Name }
    return strings.Join(names, ", ")
}

// joinGenres joins up to max genres with ", ".
func joinGenres(genres []string, max int) string {
    if len(genres) > max { genres = genres[:max] }
    return strings.Join(genres, ", ")
}

// formatFollowers returns a human-readable follower count: "12.4M", "3.2K", "847".
func formatFollowers(n int) string {
    switch {
    case n >= 1_000_000:
        return fmt.Sprintf("%.1fM followers", float64(n)/1_000_000)
    case n >= 1_000:
        return fmt.Sprintf("%.1fK followers", float64(n)/1_000)
    default:
        return fmt.Sprintf("%d followers", n)
    }
}

// formatSearchDuration converts milliseconds to "m:ss" format.
func formatSearchDuration(ms int) string {
    totalSec := ms / 1000
    return fmt.Sprintf("%d:%02d", totalSec/60, totalSec%60)
}

// extractYear returns the first 4 characters of a release date string.
func extractYear(date string) string {
    if len(date) >= 4 { return date[:4] }
    return date
}

// formatAlbumType capitalizes the album type for display.
func formatAlbumType(t string) string {
    switch t {
    case "album":       return "Album"
    case "single":      return "Single"
    case "compilation": return "Compilation"
    default:            return t
    }
}

// truncateString truncates s to max runes, appending "…" if truncated.
func truncateString(s string, max int) string {
    runes := []rune(s)
    if len(runes) > max { return string(runes[:max-1]) + "…" }
    return s
}
```

### Domain Type: AlbumType for SearchAlbum

The `SearchAlbum` type in `domain/search.go` doesn't currently have `AlbumType`. Add it:

```go
type SearchAlbum struct {
    // ... existing fields ...
    AlbumType string `json:"album_type"` // "album", "single", or "compilation"
}
```

### Fallback Path Update

The `SearchResultData`-based fallback conversion helpers (`searchTrackItemsToListItems`, etc.) must also be updated to populate the new `SearchListItem` fields. The `SearchTrackItem`, `SearchArtistItem`, `SearchAlbumItem`, `SearchPlaylistItem` types in `search.go` may need expansion if they're still used as an intermediate representation. If they're only used in tests, update them minimally.

## Acceptance Criteria

- [ ] `SearchArtist` has `Genres`, `Followers`, `Popularity` fields, correctly unmarshaled from API JSON
- [ ] `SearchAlbum` has `AlbumType` field
- [ ] `SearchPlaylist` has `Description` field
- [ ] `Track` has `Explicit` field
- [ ] `SearchListItem` carries all metadata fields per category; `Subtitle` retained for `list.Item` interface
- [ ] Track: ♪(Success) + name + [E](Warning) + duration(TextMuted) / artists(ColumnSecondary) · album(ColumnTertiary)
- [ ] Artist: ★(KeyHint) + name / genres(ColumnSecondary) · followers(TextMuted)
- [ ] Album: ◎(SeekBar) + name + type(Info) · year(TextMuted) / artists(ColumnSecondary) · trackCount(TextMuted)
- [ ] Playlist: ▤(SectionHeader) + name + trackCount(TextMuted) / "by" owner(ColumnSecondary) [· desc(TextMuted+italic)]
- [ ] All colors use theme tokens — no hardcoded hex
- [ ] Follower counts formatted as human-readable (12.4M, 3.2K, 847)
- [ ] Duration formatted as m:ss
- [ ] Release date truncated to year
- [ ] Genres truncated to top 3
- [ ] Existing tests updated for new SearchListItem fields
- [ ] `make ci` passes

## Tasks

- [ ] Add `Explicit bool` field to `domain.Track`
      - test: JSON unmarshal with `"explicit": true` populates field; `"explicit": false` is default
- [ ] Add `Genres`, `Followers`, `Popularity` fields to `domain.SearchArtist` with custom `UnmarshalJSON`
      - test: JSON with `genres: ["rock"]`, `followers: {total: 7625607}`, `popularity: 79` unmarshals correctly; empty genres → empty slice; zero followers → 0
- [ ] Add `AlbumType string` field to `domain.SearchAlbum`
      - test: JSON with `"album_type": "compilation"` populates field
- [ ] Add `Description string` field to `domain.SearchPlaylist`, extract in existing `UnmarshalJSON`
      - test: JSON with `"description": "My cool playlist"` populates field; empty description → ""
- [ ] Expand `SearchListItem` struct with metadata fields, keep `Subtitle` for `list.Item.Description()`
      - test: `Title()` returns Name; `Description()` returns Subtitle; `FilterValue()` returns Name; track Subtitle contains artist + album + duration; artist Subtitle contains genres + followers
- [ ] Implement formatting helpers: `joinArtistNames`, `joinGenres`, `formatFollowers`, `formatSearchDuration`, `extractYear`, `formatAlbumType`, `truncateString`
      - test: `formatFollowers(7625607)` → "7.6M followers"; `formatFollowers(3200)` → "3.2K followers"; `formatFollowers(847)` → "847 followers"; `joinGenres(["a","b","c","d"], 3)` → "a, b, c"; `extractYear("2020-03-20")` → "2020"; `formatSearchDuration(222000)` → "3:42"
- [ ] Update `tracksToListItems` to populate `ArtistNames`, `AlbumName`, `Duration`, `Explicit`
      - test: track with 2 artists → ArtistNames = "A, B"; album name populated; duration formatted; explicit flag set
- [ ] Update `artistsToListItems` to populate `Genres`, `Followers`, `Popularity`
      - test: artist with 5 genres → top 3 shown; followers formatted; popularity passed through
- [ ] Update `albumsToListItems` to populate `AlbumType`, `ReleaseYear`, `TrackCount`, `AlbumArtists`
      - test: album type capitalized; year extracted from date; track count formatted; all artists joined
- [ ] Update `playlistsToListItems` to populate `Owner`, `PlaylistTracks`, `PlaylistDesc`
      - test: owner name populated; track count formatted; long description truncated with "…"
- [ ] Implement `renderTrack()`: badge(♪/Success) + name + [E](Warning, if explicit) + duration(TextMuted, right-aligned) / artists(ColumnSecondary) · album(ColumnTertiary)
      - test: track with explicit=true renders `[E]` in output; track with 2 artists shows both joined; duration right-aligned; album name present on line 2
- [ ] Implement `renderArtist()`: badge(★/KeyHint) + name / genres(ColumnSecondary) · followers(TextMuted)
      - test: artist with 5 genres shows top 3; artist with 0 followers shows "0 followers"; empty genres omits genre segment
- [ ] Implement `renderAlbum()`: badge(◎/SeekBar) + name + albumType(Info) · year(TextMuted, right-aligned) / artists(ColumnSecondary) · trackCount(TextMuted)
      - test: album type "single" renders "Single"; year extracted correctly; track count formatted
- [ ] Implement `renderPlaylist()`: badge(▤/SectionHeader) + name + trackCount(TextMuted, right-aligned) / "by" owner(ColumnSecondary) [· description(TextMuted+italic)]
      - test: playlist with description renders it after dot; playlist without description shows only "by owner"; track count right-aligned
- [ ] Implement shared delegate helpers: `styledBadge`, `styledName`, `styledDot`, `rightAlign`
      - test: `styledDot()` returns styled " · "; `rightAlign` places text at correct position for given width; `styledName` applies selected styles when isSelected=true
- [ ] Update fallback conversion helpers (`searchTrackItemsToListItems` etc.) for new fields
      - test: fallback path produces valid SearchListItems with populated metadata
- [ ] Update existing search test fixtures in `testdata/fixtures/` to include new API fields
      - test: fixture JSON includes `explicit`, `genres`, `followers`, `album_type`, `description`; existing tests still pass
