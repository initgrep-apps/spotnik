# Feature 21 — Import Boundary Fixes

> **Refactoring:** Remove the two `ui/ -> api/` import violations in `devices.go` and
> `search.go` by defining UI-facing types in the panes package.

## Context

CLAUDE.md rule: "`ui/` never imports `api/` — data flows through messages and store only."
The architecture review found two production files that violate this boundary:

1. `internal/ui/panes/devices.go` (line 11) imports `api` for the `api.Device` type
2. `internal/ui/panes/search.go` (line 13) imports `api` for `api.SearchResult`,
   `api.Track`, `api.SearchArtist`, `api.SearchAlbum`, `api.SearchPlaylist`

The pattern used by other panes (player, library, queue) is to carry data through
message types defined in `messages.go` using either primitive fields or UI-facing
structs that mirror the API types. The panes package already has `messages.go` with
this pattern.

---

## Task 1: Remove api/ import from devices.go

**Problem:** `devices.go` imports `api` and uses `api.Device` in the `devicesLoadedMsg`
struct and in the `DeviceOverlay` rendering logic.

**Fix:**
1. Define a `DeviceInfo` struct in `internal/ui/panes/messages.go` with the fields
   that `DeviceOverlay` actually uses from `api.Device`:
   ```go
   type DeviceInfo struct {
       ID       string
       Name     string
       Type     string
       IsActive bool
   }
   ```
2. Update `NewDevicesLoadedMsg` (the constructor already exists in messages.go) to
   accept `[]DeviceInfo` instead of `[]api.Device`
3. The conversion from `[]api.Device` to `[]DeviceInfo` happens in `app.go` inside
   the `buildFetchDevicesCmd` closure (which already imports `api/`)
4. Update `DeviceOverlay` to use `DeviceInfo` instead of `api.Device`
5. Remove the `api` import from `devices.go`

**Files:**
- `internal/ui/panes/messages.go` — Add `DeviceInfo` struct
- `internal/ui/panes/devices.go` — Use `DeviceInfo`, remove `api` import
- `internal/app/app.go` — Convert `[]api.Device` → `[]DeviceInfo` in buildFetchDevicesCmd

**Tests:**
- Unit test: DeviceOverlay renders correctly with `DeviceInfo` data
- Unit test: verify `devices.go` no longer imports `api` (can be checked by `go vet` or
  a grep-based test)
- Existing device overlay tests should pass

---

## Task 2: Remove api/ import from search.go

**Problem:** `search.go` imports `api` and uses `api.SearchResult`, `api.Track`,
`api.SearchArtist`, `api.SearchAlbum`, `api.SearchPlaylist` in rendering helpers
(around lines 498-569).

**Fix:**
1. Define UI-facing search result types in `internal/ui/panes/messages.go`:
   ```go
   type SearchResultData struct {
       Tracks    []SearchTrackItem
       Artists   []SearchArtistItem
       Albums    []SearchAlbumItem
       Playlists []SearchPlaylistItem
   }

   type SearchTrackItem struct {
       URI    string
       Name   string
       Artist string
       Album  string
   }

   type SearchArtistItem struct {
       URI  string
       Name string
   }

   type SearchAlbumItem struct {
       URI    string
       Name   string
       Artist string
   }

   type SearchPlaylistItem struct {
       URI   string
       Name  string
       Owner string
   }
   ```
2. Update `SearchResultsMsg` to carry `SearchResultData` instead of `*api.SearchResult`
3. The conversion from `api.SearchResult` to `SearchResultData` happens in
   `buildSearchCmd` in `app.go` (which already imports `api/`)
4. Update `SearchOverlay` rendering to use the UI-facing types
5. Remove the `api` import from `search.go`

**Files:**
- `internal/ui/panes/messages.go` — Add search UI types
- `internal/ui/panes/search.go` — Use UI types, remove `api` import
- `internal/app/app.go` — Convert api search results to UI types in buildSearchCmd

**Tests:**
- Unit test: SearchOverlay renders correctly with UI-facing search data
- Existing search tests should pass
- Verify no `api` import in `search.go`

---

## Verification

After both tasks, run:
```bash
grep -r '"github.com/initgrep-apps/spotnik/internal/api"' internal/ui/
```
This must return zero results.

---

## Acceptance Criteria

- [ ] `devices.go` has zero imports from `internal/api`
- [ ] `search.go` has zero imports from `internal/api`
- [ ] `DeviceInfo` UI type defined in `messages.go`
- [ ] `SearchResultData` and related UI types defined in `messages.go`
- [ ] Conversions from api types happen in `app.go` command builders
- [ ] `grep -r 'internal/api' internal/ui/` returns zero results
- [ ] All existing tests pass
- [ ] `make ci` passes
