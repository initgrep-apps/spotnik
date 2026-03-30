# Unresolved Issues

Quick dump ground for issues found during implementation or review.
Triage into feature stories when ready to fix.

---

## Unbounded Retry-After accepted
**Found:** 2026-03-25 | **Source:** PR #42 Review
**Feature:** 11-api-gateway

`parseRetryAfter` in gateway.go accepts any integer including negative or very large values. A malicious proxy sending `Retry-After: 999999` would cause ~11.5 day backoff. Add bounds: `v > 0 && v <= 300`.

---

## entry.resp set on 429 path
**Found:** 2026-03-25 | **Source:** PR #42 Review
**Feature:** 11-api-gateway

gateway.go stores both resp and err for dedup waiters on 429 path. Currently safe because waiters check err first, but fragile. Consider setting `entry.resp = nil` when err != nil.

---

## Synthetic cached messages re-stamp fetchedAt
**Found:** 2026-03-25 | **Source:** PR #43 Review
**Feature:** 11-api-gateway

Cached data flows through the normal loaded-message handler and calls Set*() which re-stamps fetchedAt. This extends TTL indefinitely if panes periodically re-fire Init(). Consider adding `FromCache: true` flag or stamping only in Update() handler.

---

## fetchedAt len>0 guard blocks empty collections
**Found:** 2026-03-25 | **Source:** PR #43 Review
**Feature:** 04-library

Users with genuinely empty libraries (0 playlists, 0 albums) will never get fetchedAt stamped, causing repeated API calls. Distinguish "empty because error" from "empty because user has no data."

---

## Hardcoded time range strings in clearAllFetchingSentinels
**Found:** 2026-03-25 | **Source:** PR #43 Review
**Feature:** 08-stats

`app.go` iterates `{"short_term", "medium_term", "long_term"}` as literals. Extract to constants to prevent silent sentinel leak on drift.

---

## Pagination response can clear Offset=0 sentinel
**Found:** 2026-03-25 | **Source:** PR #43 Review
**Feature:** 04-library

A paginated loaded message (Offset>0) unconditionally clears the fetching sentinel. Narrow window for duplicate Offset=0 fetches during active pagination.

---

## PlaylistsPane `n` key creates with hardcoded "New Playlist"
**Found:** 2026-03-26 | **Source:** PR #52 Review
**Feature:** 09-playlists

Needs textinput integration to collect user-specified name before emitting `PlaylistCreateRequestMsg`. The old `PlaylistManager` had a `textinput.Model` for this.

---

## PlaylistsPane `r` key sends current name as NewName
**Found:** 2026-03-26 | **Source:** PR #52 Review
**Feature:** 09-playlists

`PlaylistRenameRequestMsg` gets `pl.Name` (current name) instead of a new name. Needs textinput integration to collect the new name.

---

## PlaylistsPane Title() calls store.PlaylistTracks() on every render
**Found:** 2026-03-26 | **Source:** PR #52 Review
**Feature:** 09-playlists

Could cache the track count in a field updated in `refreshTrackRows()` instead of reading from store on every `Title()` call.

---

## Playlist deletion (x key) removed
**Found:** 2026-03-26 | **Source:** PR #52 Review
**Feature:** 09-playlists

The `x` key was using `PlaylistRemoveRequestMsg` (track removal) for playlist deletion. Removed since playlist unfollow requires a different message type (`PlaylistUnfollowRequestMsg`). Add proper playlist deletion support when needed.

---

## TopTracksPane "Pop" column always shows "--"
**Found:** 2026-03-26 | **Source:** PR #53 Review
**Feature:** 08-stats

`domain.Track` lacks a `Popularity` field. The Spotify top-tracks API returns popularity, but it's not captured in the domain model. Either add `Popularity int` to `domain.Track` and populate the column, or replace the column with extra width for Track/Artist.

---

## Gateway.Snapshot() is best-effort, not atomic
**Found:** 2026-03-26 | **Source:** PR #56 Review
**Feature:** 11-api-gateway

Token bucket and gateway mutex are acquired separately. Snapshot fields may be from slightly different points in time. Acceptable for display purposes but worth documenting.

---

## PollingSnapshotMsg.TickIntervalMs is misleading
**Found:** 2026-03-26 | **Source:** PR #56 Review
**Feature:** 14-nerd-status

Shows the polling decision interval (3000ms, 10000ms) but the actual tea.Tick fires every 1000ms. Consider renaming to `PollIntervalMs` or displaying the actual tick interval separately.

---

## ARCHITECTURE.md references deleted pane names
**Found:** 2026-03-26 | **Source:** PR #58 Review
**Feature:** 00-architecture

The ASCII diagram at line 33 still shows `LibraryPane`, `PlayerPane`, and `QueuePane`. Test examples at lines 621/628 reference `PlayerPane`. These types no longer exist. Update to reflect the 10-pane grid layout.

---

## formatDuration duplication
**Found:** 2026-03-26 | **Source:** PR #58 Review
**Feature:** 13-nowplaying

`formatDuration` in `gradient.go` and `formatDurationMs` in `nowplaying.go` are duplicate implementations. Extract to a shared utility in `components/`.
