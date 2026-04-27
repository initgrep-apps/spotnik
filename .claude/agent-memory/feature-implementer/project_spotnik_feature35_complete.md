---
name: project_spotnik_feature35_complete
description: Feature 35 (Type Design Alignment): message exports, AlbumsLoadedMsg Offset, SearchResult to domain, Elm purity for DevicesLoadedMsg, package doc shadowing gotcha
type: project
---

## Feature 35 — Type Design Alignment

**What was built:**
- Move `StatsLoadedMsg` from `stats.go` → `messages.go`. Shared msgs consolidated.
- Add `Offset int` to `AlbumsLoadedMsg` + append/replace handler in app.go. Matches LibraryLoadedMsg pattern.
- Export `devicesLoadedMsg` → `DevicesLoadedMsg`. Drop `NewDevicesLoadedMsg` constructor.
- Move store mutations (`SetDevicesError`, `ClearDevicesError`, `SetDevicesFetchedAt`) from `DeviceOverlay.Update()` → `app.Update()`. Elm purity fix.
- Create `internal/domain/search.go` w/ all search types (SearchResult, SearchArtist, SearchAlbum, SearchPlaylist, Search*Result).
- Add 8 type aliases in `api/models.go` for back-compat.
- Drop `api/` import from `state/store.go`.
- Fix stale store.go comment ("Set by build*Cmd" → "Set by Update() handlers"). Drop orphaned TODO.

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/domain/search.go` — search result types moved from api/search.go. NO package-level doc comment (see gotcha).
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/api/models.go` — 8 type aliases at EOF for SearchXxx types.
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/panes/messages.go` — StatsLoadedMsg + DevicesLoadedMsg added. AlbumsLoadedMsg gained Offset field.
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/app/app.go` — new `case panes.DevicesLoadedMsg:` handler w/ store mutations + conditional overlay forward.

**Patterns established:**
- Moving store mutations from pane → root app.Update(): pane still gets `DevicesLoadedMsg` forwarded for local render state (devices list). Pane handler returns `nil` cmd, no store writes.
- `DevicesLoadErrorMsg` now dead code (no producer), kept because `toast_routing_test.go` tests it directly. Don't remove until test updated.
- AlbumsLoadedMsg.Offset=0 → replace, Offset>0 → append. Same as LibraryLoadedMsg + LikedTracksLoadedMsg.
- Forward `DevicesLoadedMsg` to `devicePane` only when `a.deviceOverlayOpen` — correct (matches pre-PR catch-all). If closed, store updates but overlay list stays stale until next open re-fetch (staleness TTL handles).

**Gotchas:**
- **Package doc shadowing**: file w/ `// Package domain ...` right before `package domain` in non-primary doc file shadows canonical doc in `types.go`. `go doc` picks last alphabetically. Fix: drop comment block entirely from secondary file (no comment between block and `package domain`). Caught in PR review, fixed in follow-up commit.
- **stats.go `domain` import unused** after removing StatsLoadedMsg (referenced `domain.Track` + `domain.FullArtist`). Drop import or build fails.
- **devices_test.go** tests for store mutations (`StampsFetchedAt`, `ErrorDoesNotStampFetchedAt`) must be deleted — behavior moved to app_test.go. Overlay tests now only verify device list population/no-change.
- `DevicesLoadedMsg` w/ error: overlay receives msg but does NOT update `d.devices` — `if m.Err == nil` guard. Error path preserves devices list as-is.

**Testing notes:**
- 4 new tests in `app_test.go`: DevicesLoadedMsg nil err, DevicesLoadedMsg w/ err, AlbumsLoadedMsg Offset=0, AlbumsLoadedMsg Offset>0.
- `errMake` helper in `devices_test.go` avoids `fmt.Errorf` where `errors.New` suffices.
- Coverage: 82.6% (above 80% threshold).
- PR: https://github.com/initgrep-apps/spotnik/pull/40