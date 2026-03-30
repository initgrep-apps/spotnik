---
title: "Type Design Alignment"
feature: 00-architecture
status: open
---

## Background
PR reviews identified 4 type design inconsistencies in message and store types. These changes improve the codebase's adherence to its own conventions.

**Source:** `docs/issues.md` -- PR #34 issues 11-14

**Depends on:** Feature 34 (Docs, Dead Code & Defensive Init)

## Design

### Task 1: Move StatsLoadedMsg from stats.go to messages.go

**Problem:** `StatsLoadedMsg` is defined in `internal/ui/panes/stats.go` (lines 48-57) while all other shared message types live in `internal/ui/panes/messages.go`. This breaks the convention established by Feature 29.

**Fix:**
1. Move the `StatsLoadedMsg` type definition from `stats.go` to `messages.go` (after `QueueLoadedMsg` or similar data-carrying messages)
2. Preserve all doc comments
3. Remove from `stats.go`

### Task 2: Add Offset field to AlbumsLoadedMsg

**Problem:** `AlbumsLoadedMsg` (messages.go lines 128-131) lacks an `Offset int` field. Both `LibraryLoadedMsg` and `LikedTracksLoadedMsg` have Offset for pagination. This inconsistency prevents future album pagination support.

**Fix:**
1. Add `Offset int` field to `AlbumsLoadedMsg`:
   ```go
   type AlbumsLoadedMsg struct {
       Items  []domain.SavedAlbum
       Offset int
       Err    error
   }
   ```
2. Update `buildFetchAlbumsCmd` in `internal/app/commands.go` to pass the offset
3. Update the `AlbumsLoadedMsg` handler in `internal/app/app.go` to use `m.Offset` for append vs replace logic

### Task 3: Export devicesLoadedMsg for consistency

**Problem:** `devicesLoadedMsg` in `internal/ui/panes/devices.go` (lines 19-22) is the only unexported data-carrying message type. All others (LibraryLoadedMsg, AlbumsLoadedMsg, SearchResultsMsg, etc.) are fully exported. The `NewDevicesLoadedMsg` constructor pattern adds complexity without clear benefit.

**Fix:**
1. Rename `devicesLoadedMsg` to `DevicesLoadedMsg` in `devices.go`
2. Export the fields: `devices` -> `Devices`, `err` -> `Err`
3. Move the type definition to `messages.go` (same convention as other messages)
4. Update `NewDevicesLoadedMsg` constructor to return the exported type directly (or remove the constructor and use struct literal like all other messages)
5. Update all references in `devices.go` and `commands.go`

### Task 4: Migrate SearchResult from api/ to domain/

**Problem:** `internal/state/store.go` imports `internal/api` (line 10) solely for `api.SearchResult` (line 69). This violates the domain boundary -- state/ should only import domain/, not api/.

**Fix:**
1. Move the `SearchResult` type from wherever it's defined in api/ to `internal/domain/types.go` (or a new `internal/domain/search.go`)
2. Add a type alias in `api/models.go` for backward compatibility: `type SearchResult = domain.SearchResult`
3. Update `store.go` to use `domain.SearchResult` instead of `api.SearchResult`
4. Remove the `api` import from `store.go`

### Task 5: Update issues.md

Mark PR #34 issues 11, 12, 13, 14 as resolved in `docs/issues.md`.

### Verification

```bash
# StatsLoadedMsg in messages.go, not stats.go
grep -n 'type StatsLoadedMsg' internal/ui/panes/messages.go
# Expected: 1 match
grep -n 'type StatsLoadedMsg' internal/ui/panes/stats.go
# Expected: ZERO matches

# AlbumsLoadedMsg has Offset
grep -A3 'type AlbumsLoadedMsg' internal/ui/panes/messages.go
# Expected: Items, Offset, Err fields

# devicesLoadedMsg exported
grep -n 'type devicesLoadedMsg' internal/ui/panes/
# Expected: ZERO matches (now DevicesLoadedMsg)

# store.go no longer imports api/
grep 'internal/api' internal/state/store.go
# Expected: ZERO matches

make ci
# Expected: Full pass
```

## Acceptance Criteria
- [ ] `StatsLoadedMsg` lives in `messages.go`, not `stats.go`
- [ ] `AlbumsLoadedMsg` has an `Offset int` field for pagination
- [ ] `devicesLoadedMsg` is exported as `DevicesLoadedMsg` in `messages.go`
- [ ] `SearchResult` type lives in `domain/`, not `api/`
- [ ] `store.go` no longer imports `internal/api`
- [ ] All existing tests pass
- [ ] `make ci` passes

## Tasks
- [ ] Move `StatsLoadedMsg` from `stats.go` to `messages.go`
      - test: `make ci` -- compilation verifies all references still work
- [ ] Add `Offset int` field to `AlbumsLoadedMsg` and update handlers
      - test: verify AlbumsLoadedMsg with Offset=0 replaces albums in store; verify Offset>0 appends
- [ ] Export `devicesLoadedMsg` as `DevicesLoadedMsg` and move to `messages.go`
      - test: `make ci`; verify DevicesLoadedMsg with nil error populates device list
- [ ] Migrate `SearchResult` from `api/` to `domain/` package
      - test: `grep -r 'internal/api' internal/state/` returns ZERO matches; `make ci` passes
- [ ] Mark resolved issues in `docs/issues.md`
      - test: None needed (documentation-only)
