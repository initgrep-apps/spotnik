---
title: "Fix: Search Rich Data Pipeline, Enter-Close, Placeholder Leak"
feature: 19-search-redesign
status: done
---

## Background

Post-implementation testing of stories 87–90 revealed three bugs:

1. **Rich metadata not showing** — search results display only name + "0:00" with empty artist/album. Root cause: the app-layer conversion pipeline (`commands.go`) strips all metadata when converting through intermediate `SearchTrackItem`/`SearchArtistItem`/`SearchAlbumItem`/`SearchPlaylistItem` types. These types only carry URI/Name/Artist. The `convertSearchTrackItems()` function then converts BACK to `domain.Track` with only URI and Name, so the store holds data-stripped objects. Story 87 added rich rendering in the delegate but didn't update this pipeline.

2. **Overlay closes on Enter** — Story 88 removed `SearchClosedMsg` from the overlay's `handleEnter()`, but `app.go` still has `a.closeSearch()` in both the `PlayTrackMsg` and `PlayContextMsg` handlers. The overlay-side fix is correct; the app-side handler was missed.

3. **Placeholder leaks with locked prefix** — When a prefix like `:songs` is locked as a Prompt tag, `input.Value()` is empty (the clean query). The `textinput` component renders `Placeholder` text whenever Value is empty, so the cycling placeholder (e.g. `:artists find your favorite artists...`) shows behind the cursor even though `:songs` is already locked. The placeholder tick also keeps cycling because the tick guard checks `o.input.Value() == ""`.

## Design

### Fix 1: Eliminate Intermediate Search Types

**The problem:** Data flows through a lossy roundtrip:
```
domain.SearchResult → panes.SearchResultData (intermediate) → domain.Track (stripped)
```

**The fix:** Replace the intermediate types in `SearchResultData` with domain types directly. This eliminates the lossy conversion entirely.

**File: `internal/ui/panes/messages.go`**

Replace `SearchResultData` fields to use domain types:

```go
import "github.com/initgrep-apps/spotnik/internal/domain"

type SearchResultData struct {
    Tracks         []domain.Track
    TracksTotal    int
    Artists        []domain.SearchArtist
    ArtistsTotal   int
    Albums         []domain.SearchAlbum
    AlbumsTotal    int
    Playlists      []domain.SearchPlaylist
    PlaylistsTotal int
}
```

Delete the four intermediate types: `SearchTrackItem`, `SearchArtistItem`, `SearchAlbumItem`, `SearchPlaylistItem`.

**File: `internal/app/commands.go`**

Simplify `convertSearchResult()` — copy domain slices directly instead of field-picking:

```go
func convertSearchResult(r *api.SearchResult) *panes.SearchResultData {
    if r == nil {
        return nil
    }
    return &panes.SearchResultData{
        Tracks:         r.Tracks.Items,
        TracksTotal:    r.Tracks.Total,
        Artists:        r.Artists.Items,
        ArtistsTotal:   r.Artists.Total,
        Albums:         r.Albums.Items,
        AlbumsTotal:    r.Albums.Total,
        Playlists:      r.Playlists.Items,
        PlaylistsTotal: r.Playlists.Total,
    }
}
```

Delete the four `convertSearch*Items()` functions (`convertSearchTrackItems`, `convertSearchArtistItems`, `convertSearchAlbumItems`, `convertSearchPlaylistItems`).

Update the `SearchPageLoadedMsg` handler in `app.go` to pass domain types directly to the store:

```go
case panes.SearchPageLoadedMsg:
    // ...
    if r := m.Results; r != nil {
        a.store.AppendSearchTracks(r.Tracks, r.TracksTotal)
        a.store.AppendSearchArtists(r.Artists, r.ArtistsTotal)
        a.store.AppendSearchAlbums(r.Albums, r.AlbumsTotal)
        a.store.AppendSearchPlaylists(r.Playlists, r.PlaylistsTotal)
    }
```

**File: `internal/ui/panes/search_delegate.go`**

Update the fallback converters (`searchTrackItemsToListItems`, etc.) to use domain types instead of the deleted intermediate types. These are the overlay-standalone/test path fallbacks used when `rebuildFromResults()` is called. Since `SearchResultData` now carries domain types, these fallback converters become identical to `tracksToListItems`/`artistsToListItems`/etc. Delete them and have `rebuildFromResults()` call the same converters as `rebuildFromStore()`.

**File: `internal/ui/panes/search.go` — `rebuildFromResults()`**

```go
func (o *SearchOverlay) rebuildFromResults() {
    if o.results == nil {
        return
    }
    // Same converters as rebuildFromStore — SearchResultData now carries domain types.
    o.rebuildFromStore(o.results.Tracks, o.results.Artists, o.results.Albums, o.results.Playlists)
}
```

**File: `internal/app/commands_test.go` / `internal/app/app_test.go`**

Update any test that constructs `SearchTrackItem`/`SearchResultData` to use domain types. The mock helper (`apitest/mock.go`) already returns `*api.SearchResult` with domain types, so tests that go through the mock path should work unchanged.

### Fix 2: Don't Close Overlay on Play

**File: `internal/app/app.go`**

Remove the `closeSearch()` calls from both `PlayTrackMsg` and `PlayContextMsg` handlers:

