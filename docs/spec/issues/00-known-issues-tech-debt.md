# Known Issues & Technical Debt

> Tracked issues from PR reviews. Items here are non-blocking but should be
> addressed in future features or cleanup passes.

---

## From PR #42 Review — Gateway Hardening (2026-03-25)

### Robustness

- [ ] **Unbounded Retry-After accepted** — `parseRetryAfter` in gateway.go accepts any integer including negative or very large values. A malicious proxy sending `Retry-After: 999999` would cause ~11.5 day backoff. Add bounds: `v > 0 && v <= 300`.
- [ ] **entry.resp set on 429 path** — gateway.go stores both resp and err for dedup waiters on 429 path. Currently safe because waiters check err first, but fragile. Consider setting `entry.resp = nil` when err != nil.

---

## From PR #43 Review — Notification & Staleness Hardening (2026-03-25)

### Design

- [ ] **Synthetic cached messages re-stamp fetchedAt** — Cached data flows through the normal loaded-message handler and calls Set*() which re-stamps fetchedAt. This extends TTL indefinitely if panes periodically re-fire Init(). Consider adding `FromCache: true` flag or stamping only in Update() handler.
- [ ] **fetchedAt len>0 guard blocks empty collections** — Users with genuinely empty libraries (0 playlists, 0 albums) will never get fetchedAt stamped, causing repeated API calls. Distinguish "empty because error" from "empty because user has no data."
- [ ] **Hardcoded time range strings in clearAllFetchingSentinels** — `app.go` iterates `{"short_term", "medium_term", "long_term"}` as literals. Extract to constants to prevent silent sentinel leak on drift.
- [ ] **Pagination response can clear Offset=0 sentinel** — A paginated loaded message (Offset>0) unconditionally clears the fetching sentinel. Narrow window for duplicate Offset=0 fetches during active pagination.

---

## From PR #48 Review — Reusable Components (2026-03-26)

### Investigation Needed

- [x] **Table emptyBorder may add phantom blank lines** — Fixed in PR #51: pageSize overhead formula adjusted to account for border lines.

---

## From PR #52 Review — Library Split (2026-03-26)

### UX Gaps

- [ ] **PlaylistsPane `n` key creates with hardcoded "New Playlist"** — Needs textinput integration to collect user-specified name before emitting `PlaylistCreateRequestMsg`. The old `PlaylistManager` had a `textinput.Model` for this.
- [ ] **PlaylistsPane `r` key sends current name as NewName** — `PlaylistRenameRequestMsg` gets `pl.Name` (current name) instead of a new name. Needs textinput integration to collect the new name. The old `PlaylistManager` had a `textinput.Model` for this.
- [ ] **PlaylistsPane `Title()` calls `store.PlaylistTracks()` on every render** — Could cache the track count in a field updated in `refreshTrackRows()` instead of reading from store on every `Title()` call.
- [ ] **Playlist deletion (`x` key in list view) removed** — The `x` key was using `PlaylistRemoveRequestMsg` (track removal) for playlist deletion. Removed the key since playlist unfollow requires a different message type (`PlaylistUnfollowRequestMsg`). Add proper playlist deletion support when needed.

---

## From PR #53 Review — Stats Split (2026-03-26)

### Data Gap

- [ ] **TopTracksPane "Pop" column always shows "--"** — `domain.Track` lacks a `Popularity` field. The Spotify top-tracks API returns popularity, but it's not captured in the domain model. Either add `Popularity int` to `domain.Track` and populate the column, or replace the column with extra width for Track/Artist.

---

## From PR #56 Review — Page B Nerd Status (2026-03-26)

### Minor

- [ ] **Gateway.Snapshot() is best-effort, not atomic** — Token bucket and gateway mutex are acquired separately. Comment updated to clarify, but snapshot fields may be from slightly different points in time. Acceptable for display purposes.
- [ ] **PollingSnapshotMsg.TickIntervalMs is misleading** — Shows the polling decision interval (3000ms, 10000ms) but the actual tea.Tick fires every 1000ms. Consider renaming to `PollIntervalMs` or displaying the actual tick interval separately.

---

## From PR #58 Review — Cleanup (2026-03-26)

### Documentation

- [ ] **ARCHITECTURE.md references deleted pane names** — The ASCII diagram at line 33 still shows `LibraryPane`, `PlayerPane`, and `QueuePane`. Test examples at lines 621/628 reference `PlayerPane`. These types no longer exist. Update the diagram and examples to reflect the new 10-pane grid layout.
- [ ] **formatDuration duplication** — `formatDuration` in `gradient.go` and `formatDurationMs` in `nowplaying.go` are duplicate implementations. Extract to a shared utility in `components/`.
