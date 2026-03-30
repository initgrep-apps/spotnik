---
title: "API Cleanup Follow-up"
feature: 11-api-gateway
status: done
---

## Background
Features 19-27 addressed the architecture review findings. Three items were deferred due to scope/risk: (1) Get prefix on API getters -- Go convention violation, 10 methods named GetX should be X, mechanical rename across 17 files; (2) TokenProvider interface -- token stored as string in BaseClient, architecture spec calls for per-request resolution via interface; (3) search.go imports api/ -- the only remaining ui/ -> api/ import boundary violation.

## Design

### Get Prefix Removal
10 methods across 4 client files + 4 interface files + 1 mock file + 10 call sites in commands.go + ~27 call sites in test files. Total: 17 files, ~64 line edits.

| Current | New | Client |
|---|---|---|
| GetPlaybackState | PlaybackState | Player |
| GetQueue | Queue | Player |
| GetPlaylists | Playlists | LibraryClient |
| GetPlaylistTracks | PlaylistTracks | LibraryClient |
| GetSavedAlbums | SavedAlbums | LibraryClient |
| GetLikedTracks | LikedTracks | LibraryClient |
| GetRecentlyPlayed | RecentlyPlayed | LibraryClient, UserClient |
| GetDevices | Devices | DevicesClient |
| GetTopTracks | TopTracks | UserClient |
| GetTopArtists | TopArtists | UserClient |

### TokenProvider Interface
```go
type TokenProvider interface {
    AccessToken(ctx context.Context) (string, error)
}

type StaticTokenProvider struct {
    Token string
}
```
Update BaseClient to use TokenProvider per-request. Add `NewBaseClientWithProvider` for future RefreshableTokenProvider.

### search.go Import Boundary Fix
Define UI-facing search types (SearchResultData, SearchTrackItem, etc.) in messages.go. Update SearchResultsMsg to carry converted data. Convert api.SearchResult to SearchResultData in buildSearchCmd. Remove api import from search.go.

### Verification
```bash
grep -r '"github.com/initgrep-apps/spotnik/internal/api"' internal/ui/
# Must return zero results
```

## Acceptance Criteria
- [ ] Zero API getter methods start with `Get`
- [ ] All interfaces, mocks, and callers updated with new names
- [ ] `TokenProvider` interface exists in `api/token.go`
- [ ] `BaseClient` uses `TokenProvider` per-request, not stored string
- [ ] `StaticTokenProvider` provides backward compatibility
- [ ] `search.go` has zero imports from `internal/api`
- [ ] `SearchResultData` UI types carry data via messages, not store reads
- [ ] `grep -r 'internal/api' internal/ui/` returns zero results
- [ ] All existing tests pass
- [ ] `make ci` passes

## Tasks
- [ ] Remove Get prefix from API getters -- mechanical rename across 17 files
      - test: all existing tests pass; compile-time interface checks still pass
- [ ] Introduce TokenProvider interface in internal/api/token.go
      - test: StaticTokenProvider returns fixed token; newRequest() calls TokenProvider.AccessToken
- [ ] Fix search.go import boundary violation -- define UI-facing types, carry data on messages
      - test: SearchOverlay renders correctly with SearchResultData; conversion correct; grep verification zero api imports in ui/
