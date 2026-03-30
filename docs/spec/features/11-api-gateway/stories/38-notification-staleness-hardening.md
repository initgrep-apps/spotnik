---
title: "Notification & Staleness Hardening"
feature: 11-api-gateway
status: done
---

## Background
PR reviews of Feature 31 (Notifications) and Feature 32 (Staleness) identified issues in error handling and data integrity: assertion safety in alerts.Update(), Init() batching, fetchedAt nil guards, stats double-stamping, TOCTOU fetching sentinel, and staleness gate data delivery.

Source: `docs/issues.md` -- PR #36 issues 2-4; PR #37 issues 6-8, 10. Depends on: Feature 36.

## Design

### alerts.Update() Type Assertion
Add defensive comment explaining BubbleUp's AlertModel.Update always returns AlertModel. Batch alerts.Init() return value into commands.

### fetchedAt Nil Guards
In each Set method, only stamp fetchedAt when data is non-nil/non-empty. Exception: SetPlaybackState -- nil state is valid (204 = nothing playing).

### Stats Double-Stamping
Remove statsFetchedAt stamping from SetTopTracks() and SetTopArtists(). Add StampStatsFetchedAt(timeRange) called once in StatsLoadedMsg handler.

### Fetching Sentinels (TOCTOU)
Add boolean fetching fields per domain. In Update() staleness gates, check fetching before dispatching. Clear fetching on loaded message.

### Cached Data Delivery
When staleness gate blocks a request with Offset=0, send a synthetic loaded message with cached data so the pane can initialize.

## Acceptance Criteria
- [ ] alerts.Init() return value batched into commands
- [ ] alerts.Update() type assertion documented
- [ ] SetPlaylists(nil) does NOT update fetchedAt
- [ ] SetPlaybackState(nil) still stamps fetchedAt (204 is valid)
- [ ] statsFetchedAt stamped once after both track and artist setters
- [ ] Fetching sentinels prevent TOCTOU duplicate fetches
- [ ] Fresh data sends cached LibraryLoadedMsg to pane Init()
- [ ] `make ci` passes

## Tasks
- [ ] Fix alerts.Update() type assertion and alerts.Init() batching in app.go
      - test: Init() returns batched commands including alerts init
- [ ] Add alert type registration validation in notifications.go
      - test: all 5 alert types produce non-nil commands after registration
- [ ] Guard fetchedAt stamping on nil/empty data in store.go
      - test: SetPlaylists(nil) does NOT update fetchedAt; SetPlaylists(data) DOES; SetPlaybackState(nil) still stamps
- [ ] Fix stats double-stamping -- add StampStatsFetchedAt() in store.go
      - test: SetTopTracks does NOT stamp; StampStatsFetchedAt updates; StatsStale true before stamp
- [ ] Add fetching sentinel for TOCTOU race in store.go and app.go
      - test: fetching=true prevents duplicate dispatch; cleared on loaded; doesn't block paginated fetches
- [ ] Send cached data when staleness gate blocks in app.go
      - test: fresh playlists still send LibraryLoadedMsg with cached data; stale trigger API fetch
- [ ] Update issues.md
      - test: docs change only