```go
case panes.PlayContextMsg:
    // Overlay stays open — only Esc closes it (Story 88).
    return a, a.buildPlayContextCmd(m.ContextURI)

case panes.PlayTrackMsg:
    // Overlay stays open — only Esc closes it (Story 88).
    return a, a.buildPlayTrackCmd(m.TrackURI)
```

### Fix 3: Clear Placeholder When Prefix Locked

**File: `internal/ui/panes/search_prefix.go` — `promoteToPromptTag()`**

After setting the Prompt tag, replace the cycling placeholder with a simple static one:

```go
func (o *SearchOverlay) promoteToPromptTag() {
    // ... existing promotion logic ...

    // Replace cycling placeholder — the prefix tag makes prefix hints redundant.
    o.input.Placeholder = "search..."
}
```

**File: `internal/ui/panes/search_prefix.go` — `demoteFromPromptTag()`**

Restore the cycling placeholder when the prefix is demoted:

```go
func (o *SearchOverlay) demoteFromPromptTag() {
    // ... existing demotion logic ...

    // Restore cycling placeholder since we're back to normal input mode.
    o.input.Placeholder = searchPlaceholders[o.placeholderIdx]
}
```

**File: `internal/ui/panes/search_prefix.go` — `syncInputToTab()`**

When tab switches to a non-All tab (which locks a prefix), set the simple placeholder. When switching back to All, restore cycling:

```go
func (o *SearchOverlay) syncInputToTab() {
    // ... existing sync logic ...

    if o.activeTab == TabAll {
        // ... existing restore logic ...
        // Restore cycling placeholder.
        o.input.Placeholder = searchPlaceholders[o.placeholderIdx]
    } else if prefix, ok := tabToPrefixMap[o.activeTab]; ok {
        // ... existing lock logic ...
        // Replace cycling placeholder.
        o.input.Placeholder = "search..."
    }
}
```

**File: `internal/ui/panes/search.go` — `searchPlaceholderTickMsg` handler**

Add a guard so the tick does not fire when a prefix is locked:

```go
case searchPlaceholderTickMsg:
    if o.input.Value() == "" && o.prefixState == PrefixNone {
        // ... existing cycling logic ...
    }
```

This guard already exists (line 322). No change needed — the tick simply won't re-arm when `prefixState != PrefixNone` because the condition fails. The placeholder text set by `promoteToPromptTag()` persists until demotion.

## Acceptance Criteria

- [ ] Search results show rich metadata: tracks display artists · album · duration; artists show genres · followers; albums show type · year · artists · tracks; playlists show owner · description · tracks
- [ ] Duration shows real values (e.g. "3:42") not "0:00"
- [ ] Explicit tracks show [E] badge
- [ ] `SearchTrackItem`, `SearchArtistItem`, `SearchAlbumItem`, `SearchPlaylistItem` types are deleted
- [ ] `SearchResultData` uses domain types directly (`domain.Track`, `domain.SearchArtist`, etc.)
- [ ] `convertSearchResult()` passes domain slices through without field-picking
- [ ] `convertSearch*Items()` converter functions are deleted
- [ ] Fallback converters in `search_delegate.go` are deleted — `rebuildFromResults()` reuses `rebuildFromStore()`
- [ ] Enter plays the selected track/context without closing the overlay
- [ ] Only Esc closes the overlay
- [ ] When a prefix is locked, placeholder shows "search..." not the cycling prefix hints
- [ ] Cycling placeholder resumes when prefix is cleared or tab switches to All
- [ ] `make ci` passes

## Tasks

- [ ] Replace `SearchResultData` fields with domain types in `messages.go`; delete 4 intermediate types
      - test: `SearchResultData` fields are `[]domain.Track`, `[]domain.SearchArtist`, etc.; old types do not exist
- [ ] Simplify `convertSearchResult()` in `commands.go` to pass domain slices directly; delete 4 `convertSearch*Items()` functions
      - test: `convertSearchResult()` returns domain types; `convertSearchTrackItems` etc. do not exist; search integration test verifies full metadata roundtrip
- [ ] Update `SearchPageLoadedMsg` handler in `app.go` to pass `r.Tracks` directly to `AppendSearchTracks`
      - test: store contains tracks with full metadata (Artists, DurationMs, Explicit) after SearchPageLoadedMsg
- [ ] Delete fallback converters in `search_delegate.go`; `rebuildFromResults()` delegates to `rebuildFromStore()`
      - test: `searchTrackItemsToListItems` etc. do not exist; `rebuildFromResults` produces same items as `rebuildFromStore`
- [ ] Update all tests that construct `SearchTrackItem`/`SearchResultData` to use domain types
      - test: `make ci` passes with no references to deleted types
- [ ] Remove `a.closeSearch()` from `PlayTrackMsg` and `PlayContextMsg` handlers in `app.go`
      - test: after PlayTrackMsg, `a.SearchOpen()` is still true; after Esc, it is false
- [ ] Set placeholder to "search..." in `promoteToPromptTag()`; restore cycling in `demoteFromPromptTag()` and `syncInputToTab()` (All tab)
      - test: after locking prefix, `o.Placeholder()` is "search..."; after demotion, it matches `searchPlaceholders[idx]`; after tab to All, cycling placeholder restored
