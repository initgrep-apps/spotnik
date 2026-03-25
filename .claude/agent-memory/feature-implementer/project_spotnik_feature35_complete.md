---
name: project_spotnik_feature35_complete
description: Feature 35 (Type Design Alignment): message exports, AlbumsLoadedMsg Offset, SearchResult to domain, Elm purity for DevicesLoadedMsg, package doc shadowing gotcha
type: project
---

## Feature 35 — Type Design Alignment

**What was built:**
- Moved `StatsLoadedMsg` from `stats.go` to `messages.go` (all shared messages now in one file)
- Added `Offset int` to `AlbumsLoadedMsg` + append/replace handler in app.go (matches LibraryLoadedMsg pattern)
- Exported `devicesLoadedMsg` → `DevicesLoadedMsg`, removed `NewDevicesLoadedMsg` constructor
- Moved store mutations (`SetDevicesError`, `ClearDevicesError`, `SetDevicesFetchedAt`) from `DeviceOverlay.Update()` to root `app.Update()` (Elm purity fix)
- Created `internal/domain/search.go` with all search types (SearchResult, SearchArtist, SearchAlbum, SearchPlaylist, Search*Result)
- Added 8 type aliases in `api/models.go` for backward compat
- Removed `api/` import from `state/store.go`
- Fixed stale store.go comment ("Set by build*Cmd" → "Set by Update() handlers") and removed orphaned TODO

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/domain/search.go` — search result types moved from api/search.go; NO package-level doc comment (see gotcha)
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/api/models.go` — 8 type aliases at end of file for SearchXxx types
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/panes/messages.go` — StatsLoadedMsg and DevicesLoadedMsg added here; AlbumsLoadedMsg updated with Offset field
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/app.go` — new `case panes.DevicesLoadedMsg:` handler with store mutations + conditional overlay forward

**Patterns established:**
- When moving store mutations from a pane to root app.Update(): the pane still gets `DevicesLoadedMsg` forwarded so it can update its local render state (devices list). The pane's handler returns `nil` command, no store writes.
- `DevicesLoadErrorMsg` is now effectively dead code (no producer), but kept alive because `toast_routing_test.go` tests it directly. Do not remove until that test is updated.
- AlbumsLoadedMsg.Offset=0 → replace, Offset>0 → append. Same as LibraryLoadedMsg and LikedTracksLoadedMsg.
- Only forward `DevicesLoadedMsg` to `devicePane` when `a.deviceOverlayOpen` — this is correct (same pre-PR behavior via catch-all); if closed, store is updated but overlay render list stays stale until next open triggers re-fetch (staleness TTL handles this).

**Gotchas:**
- **Package doc shadowing**: Adding a file with `// Package domain ...` immediately before `package domain` in a file that is NOT the primary package doc file shadows the canonical doc in `types.go`. `go doc` picks the last one alphabetically. Fix: remove the comment block entirely from the secondary file (don't add any comment between the comment block and `package domain`). This was caught in PR review and fixed in a follow-up commit.
- **stats.go `domain` import went unused** after removing StatsLoadedMsg (which referenced `domain.Track` and `domain.FullArtist`). Must remove the import from stats.go or the build fails.
- **devices_test.go** tests that previously tested store mutations (`StampsFetchedAt`, `ErrorDoesNotStampFetchedAt`) must be deleted — those behaviors moved to app_test.go. The overlay-level tests now only verify device list population/no-change behavior.
- When `DevicesLoadedMsg` is handled with an error, the overlay receives the message but does NOT update its `d.devices` field — the `if m.Err == nil` guard ensures this. On error path, devices list is preserved as-is.

**Testing notes:**
- 4 new tests in `app_test.go`: DevicesLoadedMsg nil error, DevicesLoadedMsg with error, AlbumsLoadedMsg Offset=0, AlbumsLoadedMsg Offset>0
- `errMake` helper added to `devices_test.go` to avoid `fmt.Errorf` where only `errors.New` is needed
- Coverage: 82.6% (well above 80% threshold)
- PR: https://github.com/initgrep-apps/spotnik/pull/40
